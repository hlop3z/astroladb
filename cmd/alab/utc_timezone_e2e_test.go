//go:build integration

package main

import (
	"strings"
	"testing"
	"time"

	"github.com/hlop3z/astroladb/pkg/astroladb"
)

// -----------------------------------------------------------------------------
// UTC Timezone E2E Tests
// -----------------------------------------------------------------------------
// These tests verify that UTC timezone is always configured by default for
// both PostgreSQL and CockroachDB, ensuring consistent datetime handling.

// TestE2E_UTC_TimezoneConfiguration verifies that the connection timezone is set to UTC.
func TestE2E_UTC_TimezoneConfiguration(t *testing.T) {
	for _, setup := range getDatabaseSetups() {
		t.Run(setup.name, func(t *testing.T) {
			db, dbURL := setup.setupFunc(t)
			defer db.Close()

			// Create astroladb client (which should set timezone to UTC)
			client, err := astroladb.New(
				astroladb.WithDatabaseURL(dbURL),
				astroladb.WithSchemasDir(t.TempDir()),
				astroladb.WithMigrationsDir(t.TempDir()),
			)
			if err != nil {
				t.Fatalf("Failed to create client: %v", err)
			}
			defer client.Close()

			// Verify timezone is UTC using client's DB connection
			var timezone string
			err = client.DB().QueryRow("SHOW timezone").Scan(&timezone)
			if err != nil {
				t.Fatalf("Failed to query timezone: %v", err)
			}

			// Normalize timezone value (can be "UTC", "utc", or "Etc/UTC")
			timezone = strings.ToUpper(strings.TrimSpace(timezone))
			if timezone != "UTC" && !strings.Contains(timezone, "UTC") {
				t.Errorf("Expected timezone to be UTC, got: %s", timezone)
			}

			t.Logf("✓ Connection timezone is: %s", timezone)
		})
	}
}

// TestE2E_UTC_DatetimeStorage verifies that datetime values are stored and retrieved in UTC.
func TestE2E_UTC_DatetimeStorage(t *testing.T) {
	for _, setup := range getDatabaseSetups() {
		t.Run(setup.name, func(t *testing.T) {
			db, dbURL := setup.setupFunc(t)
			env := setupTestEnv(t)

			// Create migration with datetime columns
			env.writeMigration(t, "001", "create_events_table", testutil.SimpleMigration(
			`      m.create_table("test.datetime_test", t => {
        t.id()
        t.string("name", 100)
        t.datetime("scheduled_at")
      })`,
			`      m.drop_table("test.datetime_test")`,
		))

			// Apply migration
			client, err := astroladb.New(
				astroladb.WithDatabaseURL(dbURL),
				astroladb.WithSchemasDir(env.schemasDir),
				astroladb.WithMigrationsDir(env.migrationsDir),
			)
			if err != nil {
				t.Fatalf("Failed to create client: %v", err)
			}
			defer client.Close()

			if err := client.MigrationRun(); err != nil {
				t.Fatalf("Failed to run migrations: %v", err)
			}

			// Test 1: Insert a known UTC timestamp
			testTimestamp := "2026-01-11T12:00:00Z" // RFC 3339 UTC
			_, err = db.Exec(`
				INSERT INTO test_datetime_test (id, name, scheduled_at)
				VALUES (gen_random_uuid(), 'Test Event', $1)
			`, testTimestamp)
			if err != nil {
				t.Fatalf("Failed to insert test data: %v", err)
			}

			// Test 2: Retrieve and verify timestamp is in UTC
			var retrievedTimestamp time.Time
			var retrievedString string
			err = db.QueryRow(`
				SELECT scheduled_at, scheduled_at::text
				FROM test_datetime_test
				WHERE name = 'Test Event'
			`).Scan(&retrievedTimestamp, &retrievedString)
			if err != nil {
				t.Fatalf("Failed to query test data: %v", err)
			}

			// Verify timezone is UTC
			if retrievedTimestamp.Location().String() != "UTC" {
				t.Errorf("Expected timezone to be UTC, got: %s", retrievedTimestamp.Location())
			}

			// Verify time matches expected value (allowing for microsecond precision)
			expectedTime, _ := time.Parse(time.RFC3339, testTimestamp)
			if !retrievedTimestamp.Equal(expectedTime) {
				t.Errorf("Expected timestamp %v, got %v", expectedTime, retrievedTimestamp)
			}

			// Verify string representation ends with +00 (UTC offset)
			if !strings.Contains(retrievedString, "+00") {
				t.Errorf("Expected timestamp string to contain '+00' (UTC offset), got: %s", retrievedString)
			}

			t.Logf("✓ Stored timestamp: %s", testTimestamp)
			t.Logf("✓ Retrieved timestamp: %v (Location: %s)", retrievedTimestamp, retrievedTimestamp.Location())
			t.Logf("✓ String representation: %s", retrievedString)
		})
	}
}

// TestE2E_UTC_NowFunction verifies that NOW() and current timestamp functions return UTC time.
func TestE2E_UTC_NowFunction(t *testing.T) {
	for _, setup := range getDatabaseSetups() {
		t.Run(setup.name, func(t *testing.T) {
			db, dbURL := setup.setupFunc(t)
			defer db.Close()

			// Create astroladb client (sets timezone to UTC)
			client, err := astroladb.New(
				astroladb.WithDatabaseURL(dbURL),
				astroladb.WithSchemasDir(t.TempDir()),
				astroladb.WithMigrationsDir(t.TempDir()),
			)
			if err != nil {
				t.Fatalf("Failed to create client: %v", err)
			}
			defer client.Close()

			// Query NOW() and verify it's in UTC
			var now time.Time
			var nowString string
			err = client.DB().QueryRow("SELECT NOW(), NOW()::text").Scan(&now, &nowString)
			if err != nil {
				t.Fatalf("Failed to query NOW(): %v", err)
			}

			// Verify timezone is UTC
			if now.Location().String() != "UTC" {
				t.Errorf("Expected NOW() timezone to be UTC, got: %s", now.Location())
			}

			// Verify string representation shows UTC offset (+00)
			if !strings.Contains(nowString, "+00") {
				t.Errorf("Expected NOW() string to contain '+00' (UTC offset), got: %s", nowString)
			}

			// Verify NOW() is close to actual current time in UTC
			actualNow := time.Now().UTC()
			diff := actualNow.Sub(now)
			if diff < 0 {
				diff = -diff
			}
			if diff > 5*time.Second {
				t.Errorf("NOW() differs from actual time by %v (expected < 5s)", diff)
			}

			t.Logf("✓ NOW() returned: %v (Location: %s)", now, now.Location())
			t.Logf("✓ String representation: %s", nowString)
			t.Logf("✓ Difference from actual UTC time: %v", diff)
		})
	}
}

// TestE2E_UTC_TimestampOperations verifies datetime arithmetic and comparisons work correctly in UTC.
func TestE2E_UTC_TimestampOperations(t *testing.T) {
	for _, setup := range getDatabaseSetups() {
		t.Run(setup.name, func(t *testing.T) {
			db, dbURL := setup.setupFunc(t)
			env := setupTestEnv(t)

			// Create migration
			env.writeMigration(t, "001", "create_test_table", testutil.SimpleMigration(
			`      m.create_table("test.time_ops", t => {
        t.id()
        t.datetime("start_time")
        t.datetime("end_time")
      })`,
			`      m.drop_table("test.time_ops")`,
		))

			// Apply migration
			client, err := astroladb.New(
				astroladb.WithDatabaseURL(dbURL),
				astroladb.WithSchemasDir(env.schemasDir),
				astroladb.WithMigrationsDir(env.migrationsDir),
			)
			if err != nil {
				t.Fatalf("Failed to create client: %v", err)
			}
			defer client.Close()

			if err := client.MigrationRun(); err != nil {
				t.Fatalf("Failed to run migrations: %v", err)
			}

			// Insert test data with time range
			startTime := "2026-01-11T10:00:00Z"
			endTime := "2026-01-11T14:00:00Z"
			_, err = db.Exec(`
				INSERT INTO test_time_ops (id, start_time, end_time)
				VALUES (gen_random_uuid(), $1, $2)
			`, startTime, endTime)
			if err != nil {
				t.Fatalf("Failed to insert test data: %v", err)
			}

			// Test 1: Interval calculation (should be 4 hours)
			var intervalHours float64
			err = db.QueryRow(`
				SELECT EXTRACT(EPOCH FROM (end_time - start_time)) / 3600
				FROM test_time_ops
			`).Scan(&intervalHours)
			if err != nil {
				t.Fatalf("Failed to query interval: %v", err)
			}

			expectedHours := 4.0
			if intervalHours != expectedHours {
				t.Errorf("Expected interval of %v hours, got %v", expectedHours, intervalHours)
			}

			t.Logf("✓ Interval calculation: %v hours", intervalHours)

			// Test 2: Add interval to timestamp
			var futureTime time.Time
			err = db.QueryRow(`
				SELECT start_time + INTERVAL '2 hours'
				FROM test_time_ops
			`).Scan(&futureTime)
			if err != nil {
				t.Fatalf("Failed to query future time: %v", err)
			}

			// Verify it's in UTC
			if futureTime.Location().String() != "UTC" {
				t.Errorf("Expected future time timezone to be UTC, got: %s", futureTime.Location())
			}

			expectedFuture, _ := time.Parse(time.RFC3339, "2026-01-11T12:00:00Z")
			if !futureTime.Equal(expectedFuture) {
				t.Errorf("Expected future time %v, got %v", expectedFuture, futureTime)
			}

			t.Logf("✓ Future time (start + 2h): %v", futureTime)

			// Test 3: Compare timestamps
			var isBefore bool
			err = db.QueryRow(`
				SELECT start_time < end_time
				FROM test_time_ops
			`).Scan(&isBefore)
			if err != nil {
				t.Fatalf("Failed to query comparison: %v", err)
			}

			if !isBefore {
				t.Errorf("Expected start_time < end_time to be true")
			}

			t.Logf("✓ Timestamp comparison works correctly")
		})
	}
}

// TestE2E_UTC_CrossTimezoneConsistency verifies that UTC storage is independent of server timezone.
func TestE2E_UTC_CrossTimezoneConsistency(t *testing.T) {
	for _, setup := range getDatabaseSetups() {
		t.Run(setup.name, func(t *testing.T) {
			db, dbURL := setup.setupFunc(t)
			defer db.Close()

			// Create two separate clients (simulating different connections)
			client1, err := astroladb.New(
				astroladb.WithDatabaseURL(dbURL),
				astroladb.WithSchemasDir(t.TempDir()),
				astroladb.WithMigrationsDir(t.TempDir()),
			)
			if err != nil {
				t.Fatalf("Failed to create client1: %v", err)
			}
			defer client1.Close()

			client2, err := astroladb.New(
				astroladb.WithDatabaseURL(dbURL),
				astroladb.WithSchemasDir(t.TempDir()),
				astroladb.WithMigrationsDir(t.TempDir()),
			)
			if err != nil {
				t.Fatalf("Failed to create client2: %v", err)
			}
			defer client2.Close()

			// Create temp table for test
			_, err = db.Exec(`
				CREATE TABLE IF NOT EXISTS test_tz_consistency (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					test_time TIMESTAMPTZ NOT NULL
				)
			`)
			if err != nil {
				t.Fatalf("Failed to create test table: %v", err)
			}
			defer db.Exec("DROP TABLE IF EXISTS test_tz_consistency")

			// Client 1: Insert a timestamp
			testTime := "2026-01-11T15:30:00Z"
			_, err = client1.DB().Exec(`
				INSERT INTO test_tz_consistency (test_time) VALUES ($1)
			`, testTime)
			if err != nil {
				t.Fatalf("Client1 failed to insert: %v", err)
			}

			// Client 2: Read the same timestamp
			var retrieved time.Time
			err = client2.DB().QueryRow(`
				SELECT test_time FROM test_tz_consistency
			`).Scan(&retrieved)
			if err != nil {
				t.Fatalf("Client2 failed to query: %v", err)
			}

			// Verify consistency
			expected, _ := time.Parse(time.RFC3339, testTime)
			if !retrieved.Equal(expected) {
				t.Errorf("Expected %v, got %v", expected, retrieved)
			}

			// Verify both clients see UTC
			if retrieved.Location().String() != "UTC" {
				t.Errorf("Expected timezone to be UTC, got: %s", retrieved.Location())
			}

			t.Logf("✓ Client 1 inserted: %s", testTime)
			t.Logf("✓ Client 2 retrieved: %v (Location: %s)", retrieved, retrieved.Location())
			t.Logf("✓ Values are consistent across connections")
		})
	}
}
