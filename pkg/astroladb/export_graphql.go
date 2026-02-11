package astroladb

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/hlop3z/astroladb/internal/ast"
	"github.com/hlop3z/astroladb/internal/strutil"
)

// GraphQLConverter converts types to GraphQL.
type GraphQLConverter struct {
	BaseConverter
	TableName string // Used for enum type naming
}

// NewGraphQLConverter creates a new GraphQL converter.
func NewGraphQLConverter() *GraphQLConverter {
	return &GraphQLConverter{
		BaseConverter: BaseConverter{
			TypeMap: map[string]string{
				"id":       "ID",
				"uuid":     "ID",
				"string":   "String",
				"text":     "String",
				"integer":  "Int",
				"float":    "Float",
				"decimal":  "String",
				"boolean":  "Boolean",
				"date":     "DateTime",
				"time":     "DateTime",
				"datetime": "DateTime",
				"json":     "JSON",
				"base64":   "String",
			},
			NullableFormat: "%s",
		},
	}
}

// ConvertType converts a column definition to GraphQL type.
func (c *GraphQLConverter) ConvertType(col *ast.ColumnDef) string {
	// Handle enum specially
	if col.Type == "enum" && c.TableName != "" {
		return strutil.ToPascalCase(c.TableName) + strutil.ToPascalCase(col.Name)
	}

	if baseType, ok := c.TypeMap[col.Type]; ok {
		return baseType
	}
	return "String"
}

// ConvertNullable applies the nullable format to a base type for GraphQL (uses ! for non-null).
func (c *GraphQLConverter) ConvertNullable(baseType string, nullable bool) string {
	if nullable {
		return baseType
	}
	return baseType + "!"
}

// FormatName formats a name to GraphQL conventions (camelCase).
func (c *GraphQLConverter) FormatName(name string) string {
	return strutil.ToCamelCase(name)
}

func exportGraphQLExamples(tables []*ast.TableDef, cfg *exportContext) ([]byte, error) {
	examples := make(map[string]any)

	for _, table := range tables {
		if table.Namespace == "" {
			continue // Skip join tables
		}
		typeName := lowerFirst(strutil.ToPascalCase(table.FullName()))
		examples[typeName] = generateTableExample(table, false)
	}

	return json.MarshalIndent(examples, "", "  ")
}

// exportGraphQL generates a GraphQL schema from the database schema.
func exportGraphQL(tables []*ast.TableDef, cfg *exportContext) ([]byte, error) {
	var sb strings.Builder

	sb.WriteString("# Auto-generated GraphQL schema from database schema\n")
	sb.WriteString("# Do not edit manually\n\n")

	// Custom scalars
	sb.WriteString("scalar DateTime\n")
	sb.WriteString("scalar JSON\n\n")

	// Sort tables for deterministic output
	sortedTables := make([]*ast.TableDef, len(tables))
	copy(sortedTables, tables)
	sort.Slice(sortedTables, func(i, j int) bool {
		return sortedTables[i].QualifiedName() < sortedTables[j].QualifiedName()
	})

	// First pass: generate enum types
	for _, table := range sortedTables {
		for _, col := range table.Columns {
			if col.Type == "enum" && len(col.TypeArgs) > 0 {
				enumName := strutil.ToPascalCase(table.FullName()) + strutil.ToPascalCase(col.Name)
				if col.Docs != "" {
					sb.WriteString(fmt.Sprintf("\"\"\"%s\"\"\"\n", col.Docs))
				}
				sb.WriteString(fmt.Sprintf("enum %s {\n", enumName))
				enumValues := getEnumValues(col)
				for _, v := range enumValues {
					sb.WriteString(fmt.Sprintf("  %s\n", strings.ToUpper(v)))
				}
				sb.WriteString("}\n\n")
			}
		}
	}

	// Second pass: generate types
	for _, table := range sortedTables {
		generateGraphQLType(&sb, table, cfg)
	}

	// Generate WithRelations variants if enabled
	if cfg.Relations {
		sb.WriteString("# WithRelations variants (includes relationship fields)\n\n")
		for _, table := range sortedTables {
			if table.Namespace == "" {
				continue
			}
			generateWithRelationsGraphQL(&sb, table, sortedTables, cfg)
		}
	}

	// Generate Query type for schema exploration
	sb.WriteString("type Query {\n")
	for _, table := range sortedTables {
		if table.Namespace == "" {
			continue // Skip join tables
		}
		typeName := strutil.ToPascalCase(table.FullName())
		fieldName := lowerFirst(typeName)
		sb.WriteString(fmt.Sprintf("  %s: %s\n", fieldName, typeName))
	}
	sb.WriteString("}\n")

	return []byte(sb.String()), nil
}

// sanitizeIdentifier converts a string to a valid identifier by replacing
// non-alphanumeric characters with underscores.
func sanitizeIdentifier(s string) string {
	var b strings.Builder
	for i, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || r == '_' || (r >= '0' && r <= '9' && i > 0) {
			b.WriteRune(r)
		} else {
			b.WriteRune('_')
		}
	}
	result := b.String()
	if result == "" {
		return "VALUE"
	}
	// Ensure doesn't start with digit
	if result[0] >= '0' && result[0] <= '9' {
		result = "_" + result
	}
	return result
}

// escapePyDoc escapes content for Python docstrings.
func escapePyDoc(s string) string {
	s = strings.ReplaceAll(s, `"""`, `\"\"\"`)
	return s
}

// lowerFirst returns the string with the first character lowercased.
func lowerFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToLower(s[:1]) + s[1:]
}
