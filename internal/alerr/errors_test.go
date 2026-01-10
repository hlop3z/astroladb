package alerr

import (
	"errors"
	"strings"
	"testing"
)

// -----------------------------------------------------------------------------
// Constructor Tests
// -----------------------------------------------------------------------------

func TestNew(t *testing.T) {
	tests := []struct {
		name    string
		code    Code
		message string
	}{
		{
			name:    "schema error",
			code:    ErrSchemaInvalid,
			message: "schema is invalid",
		},
		{
			name:    "validation error",
			code:    ErrInvalidIdentifier,
			message: "identifier is not valid",
		},
		{
			name:    "migration error",
			code:    ErrMigrationFailed,
			message: "migration failed to execute",
		},
		{
			name:    "SQL error",
			code:    ErrSQLExecution,
			message: "SQL statement failed",
		},
		{
			name:    "JS runtime error",
			code:    ErrJSExecution,
			message: "JavaScript execution failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := New(tt.code, tt.message)
			if err == nil {
				t.Fatal("expected non-nil error")
			}
			if err.GetCode() != tt.code {
				t.Errorf("code = %v, want %v", err.GetCode(), tt.code)
			}
			if err.GetMessage() != tt.message {
				t.Errorf("message = %v, want %v", err.GetMessage(), tt.message)
			}
			if err.GetCause() != nil {
				t.Error("expected nil cause for New()")
			}
			if err.GetStack() == "" {
				t.Error("expected stack trace to be captured")
			}
		})
	}
}

func TestWrap(t *testing.T) {
	t.Run("wrap existing error", func(t *testing.T) {
		cause := errors.New("underlying error")
		err := Wrap(ErrSQLExecution, cause, "failed to execute query")

		if err.GetCode() != ErrSQLExecution {
			t.Errorf("code = %v, want %v", err.GetCode(), ErrSQLExecution)
		}
		if err.GetCause() != cause {
			t.Error("cause should be the wrapped error")
		}
		if err.GetMessage() != "failed to execute query" {
			t.Errorf("message = %v, want %v", err.GetMessage(), "failed to execute query")
		}
	})

	t.Run("wrap nil error behaves like New", func(t *testing.T) {
		err := Wrap(ErrSchemaInvalid, nil, "schema error")

		if err.GetCode() != ErrSchemaInvalid {
			t.Errorf("code = %v, want %v", err.GetCode(), ErrSchemaInvalid)
		}
		if err.GetCause() != nil {
			t.Error("cause should be nil when wrapping nil")
		}
	})
}

func TestWrapf(t *testing.T) {
	cause := errors.New("connection refused")
	err := Wrapf(ErrSQLConnection, cause, "failed to connect to %s on port %d", "localhost", 5432)

	expected := "failed to connect to localhost on port 5432"
	if err.GetMessage() != expected {
		t.Errorf("message = %v, want %v", err.GetMessage(), expected)
	}
	if err.GetCause() != cause {
		t.Error("cause should be preserved")
	}
}

// -----------------------------------------------------------------------------
// Context Builder Tests
// -----------------------------------------------------------------------------

func TestWith(t *testing.T) {
	err := New(ErrSchemaInvalid, "invalid schema").
		With("key1", "value1").
		With("key2", 42).
		With("key3", true)

	ctx := err.GetContext()
	if ctx["key1"] != "value1" {
		t.Errorf("key1 = %v, want %v", ctx["key1"], "value1")
	}
	if ctx["key2"] != 42 {
		t.Errorf("key2 = %v, want %v", ctx["key2"], 42)
	}
	if ctx["key3"] != true {
		t.Errorf("key3 = %v, want %v", ctx["key3"], true)
	}
}

func TestWithTable(t *testing.T) {
	tests := []struct {
		name      string
		namespace string
		table     string
		want      string
	}{
		{
			name:      "with namespace",
			namespace: "auth",
			table:     "users",
			want:      "auth.users",
		},
		{
			name:      "without namespace",
			namespace: "",
			table:     "users",
			want:      "users",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := New(ErrSchemaInvalid, "invalid table").
				WithTable(tt.namespace, tt.table)

			ctx := err.GetContext()
			if ctx["table"] != tt.want {
				t.Errorf("table = %v, want %v", ctx["table"], tt.want)
			}
		})
	}
}

func TestWithColumn(t *testing.T) {
	err := New(ErrSchemaInvalid, "invalid column").
		WithColumn("user_name")

	ctx := err.GetContext()
	if ctx["column"] != "user_name" {
		t.Errorf("column = %v, want %v", ctx["column"], "user_name")
	}
}

func TestWithSQL(t *testing.T) {
	sql := "SELECT * FROM users WHERE id = $1"
	err := New(ErrSQLExecution, "query failed").
		WithSQL(sql)

	ctx := err.GetContext()
	if ctx["sql"] != sql {
		t.Errorf("sql = %v, want %v", ctx["sql"], sql)
	}
}

func TestWithFile(t *testing.T) {
	t.Run("with line number", func(t *testing.T) {
		err := New(ErrJSExecution, "syntax error").
			WithFile("schemas/auth/users.js", 42)

		ctx := err.GetContext()
		if ctx["file"] != "schemas/auth/users.js" {
			t.Errorf("file = %v, want %v", ctx["file"], "schemas/auth/users.js")
		}
		if ctx["line"] != 42 {
			t.Errorf("line = %v, want %v", ctx["line"], 42)
		}
	})

	t.Run("without line number", func(t *testing.T) {
		err := New(ErrJSExecution, "file error").
			WithFile("schemas/auth/users.js", 0)

		ctx := err.GetContext()
		if ctx["file"] != "schemas/auth/users.js" {
			t.Errorf("file = %v, want %v", ctx["file"], "schemas/auth/users.js")
		}
		if _, exists := ctx["line"]; exists {
			t.Error("line should not be set when 0")
		}
	})
}

// -----------------------------------------------------------------------------
// Error Output Format Tests
// -----------------------------------------------------------------------------

func TestErrorFormat(t *testing.T) {
	t.Run("basic error format", func(t *testing.T) {
		err := New(ErrInvalidSnakeCase, "column name must be snake_case")
		errStr := err.Error()

		if !strings.HasPrefix(errStr, "[E2002]") {
			t.Errorf("error should start with code, got: %s", errStr)
		}
		if !strings.Contains(errStr, "column name must be snake_case") {
			t.Errorf("error should contain message, got: %s", errStr)
		}
	})

	t.Run("error with context", func(t *testing.T) {
		err := New(ErrInvalidSnakeCase, "column name must be snake_case").
			WithTable("auth", "users").
			With("got", "userName").
			With("want", "user_name")

		errStr := err.Error()

		// Check code and message
		if !strings.Contains(errStr, "[E2002]") {
			t.Errorf("error should contain code, got: %s", errStr)
		}
		// Check context values are present
		if !strings.Contains(errStr, "table: auth.users") {
			t.Errorf("error should contain table context, got: %s", errStr)
		}
		if !strings.Contains(errStr, "got: userName") {
			t.Errorf("error should contain 'got' context, got: %s", errStr)
		}
		if !strings.Contains(errStr, "want: user_name") {
			t.Errorf("error should contain 'want' context, got: %s", errStr)
		}
	})

	t.Run("error with cause", func(t *testing.T) {
		cause := errors.New("connection timeout")
		err := Wrap(ErrSQLConnection, cause, "failed to connect")

		errStr := err.Error()
		if !strings.Contains(errStr, "cause: connection timeout") {
			t.Errorf("error should contain cause, got: %s", errStr)
		}
	})

	t.Run("context keys are sorted", func(t *testing.T) {
		err := New(ErrSchemaInvalid, "test").
			With("zebra", 1).
			With("alpha", 2).
			With("middle", 3)

		errStr := err.Error()
		alphaIdx := strings.Index(errStr, "alpha:")
		middleIdx := strings.Index(errStr, "middle:")
		zebraIdx := strings.Index(errStr, "zebra:")

		if alphaIdx == -1 || middleIdx == -1 || zebraIdx == -1 {
			t.Fatalf("expected all keys to be present, got: %s", errStr)
		}
		if !(alphaIdx < middleIdx && middleIdx < zebraIdx) {
			t.Errorf("context keys should be sorted alphabetically, got: %s", errStr)
		}
	})
}

// -----------------------------------------------------------------------------
// Is() and errors.Is() Tests
// -----------------------------------------------------------------------------

func TestIs(t *testing.T) {
	t.Run("same code matches", func(t *testing.T) {
		err1 := New(ErrSchemaInvalid, "first error")
		err2 := New(ErrSchemaInvalid, "second error with same code")

		if !err1.Is(err2) {
			t.Error("errors with same code should match")
		}
	})

	t.Run("different codes do not match", func(t *testing.T) {
		err1 := New(ErrSchemaInvalid, "schema error")
		err2 := New(ErrMigrationFailed, "migration error")

		if err1.Is(err2) {
			t.Error("errors with different codes should not match")
		}
	})

	t.Run("nil target does not match", func(t *testing.T) {
		err := New(ErrSchemaInvalid, "error")
		if err.Is(nil) {
			t.Error("error should not match nil")
		}
	})

	t.Run("non-alerr error does not match", func(t *testing.T) {
		err := New(ErrSchemaInvalid, "alab error")
		stdErr := errors.New("standard error")

		if err.Is(stdErr) {
			t.Error("alab error should not match standard error")
		}
	})
}

func TestErrorsIsCompatibility(t *testing.T) {
	t.Run("errors.Is finds wrapped error", func(t *testing.T) {
		cause := errors.New("original error")
		wrapped := Wrap(ErrSQLExecution, cause, "wrapped")

		if !errors.Is(wrapped, cause) {
			t.Error("errors.Is should find the wrapped cause")
		}
	})

	t.Run("errors.Is works with code matching", func(t *testing.T) {
		err1 := New(ErrSchemaInvalid, "error 1")
		err2 := New(ErrSchemaInvalid, "error 2")

		if !errors.Is(err1, err2) {
			t.Error("errors.Is should match errors with same code")
		}
	})
}

// -----------------------------------------------------------------------------
// GetErrorCode Tests
// -----------------------------------------------------------------------------

func TestGetErrorCode(t *testing.T) {
	t.Run("extract code from alerr.Error", func(t *testing.T) {
		err := New(ErrMigrationConflict, "conflict")
		code := GetErrorCode(err)

		if code != ErrMigrationConflict {
			t.Errorf("code = %v, want %v", code, ErrMigrationConflict)
		}
	})

	t.Run("extract code from wrapped error chain", func(t *testing.T) {
		inner := New(ErrSQLExecution, "inner")
		outer := Wrap(ErrMigrationFailed, inner, "outer")

		// GetErrorCode should find the outermost alerr code
		code := GetErrorCode(outer)
		if code != ErrMigrationFailed {
			t.Errorf("code = %v, want %v", code, ErrMigrationFailed)
		}
	})

	t.Run("return empty for nil error", func(t *testing.T) {
		code := GetErrorCode(nil)
		if code != "" {
			t.Errorf("code = %v, want empty string", code)
		}
	})

	t.Run("return empty for non-alerr error", func(t *testing.T) {
		stdErr := errors.New("standard error")
		code := GetErrorCode(stdErr)

		if code != "" {
			t.Errorf("code = %v, want empty string", code)
		}
	})
}

func TestIsCode(t *testing.T) {
	tests := []struct {
		name string
		err  error
		code Code
		want bool
	}{
		{
			name: "matching code",
			err:  New(ErrSchemaInvalid, "test"),
			code: ErrSchemaInvalid,
			want: true,
		},
		{
			name: "non-matching code",
			err:  New(ErrSchemaInvalid, "test"),
			code: ErrMigrationFailed,
			want: false,
		},
		{
			name: "nil error",
			err:  nil,
			code: ErrSchemaInvalid,
			want: false,
		},
		{
			name: "standard error",
			err:  errors.New("standard"),
			code: ErrSchemaInvalid,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Is(tt.err, tt.code)
			if got != tt.want {
				t.Errorf("Is() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHasCode(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "alerr error has code",
			err:  New(ErrSchemaInvalid, "test"),
			want: true,
		},
		{
			name: "standard error has no code",
			err:  errors.New("standard"),
			want: false,
		},
		{
			name: "nil error has no code",
			err:  nil,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := HasCode(tt.err)
			if got != tt.want {
				t.Errorf("HasCode() = %v, want %v", got, tt.want)
			}
		})
	}
}

// -----------------------------------------------------------------------------
// Error Code Categories Tests
// -----------------------------------------------------------------------------

func TestErrorCodeCategories(t *testing.T) {
	// Verify error codes follow the expected format E{category}xxx
	schemaErrors := []Code{ErrSchemaInvalid, ErrSchemaNotFound, ErrSchemaDuplicate, ErrSchemaCircularRef}
	for _, code := range schemaErrors {
		if !strings.HasPrefix(string(code), "E1") {
			t.Errorf("schema error %v should start with E1", code)
		}
	}

	validationErrors := []Code{ErrInvalidIdentifier, ErrInvalidSnakeCase, ErrInvalidReference, ErrReservedWord, ErrInvalidType}
	for _, code := range validationErrors {
		if !strings.HasPrefix(string(code), "E2") {
			t.Errorf("validation error %v should start with E2", code)
		}
	}

	migrationErrors := []Code{ErrMigrationFailed, ErrMigrationNotFound, ErrMigrationConflict, ErrMigrationChecksum}
	for _, code := range migrationErrors {
		if !strings.HasPrefix(string(code), "E3") {
			t.Errorf("migration error %v should start with E3", code)
		}
	}

	sqlErrors := []Code{ErrSQLExecution, ErrSQLConnection, ErrSQLTransaction}
	for _, code := range sqlErrors {
		if !strings.HasPrefix(string(code), "E4") {
			t.Errorf("SQL error %v should start with E4", code)
		}
	}

	jsErrors := []Code{ErrJSExecution, ErrJSTimeout, ErrJSNonDeterministic}
	for _, code := range jsErrors {
		if !strings.HasPrefix(string(code), "E5") {
			t.Errorf("JS error %v should start with E5", code)
		}
	}
}

// -----------------------------------------------------------------------------
// Method Chaining Tests
// -----------------------------------------------------------------------------

func TestMethodChaining(t *testing.T) {
	// Verify that all context methods return the error for chaining
	err := New(ErrSchemaInvalid, "test").
		With("key", "value").
		WithTable("ns", "table").
		WithColumn("col").
		WithSQL("SELECT 1").
		WithFile("file.js", 10)

	ctx := err.GetContext()
	if len(ctx) != 6 { // key, table, column, sql, file, line
		t.Errorf("expected 6 context entries, got %d", len(ctx))
	}
}

// -----------------------------------------------------------------------------
// Unwrap Tests
// -----------------------------------------------------------------------------

func TestUnwrap(t *testing.T) {
	cause := errors.New("root cause")
	err := Wrap(ErrSQLExecution, cause, "wrapper")

	unwrapped := err.Unwrap()
	if unwrapped != cause {
		t.Errorf("Unwrap() = %v, want %v", unwrapped, cause)
	}
}
