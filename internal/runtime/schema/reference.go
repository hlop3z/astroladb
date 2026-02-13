// Package schema provides centralized FK and reference building.
package schema

import "github.com/hlop3z/astroladb/internal/runtime/builder"

// ReferenceBuilder provides methods for creating foreign key columns and indexes.
// It centralizes the logic for belongs_to, one_to_one, and polymorphic relationships.
type ReferenceBuilder struct{}

// NewReferenceBuilder creates a new reference builder.
func NewReferenceBuilder() *ReferenceBuilder {
	return &ReferenceBuilder{}
}

// ReferenceOpts contains options for FK creation.
type ReferenceOpts struct {
	Nullable bool
	Unique   bool
	OnDelete string
	OnUpdate string
}

// BuildFK creates a column and index for a belongs_to relationship.
// The columnName should be the alias (e.g., "author"), and _id will be appended.
func (rb *ReferenceBuilder) BuildFK(ref string, columnName string, opts ReferenceOpts) (*builder.ColumnDef, *builder.IndexDef) {
	fkName := columnName + "_id"

	col := &builder.ColumnDef{
		Name:     fkName,
		Type:     "uuid",
		Nullable: opts.Nullable,
		Unique:   opts.Unique,
		Reference: &builder.RefDef{
			Table:    ref,
			Column:   "id",
			OnDelete: opts.OnDelete,
			OnUpdate: opts.OnUpdate,
		},
		XRef:           ref,
		IsRelationship: true,
	}

	idx := &builder.IndexDef{
		Columns: []string{fkName},
		Unique:  opts.Unique,
	}

	return col, idx
}

// BuildBelongsTo creates a standard belongs_to FK column and index.
func (rb *ReferenceBuilder) BuildBelongsTo(ref string, columnName string) (*builder.ColumnDef, *builder.IndexDef) {
	return rb.BuildFK(ref, columnName, ReferenceOpts{})
}

// BuildOneToOne creates a unique FK column and index for one_to_one relationships.
func (rb *ReferenceBuilder) BuildOneToOne(ref string, columnName string) (*builder.ColumnDef, *builder.IndexDef) {
	return rb.BuildFK(ref, columnName, ReferenceOpts{Unique: true})
}

// PolymorphicResult contains the columns and index for a polymorphic relationship.
type PolymorphicResult struct {
	TypeColumn *builder.ColumnDef
	IDColumn   *builder.ColumnDef
	Index      *builder.IndexDef
}

// BuildPolymorphic creates type + id columns for polymorphic relationships.
// The alias becomes the prefix: alias -> alias_type, alias_id
func (rb *ReferenceBuilder) BuildPolymorphic(targets []string, alias string, nullable bool) *PolymorphicResult {
	typeColName := alias + "_type"
	idColName := alias + "_id"

	typeCol := &builder.ColumnDef{
		Name:     typeColName,
		Type:     "string",
		TypeArgs: []any{100}, // Max 100 chars for type names
		Nullable: nullable,
	}

	idCol := &builder.ColumnDef{
		Name:     idColName,
		Type:     "uuid",
		Nullable: nullable,
	}

	idx := &builder.IndexDef{
		Columns: []string{typeColName, idColName},
		Unique:  false,
	}

	return &PolymorphicResult{
		TypeColumn: typeCol,
		IDColumn:   idCol,
		Index:      idx,
	}
}

// ExtractTableName extracts the table name from a reference.
// With singular table convention: "auth.user" -> "user", "core.order" -> "order"
func ExtractTableName(ref string) string {
	// Find the table part after the last dot
	for i := len(ref) - 1; i >= 0; i-- {
		if ref[i] == '.' {
			return ref[i+1:]
		}
	}
	return ref
}

// DefaultColumnName returns the default FK column name for a reference.
// "auth.user" -> "user_id"
func DefaultColumnName(ref string) string {
	return ExtractTableName(ref) + "_id"
}

// RelationshipDef represents relationship metadata (many_to_many, polymorphic).
// This is used for storing relationship info that isn't directly a column.
type RelationshipMeta struct {
	Type    string   // "many_to_many" or "polymorphic"
	Target  string   // Target table for many_to_many
	Targets []string // Target tables for polymorphic
	Alias   string   // Optional alias
}

// NewManyToMany creates metadata for a many_to_many relationship.
func NewManyToMany(target string, alias string) *RelationshipMeta {
	return &RelationshipMeta{
		Type:   "many_to_many",
		Target: target,
		Alias:  alias,
	}
}

// NewPolymorphic creates metadata for a polymorphic relationship.
func NewPolymorphic(targets []string, alias string) *RelationshipMeta {
	return &RelationshipMeta{
		Type:    "polymorphic",
		Targets: targets,
		Alias:   alias,
	}
}
