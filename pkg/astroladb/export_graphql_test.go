package astroladb

import (
	"strings"
	"testing"

	"github.com/hlop3z/astroladb/internal/ast"
	"github.com/hlop3z/astroladb/internal/metadata"
)

// ===========================================================================
// GraphQL Export Tests
// ===========================================================================

func TestExportGraphQL_BasicType(t *testing.T) {
	tables := []*ast.TableDef{testTable()}
	cfg := &exportContext{ExportConfig: &ExportConfig{}}

	data, err := exportGraphQL(tables, cfg)
	if err != nil {
		t.Fatalf("exportGraphQL failed: %v", err)
	}

	output := string(data)

	// Check header
	if !strings.Contains(output, "# Auto-generated GraphQL schema") {
		t.Error("missing header comment")
	}

	// Check custom scalars
	if !strings.Contains(output, "scalar DateTime") {
		t.Error("missing DateTime scalar")
	}
	if !strings.Contains(output, "scalar JSON") {
		t.Error("missing JSON scalar")
	}

	// Check type declaration
	if !strings.Contains(output, "type AuthUser {") {
		t.Error("missing type declaration")
	}

	// Check required fields have !
	if !strings.Contains(output, "id: ID!") {
		t.Error("missing required id field with !")
	}

	// Check nullable fields don't have !
	if !strings.Contains(output, "age: Int") && strings.Contains(output, "age: Int!") {
		t.Error("nullable age should not have !")
	}
}

func TestExportGraphQL_EnumTypes(t *testing.T) {
	tables := []*ast.TableDef{testTable()}
	cfg := &exportContext{ExportConfig: &ExportConfig{}}

	data, err := exportGraphQL(tables, cfg)
	if err != nil {
		t.Fatalf("exportGraphQL failed: %v", err)
	}

	output := string(data)

	// Check enum declaration
	if !strings.Contains(output, "enum AuthUserStatus {") {
		t.Error("missing enum declaration")
	}

	// Check enum values (uppercase)
	if !strings.Contains(output, "ACTIVE") {
		t.Error("missing ACTIVE enum value")
	}
	if !strings.Contains(output, "PENDING") {
		t.Error("missing PENDING enum value")
	}
	if !strings.Contains(output, "SUSPENDED") {
		t.Error("missing SUSPENDED enum value")
	}
}

func TestExportGraphQL_QueryType(t *testing.T) {
	tables := []*ast.TableDef{testTable(), testTableWithFK()}
	cfg := &exportContext{ExportConfig: &ExportConfig{}}

	data, err := exportGraphQL(tables, cfg)
	if err != nil {
		t.Fatalf("exportGraphQL failed: %v", err)
	}

	output := string(data)

	// Check Query type
	if !strings.Contains(output, "type Query {") {
		t.Error("missing Query type")
	}

	// Check query fields (lowerFirst camelCase)
	if !strings.Contains(output, "authUser: AuthUser") {
		t.Error("missing authUser query field")
	}
	if !strings.Contains(output, "blogPost: BlogPost") {
		t.Error("missing blogPost query field")
	}
}

func TestExportGraphQL_AllTypes(t *testing.T) {
	tables := []*ast.TableDef{testTable()}
	cfg := &exportContext{ExportConfig: &ExportConfig{}}

	data, err := exportGraphQL(tables, cfg)
	if err != nil {
		t.Fatalf("exportGraphQL failed: %v", err)
	}

	output := string(data)

	tests := []struct {
		field    string
		expected string
	}{
		{"id", "id: ID!"},
		{"email", "email: String!"},
		{"age", "age: Int"},             // nullable - no !
		{"balance", "balance: String!"}, // decimal as string
		{"is_active", "is_active: Boolean!"},
		{"status", "status: AuthUserStatus!"},
		{"metadata", "metadata: JSON"},         // nullable
		{"avatar", "avatar: String"},           // nullable
		{"birth_date", "birth_date: DateTime"}, // nullable
		{"created_at", "created_at: DateTime!"},
	}

	for _, tt := range tests {
		if !strings.Contains(output, tt.expected) {
			t.Errorf("field %s: expected %q not found in output:\n%s", tt.field, tt.expected, output)
		}
	}
}

func TestExportGraphQL_WithDocs(t *testing.T) {
	table := &ast.TableDef{
		Namespace: "test",
		Name:      "documented",
		Docs:      "This is a documented table",
		Columns: []*ast.ColumnDef{
			{Name: "id", Type: "uuid", PrimaryKey: true},
			{Name: "name", Type: "string", Docs: "The name field"},
		},
	}

	cfg := &exportContext{ExportConfig: &ExportConfig{}}
	data, err := exportGraphQL([]*ast.TableDef{table}, cfg)
	if err != nil {
		t.Fatalf("exportGraphQL failed: %v", err)
	}

	output := string(data)

	// Check type-level doc
	if !strings.Contains(output, "\"\"\"This is a documented table\"\"\"") {
		t.Error("missing type-level doc comment")
	}

	// Check field-level doc
	if !strings.Contains(output, "\"\"\"The name field\"\"\"") {
		t.Error("missing field-level doc comment")
	}
}

func TestExportGraphQLExamples(t *testing.T) {
	tables := []*ast.TableDef{testTable()}
	cfg := &exportContext{ExportConfig: &ExportConfig{}}

	data, err := exportGraphQLExamples(tables, cfg)
	if err != nil {
		t.Fatalf("exportGraphQLExamples failed: %v", err)
	}

	output := string(data)

	// Check it's valid JSON with example values
	if !strings.Contains(output, "\"authUser\"") {
		t.Error("missing authUser key in examples")
	}

	// Check example values present
	if !strings.Contains(output, "\"email\"") {
		t.Error("missing email field in example")
	}
}

// ===========================================================================
// Converter FormatName Tests
// ===========================================================================

func TestTypeScriptConverter_FormatName(t *testing.T) {
	converter := NewTypeScriptConverter()
	tests := []struct {
		input string
		want  string
	}{
		{"user_name", "userName"},
		{"created_at", "createdAt"},
		{"id", "id"},
		{"first_name", "firstName"},
	}

	for _, tt := range tests {
		got := converter.FormatName(tt.input)
		if got != tt.want {
			t.Errorf("FormatName(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestGoConverter_FormatName(t *testing.T) {
	converter := NewGoConverter()
	tests := []struct {
		input string
		want  string
	}{
		{"user_name", "UserName"},
		{"created_at", "CreatedAt"},
		{"id", "Id"},
		{"first_name", "FirstName"},
	}

	for _, tt := range tests {
		got := converter.FormatName(tt.input)
		if got != tt.want {
			t.Errorf("FormatName(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestPythonConverter_FormatName(t *testing.T) {
	converter := NewPythonConverter()
	tests := []struct {
		input string
		want  string
	}{
		{"userName", "user_name"},
		{"createdAt", "created_at"},
		{"ID", "id"}, // ID is treated as a known acronym
		{"firstName", "first_name"},
		{"userID", "user_id"},
	}

	for _, tt := range tests {
		got := converter.FormatName(tt.input)
		if got != tt.want {
			t.Errorf("FormatName(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestRustConverter_FormatName(t *testing.T) {
	converter := NewRustConverter(false)
	tests := []struct {
		input string
		want  string
	}{
		{"userName", "user_name"},
		{"createdAt", "created_at"},
		{"ID", "id"}, // ID is treated as a known acronym
		{"firstName", "first_name"},
		{"userID", "user_id"},
	}

	for _, tt := range tests {
		got := converter.FormatName(tt.input)
		if got != tt.want {
			t.Errorf("FormatName(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestGraphQLConverter_FormatName(t *testing.T) {
	converter := NewGraphQLConverter()
	tests := []struct {
		input string
		want  string
	}{
		{"user_name", "userName"},
		{"created_at", "createdAt"},
		{"id", "id"},
		{"first_name", "firstName"},
	}

	for _, tt := range tests {
		got := converter.FormatName(tt.input)
		if got != tt.want {
			t.Errorf("FormatName(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestOpenAPIConverter_FormatName(t *testing.T) {
	converter := NewOpenAPIConverter()
	tests := []struct {
		input string
		want  string
	}{
		{"userName", "user_name"},
		{"createdAt", "created_at"},
		{"ID", "id"}, // ID is treated as a known acronym
		{"firstName", "first_name"},
		{"userID", "user_id"},
	}

	for _, tt := range tests {
		got := converter.FormatName(tt.input)
		if got != tt.want {
			t.Errorf("FormatName(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

// ===========================================================================
// ConvertNullable Tests
// ===========================================================================

func TestBaseConverter_ConvertNullable(t *testing.T) {
	converter := &BaseConverter{
		NullableFormat: "Option<%s>",
	}

	// Non-nullable
	got := converter.ConvertNullable("String", false)
	if got != "String" {
		t.Errorf("ConvertNullable(String, false) = %q, want %q", got, "String")
	}

	// Nullable
	got = converter.ConvertNullable("String", true)
	if got != "Option<String>" {
		t.Errorf("ConvertNullable(String, true) = %q, want %q", got, "Option<String>")
	}
}

func TestGraphQLConverter_ConvertNullable(t *testing.T) {
	converter := NewGraphQLConverter()

	// Non-nullable (gets ! suffix)
	got := converter.ConvertNullable("String", false)
	if got != "String!" {
		t.Errorf("ConvertNullable(String, false) = %q, want %q", got, "String!")
	}

	// Nullable (no suffix)
	got = converter.ConvertNullable("String", true)
	if got != "String" {
		t.Errorf("ConvertNullable(String, true) = %q, want %q", got, "String")
	}
}

// ===========================================================================
// Converter ConvertType Tests
// ===========================================================================

func TestGraphQLConverter_ConvertType(t *testing.T) {
	converter := NewGraphQLConverter()
	converter.TableName = "test_table"

	tests := []struct {
		col  *ast.ColumnDef
		want string
	}{
		{&ast.ColumnDef{Name: "id", Type: "id"}, "ID"},
		{&ast.ColumnDef{Name: "uuid", Type: "uuid"}, "ID"},
		{&ast.ColumnDef{Name: "name", Type: "string"}, "String"},
		{&ast.ColumnDef{Name: "body", Type: "text"}, "String"},
		{&ast.ColumnDef{Name: "count", Type: "integer"}, "Int"},
		{&ast.ColumnDef{Name: "price", Type: "float"}, "Float"},
		{&ast.ColumnDef{Name: "amount", Type: "decimal"}, "String"},
		{&ast.ColumnDef{Name: "active", Type: "boolean"}, "Boolean"},
		{&ast.ColumnDef{Name: "created", Type: "datetime"}, "DateTime"},
		{&ast.ColumnDef{Name: "data", Type: "json"}, "JSON"},
		{&ast.ColumnDef{Name: "status", Type: "enum"}, "TestTableStatus"}, // enum with table name
		{&ast.ColumnDef{Name: "unknown", Type: "unknown_type"}, "String"}, // fallback
	}

	for _, tt := range tests {
		got := converter.ConvertType(tt.col)
		if got != tt.want {
			t.Errorf("ConvertType(%s, %s) = %q, want %q", tt.col.Name, tt.col.Type, got, tt.want)
		}
	}
}

func TestOpenAPIConverter_ConvertType(t *testing.T) {
	converter := NewOpenAPIConverter()

	tests := []struct {
		col  *ast.ColumnDef
		want string
	}{
		{&ast.ColumnDef{Name: "id", Type: "id"}, "string"},
		{&ast.ColumnDef{Name: "count", Type: "integer"}, "integer"},
		{&ast.ColumnDef{Name: "price", Type: "float"}, "number"},
		{&ast.ColumnDef{Name: "active", Type: "boolean"}, "boolean"},
		{&ast.ColumnDef{Name: "unknown", Type: "unknown_type"}, "string"}, // fallback
	}

	for _, tt := range tests {
		got := converter.ConvertType(tt.col)
		if got != tt.want {
			t.Errorf("ConvertType(%s, %s) = %q, want %q", tt.col.Name, tt.col.Type, got, tt.want)
		}
	}
}

// ===========================================================================
// Helper Function Tests
// ===========================================================================

func TestPluralize(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"user", "users"},
		{"post", "posts"},
		{"class", "classes"},       // ends with s
		{"box", "boxes"},           // ends with x
		{"church", "churches"},     // ends with ch
		{"dish", "dishes"},         // ends with sh
		{"category", "categories"}, // ends with consonant+y
		{"day", "days"},            // ends with vowel+y
		{"toy", "toys"},            // ends with vowel+y
	}

	for _, tt := range tests {
		got := pluralize(tt.input)
		if got != tt.want {
			t.Errorf("pluralize(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestMatchesTable(t *testing.T) {
	table := &ast.TableDef{
		Namespace: "auth",
		Name:      "user",
	}

	tests := []struct {
		ref  string
		want bool
	}{
		{"auth.user", true},   // Full qualified name
		{".user", true},       // Same namespace format
		{"user", true},        // Plain name
		{"other.user", false}, // Different namespace
		{"auth.post", false},  // Different table
		{"post", false},       // Different table
	}

	for _, tt := range tests {
		got := matchesTable(tt.ref, table)
		if got != tt.want {
			t.Errorf("matchesTable(%q, auth.user) = %v, want %v", tt.ref, got, tt.want)
		}
	}
}

func TestLowerFirst(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"User", "user"},
		{"AuthUser", "authUser"},
		{"ID", "iD"},
		{"a", "a"},
		{"A", "a"},
		{"", ""},
	}

	for _, tt := range tests {
		got := lowerFirst(tt.input)
		if got != tt.want {
			t.Errorf("lowerFirst(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

// ===========================================================================
// generateColumnExample Tests
// ===========================================================================

func TestGenerateColumnExample_ByName(t *testing.T) {
	tests := []struct {
		colName string
		colType string
		want    any
	}{
		{"id", "uuid", "550e8400-e29b-41d4-a716-446655440000"},
		{"email", "string", "user@example.com"},
		{"username", "string", "johndoe"},
		{"password", "string", "SecureP@ssw0rd123"},
		{"name", "string", "John Doe"},
		{"first_name", "string", "John"},
		{"last_name", "string", "Doe"},
		{"title", "string", "Sample Title"},
		{"slug", "string", "sample-title"},
		{"phone", "string", "+1234567890"},
		{"url", "string", "https://example.com"},
		{"ip", "string", "192.168.1.1"},
		{"country", "string", "US"},
		{"currency", "string", "USD"},
		{"locale", "string", "en-US"},
		{"timezone", "string", "America/New_York"},
		{"color", "string", "#3498db"},
		{"created_at", "datetime", "2024-01-15T10:30:00Z"},
		{"deleted_at", "datetime", nil},
		{"created_by", "uuid", "550e8400-e29b-41d4-a716-446655440001"},
		{"is_active", "boolean", true},
		{"is_deleted", "boolean", false},
		{"price", "decimal", "99.99"},
		{"quantity", "integer", 1},
		{"rating", "decimal", "4.5"},
		{"age", "integer", 25},
		{"description", "text", "This is a sample description text."},
		{"summary", "string", "Brief summary of the content."},
		{"token", "string", "abc123def456ghi789jkl012mno345pqr678stu901vwx234yz"},
		{"code", "string", "ABC-123"},
	}

	for _, tt := range tests {
		col := &ast.ColumnDef{Name: tt.colName, Type: tt.colType}
		got := generateColumnExample(col)
		if got != tt.want {
			t.Errorf("generateColumnExample(%q) = %v, want %v", tt.colName, got, tt.want)
		}
	}
}

func TestGenerateColumnExample_ByType(t *testing.T) {
	tests := []struct {
		colName string
		colType string
		format  string
		want    any
	}{
		{"custom_id", "uuid", "", "550e8400-e29b-41d4-a716-446655440000"},
		{"custom_text", "text", "", "Sample text"},
		{"custom_int", "integer", "", 42},
		{"custom_float", "float", "", "123.45"},
		{"custom_bool", "boolean", "", true},
		{"custom_date", "date", "", "2024-01-15"},
		{"custom_time", "time", "", "10:30:00"},
		{"custom_dt", "datetime", "", "2024-01-15T10:30:00Z"},
		{"custom_json", "json", "", map[string]any{"key": "value"}},
		{"custom_b64", "base64", "", "SGVsbG8gV29ybGQh"},
		{"custom_email", "string", "email", "user@example.com"},
		{"custom_uri", "string", "uri", "https://example.com"},
		// Note: uuid type returns the standard UUID example, not the FK pattern
		{"ref_id", "uuid", "", "550e8400-e29b-41d4-a716-446655440000"},
	}

	for _, tt := range tests {
		col := &ast.ColumnDef{Name: tt.colName, Type: tt.colType, Format: tt.format}
		got := generateColumnExample(col)

		// For maps, check they're the same type
		if wantMap, ok := tt.want.(map[string]any); ok {
			gotMap, ok := got.(map[string]any)
			if !ok {
				t.Errorf("generateColumnExample(%q) = %T, want map[string]any", tt.colName, got)
				continue
			}
			if gotMap["key"] != wantMap["key"] {
				t.Errorf("generateColumnExample(%q) = %v, want %v", tt.colName, got, tt.want)
			}
			continue
		}

		if got != tt.want {
			t.Errorf("generateColumnExample(%q, type=%s, format=%s) = %v, want %v",
				tt.colName, tt.colType, tt.format, got, tt.want)
		}
	}
}

func TestGenerateColumnExample_Enum(t *testing.T) {
	// Enum with string slice
	col := &ast.ColumnDef{
		Name:     "status",
		Type:     "enum",
		TypeArgs: []any{[]string{"active", "pending", "suspended"}},
	}
	got := generateColumnExample(col)
	if got != "active" {
		t.Errorf("generateColumnExample(enum) = %v, want 'active'", got)
	}

	// Enum with any slice (Goja conversion)
	col2 := &ast.ColumnDef{
		Name:     "status",
		Type:     "enum",
		TypeArgs: []any{[]any{"draft", "published"}},
	}
	got2 := generateColumnExample(col2)
	if got2 != "draft" {
		t.Errorf("generateColumnExample(enum with []any) = %v, want 'draft'", got2)
	}

	// Empty enum
	col3 := &ast.ColumnDef{
		Name:     "status",
		Type:     "enum",
		TypeArgs: []any{},
	}
	got3 := generateColumnExample(col3)
	if got3 != "value" {
		t.Errorf("generateColumnExample(empty enum) = %v, want 'value'", got3)
	}
}

func TestGenerateColumnExample_IntegerWithMin(t *testing.T) {
	min := 10
	col := &ast.ColumnDef{
		Name: "custom_value", // Use a name that doesn't match semantic patterns
		Type: "integer",
		Min:  &min,
	}
	got := generateColumnExample(col)
	if got != 10 {
		t.Errorf("generateColumnExample(integer with min) = %v, want 10", got)
	}
}

func TestGenerateColumnExample_BooleanWithDefault(t *testing.T) {
	col := &ast.ColumnDef{
		Name:    "custom_flag", // Use a name that doesn't match semantic patterns
		Type:    "boolean",
		Default: false,
	}
	got := generateColumnExample(col)
	if got != false {
		t.Errorf("generateColumnExample(boolean with default false) = %v, want false", got)
	}
}

// ===========================================================================
// NewRustConverter Tests
// ===========================================================================

func TestNewRustConverter_WithChrono(t *testing.T) {
	converter := NewRustConverter(true)

	if !converter.UseChrono {
		t.Error("UseChrono should be true")
	}

	// Check that chrono types are in the TypeMap
	if converter.TypeMap["date"] != "NaiveDate" {
		t.Errorf("date type = %q, want NaiveDate", converter.TypeMap["date"])
	}
	if converter.TypeMap["time"] != "NaiveTime" {
		t.Errorf("time type = %q, want NaiveTime", converter.TypeMap["time"])
	}
	if converter.TypeMap["datetime"] != "DateTime<Utc>" {
		t.Errorf("datetime type = %q, want DateTime<Utc>", converter.TypeMap["datetime"])
	}
}

func TestNewRustConverter_WithoutChrono(t *testing.T) {
	converter := NewRustConverter(false)

	if converter.UseChrono {
		t.Error("UseChrono should be false")
	}

	// Check that string types are in the TypeMap
	if converter.TypeMap["date"] != "String" {
		t.Errorf("date type = %q, want String", converter.TypeMap["date"])
	}
	if converter.TypeMap["time"] != "String" {
		t.Errorf("time type = %q, want String", converter.TypeMap["time"])
	}
	if converter.TypeMap["datetime"] != "String" {
		t.Errorf("datetime type = %q, want String", converter.TypeMap["datetime"])
	}
}

// ===========================================================================
// TypeScript Enum with []any (Goja conversion) Tests
// ===========================================================================

func TestTypeScriptConverter_ConvertType_EnumWithAnySlice(t *testing.T) {
	converter := NewTypeScriptConverter()

	// Goja converts []string to []any
	col := &ast.ColumnDef{
		Name:     "status",
		Type:     "enum",
		TypeArgs: []any{[]any{"active", "pending"}},
	}

	got := converter.ConvertType(col)
	expected := "'active' | 'pending'"

	if got != expected {
		t.Errorf("ConvertType(enum with []any) = %q, want %q", got, expected)
	}
}

func TestTypeScriptConverter_ConvertType_UnknownType(t *testing.T) {
	converter := NewTypeScriptConverter()

	col := &ast.ColumnDef{
		Name: "custom",
		Type: "unknown_custom_type",
	}

	got := converter.ConvertType(col)
	if got != "unknown" {
		t.Errorf("ConvertType(unknown) = %q, want 'unknown'", got)
	}
}

// ===========================================================================
// buildPropertyXDB Tests
// ===========================================================================

func TestBuildPropertyXDB_WithComputed(t *testing.T) {
	table := &ast.TableDef{
		Namespace: "test",
		Name:      "item",
	}
	col := &ast.ColumnDef{
		Name:     "full_name",
		Type:     "string",
		Computed: map[string]string{"expr": "first_name || ' ' || last_name"},
	}

	xdb := buildPropertyXDB(col, table, nil)

	if xdb["virtual"] != true {
		t.Error("computed column should have virtual=true")
	}
	if xdb["computed"] == nil {
		t.Error("computed column should have computed field")
	}
}

func TestBuildPropertyXDB_WithFK(t *testing.T) {
	table := &ast.TableDef{
		Namespace: "blog",
		Name:      "post",
	}
	col := &ast.ColumnDef{
		Name: "author_id",
		Type: "uuid",
		Reference: &ast.Reference{
			Table:    "auth.user",
			Column:   "id",
			OnDelete: "CASCADE",
			OnUpdate: "SET_NULL",
		},
	}

	xdb := buildPropertyXDB(col, table, nil)

	if xdb["ref"] != "auth.user" {
		t.Errorf("ref = %v, want auth.user", xdb["ref"])
	}
	if xdb["fk"] != "auth.user.id" {
		t.Errorf("fk = %v, want auth.user.id", xdb["fk"])
	}
	if xdb["onDelete"] != "cascade" {
		t.Errorf("onDelete = %v, want cascade", xdb["onDelete"])
	}
	if xdb["relation"] != "author" {
		t.Errorf("relation = %v, want author", xdb["relation"])
	}
	if xdb["inverseOf"] != "posts" {
		t.Errorf("inverseOf = %v, want posts", xdb["inverseOf"])
	}
}

func TestBuildPropertyXDB_WithDefault(t *testing.T) {
	table := &ast.TableDef{
		Namespace: "test",
		Name:      "item",
	}
	col := &ast.ColumnDef{
		Name:    "active",
		Type:    "boolean",
		Default: true,
	}

	xdb := buildPropertyXDB(col, table, nil)

	if xdb["default"] != true {
		t.Error("should have default=true")
	}
}

func TestBuildPropertyXDB_AutoManaged(t *testing.T) {
	table := &ast.TableDef{
		Namespace: "test",
		Name:      "item",
	}

	tests := []string{"created_at", "updated_at", "deleted_at", "created_by", "updated_by"}

	for _, name := range tests {
		col := &ast.ColumnDef{
			Name: name,
			Type: "datetime",
		}

		xdb := buildPropertyXDB(col, table, nil)

		if xdb["autoManaged"] != false {
			t.Errorf("%s should have autoManaged=false", name)
		}
	}
}

func TestBuildPropertyXDB_GeneratedPK(t *testing.T) {
	table := &ast.TableDef{
		Namespace: "test",
		Name:      "item",
	}
	col := &ast.ColumnDef{
		Name:       "id",
		Type:       "id",
		PrimaryKey: true,
	}

	xdb := buildPropertyXDB(col, table, nil)

	if xdb["generated"] != true {
		t.Error("id column should have generated=true")
	}
}

// ===========================================================================
// buildColumnList Tests
// ===========================================================================

func TestBuildColumnList_WithEnumAnySlice(t *testing.T) {
	table := &ast.TableDef{
		Namespace: "test",
		Name:      "item",
		Columns: []*ast.ColumnDef{
			{Name: "status", Type: "enum", TypeArgs: []any{[]any{"a", "b", "c"}}},
		},
	}

	cols := buildColumnList(table)

	if len(cols) != 1 {
		t.Fatalf("expected 1 column, got %d", len(cols))
	}

	enumVals, ok := cols[0]["enum"].([]string)
	if !ok {
		t.Fatal("enum should be []string")
	}
	if len(enumVals) != 3 {
		t.Errorf("enum values len = %d, want 3", len(enumVals))
	}
}

// ===========================================================================
// generateTableExample Tests
// ===========================================================================

func TestGenerateTableExample_ForRequest(t *testing.T) {
	table := &ast.TableDef{
		Namespace: "auth",
		Name:      "user",
		Columns: []*ast.ColumnDef{
			{Name: "id", Type: "uuid", PrimaryKey: true},
			{Name: "email", Type: "string"},
			{Name: "password", Type: "string"},
			{Name: "created_at", Type: "datetime"},
		},
	}

	example := generateTableExample(table, true) // forRequest=true

	// Should exclude readOnly (id, created_at)
	if _, ok := example["id"]; ok {
		t.Error("id should be excluded from request example")
	}
	if _, ok := example["created_at"]; ok {
		t.Error("created_at should be excluded from request example")
	}

	// Should include writeOnly (password)
	if _, ok := example["password"]; !ok {
		t.Error("password should be included in request example")
	}
}

func TestGenerateTableExample_ForResponse(t *testing.T) {
	table := &ast.TableDef{
		Namespace: "auth",
		Name:      "user",
		Columns: []*ast.ColumnDef{
			{Name: "id", Type: "uuid", PrimaryKey: true},
			{Name: "email", Type: "string"},
			{Name: "password", Type: "string"},
			{Name: "created_at", Type: "datetime"},
		},
	}

	example := generateTableExample(table, false) // forRequest=false (response)

	// Should include readOnly (id, created_at)
	if _, ok := example["id"]; !ok {
		t.Error("id should be included in response example")
	}
	if _, ok := example["created_at"]; !ok {
		t.Error("created_at should be included in response example")
	}

	// Should exclude writeOnly (password)
	if _, ok := example["password"]; ok {
		t.Error("password should be excluded from response example")
	}
}

// ===========================================================================
// buildJoinTableEntry Tests
// ===========================================================================

func TestBuildJoinTableEntry(t *testing.T) {
	table := &ast.TableDef{
		Name: "user_roles",
		Columns: []*ast.ColumnDef{
			{Name: "user_id", Type: "uuid", Reference: &ast.Reference{Table: "auth.user", Column: "id"}},
			{Name: "role_id", Type: "uuid", Reference: &ast.Reference{Table: "auth.role", Column: "id"}},
		},
	}

	entry := buildJoinTableEntry(table)

	// Check table name
	if entry["table"] != "user_roles" {
		t.Errorf("table = %v, want user_roles", entry["table"])
	}

	// Check links
	links, ok := entry["links"].([]string)
	if !ok {
		t.Fatal("links should be []string")
	}
	if len(links) != 2 {
		t.Errorf("links len = %d, want 2", len(links))
	}
	if links[0] != "auth.user" {
		t.Errorf("links[0] = %v, want auth.user", links[0])
	}
	if links[1] != "auth.role" {
		t.Errorf("links[1] = %v, want auth.role", links[1])
	}

	// Check columns
	cols, ok := entry["columns"].([]map[string]any)
	if !ok {
		t.Fatal("columns should be []map[string]any")
	}
	if len(cols) != 2 {
		t.Errorf("columns len = %d, want 2", len(cols))
	}
}

// ===========================================================================
// findJoinTableInfo Tests
// ===========================================================================

func TestFindJoinTableInfo_NoMeta(t *testing.T) {
	result := findJoinTableInfo("user_roles", nil)
	if result != nil {
		t.Errorf("findJoinTableInfo with nil meta = %v, want nil", result)
	}
}

func TestFindJoinTableInfo_NotJoinTable(t *testing.T) {
	meta := &metadata.Metadata{
		JoinTables: map[string]*metadata.JoinTableMeta{
			"user_roles": {Name: "user_roles"},
		},
		ManyToMany: []*metadata.ManyToManyMeta{},
	}

	// Table not in JoinTables map
	result := findJoinTableInfo("other_table", meta)
	if result != nil {
		t.Errorf("findJoinTableInfo for non-join table = %v, want nil", result)
	}
}

func TestFindJoinTableInfo_Valid(t *testing.T) {
	meta := &metadata.Metadata{
		JoinTables: map[string]*metadata.JoinTableMeta{
			"user_roles": {Name: "user_roles"},
		},
		ManyToMany: []*metadata.ManyToManyMeta{
			{
				Source:    "auth.user",
				Target:    "auth.role",
				JoinTable: "user_roles",
				SourceFK:  "user_id",
				TargetFK:  "role_id",
			},
		},
	}

	result := findJoinTableInfo("user_roles", meta)
	if result == nil {
		t.Fatal("findJoinTableInfo for valid join table should not be nil")
	}

	left := result["left"].(map[string]any)
	if left["schema"] != "auth.user" {
		t.Errorf("left.schema = %v, want auth.user", left["schema"])
	}
	if left["column"] != "user_id" {
		t.Errorf("left.column = %v, want user_id", left["column"])
	}

	right := result["right"].(map[string]any)
	if right["schema"] != "auth.role" {
		t.Errorf("right.schema = %v, want auth.role", right["schema"])
	}
	if right["column"] != "role_id" {
		t.Errorf("right.column = %v, want role_id", right["column"])
	}
}

// ===========================================================================
// findPolymorphicInfo Tests
// ===========================================================================

func TestFindPolymorphicInfo_NoMeta(t *testing.T) {
	result := findPolymorphicInfo("commentable_type", "blog.comment", nil)
	if result != nil {
		t.Errorf("findPolymorphicInfo with nil meta = %v, want nil", result)
	}
}

func TestFindPolymorphicInfo_TypeColumn(t *testing.T) {
	meta := &metadata.Metadata{
		Polymorphic: []*metadata.PolymorphicMeta{
			{
				Table:      "blog.comment",
				Alias:      "commentable",
				TypeColumn: "commentable_type",
				IDColumn:   "commentable_id",
				Targets:    []string{"blog.post", "blog.article"},
			},
		},
	}

	result := findPolymorphicInfo("commentable_type", "blog.comment", meta)
	if result == nil {
		t.Fatal("findPolymorphicInfo for type column should not be nil")
	}

	if result["field"] != "commentable" {
		t.Errorf("field = %v, want commentable", result["field"])
	}
	if result["role"] != "type" {
		t.Errorf("role = %v, want type", result["role"])
	}
	targets := result["targets"].([]string)
	if len(targets) != 2 {
		t.Errorf("targets len = %d, want 2", len(targets))
	}
}

func TestFindPolymorphicInfo_IDColumn(t *testing.T) {
	meta := &metadata.Metadata{
		Polymorphic: []*metadata.PolymorphicMeta{
			{
				Table:      "blog.comment",
				Alias:      "commentable",
				TypeColumn: "commentable_type",
				IDColumn:   "commentable_id",
				Targets:    []string{"blog.post"},
			},
		},
	}

	result := findPolymorphicInfo("commentable_id", "blog.comment", meta)
	if result == nil {
		t.Fatal("findPolymorphicInfo for id column should not be nil")
	}

	if result["role"] != "id" {
		t.Errorf("role = %v, want id", result["role"])
	}
}

func TestFindPolymorphicInfo_WrongTable(t *testing.T) {
	meta := &metadata.Metadata{
		Polymorphic: []*metadata.PolymorphicMeta{
			{
				Table:      "blog.comment",
				Alias:      "commentable",
				TypeColumn: "commentable_type",
				IDColumn:   "commentable_id",
				Targets:    []string{"blog.post"},
			},
		},
	}

	result := findPolymorphicInfo("commentable_type", "other.table", meta)
	if result != nil {
		t.Errorf("findPolymorphicInfo for wrong table = %v, want nil", result)
	}
}

// ===========================================================================
// buildRelationships Tests
// ===========================================================================

func TestBuildRelationships_HasMany(t *testing.T) {
	userTable := &ast.TableDef{
		Namespace: "auth",
		Name:      "user",
		Columns: []*ast.ColumnDef{
			{Name: "id", Type: "uuid", PrimaryKey: true},
		},
	}

	postTable := &ast.TableDef{
		Namespace: "blog",
		Name:      "post",
		Columns: []*ast.ColumnDef{
			{Name: "id", Type: "uuid", PrimaryKey: true},
			{Name: "author_id", Type: "uuid", Reference: &ast.Reference{Table: "auth.user", Column: "id"}},
		},
	}

	allTables := []*ast.TableDef{userTable, postTable}

	// Check user has posts relationship
	relationships := buildRelationships(userTable, allTables, nil)

	if len(relationships) == 0 {
		t.Fatal("expected relationships for user table")
	}

	posts, ok := relationships["posts"].(map[string]any)
	if !ok {
		t.Fatal("expected 'posts' relationship")
	}

	if posts["type"] != "hasMany" {
		t.Errorf("type = %v, want hasMany", posts["type"])
	}
	if posts["target"] != "blog.post" {
		t.Errorf("target = %v, want blog.post", posts["target"])
	}
	if posts["foreignKey"] != "author_id" {
		t.Errorf("foreignKey = %v, want author_id", posts["foreignKey"])
	}
}

func TestBuildRelationships_HasOne(t *testing.T) {
	userTable := &ast.TableDef{
		Namespace: "auth",
		Name:      "user",
		Columns: []*ast.ColumnDef{
			{Name: "id", Type: "uuid", PrimaryKey: true},
		},
	}

	profileTable := &ast.TableDef{
		Namespace: "auth",
		Name:      "profile",
		Columns: []*ast.ColumnDef{
			{Name: "id", Type: "uuid", PrimaryKey: true},
			{Name: "user_id", Type: "uuid", Unique: true, Reference: &ast.Reference{Table: "auth.user", Column: "id"}},
		},
	}

	allTables := []*ast.TableDef{userTable, profileTable}

	relationships := buildRelationships(userTable, allTables, nil)

	// Should have hasOne relationship due to unique FK
	user, ok := relationships["user"].(map[string]any)
	if !ok {
		t.Fatal("expected 'user' relationship (hasOne)")
	}

	if user["type"] != "hasOne" {
		t.Errorf("type = %v, want hasOne", user["type"])
	}
}

func TestBuildRelationships_ManyToMany(t *testing.T) {
	userTable := &ast.TableDef{
		Namespace: "auth",
		Name:      "user",
		Columns: []*ast.ColumnDef{
			{Name: "id", Type: "uuid", PrimaryKey: true},
		},
	}

	roleTable := &ast.TableDef{
		Namespace: "auth",
		Name:      "role",
		Columns: []*ast.ColumnDef{
			{Name: "id", Type: "uuid", PrimaryKey: true},
		},
	}

	allTables := []*ast.TableDef{userTable, roleTable}

	meta := &metadata.Metadata{
		ManyToMany: []*metadata.ManyToManyMeta{
			{
				Source:    "auth.user",
				Target:    "auth.role",
				JoinTable: "user_roles",
				SourceFK:  "user_id",
				TargetFK:  "role_id",
			},
		},
	}

	// Check user->role relationship
	userRels := buildRelationships(userTable, allTables, meta)
	roles, ok := userRels["roles"].(map[string]any)
	if !ok {
		t.Fatal("expected 'roles' relationship")
	}

	if roles["type"] != "manyToMany" {
		t.Errorf("type = %v, want manyToMany", roles["type"])
	}
	if roles["target"] != "auth.role" {
		t.Errorf("target = %v, want auth.role", roles["target"])
	}

	through := roles["through"].(map[string]any)
	if through["table"] != "user_roles" {
		t.Errorf("through.table = %v, want user_roles", through["table"])
	}

	// Check role->user relationship (inverse)
	roleRels := buildRelationships(roleTable, allTables, meta)
	users, ok := roleRels["users"].(map[string]any)
	if !ok {
		t.Fatal("expected 'users' relationship")
	}

	if users["type"] != "manyToMany" {
		t.Errorf("type = %v, want manyToMany", users["type"])
	}
}

// ===========================================================================
// getSemanticType Tests
// ===========================================================================

func TestGetSemanticType(t *testing.T) {
	tests := []struct {
		col  *ast.ColumnDef
		want string
	}{
		{&ast.ColumnDef{Name: "id", Type: "uuid"}, "id"},
		{&ast.ColumnDef{Name: "email", Type: "string"}, "email"},
		{&ast.ColumnDef{Name: "username", Type: "string"}, "username"},
		{&ast.ColumnDef{Name: "password", Type: "string"}, "password_hash"},
		{&ast.ColumnDef{Name: "password_hash", Type: "string"}, "password_hash"},
		{&ast.ColumnDef{Name: "created_at", Type: "datetime"}, "created_at"},
		{&ast.ColumnDef{Name: "updated_at", Type: "datetime"}, "updated_at"},
		{&ast.ColumnDef{Name: "deleted_at", Type: "datetime"}, "deleted_at"},
		{&ast.ColumnDef{Name: "created_by", Type: "uuid"}, "created_by"},
		{&ast.ColumnDef{Name: "updated_by", Type: "uuid"}, "updated_by"},
		// By format
		{&ast.ColumnDef{Name: "contact", Type: "string", Format: "email"}, "email"},
		{&ast.ColumnDef{Name: "website", Type: "string", Format: "uri"}, "url"},
		// By type
		{&ast.ColumnDef{Name: "pk", Type: "id"}, "id"},
		{&ast.ColumnDef{Name: "enabled", Type: "boolean", Default: true}, "flag"},
		// Money pattern
		{&ast.ColumnDef{Name: "amount", Type: "decimal", TypeArgs: []any{float64(19), float64(4)}}, "money"},
		// No semantic type
		{&ast.ColumnDef{Name: "custom", Type: "string"}, ""},
	}

	for _, tt := range tests {
		got := getSemanticType(tt.col)
		if got != tt.want {
			t.Errorf("getSemanticType(%s) = %q, want %q", tt.col.Name, got, tt.want)
		}
	}
}

// ===========================================================================
// buildIndexes Tests
// ===========================================================================

func TestBuildIndexes_WithExplicitIndexes(t *testing.T) {
	table := &ast.TableDef{
		Namespace: "test",
		Name:      "item",
		Columns: []*ast.ColumnDef{
			{Name: "id", Type: "uuid", PrimaryKey: true},
			{Name: "email", Type: "string", Unique: true},
			{Name: "category_id", Type: "uuid", Reference: &ast.Reference{Table: "test.category", Column: "id"}},
		},
		Indexes: []*ast.IndexDef{
			{Name: "idx_custom", Columns: []string{"email", "category_id"}, Unique: true},
			{Columns: []string{"category_id"}}, // No name - should be auto-generated
		},
	}

	indexes := buildIndexes(table)

	// Should have: PK, email unique, idx_custom, auto-named, FK index
	if len(indexes) < 3 {
		t.Errorf("expected at least 3 indexes, got %d", len(indexes))
	}

	// Check PK index
	found := false
	for _, idx := range indexes {
		if idx["primary"] == true {
			found = true
			if idx["name"] != "test_item_pkey" {
				t.Errorf("PK index name = %v, want test_item_pkey", idx["name"])
			}
		}
	}
	if !found {
		t.Error("missing primary key index")
	}

	// Check unique column index
	found = false
	for _, idx := range indexes {
		if idx["name"] == "test_item_email_key" {
			found = true
			if idx["unique"] != true {
				t.Error("email index should be unique")
			}
		}
	}
	if !found {
		t.Error("missing email unique index")
	}

	// Check explicit index
	found = false
	for _, idx := range indexes {
		if idx["name"] == "idx_custom" {
			found = true
		}
	}
	if !found {
		t.Error("missing explicit custom index")
	}
}

func TestBuildIndexes_FKNotDuplicated(t *testing.T) {
	table := &ast.TableDef{
		Namespace: "test",
		Name:      "item",
		Columns: []*ast.ColumnDef{
			{Name: "id", Type: "uuid", PrimaryKey: true},
			{Name: "parent_id", Type: "uuid", Unique: true, Reference: &ast.Reference{Table: "test.item", Column: "id"}},
		},
	}

	indexes := buildIndexes(table)

	// FK column is unique, so shouldn't have separate FK index
	fkIndexCount := 0
	for _, idx := range indexes {
		cols := idx["columns"].([]string)
		if len(cols) == 1 && cols[0] == "parent_id" {
			fkIndexCount++
		}
	}

	if fkIndexCount > 1 {
		t.Errorf("parent_id should only have one index (unique), got %d", fkIndexCount)
	}
}
