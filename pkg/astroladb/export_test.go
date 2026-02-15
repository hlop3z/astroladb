package astroladb

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/hlop3z/astroladb/internal/ast"
	"github.com/hlop3z/astroladb/internal/strutil"
)

// testTable creates a sample table for testing exports.
func testTable() *ast.TableDef {
	minAge := 0.0
	maxAge := 150.0
	return &ast.TableDef{
		Namespace: "auth",
		Name:      "user",
		Columns: []*ast.ColumnDef{
			{Name: "id", Type: "uuid", PrimaryKey: true},
			{Name: "email", Type: "string", TypeArgs: []any{255}, Format: "email", Unique: true},
			{Name: "username", Type: "string", TypeArgs: []any{50}},
			{Name: "password", Type: "string", TypeArgs: []any{255}},
			{Name: "age", Type: "integer", Nullable: true, Min: &minAge, Max: &maxAge},
			{Name: "balance", Type: "decimal", TypeArgs: []any{19, 4}},
			{Name: "is_active", Type: "boolean", Default: true},
			{Name: "status", Type: "enum", TypeArgs: []any{[]string{"active", "pending", "suspended"}}},
			{Name: "metadata", Type: "json", Nullable: true},
			{Name: "avatar", Type: "base64", Nullable: true},
			{Name: "birth_date", Type: "date", Nullable: true},
			{Name: "login_time", Type: "time", Nullable: true},
			{Name: "created_at", Type: "datetime", Default: &ast.SQLExpr{Expr: "NOW()"}},
			{Name: "updated_at", Type: "datetime", Default: &ast.SQLExpr{Expr: "NOW()"}},
		},
		Docs: "User account table",
	}
}

// testTableWithFK creates a table with foreign key for testing.
func testTableWithFK() *ast.TableDef {
	return &ast.TableDef{
		Namespace: "blog",
		Name:      "post",
		Columns: []*ast.ColumnDef{
			{Name: "id", Type: "uuid", PrimaryKey: true},
			{Name: "title", Type: "string", TypeArgs: []any{200}},
			{Name: "body", Type: "text"},
			{Name: "author_id", Type: "uuid", Reference: &ast.Reference{Table: "auth.user", Column: "id"}},
			{Name: "status", Type: "enum", TypeArgs: []any{[]string{"draft", "published"}}},
			{Name: "created_at", Type: "datetime"},
		},
	}
}

// ===========================================================================
// OpenAPI Export Tests
// ===========================================================================

func TestExportOpenAPI_BasicStructure(t *testing.T) {
	tables := []*ast.TableDef{testTable()}
	cfg := &exportContext{ExportConfig: &ExportConfig{}}

	data, err := exportOpenAPI(tables, cfg)
	if err != nil {
		t.Fatalf("exportOpenAPI failed: %v", err)
	}

	var spec map[string]any
	if err := json.Unmarshal(data, &spec); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	// Check OpenAPI version
	if spec["openapi"] != "3.0.3" {
		t.Errorf("openapi version = %v, want 3.0.3", spec["openapi"])
	}

	// Check info section
	info, ok := spec["info"].(map[string]any)
	if !ok {
		t.Fatal("missing info section")
	}
	if info["title"] == nil {
		t.Error("missing info.title")
	}

	// Check paths exist
	if spec["paths"] == nil {
		t.Error("missing paths section")
	}

	// Check components/schemas exist
	components, ok := spec["components"].(map[string]any)
	if !ok {
		t.Fatal("missing components section")
	}
	if components["schemas"] == nil {
		t.Error("missing components/schemas")
	}
}

func TestExportOpenAPI_SchemaProperties(t *testing.T) {
	tables := []*ast.TableDef{testTable()}
	cfg := &exportContext{ExportConfig: &ExportConfig{}}

	data, err := exportOpenAPI(tables, cfg)
	if err != nil {
		t.Fatalf("exportOpenAPI failed: %v", err)
	}

	var spec map[string]any
	json.Unmarshal(data, &spec)

	components := spec["components"].(map[string]any)
	schemas := components["schemas"].(map[string]any)
	userSchema := schemas["AuthUser"].(map[string]any)
	props := userSchema["properties"].(map[string]any)

	// Test string type with maxLength
	email := props["email"].(map[string]any)
	if email["type"] != "string" {
		t.Errorf("email type = %v, want string", email["type"])
	}
	if email["maxLength"] != float64(255) {
		t.Errorf("email maxLength = %v, want 255", email["maxLength"])
	}
	if email["format"] != "email" {
		t.Errorf("email format = %v, want email", email["format"])
	}

	// Test integer type
	age := props["age"].(map[string]any)
	if age["type"] != "integer" {
		t.Errorf("age type = %v, want integer", age["type"])
	}
	if age["nullable"] != true {
		t.Errorf("age nullable = %v, want true", age["nullable"])
	}

	// Test boolean type
	isActive := props["is_active"].(map[string]any)
	if isActive["type"] != "boolean" {
		t.Errorf("is_active type = %v, want boolean", isActive["type"])
	}

	// Test enum type
	status := props["status"].(map[string]any)
	if status["type"] != "string" {
		t.Errorf("status type = %v, want string", status["type"])
	}
	enumVals := status["enum"].([]any)
	if len(enumVals) != 3 {
		t.Errorf("status enum length = %d, want 3", len(enumVals))
	}
	if enumVals[0] != "active" {
		t.Errorf("status enum[0] = %v, want active", enumVals[0])
	}
}

func TestExportOpenAPI_XDBExtension(t *testing.T) {
	tables := []*ast.TableDef{testTable()}
	cfg := &exportContext{ExportConfig: &ExportConfig{}}

	data, err := exportOpenAPI(tables, cfg)
	if err != nil {
		t.Fatalf("exportOpenAPI failed: %v", err)
	}

	var spec map[string]any
	json.Unmarshal(data, &spec)

	components := spec["components"].(map[string]any)
	schemas := components["schemas"].(map[string]any)
	userSchema := schemas["AuthUser"].(map[string]any)

	// Check x-db extension
	xdb, ok := userSchema["x-db"].(map[string]any)
	if !ok {
		t.Fatal("missing x-db extension")
	}

	if xdb["table"] != "auth_user" {
		t.Errorf("x-db.table = %v, want auth_user", xdb["table"])
	}
	if xdb["namespace"] != "auth" {
		t.Errorf("x-db.namespace = %v, want auth", xdb["namespace"])
	}
}

func TestExportOpenAPI_TypeArgs(t *testing.T) {
	tables := []*ast.TableDef{testTable()}
	cfg := &exportContext{ExportConfig: &ExportConfig{}}

	data, err := exportOpenAPI(tables, cfg)
	if err != nil {
		t.Fatalf("exportOpenAPI failed: %v", err)
	}

	var spec map[string]any
	json.Unmarshal(data, &spec)

	components := spec["components"].(map[string]any)
	schemas := components["schemas"].(map[string]any)
	userSchema := schemas["AuthUser"].(map[string]any)
	properties := userSchema["properties"].(map[string]any)

	// Test string type_args
	emailProp := properties["email"].(map[string]any)
	emailXDB := emailProp["x-db"].(map[string]any)
	emailTypeArgs := emailXDB["type_args"].([]any)
	if len(emailTypeArgs) != 1 {
		t.Fatalf("email type_args length = %d, want 1", len(emailTypeArgs))
	}
	if emailTypeArgs[0] != float64(255) {
		t.Errorf("email type_args[0] = %v, want 255", emailTypeArgs[0])
	}

	// Test decimal type_args
	balanceProp := properties["balance"].(map[string]any)
	balanceXDB := balanceProp["x-db"].(map[string]any)
	balanceTypeArgs := balanceXDB["type_args"].([]any)
	if len(balanceTypeArgs) != 2 {
		t.Fatalf("balance type_args length = %d, want 2", len(balanceTypeArgs))
	}
	if balanceTypeArgs[0] != float64(19) {
		t.Errorf("balance type_args[0] = %v, want 19", balanceTypeArgs[0])
	}
	if balanceTypeArgs[1] != float64(4) {
		t.Errorf("balance type_args[1] = %v, want 4", balanceTypeArgs[1])
	}

	// Test enum type_args
	statusProp := properties["status"].(map[string]any)
	statusXDB := statusProp["x-db"].(map[string]any)
	statusTypeArgs := statusXDB["type_args"].([]any)
	if len(statusTypeArgs) != 1 {
		t.Fatalf("status type_args length = %d, want 1", len(statusTypeArgs))
	}
	enumValues := statusTypeArgs[0].([]any)
	if len(enumValues) != 3 {
		t.Fatalf("enum values length = %d, want 3", len(enumValues))
	}
	if enumValues[0] != "active" || enumValues[1] != "pending" || enumValues[2] != "suspended" {
		t.Errorf("enum values = %v, want [active, pending, suspended]", enumValues)
	}

	// Test that id (no type args) doesn't have type_args field
	idProp := properties["id"].(map[string]any)
	idXDB := idProp["x-db"].(map[string]any)
	if _, exists := idXDB["type_args"]; exists {
		t.Errorf("id should not have type_args field")
	}
}

func TestExportOpenAPI_CompleteForeignKeysAndChecks(t *testing.T) {
	table := &ast.TableDef{
		Namespace: "test",
		Name:      "orders",
		Columns: []*ast.ColumnDef{
			{Name: "id", Type: "uuid", PrimaryKey: true},
			{Name: "user_id", Type: "uuid"},
			{Name: "amount", Type: "decimal", TypeArgs: []any{10, 2}},
		},
		ForeignKeys: []*ast.ForeignKeyDef{
			{
				Name:       "fk_orders_user",
				Columns:    []string{"user_id"},
				RefTable:   "auth_user",
				RefColumns: []string{"id"},
				OnDelete:   "CASCADE",
				OnUpdate:   "NO ACTION",
			},
		},
		Checks: []*ast.CheckDef{
			{
				Name:       "positive_amount",
				Expression: "amount > 0",
			},
		},
	}

	cfg := &exportContext{ExportConfig: &ExportConfig{}}
	data, err := exportOpenAPI([]*ast.TableDef{table}, cfg)
	if err != nil {
		t.Fatalf("exportOpenAPI failed: %v", err)
	}

	var spec map[string]any
	json.Unmarshal(data, &spec)

	// Navigate to TestOrders schema
	components := spec["components"].(map[string]any)
	schemas := components["schemas"].(map[string]any)
	schema := schemas["TestOrders"].(map[string]any)
	xdb := schema["x-db"].(map[string]any)

	// Verify foreign_keys
	fks, ok := xdb["foreign_keys"].([]any)
	if !ok || len(fks) != 1 {
		t.Fatalf("expected 1 foreign key, got %v", fks)
	}
	fk := fks[0].(map[string]any)
	if fk["name"] != "fk_orders_user" {
		t.Errorf("fk name = %v, want fk_orders_user", fk["name"])
	}
	if fk["on_delete"] != "CASCADE" {
		t.Errorf("fk on_delete = %v, want CASCADE", fk["on_delete"])
	}

	// Verify checks
	checks, ok := xdb["checks"].([]any)
	if !ok || len(checks) != 1 {
		t.Fatalf("expected 1 check, got %v", checks)
	}
	check := checks[0].(map[string]any)
	if check["expression"] != "amount > 0" {
		t.Errorf("check expression = %v, want 'amount > 0'", check["expression"])
	}

	// Verify column_order
	colOrder, ok := xdb["column_order"].([]any)
	if !ok || len(colOrder) != 3 {
		t.Fatalf("expected 3 columns in order, got %v", colOrder)
	}
	if colOrder[0] != "id" || colOrder[1] != "user_id" || colOrder[2] != "amount" {
		t.Errorf("column_order = %v, want [id, user_id, amount]", colOrder)
	}
}

func TestExportOpenAPI_ExplicitColumnFlags(t *testing.T) {
	table := &ast.TableDef{
		Namespace: "test",
		Name:      "users",
		Columns: []*ast.ColumnDef{
			{Name: "id", Type: "uuid", PrimaryKey: true},
			{Name: "email", Type: "string", TypeArgs: []any{255}, Unique: true},
			{Name: "username", Type: "string", TypeArgs: []any{50}},
		},
	}

	cfg := &exportContext{ExportConfig: &ExportConfig{}}
	data, _ := exportOpenAPI([]*ast.TableDef{table}, cfg)

	var spec map[string]any
	json.Unmarshal(data, &spec)

	// Navigate to properties
	components := spec["components"].(map[string]any)
	schemas := components["schemas"].(map[string]any)
	schema := schemas["TestUsers"].(map[string]any)
	properties := schema["properties"].(map[string]any)

	// Verify id has primary_key flag
	idProp := properties["id"].(map[string]any)
	idXDB := idProp["x-db"].(map[string]any)
	if pk, ok := idXDB["primary_key"].(bool); !ok || !pk {
		t.Error("id should have primary_key: true")
	}

	// Verify email has unique flag
	emailProp := properties["email"].(map[string]any)
	emailXDB := emailProp["x-db"].(map[string]any)
	if unique, ok := emailXDB["unique"].(bool); !ok || !unique {
		t.Error("email should have unique: true")
	}

	// Verify username does NOT have unique flag
	usernameProp := properties["username"].(map[string]any)
	usernameXDB := usernameProp["x-db"].(map[string]any)
	if _, exists := usernameXDB["unique"]; exists {
		t.Error("username should not have unique flag")
	}
}

// ===========================================================================
// TypeScript Export Tests
// ===========================================================================

func TestExportTypeScript_BasicInterface(t *testing.T) {
	tables := []*ast.TableDef{testTable()}
	cfg := &exportContext{ExportConfig: &ExportConfig{}}

	data, err := exportTypeScript(tables, cfg)
	if err != nil {
		t.Fatalf("exportTypeScript failed: %v", err)
	}

	output := string(data)

	// Check header
	if !strings.Contains(output, "// Auto-generated TypeScript types") {
		t.Error("missing header comment")
	}

	// Check interface declaration
	if !strings.Contains(output, "export interface AuthUser {") {
		t.Error("missing interface declaration")
	}

	// Check required field (no ?)
	if !strings.Contains(output, "id: string;") {
		t.Error("missing required id field")
	}
	if !strings.Contains(output, "email: string;") {
		t.Error("missing required email field")
	}

	// Check optional field (has ?)
	if !strings.Contains(output, "age?: number;") {
		t.Error("missing optional age field")
	}
	if !strings.Contains(output, "metadata?: Record<string, unknown>;") {
		t.Error("missing optional metadata field")
	}
}

func TestExportTypeScript_EnumUnionType(t *testing.T) {
	tables := []*ast.TableDef{testTable()}
	cfg := &exportContext{ExportConfig: &ExportConfig{}}

	data, err := exportTypeScript(tables, cfg)
	if err != nil {
		t.Fatalf("exportTypeScript failed: %v", err)
	}

	output := string(data)

	// Check enum as union type
	if !strings.Contains(output, "status: 'active' | 'pending' | 'suspended';") {
		t.Errorf("missing enum union type, got:\n%s", output)
	}
}

func TestExportTypeScript_AllTypes(t *testing.T) {
	tables := []*ast.TableDef{testTable()}
	cfg := &exportContext{ExportConfig: &ExportConfig{}}

	data, err := exportTypeScript(tables, cfg)
	if err != nil {
		t.Fatalf("exportTypeScript failed: %v", err)
	}

	output := string(data)

	tests := []struct {
		field    string
		expected string
	}{
		{"id", "id: string;"},
		{"email", "email: string;"},
		{"age", "age?: number;"},
		{"balance", "balance: string;"}, // decimal as string
		{"is_active", "is_active: boolean;"},
		{"metadata", "metadata?: Record<string, unknown>;"},
		{"avatar", "avatar?: string;"}, // base64 as string
		{"birth_date", "birth_date?: string;"},
		{"login_time", "login_time?: string;"},
		{"created_at", "created_at: string;"},
		{"updated_at", "updated_at: string;"},
	}

	for _, tt := range tests {
		if !strings.Contains(output, tt.expected) {
			t.Errorf("field %s: expected %q not found in output", tt.field, tt.expected)
		}
	}
}

// ===========================================================================
// Go Export Tests
// ===========================================================================

func TestExportGo_BasicStruct(t *testing.T) {
	tables := []*ast.TableDef{testTable()}
	cfg := &exportContext{ExportConfig: &ExportConfig{}}

	data, err := exportGo(tables, cfg)
	if err != nil {
		t.Fatalf("exportGo failed: %v", err)
	}

	output := string(data)

	// Check package declaration
	if !strings.Contains(output, "package types") {
		t.Error("missing package declaration")
	}

	// Check struct declaration
	if !strings.Contains(output, "type AuthUser struct {") {
		t.Error("missing struct declaration")
	}

	// Check JSON tags
	if !strings.Contains(output, "`json:\"id\"`") {
		t.Error("missing json tag for id")
	}
	if !strings.Contains(output, "`json:\"email\"`") {
		t.Error("missing json tag for email")
	}
}

func TestExportGo_NullableFields(t *testing.T) {
	tables := []*ast.TableDef{testTable()}
	cfg := &exportContext{ExportConfig: &ExportConfig{}}

	data, err := exportGo(tables, cfg)
	if err != nil {
		t.Fatalf("exportGo failed: %v", err)
	}

	output := string(data)

	// Check nullable field with pointer type
	if !strings.Contains(output, "Age *int32 `json:\"age,omitempty\"`") {
		t.Errorf("missing nullable age field with pointer, got:\n%s", output)
	}

	// Check non-nullable field without pointer
	if !strings.Contains(output, "Id string `json:\"id\"`") {
		t.Error("id should not be pointer type")
	}
}

func TestExportGo_AllTypes(t *testing.T) {
	tables := []*ast.TableDef{testTable()}
	cfg := &exportContext{ExportConfig: &ExportConfig{}}

	data, err := exportGo(tables, cfg)
	if err != nil {
		t.Fatalf("exportGo failed: %v", err)
	}

	output := string(data)

	tests := []struct {
		field    string
		contains string
	}{
		{"id", "Id string"},
		{"email", "Email string"},
		{"age", "Age *int32"},
		{"balance", "Balance string"}, // decimal as string
		{"is_active", "IsActive bool"},
		{"status", "Status string"},   // enum as string
		{"metadata", "Metadata *any"}, // json as *any (optional)
		{"avatar", "Avatar *string"},  // base64 as string
		{"birth_date", "BirthDate *string"},
		{"created_at", "CreatedAt string"},
	}

	for _, tt := range tests {
		if !strings.Contains(output, tt.contains) {
			t.Errorf("field %s: expected %q not found in output:\n%s", tt.field, tt.contains, output)
		}
	}
}

// ===========================================================================
// Python Export Tests
// ===========================================================================

func TestExportPython_BasicDataclass(t *testing.T) {
	tables := []*ast.TableDef{testTable()}
	cfg := &exportContext{ExportConfig: &ExportConfig{}}

	data, err := exportPython(tables, cfg)
	if err != nil {
		t.Fatalf("exportPython failed: %v", err)
	}

	output := string(data)

	// Check imports
	if !strings.Contains(output, "from dataclasses import dataclass") {
		t.Error("missing dataclass import")
	}
	if !strings.Contains(output, "from typing import Optional, Any") {
		t.Error("missing typing import")
	}
	if !strings.Contains(output, "from enum import Enum") {
		t.Error("missing Enum import")
	}

	// Check dataclass decorator
	if !strings.Contains(output, "@dataclass") {
		t.Error("missing @dataclass decorator")
	}

	// Check class declaration
	if !strings.Contains(output, "class AuthUser:") {
		t.Error("missing class declaration")
	}
}

func TestExportPython_EnumClass(t *testing.T) {
	tables := []*ast.TableDef{testTable()}
	cfg := &exportContext{ExportConfig: &ExportConfig{}}

	data, err := exportPython(tables, cfg)
	if err != nil {
		t.Fatalf("exportPython failed: %v", err)
	}

	output := string(data)

	// Check enum class
	if !strings.Contains(output, "class AuthUserStatus(str, Enum):") {
		t.Error("missing enum class declaration")
	}
	if !strings.Contains(output, "ACTIVE = \"active\"") {
		t.Error("missing ACTIVE enum value")
	}
	if !strings.Contains(output, "PENDING = \"pending\"") {
		t.Error("missing PENDING enum value")
	}
	if !strings.Contains(output, "SUSPENDED = \"suspended\"") {
		t.Error("missing SUSPENDED enum value")
	}
}

func TestExportPython_OptionalFields(t *testing.T) {
	tables := []*ast.TableDef{testTable()}
	cfg := &exportContext{ExportConfig: &ExportConfig{}}

	data, err := exportPython(tables, cfg)
	if err != nil {
		t.Fatalf("exportPython failed: %v", err)
	}

	output := string(data)

	// Check optional field with default None
	if !strings.Contains(output, "age: Optional[int] = None") {
		t.Error("missing optional age field")
	}
	if !strings.Contains(output, "metadata: Optional[Any] = None") {
		t.Error("missing optional metadata field")
	}

	// Check required field without default
	if !strings.Contains(output, "id: str\n") {
		t.Error("id should not have default")
	}
}

func TestExportPython_AllTypes(t *testing.T) {
	tables := []*ast.TableDef{testTable()}
	cfg := &exportContext{ExportConfig: &ExportConfig{}}

	data, err := exportPython(tables, cfg)
	if err != nil {
		t.Fatalf("exportPython failed: %v", err)
	}

	output := string(data)

	tests := []struct {
		field    string
		contains string
	}{
		{"id", "id: str"},
		{"email", "email: str"},
		{"age", "age: Optional[int]"},
		{"balance", "balance: str"}, // decimal as str
		{"is_active", "is_active: bool"},
		{"status", "status: AuthUserStatus"}, // enum type
		{"metadata", "metadata: Optional[Any]"},
		{"avatar", "avatar: Optional[bytes]"}, // base64 as bytes
		{"birth_date", "birth_date: Optional[date]"},
		{"login_time", "login_time: Optional[time]"},
		{"created_at", "created_at: datetime"},
	}

	for _, tt := range tests {
		if !strings.Contains(output, tt.contains) {
			t.Errorf("field %s: expected %q not found in output:\n%s", tt.field, tt.contains, output)
		}
	}
}

// ===========================================================================
// Rust Export Tests
// ===========================================================================

func TestExportRust_BasicStruct(t *testing.T) {
	tables := []*ast.TableDef{testTable()}
	cfg := &exportContext{ExportConfig: &ExportConfig{}}

	data, err := exportRust(tables, cfg)
	if err != nil {
		t.Fatalf("exportRust failed: %v", err)
	}

	output := string(data)

	// Check imports - chrono NOT included by default
	if !strings.Contains(output, "use serde::{Deserialize, Serialize};") {
		t.Error("missing serde import")
	}
	if strings.Contains(output, "use chrono::") {
		t.Error("chrono import should not be present by default")
	}

	// Check derive macros
	if !strings.Contains(output, "#[derive(Debug, Clone, Serialize, Deserialize)]") {
		t.Error("missing derive macros")
	}

	// Check struct declaration (no rename_all needed - fields already snake_case)
	if !strings.Contains(output, "pub struct AuthUser {") {
		t.Error("missing struct declaration")
	}
}

func TestExportRust_EnumWithSerde(t *testing.T) {
	tables := []*ast.TableDef{testTable()}
	cfg := &exportContext{ExportConfig: &ExportConfig{}}

	data, err := exportRust(tables, cfg)
	if err != nil {
		t.Fatalf("exportRust failed: %v", err)
	}

	output := string(data)

	// Check enum declaration
	if !strings.Contains(output, "pub enum AuthUserStatus {") {
		t.Error("missing enum declaration")
	}

	// Check rename_all on enum (not individual renames)
	// Find the enum block and check it has rename_all
	enumStart := strings.Index(output, "pub enum AuthUserStatus")
	if enumStart == -1 {
		t.Fatal("enum not found")
	}

	// Check that rename_all appears before the enum
	enumBlock := output[:enumStart]
	lastDerivePos := strings.LastIndex(enumBlock, "#[derive(")
	if lastDerivePos == -1 {
		t.Fatal("derive not found before enum")
	}
	betweenDeriveAndEnum := output[lastDerivePos:enumStart]
	if !strings.Contains(betweenDeriveAndEnum, "#[serde(rename_all = \"snake_case\")]") {
		t.Errorf("missing rename_all attribute on enum, got: %s", betweenDeriveAndEnum)
	}

	// Check variants exist without individual rename attributes
	if !strings.Contains(output, "    Active,") {
		t.Error("missing Active variant")
	}
	if !strings.Contains(output, "    Pending,") {
		t.Error("missing Pending variant")
	}
	if !strings.Contains(output, "    Suspended,") {
		t.Error("missing Suspended variant")
	}
}

func TestExportRust_OptionTypes(t *testing.T) {
	tables := []*ast.TableDef{testTable()}
	cfg := &exportContext{ExportConfig: &ExportConfig{}}

	data, err := exportRust(tables, cfg)
	if err != nil {
		t.Fatalf("exportRust failed: %v", err)
	}

	output := string(data)

	// Check Option types for nullable fields
	if !strings.Contains(output, "pub age: Option<i32>,") {
		t.Error("missing Option<i32> for age")
	}
	if !strings.Contains(output, "pub metadata: Option<serde_json::Value>,") {
		t.Error("missing Option for metadata")
	}

	// Check non-Option types for required fields
	if !strings.Contains(output, "pub id: String,") {
		t.Error("id should not be Option")
	}
}

func TestExportRust_ReservedKeywords(t *testing.T) {
	// Create table with reserved keyword column
	table := &ast.TableDef{
		Namespace: "test",
		Name:      "item",
		Columns: []*ast.ColumnDef{
			{Name: "id", Type: "uuid", PrimaryKey: true},
			{Name: "type", Type: "string", TypeArgs: []any{50}}, // Reserved keyword
			{Name: "ref", Type: "string", TypeArgs: []any{50}},  // Reserved keyword
		},
	}

	cfg := &exportContext{ExportConfig: &ExportConfig{}}
	data, err := exportRust([]*ast.TableDef{table}, cfg)
	if err != nil {
		t.Fatalf("exportRust failed: %v", err)
	}

	output := string(data)

	// Check that reserved keywords are handled with serde rename
	if !strings.Contains(output, "#[serde(rename = \"type\")]") {
		t.Error("missing serde rename for type keyword")
	}
	if !strings.Contains(output, "pub type_: String,") {
		t.Error("missing type_ field (renamed from type)")
	}
	if !strings.Contains(output, "#[serde(rename = \"ref\")]") {
		t.Error("missing serde rename for ref keyword")
	}
	if !strings.Contains(output, "pub ref_: String,") {
		t.Error("missing ref_ field (renamed from ref)")
	}
}

func TestExportRust_AllTypes(t *testing.T) {
	tables := []*ast.TableDef{testTable()}
	cfg := &exportContext{ExportConfig: &ExportConfig{}} // Default: no chrono

	data, err := exportRust(tables, cfg)
	if err != nil {
		t.Fatalf("exportRust failed: %v", err)
	}

	output := string(data)

	tests := []struct {
		field    string
		contains string
	}{
		{"id", "pub id: String,"},
		{"email", "pub email: String,"},
		{"age", "pub age: Option<i32>,"},
		{"balance", "pub balance: String,"}, // decimal as String
		{"is_active", "pub is_active: bool,"},
		{"status", "pub status: AuthUserStatus,"}, // enum type
		{"metadata", "pub metadata: Option<serde_json::Value>,"},
		{"avatar", "pub avatar: Option<Vec<u8>>,"},        // base64 as Vec<u8>
		{"birth_date", "pub birth_date: Option<String>,"}, // String by default
		{"login_time", "pub login_time: Option<String>,"}, // String by default
		{"created_at", "pub created_at: String,"},         // String by default
	}

	for _, tt := range tests {
		if !strings.Contains(output, tt.contains) {
			t.Errorf("field %s: expected %q not found in output:\n%s", tt.field, tt.contains, output)
		}
	}
}

func TestExportRust_WithChrono(t *testing.T) {
	tables := []*ast.TableDef{testTable()}
	cfg := &exportContext{ExportConfig: &ExportConfig{UseChrono: true}}

	data, err := exportRust(tables, cfg)
	if err != nil {
		t.Fatalf("exportRust failed: %v", err)
	}

	output := string(data)

	// Check chrono import
	if !strings.Contains(output, "use chrono::{DateTime, NaiveDate, NaiveTime, Utc};") {
		t.Error("missing chrono import when UseChrono=true")
	}

	// Check chrono types
	tests := []struct {
		field    string
		contains string
	}{
		{"birth_date", "pub birth_date: Option<NaiveDate>,"},
		{"login_time", "pub login_time: Option<NaiveTime>,"},
		{"created_at", "pub created_at: DateTime<Utc>,"},
	}

	for _, tt := range tests {
		if !strings.Contains(output, tt.contains) {
			t.Errorf("field %s: expected %q not found in output:\n%s", tt.field, tt.contains, output)
		}
	}
}

// ===========================================================================
// Cross-Format Consistency Tests
// ===========================================================================

func TestExport_EnumValuesConsistent(t *testing.T) {
	tables := []*ast.TableDef{testTable()}
	cfg := &exportContext{ExportConfig: &ExportConfig{}}

	// Get all exports
	openapi, _ := exportOpenAPI(tables, cfg)
	ts, _ := exportTypeScript(tables, cfg)
	python, _ := exportPython(tables, cfg)
	rust, _ := exportRust(tables, cfg)

	// All should contain the same enum values
	enumValues := []string{"active", "pending", "suspended"}

	// Check OpenAPI
	var spec map[string]any
	json.Unmarshal(openapi, &spec)
	components := spec["components"].(map[string]any)
	schemas := components["schemas"].(map[string]any)
	userSchema := schemas["AuthUser"].(map[string]any)
	props := userSchema["properties"].(map[string]any)
	status := props["status"].(map[string]any)
	enumVals := status["enum"].([]any)
	for i, v := range enumValues {
		if enumVals[i] != v {
			t.Errorf("OpenAPI enum[%d] = %v, want %s", i, enumVals[i], v)
		}
	}

	// Check TypeScript
	tsOutput := string(ts)
	if !strings.Contains(tsOutput, "'active' | 'pending' | 'suspended'") {
		t.Error("TypeScript enum values inconsistent")
	}

	// Check Python
	pyOutput := string(python)
	for _, v := range enumValues {
		if !strings.Contains(pyOutput, strings.ToUpper(v)+" = \""+v+"\"") {
			t.Errorf("Python missing enum value: %s", v)
		}
	}

	// Check Rust - uses rename_all so variants are PascalCase
	rsOutput := string(rust)
	// Check enum has rename_all attribute
	if !strings.Contains(rsOutput, "#[serde(rename_all = \"snake_case\")]") {
		t.Error("Rust missing rename_all on enum")
	}
	// Check PascalCase variants (snake_case happens via rename_all)
	for _, v := range enumValues {
		pascalV := strings.ToUpper(v[:1]) + v[1:]
		if !strings.Contains(rsOutput, pascalV+",") {
			t.Errorf("Rust missing enum variant: %s", pascalV)
		}
	}
}

func TestExport_NullableConsistent(t *testing.T) {
	tables := []*ast.TableDef{testTable()}
	cfg := &exportContext{ExportConfig: &ExportConfig{}}

	// Get all exports
	openapi, _ := exportOpenAPI(tables, cfg)
	ts, _ := exportTypeScript(tables, cfg)
	go_, _ := exportGo(tables, cfg)
	python, _ := exportPython(tables, cfg)
	rust, _ := exportRust(tables, cfg)

	// age is nullable in all formats
	var spec map[string]any
	json.Unmarshal(openapi, &spec)
	components := spec["components"].(map[string]any)
	schemas := components["schemas"].(map[string]any)
	userSchema := schemas["AuthUser"].(map[string]any)
	props := userSchema["properties"].(map[string]any)
	age := props["age"].(map[string]any)
	if age["nullable"] != true {
		t.Error("OpenAPI: age should be nullable")
	}

	if !strings.Contains(string(ts), "age?: number;") {
		t.Error("TypeScript: age should be optional")
	}

	if !strings.Contains(string(go_), "Age *int32") {
		t.Error("Go: age should be pointer")
	}

	if !strings.Contains(string(python), "age: Optional[int]") {
		t.Error("Python: age should be Optional")
	}

	if !strings.Contains(string(rust), "age: Option<i32>") {
		t.Error("Rust: age should be Option")
	}
}

// ===========================================================================
// Helper Function Tests
// ===========================================================================

func TestGetEnumValues(t *testing.T) {
	tests := []struct {
		name     string
		typeArgs []any
		want     []string
	}{
		{
			name:     "string slice",
			typeArgs: []any{[]string{"a", "b", "c"}},
			want:     []string{"a", "b", "c"},
		},
		{
			name:     "any slice with strings",
			typeArgs: []any{[]any{"x", "y", "z"}},
			want:     []string{"x", "y", "z"},
		},
		{
			name:     "empty",
			typeArgs: []any{},
			want:     nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			col := &ast.ColumnDef{Type: "enum", TypeArgs: tt.typeArgs}
			got := col.EnumValues()

			if len(got) != len(tt.want) {
				t.Errorf("EnumValues() len = %d, want %d", len(got), len(tt.want))
				return
			}

			for i, v := range got {
				if v != tt.want[i] {
					t.Errorf("EnumValues()[%d] = %s, want %s", i, v, tt.want[i])
				}
			}
		})
	}
}

func TestIsRustKeyword(t *testing.T) {
	keywords := []string{"type", "ref", "fn", "let", "mut", "pub", "struct", "enum", "impl", "trait", "async", "await"}
	nonKeywords := []string{"name", "id", "email", "status", "user", "post"}

	for _, kw := range keywords {
		if !isRustKeyword(kw) {
			t.Errorf("isRustKeyword(%q) = false, want true", kw)
		}
	}

	for _, nk := range nonKeywords {
		if isRustKeyword(nk) {
			t.Errorf("isRustKeyword(%q) = true, want false", nk)
		}
	}
}

func TestPascalCase(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"user", "User"},
		{"auth_user", "AuthUser"},
		{"blog_post_comment", "BlogPostComment"},
		{"id", "Id"},
		{"", ""},
	}

	for _, tt := range tests {
		got := strutil.ToPascalCase(tt.input)
		if got != tt.want {
			t.Errorf("strutil.ToPascalCase(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

// ===========================================================================
// OpenAPI x-db: Computed/Virtual Storage Tests
// ===========================================================================

func TestBuildPropertyXDB_ComputedStored(t *testing.T) {
	table := &ast.TableDef{Name: "items"}
	col := &ast.ColumnDef{
		Name: "total",
		Type: "integer",
		Computed: map[string]any{
			"fn":   "add",
			"args": []any{map[string]any{"col": "a"}, map[string]any{"col": "b"}},
		},
	}
	xdb := buildPropertyXDB(col, table, nil)

	if xdb["virtual"] != true {
		t.Error("expected virtual=true for computed stored column")
	}
	if xdb["storage"] != "stored" {
		t.Errorf("storage = %v, want \"stored\"", xdb["storage"])
	}
	if xdb["computed"] == nil {
		t.Error("expected computed expression to be present")
	}
}

func TestBuildPropertyXDB_ComputedVirtual(t *testing.T) {
	table := &ast.TableDef{Name: "items"}
	col := &ast.ColumnDef{
		Name: "total",
		Type: "integer",
		Computed: map[string]any{
			"fn":   "add",
			"args": []any{map[string]any{"col": "a"}, map[string]any{"col": "b"}},
		},
		Virtual: true,
	}
	xdb := buildPropertyXDB(col, table, nil)

	if xdb["virtual"] != true {
		t.Error("expected virtual=true for computed virtual column")
	}
	if xdb["storage"] != "virtual" {
		t.Errorf("storage = %v, want \"virtual\"", xdb["storage"])
	}
	if xdb["computed"] == nil {
		t.Error("expected computed expression to be present")
	}
}

func TestBuildPropertyXDB_AppOnly(t *testing.T) {
	table := &ast.TableDef{Name: "items"}
	col := &ast.ColumnDef{
		Name:    "display_label",
		Type:    "text",
		Virtual: true,
	}
	xdb := buildPropertyXDB(col, table, nil)

	if xdb["virtual"] != true {
		t.Error("expected virtual=true for app-only column")
	}
	if xdb["storage"] != "app_only" {
		t.Errorf("storage = %v, want \"app_only\"", xdb["storage"])
	}
	if xdb["computed"] != nil {
		t.Error("expected no computed expression for app-only column")
	}
}

func TestBuildPropertyXDB_RegularColumn(t *testing.T) {
	table := &ast.TableDef{Name: "items"}
	col := &ast.ColumnDef{
		Name: "name",
		Type: "text",
	}
	xdb := buildPropertyXDB(col, table, nil)

	if _, ok := xdb["virtual"]; ok {
		t.Error("regular column should not have virtual key")
	}
	if _, ok := xdb["storage"]; ok {
		t.Error("regular column should not have storage key")
	}
	if _, ok := xdb["computed"]; ok {
		t.Error("regular column should not have computed key")
	}
}

func TestCamelCase(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"set_null", "setNull"},
		{"cascade", "cascade"},
		{"no_action", "noAction"},
		{"", ""},
	}

	for _, tt := range tests {
		got := strutil.ToCamelCase(tt.input)
		if got != tt.want {
			t.Errorf("strutil.ToCamelCase(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
