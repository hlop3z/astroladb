package astroladb

import (
	"bytes"
	"io"
	"testing"
	"time"
)

// ===========================================================================
// detectDialect Tests
// ===========================================================================

func TestDetectDialect(t *testing.T) {
	tests := []struct {
		url  string
		want string
	}{
		// PostgreSQL URLs
		{"postgres://user:pass@localhost:5432/db", "postgres"},
		{"postgresql://user:pass@localhost:5432/db", "postgres"},
		{"POSTGRES://user:pass@localhost:5432/db", "postgres"}, // case insensitive
		{"PostgreSQL://user:pass@localhost:5432/db", "postgres"},

		// SQLite URLs
		{"sqlite://./mydb.db", "sqlite"},
		{"sqlite3://./mydb.db", "sqlite"},
		{"file:./mydb.db", "sqlite"},
		{"./mydb.db", "sqlite"}, // Path ending with .db
		{"/absolute/path.db", "sqlite"},
		{"./data.sqlite", "sqlite"},
		{"./data.sqlite3", "sqlite"},

		// Default to postgres for unknown
		{"mysql://user:pass@localhost/db", "postgres"},
		{"unknown", "postgres"},
		{"", "postgres"},
	}

	for _, tt := range tests {
		got := detectDialect(tt.url)
		if got != tt.want {
			t.Errorf("detectDialect(%q) = %q, want %q", tt.url, got, tt.want)
		}
	}
}

// ===========================================================================
// convertSQLiteURL Tests
// ===========================================================================

func TestConvertSQLiteURL(t *testing.T) {
	tests := []struct {
		url  string
		want string
	}{
		{"sqlite://./mydb.db", "./mydb.db"},
		{"sqlite3://./mydb.db", "./mydb.db"},
		{"file:./mydb.db", "./mydb.db"},
		{"./mydb.db", "./mydb.db"},
		{"/absolute/path.db", "/absolute/path.db"},
		{"file::memory:?cache=shared", ":memory:?cache=shared"},
		{"sqlite://:memory:", ":memory:"},
	}

	for _, tt := range tests {
		got := convertSQLiteURL(tt.url)
		if got != tt.want {
			t.Errorf("convertSQLiteURL(%q) = %q, want %q", tt.url, got, tt.want)
		}
	}
}

// ===========================================================================
// redactURL Tests
// ===========================================================================

func TestRedactURL(t *testing.T) {
	tests := []struct {
		url  string
		want string
	}{
		// With password
		{"postgres://user:secret@localhost:5432/db", "postgres://user:***@localhost:5432/db"},
		{"postgres://admin:verysecret123@host/db", "postgres://admin:***@host/db"},

		// No password (just user)
		{"postgres://user@localhost:5432/db", "postgres://user@localhost:5432/db"},

		// No credentials at all
		{"postgres://localhost:5432/db", "postgres://localhost:5432/db"},

		// No scheme (should return as-is)
		{"localhost:5432/db", "localhost:5432/db"},

		// SQLite (no credentials)
		{"sqlite://./mydb.db", "sqlite://./mydb.db"},

		// Empty string
		{"", ""},
	}

	for _, tt := range tests {
		got := redactURL(tt.url)
		if got != tt.want {
			t.Errorf("redactURL(%q) = %q, want %q", tt.url, got, tt.want)
		}
	}
}

// ===========================================================================
// Option Tests (Config Options)
// ===========================================================================

func TestWithDatabaseURL(t *testing.T) {
	cfg := &Config{}
	opt := WithDatabaseURL("postgres://localhost/db")
	opt(cfg)

	if cfg.DatabaseURL != "postgres://localhost/db" {
		t.Errorf("DatabaseURL = %q, want %q", cfg.DatabaseURL, "postgres://localhost/db")
	}
}

func TestWithSchemasDir(t *testing.T) {
	cfg := &Config{}
	opt := WithSchemasDir("./custom_schemas")
	opt(cfg)

	if cfg.SchemasDir != "./custom_schemas" {
		t.Errorf("SchemasDir = %q, want %q", cfg.SchemasDir, "./custom_schemas")
	}
}

func TestWithMigrationsDir(t *testing.T) {
	cfg := &Config{}
	opt := WithMigrationsDir("./custom_migrations")
	opt(cfg)

	if cfg.MigrationsDir != "./custom_migrations" {
		t.Errorf("MigrationsDir = %q, want %q", cfg.MigrationsDir, "./custom_migrations")
	}
}

func TestWithDialect(t *testing.T) {
	cfg := &Config{}
	opt := WithDialect("postgres")
	opt(cfg)

	if cfg.Dialect != "postgres" {
		t.Errorf("Dialect = %q, want %q", cfg.Dialect, "postgres")
	}
}

type testLogger struct {
	output string
}

func (l *testLogger) Printf(format string, v ...any) {
	l.output = format
}

func TestWithLogger(t *testing.T) {
	cfg := &Config{}
	logger := &testLogger{}
	opt := WithLogger(logger)
	opt(cfg)

	if cfg.Logger != logger {
		t.Error("Logger not set correctly")
	}
}

func TestWithTimeout(t *testing.T) {
	cfg := &Config{}
	opt := WithTimeout(5 * time.Minute)
	opt(cfg)

	if cfg.Timeout != 5*time.Minute {
		t.Errorf("Timeout = %v, want %v", cfg.Timeout, 5*time.Minute)
	}
}

func TestWithSchemaOnly(t *testing.T) {
	cfg := &Config{}
	opt := WithSchemaOnly()
	opt(cfg)

	if !cfg.SchemaOnly {
		t.Error("SchemaOnly should be true")
	}
}

// ===========================================================================
// MigrationOption Tests
// ===========================================================================

func TestDryRun(t *testing.T) {
	cfg := &MigrationConfig{}
	opt := DryRun()
	opt(cfg)

	if !cfg.DryRun {
		t.Error("DryRun should be true")
	}
}

func TestDryRunTo(t *testing.T) {
	cfg := &MigrationConfig{}
	buf := &bytes.Buffer{}
	opt := DryRunTo(buf)
	opt(cfg)

	if !cfg.DryRun {
		t.Error("DryRun should be true")
	}
	if cfg.Output != buf {
		t.Error("Output should be set to the buffer")
	}
}

func TestTarget(t *testing.T) {
	cfg := &MigrationConfig{}
	opt := Target("001")
	opt(cfg)

	if cfg.Target != "001" {
		t.Errorf("Target = %q, want %q", cfg.Target, "001")
	}
}

func TestSteps(t *testing.T) {
	cfg := &MigrationConfig{}
	opt := Steps(5)
	opt(cfg)

	if cfg.Steps != 5 {
		t.Errorf("Steps = %d, want %d", cfg.Steps, 5)
	}
}

func TestForce(t *testing.T) {
	cfg := &MigrationConfig{}
	opt := Force()
	opt(cfg)

	if !cfg.Force {
		t.Error("Force should be true")
	}
}

func TestSkipLock(t *testing.T) {
	cfg := &MigrationConfig{}
	opt := SkipLock()
	opt(cfg)

	if !cfg.SkipLock {
		t.Error("SkipLock should be true")
	}
}

func TestLockTimeout(t *testing.T) {
	cfg := &MigrationConfig{}
	opt := LockTimeout(2 * time.Minute)
	opt(cfg)

	if cfg.LockTimeout != 2*time.Minute {
		t.Errorf("LockTimeout = %v, want %v", cfg.LockTimeout, 2*time.Minute)
	}
}

func TestApplyMigrationOptions(t *testing.T) {
	cfg := applyMigrationOptions([]MigrationOption{
		DryRun(),
		Target("002"),
		Steps(3),
		Force(),
		SkipLock(),
		LockTimeout(time.Minute),
	})

	if !cfg.DryRun {
		t.Error("DryRun should be true")
	}
	if cfg.Target != "002" {
		t.Errorf("Target = %q, want %q", cfg.Target, "002")
	}
	if cfg.Steps != 3 {
		t.Errorf("Steps = %d, want %d", cfg.Steps, 3)
	}
	if !cfg.Force {
		t.Error("Force should be true")
	}
	if !cfg.SkipLock {
		t.Error("SkipLock should be true")
	}
	if cfg.LockTimeout != time.Minute {
		t.Errorf("LockTimeout = %v, want %v", cfg.LockTimeout, time.Minute)
	}
}

func TestApplyMigrationOptions_Defaults(t *testing.T) {
	cfg := applyMigrationOptions(nil)

	if cfg.DryRun {
		t.Error("DryRun should default to false")
	}
	if cfg.Target != "" {
		t.Error("Target should default to empty")
	}
	if cfg.Steps != 0 {
		t.Error("Steps should default to 0")
	}
	if cfg.Force {
		t.Error("Force should default to false")
	}
	if cfg.SkipLock {
		t.Error("SkipLock should default to false")
	}
}

// ===========================================================================
// ExportOption Tests
// ===========================================================================

func TestWithNamespace(t *testing.T) {
	cfg := &ExportConfig{}
	opt := WithNamespace("auth")
	opt(cfg)

	if cfg.Namespace != "auth" {
		t.Errorf("Namespace = %q, want %q", cfg.Namespace, "auth")
	}
}

func TestWithChrono(t *testing.T) {
	cfg := &ExportConfig{}
	opt := WithChrono()
	opt(cfg)

	if !cfg.UseChrono {
		t.Error("UseChrono should be true")
	}
}

func TestWithMik(t *testing.T) {
	cfg := &ExportConfig{}
	opt := WithMik()
	opt(cfg)

	if !cfg.UseMik {
		t.Error("UseMik should be true")
	}
}

func TestApplyExportOptions(t *testing.T) {
	cfg := applyExportOptions([]ExportOption{
		WithNamespace("billing"),
		WithChrono(),
		WithMik(),
	})

	if cfg.Namespace != "billing" {
		t.Errorf("Namespace = %q, want %q", cfg.Namespace, "billing")
	}
	if !cfg.UseChrono {
		t.Error("UseChrono should be true")
	}
	if !cfg.UseMik {
		t.Error("UseMik should be true")
	}
}

func TestApplyExportOptions_Defaults(t *testing.T) {
	cfg := applyExportOptions(nil)

	if cfg.Namespace != "" {
		t.Error("Namespace should default to empty")
	}
	if cfg.UseChrono {
		t.Error("UseChrono should default to false")
	}
	if cfg.UseMik {
		t.Error("UseMik should default to false")
	}
}

// ===========================================================================
// Config Combination Tests
// ===========================================================================

func TestMultipleConfigOptions(t *testing.T) {
	cfg := &Config{}

	opts := []Option{
		WithDatabaseURL("postgres://localhost/db"),
		WithSchemasDir("./schemas"),
		WithMigrationsDir("./migrations"),
		WithDialect("postgres"),
		WithTimeout(time.Minute),
		WithSchemaOnly(),
	}

	for _, opt := range opts {
		opt(cfg)
	}

	if cfg.DatabaseURL != "postgres://localhost/db" {
		t.Error("DatabaseURL not set")
	}
	if cfg.SchemasDir != "./schemas" {
		t.Error("SchemasDir not set")
	}
	if cfg.MigrationsDir != "./migrations" {
		t.Error("MigrationsDir not set")
	}
	if cfg.Dialect != "postgres" {
		t.Error("Dialect not set")
	}
	if cfg.Timeout != time.Minute {
		t.Error("Timeout not set")
	}
	if !cfg.SchemaOnly {
		t.Error("SchemaOnly not set")
	}
}

// ===========================================================================
// DryRunTo Output Tests
// ===========================================================================

func TestDryRunTo_WritesOutput(t *testing.T) {
	buf := &bytes.Buffer{}
	cfg := &MigrationConfig{}

	DryRunTo(buf)(cfg)

	if cfg.Output != buf {
		t.Error("Output writer not set correctly")
	}

	// Write something to verify it's the same buffer
	_, err := cfg.Output.Write([]byte("test"))
	if err != nil {
		t.Fatal(err)
	}
	if buf.String() != "test" {
		t.Error("Output not writing to buffer")
	}
}

func TestDryRunTo_NilOutput(t *testing.T) {
	var w io.Writer = nil
	cfg := &MigrationConfig{}

	DryRunTo(w)(cfg)

	if !cfg.DryRun {
		t.Error("DryRun should be true even with nil output")
	}
}
