# ⏳ Alab - Database Migration Tool

★ Language-agnostic database migration tool using JavaScript DSL

## Setup

| Command   | Description                                                  |
| --------- | ------------------------------------------------------------ |
| **init**  | Initialize project structure (schemas/, migrations/, types/) |
| **table** | Create a new table schema file                               |

## Schema Management

| Command    | Description                                  |
| ---------- | -------------------------------------------- |
| **check**  | Validate schema files and detect issues      |
| **diff**   | Show differences between schema and database |
| **schema** | Show schema at a specific migration revision |

## Migrations

| Command      | Description                                           |
| ------------ | ----------------------------------------------------- |
| **new**      | Create migration (auto-generates from schema changes) |
| **migrate**  | Apply pending migrations                              |
| **rollback** | Rollback migrations (default: 1 step)                 |
| **reset**    | Drop all tables and re-run migrations (dev only)      |
| **status**   | Show applied/pending migrations                       |
| **history**  | Show applied migrations with details                  |

## Development

| Command    | Description                                                    |
| ---------- | -------------------------------------------------------------- |
| **http**   | Start local server for live API documentation                  |
| **export** | Export schema (openapi, graphql, typescript, go, python, rust) |
| **types**  | Regenerate TypeScript definitions for IDE                      |

## Verification

| Command    | Description                                     |
| ---------- | ----------------------------------------------- |
| **verify** | Verify migration chain integrity and git status |

## Global Flags

| Flag                   | Description                              |
| ---------------------- | ---------------------------------------- |
| **-c, --config**       | Path to config file (default: alab.yaml) |
| **-d, --database-url** | Database connection URL                  |
| **-h, --help**         | Show help information                    |
| **-v, --version**      | Show version information                 |

---

_Use 'alab [command] --help' for more information about a command_
