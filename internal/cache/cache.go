package cache

import (
	"database/sql"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/hlop3z/astroladb/internal/alerr"
	"github.com/hlop3z/astroladb/internal/drift"
	"github.com/hlop3z/astroladb/internal/engine"

	_ "modernc.org/sqlite" // SQLite driver
)

const (
	// CacheDir is the directory name for the cache (gitignored).
	CacheDir = ".alab"
	// CacheFile is the SQLite database file name.
	CacheFile = "cache.db"
)

// Cache provides local caching of schema snapshots and merkle hashes.
// The cache is stored in .alab/cache.db (SQLite) and is gitignored.
// It is optional and can always be rebuilt from migration files.
type Cache struct {
	db   *sql.DB
	path string
	mu   sync.RWMutex
}

// Open opens or creates the cache database at the given project root.
// If the cache directory or database does not exist, they are created.
// Returns nil cache and no error if caching is disabled.
func Open(projectRoot string) (*Cache, error) {
	cacheDir := filepath.Join(projectRoot, CacheDir)
	cachePath := filepath.Join(cacheDir, CacheFile)

	// Create cache directory if it doesn't exist
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return nil, alerr.Wrap(alerr.ErrCacheInit, err, "failed to create cache directory").
			With("path", cacheDir)
	}

	// Open SQLite database
	db, err := sql.Open("sqlite", cachePath)
	if err != nil {
		return nil, alerr.Wrap(alerr.ErrCacheInit, err, "failed to open cache database").
			With("path", cachePath)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, alerr.Wrap(alerr.ErrCacheInit, err, "failed to connect to cache database").
			With("path", cachePath)
	}

	cache := &Cache{
		db:   db,
		path: cachePath,
	}

	// Initialize schema
	if err := cache.initSchema(); err != nil {
		db.Close()
		return nil, err
	}

	return cache, nil
}

// Close closes the cache database connection.
func (c *Cache) Close() error {
	if c == nil || c.db == nil {
		return nil
	}
	return c.db.Close()
}

// Path returns the path to the cache database file.
func (c *Cache) Path() string {
	if c == nil {
		return ""
	}
	return c.path
}

// initSchema creates the cache database tables if they don't exist.
func (c *Cache) initSchema() error {
	schema := `
		-- Schema snapshots per revision
		CREATE TABLE IF NOT EXISTS schema_snapshots (
			revision     TEXT PRIMARY KEY,
			schema_json  TEXT NOT NULL,
			created_at   TEXT NOT NULL
		);

		-- Merkle tree hashes per revision
		CREATE TABLE IF NOT EXISTS merkle_hashes (
			revision     TEXT PRIMARY KEY,
			hash_json    TEXT NOT NULL,
			root_hash    TEXT NOT NULL,
			created_at   TEXT NOT NULL
		);

		-- Migration metadata
		CREATE TABLE IF NOT EXISTS migration_metadata (
			revision     TEXT PRIMARY KEY,
			checksum     TEXT NOT NULL,
			file_path    TEXT NOT NULL,
			file_size    INTEGER NOT NULL,
			created_at   TEXT NOT NULL
		);

		-- Cache metadata (version, etc.)
		CREATE TABLE IF NOT EXISTS cache_meta (
			key   TEXT PRIMARY KEY,
			value TEXT NOT NULL
		);

		-- Set cache version
		INSERT OR REPLACE INTO cache_meta (key, value) VALUES ('version', '1');
	`

	c.mu.Lock()
	defer c.mu.Unlock()

	if _, err := c.db.Exec(schema); err != nil {
		return alerr.Wrap(alerr.ErrCacheInit, err, "failed to initialize cache schema")
	}

	return nil
}

// -----------------------------------------------------------------------------
// Schema Snapshot Operations
// -----------------------------------------------------------------------------

// GetSchemaSnapshot retrieves the schema snapshot for a given revision.
// Returns nil if not found.
func (c *Cache) GetSchemaSnapshot(revision string) (*engine.Schema, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var schemaJSON string
	err := c.db.QueryRow(
		"SELECT schema_json FROM schema_snapshots WHERE revision = ?",
		revision,
	).Scan(&schemaJSON)

	if err == sql.ErrNoRows {
		return nil, nil // Not found
	}
	if err != nil {
		return nil, alerr.Wrap(alerr.ErrCacheRead, err, "failed to read schema snapshot").
			With("revision", revision)
	}

	return DeserializeSchema([]byte(schemaJSON))
}

// SetSchemaSnapshot stores the schema snapshot for a given revision.
func (c *Cache) SetSchemaSnapshot(revision string, schema *engine.Schema) error {
	data, err := SerializeSchema(schema)
	if err != nil {
		return err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	_, err = c.db.Exec(
		"INSERT OR REPLACE INTO schema_snapshots (revision, schema_json, created_at) VALUES (?, ?, ?)",
		revision, string(data), time.Now().UTC().Format(time.RFC3339),
	)
	if err != nil {
		return alerr.Wrap(alerr.ErrCacheWrite, err, "failed to write schema snapshot").
			With("revision", revision)
	}

	return nil
}

// DeleteSchemaSnapshot removes the schema snapshot for a given revision.
func (c *Cache) DeleteSchemaSnapshot(revision string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	_, err := c.db.Exec("DELETE FROM schema_snapshots WHERE revision = ?", revision)
	if err != nil {
		return alerr.Wrap(alerr.ErrCacheWrite, err, "failed to delete schema snapshot").
			With("revision", revision)
	}

	return nil
}

// ListSchemaSnapshots returns all revision IDs that have cached snapshots.
func (c *Cache) ListSchemaSnapshots() ([]string, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	rows, err := c.db.Query("SELECT revision FROM schema_snapshots ORDER BY revision")
	if err != nil {
		return nil, alerr.Wrap(alerr.ErrCacheRead, err, "failed to list schema snapshots")
	}
	defer rows.Close()

	var revisions []string
	for rows.Next() {
		var revision string
		if err := rows.Scan(&revision); err != nil {
			return nil, alerr.Wrap(alerr.ErrCacheRead, err, "failed to scan revision")
		}
		revisions = append(revisions, revision)
	}

	return revisions, rows.Err()
}

// -----------------------------------------------------------------------------
// Merkle Hash Operations
// -----------------------------------------------------------------------------

// GetMerkleHash retrieves the merkle hash for a given revision.
// Returns nil if not found.
func (c *Cache) GetMerkleHash(revision string) (*drift.SchemaHash, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var hashJSON string
	err := c.db.QueryRow(
		"SELECT hash_json FROM merkle_hashes WHERE revision = ?",
		revision,
	).Scan(&hashJSON)

	if err == sql.ErrNoRows {
		return nil, nil // Not found
	}
	if err != nil {
		return nil, alerr.Wrap(alerr.ErrCacheRead, err, "failed to read merkle hash").
			With("revision", revision)
	}

	return DeserializeSchemaHash([]byte(hashJSON))
}

// GetMerkleRootHash retrieves just the root hash for a given revision.
// Returns empty string if not found.
func (c *Cache) GetMerkleRootHash(revision string) (string, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var rootHash string
	err := c.db.QueryRow(
		"SELECT root_hash FROM merkle_hashes WHERE revision = ?",
		revision,
	).Scan(&rootHash)

	if err == sql.ErrNoRows {
		return "", nil // Not found
	}
	if err != nil {
		return "", alerr.Wrap(alerr.ErrCacheRead, err, "failed to read merkle root hash").
			With("revision", revision)
	}

	return rootHash, nil
}

// SetMerkleHash stores the merkle hash for a given revision.
func (c *Cache) SetMerkleHash(revision string, hash *drift.SchemaHash) error {
	data, err := SerializeSchemaHash(hash)
	if err != nil {
		return err
	}

	rootHash := ""
	if hash != nil {
		rootHash = hash.Root
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	_, err = c.db.Exec(
		"INSERT OR REPLACE INTO merkle_hashes (revision, hash_json, root_hash, created_at) VALUES (?, ?, ?, ?)",
		revision, string(data), rootHash, time.Now().UTC().Format(time.RFC3339),
	)
	if err != nil {
		return alerr.Wrap(alerr.ErrCacheWrite, err, "failed to write merkle hash").
			With("revision", revision)
	}

	return nil
}

// DeleteMerkleHash removes the merkle hash for a given revision.
func (c *Cache) DeleteMerkleHash(revision string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	_, err := c.db.Exec("DELETE FROM merkle_hashes WHERE revision = ?", revision)
	if err != nil {
		return alerr.Wrap(alerr.ErrCacheWrite, err, "failed to delete merkle hash").
			With("revision", revision)
	}

	return nil
}

// -----------------------------------------------------------------------------
// Migration Metadata Operations
// -----------------------------------------------------------------------------

// MigrationMeta contains metadata about a migration file.
type MigrationMeta struct {
	Revision  string
	Checksum  string
	FilePath  string
	FileSize  int64
	CreatedAt time.Time
}

// GetMigrationMeta retrieves metadata for a given revision.
// Returns nil if not found.
func (c *Cache) GetMigrationMeta(revision string) (*MigrationMeta, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var meta MigrationMeta
	var createdAt string
	err := c.db.QueryRow(
		"SELECT revision, checksum, file_path, file_size, created_at FROM migration_metadata WHERE revision = ?",
		revision,
	).Scan(&meta.Revision, &meta.Checksum, &meta.FilePath, &meta.FileSize, &createdAt)

	if err == sql.ErrNoRows {
		return nil, nil // Not found
	}
	if err != nil {
		return nil, alerr.Wrap(alerr.ErrCacheRead, err, "failed to read migration metadata").
			With("revision", revision)
	}

	meta.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	return &meta, nil
}

// SetMigrationMeta stores metadata for a given revision.
func (c *Cache) SetMigrationMeta(meta *MigrationMeta) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	_, err := c.db.Exec(
		"INSERT OR REPLACE INTO migration_metadata (revision, checksum, file_path, file_size, created_at) VALUES (?, ?, ?, ?, ?)",
		meta.Revision, meta.Checksum, meta.FilePath, meta.FileSize, time.Now().UTC().Format(time.RFC3339),
	)
	if err != nil {
		return alerr.Wrap(alerr.ErrCacheWrite, err, "failed to write migration metadata").
			With("revision", meta.Revision)
	}

	return nil
}

// DeleteMigrationMeta removes metadata for a given revision.
func (c *Cache) DeleteMigrationMeta(revision string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	_, err := c.db.Exec("DELETE FROM migration_metadata WHERE revision = ?", revision)
	if err != nil {
		return alerr.Wrap(alerr.ErrCacheWrite, err, "failed to delete migration metadata").
			With("revision", revision)
	}

	return nil
}

// ListMigrationMeta returns all cached migration metadata.
func (c *Cache) ListMigrationMeta() ([]*MigrationMeta, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	rows, err := c.db.Query(
		"SELECT revision, checksum, file_path, file_size, created_at FROM migration_metadata ORDER BY revision",
	)
	if err != nil {
		return nil, alerr.Wrap(alerr.ErrCacheRead, err, "failed to list migration metadata")
	}
	defer rows.Close()

	var metas []*MigrationMeta
	for rows.Next() {
		var meta MigrationMeta
		var createdAt string
		if err := rows.Scan(&meta.Revision, &meta.Checksum, &meta.FilePath, &meta.FileSize, &createdAt); err != nil {
			return nil, alerr.Wrap(alerr.ErrCacheRead, err, "failed to scan migration metadata")
		}
		meta.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		metas = append(metas, &meta)
	}

	return metas, rows.Err()
}

// -----------------------------------------------------------------------------
// Cache Management Operations
// -----------------------------------------------------------------------------

// Clear removes all cached data.
// This is equivalent to deleting and recreating the cache.
func (c *Cache) Clear() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	tables := []string{"schema_snapshots", "merkle_hashes", "migration_metadata"}
	for _, table := range tables {
		if _, err := c.db.Exec("DELETE FROM " + table); err != nil {
			return alerr.Wrap(alerr.ErrCacheWrite, err, "failed to clear cache").
				With("table", table)
		}
	}

	return nil
}

// DeleteRevision removes all cached data for a specific revision.
func (c *Cache) DeleteRevision(revision string) error {
	if err := c.DeleteSchemaSnapshot(revision); err != nil {
		return err
	}
	if err := c.DeleteMerkleHash(revision); err != nil {
		return err
	}
	if err := c.DeleteMigrationMeta(revision); err != nil {
		return err
	}
	return nil
}

// GetCacheVersion returns the cache schema version.
func (c *Cache) GetCacheVersion() (string, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var version string
	err := c.db.QueryRow("SELECT value FROM cache_meta WHERE key = 'version'").Scan(&version)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", alerr.Wrap(alerr.ErrCacheRead, err, "failed to read cache version")
	}
	return version, nil
}

// Stats returns cache statistics.
type Stats struct {
	SchemaSnapshots int
	MerkleHashes    int
	MigrationMetas  int
	DatabaseSize    int64
}

// GetStats returns statistics about the cache.
func (c *Cache) GetStats() (*Stats, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	stats := &Stats{}

	// Count schema snapshots
	if err := c.db.QueryRow("SELECT COUNT(*) FROM schema_snapshots").Scan(&stats.SchemaSnapshots); err != nil {
		return nil, alerr.Wrap(alerr.ErrCacheRead, err, "failed to count schema snapshots")
	}

	// Count merkle hashes
	if err := c.db.QueryRow("SELECT COUNT(*) FROM merkle_hashes").Scan(&stats.MerkleHashes); err != nil {
		return nil, alerr.Wrap(alerr.ErrCacheRead, err, "failed to count merkle hashes")
	}

	// Count migration metadata
	if err := c.db.QueryRow("SELECT COUNT(*) FROM migration_metadata").Scan(&stats.MigrationMetas); err != nil {
		return nil, alerr.Wrap(alerr.ErrCacheRead, err, "failed to count migration metadata")
	}

	// Get database file size
	if fi, err := os.Stat(c.path); err == nil {
		stats.DatabaseSize = fi.Size()
	}

	return stats, nil
}

// Vacuum compacts the database file.
func (c *Cache) Vacuum() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, err := c.db.Exec("VACUUM"); err != nil {
		return alerr.Wrap(alerr.ErrCacheWrite, err, "failed to vacuum cache database")
	}
	return nil
}

// -----------------------------------------------------------------------------
// Helper Functions
// -----------------------------------------------------------------------------

// Exists checks if a cache database exists at the given project root.
func Exists(projectRoot string) bool {
	cachePath := filepath.Join(projectRoot, CacheDir, CacheFile)
	_, err := os.Stat(cachePath)
	return err == nil
}

// Remove deletes the entire cache directory.
func Remove(projectRoot string) error {
	cacheDir := filepath.Join(projectRoot, CacheDir)
	if err := os.RemoveAll(cacheDir); err != nil {
		return alerr.Wrap(alerr.ErrCacheWrite, err, "failed to remove cache directory").
			With("path", cacheDir)
	}
	return nil
}

// GetOrCompute retrieves a schema from cache, or computes and caches it.
// The compute function is called if the schema is not in cache.
func (c *Cache) GetOrCompute(revision string, compute func() (*engine.Schema, error)) (*engine.Schema, error) {
	// Try cache first
	schema, err := c.GetSchemaSnapshot(revision)
	if err != nil {
		return nil, err
	}
	if schema != nil {
		return schema, nil
	}

	// Compute if not cached
	schema, err = compute()
	if err != nil {
		return nil, err
	}

	// Cache the result (ignore errors - cache is optional)
	_ = c.SetSchemaSnapshot(revision, schema)

	return schema, nil
}

// GetOrComputeHash retrieves a merkle hash from cache, or computes and caches it.
// The compute function is called if the hash is not in cache.
func (c *Cache) GetOrComputeHash(revision string, compute func() (*drift.SchemaHash, error)) (*drift.SchemaHash, error) {
	// Try cache first
	hash, err := c.GetMerkleHash(revision)
	if err != nil {
		return nil, err
	}
	if hash != nil {
		return hash, nil
	}

	// Compute if not cached
	hash, err = compute()
	if err != nil {
		return nil, err
	}

	// Cache the result (ignore errors - cache is optional)
	_ = c.SetMerkleHash(revision, hash)

	return hash, nil
}
