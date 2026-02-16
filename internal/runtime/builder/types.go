package builder

// Type-safe definitions for the builder package.

// -----------------------------------------------------------------------------
// Type-safe column and index definitions
// -----------------------------------------------------------------------------

// ColumnDef represents a type-safe column definition during building.
// This provides compile-time safety and IDE support when creating columns.
type ColumnDef struct {
	Name           string
	Type           string
	TypeArgs       []any
	Nullable       bool
	Unique         bool
	Index          bool
	PrimaryKey     bool
	Default        any
	Backfill       any
	Format         string
	Pattern        string
	Min            *float64
	Max            *float64
	Docs           string
	Deprecated     string
	ReadOnly       bool
	WriteOnly      bool
	Reference      *RefDef
	Hidden         bool   // x_hidden for OpenAPI
	XRef           string // Original reference (e.g., "auth.user")
	Computed       any    // Computed expression (FnExpr or map)
	Virtual        bool   // VIRTUAL instead of STORED, or app-only if no Computed
	IsRelationship bool   // True if this is a relationship column
}

// RefDef represents a foreign key reference.
type RefDef struct {
	Table    string
	Column   string
	OnDelete string
	OnUpdate string
}

// IndexDef represents a type-safe index definition.
type IndexDef struct {
	Name    string
	Columns []string
	Unique  bool
	Where   string
}

// RelationshipDef represents a many-to-many, polymorphic, or junction relationship.
type RelationshipDef struct {
	Type           string   // "many_to_many", "polymorphic", or "junction"
	Target         string   // Target table for many_to_many
	Targets        []string // Target tables for polymorphic
	As             string   // Alias
	JunctionSource string   // Source table ref for junction (optional)
	JunctionTarget string   // Target table ref for junction (optional)
}

// simpleTypes lists column types that take no extra arguments.
// Used by both ColBuilder and TableBuilder to register type methods in a loop.
var simpleTypes = []string{
	"text", "integer", "float", "boolean",
	"date", "time", "datetime",
	"uuid", "json", "base64",
}
