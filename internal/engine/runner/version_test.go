//go:build integration

package runner

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	"github.com/hlop3z/astroladb/internal/alerr"
	"github.com/hlop3z/astroladb/internal/dialect"
)

// -----------------------------------------------------------------------------
// Test Helpers
// -----------------------------------------------------------------------------

// setupTestDB creates a new in-memory SQLite database for testing.
func setupTestDB(t *testing.T) (*sql.DB, dialect.Dialect) {
	t.Helper()

	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}

	d := dialect.SQLite()
	return db, d
}

// setupVersionManager creates a VersionManager with test database.
func setupVersionManager(t *testing.T) (*VersionManager, func()) {
	t.Helper()

	db, d := setupTestDB(t)
	vm := NewVersionManager(db, d)

	cleanup := func() {
		db.Close()
	}

	return vm, cleanup
}

// -----------------------------------------------------------------------------
// NewVersionManager Tests
// -----------------------------------------------------------------------------

func TestNewVersionManager(t *testing.T) {
	db, d := setupTestDB(t)
	defer db.Close()

	vm := NewVersionManager(db, d)
	if vm == nil {
		t.Fatal("NewVersionManager() returned nil")
	}
}

// -----------------------------------------------------------------------------
// EnsureTable Tests
// -----------------------------------------------------------------------------

func TestEnsureTable(t *testing.T) {
	t.Run("creates_table", func(t *testing.T) {
		vm, cleanup := setupVersionManager(t)
		defer cleanup()

		ctx := context.Background()
		err := vm.EnsureTable(ctx)
		if err != nil {
			t.Fatalf("EnsureTable() error = %v", err)
		}

		// Verify table exists by querying it
		_, err = vm.GetApplied(ctx)
		if err != nil {
			t.Fatalf("GetApplied() after EnsureTable() error = %v", err)
		}
	})

	t.Run("idempotent", func(t *testing.T) {
		vm, cleanup := setupVersionManager(t)
		defer cleanup()

		ctx := context.Background()

		// Call twice - should not error
		if err := vm.EnsureTable(ctx); err != nil {
			t.Fatalf("EnsureTable() first call error = %v", err)
		}
		if err := vm.EnsureTable(ctx); err != nil {
			t.Fatalf("EnsureTable() second call error = %v", err)
		}
	})
}

// -----------------------------------------------------------------------------
// RecordApplied / GetApplied Tests
// -----------------------------------------------------------------------------

func TestRecordAndGetApplied(t *testing.T) {
	vm, cleanup := setupVersionManager(t)
	defer cleanup()

	ctx := context.Background()
	if err := vm.EnsureTable(ctx); err != nil {
		t.Fatalf("EnsureTable() error = %v", err)
	}

	t.Run("record_migration", func(t *testing.T) {
		err := vm.RecordApplied(ctx, "001_create_users", "abc123", 100*time.Millisecond)
		if err != nil {
			t.Fatalf("RecordApplied() error = %v", err)
		}

		migrations, err := vm.GetApplied(ctx)
		if err != nil {
			t.Fatalf("GetApplied() error = %v", err)
		}

		if len(migrations) != 1 {
			t.Fatalf("GetApplied() = %d migrations, want 1", len(migrations))
		}

		m := migrations[0]
		if m.Revision != "001_create_users" {
			t.Errorf("Revision = %q, want %q", m.Revision, "001_create_users")
		}
		if m.Checksum != "abc123" {
			t.Errorf("Checksum = %q, want %q", m.Checksum, "abc123")
		}
		if m.ExecTimeMs != 100 {
			t.Errorf("ExecTimeMs = %d, want 100", m.ExecTimeMs)
		}
	})

	t.Run("multiple_migrations_ordered", func(t *testing.T) {
		err := vm.RecordApplied(ctx, "002_create_posts", "def456", 50*time.Millisecond)
		if err != nil {
			t.Fatalf("RecordApplied() error = %v", err)
		}

		migrations, err := vm.GetApplied(ctx)
		if err != nil {
			t.Fatalf("GetApplied() error = %v", err)
		}

		if len(migrations) != 2 {
			t.Fatalf("GetApplied() = %d migrations, want 2", len(migrations))
		}

		// Should be ordered by revision
		if migrations[0].Revision != "001_create_users" {
			t.Errorf("First migration = %q, want %q", migrations[0].Revision, "001_create_users")
		}
		if migrations[1].Revision != "002_create_posts" {
			t.Errorf("Second migration = %q, want %q", migrations[1].Revision, "002_create_posts")
		}
	})
}

// -----------------------------------------------------------------------------
// IsApplied Tests
// -----------------------------------------------------------------------------

func TestIsApplied(t *testing.T) {
	vm, cleanup := setupVersionManager(t)
	defer cleanup()

	ctx := context.Background()
	if err := vm.EnsureTable(ctx); err != nil {
		t.Fatalf("EnsureTable() error = %v", err)
	}

	// Record a migration
	if err := vm.RecordApplied(ctx, "001_test", "checksum", time.Second); err != nil {
		t.Fatalf("RecordApplied() error = %v", err)
	}

	t.Run("applied_migration", func(t *testing.T) {
		applied, err := vm.IsApplied(ctx, "001_test")
		if err != nil {
			t.Fatalf("IsApplied() error = %v", err)
		}
		if !applied {
			t.Error("IsApplied() = false, want true")
		}
	})

	t.Run("not_applied_migration", func(t *testing.T) {
		applied, err := vm.IsApplied(ctx, "002_nonexistent")
		if err != nil {
			t.Fatalf("IsApplied() error = %v", err)
		}
		if applied {
			t.Error("IsApplied() = true, want false")
		}
	})
}

// -----------------------------------------------------------------------------
// GetChecksum / VerifyChecksum Tests
// -----------------------------------------------------------------------------

func TestGetChecksum(t *testing.T) {
	vm, cleanup := setupVersionManager(t)
	defer cleanup()

	ctx := context.Background()
	if err := vm.EnsureTable(ctx); err != nil {
		t.Fatalf("EnsureTable() error = %v", err)
	}

	// Record a migration
	if err := vm.RecordApplied(ctx, "001_test", "expected_checksum", time.Second); err != nil {
		t.Fatalf("RecordApplied() error = %v", err)
	}

	t.Run("existing_migration", func(t *testing.T) {
		checksum, err := vm.GetChecksum(ctx, "001_test")
		if err != nil {
			t.Fatalf("GetChecksum() error = %v", err)
		}
		if checksum != "expected_checksum" {
			t.Errorf("GetChecksum() = %q, want %q", checksum, "expected_checksum")
		}
	})

	t.Run("nonexistent_migration", func(t *testing.T) {
		_, err := vm.GetChecksum(ctx, "nonexistent")
		if err == nil {
			t.Fatal("GetChecksum() expected error for nonexistent migration")
		}

		code := alerr.GetErrorCode(err)
		if code != alerr.ErrMigrationNotFound {
			t.Errorf("GetChecksum() error code = %v, want %v", code, alerr.ErrMigrationNotFound)
		}
	})
}

func TestVerifyChecksum(t *testing.T) {
	vm, cleanup := setupVersionManager(t)
	defer cleanup()

	ctx := context.Background()
	if err := vm.EnsureTable(ctx); err != nil {
		t.Fatalf("EnsureTable() error = %v", err)
	}

	// Record a migration
	if err := vm.RecordApplied(ctx, "001_test", "checksum123", time.Second); err != nil {
		t.Fatalf("RecordApplied() error = %v", err)
	}

	t.Run("matching_checksum", func(t *testing.T) {
		err := vm.VerifyChecksum(ctx, "001_test", "checksum123")
		if err != nil {
			t.Errorf("VerifyChecksum() error = %v, want nil for matching checksum", err)
		}
	})

	t.Run("mismatched_checksum", func(t *testing.T) {
		err := vm.VerifyChecksum(ctx, "001_test", "different_checksum")
		if err == nil {
			t.Fatal("VerifyChecksum() expected error for mismatched checksum")
		}

		code := alerr.GetErrorCode(err)
		if code != alerr.ErrMigrationChecksum {
			t.Errorf("VerifyChecksum() error code = %v, want %v", code, alerr.ErrMigrationChecksum)
		}
	})
}

// -----------------------------------------------------------------------------
// RecordRollback Tests
// -----------------------------------------------------------------------------

func TestRecordRollback(t *testing.T) {
	vm, cleanup := setupVersionManager(t)
	defer cleanup()

	ctx := context.Background()
	if err := vm.EnsureTable(ctx); err != nil {
		t.Fatalf("EnsureTable() error = %v", err)
	}

	// Record a migration
	if err := vm.RecordApplied(ctx, "001_test", "checksum", time.Second); err != nil {
		t.Fatalf("RecordApplied() error = %v", err)
	}

	t.Run("rollback_existing", func(t *testing.T) {
		err := vm.RecordRollback(ctx, "001_test")
		if err != nil {
			t.Fatalf("RecordRollback() error = %v", err)
		}

		// Verify migration is removed
		applied, err := vm.IsApplied(ctx, "001_test")
		if err != nil {
			t.Fatalf("IsApplied() error = %v", err)
		}
		if applied {
			t.Error("IsApplied() = true after rollback, want false")
		}
	})

	t.Run("rollback_nonexistent", func(t *testing.T) {
		err := vm.RecordRollback(ctx, "nonexistent")
		if err == nil {
			t.Fatal("RecordRollback() expected error for nonexistent migration")
		}

		code := alerr.GetErrorCode(err)
		if code != alerr.ErrMigrationNotFound {
			t.Errorf("RecordRollback() error code = %v, want %v", code, alerr.ErrMigrationNotFound)
		}
	})
}

// -----------------------------------------------------------------------------
// Lock Table Tests
// -----------------------------------------------------------------------------

func TestEnsureLockTable(t *testing.T) {
	vm, cleanup := setupVersionManager(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("creates_table", func(t *testing.T) {
		err := vm.EnsureLockTable(ctx)
		if err != nil {
			t.Fatalf("EnsureLockTable() error = %v", err)
		}

		// Verify we can get lock info
		info, err := vm.GetLockInfo(ctx)
		if err != nil {
			t.Fatalf("GetLockInfo() error = %v", err)
		}
		if info == nil {
			t.Fatal("GetLockInfo() returned nil")
		}
		if info.Locked {
			t.Error("Initial lock state should be unlocked")
		}
	})

	t.Run("idempotent", func(t *testing.T) {
		// Call twice - should not error
		if err := vm.EnsureLockTable(ctx); err != nil {
			t.Fatalf("EnsureLockTable() second call error = %v", err)
		}
	})
}

// -----------------------------------------------------------------------------
// AcquireLock / ReleaseLock Tests
// -----------------------------------------------------------------------------

func TestAcquireAndReleaseLock(t *testing.T) {
	vm, cleanup := setupVersionManager(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("acquire_lock", func(t *testing.T) {
		err := vm.AcquireLock(ctx, 5*time.Second)
		if err != nil {
			t.Fatalf("AcquireLock() error = %v", err)
		}

		// Verify lock is held
		info, err := vm.GetLockInfo(ctx)
		if err != nil {
			t.Fatalf("GetLockInfo() error = %v", err)
		}
		if !info.Locked {
			t.Error("Lock should be acquired")
		}
		if info.LockedBy == "" {
			t.Error("LockedBy should be set")
		}
	})

	t.Run("release_lock", func(t *testing.T) {
		err := vm.ReleaseLock(ctx)
		if err != nil {
			t.Fatalf("ReleaseLock() error = %v", err)
		}

		// Verify lock is released
		info, err := vm.GetLockInfo(ctx)
		if err != nil {
			t.Fatalf("GetLockInfo() error = %v", err)
		}
		if info.Locked {
			t.Error("Lock should be released")
		}
	})
}

func TestIsLocked(t *testing.T) {
	vm, cleanup := setupVersionManager(t)
	defer cleanup()

	ctx := context.Background()

	// Ensure lock table
	if err := vm.EnsureLockTable(ctx); err != nil {
		t.Fatalf("EnsureLockTable() error = %v", err)
	}

	t.Run("not_locked", func(t *testing.T) {
		locked, err := vm.IsLocked(ctx)
		if err != nil {
			t.Fatalf("IsLocked() error = %v", err)
		}
		if locked {
			t.Error("IsLocked() = true, want false")
		}
	})

	t.Run("locked", func(t *testing.T) {
		if err := vm.AcquireLock(ctx, time.Second); err != nil {
			t.Fatalf("AcquireLock() error = %v", err)
		}
		defer vm.ReleaseLock(ctx)

		locked, err := vm.IsLocked(ctx)
		if err != nil {
			t.Fatalf("IsLocked() error = %v", err)
		}
		if !locked {
			t.Error("IsLocked() = false, want true")
		}
	})
}

func TestForceReleaseLock(t *testing.T) {
	vm, cleanup := setupVersionManager(t)
	defer cleanup()

	ctx := context.Background()

	// Acquire lock
	if err := vm.AcquireLock(ctx, time.Second); err != nil {
		t.Fatalf("AcquireLock() error = %v", err)
	}

	// Force release
	err := vm.ForceReleaseLock(ctx)
	if err != nil {
		t.Fatalf("ForceReleaseLock() error = %v", err)
	}

	// Verify unlocked
	locked, err := vm.IsLocked(ctx)
	if err != nil {
		t.Fatalf("IsLocked() error = %v", err)
	}
	if locked {
		t.Error("Lock should be released after ForceReleaseLock")
	}
}

func TestGetLockInfo(t *testing.T) {
	vm, cleanup := setupVersionManager(t)
	defer cleanup()

	ctx := context.Background()

	// Ensure lock table
	if err := vm.EnsureLockTable(ctx); err != nil {
		t.Fatalf("EnsureLockTable() error = %v", err)
	}

	t.Run("unlocked", func(t *testing.T) {
		info, err := vm.GetLockInfo(ctx)
		if err != nil {
			t.Fatalf("GetLockInfo() error = %v", err)
		}
		if info == nil {
			t.Fatal("GetLockInfo() returned nil")
		}
		if info.Locked {
			t.Error("Lock should not be held")
		}
	})

	t.Run("locked", func(t *testing.T) {
		if err := vm.AcquireLock(ctx, time.Second); err != nil {
			t.Fatalf("AcquireLock() error = %v", err)
		}
		defer vm.ReleaseLock(ctx)

		info, err := vm.GetLockInfo(ctx)
		if err != nil {
			t.Fatalf("GetLockInfo() error = %v", err)
		}
		if info == nil {
			t.Fatal("GetLockInfo() returned nil")
		}
		if !info.Locked {
			t.Error("Lock should be held")
		}
		if info.LockedBy == "" {
			t.Error("LockedBy should be set")
		}
		if info.LockedAt == nil {
			t.Error("LockedAt should be set")
		}
	})
}

// -----------------------------------------------------------------------------
// Lock Timeout Tests
// -----------------------------------------------------------------------------

func TestAcquireLockTimeout(t *testing.T) {
	// Create two version managers with the same database
	db, d := setupTestDB(t)
	defer db.Close()

	vm1 := NewVersionManager(db, d)
	vm2 := NewVersionManager(db, d)

	ctx := context.Background()

	// First manager acquires lock
	if err := vm1.AcquireLock(ctx, time.Second); err != nil {
		t.Fatalf("vm1.AcquireLock() error = %v", err)
	}

	// Second manager should timeout trying to acquire
	err := vm2.AcquireLock(ctx, 100*time.Millisecond)
	if err == nil {
		t.Fatal("vm2.AcquireLock() expected timeout error")
	}

	code := alerr.GetErrorCode(err)
	if code != alerr.ErrMigrationConflict {
		t.Errorf("AcquireLock() error code = %v, want %v", code, alerr.ErrMigrationConflict)
	}

	// Release first lock
	if err := vm1.ReleaseLock(ctx); err != nil {
		t.Fatalf("vm1.ReleaseLock() error = %v", err)
	}

	// Now second manager should be able to acquire
	if err := vm2.AcquireLock(ctx, time.Second); err != nil {
		t.Fatalf("vm2.AcquireLock() after release error = %v", err)
	}
	defer vm2.ReleaseLock(ctx)
}
