// Package cache provides local caching of schema snapshots and merkle hashes.
// The cache is stored in .alab/cache.db (SQLite) and is gitignored.
// It is optional and can always be rebuilt from migration files.
package cache

import (
	"encoding/json"

	"github.com/hlop3z/astroladb/internal/alerr"
	"github.com/hlop3z/astroladb/internal/ast"
	"github.com/hlop3z/astroladb/internal/drift"
	"github.com/hlop3z/astroladb/internal/engine"
)

// -----------------------------------------------------------------------------
// JSON-serializable schema structures
// These mirror the AST types but use JSON-safe types for storage.
// -----------------------------------------------------------------------------

// schemaJSON is the JSON-serializable representation of engine.Schema.
type schemaJSON struct {
	Tables map[string]*tableJSON `json:"tables"`
}

// tableJSON is the JSON-serializable representation of ast.TableDef.
type tableJSON struct {
	Namespace   string            `json:"namespace"`
	Name        string            `json:"name"`
	Columns     []*columnJSON     `json:"columns"`
	Indexes     []*indexJSON      `json:"indexes,omitempty"`
	ForeignKeys []*foreignKeyJSON `json:"foreign_keys,omitempty"`
	Checks      []*checkJSON      `json:"checks,omitempty"`
	Docs        string            `json:"docs,omitempty"`
	Deprecated  string            `json:"deprecated,omitempty"`
}

// columnJSON is the JSON-serializable representation of ast.ColumnDef.
type columnJSON struct {
	Name        string   `json:"name"`
	Type        string   `json:"type"`
	TypeArgs    []any    `json:"type_args,omitempty"`
	Nullable    bool     `json:"nullable"`
	NullableSet bool     `json:"nullable_set"`
	Unique      bool     `json:"unique"`
	PrimaryKey  bool     `json:"primary_key"`
	Default     any      `json:"default,omitempty"`
	DefaultSet  bool     `json:"default_set"`
	Reference   *refJSON `json:"reference,omitempty"`
	Min         *int     `json:"min,omitempty"`
	Max         *int     `json:"max,omitempty"`
	Pattern     string   `json:"pattern,omitempty"`
	Format      string   `json:"format,omitempty"`
	Docs        string   `json:"docs,omitempty"`
	Deprecated  string   `json:"deprecated,omitempty"`
}

// refJSON is the JSON-serializable representation of ast.Reference.
type refJSON struct {
	Table    string `json:"table"`
	Column   string `json:"column,omitempty"`
	OnDelete string `json:"on_delete,omitempty"`
	OnUpdate string `json:"on_update,omitempty"`
}

// indexJSON is the JSON-serializable representation of ast.IndexDef.
type indexJSON struct {
	Name    string   `json:"name"`
	Columns []string `json:"columns"`
	Unique  bool     `json:"unique"`
	Where   string   `json:"where,omitempty"`
}

// foreignKeyJSON is the JSON-serializable representation of ast.ForeignKeyDef.
type foreignKeyJSON struct {
	Name       string   `json:"name"`
	Columns    []string `json:"columns"`
	RefTable   string   `json:"ref_table"`
	RefColumns []string `json:"ref_columns"`
	OnDelete   string   `json:"on_delete,omitempty"`
	OnUpdate   string   `json:"on_update,omitempty"`
}

// checkJSON is the JSON-serializable representation of ast.CheckDef.
type checkJSON struct {
	Name       string `json:"name,omitempty"`
	Expression string `json:"expression"`
}

// schemaHashJSON is the JSON-serializable representation of drift.SchemaHash.
type schemaHashJSON struct {
	Root   string                    `json:"root"`
	Tables map[string]*tableHashJSON `json:"tables"`
}

// tableHashJSON is the JSON-serializable representation of drift.TableHash.
type tableHashJSON struct {
	Name    string            `json:"name"`
	Hash    string            `json:"hash"`
	Columns map[string]string `json:"columns"`
	Indexes map[string]string `json:"indexes"`
	FKs     map[string]string `json:"fks"`
}

// -----------------------------------------------------------------------------
// Serialization functions
// -----------------------------------------------------------------------------

// SerializeSchema converts an engine.Schema to JSON bytes for storage.
func SerializeSchema(s *engine.Schema) ([]byte, error) {
	if s == nil {
		return json.Marshal(&schemaJSON{Tables: make(map[string]*tableJSON)})
	}

	sj := &schemaJSON{
		Tables: make(map[string]*tableJSON),
	}

	for key, table := range s.Tables {
		sj.Tables[key] = tableToJSON(table)
	}

	data, err := json.Marshal(sj)
	if err != nil {
		return nil, alerr.Wrap(alerr.ErrCacheWrite, err, "failed to serialize schema")
	}
	return data, nil
}

// DeserializeSchema converts JSON bytes back to an engine.Schema.
func DeserializeSchema(data []byte) (*engine.Schema, error) {
	var sj schemaJSON
	if err := json.Unmarshal(data, &sj); err != nil {
		return nil, alerr.Wrap(alerr.ErrCacheRead, err, "failed to deserialize schema")
	}

	s := engine.NewSchema()
	for key, tj := range sj.Tables {
		s.Tables[key] = tableFromJSON(tj)
	}
	return s, nil
}

// SerializeSchemaHash converts a drift.SchemaHash to JSON bytes for storage.
func SerializeSchemaHash(h *drift.SchemaHash) ([]byte, error) {
	if h == nil {
		return json.Marshal(&schemaHashJSON{
			Root:   "",
			Tables: make(map[string]*tableHashJSON),
		})
	}

	hj := &schemaHashJSON{
		Root:   h.Root,
		Tables: make(map[string]*tableHashJSON),
	}

	for key, th := range h.Tables {
		hj.Tables[key] = &tableHashJSON{
			Name:    th.Name,
			Hash:    th.Hash,
			Columns: th.Columns,
			Indexes: th.Indexes,
			FKs:     th.FKs,
		}
	}

	data, err := json.Marshal(hj)
	if err != nil {
		return nil, alerr.Wrap(alerr.ErrCacheWrite, err, "failed to serialize schema hash")
	}
	return data, nil
}

// DeserializeSchemaHash converts JSON bytes back to a drift.SchemaHash.
func DeserializeSchemaHash(data []byte) (*drift.SchemaHash, error) {
	var hj schemaHashJSON
	if err := json.Unmarshal(data, &hj); err != nil {
		return nil, alerr.Wrap(alerr.ErrCacheRead, err, "failed to deserialize schema hash")
	}

	h := &drift.SchemaHash{
		Root:   hj.Root,
		Tables: make(map[string]*drift.TableHash),
	}

	for key, thj := range hj.Tables {
		h.Tables[key] = &drift.TableHash{
			Name:    thj.Name,
			Hash:    thj.Hash,
			Columns: thj.Columns,
			Indexes: thj.Indexes,
			FKs:     thj.FKs,
		}
	}
	return h, nil
}

// -----------------------------------------------------------------------------
// Conversion helpers
// -----------------------------------------------------------------------------

func tableToJSON(t *ast.TableDef) *tableJSON {
	tj := &tableJSON{
		Namespace:  t.Namespace,
		Name:       t.Name,
		Docs:       t.Docs,
		Deprecated: t.Deprecated,
	}

	// Convert columns
	tj.Columns = make([]*columnJSON, len(t.Columns))
	for i, col := range t.Columns {
		tj.Columns[i] = columnToJSON(col)
	}

	// Convert indexes
	if len(t.Indexes) > 0 {
		tj.Indexes = make([]*indexJSON, len(t.Indexes))
		for i, idx := range t.Indexes {
			tj.Indexes[i] = indexToJSON(idx)
		}
	}

	// Convert foreign keys
	if len(t.ForeignKeys) > 0 {
		tj.ForeignKeys = make([]*foreignKeyJSON, len(t.ForeignKeys))
		for i, fk := range t.ForeignKeys {
			tj.ForeignKeys[i] = foreignKeyToJSON(fk)
		}
	}

	// Convert checks
	if len(t.Checks) > 0 {
		tj.Checks = make([]*checkJSON, len(t.Checks))
		for i, chk := range t.Checks {
			tj.Checks[i] = checkToJSON(chk)
		}
	}

	return tj
}

func tableFromJSON(tj *tableJSON) *ast.TableDef {
	t := &ast.TableDef{
		Namespace:  tj.Namespace,
		Name:       tj.Name,
		Docs:       tj.Docs,
		Deprecated: tj.Deprecated,
	}

	// Convert columns
	t.Columns = make([]*ast.ColumnDef, len(tj.Columns))
	for i, colj := range tj.Columns {
		t.Columns[i] = columnFromJSON(colj)
	}

	// Convert indexes
	if len(tj.Indexes) > 0 {
		t.Indexes = make([]*ast.IndexDef, len(tj.Indexes))
		for i, idxj := range tj.Indexes {
			t.Indexes[i] = indexFromJSON(idxj)
		}
	}

	// Convert foreign keys
	if len(tj.ForeignKeys) > 0 {
		t.ForeignKeys = make([]*ast.ForeignKeyDef, len(tj.ForeignKeys))
		for i, fkj := range tj.ForeignKeys {
			t.ForeignKeys[i] = foreignKeyFromJSON(fkj)
		}
	}

	// Convert checks
	if len(tj.Checks) > 0 {
		t.Checks = make([]*ast.CheckDef, len(tj.Checks))
		for i, chkj := range tj.Checks {
			t.Checks[i] = checkFromJSON(chkj)
		}
	}

	return t
}

func columnToJSON(c *ast.ColumnDef) *columnJSON {
	cj := &columnJSON{
		Name:        c.Name,
		Type:        c.Type,
		TypeArgs:    c.TypeArgs,
		Nullable:    c.Nullable,
		NullableSet: c.NullableSet,
		Unique:      c.Unique,
		PrimaryKey:  c.PrimaryKey,
		DefaultSet:  c.DefaultSet,
		Min:         c.Min,
		Max:         c.Max,
		Pattern:     c.Pattern,
		Format:      c.Format,
		Docs:        c.Docs,
		Deprecated:  c.Deprecated,
	}

	// Handle default value - convert SQLExpr to a special format
	if c.DefaultSet {
		if expr, ok := ast.AsSQLExpr(c.Default); ok {
			cj.Default = map[string]string{"__sql_expr__": expr.Expr}
		} else {
			cj.Default = c.Default
		}
	}

	// Convert reference
	if c.Reference != nil {
		cj.Reference = &refJSON{
			Table:    c.Reference.Table,
			Column:   c.Reference.Column,
			OnDelete: c.Reference.OnDelete,
			OnUpdate: c.Reference.OnUpdate,
		}
	}

	return cj
}

func columnFromJSON(cj *columnJSON) *ast.ColumnDef {
	c := &ast.ColumnDef{
		Name:        cj.Name,
		Type:        cj.Type,
		TypeArgs:    cj.TypeArgs,
		Nullable:    cj.Nullable,
		NullableSet: cj.NullableSet,
		Unique:      cj.Unique,
		PrimaryKey:  cj.PrimaryKey,
		DefaultSet:  cj.DefaultSet,
		Min:         cj.Min,
		Max:         cj.Max,
		Pattern:     cj.Pattern,
		Format:      cj.Format,
		Docs:        cj.Docs,
		Deprecated:  cj.Deprecated,
	}

	// Handle default value - convert back from special format
	if cj.DefaultSet && cj.Default != nil {
		if m, ok := cj.Default.(map[string]any); ok {
			if expr, ok := m["__sql_expr__"].(string); ok {
				c.Default = ast.NewSQLExpr(expr)
			} else {
				c.Default = cj.Default
			}
		} else {
			c.Default = cj.Default
		}
	}

	// Convert reference
	if cj.Reference != nil {
		c.Reference = &ast.Reference{
			Table:    cj.Reference.Table,
			Column:   cj.Reference.Column,
			OnDelete: cj.Reference.OnDelete,
			OnUpdate: cj.Reference.OnUpdate,
		}
	}

	return c
}

func indexToJSON(idx *ast.IndexDef) *indexJSON {
	return &indexJSON{
		Name:    idx.Name,
		Columns: idx.Columns,
		Unique:  idx.Unique,
		Where:   idx.Where,
	}
}

func indexFromJSON(idxj *indexJSON) *ast.IndexDef {
	return &ast.IndexDef{
		Name:    idxj.Name,
		Columns: idxj.Columns,
		Unique:  idxj.Unique,
		Where:   idxj.Where,
	}
}

func foreignKeyToJSON(fk *ast.ForeignKeyDef) *foreignKeyJSON {
	return &foreignKeyJSON{
		Name:       fk.Name,
		Columns:    fk.Columns,
		RefTable:   fk.RefTable,
		RefColumns: fk.RefColumns,
		OnDelete:   fk.OnDelete,
		OnUpdate:   fk.OnUpdate,
	}
}

func foreignKeyFromJSON(fkj *foreignKeyJSON) *ast.ForeignKeyDef {
	return &ast.ForeignKeyDef{
		Name:       fkj.Name,
		Columns:    fkj.Columns,
		RefTable:   fkj.RefTable,
		RefColumns: fkj.RefColumns,
		OnDelete:   fkj.OnDelete,
		OnUpdate:   fkj.OnUpdate,
	}
}

func checkToJSON(chk *ast.CheckDef) *checkJSON {
	return &checkJSON{
		Name:       chk.Name,
		Expression: chk.Expression,
	}
}

func checkFromJSON(chkj *checkJSON) *ast.CheckDef {
	return &ast.CheckDef{
		Name:       chkj.Name,
		Expression: chkj.Expression,
	}
}
