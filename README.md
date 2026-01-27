<p align="center">
    <img src="docs/src/assets/logo.png" alt="astrola-db" width="180" />
</p>

<h1 align="center">AstrolaDB (alab)</h1>

<div align="center">
    <p>
        <strong>One schema: many languages.</strong>
    </p>
    <p>
        Define once in JavaScript then export strongly typed models.
    </p>
</div>

<p align="center">
    <a href="https://github.com/hlop3z/astroladb/actions/workflows/ci.yml"><img
            src="https://github.com/hlop3z/astroladb/actions/workflows/ci.yml/badge.svg" alt="CI"></a>
    <a href="https://goreportcard.com/report/github.com/hlop3z/astroladb"><img
            src="https://goreportcard.com/badge/github.com/hlop3z/astroladb" alt="Go Report Card"></a>
    <a href="LICENSE"><img src="https://img.shields.io/badge/license-BSD--3--Clause-blue.svg" alt="License"></a>
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

Welcome to **AstrolaDB** aka `alab`. A schema orchestration tool with
**multi-language** code generation. It produces `types` and database `migrations` then **it
gets out of the way**.

**No ORM. No framework lock-in. Just clean migrations and type exports.**

Languages: **Rust** | **Go** | **Python** | **TypeScript**

```mermaid
flowchart TD
    A[schema.js] --> B[Generated Artifacts]

    B --- D[Migrations]
    B --- E[Types]
    B --- F[API Contracts]
```

---

## Core Principles

| Principle                  | What it means                                               |
| -------------------------- | ----------------------------------------------------------- |
| **Schema-First**           | Define intent once; outputs are deterministic and diff-able |
| **Single Source of Truth** | One schema drives migrations, types, and API specs          |
| **Language-Agnostic**      | One schema → multiple language projections                  |
| **No Runtime Lock-In**     | Framework-agnostic. Generates contracts, not hidden logic   |

---

## Quick Start

**Install**

```bash
curl -fsSL https://raw.githubusercontent.com/hlop3z/astroladb/main/install.sh | sh
```

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

## Live Server

**The live server provides instant API exploration with automatic hot reloading.**

```bash
alab http
```

[See the full documentation →](https://hlop3z.github.io/astroladb/)

---

<p align="center">
  <img src="docs/gifs/workflow.gif" alt="CLI workflow Demo" width="800" />
  <img src="docs/gifs/status.gif" alt="CLI status Demo" width="800" />
  <img src="docs/gifs/http-preview.gif" alt="HTTP Demo" width="800" />
</p>

---

## License

BSD-3-Clause
