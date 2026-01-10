// Package dsl provides fluent builders for defining database schemas and migrations.
package dsl

// RelationshipBuilder provides a fluent API for building foreign key relationships.
// It supports chained configuration: t.belongs_to("auth.user").as("author").optional().on_delete("cascade")
type RelationshipBuilder struct {
	// Ref is the target table reference (e.g., "auth.user")
	Ref string
	// Alias is the custom name for the FK column (without _id suffix)
	Alias string
	// Optional if true, the FK is nullable
	Optional bool
	// OnDelete action (CASCADE, SET NULL, RESTRICT, NO ACTION, SET DEFAULT)
	OnDelete string
	// OnUpdate action (CASCADE, SET NULL, RESTRICT, NO ACTION, SET DEFAULT)
	OnUpdate string
	// Unique if true, creates a unique constraint (for one_to_one)
	Unique bool
}

// NewRelationshipBuilder creates a new relationship builder for the given reference.
func NewRelationshipBuilder(ref string) *RelationshipBuilder {
	return &RelationshipBuilder{
		Ref: ref,
	}
}

// As sets a custom alias for the foreign key column.
// Example: belongs_to("auth.user").as("author") creates author_id column.
func (rb *RelationshipBuilder) As(alias string) *RelationshipBuilder {
	rb.Alias = alias
	return rb
}

// SetOptional marks the foreign key as nullable.
// Alias for clearer relationship semantics.
func (rb *RelationshipBuilder) SetOptional() *RelationshipBuilder {
	rb.Optional = true
	return rb
}

// SetOnDelete sets the ON DELETE action for the foreign key.
// Supported: "cascade", "set null", "restrict", "no action", "set default"
func (rb *RelationshipBuilder) SetOnDelete(action string) *RelationshipBuilder {
	rb.OnDelete = action
	return rb
}

// SetOnUpdate sets the ON UPDATE action for the foreign key.
// Supported: "cascade", "set null", "restrict", "no action", "set default"
func (rb *RelationshipBuilder) SetOnUpdate(action string) *RelationshipBuilder {
	rb.OnUpdate = action
	return rb
}

// SetUnique marks the foreign key as unique (for one_to_one relationships).
func (rb *RelationshipBuilder) SetUnique() *RelationshipBuilder {
	rb.Unique = true
	return rb
}

// ColumnName returns the computed column name for this relationship.
// Uses alias if set, otherwise extracts table name from ref.
func (rb *RelationshipBuilder) ColumnName() string {
	name := rb.Alias
	if name == "" {
		name = extractTableNameFromRef(rb.Ref)
	}
	return name + "_id"
}

// extractTableNameFromRef extracts the table name from a reference.
// "auth.user" -> "user", "core.order" -> "order"
func extractTableNameFromRef(ref string) string {
	for i := len(ref) - 1; i >= 0; i-- {
		if ref[i] == '.' {
			return ref[i+1:]
		}
	}
	return ref
}

// ToColumnDef converts the relationship to a column definition map.
// This is used by the JS runtime to add the column.
func (rb *RelationshipBuilder) ToColumnDef() map[string]any {
	col := map[string]any{
		"name":     rb.ColumnName(),
		"type":     "uuid",
		"nullable": rb.Optional,
		"reference": map[string]any{
			"table":     rb.Ref,
			"column":    "id",
			"on_delete": rb.OnDelete,
			"on_update": rb.OnUpdate,
		},
		"x_ref": rb.Ref,
	}
	if rb.Unique {
		col["unique"] = true
	}
	return col
}

// ToIndexDef returns the index definition for this relationship.
func (rb *RelationshipBuilder) ToIndexDef() map[string]any {
	return map[string]any{
		"columns": []any{rb.ColumnName()},
		"unique":  rb.Unique,
	}
}
