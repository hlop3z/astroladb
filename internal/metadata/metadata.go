// Package metadata provides schema metadata tracking for Alab.
// It generates a JSON file (.alab/metadata.json) that stores:
// - Many-to-many relationship join tables
// - Polymorphic relationship mappings
// - Auto-generated indexes
// This makes it easy to audit and query schema information with external tools.
package metadata

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/hlop3z/astroladb/internal/ast"
	"github.com/hlop3z/astroladb/internal/strutil"
)

// Metadata holds all schema metadata for a project.
type Metadata struct {
	// Version of the metadata format
	Version string `json:"version"`

	// Generated timestamp
	GeneratedAt time.Time `json:"generated_at"`

	// Tables in the schema
	Tables map[string]*TableMeta `json:"tables"`

	// Many-to-many relationships and their join tables
	ManyToMany []*ManyToManyMeta `json:"many_to_many"`

	// Polymorphic relationships
	Polymorphic []*PolymorphicMeta `json:"polymorphic"`

	// Auto-generated join tables (derived from many_to_many)
	JoinTables map[string]*JoinTableMeta `json:"join_tables"`
}

// TableMeta holds metadata for a single table.
type TableMeta struct {
	Namespace   string   `json:"namespace"`
	Name        string   `json:"name"`
	SQLName     string   `json:"sql_name"`
	Columns     []string `json:"columns"`
	PrimaryKey  string   `json:"primary_key"`
	ForeignKeys []string `json:"foreign_keys,omitempty"`
	Indexes     []string `json:"indexes,omitempty"`
}

// ManyToManyMeta tracks a many-to-many relationship.
type ManyToManyMeta struct {
	// Source table (e.g., "auth.users")
	Source string `json:"source"`

	// Target table (e.g., "auth.roles")
	Target string `json:"target"`

	// Generated join table name (e.g., "auth_users_roles")
	JoinTable string `json:"join_table"`

	// Source FK column in join table
	SourceFK string `json:"source_fk"`

	// Target FK column in join table
	TargetFK string `json:"target_fk"`
}

// PolymorphicMeta tracks a polymorphic (belongs_to_any) relationship.
type PolymorphicMeta struct {
	// Table containing the polymorphic columns
	Table string `json:"table"`

	// Alias used (e.g., "commentable")
	Alias string `json:"alias"`

	// Type column name (e.g., "commentable_type")
	TypeColumn string `json:"type_column"`

	// ID column name (e.g., "commentable_id")
	IDColumn string `json:"id_column"`

	// Possible target tables
	Targets []string `json:"targets"`
}

// JoinTableMeta holds metadata for an auto-generated join table.
type JoinTableMeta struct {
	Name        string           `json:"name"`
	SourceTable string           `json:"source_table"`
	TargetTable string           `json:"target_table"`
	Columns     []*ast.ColumnDef `json:"-"` // Not serialized, used internally
	Definition  *ast.TableDef    `json:"-"` // Full table definition
}

// New creates a new empty Metadata instance.
func New() *Metadata {
	return &Metadata{
		Version:     "1.0",
		GeneratedAt: time.Now().UTC(),
		Tables:      make(map[string]*TableMeta),
		ManyToMany:  make([]*ManyToManyMeta, 0),
		Polymorphic: make([]*PolymorphicMeta, 0),
		JoinTables:  make(map[string]*JoinTableMeta),
	}
}

// AddTable adds a table to the metadata.
func (m *Metadata) AddTable(t *ast.TableDef) {
	sqlName := strutil.SQLName(t.Namespace, t.Name)
	key := t.Namespace + "." + t.Name

	meta := &TableMeta{
		Namespace:  t.Namespace,
		Name:       t.Name,
		SQLName:    sqlName,
		Columns:    make([]string, 0, len(t.Columns)),
		PrimaryKey: "id",
	}

	for _, col := range t.Columns {
		meta.Columns = append(meta.Columns, col.Name)
		if col.Reference != nil {
			meta.ForeignKeys = append(meta.ForeignKeys, col.Name)
		}
	}

	for _, idx := range t.Indexes {
		meta.Indexes = append(meta.Indexes, idx.Name)
	}

	m.Tables[key] = meta
}

// AddManyToMany registers a many-to-many relationship and generates the join table.
func (m *Metadata) AddManyToMany(sourceNS, sourceTable, targetRef string) *JoinTableMeta {
	// Parse target reference (e.g., "auth.roles")
	targetNS, targetTable := parseRef(targetRef)

	// Generate join table name (alphabetically sorted to ensure consistency)
	sourceSQLName := strutil.SQLName(sourceNS, sourceTable)
	targetSQLName := strutil.SQLName(targetNS, targetTable)

	// Sort alphabetically for consistent naming
	names := []string{sourceSQLName, targetSQLName}
	sort.Strings(names)
	joinTableName := names[0] + "_" + names[1]

	// Determine FK column names
	sourceFK := strutil.Singularize(sourceTable) + "_id"
	targetFK := strutil.Singularize(targetTable) + "_id"

	// Record the relationship
	rel := &ManyToManyMeta{
		Source:    sourceNS + "." + sourceTable,
		Target:    targetRef,
		JoinTable: joinTableName,
		SourceFK:  sourceFK,
		TargetFK:  targetFK,
	}
	m.ManyToMany = append(m.ManyToMany, rel)

	// Generate join table definition
	joinTable := &JoinTableMeta{
		Name:        joinTableName,
		SourceTable: sourceSQLName,
		TargetTable: targetSQLName,
	}

	// Create the table definition
	joinTable.Definition = &ast.TableDef{
		Namespace: "", // Join tables don't have a namespace
		Name:      joinTableName,
		Columns: []*ast.ColumnDef{
			{
				Name:      sourceFK,
				Type:      "uuid",
				Nullable:  false,
				Reference: &ast.Reference{Table: sourceNS + "." + sourceTable, Column: "id"},
			},
			{
				Name:      targetFK,
				Type:      "uuid",
				Nullable:  false,
				Reference: &ast.Reference{Table: targetRef, Column: "id"},
			},
		},
		Indexes: []*ast.IndexDef{
			{
				Name:    "idx_" + joinTableName + "_" + sourceFK,
				Columns: []string{sourceFK},
			},
			{
				Name:    "idx_" + joinTableName + "_" + targetFK,
				Columns: []string{targetFK},
			},
			{
				// Composite unique index to prevent duplicates
				Name:    "idx_" + joinTableName + "_unique",
				Columns: []string{sourceFK, targetFK},
				Unique:  true,
			},
		},
	}

	m.JoinTables[joinTableName] = joinTable
	return joinTable
}

// AddPolymorphic registers a polymorphic relationship.
func (m *Metadata) AddPolymorphic(tableNS, tableName, alias string, targets []string) {
	m.Polymorphic = append(m.Polymorphic, &PolymorphicMeta{
		Table:      tableNS + "." + tableName,
		Alias:      alias,
		TypeColumn: alias + "_type",
		IDColumn:   alias + "_id",
		Targets:    targets,
	})
}

// GetJoinTables returns all generated join table definitions.
func (m *Metadata) GetJoinTables() []*ast.TableDef {
	tables := make([]*ast.TableDef, 0, len(m.JoinTables))
	for _, jt := range m.JoinTables {
		if jt.Definition != nil {
			tables = append(tables, jt.Definition)
		}
	}
	return tables
}

// Save writes the metadata to a JSON file in the .alab directory.
// Deprecated: Use SaveToFile for custom paths.
func (m *Metadata) Save(projectDir string) error {
	// Create .alab directory
	metaDir := filepath.Join(projectDir, ".alab")
	if err := os.MkdirAll(metaDir, 0755); err != nil {
		return err
	}

	// Update timestamp
	m.GeneratedAt = time.Now().UTC()

	// Marshal to JSON
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}

	// Write file
	metaFile := filepath.Join(metaDir, "metadata.json")
	return os.WriteFile(metaFile, data, 0644)
}

// SaveToFile writes the metadata to a JSON file at the specified path.
func (m *Metadata) SaveToFile(filePath string) error {
	// Create parent directory if it doesn't exist
	dir := filepath.Dir(filePath)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	// Update timestamp
	m.GeneratedAt = time.Now().UTC()

	// Marshal to JSON
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}

	// Write file
	return os.WriteFile(filePath, data, 0644)
}

// Load reads metadata from a JSON file.
func Load(projectDir string) (*Metadata, error) {
	metaFile := filepath.Join(projectDir, ".alab", "metadata.json")

	data, err := os.ReadFile(metaFile)
	if err != nil {
		if os.IsNotExist(err) {
			return New(), nil // Return empty metadata if file doesn't exist
		}
		return nil, err
	}

	var m Metadata
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}

	// Initialize maps if nil
	if m.Tables == nil {
		m.Tables = make(map[string]*TableMeta)
	}
	if m.JoinTables == nil {
		m.JoinTables = make(map[string]*JoinTableMeta)
	}

	return &m, nil
}

// parseRef parses a table reference like "auth.users" into namespace and table.
func parseRef(ref string) (namespace, table string) {
	parts := strings.SplitN(ref, ".", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return "", ref
}
