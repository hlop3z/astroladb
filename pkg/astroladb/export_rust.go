package astroladb

import (
	"fmt"
	"sort"
	"strings"

	"github.com/hlop3z/astroladb/internal/ast"
	"github.com/hlop3z/astroladb/internal/strutil"
)

// RustConverter converts types to Rust.
type RustConverter struct {
	BaseConverter
	UseChrono bool
	TableName string // Used for enum type naming
}

// NewRustConverter creates a new Rust converter.
func NewRustConverter(useChrono bool) *RustConverter {
	typeMap := map[string]string{
		"id":       "String",
		"uuid":     "String",
		"string":   "String",
		"text":     "String",
		"integer":  "i32",
		"float":    "f32",
		"decimal":  "String",
		"boolean":  "bool",
		"date":     "String",
		"time":     "String",
		"datetime": "String",
		"json":     "serde_json::Value",
		"base64":   "Vec<u8>",
	}

	// Override time types if using chrono
	if useChrono {
		typeMap["date"] = "NaiveDate"
		typeMap["time"] = "NaiveTime"
		typeMap["datetime"] = "DateTime<Utc>"
	}

	return &RustConverter{
		BaseConverter: BaseConverter{
			TypeMap:        typeMap,
			NullableFormat: "Option<%s>",
		},
		UseChrono: useChrono,
	}
}

// ConvertType converts a column definition to Rust type.
func (c *RustConverter) ConvertType(col *ast.ColumnDef) string {
	// Handle enum specially
	if col.Type == "enum" && c.TableName != "" {
		return strutil.ToPascalCase(c.TableName) + strutil.ToPascalCase(col.Name)
	}

	// Check if UseChrono overrides time types
	if c.UseChrono {
		switch col.Type {
		case "date":
			return "NaiveDate"
		case "time":
			return "NaiveTime"
		case "datetime":
			return "DateTime<Utc>"
		}
	}

	if baseType, ok := c.TypeMap[col.Type]; ok {
		return baseType
	}
	return "String"
}

// FormatName formats a name to Rust conventions (snake_case).
func (c *RustConverter) FormatName(name string) string {
	return strutil.ToSnakeCase(name)
}

func exportRust(tables []*ast.TableDef, cfg *exportContext) ([]byte, error) {
	var sb strings.Builder

	sb.WriteString("// Auto-generated Rust types from database schema\n")
	sb.WriteString("// Do not edit manually\n\n")
	if cfg.UseMik {
		sb.WriteString("use mik_sdk::prelude::*;\n")
	} else {
		sb.WriteString("use serde::{Deserialize, Serialize};\n")
		if cfg.UseChrono {
			sb.WriteString("use chrono::{DateTime, NaiveDate, NaiveTime, Utc};\n")
		}
	}
	sb.WriteString("\n")

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
				if cfg.UseMik {
					sb.WriteString("#[derive(Type)]\n")
				} else {
					sb.WriteString("#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq)]\n")
					sb.WriteString("#[serde(rename_all = \"snake_case\")]\n")
				}
				sb.WriteString(fmt.Sprintf("pub enum %s {\n", enumName))
				enumValues := getEnumValues(col)
				for _, v := range enumValues {
					sb.WriteString(fmt.Sprintf("    %s,\n", strutil.ToPascalCase(v)))
				}
				sb.WriteString("}\n\n")
			}
		}
	}

	// Second pass: generate structs
	for _, table := range sortedTables {
		generateRustStruct(&sb, table, cfg)
	}

	// Generate WithRelations variants if enabled
	if cfg.Relations {
		sb.WriteString("// WithRelations variants (includes relationship fields)\n\n")
		for _, table := range sortedTables {
			if table.Namespace == "" {
				continue
			}
			generateWithRelationsRust(&sb, table, sortedTables, cfg)
		}
	}

	// Generate schema URI to type name mapping
	sb.WriteString("/// Schema URI to type name mapping.\n")
	sb.WriteString("pub const TYPES: &[(&str, &str)] = &[\n")
	for _, table := range sortedTables {
		if table.Namespace == "" {
			continue // Skip join tables
		}
		typeName := strutil.ToPascalCase(table.FullName())
		sb.WriteString(fmt.Sprintf("    (\"%s\", \"%s\"),\n", table.QualifiedName(), typeName))
	}
	sb.WriteString("];\n")

	return []byte(sb.String()), nil
}

// generateRustStruct generates a Rust struct for a single table.
func generateRustStruct(sb *strings.Builder, table *ast.TableDef, cfg *exportContext) {
	name := strutil.ToPascalCase(table.FullName())

	// Add doc comment if present
	if table.Docs != "" {
		fmt.Fprintf(sb, "/// %s\n", table.Docs)
	}

	if cfg.UseMik {
		sb.WriteString("#[derive(Type)]\n")
	} else {
		sb.WriteString("#[derive(Debug, Clone, Serialize, Deserialize)]\n")
		sb.WriteString("#[serde(rename_all = \"snake_case\")]\n")
	}
	fmt.Fprintf(sb, "pub struct %s {\n", name)

	for _, col := range table.Columns {
		rustType := columnToRustType(col, table, cfg)

		// Add doc comment if present
		if col.Docs != "" {
			fmt.Fprintf(sb, "    /// %s\n", col.Docs)
		}

		// Handle Rust reserved keywords
		fieldName := col.Name
		if isRustKeyword(col.Name) {
			if !cfg.UseMik {
				fmt.Fprintf(sb, "    #[serde(rename = \"%s\")]\n", col.Name)
			}
			fieldName = col.Name + "_"
		}

		fmt.Fprintf(sb, "    pub %s: %s,\n", fieldName, rustType)
	}

	sb.WriteString("}\n\n")
}

// isRustKeyword checks if a name is a Rust reserved keyword.
func isRustKeyword(name string) bool {
	keywords := map[string]bool{
		"as": true, "break": true, "const": true, "continue": true,
		"crate": true, "else": true, "enum": true, "extern": true,
		"false": true, "fn": true, "for": true, "if": true,
		"impl": true, "in": true, "let": true, "loop": true,
		"match": true, "mod": true, "move": true, "mut": true,
		"pub": true, "ref": true, "return": true, "self": true,
		"Self": true, "static": true, "struct": true, "super": true,
		"trait": true, "true": true, "type": true, "unsafe": true,
		"use": true, "where": true, "while": true, "async": true,
		"await": true, "dyn": true,
	}
	return keywords[name]
}

// columnToRustType converts a column type to Rust type.
func columnToRustType(col *ast.ColumnDef, table *ast.TableDef, cfg *exportContext) string {
	// Create a fresh converter to avoid mutating the shared global instance
	converter := NewRustConverter(cfg.UseChrono)
	converter.TableName = table.FullName()
	baseType := converter.ConvertType(col)

	if col.Nullable {
		return fmt.Sprintf("Option<%s>", baseType)
	}
	return baseType
}

// getEnumValues extracts enum values from a column definition.
func getEnumValues(col *ast.ColumnDef) []string {
	if len(col.TypeArgs) == 0 {
		return nil
	}
	if values, ok := col.TypeArgs[0].([]string); ok {
		return values
	}
	if values, ok := col.TypeArgs[0].([]any); ok {
		var result []string
		for _, v := range values {
			if s, ok := v.(string); ok {
				result = append(result, s)
			}
		}
		return result
	}
	return nil
}
