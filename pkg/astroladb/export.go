package astroladb

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/hlop3z/astroladb/internal/ast"
	"github.com/hlop3z/astroladb/internal/metadata"
	"github.com/hlop3z/astroladb/internal/types"
)

// Export format constants.
const (
	FormatOpenAPI    = "openapi"
	FormatTypeScript = "typescript"
	FormatGo         = "go"
	FormatPython     = "python"
	FormatRust       = "rust"
)

// exportOpenAPI generates an OpenAPI 3.0 specification from the schema.
// Uses x-db extension for complete database metadata per PLAN.md.
func exportOpenAPI(tables []*ast.TableDef, cfg *ExportConfig) ([]byte, error) {
	spec := map[string]any{
		"openapi": "3.0.3",
		"info": map[string]any{
			"title":       "Database Schema",
			"description": "Auto-generated OpenAPI specification from database schema",
			"version":     "1.0.0",
			"license": map[string]any{
				"name": "MIT",
				"url":  "https://opensource.org/licenses/MIT",
			},
			"x-db": map[string]any{
				"generator":   "alab",
				"generatedAt": time.Now().UTC().Format(time.RFC3339),
				"version":     "1.0.0",
			},
		},
		"servers": []map[string]any{
			{"url": "/api/v1", "description": "API server"},
		},
		"paths": generateOpenAPIPaths(tables, cfg),
		"components": map[string]any{
			"schemas": generateOpenAPISchemas(tables, cfg),
			"securitySchemes": map[string]any{
				"bearerAuth": map[string]any{
					"type":         "http",
					"scheme":       "bearer",
					"bearerFormat": "JWT",
				},
			},
		},
		"security": []map[string]any{
			{"bearerAuth": []string{}},
		},
	}

	if cfg.Pretty {
		return json.MarshalIndent(spec, "", "  ")
	}
	return json.Marshal(spec)
}

// generateOpenAPIPaths generates a single /schemas endpoint with all models.
func generateOpenAPIPaths(tables []*ast.TableDef, cfg *ExportConfig) map[string]any {
	// Group models and relationships by namespace
	models := make(map[string][]map[string]any)
	relationships := make(map[string][]map[string]any)

	// Build join table namespace lookup from metadata
	joinTableNS := make(map[string]string)
	if cfg.Metadata != nil {
		for _, m2m := range cfg.Metadata.ManyToMany {
			// Extract namespace from source (e.g., "auth.user" → "auth")
			if idx := strings.Index(m2m.Source, "."); idx > 0 {
				joinTableNS[m2m.JoinTable] = m2m.Source[:idx]
			}
		}
	}

	for _, table := range tables {
		// Join tables have empty namespace
		if table.Namespace == "" {
			ns := joinTableNS[table.SQLName()]
			if ns == "" {
				ns = "internal"
			}
			relationships[ns] = append(relationships[ns], buildJoinTableEntry(table))
		} else {
			models[table.Namespace] = append(models[table.Namespace], buildModelEntry(table))
		}
	}

	// Build the example response
	example := map[string]any{
		"models": models,
	}
	if len(relationships) > 0 {
		example["relationships"] = relationships
	}

	return map[string]any{
		"/schemas": map[string]any{
			"get": map[string]any{
				"summary":     "All database schemas",
				"operationId": "getSchemas",
				"tags":        []string{"schemas"},
				"responses": map[string]any{
					"200": map[string]any{
						"description": "All models and relationships with examples",
						"content": map[string]any{
							"application/json": map[string]any{
								"example": example,
							},
						},
					},
				},
			},
		},
	}
}

// buildModelEntry builds a model entry with essential metadata.
func buildModelEntry(table *ast.TableDef) map[string]any {
	entry := map[string]any{
		"name":    table.Name,
		"table":   table.SQLName(),
		"columns": buildColumnList(table),
		"example": generateTableExample(table, false),
	}

	// Primary key
	if pk := findPrimaryKey(table); len(pk) > 0 {
		entry["primaryKey"] = pk[0]
	}

	// Timestamps
	if table.HasColumn("created_at") && table.HasColumn("updated_at") {
		entry["timestamps"] = true
	}

	// Soft delete
	if table.HasColumn("deleted_at") {
		entry["softDelete"] = true
	}

	// Auditable
	if table.HasColumn("created_by") && table.HasColumn("updated_by") {
		entry["auditable"] = true
	}

	// Foreign keys
	if fks := buildForeignKeys(table); len(fks) > 0 {
		entry["foreignKeys"] = fks
	}

	return entry
}

// buildJoinTableEntry builds a join table entry with essential metadata.
func buildJoinTableEntry(table *ast.TableDef) map[string]any {
	entry := map[string]any{
		"table":   table.SQLName(),
		"columns": buildColumnList(table),
		"example": generateTableExample(table, false),
	}

	// Extract the two tables it links
	var links []string
	for _, col := range table.Columns {
		if col.Reference != nil {
			links = append(links, col.Reference.Table)
		}
	}
	if len(links) > 0 {
		entry["links"] = links
	}

	return entry
}

// buildColumnList builds a simple column list with name and type.
func buildColumnList(table *ast.TableDef) []map[string]any {
	var cols []map[string]any
	for _, col := range table.Columns {
		c := map[string]any{
			"name": col.Name,
			"type": col.Type,
		}
		if col.Nullable {
			c["nullable"] = true
		}
		if col.Unique {
			c["unique"] = true
		}
		if col.Reference != nil {
			c["ref"] = col.Reference.Table
		}
		if col.Default != nil {
			c["default"] = col.Default
		}
		// Enum values - TypeArgs[0] contains the entire []string slice
		if col.Type == "enum" && len(col.TypeArgs) > 0 {
			if values, ok := col.TypeArgs[0].([]string); ok {
				c["enum"] = values
			} else if values, ok := col.TypeArgs[0].([]any); ok {
				// Goja may convert []string to []any
				var strValues []string
				for _, v := range values {
					if s, ok := v.(string); ok {
						strValues = append(strValues, s)
					}
				}
				if len(strValues) > 0 {
					c["enum"] = strValues
				}
			}
		}
		cols = append(cols, c)
	}
	return cols
}

// buildForeignKeys extracts foreign key references from a table.
func buildForeignKeys(table *ast.TableDef) []map[string]any {
	var fks []map[string]any
	for _, col := range table.Columns {
		if col.Reference != nil {
			fks = append(fks, map[string]any{
				"column": col.Name,
				"ref":    col.Reference.Table,
			})
		}
	}
	return fks
}

// generateTableExample generates an example object for a table with realistic values.
// If forRequest is true, includes password fields and excludes readOnly fields.
// If forRequest is false (response), excludes password fields and includes all others.
func generateTableExample(table *ast.TableDef, forRequest bool) map[string]any {
	example := make(map[string]any)

	for _, col := range table.Columns {
		isPasswordField := col.Name == "password" || col.Name == "password_hash" ||
			strings.Contains(col.Name, "password")
		isReadOnly := col.PrimaryKey || col.Name == "created_at" || col.Name == "updated_at"

		if forRequest {
			// Request: exclude readOnly, include writeOnly (password)
			if isReadOnly {
				continue
			}
		} else {
			// Response: exclude writeOnly (password), include readOnly
			if isPasswordField {
				continue
			}
		}

		example[col.Name] = generateColumnExample(col)
	}

	return example
}

// generateColumnExample generates an example value for a column based on its type and name.
func generateColumnExample(col *ast.ColumnDef) any {
	// Handle nullable columns - could return nil for some
	// but for examples, we'll show actual values

	// Check for semantic types by column name first
	switch col.Name {
	case "id":
		return "550e8400-e29b-41d4-a716-446655440000"
	case "email":
		return "user@example.com"
	case "username":
		return "johndoe"
	case "password", "password_hash":
		return "SecureP@ssw0rd123"
	case "name", "full_name":
		return "John Doe"
	case "first_name":
		return "John"
	case "last_name":
		return "Doe"
	case "title":
		return "Sample Title"
	case "slug":
		return "sample-title"
	case "phone":
		return "+1234567890"
	case "url", "website":
		return "https://example.com"
	case "ip", "ip_address":
		return "192.168.1.1"
	case "user_agent":
		return "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36"
	case "country":
		return "US"
	case "currency":
		return "USD"
	case "locale":
		return "en-US"
	case "timezone":
		return "America/New_York"
	case "color":
		return "#3498db"
	case "created_at", "updated_at":
		return "2024-01-15T10:30:00Z"
	case "deleted_at":
		return nil
	case "created_by", "updated_by":
		return "550e8400-e29b-41d4-a716-446655440001"
	case "is_active", "is_verified", "is_admin", "active", "verified", "enabled":
		return true
	case "is_deleted", "disabled":
		return false
	case "price", "amount", "total", "cost":
		return "99.99"
	case "quantity", "count":
		return 1
	case "percentage", "rate":
		return "50.00"
	case "rating", "score":
		return "4.5"
	case "duration", "length":
		return 3600
	case "age", "years":
		return 25
	case "description", "body", "content":
		return "This is a sample description text."
	case "summary":
		return "Brief summary of the content."
	case "token", "api_key":
		return "abc123def456ghi789jkl012mno345pqr678stu901vwx234yz"
	case "code", "sku":
		return "ABC-123"
	}

	// Fall back to type-based examples
	switch col.Type {
	case "id", "uuid":
		return "550e8400-e29b-41d4-a716-446655440000"
	case "string", "text":
		if col.Format == "email" {
			return "user@example.com"
		}
		if col.Format == "uri" {
			return "https://example.com"
		}
		return "Sample text"
	case "integer":
		if col.Min != nil {
			return *col.Min
		}
		return 42
	case "float", "decimal":
		return "123.45"
	case "boolean":
		if col.Default != nil {
			if b, ok := col.Default.(bool); ok {
				return b
			}
		}
		return true
	case "date":
		return "2024-01-15"
	case "time":
		return "10:30:00"
	case "datetime", "date_time":
		return "2024-01-15T10:30:00Z"
	case "json":
		return map[string]any{"key": "value"}
	case "base64":
		return "SGVsbG8gV29ybGQh"
	case "enum":
		if len(col.TypeArgs) > 0 {
			if values, ok := col.TypeArgs[0].([]string); ok && len(values) > 0 {
				return values[0] // Return first enum value as example
			} else if values, ok := col.TypeArgs[0].([]any); ok && len(values) > 0 {
				// Goja may convert []string to []any
				if s, ok := values[0].(string); ok {
					return s
				}
			}
		}
		return "value"
	default:
		// For FKs and other references
		if strings.HasSuffix(col.Name, "_id") {
			return "550e8400-e29b-41d4-a716-446655440002"
		}
		return "example"
	}
}

// generateOpenAPISchemas generates OpenAPI schema components for all tables.
func generateOpenAPISchemas(tables []*ast.TableDef, cfg *ExportConfig) map[string]any {
	schemas := make(map[string]any)

	for _, table := range tables {
		name := pascalCase(table.SQLName())
		schemas[name] = tableToOpenAPISchema(table, tables, cfg)
	}

	return schemas
}

// tableToOpenAPISchema converts a table definition to an OpenAPI schema object.
func tableToOpenAPISchema(table *ast.TableDef, allTables []*ast.TableDef, cfg *ExportConfig) map[string]any {
	properties := make(map[string]any)
	required := []string{}

	for _, col := range table.Columns {
		prop := columnToOpenAPIProperty(col, table, cfg)
		properties[col.Name] = prop

		// NOT NULL columns are required
		if !col.Nullable {
			required = append(required, col.Name)
		}
	}

	schema := map[string]any{
		"type":       "object",
		"properties": properties,
	}

	if len(required) > 0 {
		schema["required"] = required
	}

	if cfg.IncludeDescriptions && table.Docs != "" {
		schema["description"] = table.Docs
	}

	if table.Deprecated != "" {
		schema["deprecated"] = true
	}

	// Build comprehensive x-db extension
	schema["x-db"] = buildSchemaXDB(table, allTables, cfg.Metadata)

	return schema
}

// buildSchemaXDB builds the schema-level x-db extension object.
func buildSchemaXDB(table *ast.TableDef, allTables []*ast.TableDef, meta *metadata.Metadata) map[string]any {
	xdb := map[string]any{
		"table":     table.SQLName(),
		"namespace": table.Namespace,
	}

	// Primary key
	pk := findPrimaryKey(table)
	if len(pk) > 0 {
		xdb["primaryKey"] = pk
	}

	// Check if this is a join table and add joinTable metadata
	if meta != nil {
		if joinTableInfo := findJoinTableInfo(table.SQLName(), meta); joinTableInfo != nil {
			xdb["joinTable"] = joinTableInfo
		}
	}

	// Check for timestamp columns
	hasCreatedAt := table.HasColumn("created_at")
	hasUpdatedAt := table.HasColumn("updated_at")
	if hasCreatedAt && hasUpdatedAt {
		xdb["timestamps"] = true
	}

	// Check for audit columns
	if table.Auditable || (table.HasColumn("created_by") && table.HasColumn("updated_by")) {
		xdb["auditable"] = true
	}

	// Check for soft delete
	if table.HasColumn("deleted_at") {
		xdb["softDelete"] = true
	}

	// Sort by
	if len(table.SortBy) > 0 {
		xdb["sortBy"] = table.SortBy
	}

	// Searchable columns
	if len(table.Searchable) > 0 {
		xdb["searchable"] = table.Searchable
	}

	// Filterable columns
	if len(table.Filterable) > 0 {
		xdb["filterable"] = table.Filterable
	}

	// Indexes
	indexes := buildIndexes(table)
	if len(indexes) > 0 {
		xdb["indexes"] = indexes
	}

	// Relationships (hasMany, hasOne from other tables, and manyToMany)
	relationships := buildRelationships(table, allTables, meta)
	if len(relationships) > 0 {
		xdb["relationships"] = relationships
	}

	return xdb
}

// findJoinTableInfo returns join table metadata if the table is a join table.
func findJoinTableInfo(sqlName string, meta *metadata.Metadata) map[string]any {
	if meta == nil || meta.JoinTables == nil {
		return nil
	}

	if _, ok := meta.JoinTables[sqlName]; !ok {
		return nil
	}

	// Find the m2m relationship for this join table
	var m2m *metadata.ManyToManyMeta
	for _, rel := range meta.ManyToMany {
		if rel.JoinTable == sqlName {
			m2m = rel
			break
		}
	}

	if m2m == nil {
		return nil
	}

	return map[string]any{
		"left": map[string]any{
			"schema": m2m.Source,
			"column": m2m.SourceFK,
		},
		"right": map[string]any{
			"schema": m2m.Target,
			"column": m2m.TargetFK,
		},
	}
}

// findPrimaryKey returns the primary key column names.
func findPrimaryKey(table *ast.TableDef) []string {
	var pk []string
	for _, col := range table.Columns {
		if col.PrimaryKey {
			pk = append(pk, col.Name)
		}
	}
	return pk
}

// buildIndexes builds the indexes array for x-db.
func buildIndexes(table *ast.TableDef) []map[string]any {
	var indexes []map[string]any

	// Add primary key index
	pk := findPrimaryKey(table)
	if len(pk) > 0 {
		indexes = append(indexes, map[string]any{
			"name":    table.SQLName() + "_pkey",
			"columns": pk,
			"unique":  true,
			"primary": true,
		})
	}

	// Add explicit indexes from table definition
	for _, idx := range table.Indexes {
		idxDef := map[string]any{
			"columns": idx.Columns,
		}
		if idx.Name != "" {
			idxDef["name"] = idx.Name
		} else {
			idxDef["name"] = table.SQLName() + "_" + strings.Join(idx.Columns, "_") + "_idx"
		}
		if idx.Unique {
			idxDef["unique"] = true
		}
		indexes = append(indexes, idxDef)
	}

	// Add indexes for unique columns
	for _, col := range table.Columns {
		if col.Unique && !col.PrimaryKey {
			indexes = append(indexes, map[string]any{
				"name":    table.SQLName() + "_" + col.Name + "_key",
				"columns": []string{col.Name},
				"unique":  true,
			})
		}
	}

	// Add indexes for foreign keys
	for _, col := range table.Columns {
		if col.Reference != nil {
			// Check if already covered by unique index
			alreadyIndexed := false
			for _, idx := range indexes {
				cols := idx["columns"].([]string)
				if len(cols) == 1 && cols[0] == col.Name {
					alreadyIndexed = true
					break
				}
			}
			if !alreadyIndexed {
				indexes = append(indexes, map[string]any{
					"name":    table.SQLName() + "_" + col.Name + "_idx",
					"columns": []string{col.Name},
				})
			}
		}
	}

	return indexes
}

// buildRelationships builds the relationships map for x-db.
// This includes hasMany/hasOne relationships from other tables pointing to this one,
// and manyToMany relationships from metadata.
func buildRelationships(table *ast.TableDef, allTables []*ast.TableDef, meta *metadata.Metadata) map[string]any {
	relationships := make(map[string]any)

	// Find all foreign keys in other tables that point to this table
	for _, otherTable := range allTables {
		if otherTable.QualifiedName() == table.QualifiedName() {
			continue
		}

		for _, col := range otherTable.Columns {
			if col.Reference == nil {
				continue
			}

			// Check if this FK points to our table
			refTable := col.Reference.Table
			if !matchesTable(refTable, table) {
				continue
			}

			// Determine relationship name (strip _id suffix if present)
			relationName := strings.TrimSuffix(col.Name, "_id")
			if relationName == "" {
				relationName = otherTable.Name
			}

			// Create hasMany relationship (or hasOne if unique)
			relType := "hasMany"
			if col.Unique {
				relType = "hasOne"
			}

			// Use plural form for hasMany
			relName := relationName
			if relType == "hasMany" {
				relName = pluralize(otherTable.Name)
			}

			relationships[relName] = map[string]any{
				"type":        relType,
				"target":      otherTable.QualifiedName(),
				"targetTable": otherTable.SQLName(),
				"foreignKey":  col.Name,
				"localKey":    "id",
				"backref":     relationName,
			}
		}
	}

	// Add manyToMany relationships from metadata
	if meta != nil {
		for _, m2m := range meta.ManyToMany {
			// Check if this table is the source of the m2m relationship
			if m2m.Source == table.QualifiedName() {
				// This table has a manyToMany to target
				targetParts := strings.Split(m2m.Target, ".")
				targetName := targetParts[len(targetParts)-1]

				relationships[pluralize(targetName)] = map[string]any{
					"type":        "manyToMany",
					"target":      m2m.Target,
					"targetTable": strings.ReplaceAll(m2m.Target, ".", "_"),
					"localKey":    "id",
					"backref":     pluralize(table.Name),
					"through": map[string]any{
						"table":      m2m.JoinTable,
						"localKey":   m2m.SourceFK,
						"foreignKey": m2m.TargetFK,
					},
				}
			}

			// Check if this table is the target of the m2m relationship
			if m2m.Target == table.QualifiedName() {
				// This table has an inverse manyToMany
				sourceParts := strings.Split(m2m.Source, ".")
				sourceName := sourceParts[len(sourceParts)-1]

				relationships[pluralize(sourceName)] = map[string]any{
					"type":        "manyToMany",
					"target":      m2m.Source,
					"targetTable": strings.ReplaceAll(m2m.Source, ".", "_"),
					"localKey":    "id",
					"backref":     pluralize(table.Name),
					"through": map[string]any{
						"table":      m2m.JoinTable,
						"localKey":   m2m.TargetFK,
						"foreignKey": m2m.SourceFK,
					},
				}
			}
		}
	}

	return relationships
}

// matchesTable checks if a reference matches a table.
func matchesTable(ref string, table *ast.TableDef) bool {
	// Handle "namespace.table" format
	if ref == table.QualifiedName() {
		return true
	}
	// Handle ".table" (same namespace) format
	if strings.HasPrefix(ref, ".") && ref[1:] == table.Name {
		return true
	}
	// Handle plain "table" format (assumes same namespace)
	if ref == table.Name {
		return true
	}
	return false
}

// pluralize provides simple English pluralization.
func pluralize(s string) string {
	if strings.HasSuffix(s, "s") || strings.HasSuffix(s, "x") ||
		strings.HasSuffix(s, "ch") || strings.HasSuffix(s, "sh") {
		return s + "es"
	}
	if strings.HasSuffix(s, "y") && len(s) > 1 {
		vowels := "aeiou"
		if !strings.ContainsRune(vowels, rune(s[len(s)-2])) {
			return s[:len(s)-1] + "ies"
		}
	}
	return s + "s"
}

// columnToOpenAPIProperty converts a column definition to an OpenAPI property.
func columnToOpenAPIProperty(col *ast.ColumnDef, table *ast.TableDef, cfg *ExportConfig) map[string]any {
	prop := make(map[string]any)

	// Get base type from registry
	typeDef := types.Get(col.Type)
	if typeDef != nil {
		prop["type"] = typeDef.OpenAPI.Type
		if typeDef.OpenAPI.Format != "" {
			prop["format"] = typeDef.OpenAPI.Format
		}
	} else {
		prop["type"] = "string"
	}

	// Handle type-specific properties
	switch col.Type {
	case "string":
		// Extract maxLength from TypeArgs
		if len(col.TypeArgs) > 0 {
			if max, ok := col.TypeArgs[0].(float64); ok {
				prop["maxLength"] = int(max)
			} else if max, ok := col.TypeArgs[0].(int); ok {
				prop["maxLength"] = max
			}
		}

	case "decimal":
		// Decimal pattern for validation
		prop["pattern"] = `^-?\d+(\.\d+)?$`

	case "json":
		// Allow any properties
		prop["additionalProperties"] = true

	case "enum":
		// Extract enum values from TypeArgs - TypeArgs[0] contains the entire []string slice
		if len(col.TypeArgs) > 0 {
			if enumValues, ok := col.TypeArgs[0].([]string); ok {
				prop["enum"] = enumValues
			} else if values, ok := col.TypeArgs[0].([]any); ok {
				// Goja may convert []string to []any
				var enumValues []string
				for _, v := range values {
					if s, ok := v.(string); ok {
						enumValues = append(enumValues, s)
					}
				}
				if len(enumValues) > 0 {
					prop["enum"] = enumValues
				}
			}
		}

	case "computed":
		// Computed columns are read-only
		prop["readOnly"] = true
	}

	// Apply constraints
	if col.Min != nil {
		if prop["type"] == "string" {
			prop["minLength"] = *col.Min
		} else {
			prop["minimum"] = *col.Min
		}
	}

	if col.Max != nil {
		if prop["type"] == "string" {
			prop["maxLength"] = *col.Max
		} else {
			prop["maximum"] = *col.Max
		}
	}

	if col.Pattern != "" {
		prop["pattern"] = col.Pattern
	}

	if col.Format != "" && prop["format"] == nil {
		prop["format"] = col.Format
	}

	// Description
	if cfg.IncludeDescriptions && col.Docs != "" {
		prop["description"] = col.Docs
	}

	if col.Deprecated != "" {
		prop["deprecated"] = true
	}

	// Nullable
	if col.Nullable {
		prop["nullable"] = true
	}

	// Mark generated/readonly columns
	if col.PrimaryKey || col.Name == "created_at" || col.Name == "updated_at" {
		prop["readOnly"] = true
	}

	// Mark password as writeOnly
	if col.Type == "string" && (col.Name == "password" || col.Name == "password_hash" ||
		strings.Contains(col.Name, "password")) {
		prop["writeOnly"] = true
	}

	// Build property-level x-db extension
	xdb := buildPropertyXDB(col, table, cfg.Metadata)
	if len(xdb) > 0 {
		prop["x-db"] = xdb
	}

	return prop
}

// buildPropertyXDB builds the property-level x-db extension object.
func buildPropertyXDB(col *ast.ColumnDef, table *ast.TableDef, meta *metadata.Metadata) map[string]any {
	xdb := make(map[string]any)

	// Semantic type (original column type)
	semantic := getSemanticType(col)
	if semantic != "" {
		xdb["semantic"] = semantic
	}

	// Generated columns (like id)
	if col.PrimaryKey && col.Type == "id" {
		xdb["generated"] = true
	}

	// Auto-managed (timestamps - adapters must set these)
	if col.Name == "created_at" || col.Name == "updated_at" || col.Name == "deleted_at" {
		xdb["autoManaged"] = false
	}
	if col.Name == "created_by" || col.Name == "updated_by" {
		xdb["autoManaged"] = false
	}

	// Default value
	if col.Default != nil {
		xdb["default"] = col.Default
	}

	// Foreign key reference
	if col.Reference != nil {
		xdb["ref"] = col.Reference.Table
		xdb["fk"] = col.Reference.Table + "." + col.Reference.Column

		if col.Reference.OnDelete != "" {
			xdb["onDelete"] = camelCase(col.Reference.OnDelete)
		}
		if col.Reference.OnUpdate != "" {
			xdb["onUpdate"] = camelCase(col.Reference.OnUpdate)
		}

		// Relationship name (strip _id suffix)
		relationName := strings.TrimSuffix(col.Name, "_id")
		if relationName != "" {
			xdb["relation"] = relationName
		}

		// Inverse relationship name
		inverseOf := pluralize(table.Name)
		xdb["inverseOf"] = inverseOf
	}

	// SQL type per dialect
	sqlType := buildSQLType(col)
	if len(sqlType) > 0 {
		xdb["sqlType"] = sqlType
	}

	// Computed columns
	if col.Computed != nil {
		xdb["virtual"] = true
		xdb["computed"] = col.Computed
	}

	// Polymorphic columns (from belongs_to_any)
	if meta != nil {
		if polyInfo := findPolymorphicInfo(col.Name, table.QualifiedName(), meta); polyInfo != nil {
			xdb["polymorphic"] = polyInfo
		}
	}

	return xdb
}

// findPolymorphicInfo returns polymorphic metadata for a column if it's part of a polymorphic relationship.
func findPolymorphicInfo(colName, tableName string, meta *metadata.Metadata) map[string]any {
	if meta == nil {
		return nil
	}

	for _, poly := range meta.Polymorphic {
		if poly.Table != tableName {
			continue
		}

		// Check if this column is the type column
		if colName == poly.TypeColumn {
			return map[string]any{
				"field":   poly.Alias,
				"role":    "type",
				"targets": poly.Targets,
			}
		}

		// Check if this column is the id column
		if colName == poly.IDColumn {
			return map[string]any{
				"field":   poly.Alias,
				"role":    "id",
				"targets": poly.Targets,
			}
		}
	}

	return nil
}

// getSemanticType returns the semantic type name for a column.
func getSemanticType(col *ast.ColumnDef) string {
	// Check for special column names that indicate semantic types
	switch col.Name {
	case "id":
		return "id"
	case "email":
		return "email"
	case "username":
		return "username"
	case "password", "password_hash":
		return "password_hash"
	case "created_at":
		return "created_at"
	case "updated_at":
		return "updated_at"
	case "deleted_at":
		return "deleted_at"
	case "created_by":
		return "created_by"
	case "updated_by":
		return "updated_by"
	}

	// Check by type and format
	if col.Format == "email" {
		return "email"
	}
	if col.Format == "uri" {
		return "url"
	}

	// Check by type
	switch col.Type {
	case "id":
		return "id"
	case "boolean":
		if col.Default != nil {
			return "flag"
		}
	case "decimal":
		// Check for money pattern (19,4)
		if len(col.TypeArgs) >= 2 {
			if p, ok := col.TypeArgs[0].(float64); ok && p == 19 {
				if s, ok := col.TypeArgs[1].(float64); ok && s == 4 {
					return "money"
				}
			}
		}
	}

	return ""
}

// buildSQLType builds the sqlType object with database-specific SQL.
// Uses the centralized type registry for consistency with dialect implementations.
func buildSQLType(col *ast.ColumnDef) map[string]string {
	sqlType := make(map[string]string)

	// Get type definition from registry
	typeDef := types.Get(col.Type)
	if typeDef == nil {
		return sqlType
	}

	// Handle types with arguments (string, decimal)
	switch col.Type {
	case "id":
		// id type includes DEFAULT gen_random_uuid() in the type itself
		sqlType["postgres"] = typeDef.SQLTypes.Postgres
		sqlType["sqlite"] = typeDef.SQLTypes.SQLite

	case "string":
		length := 255
		if len(col.TypeArgs) > 0 {
			if l, ok := col.TypeArgs[0].(float64); ok {
				length = int(l)
			} else if l, ok := col.TypeArgs[0].(int); ok {
				length = l
			}
		}
		sqlType["postgres"] = fmt.Sprintf(typeDef.SQLTypes.Postgres, length)
		sqlType["sqlite"] = typeDef.SQLTypes.SQLite // SQLite ignores length

	case "decimal":
		p, s := 10, 2
		if len(col.TypeArgs) >= 2 {
			if prec, ok := col.TypeArgs[0].(float64); ok {
				p = int(prec)
			}
			if scale, ok := col.TypeArgs[1].(float64); ok {
				s = int(scale)
			}
		}
		sqlType["postgres"] = fmt.Sprintf(typeDef.SQLTypes.Postgres, p, s)
		sqlType["sqlite"] = typeDef.SQLTypes.SQLite // SQLite uses TEXT

	case "enum":
		// ENUMs are handled differently per database (special case)
		// TypeArgs[0] contains the entire []string slice
		if len(col.TypeArgs) > 0 {
			var enumValues []string
			if values, ok := col.TypeArgs[0].([]string); ok {
				enumValues = values
			} else if values, ok := col.TypeArgs[0].([]any); ok {
				// Goja may convert []string to []any
				for _, v := range values {
					if s, ok := v.(string); ok {
						enumValues = append(enumValues, s)
					}
				}
			}
			if len(enumValues) > 0 {
				var quotedValues []string
				for _, v := range enumValues {
					quotedValues = append(quotedValues, fmt.Sprintf("'%s'", v))
				}
				valuesStr := strings.Join(quotedValues, ", ")
				sqlType["postgres"] = fmt.Sprintf("VARCHAR(50) CHECK (%s IN (%s))", col.Name, valuesStr)
				sqlType["sqlite"] = fmt.Sprintf("TEXT CHECK (%s IN (%s))", col.Name, valuesStr)
			}
		}

	default:
		// Use registry values directly for types without arguments
		if typeDef.SQLTypes.Postgres != "" {
			sqlType["postgres"] = typeDef.SQLTypes.Postgres
		}
		if typeDef.SQLTypes.SQLite != "" {
			sqlType["sqlite"] = typeDef.SQLTypes.SQLite
		}
	}

	return sqlType
}

// camelCase converts snake_case to camelCase.
func camelCase(s string) string {
	parts := strings.Split(s, "_")
	for i := 1; i < len(parts); i++ {
		if len(parts[i]) > 0 {
			parts[i] = strings.ToUpper(parts[i][:1]) + strings.ToLower(parts[i][1:])
		}
	}
	if len(parts[0]) > 0 {
		parts[0] = strings.ToLower(parts[0])
	}
	return strings.Join(parts, "")
}

// exportTypeScript generates TypeScript type definitions from the schema.
func exportTypeScript(tables []*ast.TableDef, cfg *ExportConfig) ([]byte, error) {
	var sb strings.Builder

	sb.WriteString("// Auto-generated TypeScript types from database schema\n")
	sb.WriteString("// Do not edit manually\n\n")

	// Sort tables for deterministic output
	sortedTables := make([]*ast.TableDef, len(tables))
	copy(sortedTables, tables)
	sort.Slice(sortedTables, func(i, j int) bool {
		return sortedTables[i].QualifiedName() < sortedTables[j].QualifiedName()
	})

	for i, table := range sortedTables {
		if i > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString(tableToTypeScript(table, cfg))
	}

	// Generate schema URI to type name mapping
	sb.WriteString("\n// Schema URI to type name mapping\n")
	sb.WriteString("export const TYPES: Record<string, string> = {\n")
	for _, table := range sortedTables {
		if table.Namespace == "" {
			continue // Skip join tables
		}
		typeName := pascalCase(table.SQLName())
		sb.WriteString(fmt.Sprintf("  \"%s\": \"%s\",\n", table.QualifiedName(), typeName))
	}
	sb.WriteString("};\n")

	return []byte(sb.String()), nil
}

// tableToTypeScript converts a table definition to TypeScript interface.
func tableToTypeScript(table *ast.TableDef, cfg *ExportConfig) string {
	var sb strings.Builder

	name := pascalCase(table.SQLName())

	// Add JSDoc comment
	if cfg.IncludeDescriptions && table.Docs != "" {
		sb.WriteString("/**\n")
		sb.WriteString(fmt.Sprintf(" * %s\n", table.Docs))
		if table.Deprecated != "" {
			sb.WriteString(fmt.Sprintf(" * @deprecated %s\n", table.Deprecated))
		}
		sb.WriteString(" */\n")
	}

	sb.WriteString(fmt.Sprintf("export interface %s {\n", name))

	for _, col := range table.Columns {
		// Add JSDoc for column
		if cfg.IncludeDescriptions && col.Docs != "" {
			sb.WriteString(fmt.Sprintf("  /** %s */\n", col.Docs))
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
	// Handle enum specially - create union type from values
	// TypeArgs[0] contains the entire []string slice
	if col.Type == "enum" && len(col.TypeArgs) > 0 {
		var enumValues []string
		if values, ok := col.TypeArgs[0].([]string); ok {
			enumValues = values
		} else if values, ok := col.TypeArgs[0].([]any); ok {
			// Goja may convert []string to []any
			for _, v := range values {
				if s, ok := v.(string); ok {
					enumValues = append(enumValues, s)
				}
			}
		}
		if len(enumValues) > 0 {
			var quotedValues []string
			for _, v := range enumValues {
				quotedValues = append(quotedValues, fmt.Sprintf("'%s'", v))
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

// pascalCase converts a snake_case string to PascalCase.
func pascalCase(s string) string {
	parts := strings.Split(s, "_")
	for i, part := range parts {
		if len(part) > 0 {
			parts[i] = strings.ToUpper(part[:1]) + strings.ToLower(part[1:])
		}
	}
	return strings.Join(parts, "")
}

// exportGo generates Go struct definitions from the schema.
func exportGo(tables []*ast.TableDef, cfg *ExportConfig) ([]byte, error) {
	var buf bytes.Buffer

	// Package declaration
	pkgName := "types"
	fmt.Fprintf(&buf, "// Package %s provides generated Go types from the database schema.\n", pkgName)
	fmt.Fprintf(&buf, "// Generated by: alab export --format go\n")
	fmt.Fprintf(&buf, "package %s\n\n", pkgName)

	// Sort tables for deterministic output
	sortedTables := make([]*ast.TableDef, len(tables))
	copy(sortedTables, tables)
	sort.Slice(sortedTables, func(i, j int) bool {
		return sortedTables[i].QualifiedName() < sortedTables[j].QualifiedName()
	})

	// Generate structs for each table
	for _, table := range sortedTables {
		generateGoStruct(&buf, table, cfg)
	}

	// Generate schema URI to type name mapping
	buf.WriteString("// TYPES maps schema URIs to type names.\n")
	buf.WriteString("var TYPES = map[string]string{\n")
	for _, table := range sortedTables {
		if table.Namespace == "" {
			continue // Skip join tables
		}
		typeName := pascalCase(table.SQLName())
		fmt.Fprintf(&buf, "\t\"%s\": \"%s\",\n", table.QualifiedName(), typeName)
	}
	buf.WriteString("}\n")

	return buf.Bytes(), nil
}

// generateGoStruct generates a Go struct for a single table.
func generateGoStruct(buf *bytes.Buffer, table *ast.TableDef, cfg *ExportConfig) {
	name := pascalCase(table.SQLName())

	// Add doc comment if present
	if cfg.IncludeDescriptions && table.Docs != "" {
		fmt.Fprintf(buf, "// %s %s\n", name, table.Docs)
	}
	if table.Deprecated != "" {
		fmt.Fprintf(buf, "// Deprecated: %s\n", table.Deprecated)
	}

	fmt.Fprintf(buf, "type %s struct {\n", name)

	for _, col := range table.Columns {
		goType := mapToGoType(col)
		jsonTag := fmt.Sprintf("`json:\"%s", col.Name)
		if col.Nullable {
			jsonTag += ",omitempty"
		}
		jsonTag += "\"`"

		// Add doc comment if present
		if cfg.IncludeDescriptions && col.Docs != "" {
			fmt.Fprintf(buf, "\t// %s\n", col.Docs)
		}
		fmt.Fprintf(buf, "\t%s %s %s\n", pascalCase(col.Name), goType, jsonTag)
	}

	buf.WriteString("}\n\n")
}

// mapToGoType converts a column type to its Go equivalent.
func mapToGoType(col *ast.ColumnDef) string {
	// Get base type from TypeRegistry
	base := "string" // default
	if typeDef := types.Get(col.Type); typeDef != nil {
		base = typeDef.GoType
	}

	// Handle nullable types
	if col.Nullable {
		if base == "time.Time" {
			return "*time.Time"
		}
		if base == "[]byte" || strings.HasPrefix(base, "map[") {
			return base // Already reference types
		}
		return "*" + base
	}
	return base
}

// exportPython generates Python dataclass definitions from the schema.
func exportPython(tables []*ast.TableDef, cfg *ExportConfig) ([]byte, error) {
	var sb strings.Builder

	sb.WriteString("# Auto-generated Python types from database schema\n")
	sb.WriteString("# Do not edit manually\n\n")
	sb.WriteString("from __future__ import annotations\n")
	sb.WriteString("from dataclasses import dataclass\n")
	sb.WriteString("from typing import Optional, Any\n")
	sb.WriteString("from datetime import datetime, date, time\n")
	sb.WriteString("from enum import Enum\n\n")

	// Sort tables for deterministic output
	sortedTables := make([]*ast.TableDef, len(tables))
	copy(sortedTables, tables)
	sort.Slice(sortedTables, func(i, j int) bool {
		return sortedTables[i].QualifiedName() < sortedTables[j].QualifiedName()
	})

	// First pass: generate enum classes
	for _, table := range sortedTables {
		for _, col := range table.Columns {
			if col.Type == "enum" && len(col.TypeArgs) > 0 {
				enumName := pascalCase(table.SQLName()) + pascalCase(col.Name)
				sb.WriteString(fmt.Sprintf("class %s(str, Enum):\n", enumName))
				enumValues := getEnumValues(col)
				for _, v := range enumValues {
					sb.WriteString(fmt.Sprintf("    %s = \"%s\"\n", strings.ToUpper(v), v))
				}
				sb.WriteString("\n")
			}
		}
	}

	// Second pass: generate dataclasses
	for _, table := range sortedTables {
		generatePythonDataclass(&sb, table, cfg)
	}

	// Generate schema URI to type name mapping
	sb.WriteString("# Schema URI to type name mapping\n")
	sb.WriteString("TYPES: dict[str, str] = {\n")
	for _, table := range sortedTables {
		if table.Namespace == "" {
			continue // Skip join tables
		}
		className := pascalCase(table.SQLName())
		sb.WriteString(fmt.Sprintf("    \"%s\": \"%s\",\n", table.QualifiedName(), className))
	}
	sb.WriteString("}\n")

	return []byte(sb.String()), nil
}

// generatePythonDataclass generates a Python dataclass for a single table.
func generatePythonDataclass(sb *strings.Builder, table *ast.TableDef, cfg *ExportConfig) {
	name := pascalCase(table.SQLName())

	// Add docstring if present
	if cfg.IncludeDescriptions && table.Docs != "" {
		sb.WriteString(fmt.Sprintf("\"\"\"%s\"\"\"\n", table.Docs))
	}

	sb.WriteString("@dataclass\n")
	sb.WriteString(fmt.Sprintf("class %s:\n", name))

	// Python dataclass requires fields with defaults (Optional) to come after required fields
	// First pass: required fields
	for _, col := range table.Columns {
		if col.Nullable {
			continue
		}
		pyType := columnToPythonType(col, table)

		// Add comment if present
		if cfg.IncludeDescriptions && col.Docs != "" {
			sb.WriteString(fmt.Sprintf("    # %s\n", col.Docs))
		}

		sb.WriteString(fmt.Sprintf("    %s: %s\n", col.Name, pyType))
	}

	// Second pass: optional fields (with = None default)
	for _, col := range table.Columns {
		if !col.Nullable {
			continue
		}
		pyType := columnToPythonType(col, table)
		pyType = fmt.Sprintf("Optional[%s]", pyType)

		// Add comment if present
		if cfg.IncludeDescriptions && col.Docs != "" {
			sb.WriteString(fmt.Sprintf("    # %s\n", col.Docs))
		}

		sb.WriteString(fmt.Sprintf("    %s: %s = None\n", col.Name, pyType))
	}

	sb.WriteString("\n")
}

// columnToPythonType converts a column type to Python type.
func columnToPythonType(col *ast.ColumnDef, table *ast.TableDef) string {
	// Handle enum specially
	if col.Type == "enum" {
		return pascalCase(table.SQLName()) + pascalCase(col.Name)
	}

	// Map types
	switch col.Type {
	case "id", "uuid":
		return "str"
	case "string", "text":
		return "str"
	case "integer":
		return "int"
	case "float":
		return "float"
	case "decimal":
		return "str" // Decimal as string for precision
	case "boolean":
		return "bool"
	case "date":
		return "date"
	case "time":
		return "time"
	case "date_time":
		return "datetime"
	case "json":
		return "Any"
	case "base64":
		return "bytes"
	default:
		return "str"
	}
}

// exportRust generates Rust struct definitions with serde from the schema.
func exportRust(tables []*ast.TableDef, cfg *ExportConfig) ([]byte, error) {
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
				enumName := pascalCase(table.SQLName()) + pascalCase(col.Name)
				if cfg.UseMik {
					sb.WriteString("#[derive(Type)]\n")
				} else {
					sb.WriteString("#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq)]\n")
					sb.WriteString("#[serde(rename_all = \"snake_case\")]\n")
				}
				sb.WriteString(fmt.Sprintf("pub enum %s {\n", enumName))
				enumValues := getEnumValues(col)
				for _, v := range enumValues {
					sb.WriteString(fmt.Sprintf("    %s,\n", pascalCase(v)))
				}
				sb.WriteString("}\n\n")
			}
		}
	}

	// Second pass: generate structs
	for _, table := range sortedTables {
		generateRustStruct(&sb, table, cfg)
	}

	// Generate schema URI to type name mapping
	sb.WriteString("/// Schema URI to type name mapping.\n")
	sb.WriteString("pub const TYPES: &[(&str, &str)] = &[\n")
	for _, table := range sortedTables {
		if table.Namespace == "" {
			continue // Skip join tables
		}
		typeName := pascalCase(table.SQLName())
		sb.WriteString(fmt.Sprintf("    (\"%s\", \"%s\"),\n", table.QualifiedName(), typeName))
	}
	sb.WriteString("];\n")

	return []byte(sb.String()), nil
}

// generateRustStruct generates a Rust struct for a single table.
func generateRustStruct(sb *strings.Builder, table *ast.TableDef, cfg *ExportConfig) {
	name := pascalCase(table.SQLName())

	// Add doc comment if present
	if cfg.IncludeDescriptions && table.Docs != "" {
		sb.WriteString(fmt.Sprintf("/// %s\n", table.Docs))
	}

	if cfg.UseMik {
		sb.WriteString("#[derive(Type)]\n")
	} else {
		sb.WriteString("#[derive(Debug, Clone, Serialize, Deserialize)]\n")
	}
	sb.WriteString(fmt.Sprintf("pub struct %s {\n", name))

	for _, col := range table.Columns {
		rustType := columnToRustType(col, table, cfg)

		// Add doc comment if present
		if cfg.IncludeDescriptions && col.Docs != "" {
			sb.WriteString(fmt.Sprintf("    /// %s\n", col.Docs))
		}

		// Handle Rust reserved keywords
		fieldName := col.Name
		if isRustKeyword(col.Name) {
			if !cfg.UseMik {
				sb.WriteString(fmt.Sprintf("    #[serde(rename = \"%s\")]\n", col.Name))
			}
			fieldName = col.Name + "_"
		}

		sb.WriteString(fmt.Sprintf("    pub %s: %s,\n", fieldName, rustType))
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
func columnToRustType(col *ast.ColumnDef, table *ast.TableDef, cfg *ExportConfig) string {
	var baseType string

	// Handle enum specially
	if col.Type == "enum" {
		baseType = pascalCase(table.SQLName()) + pascalCase(col.Name)
	} else {
		// Map types
		switch col.Type {
		case "id", "uuid":
			baseType = "String"
		case "string", "text":
			baseType = "String"
		case "integer":
			baseType = "i32"
		case "float":
			baseType = "f32"
		case "decimal":
			baseType = "String" // Decimal as string for precision
		case "boolean":
			baseType = "bool"
		case "date":
			if cfg.UseChrono {
				baseType = "NaiveDate"
			} else {
				baseType = "String"
			}
		case "time":
			if cfg.UseChrono {
				baseType = "NaiveTime"
			} else {
				baseType = "String"
			}
		case "date_time":
			if cfg.UseChrono {
				baseType = "DateTime<Utc>"
			} else {
				baseType = "String"
			}
		case "json":
			baseType = "serde_json::Value"
		case "base64":
			baseType = "Vec<u8>"
		default:
			baseType = "String"
		}
	}

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

// exportGraphQLExamples generates example data for GraphQL types as JSON.
func exportGraphQLExamples(tables []*ast.TableDef, cfg *ExportConfig) ([]byte, error) {
	examples := make(map[string]any)

	for _, table := range tables {
		if table.Namespace == "" {
			continue // Skip join tables
		}
		typeName := lowerFirst(pascalCase(table.SQLName()))
		examples[typeName] = generateTableExample(table, false)
	}

	return json.MarshalIndent(examples, "", "  ")
}

// exportGraphQL generates a GraphQL schema from the database schema.
func exportGraphQL(tables []*ast.TableDef, cfg *ExportConfig) ([]byte, error) {
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
				enumName := pascalCase(table.SQLName()) + pascalCase(col.Name)
				if cfg.IncludeDescriptions && col.Docs != "" {
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

	// Generate Query type for schema exploration
	sb.WriteString("type Query {\n")
	for _, table := range sortedTables {
		if table.Namespace == "" {
			continue // Skip join tables
		}
		typeName := pascalCase(table.SQLName())
		fieldName := lowerFirst(typeName)
		sb.WriteString(fmt.Sprintf("  %s(id: ID!): %s\n", fieldName, typeName))
	}
	sb.WriteString("}\n")

	return []byte(sb.String()), nil
}

// lowerFirst returns the string with the first character lowercased.
func lowerFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToLower(s[:1]) + s[1:]
}

// generateGraphQLType generates a GraphQL type for a single table.
func generateGraphQLType(sb *strings.Builder, table *ast.TableDef, cfg *ExportConfig) {
	name := pascalCase(table.SQLName())

	// Add doc comment if present
	if cfg.IncludeDescriptions && table.Docs != "" {
		sb.WriteString(fmt.Sprintf("\"\"\"%s\"\"\"\n", table.Docs))
	}

	sb.WriteString(fmt.Sprintf("type %s {\n", name))

	for _, col := range table.Columns {
		gqlType := columnToGraphQLType(col, table, cfg)

		// Add doc comment if present
		if cfg.IncludeDescriptions && col.Docs != "" {
			sb.WriteString(fmt.Sprintf("  \"\"\"%s\"\"\"\n", col.Docs))
		}

		sb.WriteString(fmt.Sprintf("  %s: %s\n", col.Name, gqlType))
	}

	sb.WriteString("}\n\n")
}

// columnToGraphQLType converts a column type to GraphQL type.
func columnToGraphQLType(col *ast.ColumnDef, table *ast.TableDef, cfg *ExportConfig) string {
	var baseType string

	// Handle enum specially
	if col.Type == "enum" {
		baseType = pascalCase(table.SQLName()) + pascalCase(col.Name)
	} else {
		// Map types
		switch col.Type {
		case "id", "uuid":
			baseType = "ID"
		case "string", "text":
			baseType = "String"
		case "integer":
			baseType = "Int"
		case "float":
			baseType = "Float"
		case "decimal":
			baseType = "String" // Decimal as string for precision
		case "boolean":
			baseType = "Boolean"
		case "date", "time", "date_time":
			baseType = "DateTime"
		case "json":
			baseType = "JSON"
		case "base64":
			baseType = "String"
		default:
			baseType = "String"
		}
	}

	if col.Nullable {
		return baseType
	}
	return baseType + "!"
}
