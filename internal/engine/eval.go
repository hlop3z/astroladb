// Package engine provides the core evaluation and diffing engine for Alab.
// It handles evaluating schema files, merging multi-namespace schemas,
// and computing the diff between schema versions to generate migrations.
package engine

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/hlop3z/astroladb/internal/alerr"
	"github.com/hlop3z/astroladb/internal/ast"
	"github.com/hlop3z/astroladb/internal/metadata"
	"github.com/hlop3z/astroladb/internal/registry"
	"github.com/hlop3z/astroladb/internal/runtime"
)

// Evaluator evaluates schema files and converts them to AST representations.
// It uses the Goja JavaScript runtime sandbox for secure execution.
type Evaluator struct {
	sandbox  *runtime.Sandbox
	registry *registry.ModelRegistry
}

// NewEvaluator creates a new Evaluator with the given registry.
// If registry is nil, a new empty registry is created.
func NewEvaluator(reg *registry.ModelRegistry) *Evaluator {
	if reg == nil {
		reg = registry.NewModelRegistry()
	}
	return &Evaluator{
		sandbox:  runtime.NewSandbox(reg),
		registry: reg,
	}
}

// EvalFile evaluates a single schema file and returns the table definition.
// The namespace and table name are inferred from the file path:
//   - schemas/auth/users.js -> namespace="auth", table="users"
//   - schemas/blog/posts.js -> namespace="blog", table="posts"
func (e *Evaluator) EvalFile(path string) (*ast.TableDef, error) {
	// Infer namespace and table from path
	namespace, tableName, err := inferFromPath(path)
	if err != nil {
		return nil, err
	}

	// Evaluate the schema using EvalSchemaFile for rich error context
	tableDef, err := e.sandbox.EvalSchemaFile(path, namespace, tableName)
	if err != nil {
		// Error already has file context from EvalSchemaFile
		return nil, err
	}

	return tableDef, nil
}

// EvalFileWithMeta evaluates a schema file with explicit namespace and table name.
// This is useful when the namespace/table cannot be inferred from the path.
func (e *Evaluator) EvalFileWithMeta(path, namespace, tableName string) (*ast.TableDef, error) {
	// Evaluate the schema using EvalSchemaFile for rich error context
	tableDef, err := e.sandbox.EvalSchemaFile(path, namespace, tableName)
	if err != nil {
		// Error already has file context from EvalSchemaFile
		return nil, err
	}

	return tableDef, nil
}

// EvalCode evaluates schema code directly (not from a file).
// Namespace and table name must be provided explicitly.
func (e *Evaluator) EvalCode(code, namespace, tableName string) (*ast.TableDef, error) {
	tableDef, err := e.sandbox.EvalSchema(code, namespace, tableName)
	if err != nil {
		return nil, alerr.Wrap(alerr.ErrSchemaInvalid, err, "failed to evaluate schema code").
			WithTable(namespace, tableName)
	}
	return tableDef, nil
}

// EvalDir evaluates all schema files in a directory and returns the table definitions.
// The directory structure should be:
//
//	schemas/
//	├── <namespace>/
//	│   └── <table>.js
//
// This function auto-discovers namespaces (directories) and tables (*.js files).
func (e *Evaluator) EvalDir(dir string) ([]*ast.TableDef, error) {
	// Reset metadata for fresh evaluation
	e.sandbox.ClearMetadata()

	// Check if directory exists
	info, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, alerr.New(alerr.ErrSchemaNotFound, "schema directory does not exist").
				With("path", dir)
		}
		return nil, alerr.Wrap(alerr.ErrSchemaNotFound, err, "failed to access schema directory").
			With("path", dir)
	}
	if !info.IsDir() {
		return nil, alerr.New(alerr.ErrSchemaInvalid, "path is not a directory").
			With("path", dir)
	}

	var tables []*ast.TableDef
	var evalErrors []error

	// Walk the directory
	err = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories (we'll process files directly)
		if info.IsDir() {
			return nil
		}

		// Only process .js files
		if !strings.HasSuffix(info.Name(), ".js") {
			return nil
		}

		// Evaluate the schema file
		tableDef, err := e.EvalFile(path)
		if err != nil {
			evalErrors = append(evalErrors, err)
			return nil // Continue processing other files
		}

		tables = append(tables, tableDef)
		return nil
	})

	if err != nil {
		return nil, alerr.Wrap(alerr.ErrSchemaNotFound, err, "failed to walk schema directory").
			With("path", dir)
	}

	// If there were any evaluation errors, report them
	if len(evalErrors) > 0 {
		// Combine all errors into one
		var errMsgs []string
		for _, e := range evalErrors {
			errMsgs = append(errMsgs, e.Error())
		}
		return tables, alerr.New(alerr.ErrSchemaInvalid, "some schema files failed to evaluate").
			With("errors", strings.Join(errMsgs, "; ")).
			With("failed_count", len(evalErrors)).
			With("success_count", len(tables))
	}

	// Sort tables by qualified name for deterministic output
	sort.Slice(tables, func(i, j int) bool {
		return tables[i].QualifiedName() < tables[j].QualifiedName()
	})

	return tables, nil
}

// EvalDirStrict evaluates all schema files and returns an error if any fail.
// Unlike EvalDir, this function stops at the first error.
func (e *Evaluator) EvalDirStrict(dir string) ([]*ast.TableDef, error) {
	// Reset metadata for fresh evaluation (important for determinism check)
	e.sandbox.ClearMetadata()

	info, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, alerr.New(alerr.ErrSchemaNotFound, "schema directory does not exist").
				With("path", dir)
		}
		return nil, alerr.Wrap(alerr.ErrSchemaNotFound, err, "failed to access schema directory").
			With("path", dir)
	}
	if !info.IsDir() {
		return nil, alerr.New(alerr.ErrSchemaInvalid, "path is not a directory").
			With("path", dir)
	}

	var tables []*ast.TableDef

	err = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		if !strings.HasSuffix(info.Name(), ".js") {
			return nil
		}

		tableDef, err := e.EvalFile(path)
		if err != nil {
			return err
		}

		tables = append(tables, tableDef)
		return nil
	})

	if err != nil {
		return nil, err
	}

	sort.Slice(tables, func(i, j int) bool {
		return tables[i].QualifiedName() < tables[j].QualifiedName()
	})

	return tables, nil
}

// Registry returns the underlying model registry.
func (e *Evaluator) Registry() *registry.ModelRegistry {
	return e.registry
}

// Metadata returns the schema metadata collected during evaluation.
// This includes many_to_many relationships and polymorphic mappings.
func (e *Evaluator) Metadata() *metadata.Metadata {
	return e.sandbox.Metadata()
}

// GetJoinTables returns all auto-generated join table definitions.
// These are created from many_to_many relationships defined in schemas.
func (e *Evaluator) GetJoinTables() []*ast.TableDef {
	return e.sandbox.GetJoinTables()
}

// SaveMetadata saves the metadata to the .alab directory.
func (e *Evaluator) SaveMetadata(projectDir string) error {
	return e.sandbox.SaveMetadata(projectDir)
}

// inferFromPath extracts namespace and table name from a schema file path.
// Expected format: .../schemas/<namespace>/<table>.js
func inferFromPath(path string) (namespace, tableName string, err error) {
	// Normalize path separators
	path = filepath.ToSlash(path)

	// Get the file name without extension
	baseName := filepath.Base(path)
	if !strings.HasSuffix(baseName, ".js") {
		return "", "", alerr.New(alerr.ErrSchemaInvalid, "schema file must have .js extension").
			WithFile(path, 0)
	}
	tableName = strings.TrimSuffix(baseName, ".js")

	// Get the parent directory (namespace)
	dir := filepath.Dir(path)
	namespace = filepath.Base(dir)

	// Validate
	if namespace == "" || namespace == "." || namespace == "/" {
		return "", "", alerr.New(alerr.ErrSchemaInvalid, "could not infer namespace from path").
			WithFile(path, 0).
			With("hint", "expected path format: schemas/<namespace>/<table>.js")
	}

	if tableName == "" {
		return "", "", alerr.New(alerr.ErrSchemaInvalid, "could not infer table name from path").
			WithFile(path, 0)
	}

	return namespace, tableName, nil
}

// ValidateEvaluatedTables validates a slice of table definitions.
// It checks for duplicate tables, invalid references, and other schema errors.
func ValidateEvaluatedTables(tables []*ast.TableDef) error {
	// Check for duplicates
	seen := make(map[string]bool)
	for _, t := range tables {
		key := t.QualifiedName()
		if seen[key] {
			return alerr.New(alerr.ErrSchemaDuplicate, "duplicate table definition").
				WithTable(t.Namespace, t.Name)
		}
		seen[key] = true
	}

	// Validate each table
	for _, t := range tables {
		if err := validateTableDef(t); err != nil {
			return err
		}
	}

	return nil
}

// validateTableDef validates a single table definition.
func validateTableDef(t *ast.TableDef) error {
	if t.Name == "" {
		return alerr.New(alerr.ErrSchemaInvalid, "table name is required")
	}

	if len(t.Columns) == 0 {
		return alerr.New(alerr.ErrSchemaInvalid, "table must have at least one column").
			WithTable(t.Namespace, t.Name)
	}

	// Check for duplicate columns
	colSeen := make(map[string]bool)
	for _, col := range t.Columns {
		if col.Name == "" {
			return alerr.New(alerr.ErrSchemaInvalid, "column name is required").
				WithTable(t.Namespace, t.Name)
		}
		if colSeen[col.Name] {
			return alerr.New(alerr.ErrSchemaDuplicate, "duplicate column name").
				WithTable(t.Namespace, t.Name).
				WithColumn(col.Name)
		}
		colSeen[col.Name] = true
	}

	return nil
}
