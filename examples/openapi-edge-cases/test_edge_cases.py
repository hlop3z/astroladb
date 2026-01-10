#!/usr/bin/env python3
"""
E2E Edge Cases Test for Alab

Tests all relationship types and edge cases:
1. Self-referential FK (user referred_by)
2. One-to-one (profile → user)
3. Many-to-many (user ↔ role, post ↔ tag)
4. Self-referential hierarchy (category parent)
5. Multiple FKs to same table (post author/editor)
6. Self-referential with unique constraint (follow)
7. Polymorphic relationships (reaction, bookmark, media)

Usage:
    python test_edge_cases.py sqlite
    python test_edge_cases.py postgres
    python test_edge_cases.py all
"""

import json
import shutil
import subprocess
import sys
import uuid
from datetime import datetime
from pathlib import Path

# Database drivers
try:
    import sqlite3
except ImportError:
    sqlite3 = None

try:
    import pg8000
except ImportError:
    pg8000 = None



SCRIPT_DIR = Path(__file__).parent
ALAB_EXE = SCRIPT_DIR.parent.parent / "alab.exe"


def clean_migrations():
    """Remove all migration files to start fresh."""
    migrations_dir = SCRIPT_DIR / "migrations"
    if migrations_dir.exists():
        shutil.rmtree(migrations_dir)
    migrations_dir.mkdir(exist_ok=True)


def run_alab(config_file: str, *args) -> subprocess.CompletedProcess:
    """Run alab command with specified config."""
    cmd = [str(ALAB_EXE), "-c", config_file] + list(args)
    return subprocess.run(cmd, cwd=SCRIPT_DIR, capture_output=True, text=True)


class BaseTest:
    """Base class for database tests."""

    def __init__(self):
        self.conn = None
        self.test_ids = {}

    def execute(self, sql: str, params=None):
        """Execute SQL - must be implemented by subclass."""
        raise NotImplementedError

    def generate_uuid(self) -> str:
        return str(uuid.uuid4())

    def now(self) -> str:
        return datetime.now().isoformat()

    # Test 1: Self-referential FK (user referred_by)
    def test_self_referential_user(self):
        """Test user with self-referential referred_by."""
        # Create first user (no referrer)
        user1_id = self.generate_uuid()
        self.execute("""
            INSERT INTO auth_user (id, email, username, password_hash, is_active, is_verified, created_at, updated_at)
            VALUES (%s, %s, %s, %s, %s, %s, %s, %s)
        """, (user1_id, "user1@test.com", "user1", "hash1", True, True, self.now(), self.now()))

        # Create second user referred by first
        user2_id = self.generate_uuid()
        self.execute("""
            INSERT INTO auth_user (id, email, username, password_hash, referred_by_id, is_active, is_verified, created_at, updated_at)
            VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s)
        """, (user2_id, "user2@test.com", "user2", "hash2", user1_id, True, False, self.now(), self.now()))

        self.test_ids["user1"] = user1_id
        self.test_ids["user2"] = user2_id
        return True

    # Test 2: One-to-one (profile → user)
    def test_one_to_one_profile(self):
        """Test one-to-one profile relationship."""
        profile_id = self.generate_uuid()
        self.execute("""
            INSERT INTO auth_profile (id, user_id, display_name, visibility, created_at, updated_at)
            VALUES (%s, %s, %s, %s, %s, %s)
        """, (profile_id, self.test_ids["user1"], "User One", "public", self.now(), self.now()))

        self.test_ids["profile1"] = profile_id
        return True

    # Test 3: Many-to-many (user ↔ role)
    def test_many_to_many_user_role(self):
        """Test many-to-many user ↔ role relationship."""
        # Create roles
        role1_id = self.generate_uuid()
        role2_id = self.generate_uuid()
        self.execute("""
            INSERT INTO auth_role (id, name, permissions, created_at, updated_at)
            VALUES (%s, %s, %s, %s, %s)
        """, (role1_id, "admin", json.dumps({"all": True}), self.now(), self.now()))
        self.execute("""
            INSERT INTO auth_role (id, name, permissions, created_at, updated_at)
            VALUES (%s, %s, %s, %s, %s)
        """, (role2_id, "editor", json.dumps({"edit": True}), self.now(), self.now()))

        # Assign roles to user (many-to-many join table)
        self.execute("""
            INSERT INTO auth_role_auth_user (user_id, role_id)
            VALUES (%s, %s)
        """, (self.test_ids["user1"], role1_id))
        self.execute("""
            INSERT INTO auth_role_auth_user (user_id, role_id)
            VALUES (%s, %s)
        """, (self.test_ids["user1"], role2_id))

        self.test_ids["role1"] = role1_id
        self.test_ids["role2"] = role2_id
        return True

    # Test 4: Self-referential hierarchy (category parent)
    def test_self_referential_category(self):
        """Test self-referential category hierarchy."""
        # Root category
        cat1_id = self.generate_uuid()
        self.execute("""
            INSERT INTO content_category (id, name, slug, sort_order, is_active, created_at, updated_at)
            VALUES (%s, %s, %s, %s, %s, %s, %s)
        """, (cat1_id, "Technology", "technology", 1, True, self.now(), self.now()))

        # Child category
        cat2_id = self.generate_uuid()
        self.execute("""
            INSERT INTO content_category (id, name, slug, parent_id, sort_order, is_active, created_at, updated_at)
            VALUES (%s, %s, %s, %s, %s, %s, %s, %s)
        """, (cat2_id, "Programming", "programming", cat1_id, 1, True, self.now(), self.now()))

        self.test_ids["category1"] = cat1_id
        self.test_ids["category2"] = cat2_id
        return True

    # Test 5: One-to-many (user has many posts via belongs_to)
    def test_one_to_many_posts(self):
        """Test one-to-many: one user has many posts."""
        # Create multiple posts for the same user
        post1_id = self.generate_uuid()
        post2_id = self.generate_uuid()
        post3_id = self.generate_uuid()

        for i, post_id in enumerate([post1_id, post2_id, post3_id], 1):
            self.execute("""
                INSERT INTO content_post (id, author_id, category_id, title, slug, content, status, is_featured, allow_comments, view_count, created_at, updated_at)
                VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s)
            """, (
                post_id,
                self.test_ids["user1"],  # Same author for all
                self.test_ids["category2"],
                f"Post {i}", f"post-{i}", f"Content {i}", "published",
                False, True, 0, self.now(), self.now()
            ))

        self.test_ids["post1"] = post1_id
        self.test_ids["post2"] = post2_id
        self.test_ids["post3"] = post3_id
        return True

    # Test 6: Many-to-one (many comments belong to one post)
    def test_many_to_one_comments(self):
        """Test many-to-one: many comments belong to one post."""
        # Create multiple comments on the same post
        comment1_id = self.generate_uuid()
        comment2_id = self.generate_uuid()
        comment3_id = self.generate_uuid()

        for i, (comment_id, author_id) in enumerate([
            (comment1_id, self.test_ids["user1"]),
            (comment2_id, self.test_ids["user2"]),
            (comment3_id, self.test_ids["user1"]),
        ], 1):
            self.execute("""
                INSERT INTO content_comment (id, post_id, author_id, body, is_edited, is_hidden, created_at, updated_at)
                VALUES (%s, %s, %s, %s, %s, %s, %s, %s)
            """, (comment_id, self.test_ids["post1"], author_id, f"Comment {i}", False, False, self.now(), self.now()))

        self.test_ids["comment1"] = comment1_id
        self.test_ids["comment2"] = comment2_id
        self.test_ids["comment3"] = comment3_id
        return True

    # Test 7: Multiple FKs to same table (post author/editor)
    def test_multiple_fks_post(self):
        """Test post with author and editor FKs to same table."""
        # Update post1 to also have an editor
        self.execute("""
            UPDATE content_post SET editor_id = %s WHERE id = %s
        """, (self.test_ids["user2"], self.test_ids["post1"]))
        return True

    # Test 8: Self-referential comment (threaded replies)
    def test_self_referential_comment(self):
        """Test threaded comments with self-referential FK (reply_to)."""
        # Create a reply to comment1 (already created in many-to-one test)
        reply_id = self.generate_uuid()
        self.execute("""
            INSERT INTO content_comment (id, post_id, author_id, reply_to_id, body, is_edited, is_hidden, created_at, updated_at)
            VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s)
        """, (reply_id, self.test_ids["post1"], self.test_ids["user2"], self.test_ids["comment1"], "Reply to comment 1", False, False, self.now(), self.now()))

        self.test_ids["reply1"] = reply_id
        return True

    # Test 7: Many-to-many (post ↔ tag)
    def test_many_to_many_post_tag(self):
        """Test many-to-many post ↔ tag relationship."""
        tag1_id = self.generate_uuid()
        tag2_id = self.generate_uuid()
        self.execute("""
            INSERT INTO content_tag (id, name, slug, color, created_at, updated_at)
            VALUES (%s, %s, %s, %s, %s, %s)
        """, (tag1_id, "Python", "python", "#3776AB", self.now(), self.now()))
        self.execute("""
            INSERT INTO content_tag (id, name, slug, color, created_at, updated_at)
            VALUES (%s, %s, %s, %s, %s, %s)
        """, (tag2_id, "Go", "go", "#00ADD8", self.now(), self.now()))

        # Tag the post
        self.execute("""
            INSERT INTO content_post_content_tag (post_id, tag_id)
            VALUES (%s, %s)
        """, (self.test_ids["post1"], tag1_id))
        self.execute("""
            INSERT INTO content_post_content_tag (post_id, tag_id)
            VALUES (%s, %s)
        """, (self.test_ids["post1"], tag2_id))

        self.test_ids["tag1"] = tag1_id
        self.test_ids["tag2"] = tag2_id
        return True

    # Test 8: Self-referential with unique constraint (follow)
    def test_follow_unique(self):
        """Test follow with unique constraint."""
        follow_id = self.generate_uuid()
        self.execute("""
            INSERT INTO social_follow (id, follower_id, following_id, followed_at, notifications_enabled, created_at, updated_at)
            VALUES (%s, %s, %s, %s, %s, %s, %s)
        """, (follow_id, self.test_ids["user1"], self.test_ids["user2"], self.now(), True, self.now(), self.now()))

        self.test_ids["follow1"] = follow_id
        return True

    # Test 9: Polymorphic reaction
    def test_polymorphic_reaction(self):
        """Test polymorphic reaction (on post)."""
        reaction_id = self.generate_uuid()
        self.execute("""
            INSERT INTO social_reaction (id, user_id, reactable_type, reactable_id, type, created_at, updated_at)
            VALUES (%s, %s, %s, %s, %s, %s, %s)
        """, (reaction_id, self.test_ids["user1"], "content.post", self.test_ids["post1"], "like", self.now(), self.now()))

        # Reaction on comment
        reaction2_id = self.generate_uuid()
        self.execute("""
            INSERT INTO social_reaction (id, user_id, reactable_type, reactable_id, type, created_at, updated_at)
            VALUES (%s, %s, %s, %s, %s, %s, %s)
        """, (reaction2_id, self.test_ids["user2"], "content.comment", self.test_ids["comment1"], "love", self.now(), self.now()))

        self.test_ids["reaction1"] = reaction_id
        self.test_ids["reaction2"] = reaction2_id
        return True

    # Test 10: Polymorphic bookmark
    def test_polymorphic_bookmark(self):
        """Test polymorphic bookmark."""
        bookmark_id = self.generate_uuid()
        self.execute("""
            INSERT INTO social_bookmark (id, user_id, bookmarkable_type, bookmarkable_id, note, created_at, updated_at)
            VALUES (%s, %s, %s, %s, %s, %s, %s)
        """, (bookmark_id, self.test_ids["user1"], "content.post", self.test_ids["post1"], "Save for later", self.now(), self.now()))

        self.test_ids["bookmark1"] = bookmark_id
        return True

    # Test 11: Polymorphic media
    def test_polymorphic_media(self):
        """Test polymorphic media attachment."""
        media_id = self.generate_uuid()
        self.execute("""
            INSERT INTO social_media (id, uploaded_by_id, attachable_type, attachable_id, filename, url, mime_type, size_bytes, type, sort_order, created_at, updated_at)
            VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s)
        """, (
            media_id, self.test_ids["user1"], "content.post", self.test_ids["post1"],
            "image.jpg", "https://example.com/image.jpg", "image/jpeg", 12345,
            "image", 0, self.now(), self.now()
        ))

        self.test_ids["media1"] = media_id
        return True


class SQLiteTest(BaseTest):
    """SQLite test implementation."""

    def __init__(self):
        super().__init__()
        self.db_path = SCRIPT_DIR / "blog.db"
        self.config = "alab.yaml"

    def setup(self):
        if self.db_path.exists():
            self.db_path.unlink()

        clean_migrations()

        # Generate and run migrations
        result = run_alab(self.config, "migration:generate", "init")
        if result.returncode != 0:
            print(f"  ERROR generating migration: {result.stderr}")
            return False

        result = run_alab(self.config, "migration:run")
        if result.returncode != 0:
            print(f"  ERROR running migration: {result.stderr}")
            return False

        self.conn = sqlite3.connect(str(self.db_path))
        return True

    def teardown(self):
        if self.conn:
            self.conn.close()
        if self.db_path.exists():
            self.db_path.unlink()

    def execute(self, sql: str, params=None):
        # Convert %s placeholders to ?
        sql = sql.replace("%s", "?")
        cursor = self.conn.cursor()
        cursor.execute(sql, params or ())
        self.conn.commit()


class PostgresTest(BaseTest):
    """PostgreSQL test implementation."""

    def __init__(self):
        super().__init__()
        self.config = "alab.postgres.yaml"

    def setup(self):
        if not pg8000:
            print("  SKIP: pg8000 not installed")
            return None

        try:
            self.conn = pg8000.connect(
                host="localhost", port=5432, database="alab_test",
                user="alab", password="alab"
            )
        except Exception as e:
            print(f"  SKIP: Cannot connect to PostgreSQL: {e}")
            return None

        # Drop all tables
        tables = [
            "social_media", "social_bookmark", "social_reaction", "social_follow",
            "content_post_content_tag", "content_tag", "content_comment", "content_post", "content_category",
            "auth_role_auth_user", "auth_profile", "auth_user", "auth_role", "alab_migrations"
        ]
        cursor = self.conn.cursor()
        for t in tables:
            try:
                cursor.execute(f"DROP TABLE IF EXISTS {t} CASCADE")
            except:
                pass
        self.conn.commit()

        clean_migrations()

        result = run_alab(self.config, "migration:generate", "init")
        if result.returncode != 0:
            print(f"  ERROR generating migration: {result.stderr}")
            return False

        result = run_alab(self.config, "migration:run")
        if result.returncode != 0:
            print(f"  ERROR running migration: {result.stderr}")
            return False

        return True

    def teardown(self):
        if self.conn:
            self.conn.close()

    def execute(self, sql: str, params=None):
        cursor = self.conn.cursor()
        cursor.execute(sql, params or ())
        self.conn.commit()


def run_tests(test_class, name: str) -> tuple[int, int]:
    """Run all tests for a database."""
    print(f"\n{'='*60}")
    print(f"Testing {name}")
    print(f"{'='*60}")

    test = test_class()
    setup_result = test.setup()

    if setup_result is None:
        return (0, 0)  # Skipped

    if not setup_result:
        print("  FAILED: Setup failed")
        return (0, 1)

    tests = [
        ("Self-referential user (referred_by)", test.test_self_referential_user),
        ("One-to-one profile", test.test_one_to_one_profile),
        ("Many-to-many user-role", test.test_many_to_many_user_role),
        ("Self-referential category (parent)", test.test_self_referential_category),
        ("One-to-many (user has posts)", test.test_one_to_many_posts),
        ("Many-to-one (comments to post)", test.test_many_to_one_comments),
        ("Multiple FKs (post author/editor)", test.test_multiple_fks_post),
        ("Self-referential comment (threaded)", test.test_self_referential_comment),
        ("Many-to-many post-tag", test.test_many_to_many_post_tag),
        ("Unique constraint (follow)", test.test_follow_unique),
        ("Polymorphic reaction", test.test_polymorphic_reaction),
        ("Polymorphic bookmark", test.test_polymorphic_bookmark),
        ("Polymorphic media", test.test_polymorphic_media),
    ]

    passed = 0
    failed = 0

    for test_name, test_func in tests:
        try:
            result = test_func()
            if result:
                print(f"  PASS: {test_name}")
                passed += 1
            else:
                print(f"  FAIL: {test_name}")
                failed += 1
        except Exception as e:
            print(f"  FAIL: {test_name} - {e}")
            failed += 1

    test.teardown()
    return (passed, failed)


def main():
    if len(sys.argv) < 2:
        print(__doc__)
        sys.exit(1)

    target = sys.argv[1].lower()

    total_passed = 0
    total_failed = 0

    if target in ("sqlite", "all"):
        p, f = run_tests(SQLiteTest, "SQLite")
        total_passed += p
        total_failed += f

    if target in ("postgres", "postgresql", "all"):
        p, f = run_tests(PostgresTest, "PostgreSQL")
        total_passed += p
        total_failed += f

    print(f"\n{'='*60}")
    print(f"TOTAL: {total_passed} passed, {total_failed} failed")
    print(f"{'='*60}")

    sys.exit(0 if total_failed == 0 else 1)


if __name__ == "__main__":
    main()
