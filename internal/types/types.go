// Package types defines the portable type system for Alab.
// These types map from the JS DSL to SQL types for each supported database dialect.
//
// The type system is designed to be:
//   - JS-Friendly: No types that lose precision in JavaScript (no 64-bit integers/floats, auto-increment)
//   - Portable: Works across PostgreSQL and SQLite
//   - Simple: One way to do things, minimal options
//
// All types use snake_case in the JS DSL to maintain consistency.
package types

import (
	"github.com/hlop3z/astroladb/internal/alerr"
)

// -----------------------------------------------------------------------------
// TypeDef - Type definition
// -----------------------------------------------------------------------------

// TypeDef represents a portable type definition.
type TypeDef struct {
	Name     string      // Internal type name (e.g., "string", "integer")
	JSName   string      // JS DSL name (snake_case, e.g., "string", "datetime")
	GoType   string      // Go type for the value (e.g., "string", "int32")
	TSType   string      // TypeScript type (e.g., "string", "number")
	OpenAPI  OpenAPIType // OpenAPI/JSON Schema type info
	SQLTypes SQLTypeMap  // Database-specific SQL types
	HasArgs  bool        // True if type accepts arguments (e.g., string(255))
	MinArgs  int         // Minimum number of arguments (0 if optional)
	MaxArgs  int         // Maximum number of arguments
}

// SQLTypeMap holds database-specific SQL type strings.
type SQLTypeMap struct {
	Postgres string // PostgreSQL type (e.g., "UUID", "TIMESTAMPTZ")
	SQLite   string // SQLite type (e.g., "TEXT", "INTEGER")
}

// OpenAPIType represents the OpenAPI/JSON Schema type information.
type OpenAPIType struct {
	Type   string // JSON Schema type: string, integer, number, boolean, object, array
	Format string // Format hint: uuid, date-time, email, uri, etc.
}

// -----------------------------------------------------------------------------
// Type Registry
// -----------------------------------------------------------------------------

// registry holds all registered types indexed by name.
var registry = make(map[string]*TypeDef)

// Register adds a type to the registry.
// Panics if a type with the same name is already registered.
func Register(t *TypeDef) {
	if _, exists := registry[t.Name]; exists {
		panic("type already registered: " + t.Name)
	}
	registry[t.Name] = t
}

// Get returns the type definition for the given name.
// Returns nil if the type is not found.
func Get(name string) *TypeDef {
	return registry[name]
}

// Exists returns true if a type with the given name is registered.
func Exists(name string) bool {
	return registry[name] != nil
}

// All returns all registered types.
func All() []*TypeDef {
	types := make([]*TypeDef, 0, len(registry))
	for _, t := range registry {
		types = append(types, t)
	}
	return types
}

// Validate checks if a type name is valid and the arguments are correct.
func Validate(typeName string, args []any) error {
	// Check forbidden types first
	if forbidden, reason := IsForbidden(typeName); forbidden {
		return alerr.New(alerr.ErrInvalidType, "type is not allowed (JS-unsafe)").
			With("type", typeName).
			With("reason", reason)
	}

	t := Get(typeName)
	if t == nil {
		return alerr.New(alerr.ErrInvalidType, "unknown type").
			With("type", typeName)
	}

	// Validate argument count
	argCount := len(args)
	if !t.HasArgs && argCount > 0 {
		return alerr.New(alerr.ErrInvalidType, "type does not accept arguments").
			With("type", typeName).
			With("got", argCount)
	}
	if t.HasArgs {
		if argCount < t.MinArgs {
			return alerr.New(alerr.ErrInvalidType, "type requires more arguments").
				With("type", typeName).
				With("minimum", t.MinArgs).
				With("got", argCount)
		}
		if argCount > t.MaxArgs {
			return alerr.New(alerr.ErrInvalidType, "type has too many arguments").
				With("type", typeName).
				With("maximum", t.MaxArgs).
				With("got", argCount)
		}
	}

	return nil
}

// -----------------------------------------------------------------------------
// Built-in Types
// -----------------------------------------------------------------------------

func init() {
	// UUID primary key (auto-generated)
	Register(&TypeDef{
		Name:    "id",
		JSName:  "id",
		GoType:  "string",
		TSType:  "string",
		OpenAPI: OpenAPIType{Type: "string", Format: "uuid"},
		SQLTypes: SQLTypeMap{
			Postgres: "UUID DEFAULT gen_random_uuid()",
			SQLite:   "TEXT",
		},
		HasArgs: false,
	})

	// Variable-length string with max length (required)
	// Note: SQLTypes use placeholder %d for length substitution
	Register(&TypeDef{
		Name:    "string",
		JSName:  "string",
		GoType:  "string",
		TSType:  "string",
		OpenAPI: OpenAPIType{Type: "string"},
		SQLTypes: SQLTypeMap{
			Postgres: "VARCHAR(%d)",
			SQLite:   "TEXT",
		},
		HasArgs: true,
		MinArgs: 1, // Length is required
		MaxArgs: 1,
	})

	// Unlimited length text
	Register(&TypeDef{
		Name:    "text",
		JSName:  "text",
		GoType:  "string",
		TSType:  "string",
		OpenAPI: OpenAPIType{Type: "string"},
		SQLTypes: SQLTypeMap{
			Postgres: "TEXT",
			SQLite:   "TEXT",
		},
		HasArgs: false,
	})

	// 32-bit signed integer (JS-safe: fits in JS number)
	Register(&TypeDef{
		Name:    "integer",
		JSName:  "integer",
		GoType:  "int32",
		TSType:  "number",
		OpenAPI: OpenAPIType{Type: "integer", Format: "int32"},
		SQLTypes: SQLTypeMap{
			Postgres: "INTEGER",
			SQLite:   "INTEGER",
		},
		HasArgs: false,
	})

	// 32-bit floating point (JS-safe)
	Register(&TypeDef{
		Name:    "float",
		JSName:  "float",
		GoType:  "float32",
		TSType:  "number",
		OpenAPI: OpenAPIType{Type: "number", Format: "float"},
		SQLTypes: SQLTypeMap{
			Postgres: "REAL",
			SQLite:   "REAL",
		},
		HasArgs: false,
	})

	// Arbitrary-precision decimal (serialized as string to preserve precision)
	// Use for money and other values requiring exact precision
	// Note: SQLTypes use %d,%d for precision,scale substitution
	Register(&TypeDef{
		Name:    "decimal",
		JSName:  "decimal",
		GoType:  "string", // String-serialized to preserve precision in JS
		TSType:  "string",
		OpenAPI: OpenAPIType{Type: "string", Format: "decimal"},
		SQLTypes: SQLTypeMap{
			Postgres: "NUMERIC(%d,%d)",
			SQLite:   "TEXT",
		},
		HasArgs: true,
		MinArgs: 2, // (precision, scale)
		MaxArgs: 2,
	})

	// Boolean (true/false)
	Register(&TypeDef{
		Name:    "boolean",
		JSName:  "boolean",
		GoType:  "bool",
		TSType:  "boolean",
		OpenAPI: OpenAPIType{Type: "boolean"},
		SQLTypes: SQLTypeMap{
			Postgres: "BOOLEAN",
			SQLite:   "INTEGER",
		},
		HasArgs: false,
	})

	// Date only (YYYY-MM-DD) - serialized as ISO 8601 string
	Register(&TypeDef{
		Name:    "date",
		JSName:  "date",
		GoType:  "string", // ISO 8601 date string
		TSType:  "string",
		OpenAPI: OpenAPIType{Type: "string", Format: "date"},
		SQLTypes: SQLTypeMap{
			Postgres: "DATE",
			SQLite:   "TEXT",
		},
		HasArgs: false,
	})

	// Time only (HH:MM:SS) - serialized as RFC 3339 string
	Register(&TypeDef{
		Name:    "time",
		JSName:  "time",
		GoType:  "string", // RFC 3339 time string
		TSType:  "string",
		OpenAPI: OpenAPIType{Type: "string", Format: "time"},
		SQLTypes: SQLTypeMap{
			Postgres: "TIME",
			SQLite:   "TEXT",
		},
		HasArgs: false,
	})

	// Timestamp with timezone - stored in UTC, serialized as RFC 3339 string
	// PostgreSQL: TIMESTAMPTZ stores in UTC, connection timezone set to UTC
	// SQLite: TEXT in ISO 8601 format
	Register(&TypeDef{
		Name:    "datetime",
		JSName:  "datetime",
		GoType:  "string", // RFC 3339 date-time string
		TSType:  "string",
		OpenAPI: OpenAPIType{Type: "string", Format: "date-time"},
		SQLTypes: SQLTypeMap{
			Postgres: "TIMESTAMPTZ",
			SQLite:   "TEXT",
		},
		HasArgs: false,
	})

	// UUID value (not primary key)
	Register(&TypeDef{
		Name:    "uuid",
		JSName:  "uuid",
		GoType:  "string",
		TSType:  "string",
		OpenAPI: OpenAPIType{Type: "string", Format: "uuid"},
		SQLTypes: SQLTypeMap{
			Postgres: "UUID",
			SQLite:   "TEXT",
		},
		HasArgs: false,
	})

	// JSON/JSONB value - native JS object
	Register(&TypeDef{
		Name:    "json",
		JSName:  "json",
		GoType:  "any", // map[string]any or []any
		TSType:  "Record<string, unknown>",
		OpenAPI: OpenAPIType{Type: "object"},
		SQLTypes: SQLTypeMap{
			Postgres: "JSONB",
			SQLite:   "TEXT",
		},
		HasArgs: false,
	})

	// Binary data - serialized as base64 string
	Register(&TypeDef{
		Name:    "base64",
		JSName:  "base64",
		GoType:  "string", // Base64-encoded string
		TSType:  "string",
		OpenAPI: OpenAPIType{Type: "string", Format: "byte"},
		SQLTypes: SQLTypeMap{
			Postgres: "BYTEA",
			SQLite:   "BLOB",
		},
		HasArgs: false,
	})

	// Enumerated values - stored as string
	// Note: Enum SQL types are handled specially per dialect
	Register(&TypeDef{
		Name:    "enum",
		JSName:  "enum",
		GoType:  "string",
		TSType:  "string", // Overridden to union type when values are known
		OpenAPI: OpenAPIType{Type: "string"},
		SQLTypes: SQLTypeMap{
			Postgres: "", // Uses named type
			SQLite:   "TEXT",
		},
		HasArgs: true,
		MinArgs: 1,   // At least one enum value required
		MaxArgs: 100, // Reasonable limit on enum values
	})

	// Computed column (virtual, read-only)
	Register(&TypeDef{
		Name:    "computed",
		JSName:  "computed",
		GoType:  "string",
		TSType:  "string",
		OpenAPI: OpenAPIType{Type: "string"},
		SQLTypes: SQLTypeMap{
			Postgres: "",
			SQLite:   "",
		},
		HasArgs: false,
	})
}

// -----------------------------------------------------------------------------
// Forbidden Types (for validation and error messages)
// -----------------------------------------------------------------------------

// forbiddenTypes lists types that are not allowed because they break in JavaScript.
// Key is the type name, value is the reason it's forbidden.
var forbiddenTypes = map[string]string{
	"bigint":    "JavaScript loses precision for values > 2^53. Use integer (32-bit) or decimal (string).",
	"double":    "JavaScript has precision issues with 64-bit floats. Use float (32-bit) or decimal (string).",
	"serial":    "Auto-increment IDs are not allowed. Use id (UUID) for primary keys.",
	"bigserial": "Auto-increment IDs are not allowed. Use id (UUID) for primary keys.",
}

// IsForbidden returns true if the type is forbidden, along with the reason.
func IsForbidden(typeName string) (bool, string) {
	reason, forbidden := forbiddenTypes[typeName]
	return forbidden, reason
}

// -----------------------------------------------------------------------------
// Standard Formats (fmt.X in JS DSL)
// -----------------------------------------------------------------------------

// Format represents a standard format with its OpenAPI name and RFC pattern.
type Format struct {
	Name    string // OpenAPI format name (e.g., "email", "uri")
	Pattern string // Regex pattern for validation (empty if none)
	RFC     string // RFC reference for documentation
}

// StandardFormats defines the standard formats available via fmt.X in the JS DSL.
// These provide both OpenAPI format hints and validation patterns.
var StandardFormats = map[string]Format{
	"email": {
		Name:    "email",
		Pattern: `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`,
		RFC:     "RFC 5322",
	},
	"uri": {
		Name:    "uri",
		Pattern: `^[a-zA-Z][a-zA-Z0-9+.-]*://[^\s]*$`,
		RFC:     "RFC 3986",
	},
	"uuid": {
		Name:    "uuid",
		Pattern: `^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`,
		RFC:     "RFC 4122",
	},
	"date": {
		Name:    "date",
		Pattern: `^\d{4}-\d{2}-\d{2}$`,
		RFC:     "ISO 8601",
	},
	"time": {
		Name:    "time",
		Pattern: `^\d{2}:\d{2}:\d{2}(.\d+)?(Z|[+-]\d{2}:\d{2})?$`,
		RFC:     "RFC 3339",
	},
	"datetime": {
		Name:    "date-time",
		Pattern: `^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(.\d+)?(Z|[+-]\d{2}:\d{2})?$`,
		RFC:     "RFC 3339",
	},
	"hostname": {
		Name:    "hostname",
		Pattern: `^[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$`,
		RFC:     "RFC 1123",
	},
	"ipv4": {
		Name:    "ipv4",
		Pattern: `^((25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$`,
		RFC:     "RFC 791",
	},
	"ipv6": {
		Name:    "ipv6",
		Pattern: `^([0-9a-fA-F]{1,4}:){7}[0-9a-fA-F]{1,4}$`,
		RFC:     "RFC 4291",
	},
	"password": {
		Name:    "password",
		Pattern: "", // No pattern - just a UI hint
		RFC:     "",
	},
}

// GetFormat returns the format definition for the given name.
// Returns nil if the format is not found.
func GetFormat(name string) *Format {
	f, ok := StandardFormats[name]
	if !ok {
		return nil
	}
	return &f
}
