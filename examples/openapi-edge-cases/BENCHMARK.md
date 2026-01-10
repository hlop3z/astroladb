# Relationship Edge Cases Benchmark

This benchmark validates that Alab correctly handles all common database relationship patterns used in 90-99% of CRUD applications.

## Relationship Types Tested

| #   | Relationship Type          | Test            | Description                       |
|-----|----------------------------|-----------------|-----------------------------------|
| 1   | Self-referential FK        | `referred_by`   | User referred by another user     |
| 2   | One-to-one                 | `profile`       | Each user has exactly one profile |
| 3   | Many-to-many               | `user - role`   | Users can have multiple roles     |
| 4   | Self-referential hierarchy | `category.parent` | Nested categories               |
| 5   | One-to-many                | `user -> posts` | One user has many posts           |
| 6   | Many-to-one                | `comments -> post` | Many comments belong to one post |
| 7   | Multiple FKs to same table | `author/editor` | Post has two FKs to user          |
| 8   | Self-referential threaded  | `reply_to`      | Threaded comments                 |
| 9   | Many-to-many               | `post - tag`    | Posts can have multiple tags      |
| 10  | Unique constraint          | `follow`        | Unique follower/following pairs   |
| 11  | Polymorphic                | `reaction`      | React to post OR comment          |
| 12  | Polymorphic                | `bookmark`      | Bookmark post OR comment          |
| 13  | Polymorphic                | `media`         | Attach to post/comment/profile    |

## Test Results

```
============================================================
Testing SQLite
============================================================
  PASS: Self-referential user (referred_by)
  PASS: One-to-one profile
  PASS: Many-to-many user-role
  PASS: Self-referential category (parent)
  PASS: One-to-many (user has posts)
  PASS: Many-to-one (comments to post)
  PASS: Multiple FKs (post author/editor)
  PASS: Self-referential comment (threaded)
  PASS: Many-to-many post-tag
  PASS: Unique constraint (follow)
  PASS: Polymorphic reaction
  PASS: Polymorphic bookmark
  PASS: Polymorphic media

============================================================
Testing PostgreSQL
============================================================
  PASS: Self-referential user (referred_by)
  PASS: One-to-one profile
  PASS: Many-to-many user-role
  PASS: Self-referential category (parent)
  PASS: One-to-many (user has posts)
  PASS: Many-to-one (comments to post)
  PASS: Multiple FKs (post author/editor)
  PASS: Self-referential comment (threaded)
  PASS: Many-to-many post-tag
  PASS: Unique constraint (follow)
  PASS: Polymorphic reaction
  PASS: Polymorphic bookmark
  PASS: Polymorphic media

============================================================
TOTAL: 26 passed, 0 failed
============================================================
```

## Summary

| Database   | Passed | Failed | Total |
|------------|--------|--------|-------|
| SQLite     | 13     | 0      | 13    |
| PostgreSQL | 13     | 0      | 13    |
| **TOTAL**  | **26** | **0**  | **26**|

## Schema DSL Examples (Chained API)

### Self-referential FK
```js
// auth/user.js
t.belongs_to("auth.user").as("referred_by").optional()
```

### One-to-one
```js
// auth/profile.js
t.one_to_one("auth.user")
```

### Many-to-many
```js
// auth/user.js
t.many_to_many("auth.role")
```

### One-to-many / Many-to-one
```js
// content/post.js
t.belongs_to("auth.user").as("author")  // Many posts belong to one user

// content/comment.js
t.belongs_to("content.post")  // Many comments belong to one post
```

### Multiple FKs to same table
```js
// content/post.js
t.belongs_to("auth.user").as("author")
t.belongs_to("auth.user").as("editor").optional()
```

### Self-referential hierarchy
```js
// content/category.js
t.belongs_to("content.category").as("parent").optional().on_delete("cascade")
```

### Unique constraint
```js
// social/follow.js
t.belongs_to("auth.user").as("follower")
t.belongs_to("auth.user").as("following")
t.unique("follower", "following")
```

### Polymorphic
```js
// social/reaction.js
t.belongs_to_any(["content.post", "content.comment"], { as: "reactable" })
```

## Running the Benchmark

```bash
# Run all databases
python test_edge_cases.py all

# Run specific database
python test_edge_cases.py sqlite
python test_edge_cases.py postgres
```

## Requirements

- Python 3.8+
- Database drivers: `sqlite3` (builtin), `pg8000`
- Running database instance for PostgreSQL
