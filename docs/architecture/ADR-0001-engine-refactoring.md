# ADR-0001: Engine Refactoring into Diff + Runner Packages

**Status:** Accepted
**Date:** 2026-02-13
**Commit:** b138fe6

## Context

The migration engine was a monolithic `internal/engine` package containing all migration logic: schema diffing, rename detection, linting, execution, locking, backfill, and version management. As the feature set grew, this created:

- Tight coupling between planning logic and execution logic
- Difficulty testing diff algorithms without a database connection
- No clear ownership boundary for new contributors

## Decision

Split `internal/engine` into focused sub-packages:

```
internal/engine/
  diff/       Schema comparison and operation generation
  runner/     Migration execution, locking, backfill
  state/      Shared utilities (topo sort, merge, replay, validate)
  migration.go  Core types (Migration, Plan, Direction)
  schema.go     Schema facade
```

### Diff Package

Handles all schema comparison with no database dependency:

- `Diff(old, new *Schema) ([]Operation, error)` - core diffing
- Rename detection via Jaro-Winkler similarity scoring (weighted: type 40%, name 30%, constraints 20%, position 10%)
- Topological sorting of operations respecting FK dependencies
- Migration safety linting

### Runner Package

Handles execution against a live database:

- `Run(ctx, plan) error` - execute migrations
- `RunWithLock(ctx, plan, timeout) error` - distributed locking
- Backfill support for NOT NULL column additions
- Version tracking (applied migrations table)
- Transaction support when dialect supports transactional DDL
- Dry-run mode for SQL preview

### State Package

Shared utilities consumed by both diff and runner:

- Topological sorting (Kahn's algorithm)
- Schema merging
- Operation replay
- Table validation

## Consequences

**Positive:**
- Diff logic is testable without any database
- Clear separation of concerns (planning vs execution)
- Execution strategy can be swapped without touching diff logic
- Each package has a focused, reviewable API surface

**Negative:**
- More packages to navigate for newcomers
- Some types (Migration, Plan) live in the parent `engine` package as shared types

## Alternatives Considered

1. **Keep monolithic package** - Rejected due to growing complexity and test difficulty
2. **Separate Go modules** - Over-engineered for an internal package split; standard packages suffice
