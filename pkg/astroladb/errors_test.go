package astroladb

import (
	"errors"
	"strings"
	"testing"

	"github.com/hlop3z/astroladb/internal/chain"
)

// ===========================================================================
// MigrationError Tests
// ===========================================================================

func TestMigrationError_Error_WithSQL(t *testing.T) {
	err := &MigrationError{
		Revision:  "001",
		Operation: "CREATE TABLE",
		SQL:       "CREATE TABLE users (id INT)",
		Cause:     errors.New("table already exists"),
	}

	msg := err.Error()

	if !strings.Contains(msg, "migration 001 failed") {
		t.Error("error message should contain revision")
	}
	if !strings.Contains(msg, "CREATE TABLE") {
		t.Error("error message should contain operation")
	}
	if !strings.Contains(msg, "table already exists") {
		t.Error("error message should contain cause")
	}
	if !strings.Contains(msg, "SQL: CREATE TABLE users") {
		t.Error("error message should contain SQL statement")
	}
}

func TestMigrationError_Error_WithoutSQL(t *testing.T) {
	err := &MigrationError{
		Revision:  "002",
		Operation: "ROLLBACK",
		Cause:     errors.New("rollback failed"),
	}

	msg := err.Error()

	if !strings.Contains(msg, "migration 002 failed") {
		t.Error("error message should contain revision")
	}
	if !strings.Contains(msg, "ROLLBACK") {
		t.Error("error message should contain operation")
	}
	if strings.Contains(msg, "SQL:") {
		t.Error("error message should not contain SQL section when empty")
	}
}

func TestMigrationError_Unwrap(t *testing.T) {
	cause := errors.New("underlying error")
	err := &MigrationError{
		Revision: "001",
		Cause:    cause,
	}

	if err.Unwrap() != cause {
		t.Error("Unwrap should return the cause")
	}
}

func TestMigrationError_Is(t *testing.T) {
	err := &MigrationError{
		Revision: "001",
		Cause:    errors.New("some error"),
	}

	if !errors.Is(err, ErrMigrationFailed) {
		t.Error("MigrationError should match ErrMigrationFailed")
	}

	if errors.Is(err, ErrSchemaInvalid) {
		t.Error("MigrationError should not match ErrSchemaInvalid")
	}
}

// ===========================================================================
// SchemaError Tests
// ===========================================================================

func TestSchemaError_Error_Full(t *testing.T) {
	cause := errors.New("invalid type")
	err := &SchemaError{
		Namespace: "auth",
		Table:     "user",
		Column:    "email",
		File:      "/path/to/schema.alab",
		Line:      42,
		Message:   "unsupported type",
		Cause:     cause,
	}

	msg := err.Error()

	if !strings.Contains(msg, "/path/to/schema.alab:42:") {
		t.Error("error message should contain file and line")
	}
	if !strings.Contains(msg, "auth.user.email:") {
		t.Error("error message should contain namespace.table.column")
	}
	if !strings.Contains(msg, "unsupported type") {
		t.Error("error message should contain message")
	}
	if !strings.Contains(msg, "invalid type") {
		t.Error("error message should contain cause")
	}
}

func TestSchemaError_Error_FileNoLine(t *testing.T) {
	err := &SchemaError{
		File:    "/path/to/schema.alab",
		Message: "parse error",
	}

	msg := err.Error()

	if !strings.Contains(msg, "/path/to/schema.alab: ") {
		t.Error("error message should contain file without line number")
	}
}

func TestSchemaError_Error_TableOnly(t *testing.T) {
	err := &SchemaError{
		Table:   "user",
		Message: "table error",
	}

	msg := err.Error()

	if !strings.Contains(msg, "user: ") {
		t.Error("error message should contain table name")
	}
}

func TestSchemaError_Error_TableWithColumn(t *testing.T) {
	err := &SchemaError{
		Table:   "user",
		Column:  "email",
		Message: "column error",
	}

	msg := err.Error()

	if !strings.Contains(msg, "user.email: ") {
		t.Error("error message should contain table.column")
	}
}

func TestSchemaError_Error_NamespaceAndTable(t *testing.T) {
	err := &SchemaError{
		Namespace: "auth",
		Table:     "user",
		Message:   "table error",
	}

	msg := err.Error()

	if !strings.Contains(msg, "auth.user: ") {
		t.Error("error message should contain namespace.table")
	}
}

func TestSchemaError_Error_NoCause(t *testing.T) {
	err := &SchemaError{
		Message: "simple error",
	}

	msg := err.Error()

	if !strings.Contains(msg, "simple error") {
		t.Error("error message should contain message")
	}
	if strings.Count(msg, ":") > 1 {
		t.Error("error message should not have extra colons when no cause")
	}
}

func TestSchemaError_Unwrap(t *testing.T) {
	cause := errors.New("underlying error")
	err := &SchemaError{
		Cause: cause,
	}

	if err.Unwrap() != cause {
		t.Error("Unwrap should return the cause")
	}
}

func TestSchemaError_Is(t *testing.T) {
	err := &SchemaError{
		Message: "test",
	}

	if !errors.Is(err, ErrSchemaInvalid) {
		t.Error("SchemaError should match ErrSchemaInvalid")
	}

	if errors.Is(err, ErrMigrationFailed) {
		t.Error("SchemaError should not match ErrMigrationFailed")
	}
}

// ===========================================================================
// ConnectionError Tests
// ===========================================================================

func TestConnectionError_Error(t *testing.T) {
	err := &ConnectionError{
		URL:     "postgres://user:***@localhost:5432/db",
		Dialect: "postgres",
		Cause:   errors.New("connection refused"),
	}

	msg := err.Error()

	if !strings.Contains(msg, "failed to connect to postgres database") {
		t.Error("error message should contain dialect")
	}
	if !strings.Contains(msg, "connection refused") {
		t.Error("error message should contain cause")
	}
}

func TestConnectionError_Unwrap(t *testing.T) {
	cause := errors.New("network error")
	err := &ConnectionError{
		Cause: cause,
	}

	if err.Unwrap() != cause {
		t.Error("Unwrap should return the cause")
	}
}

func TestConnectionError_Is(t *testing.T) {
	err := &ConnectionError{
		Dialect: "sqlite",
		Cause:   errors.New("file not found"),
	}

	if !errors.Is(err, ErrConnectionFailed) {
		t.Error("ConnectionError should match ErrConnectionFailed")
	}

	if errors.Is(err, ErrSchemaInvalid) {
		t.Error("ConnectionError should not match ErrSchemaInvalid")
	}
}

// ===========================================================================
// ChainError Tests
// ===========================================================================

func TestChainError_Error_NilResult(t *testing.T) {
	err := &ChainError{
		Result: nil,
	}

	msg := err.Error()

	if msg != "astroladb: chain integrity broken" {
		t.Errorf("error message = %q, want 'astroladb: chain integrity broken'", msg)
	}
}

func TestChainError_Error_WithResult(t *testing.T) {
	err := &ChainError{
		Result: &chain.VerificationResult{
			Valid:  false,
			Errors: []chain.ChainError{{Type: chain.ErrorTampered, Message: "hash mismatch"}},
		},
	}

	msg := err.Error()

	if !strings.Contains(msg, "chain integrity broken") {
		t.Error("error message should contain 'chain integrity broken'")
	}
}

func TestChainError_Is(t *testing.T) {
	err := &ChainError{
		Result: nil,
	}

	if !errors.Is(err, ErrChainIntegrity) {
		t.Error("ChainError should match ErrChainIntegrity")
	}

	if errors.Is(err, ErrMigrationFailed) {
		t.Error("ChainError should not match ErrMigrationFailed")
	}
}

// ===========================================================================
// Sentinel Errors Tests
// ===========================================================================

func TestSentinelErrors(t *testing.T) {
	sentinels := []struct {
		err  error
		name string
	}{
		{ErrMissingDatabaseURL, "ErrMissingDatabaseURL"},
		{ErrConnectionFailed, "ErrConnectionFailed"},
		{ErrMigrationFailed, "ErrMigrationFailed"},
		{ErrSchemaInvalid, "ErrSchemaInvalid"},
		{ErrUnsupportedDialect, "ErrUnsupportedDialect"},
		{ErrSchemaNotFound, "ErrSchemaNotFound"},
		{ErrExportFormatUnknown, "ErrExportFormatUnknown"},
		{ErrChainIntegrity, "ErrChainIntegrity"},
	}

	for _, tt := range sentinels {
		if tt.err == nil {
			t.Errorf("%s should not be nil", tt.name)
		}
		if tt.err.Error() == "" {
			t.Errorf("%s should have an error message", tt.name)
		}
		if !strings.Contains(tt.err.Error(), "astroladb:") {
			t.Errorf("%s should have 'astroladb:' prefix", tt.name)
		}
	}
}

// ===========================================================================
// Error Wrapping Integration Tests
// ===========================================================================

func TestErrorWrapping_MigrationError(t *testing.T) {
	dbErr := errors.New("constraint violation")
	migErr := &MigrationError{
		Revision:  "003",
		Operation: "INSERT",
		Cause:     dbErr,
	}

	// Should be able to unwrap to the original error
	if !errors.Is(migErr, dbErr) {
		t.Error("should be able to unwrap to original db error")
	}

	// Should match the sentinel
	if !errors.Is(migErr, ErrMigrationFailed) {
		t.Error("should match ErrMigrationFailed sentinel")
	}
}

func TestErrorWrapping_SchemaError(t *testing.T) {
	parseErr := errors.New("unexpected token")
	schemaErr := &SchemaError{
		File:    "schema.alab",
		Line:    10,
		Message: "parse failed",
		Cause:   parseErr,
	}

	// Should be able to unwrap to the original error
	if !errors.Is(schemaErr, parseErr) {
		t.Error("should be able to unwrap to original parse error")
	}

	// Should match the sentinel
	if !errors.Is(schemaErr, ErrSchemaInvalid) {
		t.Error("should match ErrSchemaInvalid sentinel")
	}
}

func TestErrorWrapping_ConnectionError(t *testing.T) {
	netErr := errors.New("network unreachable")
	connErr := &ConnectionError{
		Dialect: "postgres",
		Cause:   netErr,
	}

	// Should be able to unwrap to the original error
	if !errors.Is(connErr, netErr) {
		t.Error("should be able to unwrap to original network error")
	}

	// Should match the sentinel
	if !errors.Is(connErr, ErrConnectionFailed) {
		t.Error("should match ErrConnectionFailed sentinel")
	}
}
