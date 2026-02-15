# AstrolaDB IR Specification v1.0.0

**Version**: 1.0.0
**Last Updated**: 2026-02-14
**Status**: Stable

---

## Table of Contents

1. [Overview](#overview)
2. [IR Version](#ir-version)
3. [Core Types](#core-types)
4. [Type System](#type-system)
5. [Operation Types](#operation-types)
6. [Validation Rules](#validation-rules)
7. [Versioning Policy](#versioning-policy)
8. [Migration Path](#migration-path)

---

## Overview

The AstrolaDB Intermediate Representation (IR) is a **language-agnostic schema representation** used for:

- **Schema Definition**: Declarative table and column definitions
- **Migration Generation**: Automatic diff-based migration creation
- **Code Generation**: Type-safe code generation for multiple languages
- **Export Systems**: OpenAPI, TypeScript, Go, Python, Rust, GraphQL exports

### Design Principles

1. **Language-Agnostic**: No database-specific or programming language-specific types
2. **JS-Safe**: All types are safe in JavaScript (no int64, float64, auto-increment)
3. **Portable**: Works across PostgreSQL and SQLite
4. **Validated**: All definitions are validated before use
5. **Deterministic**: Same input always produces the same output

### Architecture

```
User Schema (JS)
    ↓
Sandbox Runtime (Goja VM)
    ↓ Evaluates and normalizes
IR (internal/ast/)
    ↓ Used by:
    ├─ Migration Engine (internal/engine/)
    ├─ SQL Generators (internal/dialect/)
    ├─ Export Systems (pkg/astroladb/export_*.go)
    └─ Code Generators (examples/generators/)
```

---

## IR Version

### Version Format

The IR follows **semantic versioning** (MAJOR.MINOR.PATCH):

- **MAJOR (1.x.x)**: Breaking changes (field removal, type changes, required field additions)
- **MINOR (x.1.x)**: Additive changes (new optional fields, new operation types, new portable types)
- **PATCH (x.x.1)**: Bug fixes, documentation updates, internal refactoring

### Current Version

**1.0.0** - Initial stable release (2026-02-14)

### Version Detection

Generators receive the IR version in the schema object:

```typescript
export interface GeneratorSchema {
  readonly version: string;      // e.g., "1.0.0"
  readonly generatedAt: string;  // ISO 8601 timestamp
  readonly models: { ... };
  readonly tables: [ ... ];
}
```

**Example Version Check**:

```javascript
export default gen((schema) => {
  // Check minimum required version
  if (schema.version < "1.0.0") {
    throw new Error("This generator requires schema version >= 1.0.0");
  }

  // Generator implementation
  return render(files);
});
```

---

## Core Types

### TableDef

Represents a complete table definition with columns, indexes, and constraints.

**Go Definition** (`internal/ast/table.go`):

```go
type TableDef struct {
    Namespace   string           // Logical grouping (e.g., "auth", "billing")
    Name        string           // Table name (snake_case)
    Columns     []*ColumnDef     // Column definitions in order
    Indexes     []*IndexDef      // Index definitions
    ForeignKeys []*ForeignKeyDef // Foreign key constraints
    Checks      []*CheckDef      // CHECK constraints
    Docs        string           // Documentation
    Deprecated  string           // Deprecation notice
    Auditable   bool             // Has created_by/updated_by columns
    SortBy      []string         // Default ordering
    Searchable  []string         // Columns for fulltext search
    Filterable  []string         // Columns allowed in WHERE
    SourceFile  string           // Source file path (for errors)
}
```

**JSON Schema**:

```json
{
  "type": "object",
  "properties": {
    "namespace": {
      "type": "string",
      "pattern": "^[a-z_][a-z0-9_]*$",
      "description": "Logical grouping (e.g., 'auth', 'billing')"
    },
    "name": {
      "type": "string",
      "pattern": "^[a-z_][a-z0-9_]*$",
      "description": "Table name in snake_case"
    },
    "columns": {
      "type": "array",
      "items": { "$ref": "#/definitions/ColumnDef" },
      "minItems": 1,
      "description": "Column definitions"
    },
    "indexes": {
      "type": "array",
      "items": { "$ref": "#/definitions/IndexDef" },
      "description": "Index definitions"
    },
    "foreignKeys": {
      "type": "array",
      "items": { "$ref": "#/definitions/ForeignKeyDef" },
      "description": "Foreign key constraints"
    },
    "checks": {
      "type": "array",
      "items": { "$ref": "#/definitions/CheckDef" },
      "description": "CHECK constraints"
    },
    "docs": {
      "type": "string",
      "description": "Documentation for the table"
    },
    "sourceFile": {
      "type": "string",
      "description": "Source file path for error reporting"
    }
  },
  "required": ["name", "columns"]
}
```

**Methods**:

- `FullName()` → `namespace_tablename` (SQL identifier)
- `QualifiedName()` → `namespace.table` (reference format)
- `GetColumn(name)` → Find column by name
- `PrimaryKey()` → Get primary key column
- `Validate()` → Validate the table definition

---

### ColumnDef

Represents a column definition with type, constraints, validation, and metadata.

**Go Definition** (`internal/ast/table.go`):

```go
type ColumnDef struct {
    Name        string     // Column name (snake_case)
    Type        string     // Type name (id, string, integer, etc.)
    TypeArgs    []any      // Type arguments (e.g., [255] for string(255))

    // Source location (for error reporting)
    Line        int        // Line number in source file
    Column      int        // Column number in source file
    Source      string     // The source code line

    // Nullability
    Nullable    bool       // Allows NULL
    NullableSet bool       // Whether Nullable was explicitly set

    // Constraints
    Unique      bool       // UNIQUE constraint
    PrimaryKey  bool       // PRIMARY KEY constraint

    // Default values
    Default       any      // Default value (Go value or SQLExpr)
    DefaultSet    bool     // Whether Default was explicitly set
    ServerDefault string   // Raw SQL expression (deprecated: use SQLExpr)

    // Reference (for belongs_to, one_to_one)
    Reference   *Reference // Foreign key reference

    // Migration-only
    Backfill    any        // Value for existing rows when adding NOT NULL column
    BackfillSet bool       // Whether Backfill was explicitly set

    // Validation constraints
    Min     *int           // Minimum (string: minLength, number: minimum)
    Max     *int           // Maximum (string: maxLength, number: maximum)
    Pattern string         // Regex pattern for validation
    Format  string         // OpenAPI format (email, uri, uuid, etc.)

    // Documentation
    Docs       string      // Description
    Deprecated string      // Deprecation notice

    // Computed columns
    Computed any           // fn.* expression or raw SQL
    Virtual  bool          // VIRTUAL (not stored) or app-only
}
```

**JSON Schema**:

```json
{
  "type": "object",
  "properties": {
    "name": {
      "type": "string",
      "pattern": "^[a-z_][a-z0-9_]*$",
      "description": "Column name in snake_case"
    },
    "type": {
      "type": "string",
      "enum": [
        "id",
        "uuid",
        "string",
        "text",
        "integer",
        "float",
        "decimal",
        "boolean",
        "date",
        "time",
        "datetime",
        "json",
        "base64",
        "enum",
        "computed"
      ],
      "description": "Type name from portable types registry"
    },
    "typeArgs": {
      "type": "array",
      "description": "Type arguments (e.g., [255] for string(255), [10, 2] for decimal(10,2))"
    },
    "nullable": {
      "type": "boolean",
      "default": false,
      "description": "Whether column allows NULL"
    },
    "unique": {
      "type": "boolean",
      "default": false,
      "description": "Whether column has UNIQUE constraint"
    },
    "primaryKey": {
      "type": "boolean",
      "default": false,
      "description": "Whether column is PRIMARY KEY"
    },
    "default": {
      "description": "Default value for new rows"
    },
    "min": {
      "type": "integer",
      "description": "Minimum value/length"
    },
    "max": {
      "type": "integer",
      "description": "Maximum value/length"
    },
    "pattern": {
      "type": "string",
      "description": "Regex pattern for validation"
    },
    "format": {
      "type": "string",
      "description": "OpenAPI format hint (email, uri, uuid, etc.)"
    },
    "docs": {
      "type": "string",
      "description": "Column documentation"
    }
  },
  "required": ["name", "type"]
}
```

**Methods**:

- `Validate()` → Validate the column definition
- `IsNullable()` → Check if column allows NULL
- `HasDefault()` → Check if column has a default value
- `GetDefaultSQL()` → Get SQL representation of default
- `EnumValues()` → Extract enum values from TypeArgs

---

### IndexDef

Represents an index definition.

**Go Definition** (`internal/ast/table.go`):

```go
type IndexDef struct {
    Name    string   // Index name (auto-generated if empty)
    Columns []string // Columns to index (in order)
    Unique  bool     // UNIQUE index
    Where   string   // Partial index condition (if supported)
}
```

**JSON Schema**:

```json
{
  "type": "object",
  "properties": {
    "name": {
      "type": "string",
      "pattern": "^[a-z_][a-z0-9_]*$",
      "description": "Index name (auto-generated if empty)"
    },
    "columns": {
      "type": "array",
      "items": { "type": "string", "pattern": "^[a-z_][a-z0-9_]*$" },
      "minItems": 1,
      "description": "Columns to index"
    },
    "unique": {
      "type": "boolean",
      "default": false,
      "description": "Whether index is UNIQUE"
    },
    "where": {
      "type": "string",
      "description": "Partial index condition (WHERE clause)"
    }
  },
  "required": ["columns"]
}
```

---

### ForeignKeyDef

Represents a foreign key constraint.

**Go Definition** (`internal/ast/table.go`):

```go
type ForeignKeyDef struct {
    Name       string   // Constraint name (auto-generated if empty)
    Columns    []string // Local columns
    RefTable   string   // Referenced table (namespace_tablename format)
    RefColumns []string // Referenced columns (usually ["id"])
    OnDelete   string   // CASCADE, SET NULL, RESTRICT, NO ACTION
    OnUpdate   string   // CASCADE, SET NULL, RESTRICT, NO ACTION
}
```

**JSON Schema**:

```json
{
  "type": "object",
  "properties": {
    "name": {
      "type": "string",
      "pattern": "^[a-z_][a-z0-9_]*$"
    },
    "columns": {
      "type": "array",
      "items": { "type": "string" },
      "minItems": 1
    },
    "refTable": {
      "type": "string",
      "pattern": "^[a-z_][a-z0-9_]*$"
    },
    "refColumns": {
      "type": "array",
      "items": { "type": "string" },
      "minItems": 1
    },
    "onDelete": {
      "type": "string",
      "enum": [
        "",
        "CASCADE",
        "SET NULL",
        "SET DEFAULT",
        "RESTRICT",
        "NO ACTION"
      ],
      "default": ""
    },
    "onUpdate": {
      "type": "string",
      "enum": [
        "",
        "CASCADE",
        "SET NULL",
        "SET DEFAULT",
        "RESTRICT",
        "NO ACTION"
      ],
      "default": ""
    }
  },
  "required": ["columns", "refTable", "refColumns"]
}
```

**Validation Rules**:

- Column count must match referenced column count
- Referenced table must exist
- Referenced columns must exist in referenced table

---

### CheckDef

Represents a CHECK constraint.

**Go Definition** (`internal/ast/table.go`):

```go
type CheckDef struct {
    Name       string // Constraint name (auto-generated if empty)
    Expression string // SQL expression for the check
}
```

**JSON Schema**:

```json
{
  "type": "object",
  "properties": {
    "name": {
      "type": "string",
      "pattern": "^[a-z_][a-z0-9_]*$"
    },
    "expression": {
      "type": "string",
      "description": "SQL expression (validated for safety)"
    }
  },
  "required": ["expression"]
}
```

**Safety Validation**:

CHECK expressions are validated to prevent SQL injection:

- No semicolons (`;`)
- No double-dashes (`--`) for comments
- No DDL keywords (DROP, ALTER, CREATE, GRANT, etc.)
- No DML keywords (INSERT, UPDATE, DELETE, etc.)

---

### Reference

Represents a foreign key reference from a column (used by `belongs_to()` and `one_to_one()`).

**Go Definition** (`internal/ast/table.go`):

```go
type Reference struct {
    Table    string // Referenced table ("ns.table", ".table", or "table")
    Column   string // Referenced column (default: "id")
    OnDelete string // CASCADE, SET NULL, RESTRICT, NO ACTION
    OnUpdate string // CASCADE, SET NULL, RESTRICT, NO ACTION
}
```

**JSON Schema**:

```json
{
  "type": "object",
  "properties": {
    "table": {
      "type": "string",
      "description": "Referenced table (qualified name or relative)"
    },
    "column": {
      "type": "string",
      "default": "id",
      "description": "Referenced column"
    },
    "onDelete": {
      "type": "string",
      "enum": [
        "",
        "CASCADE",
        "SET NULL",
        "SET DEFAULT",
        "RESTRICT",
        "NO ACTION"
      ]
    },
    "onUpdate": {
      "type": "string",
      "enum": [
        "",
        "CASCADE",
        "SET NULL",
        "SET DEFAULT",
        "RESTRICT",
        "NO ACTION"
      ]
    }
  },
  "required": ["table"]
}
```

---

## Type System

### Portable Types Registry

All types are registered in `internal/types/types.go` and mapped to:

- **JavaScript DSL names** (snake_case)
- **Go types** (for Go code generation)
- **TypeScript types** (for TypeScript code generation)
- **OpenAPI types** (for OpenAPI spec generation)
- **SQL types** (per dialect: PostgreSQL, SQLite)

### TypeDef Structure

```go
type TypeDef struct {
    Name     string      // Internal name (e.g., "string", "integer")
    JSName   string      // JS DSL name (e.g., "string", "datetime")
    GoType   string      // Go type (e.g., "string", "int32")
    TSType   string      // TypeScript type (e.g., "string", "number")
    OpenAPI  OpenAPIType // OpenAPI type info
    SQLTypes SQLTypeMap  // Database-specific SQL types
    HasArgs  bool        // Whether type accepts arguments
    MinArgs  int         // Minimum number of arguments
    MaxArgs  int         // Maximum number of arguments
}

type SQLTypeMap struct {
    Postgres string // PostgreSQL type (e.g., "VARCHAR(%d)", "TIMESTAMPTZ")
    SQLite   string // SQLite type (e.g., "TEXT", "INTEGER")
}

type OpenAPIType struct {
    Type   string // JSON Schema type: string, integer, number, boolean
    Format string // Format hint: uuid, date-time, email, uri, etc.
}
```

### Built-in Types

#### id

UUID primary key (auto-generated).

```json
{
  "name": "id",
  "jsName": "id",
  "goType": "string",
  "tsType": "string",
  "openAPI": { "type": "string", "format": "uuid" },
  "sqlTypes": {
    "postgres": "UUID DEFAULT gen_random_uuid()",
    "sqlite": "TEXT"
  },
  "hasArgs": false
}
```

#### string

Variable-length string with maximum length.

```json
{
  "name": "string",
  "jsName": "string",
  "goType": "string",
  "tsType": "string",
  "openAPI": { "type": "string" },
  "sqlTypes": {
    "postgres": "VARCHAR(%d)",
    "sqlite": "TEXT"
  },
  "hasArgs": true,
  "minArgs": 1,
  "maxArgs": 1
}
```

**Usage**: `col.string(255)` → `VARCHAR(255)` in PostgreSQL, `TEXT` in SQLite

#### text

Unlimited length text.

```json
{
  "name": "text",
  "jsName": "text",
  "goType": "string",
  "tsType": "string",
  "openAPI": { "type": "string" },
  "sqlTypes": {
    "postgres": "TEXT",
    "sqlite": "TEXT"
  },
  "hasArgs": false
}
```

#### integer

32-bit signed integer (JS-safe: fits in JavaScript `number`).

```json
{
  "name": "integer",
  "jsName": "integer",
  "goType": "int32",
  "tsType": "number",
  "openAPI": { "type": "integer", "format": "int32" },
  "sqlTypes": {
    "postgres": "INTEGER",
    "sqlite": "INTEGER"
  },
  "hasArgs": false
}
```

**Why 32-bit?** JavaScript `number` is a 64-bit float (IEEE 754) with only 53 bits of integer precision. Values above 2^53 lose precision. 32-bit integers are always safe.

#### float

32-bit floating point (JS-safe).

```json
{
  "name": "float",
  "jsName": "float",
  "goType": "float32",
  "tsType": "number",
  "openAPI": { "type": "number", "format": "float" },
  "sqlTypes": {
    "postgres": "REAL",
    "sqlite": "REAL"
  },
  "hasArgs": false
}
```

#### decimal

Arbitrary-precision decimal (serialized as string to preserve precision in JavaScript).

```json
{
  "name": "decimal",
  "jsName": "decimal",
  "goType": "string",
  "tsType": "string",
  "openAPI": { "type": "string", "format": "decimal" },
  "sqlTypes": {
    "postgres": "NUMERIC(%d,%d)",
    "sqlite": "TEXT"
  },
  "hasArgs": true,
  "minArgs": 2,
  "maxArgs": 2
}
```

**Usage**: `col.decimal(10, 2)` → `NUMERIC(10,2)` (precision=10, scale=2)

**Why string?** Decimal values are serialized as strings to prevent precision loss in JavaScript.

#### boolean

Boolean (true/false).

```json
{
  "name": "boolean",
  "jsName": "boolean",
  "goType": "bool",
  "tsType": "boolean",
  "openAPI": { "type": "boolean" },
  "sqlTypes": {
    "postgres": "BOOLEAN",
    "sqlite": "INTEGER"
  },
  "hasArgs": false
}
```

#### date

Date only (YYYY-MM-DD), serialized as ISO 8601 string.

```json
{
  "name": "date",
  "jsName": "date",
  "goType": "string",
  "tsType": "string",
  "openAPI": { "type": "string", "format": "date" },
  "sqlTypes": {
    "postgres": "DATE",
    "sqlite": "TEXT"
  },
  "hasArgs": false
}
```

#### time

Time only (HH:MM:SS), serialized as RFC 3339 string.

```json
{
  "name": "time",
  "jsName": "time",
  "goType": "string",
  "tsType": "string",
  "openAPI": { "type": "string", "format": "time" },
  "sqlTypes": {
    "postgres": "TIME",
    "sqlite": "TEXT"
  },
  "hasArgs": false
}
```

#### datetime

Timestamp with timezone, stored in UTC, serialized as RFC 3339 string.

```json
{
  "name": "datetime",
  "jsName": "datetime",
  "goType": "string",
  "tsType": "string",
  "openAPI": { "type": "string", "format": "date-time" },
  "sqlTypes": {
    "postgres": "TIMESTAMPTZ",
    "sqlite": "TEXT"
  },
  "hasArgs": false
}
```

**Storage**:

- PostgreSQL: `TIMESTAMPTZ` (stored in UTC, displayed in connection timezone)
- SQLite: `TEXT` in ISO 8601 format

#### uuid

UUID value (not primary key).

```json
{
  "name": "uuid",
  "jsName": "uuid",
  "goType": "string",
  "tsType": "string",
  "openAPI": { "type": "string", "format": "uuid" },
  "sqlTypes": {
    "postgres": "UUID",
    "sqlite": "TEXT"
  },
  "hasArgs": false
}
```

#### json

JSON/JSONB value (native JavaScript object).

```json
{
  "name": "json",
  "jsName": "json",
  "goType": "any",
  "tsType": "Record<string, unknown>",
  "openAPI": { "type": "object" },
  "sqlTypes": {
    "postgres": "JSONB",
    "sqlite": "TEXT"
  },
  "hasArgs": false
}
```

#### base64

Binary data (serialized as base64 string).

```json
{
  "name": "base64",
  "jsName": "base64",
  "goType": "string",
  "tsType": "string",
  "openAPI": { "type": "string", "format": "byte" },
  "sqlTypes": {
    "postgres": "BYTEA",
    "sqlite": "BLOB"
  },
  "hasArgs": false
}
```

#### enum

Enumerated values (stored as string).

```json
{
  "name": "enum",
  "jsName": "enum",
  "goType": "string",
  "tsType": "string",
  "openAPI": { "type": "string" },
  "sqlTypes": {
    "postgres": "",
    "sqlite": "TEXT"
  },
  "hasArgs": true,
  "minArgs": 1,
  "maxArgs": 100
}
```

**Usage**: `col.enum(['active', 'inactive', 'pending'])`

**PostgreSQL**: Uses named ENUM type (e.g., `CREATE TYPE status_enum AS ENUM ('active', 'inactive', 'pending')`)

**SQLite**: Uses TEXT with CHECK constraint

### Forbidden Types

These types are **NOT allowed** because they break in JavaScript:

| Type        | Reason                                             | Alternative                                               |
| ----------- | -------------------------------------------------- | --------------------------------------------------------- |
| `bigint`    | JavaScript loses precision for values > 2^53       | Use `integer` (32-bit) or `decimal` (arbitrary precision) |
| `double`    | JavaScript has precision issues with 64-bit floats | Use `float` (32-bit) or `decimal` (arbitrary precision)   |
| `serial`    | Auto-increment IDs are not allowed                 | Use `id` (UUID) for primary keys                          |
| `bigserial` | Auto-increment IDs are not allowed                 | Use `id` (UUID) for primary keys                          |

---

## Operation Types

Operations represent atomic schema changes that can be serialized to SQL for any supported dialect.

**Location**: `internal/ast/types.go`

```go
type OpType int

const (
    OpCreateTable    OpType = iota  // Create new table
    OpDropTable                      // Drop existing table
    OpRenameTable                    // Rename table
    OpAddColumn                      // Add column to table
    OpDropColumn                     // Drop column from table
    OpRenameColumn                   // Rename column
    OpAlterColumn                    // Modify column type/constraints
    OpCreateIndex                    // Create index
    OpDropIndex                      // Drop index
    OpAddForeignKey                  // Add FK constraint
    OpDropForeignKey                 // Drop FK constraint
    OpAddCheck                       // Add CHECK constraint
    OpDropCheck                      // Drop CHECK constraint
    OpRawSQL                         // Raw SQL escape hatch
)
```

### Operation Definitions

Each operation type has a corresponding struct in `internal/ast/operation.go`:

- `CreateTable` - Create table with columns, indexes, constraints
- `DropTable` - Drop table
- `RenameTable` - Rename table
- `AddColumn` - Add column to existing table
- `DropColumn` - Drop column from table
- `RenameColumn` - Rename column
- `AlterColumn` - Modify column type, nullability, or default
- `CreateIndex` - Create index on columns
- `DropIndex` - Drop index
- `AddForeignKey` - Add FK constraint
- `DropForeignKey` - Drop FK constraint
- `AddCheck` - Add CHECK constraint
- `DropCheck` - Drop CHECK constraint
- `RawSQL` - Execute raw SQL (escape hatch)

---

## Validation Rules

### Identifier Validation

All identifiers (table names, column names, index names) must match:

```regex
^[a-z_][a-z0-9_]*$
```

**Rules**:

- Must start with lowercase letter or underscore
- Can contain lowercase letters, digits, and underscores
- Must be lowercase (snake_case)

**Valid**: `user`, `auth_user`, `created_at`, `_internal`

**Invalid**: `User`, `auth-user`, `user.name`, `123user`

### Qualified Name Validation

Qualified names (references) can be:

- `table` - Single identifier
- `namespace.table` - Two identifiers separated by dot

### SQL Expression Validation

SQL expressions (CHECK constraints, WHERE clauses, defaults) are validated for safety:

**Forbidden Patterns**:

- Semicolons (`;`) - Prevents multiple statements
- Double-dashes (`--`) - Prevents comment-based injection
- DDL keywords: `DROP`, `ALTER`, `CREATE`, `GRANT`, `REVOKE`, `TRUNCATE`
- DML keywords: `INSERT`, `UPDATE`, `DELETE`, `COPY`
- Dangerous functions: `EXEC`, `EXECUTE`, `UNION`, `pg_read_file`, `lo_import`, `pg_sleep`

**Valid Expressions**:

```sql
price > 0
status IN ('active', 'pending')
created_at >= NOW()
email LIKE '%@%.%'
```

**Invalid Expressions**:

```sql
price > 0; DROP TABLE users;  -- Semicolon
-- comment
DROP TABLE users
```

### Foreign Key Validation

Foreign keys must satisfy:

- Column count matches referenced column count
- Referenced table exists
- Referenced columns exist in referenced table
- OnDelete/OnUpdate actions are valid: `CASCADE`, `SET NULL`, `SET DEFAULT`, `RESTRICT`, `NO ACTION`

---

## Versioning Policy

### Version Numbers

The IR follows **semantic versioning** (MAJOR.MINOR.PATCH):

#### MAJOR Version (Breaking Changes)

Increment when making incompatible API changes:

- Field removal from IR types
- Type changes (e.g., string → number)
- Operation type removal
- Required field additions
- Validation rule changes that reject previously valid schemas

**Example**: Removing `ServerDefault` field from `ColumnDef` would be v2.0.0

#### MINOR Version (Additive Changes)

Increment when adding functionality in a backward-compatible manner:

- New optional fields in IR types
- New operation types
- New portable types in the type registry
- New generator helper functions
- New export formats

**Example**: Adding `Auditable` field to `TableDef` would be v1.1.0

#### PATCH Version (Non-Breaking Fixes)

Increment when making backward-compatible bug fixes:

- Bug fixes in validation
- Documentation updates
- Internal refactoring
- Performance improvements

**Example**: Fixing a bug in foreign key validation would be v1.0.1

### Deprecation Process

1. **Announce** (vX.Y.0): Mark feature as deprecated in release notes
   - Add deprecation notice to documentation
   - No runtime changes yet

2. **Warn** (vX.Y.0 - vN.0.0): Add runtime warnings for 1+ major versions
   - Log deprecation warnings when deprecated feature is used
   - Provide migration path in warning message
   - Update all examples to use new pattern

3. **Remove** (vN.0.0): Remove deprecated feature in next major version
   - Breaking change documented in migration guide
   - Automated migration tool provided (when possible)

**Example Timeline**:

- **v1.5.0**: Announce deprecation of `ServerDefault` field
  - Documentation: "⚠️ DEPRECATED: Use `Default` with `sql()` instead"
  - No runtime changes

- **v1.6.0 - v2.0.0**: Runtime warnings
  - When `ServerDefault` is used: "WARNING: ServerDefault is deprecated and will be removed in v2.0.0. Use Default with sql() instead."
  - Examples updated to use `Default` with `sql()`

- **v2.0.0**: Remove `ServerDefault`
  - Breaking change
  - Migration guide provided
  - Automated migration tool: `alab migrate-schema --from 1.x --to 2.0`

### Compatibility Guarantees

- **Within MAJOR version**: Full backward compatibility
  - v1.0.0 schemas work with v1.9.9
  - Generators written for v1.0.0 work with v1.9.9

- **Across MAJOR versions**: Breaking changes allowed
  - v1.x schemas may need updates for v2.0
  - Migration guide and tools provided

- **Deprecation timeline**: Minimum 1 major version
  - Features deprecated in v1.x remain supported until v2.0
  - Warnings provided throughout v1.x series

### Support Timeline

| Version                   | Support Level                      | Duration            |
| ------------------------- | ---------------------------------- | ------------------- |
| **Current major (v1.x)**  | Full support (features + security) | Ongoing             |
| **Previous major (v0.x)** | Security fixes only                | 1 year after v1.0   |
| **Older versions**        | Community support only             | No official support |

**Example**:

- **v1.9.0** (current): Full support
- **v0.9.0** (previous major): Security fixes until 2027-02-14
- **v0.8.0** (older): Community support only

---

## Migration Path

### Version Detection in Generators

Generators should check the schema version and fail gracefully:

```javascript
export default gen((schema) => {
  // Parse version
  const [major, minor, patch] = schema.version.split(".").map(Number);

  // Check minimum required version
  if (major < 1) {
    throw new Error(
      `This generator requires IR version >= 1.0.0, got ${schema.version}`,
    );
  }

  // Check for specific features (added in v1.2.0)
  if (major === 1 && minor < 2) {
    console.warn(
      `Warning: Some features require IR version >= 1.2.0 (current: ${schema.version})`,
    );
  }

  // Generator implementation
  return render(files);
});
```

### Handling Breaking Changes

When a breaking change is introduced (new major version):

1. **Read Migration Guide**: Check `docs/migration/v1-to-v2.md`

2. **Use Migration Tool**: Run automated migration (when available)

   ```bash
   alab migrate-schema --from 1.x --to 2.0
   ```

3. **Update Generators**: Follow migration guide for generator updates

4. **Test Thoroughly**: Run tests with new version before deployment

### Example: v1 → v2 Migration

**Breaking Change**: `ServerDefault` field removed from `ColumnDef`

**Migration Steps**:

1. **Update Schema Files**:

   ```javascript
   // v1.x (deprecated)
   name: col.string(255).serverDefault("'Unknown'");

   // v2.0 (new pattern)
   name: col.string(255).default(sql("'Unknown'"));
   ```

2. **Run Migration Tool**:

   ```bash
   alab migrate-schema --from 1.x --to 2.0
   ```

3. **Update Generators** (if using `ServerDefault`):

   ```javascript
   // v1.x
   const defaultSQL = col.serverDefault || formatDefault(col.default);

   // v2.0
   const defaultSQL = formatDefault(col.default);
   ```

4. **Test**:
   ```bash
   alab check
   alab test-generators
   ```

---

## Reference Implementation

### Reading the IR

**Go Example** (`internal/engine/`):

```go
import "github.com/hlop3z/astroladb/internal/ast"

// Load schema from files
tables, err := evaluator.EvalDirStrict("./schemas")
if err != nil {
    return err
}

// Process each table
for _, table := range tables {
    fmt.Printf("Table: %s\n", table.QualifiedName())

    // Process columns
    for _, col := range table.Columns {
        fmt.Printf("  Column: %s (%s)\n", col.Name, col.Type)

        // Check type
        typeDef := types.Get(col.Type)
        if typeDef != nil {
            fmt.Printf("    Go type: %s\n", typeDef.GoType)
            fmt.Printf("    TS type: %s\n", typeDef.TSType)
        }

        // Check constraints
        if col.PrimaryKey {
            fmt.Println("    PRIMARY KEY")
        }
        if col.Unique {
            fmt.Println("    UNIQUE")
        }
        if !col.Nullable {
            fmt.Println("    NOT NULL")
        }
    }

    // Process indexes
    for _, idx := range table.Indexes {
        fmt.Printf("  Index: %v (%s)\n", idx.Columns, idx.Name)
    }

    // Process foreign keys
    for _, fk := range table.ForeignKeys {
        fmt.Printf("  FK: %v → %s(%v)\n", fk.Columns, fk.RefTable, fk.RefColumns)
    }
}
```

### Generating Code from IR

**JavaScript Generator Example**:

```javascript
export default gen((schema) => {
  const files = {};

  // Check IR version
  if (!schema.version || schema.version < "1.0.0") {
    throw new Error("This generator requires IR version >= 1.0.0");
  }

  // Type mapping
  const typeMap = {
    id: "UUID",
    uuid: "UUID",
    string: "str",
    text: "str",
    integer: "int",
    float: "float",
    decimal: "Decimal",
    boolean: "bool",
    date: "date",
    time: "time",
    datetime: "datetime",
    json: "dict",
    base64: "str",
  };

  // Process each namespace
  for (const [namespace, tables] of Object.entries(schema.models)) {
    // Generate models for this namespace
    const models = tables
      .map((table) => {
        // Map columns to types
        const fields = table.columns
          .map((col) => {
            const pyType =
              col.type === "enum" && col.enum
                ? `${capitalize(col.name)}Enum`
                : typeMap[col.type] || "Any";

            const nullable = col.nullable ? `Optional[${pyType}]` : pyType;
            const default_ =
              col.default !== undefined
                ? ` = ${formatDefault(col.default)}`
                : col.nullable
                  ? " = None"
                  : "";

            return `    ${col.name}: ${nullable}${default_}`;
          })
          .join("\n");

        return `
class ${pascalCase(table.name)}(BaseModel):
${fields}

    class Config:
        from_attributes = True
      `.trim();
      })
      .join("\n\n");

    files[`${namespace}/models.py`] = models;
  }

  return render(files);
});
```

---

## Appendix

### JSON Schema Definitions

Complete JSON Schema for the IR is available in `schemas/ir.schema.json` (to be created).

### Type Converter Reference

Type converters for all supported languages:

| Language   | Location                             | Class                 |
| ---------- | ------------------------------------ | --------------------- |
| TypeScript | `pkg/astroladb/export_typescript.go` | `TypeScriptConverter` |
| Go         | `pkg/astroladb/export_golang.go`     | `GoConverter`         |
| Python     | `pkg/astroladb/export_python.go`     | `PythonConverter`     |
| Rust       | `pkg/astroladb/export_rust.go`       | `RustConverter`       |
| GraphQL    | `pkg/astroladb/export_graphql.go`    | `GraphQLConverter`    |
| OpenAPI    | `pkg/astroladb/export_openapi.go`    | `OpenAPIConverter`    |

### Further Reading

- [Architecture Review](../docs-devs/architecture-review.md) - Platform architecture analysis
- [TESTING.md](../TESTING.md) - Testing strategy and patterns
- [CONTRIBUTING.md](../CONTRIBUTING.md) - Contributing guidelines
- [CLAUDE.md](../CLAUDE.md) - Development guide for Claude Code

---

**Last Updated**: 2026-02-14
**IR Version**: 1.0.0
**Status**: Stable
