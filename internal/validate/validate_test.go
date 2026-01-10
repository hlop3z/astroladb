package validate

import (
	"strings"
	"testing"
)

// -----------------------------------------------------------------------------
// Identifier Tests
// -----------------------------------------------------------------------------

func TestIdentifier(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		errCode Code
	}{
		// Valid identifiers (note: "user" is a SQL reserved word)
		{"simple", "account", false, ""},
		{"with_underscore", "user_name", false, ""},
		{"starts_underscore", "_account", false, ""},
		{"with_numbers", "account123", false, ""},
		{"mixed", "account_123_name", false, ""},
		{"uppercase", "Account", false, ""},
		{"all_caps", "ACCOUNT_NAME", false, ""},

		// Invalid: empty
		{"empty", "", true, ErrInvalidIdentifier},

		// Invalid: too long (>63 chars)
		{"too_long", strings.Repeat("a", 64), true, ErrInvalidIdentifier},

		// Invalid: starts with number
		{"starts_with_number", "1user", true, ErrInvalidIdentifier},

		// Invalid: special characters
		{"has_dash", "user-name", true, ErrInvalidIdentifier},
		{"has_space", "user name", true, ErrInvalidIdentifier},
		{"has_dot", "user.name", true, ErrInvalidIdentifier},
		{"has_at", "user@name", true, ErrInvalidIdentifier},

		// Invalid: reserved words
		{"reserved_select", "select", true, ErrReservedWord},
		{"reserved_from", "from", true, ErrReservedWord},
		{"reserved_where", "where", true, ErrReservedWord},
		{"reserved_table", "table", true, ErrReservedWord},
		{"reserved_index", "index", true, ErrReservedWord},
		{"reserved_user", "user", true, ErrReservedWord},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Identifier(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Identifier(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if err != nil && tt.errCode != "" {
				vErr, ok := err.(*Error)
				if !ok {
					t.Errorf("expected *Error, got %T", err)
					return
				}
				if vErr.Code != tt.errCode {
					t.Errorf("error code = %v, want %v", vErr.Code, tt.errCode)
				}
			}
		})
	}
}

// -----------------------------------------------------------------------------
// SnakeCase Tests
// -----------------------------------------------------------------------------

func TestSnakeCaseValidation(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantErr     bool
		errCode     Code
		wantSuggest string // expected suggestion in error context
	}{
		// Valid snake_case (note: "user" is a SQL reserved word)
		{"simple", "account", false, "", ""},
		{"with_underscore", "user_name", false, "", ""},
		{"with_numbers", "account123", false, "", ""},
		{"number_middle", "user_1_name", false, "", ""},

		// Invalid: empty
		{"empty", "", true, ErrInvalidSnakeCase, ""},

		// Invalid: uppercase (should suggest snake_case)
		{"camelCase", "userName", true, ErrInvalidSnakeCase, "user_name"},
		{"PascalCase", "UserName", true, ErrInvalidSnakeCase, "user_name"},
		{"has_Upper", "user_Name", true, ErrInvalidSnakeCase, "user_name"},

		// Invalid: starts with underscore
		{"starts_underscore", "_user", true, ErrInvalidSnakeCase, "user"},

		// Invalid: ends with underscore
		{"ends_underscore", "user_", true, ErrInvalidSnakeCase, "user"},

		// Invalid: consecutive underscores
		{"consecutive_underscores", "user__name", true, ErrInvalidSnakeCase, "user_name"},

		// Invalid: starts with number
		{"starts_number", "1user", true, ErrInvalidSnakeCase, ""},

		// Invalid: dashes (should suggest underscores)
		{"has_dash", "user-name", true, ErrInvalidSnakeCase, "user_name"},

		// Invalid: reserved word
		{"reserved_select", "select", true, ErrReservedWord, ""},
		{"reserved_from", "from", true, ErrReservedWord, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := SnakeCase(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("SnakeCase(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if err != nil {
				vErr, ok := err.(*Error)
				if !ok {
					t.Errorf("expected *Error, got %T", err)
					return
				}
				if tt.errCode != "" && vErr.Code != tt.errCode {
					t.Errorf("error code = %v, want %v", vErr.Code, tt.errCode)
				}
				if tt.wantSuggest != "" {
					suggestion, hasSuggest := vErr.Context["suggestion"]
					if !hasSuggest {
						t.Errorf("expected suggestion in error context")
					} else if suggestion != tt.wantSuggest {
						t.Errorf("suggestion = %v, want %v", suggestion, tt.wantSuggest)
					}
				}
			}
		})
	}
}

func TestIsSnakeCase(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		// Valid
		{"user", true},
		{"user_name", true},
		{"user_name_field", true},
		{"user1", true},
		{"user_2", true},
		{"a", true},

		// Invalid
		{"", false},
		{"User", false},
		{"userName", false},
		{"_user", false},
		{"user_", false},
		{"user__name", false},
		{"1user", false},
		{"user-name", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := IsSnakeCase(tt.input)
			if got != tt.want {
				t.Errorf("IsSnakeCase(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

// -----------------------------------------------------------------------------
// Reserved Word Tests
// -----------------------------------------------------------------------------

func TestIsReservedWord(t *testing.T) {
	// SQL Standard Keywords
	standardKeywords := []string{
		"select", "from", "where", "insert", "update", "delete",
		"create", "drop", "alter", "table", "index", "constraint",
		"primary", "foreign", "references", "unique", "null", "not",
		"and", "or", "join", "left", "right", "inner", "outer",
		"group", "order", "by", "having", "limit", "offset",
	}

	for _, word := range standardKeywords {
		t.Run(word, func(t *testing.T) {
			if !IsReservedWord(word) {
				t.Errorf("IsReservedWord(%q) = false, want true", word)
			}
			// Case insensitive
			if !IsReservedWord(strings.ToUpper(word)) {
				t.Errorf("IsReservedWord(%q) = false, want true (uppercase)", strings.ToUpper(word))
			}
		})
	}

	// PostgreSQL specific
	pgKeywords := []string{"array", "analyze", "vacuum", "returning", "ilike"}
	for _, word := range pgKeywords {
		t.Run("pg_"+word, func(t *testing.T) {
			if !IsReservedWord(word) {
				t.Errorf("IsReservedWord(%q) = false, want true (PostgreSQL)", word)
			}
		})
	}

	// SQLite specific
	sqliteKeywords := []string{"glob", "pragma", "reindex", "virtual"}
	for _, word := range sqliteKeywords {
		t.Run("sqlite_"+word, func(t *testing.T) {
			if !IsReservedWord(word) {
				t.Errorf("IsReservedWord(%q) = false, want true (SQLite)", word)
			}
		})
	}

	// Type names that should be reserved
	typeKeywords := []string{"boolean", "bool", "date", "enum", "json", "jsonb", "uuid", "serial"}
	for _, word := range typeKeywords {
		t.Run("type_"+word, func(t *testing.T) {
			if !IsReservedWord(word) {
				t.Errorf("IsReservedWord(%q) = false, want true (type name)", word)
			}
		})
	}

	// Non-reserved words
	nonReserved := []string{
		"users", "posts", "comments", "email", "name", "title",
		"foo", "bar", "my_table", "custom_column",
	}
	for _, word := range nonReserved {
		t.Run("allowed_"+word, func(t *testing.T) {
			if IsReservedWord(word) {
				t.Errorf("IsReservedWord(%q) = true, want false", word)
			}
		})
	}
}

func TestReservedWordError(t *testing.T) {
	t.Run("returns error for reserved word", func(t *testing.T) {
		err := ReservedWordError("select")
		if err == nil {
			t.Fatal("expected error for reserved word")
		}
		vErr, ok := err.(*Error)
		if !ok {
			t.Fatalf("expected *Error, got %T", err)
		}
		if vErr.Code != ErrReservedWord {
			t.Errorf("code = %v, want %v", vErr.Code, ErrReservedWord)
		}
		if vErr.Context["identifier"] != "select" {
			t.Errorf("identifier = %v, want 'select'", vErr.Context["identifier"])
		}
		// Should have suggestion
		if _, ok := vErr.Context["suggestion"]; !ok {
			t.Error("expected suggestion in error context")
		}
	})

	t.Run("returns nil for non-reserved word", func(t *testing.T) {
		err := ReservedWordError("users")
		if err != nil {
			t.Errorf("expected nil error for non-reserved word, got %v", err)
		}
	})
}

// -----------------------------------------------------------------------------
// TableRef Tests
// -----------------------------------------------------------------------------

func TestTableRef(t *testing.T) {
	tests := []struct {
		name      string
		ref       string
		wantNS    string
		wantTable string
		wantErr   bool
	}{
		// Valid: namespace.table
		{"full_ref", "auth.users", "auth", "users", false},
		{"billing_ref", "billing.invoices", "billing", "invoices", false},

		// Valid: just table (no namespace)
		{"table_only", "users", "", "users", false},
		{"table_only_2", "posts", "", "posts", false},

		// Valid: leading dot (.table) for same-namespace reference
		{"leading_dot", ".users", "", "users", false},
		{"leading_dot_2", ".posts", "", "posts", false},

		// Invalid: empty
		{"empty", "", "", "", true},

		// Invalid: empty namespace with dot
		{"empty_ns", ".users", "", "users", false}, // This is valid - leading dot

		// Invalid: empty table after dot
		{"empty_table", "auth.", "", "", true},

		// Invalid: too many parts
		{"too_many_parts", "a.b.c", "", "", true},

		// Invalid: empty with just dot
		{"just_dot", ".", "", "", true},

		// Invalid: invalid table name
		{"invalid_table", "auth.User", "", "", true}, // PascalCase not allowed
		{"reserved_table", "auth.select", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ns, table, err := TableRef(tt.ref)
			if (err != nil) != tt.wantErr {
				t.Errorf("TableRef(%q) error = %v, wantErr %v", tt.ref, err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if ns != tt.wantNS {
					t.Errorf("namespace = %q, want %q", ns, tt.wantNS)
				}
				if table != tt.wantTable {
					t.Errorf("table = %q, want %q", table, tt.wantTable)
				}
			}
		})
	}
}

func TestParseRef(t *testing.T) {
	tests := []struct {
		name      string
		ref       string
		currentNS string
		wantNS    string
		wantTable string
		wantErr   bool
	}{
		// Full reference (namespace.table) - ignores currentNS
		{"full_ref", "auth.users", "billing", "auth", "users", false},
		{"full_ref_same", "auth.users", "auth", "auth", "users", false},

		// Relative reference (.table) - uses currentNS
		{"leading_dot", ".users", "auth", "auth", "users", false},
		{"leading_dot_billing", ".invoices", "billing", "billing", "invoices", false},

		// Table only - uses currentNS
		{"table_only", "users", "auth", "auth", "users", false},
		{"table_only_billing", "invoices", "billing", "billing", "invoices", false},

		// Error: relative ref without currentNS
		{"relative_no_ns", "users", "", "", "", true},
		{"dot_no_ns", ".users", "", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ns, table, err := ParseRef(tt.ref, tt.currentNS)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseRef(%q, %q) error = %v, wantErr %v", tt.ref, tt.currentNS, err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if ns != tt.wantNS {
					t.Errorf("namespace = %q, want %q", ns, tt.wantNS)
				}
				if table != tt.wantTable {
					t.Errorf("table = %q, want %q", table, tt.wantTable)
				}
			}
		})
	}
}

// -----------------------------------------------------------------------------
// ValidationErrors Tests
// -----------------------------------------------------------------------------

func TestValidationErrors(t *testing.T) {
	t.Run("empty collection", func(t *testing.T) {
		var ve ValidationErrors

		if ve.HasErrors() {
			t.Error("empty collection should not have errors")
		}
		if ve.ToError() != nil {
			t.Error("empty collection ToError should return nil")
		}
		if ve.Error() != "" {
			t.Error("empty collection Error() should return empty string")
		}
	})

	t.Run("add errors", func(t *testing.T) {
		var ve ValidationErrors

		// Add nil (should be ignored)
		ve.Add(nil)
		if ve.HasErrors() {
			t.Error("nil errors should be ignored")
		}

		// Add actual error
		ve.Add(newError(ErrInvalidIdentifier, "first error"))
		if !ve.HasErrors() {
			t.Error("should have errors after adding one")
		}
		if len(ve) != 1 {
			t.Errorf("length = %d, want 1", len(ve))
		}

		// Add another error
		ve.Add(newError(ErrInvalidSnakeCase, "second error"))
		if len(ve) != 2 {
			t.Errorf("length = %d, want 2", len(ve))
		}
	})

	t.Run("merge collections", func(t *testing.T) {
		var ve1 ValidationErrors
		ve1.Add(newError(ErrInvalidIdentifier, "error 1"))

		var ve2 ValidationErrors
		ve2.Add(newError(ErrInvalidSnakeCase, "error 2"))
		ve2.Add(newError(ErrReservedWord, "error 3"))

		ve1.Merge(ve2)
		if len(ve1) != 3 {
			t.Errorf("after merge: length = %d, want 3", len(ve1))
		}
	})

	t.Run("error string format", func(t *testing.T) {
		var ve ValidationErrors
		ve.Add(newError(ErrInvalidIdentifier, "first error"))
		ve.Add(newError(ErrInvalidSnakeCase, "second error"))

		errStr := ve.Error()

		// Should start with count
		if !strings.HasPrefix(errStr, "2 validation error(s):") {
			t.Errorf("error string should start with count, got: %s", errStr)
		}

		// Should contain numbered errors
		if !strings.Contains(errStr, "1.") {
			t.Errorf("error string should contain '1.', got: %s", errStr)
		}
		if !strings.Contains(errStr, "2.") {
			t.Errorf("error string should contain '2.', got: %s", errStr)
		}
	})

	t.Run("ToError with errors", func(t *testing.T) {
		var ve ValidationErrors
		ve.Add(newError(ErrInvalidIdentifier, "error"))

		err := ve.ToError()
		if err == nil {
			t.Error("ToError should return error when collection has errors")
		}
		// The returned error should be the ValidationErrors itself
		if _, ok := err.(ValidationErrors); !ok {
			t.Errorf("ToError should return ValidationErrors, got %T", err)
		}
	})
}

// -----------------------------------------------------------------------------
// Namespace and Column Validation Tests
// -----------------------------------------------------------------------------

func TestNamespace(t *testing.T) {
	tests := []struct {
		input   string
		wantErr bool
	}{
		// Valid
		{"auth", false},
		{"billing", false},
		{"core_services", false},

		// Invalid (same rules as SnakeCase)
		{"", true},
		{"Auth", true},
		{"auth-billing", true},
		{"select", true}, // reserved word
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			err := Namespace(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Namespace(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			// Check error message mentions namespace
			if err != nil {
				if !strings.Contains(err.Error(), "namespace") {
					t.Errorf("error should mention 'namespace': %v", err)
				}
			}
		})
	}
}

func TestTableName(t *testing.T) {
	tests := []struct {
		input   string
		wantErr bool
	}{
		// Valid
		{"users", false},
		{"user_profiles", false},
		{"posts", false},

		// Invalid
		{"", true},
		{"Users", true},
		{"user-profiles", true},
		{"select", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			err := TableName(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("TableName(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			// Check error message mentions table
			if err != nil {
				if !strings.Contains(err.Error(), "table") {
					t.Errorf("error should mention 'table': %v", err)
				}
			}
		})
	}
}

func TestColumnName(t *testing.T) {
	tests := []struct {
		input   string
		wantErr bool
	}{
		// Valid
		{"id", false},
		{"user_id", false},
		{"created_at", false},
		{"email", false},

		// Invalid
		{"", true},
		{"userId", true},
		{"user-id", true},
		{"select", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			err := ColumnName(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ColumnName(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			// Check error message mentions column
			if err != nil {
				if !strings.Contains(err.Error(), "column") {
					t.Errorf("error should mention 'column': %v", err)
				}
			}
		})
	}
}

// -----------------------------------------------------------------------------
// Error Type Tests
// -----------------------------------------------------------------------------

func TestErrorWith(t *testing.T) {
	err := newError(ErrInvalidIdentifier, "test").
		With("key1", "value1").
		With("key2", 42)

	if err.Context["key1"] != "value1" {
		t.Errorf("key1 = %v, want 'value1'", err.Context["key1"])
	}
	if err.Context["key2"] != 42 {
		t.Errorf("key2 = %v, want 42", err.Context["key2"])
	}
}

func TestErrorFormat(t *testing.T) {
	err := newError(ErrInvalidSnakeCase, "name must be snake_case").
		With("got", "userName").
		With("suggestion", "user_name")

	errStr := err.Error()

	// Should have code
	if !strings.Contains(errStr, "[E2002]") {
		t.Errorf("error should contain code: %s", errStr)
	}

	// Should have message
	if !strings.Contains(errStr, "name must be snake_case") {
		t.Errorf("error should contain message: %s", errStr)
	}

	// Should have context
	if !strings.Contains(errStr, "got: userName") {
		t.Errorf("error should contain got context: %s", errStr)
	}
	if !strings.Contains(errStr, "suggestion: user_name") {
		t.Errorf("error should contain suggestion context: %s", errStr)
	}
}
