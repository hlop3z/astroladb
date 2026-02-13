package astroladb

import (
	"encoding/json"
	"fmt"
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
	sortedTables := sortTablesByQualifiedName(tables)

	// First pass: generate enum types
	forEachEnum(sortedTables, func(e enumInfo) {
		if e.Column.Docs != "" {
			sb.WriteString(fmt.Sprintf("\"\"\"%s\"\"\"\n", e.Column.Docs))
		}
		sb.WriteString(fmt.Sprintf("enum %s {\n", e.EnumName))
		for _, v := range e.Values {
			sb.WriteString(fmt.Sprintf("  %s\n", strings.ToUpper(v)))
		}
		sb.WriteString("}\n\n")
	})

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
