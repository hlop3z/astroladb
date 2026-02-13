package astroladb

import (
	"bytes"
	"fmt"
	"slices"
	"strings"

	"github.com/hlop3z/astroladb/internal/ast"
	"github.com/hlop3z/astroladb/internal/strutil"
)

// sortTablesByQualifiedName returns a sorted copy of the table slice for deterministic output.
func sortTablesByQualifiedName(tables []*ast.TableDef) []*ast.TableDef {
	sorted := slices.Clone(tables)
	slices.SortFunc(sorted, func(a, b *ast.TableDef) int {
		return strings.Compare(a.QualifiedName(), b.QualifiedName())
	})
	return sorted
}

// enumInfo holds precomputed information about an enum column for export code generators.
type enumInfo struct {
	Table    *ast.TableDef
	Column   *ast.ColumnDef
	EnumName string   // PascalCase(fullName) + PascalCase(colName)
	Values   []string // extracted enum values
}

// forEachEnum iterates over all enum columns in the given tables and calls fn for each.
func forEachEnum(tables []*ast.TableDef, fn func(e enumInfo)) {
	for _, table := range tables {
		for _, col := range table.Columns {
			if col.Type == "enum" && len(col.TypeArgs) > 0 {
				values := col.EnumValues()
				if len(values) == 0 {
					continue
				}
				fn(enumInfo{
					Table:    table,
					Column:   col,
					EnumName: strutil.ToPascalCase(table.FullName()) + strutil.ToPascalCase(col.Name),
					Values:   values,
				})
			}
		}
	}
}

// sanitizeIdentifier converts a string to a valid identifier by replacing
// non-alphanumeric characters with underscores. Used by Python and GraphQL exporters.
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

// escapeJSDoc escapes content for JSDoc comments (TypeScript).
func escapeJSDoc(s string) string {
	s = strings.ReplaceAll(s, "*/", "* /")
	return s
}

// relationField represents a relationship field to add to a WithRelations variant.
type relationField struct {
	FieldName string // e.g., "posts"
	TypeName  string // e.g., "Post"
	IsMany    bool   // true for has-many, false for belongs-to
}

// buildRelationFields computes relationship fields for a table based on FK references
// and many-to-many metadata.
func buildRelationFields(table *ast.TableDef, allTables []*ast.TableDef, cfg *exportContext) []relationField {
	var fields []relationField

	// Belongs-to: this table has FK columns referencing other tables
	for _, col := range table.Columns {
		if col.Reference == nil {
			continue
		}
		// Find the referenced table name
		refTable := col.Reference.Table
		// Strip suffix like "_id" from column name for field name
		fieldName := strings.TrimSuffix(col.Name, "_id")
		// Find the referenced table type name
		typeName := ""
		for _, t := range allTables {
			if t.MatchesReference(refTable) {
				typeName = strutil.ToPascalCase(t.FullName())
				break
			}
		}
		if typeName != "" {
			fields = append(fields, relationField{
				FieldName: fieldName,
				TypeName:  typeName,
				IsMany:    false,
			})
		}
	}

	// Has-many: other tables have FK columns referencing this table
	for _, other := range allTables {
		if other == table || other.Namespace == "" {
			continue // skip self and join tables
		}
		for _, col := range other.Columns {
			if col.Reference == nil {
				continue
			}
			if table.MatchesReference(col.Reference.Table) {
				// Use FK column name without "_id" suffix, ensure snake_case (no pluralization)
				fieldName := strutil.ToSnakeCase(strings.TrimSuffix(col.Name, "_id"))
				fields = append(fields, relationField{
					FieldName: fieldName,
					TypeName:  strutil.ToPascalCase(other.FullName()),
					IsMany:    true,
				})
				break // one relation per referencing table
			}
		}
	}

	// Many-to-many from metadata
	if cfg.Metadata != nil {
		for _, m2m := range cfg.Metadata.ManyToMany {
			if m2m.Source == table.QualifiedName() {
				// Find target type
				for _, t := range allTables {
					if t.QualifiedName() == m2m.Target {
						// Use TargetFK without "_id" suffix, ensure snake_case (no pluralization)
						fieldName := strutil.ToSnakeCase(strings.TrimSuffix(m2m.TargetFK, "_id"))
						fields = append(fields, relationField{
							FieldName: fieldName,
							TypeName:  strutil.ToPascalCase(t.FullName()),
							IsMany:    true,
						})
						break
					}
				}
			} else if m2m.Target == table.QualifiedName() {
				for _, t := range allTables {
					if t.QualifiedName() == m2m.Source {
						// Use SourceFK without "_id" suffix, ensure snake_case (no pluralization)
						fieldName := strutil.ToSnakeCase(strings.TrimSuffix(m2m.SourceFK, "_id"))
						fields = append(fields, relationField{
							FieldName: fieldName,
							TypeName:  strutil.ToPascalCase(t.FullName()),
							IsMany:    true,
						})
						break
					}
				}
			}
		}
	}

	return fields
}

// generateWithRelationsTypeScript generates WithRelations variant for TypeScript.
func generateWithRelationsTypeScript(sb *strings.Builder, table *ast.TableDef, allTables []*ast.TableDef, cfg *exportContext) {
	fields := buildRelationFields(table, allTables, cfg)
	if len(fields) == 0 {
		return
	}

	baseName := strutil.ToPascalCase(table.FullName())
	fmt.Fprintf(sb, "export interface %sWithRelations extends %s {\n", baseName, baseName)
	for _, f := range fields {
		if f.IsMany {
			fmt.Fprintf(sb, "  %s?: %s[];\n", f.FieldName, f.TypeName)
		} else {
			fmt.Fprintf(sb, "  %s?: %s;\n", f.FieldName, f.TypeName)
		}
	}
	sb.WriteString("}\n\n")
}

// generateWithRelationsGo generates WithRelations variant for Go.
func generateWithRelationsGo(buf *bytes.Buffer, table *ast.TableDef, allTables []*ast.TableDef, cfg *exportContext) {
	fields := buildRelationFields(table, allTables, cfg)
	if len(fields) == 0 {
		return
	}

	baseName := strutil.ToPascalCase(table.FullName())
	fmt.Fprintf(buf, "type %sWithRelations struct {\n", baseName)
	fmt.Fprintf(buf, "\t%s\n", baseName)
	for _, f := range fields {
		goField := strutil.ToPascalCase(f.FieldName)
		if f.IsMany {
			fmt.Fprintf(buf, "\t%s *[]%s `json:\"%s,omitempty\"`\n", goField, f.TypeName, f.FieldName)
		} else {
			fmt.Fprintf(buf, "\t%s *%s `json:\"%s,omitempty\"`\n", goField, f.TypeName, f.FieldName)
		}
	}
	buf.WriteString("}\n\n")
}

// generateWithRelationsPython generates WithRelations variant for Python.
func generateWithRelationsPython(sb *strings.Builder, table *ast.TableDef, allTables []*ast.TableDef, cfg *exportContext) {
	fields := buildRelationFields(table, allTables, cfg)
	if len(fields) == 0 {
		return
	}

	baseName := strutil.ToPascalCase(table.FullName())
	sb.WriteString("@dataclass\n")
	fmt.Fprintf(sb, "class %sWithRelations(%s):\n", baseName, baseName)
	for _, f := range fields {
		if f.IsMany {
			fmt.Fprintf(sb, "    %s: Optional[list[%s]] = None\n", f.FieldName, f.TypeName)
		} else {
			fmt.Fprintf(sb, "    %s: Optional[%s] = None\n", f.FieldName, f.TypeName)
		}
	}
	sb.WriteString("\n")
}

// generateWithRelationsGraphQL generates WithRelations variant for GraphQL.
func generateWithRelationsGraphQL(sb *strings.Builder, table *ast.TableDef, allTables []*ast.TableDef, cfg *exportContext) {
	fields := buildRelationFields(table, allTables, cfg)
	if len(fields) == 0 {
		return
	}

	baseName := strutil.ToPascalCase(table.FullName())
	fmt.Fprintf(sb, "type %sWithRelations {\n", baseName)

	// Include all base fields
	for _, col := range table.Columns {
		gqlType := columnToGraphQLType(col, table, cfg)
		fmt.Fprintf(sb, "  %s: %s\n", strutil.ToCamelCase(col.Name), gqlType)
	}

	// Add relationship fields
	for _, f := range fields {
		if f.IsMany {
			fmt.Fprintf(sb, "  %s: [%s!]\n", f.FieldName, f.TypeName)
		} else {
			fmt.Fprintf(sb, "  %s: %s\n", f.FieldName, f.TypeName)
		}
	}
	sb.WriteString("}\n\n")
}

// generateWithRelationsRust generates WithRelations variant for Rust.
func generateWithRelationsRust(sb *strings.Builder, table *ast.TableDef, allTables []*ast.TableDef, cfg *exportContext) {
	fields := buildRelationFields(table, allTables, cfg)
	if len(fields) == 0 {
		return
	}

	baseName := strutil.ToPascalCase(table.FullName())
	if cfg.UseMik {
		sb.WriteString("#[derive(Type)]\n")
	} else {
		sb.WriteString("#[derive(Debug, Clone, Serialize, Deserialize)]\n")
		sb.WriteString("#[serde(rename_all = \"snake_case\")]\n")
	}
	fmt.Fprintf(sb, "pub struct %sWithRelations {\n", baseName)
	sb.WriteString("    #[serde(flatten)]\n")
	fmt.Fprintf(sb, "    pub base: %s,\n", baseName)
	for _, f := range fields {
		if f.IsMany {
			fmt.Fprintf(sb, "    pub %s: Option<Vec<%s>>,\n", strutil.ToSnakeCase(f.FieldName), f.TypeName)
		} else {
			fmt.Fprintf(sb, "    pub %s: Option<%s>,\n", strutil.ToSnakeCase(f.FieldName), f.TypeName)
		}
	}
	sb.WriteString("}\n\n")
}

// generateGraphQLType generates a GraphQL type for a single table.
func generateGraphQLType(sb *strings.Builder, table *ast.TableDef, cfg *exportContext) {
	name := strutil.ToPascalCase(table.FullName())

	// Add doc comment if present
	if table.Docs != "" {
		fmt.Fprintf(sb, "\"\"\"%s\"\"\"\n", table.Docs)
	}

	fmt.Fprintf(sb, "type %s {\n", name)

	for _, col := range table.Columns {
		gqlType := columnToGraphQLType(col, table, cfg)

		// Add doc comment if present
		if col.Docs != "" {
			fmt.Fprintf(sb, "  \"\"\"%s\"\"\"\n", col.Docs)
		}

		fmt.Fprintf(sb, "  %s: %s\n", strutil.ToCamelCase(col.Name), gqlType)
	}

	sb.WriteString("}\n\n")
}

// columnToGraphQLType converts a column type to GraphQL type.
func columnToGraphQLType(col *ast.ColumnDef, table *ast.TableDef, cfg *exportContext) string {
	// Create a fresh converter to avoid mutating the shared global instance
	converter := NewGraphQLConverter()
	converter.TableName = table.FullName()
	baseType := converter.ConvertType(col)

	if col.Nullable {
		return baseType
	}
	return baseType + "!"
}
