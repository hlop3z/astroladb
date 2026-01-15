package main

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"

	"github.com/hlop3z/astroladb/pkg/astroladb"
)

// DatabaseConfig represents the database section of alab.yaml.
type DatabaseConfig struct {
	Dialect string `yaml:"dialect"`
	URL     string `yaml:"url"`
}

// Config represents the alab.yaml configuration file.
type Config struct {
	Database      DatabaseConfig `yaml:"database"`
	SchemasDir    string         `yaml:"schemas"`
	MigrationsDir string         `yaml:"migrations"`
}

// loadConfig loads configuration from file, env vars, and CLI flags.
// Precedence: CLI flags > env vars > config file > defaults
func loadConfig() (*Config, error) {
	cfg := &Config{
		SchemasDir:    "./schemas",
		MigrationsDir: "./migrations",
	}

	// Load config file if it exists
	if data, err := os.ReadFile(configFile); err == nil {
		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("failed to parse config file: %w", err)
		}
		// Handle env var interpolation in database url
		cfg.Database.URL = expandEnvVars(cfg.Database.URL)
	}

	// Override with env vars
	if envURL := os.Getenv("DATABASE_URL"); envURL != "" && databaseURL == "" {
		cfg.Database.URL = envURL
	}
	if envSchemas := os.Getenv("ALAB_SCHEMAS_DIR"); envSchemas != "" {
		cfg.SchemasDir = envSchemas
	}
	if envMigrations := os.Getenv("ALAB_MIGRATIONS_DIR"); envMigrations != "" {
		cfg.MigrationsDir = envMigrations
	}

	// Override with CLI flag (highest priority)
	if databaseURL != "" {
		cfg.Database.URL = databaseURL
	}

	return cfg, nil
}

// expandEnvVars expands ${VAR} patterns in a string.
func expandEnvVars(s string) string {
	return os.Expand(s, os.Getenv)
}

// newClient creates a new astroladb client from config.
// It returns enhanced errors that are suitable for direct display to users.
func newClient() (*astroladb.Client, error) {
	cfg, err := loadConfig()
	if err != nil {
		return nil, err
	}

	if cfg.Database.URL == "" {
		return nil, astroladb.ErrMissingDatabaseURL
	}

	opts := []astroladb.Option{
		astroladb.WithDatabaseURL(cfg.Database.URL),
		astroladb.WithSchemasDir(cfg.SchemasDir),
		astroladb.WithMigrationsDir(cfg.MigrationsDir),
	}

	if cfg.Database.Dialect != "" {
		opts = append(opts, astroladb.WithDialect(cfg.Database.Dialect))
	}

	client, err := astroladb.New(opts...)
	if err != nil {
		return nil, err
	}

	return client, nil
}

// newSchemaOnlyClient creates a client that only reads schema files.
// It does not require a database connection.
// Use for operations like export and check.
func newSchemaOnlyClient() (*astroladb.Client, error) {
	cfg, err := loadConfig()
	if err != nil {
		return nil, err
	}

	opts := []astroladb.Option{
		astroladb.WithSchemasDir(cfg.SchemasDir),
		astroladb.WithMigrationsDir(cfg.MigrationsDir),
		astroladb.WithSchemaOnly(),
	}

	return astroladb.New(opts...)
}
