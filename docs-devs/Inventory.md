# Canonical Types and Patterns Inventory

This document provides a comprehensive inventory of all canonical types, conversion patterns, and instances where the codebase has multiple ways to accomplish the same thing.

---

## Table of Contents

1. [Canonical Type Definitions](#1-canonical-type-definitions)
2. [Multiple Ways to Do the Same Thing](#2-multiple-ways-to-do-the-same-thing)
3. [Canonicalization and Normalization Logic](#3-canonicalization-and-normalization-logic)
4. [Type Mapping and Conversion Systems](#4-type-mapping-and-conversion-systems)
5. [Canonical Patterns Across Layers](#5-canonical-patterns-across-layers)
6. [Complete Inventory Summary](#6-complete-inventory-summary)

---

## 1. Canonical Type Definitions

### 1.1 Primary Type System Definition

**File**: `internal/types/types.go` (Lines 20-345)

The core type registry defines all canonical Alab types:

| Type       | Description                                   | Canonical SQL (Postgres)         |
| ---------- | --------------------------------------------- | -------------------------------- |
| `id`       | UUID primary key                              | `UUID DEFAULT gen_random_uuid()` |
| `string`   | VARCHAR with required length                  | `VARCHAR(n)`                     |
| `text`     | Unlimited text                                | `TEXT`                           |
| `integer`  | 32-bit signed (JS-safe)                       | `INTEGER`                        |
| `float`    | 32-bit floating-point                         | `REAL`                           |
| `decimal`  | Arbitrary precision (stored as string for JS) | `NUMERIC(p,s)`                   |
| `boolean`  | Boolean value                                 | `BOOLEAN`                        |
| `date`     | Date only (ISO 8601 string)                   | `DATE`                           |
| `time`     | Time only (RFC 3339 string)                   | `TIME`                           |
| `datetime` | Timestamp with timezone                       | `TIMESTAMPTZ`                    |
| `uuid`     | UUID value (not primary key)                  | `UUID`                           |
| `json`     | JSON/JSONB object                             | `JSONB`                          |
| `base64`   | Binary/base64 data                            | `BYTEA`                          |
| `enum`     | Enumerated values                             | Custom ENUM type                 |

**Registry Pattern** (Lines 49-79):

```go
var registry = map[string]*TypeDef

func Register(t *TypeDef)     // adds to registry
func Get(name string) *TypeDef // retrieves from registry
func Exists(name string) bool  // checks existence
func All() []*TypeDef          // returns all types
```

### 1.2 Table Definition Canonical Forms

**File**: `internal/ast/table.go` (Lines 1-333)

#### TableDef Structure (Lines 11-30)

The canonical table structure with:

- `Namespace` - logical grouping
- `Name` - snake_case identifier
- `Columns`, `Indexes`, `ForeignKeys`, `Checks`
- `Doc` and `Deprecated` metadata

#### ColumnDef Structure (Lines 125-163)

Complete column definition with:

- `Type` and `TypeArgs`
- `Default` vs `ServerDefault` (dual representation)
- `DefaultSet` and `BackfillSet` flags
- `Reference` for foreign keys

#### Three Name Representation Forms (Lines 32-52)

| Method            | Format              | Example      |
| ----------------- | ------------------- | ------------ |
| `FullName()`      | namespace_tablename | `auth_users` |
| `FullName()`      | namespace_tablename | `auth_users` |
| `QualifiedName()` | namespace.table     | `auth.users` |

### 1.3 Introspection Type Mappings

**File**: `internal/introspect/types.go` (Lines 14-122)

**TypeMapping struct** (Lines 8-12):

```go
type TypeMapping struct {
    AlabType string // Canonical Alab type name
    TypeArgs []any  // Parsed arguments
}
```

**Database → Canonical Conversion Functions**:

- `MapPostgresType()` (Lines 15-83) - 11 distinct type mappings
- `MapSQLiteType()` (Lines 88-122) - 9 distinct type mappings

### 1.4 Operation Type Enum

**File**: `internal/ast/types.go` (Lines 6-87)

13 canonical operation types:

- `OpCreateTable`, `OpDropTable`, `OpRenameTable`
- `OpAddColumn`, `OpDropColumn`, `OpRenameColumn`, `OpAlterColumn`
- `OpCreateIndex`, `OpDropIndex`
- `OpAddForeignKey`, `OpDropForeignKey`
- `OpAddCheck`, `OpDropCheck`
- `OpRawSQL`

---

## 2. Multiple Ways to Do the Same Thing

### 2.1 Default Values - Dual Representation

**Files**:

- `internal/ast/table.go` (Lines 140-142)
- `internal/drift/merkle.go` (Lines 190-196)
- `internal/cache/schema.go` (Lines 302-355)

**Two Representations**:

| Field           | Type                     | Source                 | Usage                      |
| --------------- | ------------------------ | ---------------------- | -------------------------- |
| `Default`       | `any` (SQLExpr or value) | DSL builder            | Building schemas from code |
| `ServerDefault` | `string`                 | Database introspection | Raw SQL expression from DB |

**When Each is Used**:

- DSL builder → uses `Default` field with SQLExpr wrapper
- Introspection → populates `ServerDefault` from database
- Drift detection checks BOTH fields (line 195):

```go
if col.ServerDefault != "" {
    data += fmt.Sprintf("|serverdefault:%s", col.ServerDefault)
}
```

### 2.2 Table Name References - Multiple Formats

**Files**:

- `internal/registry/resolver.go` (Lines 11-47)
- `internal/dsl/table.go` (Lines 408-430)
- `internal/validate/validate.go` (Lines 363-427)

**Three Reference Formats**:

| Format          | Example      | Description                     |
| --------------- | ------------ | ------------------------------- |
| Fully qualified | `auth.users` | Explicit namespace              |
| Relative        | `.users`     | Current namespace (leading dot) |

**Resolution Table**:

| Input        | Current NS | Resolves To  | SQL Name     |
| ------------ | ---------- | ------------ | ------------ |
| `auth.users` | any        | `auth.users` | `auth_users` |
| `.roles`     | auth       | `auth.roles` | `auth_roles` |

**Resolution Logic** (Lines 56-82 in resolver.go):

```go
func NormalizeReference(ref string, currentNS string) (string, error)
func ParseReference(ref string) (namespace, table string, isRelative bool)
```

### 2.3 Foreign Key Actions - Standardized Enum

**File**: `internal/dsl/column.go` (Lines 169-176)

Canonical action constants:

```go
const (
    Cascade    = "CASCADE"
    SetNull    = "SET NULL"
    Restrict   = "RESTRICT"
    NoAction   = "NO ACTION"
    SetDefault = "SET DEFAULT"
)
```

**Used by multiple structures**:

- `Reference.OnDelete` and `Reference.OnUpdate`
- `ForeignKeyDef.OnDelete` and `ForeignKeyDef.OnUpdate`
- `AlterColumn.OnDelete` and `AlterColumn.OnUpdate`

**Normalization**: `normalizeAction()` (Lines 83-97 in introspect/introspect.go)

### 2.4 Boolean Literals - Dialect-Specific

**File**: `internal/dialect/base.go` (Lines 104-114)

**Two Boolean Literal Sets**:

```go
var PostgresBooleans = BooleanLiterals{True: "TRUE", False: "FALSE"}
var SQLiteBooleans   = BooleanLiterals{True: "1",    False: "0"}
```

Used in `buildDefaultValueSQL()` function (Lines 116-141)

### 2.5 Table Name Format Conversions

**SQL Format Conversions**:

- `FullName()` / `FullName()` → `namespace_tablename` (underscore-separated, flat)
- `QualifiedName()` → `namespace.table` (dot-separated, hierarchical)

**Reverse Parsing** (Lines 72-80 in introspect/introspect.go):

```go
func parseTableName(FullName string) (namespace, name string)
// "auth_user" → ("auth", "user")
// "user" → ("", "user")
```

---

## 3. Canonicalization and Normalization Logic

### 3.1 Schema Normalization Through Dev Database

**Files**:

- `pkg/astroladb/astroladb.go` (Lines 437-456)
- `pkg/astroladb/schema.go` (Lines 233-276, 349-373)
- `internal/devdb/devdb.go` (Lines 1-87)

**The Pattern** (Atlas-inspired):

1. **Input**: Natural/raw schema (from DSL or introspection)
2. **Apply**: To temporary dev database via DDL
3. **Introspect**: Dev database to get normalized form
4. **Result**: Canonical representation for comparison

**Two-Step Normalization Function** (schema.go line 358):

```go
func (c *Client) normalizeSchema(schema *engine.Schema) (*engine.Schema, error) {
    // Step 1: Create dev database context
    // Step 2: Apply schema to dev DB
    // Step 3: Introspect normalized result
    return dev.NormalizeSchema(ctx, schema)
}
```

**Why This Matters** (Lines 437-446):

- Databases store in "canonical form" (normalized representation)
- SQL like `NOW()` gets normalized to `CURRENT_TIMESTAMP`
- Default values get their SQL forms normalized
- Comparison without normalization gives false positives

### 3.2 Column Type Normalization

**File**: `internal/drift/merkle.go` (Lines 231-260)

**Function**: `normalizeType()` (Line 232)

```go
func normalizeType(typeName string, args []any) string
// Combines type name with arguments into deterministic string
```

**SQL Normalization** (Lines 83-97 in introspect/introspect.go):

```go
func normalizeAction(action string) string
// Converts to canonical actions:
// "CASCADE" → "CASCADE"
// "SET NULL" → "SET NULL"
// "RESTRICT" → "RESTRICT"
// "NO ACTION", "" → "" (default)
```

### 3.3 Merkle Hash Computation

**File**: `internal/drift/merkle.go` (Lines 53-150)

**Hierarchical Canonical Hash**:

- Root hash = merkle tree of all table hashes
- Table hash = merkle tree of columns/indexes/FKs
- Column hash = deterministic hash of column properties
- Index hash = deterministic hash of index properties
- FK hash = deterministic hash of constraint properties

**Deterministic Ordering**:

- TableNames sorted (line 69)
- Column names sorted (lines 108-112)
- Ensures same schema = same hash regardless of order

---

## 4. Type Mapping and Conversion Systems

### 4.1 Dialect-Specific Type Mapping

**File**: `internal/dialect/base.go` (Lines 26-102)

**TypeMapper Interface** (Lines 28-43):

```go
type TypeMapper interface {
    IDType() string
    StringType(length int) string
    TextType() string
    IntegerType() string
    FloatType() string
    DecimalType(precision, scale int) string
    BooleanType() string
    DateType() string
    TimeType() string
    DateTimeType() string
    UUIDType() string
    JSONType() string
    Base64Type() string
}
```

**Implementations**:

- PostgreSQL dialect (`internal/dialect/postgres.go`)
- SQLite dialect (`internal/dialect/sqlite.go`)

**Central Type Builder** (Lines 45-102):

```go
func buildColumnTypeSQL(typeName string, typeArgs []any, mapper TypeMapper) string
```

### 4.2 DSL Builder to AST Conversion

**Files**: `internal/dsl/table.go` & `internal/dsl/column.go`

**Fluent API Conversion Pattern**:

```go
col := NewColumnBuilder("email", "string", 255)
    .Unique()
    .Default("no-reply@example.com")
    .Build()
// Returns: ColumnDef with all properties set
```

### 4.3 Schema Serialization

**File**: `internal/cache/schema.go` (Lines 108-174)

**Conversion Functions**:

- `SerializeSchema()` - engine.Schema → JSON bytes
- `DeserializeSchema()` - JSON bytes → engine.Schema
- `SerializeSchemaHash()` - drift.SchemaHash → JSON
- `DeserializeSchemaHash()` - JSON → drift.SchemaHash

**Special Handling** (Lines 302-355):

```go
// SQLExpr stored as per-dialect format:
{"_type": "sql_expr", "postgres": "...", "sqlite": "..."}
// When deserializing, reconstructs as &ast.SQLExpr{Postgres: ..., SQLite: ...}
```

### 4.4 Export Format Conversions

**File**: `pkg/astroladb/export.go` (Lines 16-1788)

**Multiple Export Converters**:

| Function                    | Line | Output          |
| --------------------------- | ---- | --------------- |
| `columnToOpenAPIProperty()` | 765  | OpenAPI schema  |
| `columnToTypeScriptType()`  | 1213 | TypeScript type |
| `mapToGoType()`             | 1325 | Go type         |
| `columnToPythonType()`      | 1447 | Python type     |
| `columnToRustType()`        | 1603 | Rust type       |
| `columnToGraphQLType()`     | 1782 | GraphQL type    |

**Name Converters**:

- `camelCase()` (line 1125) - snake_case → camelCase
- `pascalCase()` (line 1245) - snake_case → PascalCase

### 4.5 String Utility Conversions

**File**: `internal/strutil/strutil.go`

| Function         | Line | Description                          |
| ---------------- | ---- | ------------------------------------ |
| `ToSnakeCase()`  | 14   | various → snake_case                 |
| `ToPascalCase()` | 50   | various → PascalCase                 |
| `ToCamelCase()`  | 77   | various → camelCase                  |
| `FullName()`     | -    | (namespace, table) → namespace_table |

---

## 5. Canonical Patterns Across Layers

### 5.1 Layer-Specific Canonical Forms

| Layer             | Canonical Form      | Example                                    | File                |
| ----------------- | ------------------- | ------------------------------------------ | ------------------- |
| **JS DSL**        | Builder API         | `t.string("name", 100)`                    | runtime/builder.go  |
| **AST**           | ColumnDef, TableDef | `ColumnDef{Type:"string", TypeArgs:[100]}` | ast/table.go        |
| **Database**      | SQL DDL             | `VARCHAR(100) NOT NULL`                    | dialect/\*.go       |
| **Introspection** | TypeMapping         | `{AlabType:"string", TypeArgs:[100]}`      | introspect/types.go |
| **Export**        | Format-specific     | `string`, `String`, `str`, etc.            | export.go           |
| **Cache**         | JSON                | Serialized schema                          | cache/schema.go     |
| **Drift**         | Merkle Hash         | SHA256 hash tree                           | drift/merkle.go     |

### 5.2 Conversion Flow Example

```
JS DSL:                t.string("email", 255)
     ↓
ColumnBuilder:         NewColumnBuilder("email", "string", 255)
     ↓
AST (TableDef):       ColumnDef{Name:"email", Type:"string", TypeArgs:[255]}
     ↓
Type Validation:      types.Validate("string", [255])
     ↓
Dialect SQL:          buildColumnTypeSQL("string", [255], postgresMapper)
                      → mapper.StringType(255) → "VARCHAR(255)"
     ↓
Database:             CREATE TABLE t (email VARCHAR(255) NOT NULL)
     ↓
Introspection:        SELECT type, max_length FROM columns
     ↓
TypeMapping:          MapPostgresType("VARCHAR", 255)
                      → TypeMapping{AlabType:"string", TypeArgs:[255]}
     ↓
Normalization:        Apply to dev DB, introspect back
     ↓
Drift Detection:      Compare hashes (deterministic)
     ↓
Export (TypeScript):  columnToTypeScriptType → "string"
```

---

## 6. Complete Inventory Summary

### Canonical Type Definitions (15 total)

**Location**: `internal/types/types.go`

`id`, `string`, `text`, `integer`, `float`, `decimal`, `boolean`, `date`, `time`, `datetime`, `uuid`, `json`, `base64`, `enum`, `computed`

### Canonical Structures (7 total)

**Location**: `internal/ast/table.go`

`TableDef`, `ColumnDef`, `SQLExpr`, `IndexDef`, `ForeignKeyDef`, `CheckDef`, `Reference`

### Canonical Operations (13 total)

**Location**: `internal/ast/types.go`

`CreateTable`, `DropTable`, `RenameTable`, `AddColumn`, `DropColumn`, `RenameColumn`, `AlterColumn`, `CreateIndex`, `DropIndex`, `AddForeignKey`, `DropForeignKey`, `AddCheck`, `DropCheck` (+ `OpRawSQL`)

### Canonical Conversion Functions (20+ total)

**Type Conversions**:

- `MapPostgresType` (11 cases)
- `MapSQLiteType` (9 cases)
- `buildColumnTypeSQL` (12 type branches)

**Name Conversions**:

- `ToSnakeCase`, `ToPascalCase`, `ToCamelCase`
- `FullName`, `parseTableName`

**Reference Conversions**:

- `ParseReference`, `NormalizeReference`
- `Resolve`, `ResolveColumnReference`

**Export Conversions**:

- `columnToOpenAPIProperty`, `columnToTypeScriptType`
- `mapToGoType`, `columnToPythonType`
- `columnToRustType`, `columnToGraphQLType`

**Serialization**:

- `SerializeSchema`, `DeserializeSchema`
- `SerializeSchemaHash`, `DeserializeSchemaHash`

### Canonical Normalization Points (6 total)

| Normalization | Function                              | Location                          |
| ------------- | ------------------------------------- | --------------------------------- |
| Schema        | `normalizeSchema()`                   | pkg/astroladb/schema.go           |
| Action        | `normalizeAction()`                   | internal/introspect/introspect.go |
| Type          | `normalizeType()`                     | internal/drift/merkle.go          |
| Merkle Hash   | `ComputeSchemaHash()`                 | internal/drift/merkle.go          |
| Reference     | `NormalizeReference()`                | internal/registry/resolver.go     |
| Boolean       | `PostgresBooleans` / `SQLiteBooleans` | internal/dialect/base.go          |

### Summary of Multiple Code Paths

| Pattern            | Path 1                       | Path 2                   | Convergence                 |
| ------------------ | ---------------------------- | ------------------------ | --------------------------- |
| **Defaults**       | `Default` (SQLExpr)          | `ServerDefault` (string) | Both hashed for drift       |
| **Table Names**    | `QualifiedName()` (ns.table) | `FullName()` (ns_table)  | `parseTableName()` reverse  |
| **References**     | Fully qualified              | Relative                 | `NormalizeReference()`      |
| **FK Actions**     | Constants (Cascade, etc)     | Raw strings from DB      | `normalizeAction()`         |
| **Booleans**       | Postgres TRUE/FALSE          | SQLite 1/0               | `buildDefaultValueSQL()`    |
| **Type Mapping**   | Postgres types               | SQLite types             | `TypeMapper` interface      |
| **Export Formats** | OpenAPI, TypeScript, Go      | Python, Rust, GraphQL    | `columnToXType()` functions |
| **Storage**        | AST in memory                | JSON in cache            | Serialization funcs         |
| **Schema State**   | Expected (from code)         | Actual (from DB)         | Dev DB normalization        |

## Compare Histoy

1. `alab check` - Compares migration history vs database

- Purpose: Detect if database has drifted from migrations
- Use case: Validate migration integrity, detect manual schema changes

2. `alab diff` - Compares schema files vs database

- Purpose: Show what migrations need to be created
- Use case: Generate new migrations when schema files change
