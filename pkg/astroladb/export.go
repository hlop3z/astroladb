package astroladb

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/hlop3z/astroladb/internal/ast"
	"github.com/hlop3z/astroladb/internal/metadata"
	"github.com/hlop3z/astroladb/internal/strutil"
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

// exportContext is the internal config used by export functions.
// It combines the public ExportConfig with internal metadata.
type exportContext struct {
	*ExportConfig
	Metadata *metadata.Metadata
}

// TypeConverter interface defines methods for converting Alab types to target language types.
type TypeConverter interface {
	ConvertType(col *ast.ColumnDef) string
	ConvertNullable(baseType string, nullable bool) string
	FormatName(name string) string
}

// BaseConverter provides common functionality for type converters.
type BaseConverter struct {
	TypeMap        map[string]string
	NullableFormat string
}

// ConvertNullable applies the nullable format to a base type.
func (b *BaseConverter) ConvertNullable(baseType string, nullable bool) string {
	if !nullable {
		return baseType
	}
	return fmt.Sprintf(b.NullableFormat, baseType)
}

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
	if col.Type == "enum" && len(col.TypeArgs) > 0 {
		var enumValues []string
		if values, ok := col.TypeArgs[0].([]string); ok {
			enumValues = values
		} else if values, ok := col.TypeArgs[0].([]any); ok {
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

// FormatName formats a name to TypeScript conventions (camelCase for fields).
func (c *TypeScriptConverter) FormatName(name string) string {
	return strutil.ToCamelCase(name)
}

// GoConverter converts types to Go.
type GoConverter struct {
	BaseConverter
}

// NewGoConverter creates a new Go converter.
func NewGoConverter() *GoConverter {
	return &GoConverter{
		BaseConverter: BaseConverter{
			TypeMap: map[string]string{
				"id":       "string",
				"uuid":     "string",
				"string":   "string",
				"text":     "string",
				"integer":  "int",
				"float":    "float64",
				"decimal":  "string",
				"boolean":  "bool",
				"date":     "time.Time",
				"time":     "time.Time",
				"datetime": "time.Time",
				"json":     "map[string]any",
				"base64":   "[]byte",
			},
			NullableFormat: "*%s",
		},
	}
}

// ConvertType converts a column definition to Go type.
func (c *GoConverter) ConvertType(col *ast.ColumnDef) string {
	// Get base type from TypeRegistry
	base := "string"
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

// FormatName formats a name to Go conventions (PascalCase).
func (c *GoConverter) FormatName(name string) string {
	return strutil.ToPascalCase(name)
}

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

// FormatName formats a name to Python conventions (snake_case).
func (c *PythonConverter) FormatName(name string) string {
	return strutil.ToSnakeCase(name)
}

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

// OpenAPIConverter converts types to OpenAPI specification.
type OpenAPIConverter struct {
	BaseConverter
}

// NewOpenAPIConverter creates a new OpenAPI converter.
func NewOpenAPIConverter() *OpenAPIConverter {
	return &OpenAPIConverter{
		BaseConverter: BaseConverter{
			TypeMap: map[string]string{
				"id":       "string",
				"uuid":     "string",
				"string":   "string",
				"text":     "string",
				"integer":  "integer",
				"float":    "number",
				"decimal":  "string",
				"boolean":  "boolean",
				"date":     "string",
				"time":     "string",
				"datetime": "string",
				"json":     "object",
				"base64":   "string",
			},
			NullableFormat: "%s",
		},
	}
}

// ConvertType converts a column definition to OpenAPI type (returns the type string).
func (c *OpenAPIConverter) ConvertType(col *ast.ColumnDef) string {
	// Get base type from registry
	typeDef := types.Get(col.Type)
	if typeDef != nil {
		return typeDef.OpenAPI.Type
	}
	return "string"
}

// FormatName formats a name to OpenAPI conventions (snake_case typically).
func (c *OpenAPIConverter) FormatName(name string) string {
	return strutil.ToSnakeCase(name)
}

// typeConverters is a global map of all available type converters.
var typeConverters = map[string]TypeConverter{
	FormatTypeScript: NewTypeScriptConverter(),
	FormatGo:         NewGoConverter(),
	FormatPython:     NewPythonConverter(),
	FormatRust:       NewRustConverter(false),
	FormatOpenAPI:    NewOpenAPIConverter(),
	"graphql":        NewGraphQLConverter(),
}

// GetConverter returns the type converter for the given format.
func GetConverter(format string) TypeConverter {
	return typeConverters[format]
}

// exportOpenAPI generates an OpenAPI 3.0 specification from the schema.
// Uses x-db extension for complete database metadata per PLAN.md.
func exportOpenAPI(tables []*ast.TableDef, cfg *exportContext) ([]byte, error) {
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
				"generator": "alab",
				"version":   "1.0.0",
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

	return json.MarshalIndent(spec, "", "  ")
}

// generateOpenAPIPaths generates a single /schemas endpoint with all models.
func generateOpenAPIPaths(tables []*ast.TableDef, cfg *exportContext) map[string]any {
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
			ns := joinTableNS[table.FullName()]
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
		"table":   table.FullName(),
		"columns": buildColumnList(table),
		"example": generateTableExample(table, false),
	}

	// Primary key
	if pk := findPrimaryKey(table); len(pk) > 0 {
		entry["primary_key"] = pk[0]
	}

	// Timestamps
	if table.HasColumn("created_at") && table.HasColumn("updated_at") {
		entry["timestamps"] = true
	}

	// Soft delete
	if table.HasColumn("deleted_at") {
		entry["soft_delete"] = true
	}

	// Auditable
	if table.HasColumn("created_by") && table.HasColumn("updated_by") {
		entry["auditable"] = true
	}

	// Foreign keys
	if fks := buildForeignKeys(table); len(fks) > 0 {
		entry["foreign_keys"] = fks
	}

	return entry
}

// buildJoinTableEntry builds a join table entry with essential metadata.
func buildJoinTableEntry(table *ast.TableDef) map[string]any {
	entry := map[string]any{
		"table":   table.FullName(),
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
	case "datetime":
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
func generateOpenAPISchemas(tables []*ast.TableDef, cfg *exportContext) map[string]any {
	schemas := make(map[string]any)

	for _, table := range tables {
		name := strutil.ToPascalCase(table.FullName())
		schemas[name] = tableToOpenAPISchema(table, tables, cfg)
	}

	return schemas
}

// tableToOpenAPISchema converts a table definition to an OpenAPI schema object.
func tableToOpenAPISchema(table *ast.TableDef, allTables []*ast.TableDef, cfg *exportContext) map[string]any {
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

	if table.Docs != "" {
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
		"table":     table.FullName(),
		"namespace": table.Namespace,
	}

	// Primary key
	pk := findPrimaryKey(table)
	if len(pk) > 0 {
		xdb["primary_key"] = pk
	}

	// Check if this is a join table and add joinTable metadata
	if meta != nil {
		if joinTableInfo := findJoinTableInfo(table.FullName(), meta); joinTableInfo != nil {
			xdb["join_table"] = joinTableInfo
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
		xdb["soft_delete"] = true
	}

	// Sort by
	if len(table.SortBy) > 0 {
		xdb["sort_by"] = table.SortBy
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
			"name":    table.FullName() + "_pkey",
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
			idxDef["name"] = table.FullName() + "_" + strings.Join(idx.Columns, "_") + "_idx"
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
				"name":    table.FullName() + "_" + col.Name + "_key",
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
					"name":    table.FullName() + "_" + col.Name + "_idx",
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
			relType := "has_many"
			if col.Unique {
				relType = "has_one"
			}

			relationships[relationName] = map[string]any{
				"type":         relType,
				"target":       otherTable.QualifiedName(),
				"target_table": otherTable.FullName(),
				"foreign_key":  col.Name,
				"local_key":    "id",
				"backref":      relationName,
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

				relationships[targetName] = map[string]any{
					"type":         "many_to_many",
					"target":       m2m.Target,
					"target_table": strings.ReplaceAll(m2m.Target, ".", "_"),
					"local_key":    "id",
					"backref":      table.Name,
					"through": map[string]any{
						"table":       m2m.JoinTable,
						"local_key":   m2m.SourceFK,
						"foreign_key": m2m.TargetFK,
					},
				}
			}

			// Check if this table is the target of the m2m relationship
			if m2m.Target == table.QualifiedName() {
				// This table has an inverse manyToMany
				sourceParts := strings.Split(m2m.Source, ".")
				sourceName := sourceParts[len(sourceParts)-1]

				relationships[sourceName] = map[string]any{
					"type":         "many_to_many",
					"target":       m2m.Source,
					"target_table": strings.ReplaceAll(m2m.Source, ".", "_"),
					"local_key":    "id",
					"backref":      table.Name,
					"through": map[string]any{
						"table":       m2m.JoinTable,
						"local_key":   m2m.TargetFK,
						"foreign_key": m2m.SourceFK,
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

// columnToOpenAPIProperty converts a column definition to an OpenAPI property.
func columnToOpenAPIProperty(col *ast.ColumnDef, table *ast.TableDef, cfg *exportContext) map[string]any {
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
	if col.Docs != "" {
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
		xdb["auto_managed"] = false
	}
	if col.Name == "created_by" || col.Name == "updated_by" {
		xdb["auto_managed"] = false
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
			xdb["on_delete"] = col.Reference.OnDelete
		}
		if col.Reference.OnUpdate != "" {
			xdb["on_update"] = col.Reference.OnUpdate
		}

		// Relationship name (strip _id suffix)
		relationName := strings.TrimSuffix(col.Name, "_id")
		if relationName != "" {
			xdb["relation"] = relationName
		}

		// Inverse relationship name
		xdb["inverse_of"] = table.Name
	}

	// SQL type per dialect
	sqlType := buildSQLType(col)
	if len(sqlType) > 0 {
		xdb["sql_type"] = sqlType
	}

	// Computed and virtual columns
	if col.Computed != nil {
		xdb["virtual"] = true
		xdb["computed"] = col.Computed
		if col.Virtual {
			xdb["storage"] = "virtual"
		} else {
			xdb["storage"] = "stored"
		}
	} else if col.Virtual {
		xdb["virtual"] = true
		xdb["storage"] = "app_only"
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

// exportTypeScript generates TypeScript type definitions from the schema.
func exportTypeScript(tables []*ast.TableDef, cfg *exportContext) ([]byte, error) {
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
		sb.WriteString(fmt.Sprintf(" * %s\n", table.Docs))
		if table.Deprecated != "" {
			sb.WriteString(fmt.Sprintf(" * @deprecated %s\n", table.Deprecated))
		}
		sb.WriteString(" */\n")
	}

	sb.WriteString(fmt.Sprintf("export interface %s {\n", name))

	for _, col := range table.Columns {
		// Add JSDoc for column
		if col.Docs != "" {
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
	return GetConverter(FormatTypeScript).ConvertType(col)
}

// exportGo generates Go struct definitions from the schema.
func exportGo(tables []*ast.TableDef, cfg *exportContext) ([]byte, error) {
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
		typeName := strutil.ToPascalCase(table.FullName())
		fmt.Fprintf(&buf, "\t\"%s\": \"%s\",\n", table.QualifiedName(), typeName)
	}
	buf.WriteString("}\n")

	return buf.Bytes(), nil
}

// generateGoStruct generates a Go struct for a single table.
func generateGoStruct(buf *bytes.Buffer, table *ast.TableDef, cfg *exportContext) {
	name := strutil.ToPascalCase(table.FullName())

	// Add doc comment if present
	if table.Docs != "" {
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
		if col.Docs != "" {
			fmt.Fprintf(buf, "\t// %s\n", col.Docs)
		}
		fmt.Fprintf(buf, "\t%s %s %s\n", strutil.ToPascalCase(col.Name), goType, jsonTag)
	}

	buf.WriteString("}\n\n")
}

// mapToGoType converts a column type to its Go equivalent.
func mapToGoType(col *ast.ColumnDef) string {
	return GetConverter(FormatGo).ConvertType(col)
}

// exportPython generates Python dataclass definitions from the schema.
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
	sortedTables := make([]*ast.TableDef, len(tables))
	copy(sortedTables, tables)
	sort.Slice(sortedTables, func(i, j int) bool {
		return sortedTables[i].QualifiedName() < sortedTables[j].QualifiedName()
	})

	// First pass: generate enum classes
	for _, table := range sortedTables {
		for _, col := range table.Columns {
			if col.Type == "enum" && len(col.TypeArgs) > 0 {
				enumName := strutil.ToPascalCase(table.FullName()) + strutil.ToPascalCase(col.Name)
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
		if col.Docs != "" {
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
		if col.Docs != "" {
			sb.WriteString(fmt.Sprintf("    # %s\n", col.Docs))
		}

		sb.WriteString(fmt.Sprintf("    %s: %s = None\n", col.Name, pyType))
	}

	sb.WriteString("\n")
}

// columnToPythonType converts a column type to Python type.
func columnToPythonType(col *ast.ColumnDef, table *ast.TableDef) string {
	converter := GetConverter(FormatPython).(*PythonConverter)
	converter.TableName = table.FullName()
	return converter.ConvertType(col)
}

// exportRust generates Rust struct definitions with serde from the schema.
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
		if col.Docs != "" {
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
func columnToRustType(col *ast.ColumnDef, table *ast.TableDef, cfg *exportContext) string {
	converter := GetConverter(FormatRust).(*RustConverter)
	converter.UseChrono = cfg.UseChrono
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

// exportGraphQLExamples generates example data for GraphQL types as JSON.
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

// lowerFirst returns the string with the first character lowercased.
func lowerFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToLower(s[:1]) + s[1:]
}

// generateGraphQLType generates a GraphQL type for a single table.
func generateGraphQLType(sb *strings.Builder, table *ast.TableDef, cfg *exportContext) {
	name := strutil.ToPascalCase(table.FullName())

	// Add doc comment if present
	if table.Docs != "" {
		sb.WriteString(fmt.Sprintf("\"\"\"%s\"\"\"\n", table.Docs))
	}

	sb.WriteString(fmt.Sprintf("type %s {\n", name))

	for _, col := range table.Columns {
		gqlType := columnToGraphQLType(col, table, cfg)

		// Add doc comment if present
		if col.Docs != "" {
			sb.WriteString(fmt.Sprintf("  \"\"\"%s\"\"\"\n", col.Docs))
		}

		sb.WriteString(fmt.Sprintf("  %s: %s\n", col.Name, gqlType))
	}

	sb.WriteString("}\n\n")
}

// columnToGraphQLType converts a column type to GraphQL type.
func columnToGraphQLType(col *ast.ColumnDef, table *ast.TableDef, cfg *exportContext) string {
	converter := GetConverter("graphql").(*GraphQLConverter)
	converter.TableName = table.FullName()
	baseType := converter.ConvertType(col)

	if col.Nullable {
		return baseType
	}
	return baseType + "!"
}
