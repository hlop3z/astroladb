package runtime

import (
	"log/slog"
)

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
	PrimaryKey     bool
	Default        any
	Backfill       any
	Format         string
	Pattern        string
	Min            *float64
	Max            *float64
	Docs           string
	Deprecated     string
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

// RelationshipDef represents a many-to-many or polymorphic relationship.
type RelationshipDef struct {
	Type    string   // "many_to_many" or "polymorphic"
	Target  string   // Target table for many_to_many
	Targets []string // Target tables for polymorphic
	As      string   // Alias
}

func columnDefToMap(col *ColumnDef) map[string]any {
	slog.Debug("columnDefToMap: converting column",
		"name", col.Name,
		"type", col.Type,
		"default", col.Default)

	m := map[string]any{
		"name":     col.Name,
		"type":     col.Type,
		"nullable": col.Nullable,
	}

	if len(col.TypeArgs) > 0 {
		m["type_args"] = col.TypeArgs
	}
	if col.Unique {
		m["unique"] = true
	}
	if col.PrimaryKey {
		m["primary_key"] = true
	}
	if col.Default != nil {
		slog.Debug("columnDefToMap: adding default to map",
			"column", col.Name,
			"default", col.Default)
		m["default"] = col.Default
	} else {
		slog.Debug("columnDefToMap: NO default for column",
			"column", col.Name)
	}
	if col.Backfill != nil {
		m["backfill"] = col.Backfill
	}
	if col.Format != "" {
		m["format"] = col.Format
	}
	if col.Pattern != "" {
		m["pattern"] = col.Pattern
	}
	if col.Min != nil {
		m["min"] = *col.Min
	}
	if col.Max != nil {
		m["max"] = *col.Max
	}
	if col.Docs != "" {
		m["docs"] = col.Docs
	}
	if col.Deprecated != "" {
		m["deprecated"] = col.Deprecated
	}
	if col.Hidden {
		m["x_hidden"] = true
	}
	if col.XRef != "" {
		m["x_ref"] = col.XRef
	}

	if col.Reference != nil {
		m["reference"] = map[string]any{
			"table":     col.Reference.Table,
			"column":    col.Reference.Column,
			"on_delete": col.Reference.OnDelete,
			"on_update": col.Reference.OnUpdate,
		}
	}

	if col.Computed != nil {
		m["computed"] = col.Computed
	}

	return m
}

func indexDefToMap(idx *IndexDef) map[string]any {
	cols := make([]any, len(idx.Columns))
	for i, c := range idx.Columns {
		cols[i] = c
	}

	m := map[string]any{
		"columns": cols,
		"unique":  idx.Unique,
	}
	if idx.Name != "" {
		m["name"] = idx.Name
	}
	if idx.Where != "" {
		m["where"] = idx.Where
	}
	return m
}

func relationshipDefToMap(rel *RelationshipDef) map[string]any {
	switch rel.Type {
	case "many_to_many":
		relMap := map[string]any{
			"type":   "many_to_many",
			"target": rel.Target,
		}
		if rel.As != "" {
			relMap["as"] = rel.As
		}
		return map[string]any{
			"_type":        "relationship",
			"relationship": relMap,
		}
	case "polymorphic":
		return map[string]any{
			"_type": "polymorphic",
			"polymorphic": map[string]any{
				"targets": rel.Targets,
				"as":      rel.As,
			},
		}
	default:
		return map[string]any{}
	}
}
func extractTableName(ref string) string {
	// Find the table part after the last dot
	for i := len(ref) - 1; i >= 0; i-- {
		if ref[i] == '.' {
			return ref[i+1:]
		}
	}
	return ref
}
