package lockfile

import (
	"os"
	"path/filepath"
	"testing"
)

func setupMigrations(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	migDir := filepath.Join(dir, "migrations")
	if err := os.MkdirAll(migDir, 0755); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(migDir, "001_init.js"), []byte("migration({up(m){},down(m){}})"), 0644)
	os.WriteFile(filepath.Join(migDir, "002_users.js"), []byte("migration({up(m){m.create_table('auth.user')},down(m){}})"), 0644)
	return dir
}

func TestWriteAndRead(t *testing.T) {
	dir := setupMigrations(t)
	migDir := filepath.Join(dir, "migrations")
	lockPath := filepath.Join(dir, "schema.lock")

	if err := Write(migDir, lockPath); err != nil {
		t.Fatalf("Write: %v", err)
	}

	lf, err := Read(lockPath)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if lf == nil {
		t.Fatal("expected lock file, got nil")
	}
	if lf.Aggregate == "" {
		t.Error("aggregate should not be empty")
	}
	if len(lf.Entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(lf.Entries))
	}
	if lf.Entries[0].Filename != "001_init.js" {
		t.Errorf("expected first entry '001_init.js', got %q", lf.Entries[0].Filename)
	}
}

func TestRead_NotFound(t *testing.T) {
	lf, err := Read(filepath.Join(t.TempDir(), "nonexistent.lock"))
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if lf != nil {
		t.Fatalf("expected nil lock file, got %+v", lf)
	}
}

func TestVerify_OK(t *testing.T) {
	dir := setupMigrations(t)
	migDir := filepath.Join(dir, "migrations")
	lockPath := filepath.Join(dir, "schema.lock")

	Write(migDir, lockPath)

	if err := Verify(migDir, lockPath); err != nil {
		t.Fatalf("Verify should pass: %v", err)
	}
}

func TestVerify_ModifiedFile(t *testing.T) {
	dir := setupMigrations(t)
	migDir := filepath.Join(dir, "migrations")
	lockPath := filepath.Join(dir, "schema.lock")

	Write(migDir, lockPath)

	// Modify a migration file
	os.WriteFile(filepath.Join(migDir, "001_init.js"), []byte("changed content"), 0644)

	err := Verify(migDir, lockPath)
	if err == nil {
		t.Fatal("Verify should fail after file modification")
	}
}

func TestVerify_NewFile(t *testing.T) {
	dir := setupMigrations(t)
	migDir := filepath.Join(dir, "migrations")
	lockPath := filepath.Join(dir, "schema.lock")

	Write(migDir, lockPath)

	// Add a new migration
	os.WriteFile(filepath.Join(migDir, "003_new.js"), []byte("new"), 0644)

	err := Verify(migDir, lockPath)
	if err == nil {
		t.Fatal("Verify should fail with new file")
	}
}

func TestVerify_DeletedFile(t *testing.T) {
	dir := setupMigrations(t)
	migDir := filepath.Join(dir, "migrations")
	lockPath := filepath.Join(dir, "schema.lock")

	Write(migDir, lockPath)

	os.Remove(filepath.Join(migDir, "002_users.js"))

	err := Verify(migDir, lockPath)
	if err == nil {
		t.Fatal("Verify should fail with deleted file")
	}
}

func TestVerify_NoLockFile(t *testing.T) {
	dir := setupMigrations(t)
	migDir := filepath.Join(dir, "migrations")

	err := Verify(migDir, filepath.Join(dir, "missing.lock"))
	if err == nil {
		t.Fatal("Verify should fail when lock file missing")
	}
}

func TestRepair(t *testing.T) {
	dir := setupMigrations(t)
	migDir := filepath.Join(dir, "migrations")
	lockPath := filepath.Join(dir, "schema.lock")

	Write(migDir, lockPath)

	// Modify a file, breaking the lock
	os.WriteFile(filepath.Join(migDir, "001_init.js"), []byte("changed"), 0644)
	if err := Verify(migDir, lockPath); err == nil {
		t.Fatal("should be broken")
	}

	// Repair
	if err := Repair(migDir, lockPath); err != nil {
		t.Fatalf("Repair: %v", err)
	}

	// Should pass now
	if err := Verify(migDir, lockPath); err != nil {
		t.Fatalf("Verify after repair: %v", err)
	}
}

func TestDefaultPath(t *testing.T) {
	p := DefaultPath()
	if p != filepath.Join(".alab", "schema.lock") {
		t.Errorf("unexpected default path: %q", p)
	}
}

func TestWrite_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	migDir := filepath.Join(dir, "migrations")
	os.MkdirAll(migDir, 0755)
	lockPath := filepath.Join(dir, "schema.lock")

	if err := Write(migDir, lockPath); err != nil {
		t.Fatalf("Write empty dir: %v", err)
	}

	lf, _ := Read(lockPath)
	if lf == nil {
		t.Fatal("expected lock file")
	}
	if len(lf.Entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(lf.Entries))
	}
}

func TestWrite_IgnoresNonJS(t *testing.T) {
	dir := t.TempDir()
	migDir := filepath.Join(dir, "migrations")
	os.MkdirAll(migDir, 0755)
	os.WriteFile(filepath.Join(migDir, "001_init.js"), []byte("js"), 0644)
	os.WriteFile(filepath.Join(migDir, "readme.md"), []byte("docs"), 0644)
	os.WriteFile(filepath.Join(migDir, "notes.txt"), []byte("notes"), 0644)
	lockPath := filepath.Join(dir, "schema.lock")

	Write(migDir, lockPath)
	lf, _ := Read(lockPath)
	if len(lf.Entries) != 1 {
		t.Errorf("expected 1 entry (only .js), got %d", len(lf.Entries))
	}
}
