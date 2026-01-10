// Package testutil provides test helpers for the Alab project.
//
// This package includes:
//   - Database setup functions for PostgreSQL and SQLite
//   - SQL assertion helpers for comparing and validating SQL statements
//   - Error assertion helpers for checking error codes
//   - Golden file testing support
//
// # Build Tags
//
// Database-specific tests can be run using build tags:
//
//	go test ./... -tags=integration  # Run all database tests
//	go test ./... -tags=postgres     # PostgreSQL only
//	go test ./... -tags=sqlite       # SQLite only
//
// # Database Setup
//
// Before running integration tests, start the test databases:
//
//	docker-compose -f docker-compose.test.yml up -d
//
// Then run tests:
//
//	go test ./... -tags=integration
//
// Stop databases when done:
//
//	docker-compose -f docker-compose.test.yml down
//
// # Environment Variables
//
// Database connection strings can be overridden:
//
//	POSTGRES_URL - PostgreSQL connection string
//
// Defaults (from docker-compose.test.yml):
//
//	PostgreSQL: postgres://alab:alab@localhost:5432/alab_test?sslmode=disable
//	SQLite:     :memory: (in-memory)
//
// # Golden Files
//
// Golden files are stored in testdata/ directory. Update them with:
//
//	go test ./... -update-golden
//
// # Example Usage
//
//	func TestMyFeature(t *testing.T) {
//	    db := testutil.SetupPostgres(t)
//
//	    // Create table
//	    testutil.ExecSQL(t, db, `CREATE TABLE users (id UUID PRIMARY KEY)`)
//
//	    // Assert table exists
//	    testutil.AssertTableExists(t, db, "users")
//
//	    // Assert SQL matches
//	    got := generateSQL()
//	    testutil.AssertSQL(t, got, "SELECT * FROM users")
//	}
package testutil
