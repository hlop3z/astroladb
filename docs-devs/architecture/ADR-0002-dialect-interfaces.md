# ADR-0002: Dialect Interface Design

**Status:** Accepted
**Date:** 2026-02-13
**Commit:** 972b5bf

## Context

AstrolaDB needs to generate SQL for multiple database engines (PostgreSQL, SQLite, and potentially others). The dialect layer must:

- Map DSL types to engine-specific SQL types
- Generate DDL statements (CREATE TABLE, ALTER COLUMN, etc.)
- Handle identifier quoting and placeholder syntax
- Expose feature capability flags

## Decision

Use a composite interface with sub-interfaces for interface segregation:

```go
type Dialect interface {
    Name() string
    TypeMapper       // DSL type -> SQL type
    DDLGenerator     // SQL DDL statement generation
    SQLFormatter     // Identifier quoting, placeholders
    FeatureDetector  // Database capability flags
}
```

### Sub-interfaces

**TypeMapper** - 13 methods mapping logical types to SQL:
- `IDType()`, `StringType(length)`, `TextType()`, `IntegerType()`, `FloatType()`
- `DecimalType(precision, scale)`, `BooleanType()`, `DateType()`, `TimeType()`
- `DateTimeType()`, `UUIDType()`, `JSONType()`, `Base64Type()`, `EnumType()`

**DDLGenerator** - 14 methods for SQL statement generation:
- Table: `CreateTableSQL()`, `DropTableSQL()`, `RenameTableSQL()`
- Column: `AddColumnSQL()`, `DropColumnSQL()`, `RenameColumnSQL()`, `AlterColumnSQL()`
- Index: `CreateIndexSQL()`, `DropIndexSQL()`
- Constraints: `AddForeignKeySQL()`, `DropForeignKeySQL()`, `AddCheckSQL()`, `DropCheckSQL()`
- Escape hatch: `RawSQLFor()`

**SQLFormatter** - 2 methods:
- `QuoteIdent(name) string` - Identifier quoting (both use `"name"`)
- `Placeholder(index) string` - Postgres: `$1`, SQLite: `?`

**FeatureDetector** - 2 methods:
- `SupportsTransactionalDDL() bool`
- `SupportsIfExists() bool`

### Shared Base Helpers (`base.go`)

Common logic lives in `base.go` (~800 lines) to avoid duplication:
- `buildColumnTypeSQL()` - Shared type mapping
- `buildCreateTableSQL()` - Parameterized CREATE TABLE with dialect callbacks
- `buildColumnDefSQL()` - Column definition with configurable constraint ordering
- `buildDefaultValueSQL()` - Literal formatting
- `buildForeignKeyConstraintSQL()` - FK clause generation
- `computedExprSQL()` - Computed column expression rendering
- `truncateIdentifier()` - PostgreSQL 63-char NAMEDATALEN limit handling

### Adding a New Dialect

1. Create `newdialect.go` implementing all sub-interfaces
2. Reuse base helpers where applicable, override where needed
3. Register in `Get(name string) Dialect` factory function
4. Add test file following `postgres_test.go` / `sqlite_test.go` pattern

## Consequences

**Positive:**
- Interface segregation: consumers depend only on what they need
- DRY: shared helpers prevent dialect-specific bugs from diverging
- Testable: `interface_test.go` and `consistency_test.go` verify all implementations satisfy contracts
- Extensible: new dialects plug in via factory pattern

**Negative:**
- Large interface surface (~31 methods total) - necessary given the DDL breadth
- Base helpers use callback parameters which adds indirection

## Alternatives Considered

1. **Single flat interface** - Would force consumers to depend on everything; rejected for poor interface segregation
2. **Template-based SQL generation** - Rejected due to difficulty handling dialect-specific edge cases (e.g., PostgreSQL enum types, SQLite type affinity)
3. **External SQL builder library** - Rejected to avoid external dependency and maintain control over generated SQL
