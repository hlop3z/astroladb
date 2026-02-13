package astroladb

import (
	"fmt"
	"strings"

	"github.com/hlop3z/astroladb/internal/ast"
	"github.com/hlop3z/astroladb/internal/strutil"
)

// PythonConverter converts types to Python.
type PythonConverter struct {
	BaseConverter
	TableName string // Used for enum type naming
}

// NewPythonConverter creates a new Python converter.
func NewPythonConverter() *PythonConverter {
	return &PythonConverter{
		BaseConverter: BaseConverter{
			TypeMap: map[string]string{
				"id":       "str",
				"uuid":     "str",
				"string":   "str",
				"text":     "str",
				"integer":  "int",
				"float":    "float",
				"decimal":  "str",
				"boolean":  "bool",
				"date":     "date",
				"time":     "time",
				"datetime": "datetime",
				"json":     "Any",
				"base64":   "bytes",
			},
			NullableFormat: "Optional[%s]",
		},
	}
}

// ConvertType converts a column definition to Python type.
func (c *PythonConverter) ConvertType(col *ast.ColumnDef) string {
	// Handle enum specially
	if col.Type == "enum" && c.TableName != "" {
		return strutil.ToPascalCase(c.TableName) + strutil.ToPascalCase(col.Name)
	}

	if baseType, ok := c.TypeMap[col.Type]; ok {
		return baseType
	}
	return "str"
}

// pyReservedKeywords contains Python reserved words that cannot be used as identifiers.
var pyReservedKeywords = map[string]bool{
	"False": true, "None": true, "True": true, "and": true, "as": true,
	"assert": true, "async": true, "await": true, "break": true, "class": true,
	"continue": true, "def": true, "del": true, "elif": true, "else": true,
	"except": true, "finally": true, "for": true, "from": true, "global": true,
	"if": true, "import": true, "in": true, "is": true, "lambda": true,
	"nonlocal": true, "not": true, "or": true, "pass": true, "raise": true,
	"return": true, "try": true, "while": true, "with": true, "yield": true,
	"type": true,
}

// FormatName formats a name to Python conventions (snake_case).
// Reserved keywords are suffixed with an underscore.
func (c *PythonConverter) FormatName(name string) string {
	formatted := strutil.ToSnakeCase(name)
	if pyReservedKeywords[formatted] {
		return formatted + "_"
	}
	return formatted
}

func exportPython(tables []*ast.TableDef, cfg *exportContext) ([]byte, error) {
	var sb strings.Builder

	sb.WriteString("# Auto-generated Python types from database schema\n")
	sb.WriteString("# Do not edit manually\n\n")
	sb.WriteString("from __future__ import annotations\n")
	sb.WriteString("from dataclasses import dataclass\n")
	sb.WriteString("from typing import Optional, Any\n")
	sb.WriteString("from datetime import datetime, date, time\n")
	sb.WriteString("from enum import Enum\n\n")

	// Sort tables for deterministic output
	sortedTables := sortTablesByQualifiedName(tables)

	// First pass: generate enum classes
	forEachEnum(sortedTables, func(e enumInfo) {
		sb.WriteString(fmt.Sprintf("class %s(str, Enum):\n", e.EnumName))
		for _, v := range e.Values {
			memberName := sanitizeIdentifier(strings.ToUpper(v))
			escaped := strings.ReplaceAll(v, `"`, `\"`)
			sb.WriteString(fmt.Sprintf("    %s = \"%s\"\n", memberName, escaped))
		}
		sb.WriteString("\n")
	})

	// Second pass: generate dataclasses
	for _, table := range sortedTables {
		generatePythonDataclass(&sb, table, cfg)
	}

	// Generate WithRelations variants if enabled
	if cfg.Relations {
		sb.WriteString("# WithRelations variants (includes relationship fields)\n\n")
		for _, table := range sortedTables {
			if table.Namespace == "" {
				continue
			}
			generateWithRelationsPython(&sb, table, sortedTables, cfg)
		}
	}

	// Generate schema URI to type name mapping
	sb.WriteString("# Schema URI to type name mapping\n")
	sb.WriteString("TYPES: dict[str, str] = {\n")
	for _, table := range sortedTables {
		if table.Namespace == "" {
			continue // Skip join tables
		}
		className := strutil.ToPascalCase(table.FullName())
		sb.WriteString(fmt.Sprintf("    \"%s\": \"%s\",\n", table.QualifiedName(), className))
	}
	sb.WriteString("}\n")

	return []byte(sb.String()), nil
}

// generatePythonDataclass generates a Python dataclass for a single table.
func generatePythonDataclass(sb *strings.Builder, table *ast.TableDef, cfg *exportContext) {
	name := strutil.ToPascalCase(table.FullName())

	// Add docstring if present
	if table.Docs != "" {
		fmt.Fprintf(sb, "\"\"\"%s\"\"\"\n", escapePyDoc(table.Docs))
	}

	sb.WriteString("@dataclass\n")
	fmt.Fprintf(sb, "class %s:\n", name)

	// Python dataclass requires fields with defaults (Optional) to come after required fields
	// First pass: required fields
	for _, col := range table.Columns {
		if col.Nullable {
			continue
		}
		pyType := columnToPythonType(col, table)

		// Add comment if present
		if col.Docs != "" {
			fmt.Fprintf(sb, "    # %s\n", col.Docs)
		}

		fmt.Fprintf(sb, "    %s: %s\n", col.Name, pyType)
	}

	// Second pass: optional fields (with = None default)
	for _, col := range table.Columns {
		if !col.Nullable {
			continue
		}
		pyType := columnToPythonType(col, table)
		pyType = fmt.Sprintf("Optional[%s]", pyType)

		// Add comment if present
		if col.Docs != "" {
			fmt.Fprintf(sb, "    # %s\n", col.Docs)
		}

		fmt.Fprintf(sb, "    %s: %s = None\n", col.Name, pyType)
	}

	sb.WriteString("\n")
}

// columnToPythonType converts a column type to Python type.
func columnToPythonType(col *ast.ColumnDef, table *ast.TableDef) string {
	// Create a fresh converter to avoid mutating the shared global instance
	converter := NewPythonConverter()
	converter.TableName = table.FullName()
	return converter.ConvertType(col)
}
