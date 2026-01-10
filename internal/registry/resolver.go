package registry

import (
	"strings"

	"github.com/hlop3z/astroladb/internal/alerr"
	"github.com/hlop3z/astroladb/internal/ast"
	"github.com/hlop3z/astroladb/internal/strutil"
)

// Reference Resolution Rules:
//
// | Input        | Current NS | Resolves To  | SQL Name     |
// |--------------|------------|--------------|--------------|
// | auth.users   | any        | auth.users   | auth_users   |
// | .roles       | auth       | auth.roles   | auth_roles   |
// | posts        | blog       | blog.posts   | blog_posts   |
//
// - "auth.users" → fully qualified, always resolves to auth namespace
// - ".roles"     → relative (leading dot), resolves to current namespace
// - "posts"      → unqualified, assumes current namespace

// ParseReference parses a reference string into its components.
// Returns namespace, table, and whether the reference is relative (starts with dot).
//
// Examples:
//   - "auth.users" -> ("auth", "users", false)
//   - ".roles"     -> ("", "roles", true)
//   - "posts"      -> ("", "posts", false)
func ParseReference(ref string) (namespace, table string, isRelative bool) {
	if ref == "" {
		return "", "", false
	}

	// Check for relative reference (leading dot)
	if strings.HasPrefix(ref, ".") {
		return "", ref[1:], true
	}

	// Check for fully qualified reference (contains dot)
	if idx := strings.Index(ref, "."); idx > 0 {
		return ref[:idx], ref[idx+1:], false
	}

	// Unqualified - just a table name
	return "", ref, false
}

// NormalizeReference converts any reference format to a fully qualified "ns.table" format.
// The currentNS is used for relative and unqualified references.
//
// Examples (with currentNS="auth"):
//   - "blog.posts" -> "blog.posts"
//   - ".roles"     -> "auth.roles"
//   - "users"      -> "auth.users"
func NormalizeReference(ref string, currentNS string) (string, error) {
	ns, table, isRelative := ParseReference(ref)

	if table == "" {
		return "", alerr.New(alerr.ErrInvalidReference, "empty table name in reference").
			With("ref", ref)
	}

	// Fully qualified reference - use as-is
	if ns != "" {
		return ns + "." + table, nil
	}

	// Relative or unqualified - need current namespace
	if currentNS == "" {
		if isRelative {
			return "", alerr.New(alerr.ErrInvalidReference, "relative reference requires current namespace").
				With("ref", ref).
				With("hint", "use fully qualified reference (namespace.table)")
		}
		return "", alerr.New(alerr.ErrInvalidReference, "unqualified reference requires current namespace").
			With("ref", ref).
			With("hint", "use fully qualified reference (namespace.table)")
	}

	return currentNS + "." + table, nil
}

// Resolve resolves a reference to a table definition.
// The reference can be in any of the supported formats.
//
// Reference formats:
//   - "auth.users" → fully qualified, always uses auth namespace
//   - ".roles"     → relative, uses currentNS
//   - "users"      → unqualified, uses currentNS
func (r *ModelRegistry) Resolve(ref string, currentNS string) (*ast.TableDef, error) {
	normalized, err := NormalizeReference(ref, currentNS)
	if err != nil {
		return nil, err
	}

	def, err := r.GetByRef(normalized)
	if err != nil {
		// Add more context to the error
		return nil, alerr.New(alerr.ErrSchemaNotFound, "referenced table not found").
			With("ref", ref).
			With("resolved_to", normalized).
			With("current_ns", currentNS)
	}

	return def, nil
}

// SQLName resolves a reference and returns the flat SQL table name.
// Returns an error if the reference cannot be resolved.
//
// Examples (with currentNS="auth"):
//   - "blog.posts" -> "blog_posts"
//   - ".roles"     -> "auth_roles"
//   - "users"      -> "auth_users"
func (r *ModelRegistry) SQLName(ref string, currentNS string) (string, error) {
	ns, table, _ := ParseReference(ref)

	// If fully qualified, we can compute SQL name without lookup
	if ns != "" {
		return strutil.SQLName(ns, table), nil
	}

	// For relative/unqualified, need current namespace
	if currentNS == "" {
		return "", alerr.New(alerr.ErrInvalidReference, "cannot compute SQL name without namespace").
			With("ref", ref).
			With("hint", "provide currentNS or use fully qualified reference")
	}

	return strutil.SQLName(currentNS, table), nil
}

// SQLNameChecked resolves a reference, verifies the table exists, and returns the SQL name.
// This is slower than SQLName but validates the reference.
func (r *ModelRegistry) SQLNameChecked(ref string, currentNS string) (string, error) {
	def, err := r.Resolve(ref, currentNS)
	if err != nil {
		return "", err
	}
	return def.SQLName(), nil
}

// ValidateReferences validates all references in a table definition.
// Returns an error if any referenced table does not exist.
func (r *ModelRegistry) ValidateReferences(def *ast.TableDef) error {
	if def == nil {
		return alerr.New(alerr.ErrSchemaInvalid, "table definition cannot be nil")
	}

	currentNS := def.Namespace

	for _, col := range def.Columns {
		if col.Reference == nil {
			continue
		}

		ref := col.Reference.Table
		if ref == "" {
			return alerr.New(alerr.ErrInvalidReference, "foreign key reference has empty table").
				WithTable(def.Namespace, def.Name).
				WithColumn(col.Name)
		}

		_, err := r.Resolve(ref, currentNS)
		if err != nil {
			return alerr.New(alerr.ErrInvalidReference, "foreign key references non-existent table").
				WithTable(def.Namespace, def.Name).
				WithColumn(col.Name).
				With("referenced_table", ref)
		}
	}

	return nil
}

// ValidateAllReferences validates references for all registered tables.
// Returns all validation errors, not just the first one.
func (r *ModelRegistry) ValidateAllReferences() []error {
	r.mu.RLock()
	tables := make([]*ast.TableDef, 0, len(r.tables))
	for _, def := range r.tables {
		tables = append(tables, def)
	}
	r.mu.RUnlock()

	var errs []error
	for _, def := range tables {
		if err := r.ValidateReferences(def); err != nil {
			errs = append(errs, err)
		}
	}
	return errs
}

// ResolveColumnReference resolves a reference from a column's Reference field.
// This is a convenience method for resolving foreign key references.
func (r *ModelRegistry) ResolveColumnReference(col *ast.ColumnDef, currentNS string) (*ast.TableDef, error) {
	if col == nil {
		return nil, alerr.New(alerr.ErrSchemaInvalid, "column definition cannot be nil")
	}
	if col.Reference == nil {
		return nil, alerr.New(alerr.ErrInvalidReference, "column has no reference").
			WithColumn(col.Name)
	}
	return r.Resolve(col.Reference.Table, currentNS)
}
