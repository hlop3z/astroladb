package main

import (
	"fmt"
	"os"

	"github.com/hlop3z/astroladb/pkg/astroladb"
	"gopkg.in/yaml.v3"
)

// Config represents the alab.yaml configuration file.
type Config struct {
	DatabaseURL   string `yaml:"database_url"`
	SchemasDir    string `yaml:"schemas_dir"`
	MigrationsDir string `yaml:"migrations_dir"`
	Dialect       string `yaml:"dialect"`
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
		// Handle env var interpolation in database_url
		cfg.DatabaseURL = expandEnvVars(cfg.DatabaseURL)
	}

	// Override with env vars
	if envURL := os.Getenv("DATABASE_URL"); envURL != "" && databaseURL == "" {
		cfg.DatabaseURL = envURL
	}
	if envSchemas := os.Getenv("ALAB_SCHEMAS_DIR"); envSchemas != "" && schemasDir == "./schemas" {
		cfg.SchemasDir = envSchemas
	}
	if envMigrations := os.Getenv("ALAB_MIGRATIONS_DIR"); envMigrations != "" && migrationsDir == "./migrations" {
		cfg.MigrationsDir = envMigrations
	}

	// Override with CLI flags (highest priority)
	if databaseURL != "" {
		cfg.DatabaseURL = databaseURL
	}
	if schemasDir != "" && schemasDir != "./schemas" {
		cfg.SchemasDir = schemasDir
	}
	if migrationsDir != "" && migrationsDir != "./migrations" {
		cfg.MigrationsDir = migrationsDir
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

	if cfg.DatabaseURL == "" {
		return nil, astroladb.ErrMissingDatabaseURL
	}

	opts := []astroladb.Option{
		astroladb.WithDatabaseURL(cfg.DatabaseURL),
		astroladb.WithSchemasDir(cfg.SchemasDir),
		astroladb.WithMigrationsDir(cfg.MigrationsDir),
	}

	if cfg.Dialect != "" {
		opts = append(opts, astroladb.WithDialect(cfg.Dialect))
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
