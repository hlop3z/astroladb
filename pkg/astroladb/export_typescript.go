package astroladb

import (
	"fmt"
	"strings"

	"github.com/hlop3z/astroladb/internal/ast"
	"github.com/hlop3z/astroladb/internal/strutil"
	"github.com/hlop3z/astroladb/internal/types"
)

// TypeScriptConverter converts types to TypeScript.
type TypeScriptConverter struct {
	BaseConverter
}

// NewTypeScriptConverter creates a new TypeScript converter.
func NewTypeScriptConverter() *TypeScriptConverter {
	return &TypeScriptConverter{
		BaseConverter: BaseConverter{
			TypeMap: map[string]string{
				"id":       "string",
				"uuid":     "string",
				"string":   "string",
				"text":     "string",
				"integer":  "number",
				"float":    "number",
				"decimal":  "string",
				"boolean":  "boolean",
				"date":     "string",
				"time":     "string",
				"datetime": "string",
				"json":     "any",
				"base64":   "string",
			},
			NullableFormat: "%s | null",
		},
	}
}

// ConvertType converts a column definition to TypeScript type.
func (c *TypeScriptConverter) ConvertType(col *ast.ColumnDef) string {
	// Handle enum specially - create union type from values
	if col.Type == "enum" {
		enumValues := getEnumValues(col)
		if len(enumValues) > 0 {
			var quotedValues []string
			for _, v := range enumValues {
				escaped := strings.ReplaceAll(v, "'", "\\'")
				quotedValues = append(quotedValues, fmt.Sprintf("'%s'", escaped))
			}
			return strings.Join(quotedValues, " | ")
		}
	}

	// Use TypeRegistry for all other types
	if typeDef := types.Get(col.Type); typeDef != nil {
		return typeDef.TSType
	}
	return "unknown"
}

// tsReservedKeywords contains TypeScript/JavaScript reserved words that cannot be used as identifiers.
var tsReservedKeywords = map[string]bool{
	"break": true, "case": true, "catch": true, "class": true, "const": true,
	"continue": true, "debugger": true, "default": true, "delete": true, "do": true,
	"else": true, "enum": true, "export": true, "extends": true, "false": true,
	"finally": true, "for": true, "function": true, "if": true, "import": true,
	"in": true, "instanceof": true, "new": true, "null": true, "return": true,
	"super": true, "switch": true, "this": true, "throw": true, "true": true,
	"try": true, "typeof": true, "var": true, "void": true, "while": true,
	"with": true, "yield": true, "let": true, "static": true, "implements": true,
	"interface": true, "package": true, "private": true, "protected": true, "public": true,
	"type": true, "async": true, "await": true,
}

// FormatName formats a name to TypeScript conventions (camelCase for fields).
// Reserved keywords are suffixed with an underscore.
func (c *TypeScriptConverter) FormatName(name string) string {
	formatted := strutil.ToCamelCase(name)
	if tsReservedKeywords[formatted] {
		return formatted + "_"
	}
	return formatted
}

func exportTypeScript(tables []*ast.TableDef, cfg *exportContext) ([]byte, error) {
	var sb strings.Builder

	sb.WriteString("// Auto-generated TypeScript types from database schema\n")
	sb.WriteString("// Do not edit manually\n\n")

	// Sort tables for deterministic output
	sortedTables := sortTablesByQualifiedName(tables)

	for i, table := range sortedTables {
		if i > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString(tableToTypeScript(table, cfg))
	}

	// Generate WithRelations variants if enabled
	if cfg.Relations {
		sb.WriteString("\n// WithRelations variants (includes relationship fields)\n\n")
		for _, table := range sortedTables {
			if table.Namespace == "" {
				continue
			}
			generateWithRelationsTypeScript(&sb, table, sortedTables, cfg)
		}
	}

	// Generate schema URI to type name mapping
	sb.WriteString("\n// Schema URI to type name mapping\n")
	sb.WriteString("export const TYPES: Record<string, string> = {\n")
	for _, table := range sortedTables {
		if table.Namespace == "" {
			continue // Skip join tables
		}
		typeName := strutil.ToPascalCase(table.FullName())
		sb.WriteString(fmt.Sprintf("  \"%s\": \"%s\",\n", table.QualifiedName(), typeName))
	}
	sb.WriteString("};\n")

	return []byte(sb.String()), nil
}

// tableToTypeScript converts a table definition to TypeScript interface.
func tableToTypeScript(table *ast.TableDef, cfg *exportContext) string {
	var sb strings.Builder

	name := strutil.ToPascalCase(table.FullName())

	// Add JSDoc comment
	if table.Docs != "" {
		sb.WriteString("/**\n")
		sb.WriteString(fmt.Sprintf(" * %s\n", escapeJSDoc(table.Docs)))
		if table.Deprecated != "" {
			sb.WriteString(fmt.Sprintf(" * @deprecated %s\n", escapeJSDoc(table.Deprecated)))
		}
		sb.WriteString(" */\n")
	}

	sb.WriteString(fmt.Sprintf("export interface %s {\n", name))

	for _, col := range table.Columns {
		// Add JSDoc for column
		if col.Docs != "" {
			sb.WriteString(fmt.Sprintf("  /** %s */\n", escapeJSDoc(col.Docs)))
		}

		optional := ""
		if col.Nullable {
			optional = "?"
		}

		tsType := columnToTypeScriptType(col)
		sb.WriteString(fmt.Sprintf("  %s%s: %s;\n", col.Name, optional, tsType))
	}

	sb.WriteString("}\n")

	return sb.String()
}

// columnToTypeScriptType converts a column type to TypeScript type.
func columnToTypeScriptType(col *ast.ColumnDef) string {
	return GetConverter(FormatTypeScript).ConvertType(col)
}
