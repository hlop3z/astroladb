<p align="center">
  <img src="docs/src/assets/logo.png" alt="astrola-db" width="180" />
</p>

<h1 align="center">AstrolaDB (alab)</h1>

<p align="center">
    <a href="https://github.com/hlop3z/astroladb/actions/workflows/ci.yml"><img src="https://github.com/hlop3z/astroladb/actions/workflows/ci.yml/badge.svg" alt="CI" /></a>
    <a href="https://goreportcard.com/report/github.com/hlop3z/astroladb"><img src="https://goreportcard.com/badge/github.com/hlop3z/astroladb" alt="Go Report Card" /></a>
    <a href="https://github.com/hlop3z/astroladb/releases"><img src="https://img.shields.io/github/v/release/hlop3z/astroladb?color=teal" alt="Release" /></a>
    <img src="https://img.shields.io/badge/status-preview-indigo" alt="Preview" />
</p>

<p align="center">
  <strong>One schema: many outputs.</strong>
</p>

---

**Stop writing boilerplate.** Define your data model once in JavaScript. Use **custom generators** to produce REST APIs in **FastAPI**, **Go Chi**, **Rust Axum**, or **tRPC**. Export types, SQL migrations, and OpenAPI specs directly from the core engine.

## One Schema

```js
// schemas/auth/user.js
export default table({
  id: col.id(),
  username: col.username().unique(),
  email: col.email().unique(),
  password: col.password_hash(),
  is_active: col.flag(true),
}).timestamps();
```

## Many Outputs

AstrolaDB turns a single schema into multiple artifacts. Some built-in, others produced via custom generators.

---

### Built-in Outputs (Core Engine)

Deterministic outputs provided natively by AstrolaDB.

| Output             | Languages / Formats          |
| ------------------ | ---------------------------- |
| **Type Exports**   | Rust, Go, Python, TypeScript |
| **SQL Migrations** | PostgreSQL, SQLite           |
| **API Specs**      | OpenAPI, GraphQL             |

These features are **first-party** and maintained as part of the **core engine**.

---

### Generator-Powered Outputs

Produced via sandboxed JavaScript generators you write or share.

| What You Can Generate   | Examples / Targets                          |
| ----------------------- | ------------------------------------------- |
| **Complete APIs**       | FastAPI, Chi, Axum, tRPC                    |
| **Infra Configs**       | Terraform, Docker, Helm                     |
| **SDKs & Clients**      | Language SDKs, RPC clients                  |
| **Tooling**             | CLIs, test suites, documentation            |
| **Anything Text-Based** | Any framework or format expressible as code |

Generators transform the schema object into files: giving you full control over structure, frameworks, and conventions.

---

> **Core outputs** are built-in and versioned with AstrolaDB.
> **Generator outputs** are user-extensible and defined as JavaScript functions.

---

## Documentation

- **[Quick Start Guide](https://hlop3z.github.io/astroladb/quick-start/)** — Get started in 5 minutes
- **[Tutorial](https://hlop3z.github.io/astroladb/tutorial/first-project/)** — Build your first project
- **[Generator Guide](https://hlop3z.github.io/astroladb/advanced_users/generators/)** — Write custom generators
- **[Migration Reference](https://hlop3z.github.io/astroladb/migrations/overview/)** — Schema evolution
- **[Field Types](https://hlop3z.github.io/astroladb/cols/semantics/)** — All available column types
- **[CLI Commands](https://hlop3z.github.io/astroladb/commands/)** — Complete command reference

---

## Download Release

[![Windows](https://img.shields.io/badge/Windows-Download-blue?logo=windows)](https://github.com/hlop3z/astroladb/releases/latest/download/alab-windows-amd64.zip)
[![Mac (Intel)](https://img.shields.io/badge/Mac%20Intel-Download-black?logo=apple)](https://github.com/hlop3z/astroladb/releases/latest/download/alab-darwin-amd64.tar.gz)
[![Mac (Apple Silicon)](https://img.shields.io/badge/Mac%20ARM-Download-black?logo=apple)](https://github.com/hlop3z/astroladb/releases/latest/download/alab-darwin-arm64.tar.gz)
[![Linux (amd64)](https://img.shields.io/badge/Linux%20amd64-Download-green?logo=linux)](https://github.com/hlop3z/astroladb/releases/latest/download/alab-linux-amd64.tar.gz)
[![Linux (arm64)](https://img.shields.io/badge/Linux%20ARM-Download-green?logo=linux)](https://github.com/hlop3z/astroladb/releases/latest/download/alab-linux-arm64.tar.gz)

**Or install via script:**

```bash
curl -fsSL https://raw.githubusercontent.com/hlop3z/astroladb/main/install.sh | sh
```

---

## See It In Action

**From schema to running API using an example generator:**

```bash
# 1. Initialize project
alab init

# 2. Create your schema
alab table auth user

# 3. Download an example generator
alab gen add https://raw.githubusercontent.com/hlop3z/astroladb/main/examples/generators/generators/fastapi.js

# 4. Run the generator
alab gen run generators/fastapi -o ./backend

# 5. Run it
cd backend
uv add fastapi[standard]
uv run fastapi dev main.py
# → http://localhost:8000/docs
```

**What the generator produced:**

```
backend/
├── auth/
│   ├── models.py      # Pydantic models with validation
│   ├── router.py      # CRUD endpoints (list, get, create, update, delete)
│   └── __init__.py
└── main.py            # FastAPI app with Swagger docs
```

**All from that one schema file.** The generator handles the boilerplate.

[View all generator examples →](https://github.com/hlop3z/astroladb/tree/main/examples/generators)

---

## 💎 Example Generators

Example generators demonstrating what's possible with the platform:

| Framework   | Language   | What It Generates                            | Example Output                                                                                           |
| ----------- | ---------- | -------------------------------------------- | -------------------------------------------------------------------------------------------------------- |
| **FastAPI** | Python     | Pydantic models + CRUD routers + FastAPI app | [View Code](https://github.com/hlop3z/astroladb/tree/main/examples/generators/generated/python-fastapi)  |
| **Chi**     | Go         | Structs + handlers + Chi router              | [View Code](https://github.com/hlop3z/astroladb/tree/main/examples/generators/generated/go-chi)          |
| **Axum**    | Rust       | Structs + handlers + Axum server             | [View Code](https://github.com/hlop3z/astroladb/tree/main/examples/generators/generated/rust-axum)       |
| **tRPC**    | TypeScript | Zod schemas + type-safe RPC routers          | [View Code](https://github.com/hlop3z/astroladb/tree/main/examples/generators/generated/typescript-trpc) |

These are **reference implementations** you can use or modify. Or [write your own generator](https://hlop3z.github.io/astroladb/advanced_users/generators/) in JavaScript to produce any text output.

---

## ⚡ Core Features

The platform provides built-in schema orchestration and a generator runtime:

| Feature                   | Description                                                      |
| ------------------------- | ---------------------------------------------------------------- |
| **Single Binary**         | ~9 MB static executable. Zero dependencies.                      |
| **Schema Engine**         | Type-safe JavaScript DSL for data modeling.                      |
| **Auto Migrations**       | SQL migrations generated from schema changes.                    |
| **Multi-Language Export** | Built-in types for Rust, Go, Python, TypeScript.                 |
| **Generator Runtime**     | Sandboxed JS execution for custom code generation.               |
| **No Runtime Lock-in**    | Works without Node.js, JVM, or Python runtime.                   |
| **Live Development**      | Built-in HTTP server (`alab live`) with hot reload.              |
| **OpenAPI Ready**         | Exports `openapi.json` for integration with 25+ languages.       |
| **Namespace Support**     | Logical grouping (e.g., `auth.user`) prevents naming collisions. |

---

## Generator Platform = Extensibility

AstrolaDB is **not just** a schema tool. It's a meta-programming platform:

```
Schema Engine
   ↓
Normalized Schema Object
   ↓
Generator Runtime (sandboxed JS)
   ↓
Arbitrary Text Outputs
```

**You can generate:**

- REST APIs (FastAPI, Chi, Axum, Express)
- GraphQL servers
- tRPC routers
- SDK clients
- Terraform configs
- Docker Compose files
- Kubernetes manifests
- Test suites
- Documentation sites
- CLI tools

Anything that can be derived from your schema structure.

---

## AstrolaDB vs. Writing It Manually

| Task                    | Manual Approach                     | With AstrolaDB             |
| ----------------------- | ----------------------------------- | -------------------------- |
| **Define models**       | Write in each language separately   | Once, in schema            |
| **API endpoints**       | ~500 lines of boilerplate per model | Via generators             |
| **Database migrations** | Handwrite SQL                       | Auto-generated (core)      |
| **OpenAPI docs**        | Manual annotations                  | Included (core)            |
| **Type safety**         | Manually keep languages in sync     | Always synchronized (core) |
| **Validation**          | Write validators in each language   | Generated from schema      |

---

## 🎸 Complete Workflow

### 1. Initialize Project

```bash
alab init
```

Creates:

```
project/
├── alab.yaml        # Configuration
├── schemas/         # Your schema definitions
├── migrations/      # Generated SQL migrations
└── types/           # TypeScript definitions for IDE support
```

### 2. Define Your Schema

```bash
alab table auth user
```

Edit `schemas/auth/user.js`:

```js
export default table({
  id: col.id(),
  email: col.email().unique(),
  username: col.username().unique(),
  password: col.password_hash(),
  role: col.enum(["admin", "editor", "viewer"]).default("viewer"),
  is_active: col.flag(true),
}).timestamps();
```

### 3. Generate & Apply Migrations

```bash
# Generate migration
alab new create_users

# Preview SQL
alab migrate --dry

# Apply to database
alab migrate
```

### 4. Export Types

```bash
# Export to all languages
alab export -f all

# Or specific language
alab export -f typescript
alab export -f python
alab export -f go
alab export -f rust
```

### 5. Use Generators (Optional)

```bash
# Download example generator
alab gen add https://raw.githubusercontent.com/hlop3z/astroladb/main/examples/generators/generators/fastapi.js

# Run generator
alab gen run generators/fastapi -o ./backend
```

---

## Who Is This For?

- **Platform engineers** building code generation pipelines
- **Polyglot developers** maintaining services in multiple languages
- **Meta-programmers** who want deterministic code generation
- **Startups** needing to move fast without sacrificing type safety
- **Solo developers** prototyping full-stack applications
- **API-first teams** who want to skip repetitive boilerplate

---

## Live Development Mode

Instant schema exploration with automatic hot reloading:

```bash
alab live
```

Opens an interactive HTTP server where you can explore your schema, test the normalized object, and see changes in real-time.

---

<p align="center">
  <img src="docs/gifs/http-preview.gif" alt="HTTP Demo" width="800" />
  <img src="docs/gifs/dx.gif" alt="DX Demo" width="800" />
  <img src="docs/gifs/status.gif" alt="CLI status Demo" width="800" />
</p>

---

## License

BSD-3-Clause

---

## Stability Status

### Core Engine — Stable

- **Schema DSL**
- **Type Exports** (Rust, Go, Python, TypeScript)
- **OpenAPI Export**
- **Generator Runtime**
- Does **not** modify your database directly

### Migrations — Preview Status

- Migration engine is **actively evolving**
- APIs may introduce breaking changes
- **Recommendation**: Always test thoroughly in staging before production

---

<p align="center">
  <sub>Built with ❤️ by hlop3z</sub>
</p>
