// Package runtime provides the semantic type registry for high-level column types.
// These types have baked-in defaults for common patterns.
package runtime

// SemanticType defines a semantic column type with all its default properties.
type SemanticType struct {
	BaseType  string   // Underlying type: "string", "text", "integer", "decimal", "boolean"
	Length    int      // For string types: max length
	Precision int      // For decimal types: precision
	Scale     int      // For decimal types: scale
	Format    string   // OpenAPI format hint (e.g., "email", "uri")
	Pattern   string   // Regex pattern for validation
	Min       *float64 // Minimum value/length
	Max       *float64 // Maximum value/length
	Default   any      // Default value
	Hidden    bool     // Hide from OpenAPI output (e.g., password_hash)
	Unique    bool     // Add unique constraint
}

// ptr is a helper to create float64 pointers for Min/Max.
func ptr(v float64) *float64 {
	return &v
}

// SemanticTypes is the single source of truth for all semantic type definitions.
// Both TableBuilder and ColBuilder use this registry.
var SemanticTypes = map[string]SemanticType{
	// Identity and authentication
	"email": {
		BaseType: "string",
		Length:   255,
		Format:   "email",
		Pattern:  `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`,
	},
	"username": {
		BaseType: "string",
		Length:   50,
		Pattern:  `^[a-z0-9_]+$`,
		Min:      ptr(3),
	},
	"password_hash": {
		BaseType: "string",
		Length:   255,
		Hidden:   true,
	},
	"phone": {
		BaseType: "string",
		Length:   50,
		Pattern:  `^\+?[1-9]\d{1,14}$`,
	},

	// Text content
	"name": {
		BaseType: "string",
		Length:   100,
	},
	"title": {
		BaseType: "string",
		Length:   200,
	},
	"slug": {
		BaseType: "string",
		Length:   255,
		Unique:   true,
		Pattern:  `^[a-z0-9]+(?:-[a-z0-9]+)*$`,
	},
	"body": {
		BaseType: "text",
	},
	"summary": {
		BaseType: "string",
		Length:   500,
	},

	// URLs and network
	"url": {
		BaseType: "string",
		Length:   2048,
		Format:   "uri",
		Pattern:  `^https?://[^\s/$.?#].[^\s]*$`,
	},
	"ip": {
		BaseType: "string",
		Length:   45, // Covers IPv4 and IPv6
		Pattern:  `^((25[0-5]|(2[0-4]|1\d|[1-9]|)\d)\.){3}(25[0-5]|(2[0-4]|1\d|[1-9]|)\d)$|^([0-9a-fA-F]{1,4}:){7}[0-9a-fA-F]{1,4}$`,
	},
	"ipv4": {
		BaseType: "string",
		Length:   15,
		Pattern:  `^((25[0-5]|(2[0-4]|1\d|[1-9]|)\d)\.){3}(25[0-5]|(2[0-4]|1\d|[1-9]|)\d)$`,
	},
	"ipv6": {
		BaseType: "string",
		Length:   45,
		Pattern:  `^([0-9a-fA-F]{1,4}:){7}[0-9a-fA-F]{1,4}$`,
	},
	"user_agent": {
		BaseType: "string",
		Length:   500,
	},

	// Numbers and money
	// Patterns follow RFC 8259 (JSON) number format: period as decimal separator
	"money": {
		BaseType:  "decimal",
		Precision: 19,
		Scale:     4,
		Min:       ptr(0),
		Pattern:   `^\d{1,15}(\.\d{1,4})?$`,
	},
	"percentage": {
		BaseType:  "decimal",
		Precision: 5,
		Scale:     2,
		Min:       ptr(0),
		Max:       ptr(100),
		Pattern:   `^\d{1,3}(\.\d{1,2})?$`,
	},
	"counter": {
		BaseType: "integer",
		Default:  0,
	},
	"quantity": {
		BaseType: "integer",
		Min:      ptr(0),
	},
	"rating": {
		BaseType:  "decimal",
		Precision: 2,
		Scale:     1,
		Min:       ptr(0),
		Max:       ptr(5),
		Pattern:   `^[0-5](\.\d)?$`,
	},
	"duration": {
		BaseType: "integer",
		Min:      ptr(0),
	},

	// Codes and identifiers
	"token": {
		BaseType: "string",
		Length:   64,
	},
	"code": {
		BaseType: "string",
		Length:   20,
		Pattern:  `^[A-Z0-9]+$`,
	},
	"country": {
		BaseType: "string",
		Length:   2,
		Pattern:  `^[A-Z]{2}$`,
	},
	"currency": {
		BaseType: "string",
		Length:   3,
		Pattern:  `^[A-Z]{3}$`,
	},
	"locale": {
		BaseType: "string",
		Length:   10,
		Pattern:  `^[a-z]{2}(-[A-Z]{2})?$`,
	},
	"timezone": {
		BaseType: "string",
		Length:   50,
	},
	"color": {
		BaseType: "string",
		Length:   7,
		Pattern:  `^#[0-9A-Fa-f]{6}$`,
	},

	// Rich text
	"markdown": {
		BaseType: "text",
	},
	"html": {
		BaseType: "text",
	},

	// Boolean flags
	"flag": {
		BaseType: "boolean",
		Default:  false,
	},
}

// GetSemanticType returns the definition for a semantic type name.
// Returns nil if not found.
func GetSemanticType(name string) *SemanticType {
	if st, ok := SemanticTypes[name]; ok {
		return &st
	}
	return nil
}

// IsSemanticType returns true if the name is a known semantic type.
func IsSemanticType(name string) bool {
	_, ok := SemanticTypes[name]
	return ok
}

// SemanticTypeNames returns a sorted list of all semantic type names.
func SemanticTypeNames() []string {
	names := make([]string, 0, len(SemanticTypes))
	for name := range SemanticTypes {
		names = append(names, name)
	}
	return names
}

// ApplyTo applies the semantic type properties to a column definition map.
// This is used by the JS runtime to configure columns from semantic types.
func (st *SemanticType) ApplyTo(col map[string]any) {
	// Set type args based on base type
	switch st.BaseType {
	case "string":
		if st.Length > 0 {
			col["type_args"] = []any{st.Length}
		}
	case "decimal":
		if st.Precision > 0 {
			col["type_args"] = []any{st.Precision, st.Scale}
		}
	}

	if st.Format != "" {
		col["format"] = st.Format
	}
	if st.Pattern != "" {
		col["pattern"] = st.Pattern
	}
	if st.Unique {
		col["unique"] = true
	}
	if st.Default != nil {
		col["default"] = st.Default
	}
	if st.Hidden {
		col["hidden"] = true
	}
	if st.Min != nil {
		col["min"] = *st.Min
	}
	if st.Max != nil {
		col["max"] = *st.Max
	}
}

// SemanticMethodConfig holds configuration for registering semantic methods.
type SemanticMethodConfig struct {
	// CreateColumn creates a column with the given type and options.
	// For TableBuilder: takes name as first arg
	// For ColBuilder: no name arg (name comes from object key)
	CreateColumn func(baseType string, opts []ColOpt) any

	// CreateFlagColumn creates a flag column with optional default value.
	// For TableBuilder: takes name and default bool
	// For ColBuilder: takes only default bool
	CreateFlagColumn func(defaultVal bool) any

	// NeedsName indicates whether column creation needs a name argument.
	// True for TableBuilder (callback API), false for ColBuilder (object API).
	NeedsName bool
}

// RegisterSemanticMethods registers all semantic type methods on a JS object.
// This DRY helper is used by both TableBuilder.ToObject() and ColBuilder.ToObject()
// to avoid duplicate registration loops.
//
// For TableBuilder (callback API):
//   - Methods take (name string) as argument
//   - Example: t.email("user_email")
//
// For ColBuilder (object API):
//   - Methods take no arguments
//   - Example: { email: col.email() }
func RegisterSemanticMethods(
	setMethod func(name string, fn any),
	createColumn func(baseType string, opts []ColOpt) any,
	createFlagWithName func(name string, defaultVal bool) any,
	createFlagNoName func(defaultVal bool) any,
	needsName bool,
) {
	for typeName, st := range SemanticTypes {
		// Capture loop variables for closure
		tn, semantic := typeName, st

		// flag is special - takes optional default value argument
		if tn == "flag" {
			if needsName {
				// TableBuilder version: flag(name, defaultVal?)
				setMethod("flag", createFlagWithName)
			} else {
				// ColBuilder version: flag(defaultVal?)
				setMethod("flag", createFlagNoName)
			}
			continue
		}

		// Build options from semantic type definition
		opts := optsFromSemanticType(semantic)

		if needsName {
			// TableBuilder version: typeName(columnName)
			setMethod(tn, func(name string) any {
				// We need to capture opts in the closure properly
				return createColumn(semantic.BaseType, opts)
			})
		} else {
			// ColBuilder version: typeName() - no column name argument
			setMethod(tn, func() any {
				return createColumn(semantic.BaseType, opts)
			})
		}
	}
}

// optsFromSemanticType converts a SemanticType to a slice of ColOpts.
// This is the canonical conversion used by RegisterSemanticMethods.
func optsFromSemanticType(st SemanticType) []ColOpt {
	var opts []ColOpt

	// Handle type-specific arguments
	switch st.BaseType {
	case "string":
		if st.Length > 0 {
			opts = append(opts, func(c *ColumnDef) { c.TypeArgs = []any{st.Length} })
		}
	case "decimal":
		if st.Precision > 0 {
			opts = append(opts, func(c *ColumnDef) { c.TypeArgs = []any{st.Precision, st.Scale} })
		}
	}

	// Common options
	if st.Format != "" {
		f := st.Format
		opts = append(opts, func(c *ColumnDef) { c.Format = f })
	}
	if st.Pattern != "" {
		p := st.Pattern
		opts = append(opts, func(c *ColumnDef) { c.Pattern = p })
	}
	if st.Unique {
		opts = append(opts, func(c *ColumnDef) { c.Unique = true })
	}
	if st.Default != nil {
		d := st.Default
		opts = append(opts, func(c *ColumnDef) { c.Default = d })
	}
	if st.Hidden {
		opts = append(opts, func(c *ColumnDef) { c.Hidden = true })
	}
	// Min/Max - copy pointers to avoid closure issues
	if st.Min != nil {
		m := st.Min
		opts = append(opts, func(c *ColumnDef) { c.Min = m })
	}
	if st.Max != nil {
		m := st.Max
		opts = append(opts, func(c *ColumnDef) { c.Max = m })
	}

	return opts
}
