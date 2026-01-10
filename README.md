<p align="center">
  <img src="docs/src/assets/logo.png" alt="astrola-db" width="180" />
</p>

<h1 align="center">Astroladb</h1>

<p align="center">
  <strong>Schema-first database migrations. Write once, export everywhere.</strong>
</p>

<p align="center">
  <a href="https://github.com/hlop3z/astroladb/releases"><img src="https://img.shields.io/github/v/release/hlop3z/astroladb" alt="Release"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-BSD--3--Clause-blue.svg" alt="License"></a>
  <img src="https://img.shields.io/badge/databases-PostgreSQL%20%7C%20SQLite-336791" alt="Databases">
</p>

---

## What is Astroladb?

Astroladb is a **language-agnostic database migration tool** that lets you define your schema in JavaScript and automatically generates:

- Database migrations (PostgreSQL & SQLite)
- OpenAPI specifications
- GraphQL schemas
- TypeScript, Go, Python & Rust types

**No ORM. No framework lock-in. Just clean migrations and type exports.**

---

## Why Astroladb?

| Problem                                 | Astroladb Solution                           |
| --------------------------------------- | -------------------------------------------- |
| ORMs tie you to one language            | Define schema once, export to any language   |
| Manual migration writing is error-prone | Auto-generate migrations from schema changes |
| Keeping API docs in sync is tedious     | OpenAPI/GraphQL generated from your schema   |
| Type definitions drift from database    | Types always match your actual schema        |

---

## Installation

```bash
# Quick install (Linux/macOS)
curl -fsSL https://raw.githubusercontent.com/hlop3z/astroladb/main/install.sh | sh

# Or with Go
go install github.com/hlop3z/astroladb/cmd/alab@latest

# Verify installation
alab --version
```

---

## Quick Start

### 1. Initialize your project

```bash
mkdir myapp && cd myapp
alab init
```

This creates:

```
myapp/
├── alab.yaml        # Configuration
├── schemas/         # Your schema definitions
├── migrations/      # Generated migrations
└── types/           # TypeScript definitions for IDE support
```

### 2. Define your first schema

```js
// schemas/auth/user.js
export default table({
  id: col.id(),
  email: col.email().unique(),
  username: col.username().unique(),
  password: col.password_hash(),
  is_active: col.flag(true),
}).timestamps();
```

### 3. Generate and apply migrations

```bash
# Generate migration from your schema
alab new create_users

# Preview the SQL (optional)
alab migrate --dry-run

# Apply to database
alab migrate
```

### 4. Export types for your app

```bash
# Export everything
alab export -f all

# Or specific formats
alab export -f typescript
alab export -f openapi
```

---

## Live Documentation Server

Preview your API documentation with hot reload:

```bash
alab http
```

Then open:

- **Swagger UI**: http://localhost:8080/
- **GraphiQL**: http://localhost:8080/graphiql

Changes to your schema files refresh automatically!

---

## Schema Examples

### Basic Table with Relationships

```js
// schemas/blog/post.js
export default table({
  id: col.id(),
  author: col.belongs_to("auth.user"),
  title: col.title(),
  slug: col.slug().unique(),
  body: col.body(),
  status: col.enum(["draft", "published", "archived"]).default("draft"),
  published_at: col.datetime().optional(),
})
  .timestamps()
  .soft_delete()
  .searchable(["title", "body"]);
```

### Many-to-Many Relationship

```js
// schemas/auth/user.js
export default table({
  id: col.id(),
  email: col.email().unique(),
  // ... other fields
})
  .timestamps()
  .many_to_many("auth.role"); // Auto-creates join table
```

### Self-Referential (Hierarchies)

```js
// schemas/catalog/category.js
export default table({
  id: col.id(),
  name: col.name(),
  parent: col.belongs_to("self").optional(),
}).timestamps();
```

---

## Column Types

### Semantic Types (Recommended)

These come with built-in validation, formatting, and sensible defaults:

| Type                            | What you get                    |
| ------------------------------- | ------------------------------- |
| `col.id()`                      | UUID primary key                |
| `col.email()`                   | string(255) + email validation  |
| `col.username()`                | string(50) + pattern + min(3)   |
| `col.password_hash()`           | string(255), hidden from API    |
| `col.money()`                   | decimal(19,4), precise currency |
| `col.flag()` / `col.flag(true)` | boolean with default            |
| `col.slug()`                    | URL-safe string, auto-unique    |
| `col.country()`                 | ISO 3166-1 (US, GB, DE)         |
| `col.currency()`                | ISO 4217 (USD, EUR, GBP)        |

[See all 30+ semantic types →](#full-type-reference)

### Low-Level Types

For custom needs:

```js
col.string(100); // VARCHAR(100)
col.integer(); // 32-bit int
col.decimal(10, 2); // DECIMAL(10,2)
col.json(); // JSONB
col.enum(["a", "b"]); // CHECK constraint
```

---

## CLI Commands

| Command             | Description                                         |
| ------------------- | --------------------------------------------------- |
| `alab init`         | Initialize new project                              |
| `alab new <name>`   | Generate migration from schema changes              |
| `alab migrate`      | Apply pending migrations                            |
| `alab rollback [n]` | Rollback last n migrations                          |
| `alab status`       | Show migration status                               |
| `alab diff`         | Show schema vs database diff                        |
| `alab check`        | Validate schema files                               |
| `alab export`       | Export to OpenAPI/GraphQL/TypeScript/Go/Python/Rust |
| `alab http`         | Start live documentation server                     |
| `alab types`        | Regenerate IDE autocomplete definitions             |

### Useful Flags

```bash
alab migrate --dry-run      # Preview SQL without executing
alab status --json          # JSON output for CI/CD
alab export -f all          # Export all formats
alab export -f all --merge  # Single file per format
alab http -p 3000           # Custom port
```

---

## Configuration

```yaml
# alab.yaml
database:
  dialect: postgres # or sqlite
  url: postgres://user:pass@localhost:5432/mydb

schemas: ./schemas
migrations: ./migrations
```

---

## Full Type Reference

### Semantic Types

| Type                  | Expands To                     |
| --------------------- | ------------------------------ |
| `col.email()`         | string(255) + email format     |
| `col.username()`      | string(50) + pattern + min(3)  |
| `col.password_hash()` | string(255), hidden            |
| `col.phone()`         | string(50) + E.164             |
| `col.name()`          | string(100)                    |
| `col.title()`         | string(200)                    |
| `col.slug()`          | string(255) + unique + pattern |
| `col.body()`          | text                           |
| `col.summary()`       | string(500)                    |
| `col.url()`           | string(2048) + uri format      |
| `col.ip()`            | string(45)                     |
| `col.user_agent()`    | string(500)                    |
| `col.money()`         | decimal(19,4) + min(0)         |
| `col.percentage()`    | decimal(5,2) + 0-100           |
| `col.counter()`       | integer + default(0)           |
| `col.quantity()`      | integer + min(0)               |
| `col.flag()`          | boolean + default(false)       |
| `col.token()`         | string(64), for API keys       |
| `col.code()`          | string(20), for SKUs           |
| `col.country()`       | string(2), ISO 3166-1          |
| `col.currency()`      | string(3), ISO 4217            |
| `col.locale()`        | string(10), BCP 47             |
| `col.timezone()`      | string(50), IANA               |
| `col.rating()`        | decimal(2,1), 0-5              |
| `col.duration()`      | integer, seconds               |
| `col.color()`         | string(7), #RRGGBB             |
| `col.markdown()`      | text + format hint             |
| `col.html()`          | text + format hint             |

### Column Modifiers

```js
col.email()
  .optional()           // Allow NULL
  .unique()             // Unique constraint
  .default("value")     // Default for new rows
  .backfill("value")    // Fill existing rows in migration
  .min(n) / .max(n)     // Validation constraints
  .pattern(/regex/)     // Custom regex
  .docs("Description")  // SQL comment + OpenAPI description
```

### Table Modifiers

```js
table({ ... })
  .timestamps()                          // created_at, updated_at
  .auditable()                           // created_by, updated_by
  .soft_delete()                         // deleted_at
  .sort_by(["-created_at", "name"])      // Default ORDER BY
  .searchable(["col1", "col2"])          // Text search columns
  .filterable(["col1", "col2"])          // Filter columns
  .unique("col1", "col2")                // Composite unique
  .index("col1", "col2")                 // Composite index
  .many_to_many("other.table")           // M2M relationship
```

### Relationships

```js
// Foreign key (creates author_id)
author: col.belongs_to("auth.user");

// Optional with cascade delete
parent: col.belongs_to("self").optional().on_delete("cascade");

// One-to-one (unique FK)
profile: col
  .one_to_one("auth.user")

  // Many-to-many (table-level)
  .many_to_many("auth.role");

// Polymorphic
target: col.belongs_to_any(["blog.post", "blog.comment"]);
```

---

## Supported Databases

| Database       | Use Case                       |
| -------------- | ------------------------------ |
| **PostgreSQL** | Production                     |
| **SQLite**     | Development, testing, embedded |

---

## License

BSD-3-Clause
