<p align="center">
    <img src="docs/src/assets/logo.png" alt="astrola-db" width="180" />
</p>

<h1 align="center">AstrolaDB (alab)</h1>

<p align="center">
    <a href="https://github.com/hlop3z/astroladb/actions/workflows/ci.yml"><img
            src="https://github.com/hlop3z/astroladb/actions/workflows/ci.yml/badge.svg" alt="CI"></a>
    <a href="https://goreportcard.com/report/github.com/hlop3z/astroladb"><img
            src="https://goreportcard.com/badge/github.com/hlop3z/astroladb" alt="Go Report Card"></a>
    <a href="LICENSE"><img src="https://img.shields.io/badge/license-BSD--3--Clause-blue.svg" alt="License"></a>
</p>

<p align="center">
    <strong>One schema: many languages.</strong>
</p>

<p align="center">
    <a href="https://github.com/hlop3z/astroladb/releases"><img
            src="https://img.shields.io/github/v/release/hlop3z/astroladb" alt="Release"></a>
    <img src="https://img.shields.io/badge/databases-PostgreSQL%20%7C%20SQLite-336791" alt="Databases">
    <img src="https://img.shields.io/badge/status-experimental-orange" alt="Experimental">
</p>

<p align="center">
    <a href="https://hlop3z.github.io/astroladb/">Documentation</a>
</p>

---

Welcome to **AstrolaDB** aka `alab`. A schema orchestration tool with a `one-to-many` **multi-language** code generation.

> **No** Node.js **dependency**. **No** runtime **lock-in**.

Define your schema **once** in **JavaScript**, then generate **SQL** migrations, **Rust** structs, **Go** models, **Python** classes, **TypeScript** types, **GraphQL** schemas, and **OpenAPI** specs.

Schemas are written in **constrained** JavaScript **not for logic**, but for **type safety**, **autocomplete**, IDE support and explicit **configuration**. Think of it as a **fancy** `JSON`.

```mermaid
flowchart TD
    A[model.js] --> B[Generated Artifacts]

    B --- D[Migrations]
    B --- E[Code - Types]
    B --- F[API Contracts]
```

---

## Key Highlights

- **Lightweight & Portable**
- Zero-dependency, **single binary (~8 MB)**
- Fast, portable and **CI/CD-friendly**
- **No heavy runtimes**: JVM | Node.js | Python

---

## Install

```bash
curl -fsSL https://raw.githubusercontent.com/hlop3z/astroladb/main/install.sh | sh
```

## Experimental Status

**AstrolaDB** is actively under development. Please review the current feature stability:

### 1. Migrations: **Experimental**

The migration engine is **not yet battle-tested** in large-scale production. Use `alab migrate` with caution:

- **Stability:** Migration logic is evolving and hasn't undergone extensive stress testing.
- **API Changes:** The migration API may change, potentially introducing breaking changes.

### 2. **Code Generation**: Stable

Schema orchestration and type generation are **safe for development**:

- `alab export` produces types for **Rust, Go, Python, and TypeScript** without touching your live database.
- **Recommended for development workflows**. For production, review generated SQL and verify migrations manually.

> **Note:** Always validate migrations in a staging environment before applying to production.

---

## Features 🎸

**Alab** acts as a database instrument, giving you **a
lab** to play, explore and design your schemas.

| Feature                     | Concept     | Technical Capability                                                  |
| --------------------------- | ----------- | --------------------------------------------------------------------- |
| **Unified Source of Truth** | Lead Singer | Centralizes schemas in **JavaScript**, preventing database/app drift. |
| **Multi-Language Export**   | Instruments | Generates **type-safe** Rust, Go, Python, and TypeScript structures.  |
| **Live Development**        | Sound Check | Built-in HTTP server (`alab live`) for instant schema previews.       |
| **Embedded Engine**         | -           | Runs `.js` schemas via Goja in a standalone **Go binary**.            |
| **Schema Orchestration**    | -           | Manages **SQL migrations** from generation to deployment.             |
| **OpenAPI Integration**     | -           | Exports `openapi.json` for **25+ languages** via Quicktype.           |
| **Logical Namespacing**     | -           | Groups tables (`auth.user`) to avoid naming collisions.               |
| **Runtime Independence**    | -           | Produces **SQL** and **native types**.                                |

## Live Server

**The live server provides instant API exploration with automatic hot reloading.**

```bash
alab live
```

---

<p align="center">
  <img src="docs/gifs/http-preview.gif" alt="HTTP Demo" width="800" />
  <img src="docs/gifs/dx.gif" alt="HTTP Demo" width="800" />
  <img src="docs/gifs/status.gif" alt="CLI status Demo" width="800" />
  <img src="docs/gifs/workflow.gif" alt="CLI workflow Demo" width="800" />
</p>

---

## Quick Start

**Initialize project**

```bash
alab init
```

**Create a table schema**

```bash
alab table auth user
```

**Edit your schema**

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

**Generate migration**

```bash
alab new create_users
```

**Apply migration**

```bash
alab migrate
```

**Export types**

```bash
alab export -f all
```

## License

BSD-3-Clause
