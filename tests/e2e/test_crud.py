#!/usr/bin/env python3
"""
End-to-End CRUD Test for Alab

Tests that the schema produces correct database structures and
that data can be inserted/read correctly across all dialects.

Usage:
    python test_crud.py sqlite      # Test SQLite
    python test_crud.py postgres    # Test PostgreSQL
    python test_crud.py all         # Test all databases
"""

import json
import shutil
import subprocess
import sys
import uuid
from datetime import date, time, datetime
from decimal import Decimal
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

# Test data
TEST_USER = {
    "id": str(uuid.uuid4()),
    "username": "testuser",
    "email": "test@example.com",
    "bio": "Hello, I am a test user!",
    "age": 25,
    "score": 95.5,
    "balance": Decimal("1234.56"),
    "active": True,
    "birth_date": date(1999, 1, 15),
    "login_time": time(14, 30, 0),
    "last_seen": datetime(2024, 1, 7, 12, 0, 0),
    "settings": {"theme": "dark", "notifications": True},
    "role": "admin",
}

TEST_POST = {
    "id": str(uuid.uuid4()),
    "title": "Test Post Title",
    "content": "This is the content of the test post.",
    "published": True,
}


def run_alab(config_file: str, *args) -> subprocess.CompletedProcess:
    """Run alab command with specified config."""
    cmd = [str(ALAB_EXE), "-c", config_file] + list(args)
    return subprocess.run(cmd, cwd=SCRIPT_DIR, capture_output=True, text=True)


def clean_migrations():
    """Remove all migration files to start fresh."""
    migrations_dir = SCRIPT_DIR / "migrations"
    if migrations_dir.exists():
        shutil.rmtree(migrations_dir)
    migrations_dir.mkdir(exist_ok=True)


def setup_database(config_file: str):
    """Reset database and run migrations."""
    print(f"  Setting up database with {config_file}...")

    # Clean migrations directory for fresh start
    clean_migrations()

    # Generate and run migrations
    result = run_alab(config_file, "migration:generate", "init")
    if result.returncode != 0:
        print(f"  ERROR generating migration: {result.stderr}")
        return False

    result = run_alab(config_file, "migration:run")
    if result.returncode != 0:
        print(f"  ERROR running migration: {result.stderr}")
        return False

    print("  Database setup complete")
    return True


class SQLiteTest:
    """SQLite CRUD test."""

    def __init__(self):
        self.db_path = SCRIPT_DIR / "test.db"
        self.config = "alab.sqlite.yaml"
        self.conn = None

    def setup(self):
        # Remove old database
        if self.db_path.exists():
            self.db_path.unlink()

        if not setup_database(self.config):
            return False

        self.conn = sqlite3.connect(str(self.db_path))
        self.conn.row_factory = sqlite3.Row
        return True

    def teardown(self):
        if self.conn:
            self.conn.close()
        if self.db_path.exists():
            self.db_path.unlink()

    def test_create_user(self):
        """Test INSERT user."""
        cursor = self.conn.cursor()
        cursor.execute("""
            INSERT INTO core_user (
                id, username, email, bio, age, score, balance,
                active, birth_date, login_time, last_seen, settings, role,
                created_at, updated_at
            ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
        """, (
            TEST_USER["id"],
            TEST_USER["username"],
            TEST_USER["email"],
            TEST_USER["bio"],
            TEST_USER["age"],
            TEST_USER["score"],
            str(TEST_USER["balance"]),
            1 if TEST_USER["active"] else 0,
            TEST_USER["birth_date"].isoformat(),
            TEST_USER["login_time"].isoformat(),
            TEST_USER["last_seen"].isoformat(),
            json.dumps(TEST_USER["settings"]),
            TEST_USER["role"],
            datetime.now().isoformat(),
            datetime.now().isoformat(),
        ))
        self.conn.commit()
        return True

    def test_read_user(self):
        """Test SELECT user."""
        cursor = self.conn.cursor()
        cursor.execute("SELECT * FROM core_user WHERE id = ?", (TEST_USER["id"],))
        row = cursor.fetchone()

        if not row:
            print("  ERROR: User not found")
            return False

        # Verify fields
        assert row["username"] == TEST_USER["username"], f"username mismatch: {row['username']}"
        assert row["email"] == TEST_USER["email"], f"email mismatch: {row['email']}"
        assert row["age"] == TEST_USER["age"], f"age mismatch: {row['age']}"
        assert abs(row["score"] - TEST_USER["score"]) < 0.01, f"score mismatch: {row['score']}"
        assert row["role"] == TEST_USER["role"], f"role mismatch: {row['role']}"

        # Verify JSON
        settings = json.loads(row["settings"])
        assert settings["theme"] == "dark", f"settings.theme mismatch: {settings}"

        return True

    def test_create_post(self):
        """Test INSERT post with foreign key."""
        cursor = self.conn.cursor()
        cursor.execute("""
            INSERT INTO core_post (
                id, author_id, title, content, published, created_at, updated_at
            ) VALUES (?, ?, ?, ?, ?, ?, ?)
        """, (
            TEST_POST["id"],
            TEST_USER["id"],  # FK to user
            TEST_POST["title"],
            TEST_POST["content"],
            1 if TEST_POST["published"] else 0,
            datetime.now().isoformat(),
            datetime.now().isoformat(),
        ))
        self.conn.commit()
        return True

    def test_read_post_with_join(self):
        """Test SELECT post with JOIN to user."""
        cursor = self.conn.cursor()
        cursor.execute("""
            SELECT p.*, u.username as author_name
            FROM core_post p
            JOIN core_user u ON p.author_id = u.id
            WHERE p.id = ?
        """, (TEST_POST["id"],))
        row = cursor.fetchone()

        if not row:
            print("  ERROR: Post not found")
            return False

        assert row["title"] == TEST_POST["title"], f"title mismatch: {row['title']}"
        assert row["author_name"] == TEST_USER["username"], f"author mismatch: {row['author_name']}"

        return True

    def test_update_user(self):
        """Test UPDATE user."""
        cursor = self.conn.cursor()
        cursor.execute("""
            UPDATE core_user SET age = ?, score = ? WHERE id = ?
        """, (30, 99.9, TEST_USER["id"]))
        self.conn.commit()

        cursor.execute("SELECT age, score FROM core_user WHERE id = ?", (TEST_USER["id"],))
        row = cursor.fetchone()

        assert row["age"] == 30, f"age not updated: {row['age']}"
        assert abs(row["score"] - 99.9) < 0.01, f"score not updated: {row['score']}"

        return True

    def test_delete_post(self):
        """Test DELETE post."""
        cursor = self.conn.cursor()
        cursor.execute("DELETE FROM core_post WHERE id = ?", (TEST_POST["id"],))
        self.conn.commit()

        cursor.execute("SELECT COUNT(*) as cnt FROM core_post WHERE id = ?", (TEST_POST["id"],))
        row = cursor.fetchone()

        assert row["cnt"] == 0, f"post not deleted"
        return True


class PostgresTest:
    """PostgreSQL CRUD test using pg8000."""

    def __init__(self):
        self.config = "alab.postgres.yaml"
        self.conn = None

    def setup(self):
        if not pg8000:
            print("  SKIP: pg8000 not installed")
            return None

        try:
            self.conn = pg8000.connect(
                host="localhost",
                port=5432,
                database="alab_test",
                user="alab",
                password="alab"
            )
            self.conn.autocommit = False
        except Exception as e:
            print(f"  SKIP: Cannot connect to PostgreSQL: {e}")
            return None

        # Drop existing tables
        cursor = self.conn.cursor()
        cursor.execute("DROP TABLE IF EXISTS core_post CASCADE")
        cursor.execute("DROP TABLE IF EXISTS core_user CASCADE")
        cursor.execute("DROP TABLE IF EXISTS alab_migrations CASCADE")
        self.conn.commit()

        if not setup_database(self.config):
            return False

        return True

    def teardown(self):
        if self.conn:
            self.conn.close()

    def test_create_user(self):
        cursor = self.conn.cursor()
        cursor.execute("""
            INSERT INTO core_user (
                id, username, email, bio, age, score, balance,
                active, birth_date, login_time, last_seen, settings, role
            ) VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s)
        """, (
            TEST_USER["id"],
            TEST_USER["username"],
            TEST_USER["email"],
            TEST_USER["bio"],
            TEST_USER["age"],
            TEST_USER["score"],
            str(TEST_USER["balance"]),
            TEST_USER["active"],
            TEST_USER["birth_date"],
            TEST_USER["login_time"],
            TEST_USER["last_seen"],
            json.dumps(TEST_USER["settings"]),
            TEST_USER["role"],
        ))
        self.conn.commit()
        return True

    def test_read_user(self):
        cursor = self.conn.cursor()
        cursor.execute("SELECT * FROM core_user WHERE id = %s", (TEST_USER["id"],))
        row = cursor.fetchone()

        if not row:
            print("  ERROR: User not found")
            return False

        return True

    def test_create_post(self):
        cursor = self.conn.cursor()
        cursor.execute("""
            INSERT INTO core_post (id, author_id, title, content, published)
            VALUES (%s, %s, %s, %s, %s)
        """, (
            TEST_POST["id"],
            TEST_USER["id"],
            TEST_POST["title"],
            TEST_POST["content"],
            TEST_POST["published"],
        ))
        self.conn.commit()
        return True

    def test_read_post_with_join(self):
        cursor = self.conn.cursor()
        cursor.execute("""
            SELECT p.*, u.username as author_name
            FROM core_post p
            JOIN core_user u ON p.author_id = u.id
            WHERE p.id = %s
        """, (TEST_POST["id"],))
        row = cursor.fetchone()
        return row is not None

    def test_update_user(self):
        cursor = self.conn.cursor()
        cursor.execute("UPDATE core_user SET age = %s WHERE id = %s", (30, TEST_USER["id"]))
        self.conn.commit()
        return True

    def test_delete_post(self):
        cursor = self.conn.cursor()
        cursor.execute("DELETE FROM core_post WHERE id = %s", (TEST_POST["id"],))
        self.conn.commit()
        return True


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
        print(f"  FAILED: Setup failed")
        return (0, 1)

    tests = [
        ("Create User", test.test_create_user),
        ("Read User", test.test_read_user),
        ("Create Post (FK)", test.test_create_post),
        ("Read Post (JOIN)", test.test_read_post_with_join),
        ("Update User", test.test_update_user),
        ("Delete Post", test.test_delete_post),
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
