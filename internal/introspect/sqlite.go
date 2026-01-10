package introspect

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/hlop3z/astroladb/internal/alerr"
	"github.com/hlop3z/astroladb/internal/ast"
	"github.com/hlop3z/astroladb/internal/dialect"
	"github.com/hlop3z/astroladb/internal/engine"
)

type sqliteIntrospector struct {
	db      *sql.DB
	dialect dialect.Dialect
}

func (s *sqliteIntrospector) IntrospectSchema(ctx context.Context) (*engine.Schema, error) {
	return introspectSchemaCommon(ctx, s, s)
}

func (s *sqliteIntrospector) listTables(ctx context.Context) ([]string, error) {
	query := `
		SELECT name FROM sqlite_master
		WHERE type = 'table' AND name NOT LIKE 'sqlite_%'
		ORDER BY name
	`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, alerr.WrapSQL(err, "list tables", "")
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, alerr.WrapSQL(err, "scan table name", "")
		}
		tables = append(tables, name)
	}

	return tables, rows.Err()
}

func (s *sqliteIntrospector) IntrospectTable(ctx context.Context, tableName string) (*ast.TableDef, error) {
	return introspectTableCommon(ctx, tableName, s, s, s)
}

func (s *sqliteIntrospector) introspectColumns(ctx context.Context, tableName string) ([]*ast.ColumnDef, error) {
	// Use PRAGMA table_info to get column information
	// Returns: cid, name, type, notnull, dflt_value, pk
	query := fmt.Sprintf("PRAGMA table_info(%s)", s.dialect.QuoteIdent(tableName))

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, alerr.WrapSQL(err, "introspect columns", tableName)
	}
	defer rows.Close()

	var columns []*ast.ColumnDef
	for rows.Next() {
		var cid int
		var name, dataType string
		var notNull, pk int
		var defaultVal sql.NullString

		err := rows.Scan(&cid, &name, &dataType, &notNull, &defaultVal, &pk)
		if err != nil {
			return nil, alerr.WrapSQL(err, "scan column", tableName)
		}

		raw := RawColumn{
			Name:         name,
			DataType:     dataType,
			IsNullable:   notNull == 0,
			Default:      defaultVal,
			IsPrimaryKey: pk > 0,
		}

		// Map SQL type to Alab type
		typeMapping := MapSQLiteType(raw.DataType)
		typeMapping = InferAlabTypeFromContext(raw, typeMapping)

		col := &ast.ColumnDef{
			Name:       raw.Name,
			Type:       typeMapping.AlabType,
			TypeArgs:   typeMapping.TypeArgs,
			Nullable:   raw.IsNullable && !raw.IsPrimaryKey, // PK columns are never nullable
			PrimaryKey: raw.IsPrimaryKey,
		}

		// Handle default - skip auto-generated defaults for id columns
		if raw.Default.Valid && typeMapping.AlabType != "id" {
			col.DefaultSet = true
			col.ServerDefault = raw.Default.String
		}

		columns = append(columns, col)
	}

	return columns, rows.Err()
}

func (s *sqliteIntrospector) introspectIndexes(ctx context.Context, tableName string) ([]*ast.IndexDef, error) {
	// Get list of indexes from sqlite_master
	// Note: We must collect all index names first, then close the rows,
	// before opening additional queries. SQLite in-memory mode with shared
	// cache can have issues with concurrent queries on different connections.
	query := `
		SELECT name FROM sqlite_master
		WHERE type = 'index' AND tbl_name = ? AND sql IS NOT NULL
		ORDER BY name
	`

	rows, err := s.db.QueryContext(ctx, query, tableName)
	if err != nil {
		return nil, alerr.WrapSQL(err, "introspect indexes", tableName)
	}

	// Collect all index names first
	var indexNames []string
	for rows.Next() {
		var indexName string
		if err := rows.Scan(&indexName); err != nil {
			rows.Close()
			return nil, alerr.WrapSQL(err, "scan index", tableName)
		}
		indexNames = append(indexNames, indexName)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, alerr.WrapSQL(err, "iterate indexes", tableName)
	}
	rows.Close() // Close before opening additional queries

	// Now get details for each index
	var indexes []*ast.IndexDef
	for _, indexName := range indexNames {
		idx, err := s.getIndexDetails(ctx, tableName, indexName)
		if err != nil {
			return nil, err
		}
		if idx != nil {
			indexes = append(indexes, idx)
		}
	}

	return indexes, nil
}

func (s *sqliteIntrospector) getIndexDetails(ctx context.Context, tableName, indexName string) (*ast.IndexDef, error) {
	// Check if unique via pragma index_list
	// Note: We must close listRows before opening infoRows to avoid
	// connection pool issues with in-memory SQLite databases.
	unique := false
	listQuery := fmt.Sprintf("PRAGMA index_list(%s)", s.dialect.QuoteIdent(tableName))
	listRows, err := s.db.QueryContext(ctx, listQuery)
	if err != nil {
		return nil, alerr.WrapSQL(err, "get index list", tableName)
	}

	for listRows.Next() {
		var seq int
		var name string
		var isUnique int
		var origin, partial string

		// index_list returns: seq, name, unique, origin, partial
		err := listRows.Scan(&seq, &name, &isUnique, &origin, &partial)
		if err != nil {
			// Try scanning with fewer columns (older SQLite versions)
			continue
		}

		if name == indexName {
			unique = isUnique == 1
			break
		}
	}
	listRows.Close() // Close before opening next query

	// Get columns via pragma index_info
	infoQuery := fmt.Sprintf("PRAGMA index_info(%s)", s.dialect.QuoteIdent(indexName))
	infoRows, err := s.db.QueryContext(ctx, infoQuery)
	if err != nil {
		return nil, alerr.WrapSQL(err, "get index info", indexName)
	}
	defer infoRows.Close()

	var columns []string
	for infoRows.Next() {
		var seqno, cid int
		var colName string

		if err := infoRows.Scan(&seqno, &cid, &colName); err != nil {
			return nil, alerr.WrapSQL(err, "scan index column", indexName)
		}
		columns = append(columns, colName)
	}

	if len(columns) == 0 {
		return nil, nil
	}

	return &ast.IndexDef{
		Name:    indexName,
		Columns: columns,
		Unique:  unique,
	}, nil
}

func (s *sqliteIntrospector) introspectForeignKeys(ctx context.Context, tableName string) ([]*ast.ForeignKeyDef, error) {
	// Use PRAGMA foreign_key_list to get FK information
	// Returns: id, seq, table, from, to, on_update, on_delete, match
	query := fmt.Sprintf("PRAGMA foreign_key_list(%s)", s.dialect.QuoteIdent(tableName))

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, alerr.WrapSQL(err, "introspect foreign keys", tableName)
	}
	defer rows.Close()

	acc := NewFKAccumulator()
	for rows.Next() {
		var id, seq int
		var refTable, from, to, onUpdate, onDelete, match string

		err := rows.Scan(&id, &seq, &refTable, &from, &to, &onUpdate, &onDelete, &match)
		if err != nil {
			return nil, alerr.WrapSQL(err, "scan foreign key", tableName)
		}

		// SQLite uses numeric IDs for FKs, so generate a name
		name := fmt.Sprintf("fk_%s_%d", tableName, id)
		acc.Add(name, from, refTable, to, onDelete, onUpdate)
	}

	return acc.Values(), rows.Err()
}

func (s *sqliteIntrospector) TableExists(ctx context.Context, tableName string) (bool, error) {
	var name string
	err := s.db.QueryRowContext(ctx, `
		SELECT name FROM sqlite_master
		WHERE type = 'table' AND name = ?
	`, tableName).Scan(&name)

	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, alerr.WrapSQL(err, "check table existence", tableName)
	}
	return true, nil
}

// parseEnumValues extracts enum values from a CHECK constraint.
// SQLite represents enums as CHECK constraints like: CHECK(status IN ('draft', 'published'))
func parseEnumValues(checkSQL string) []string {
	// Simple parser - look for IN (...) pattern
	upper := strings.ToUpper(checkSQL)
	inIdx := strings.Index(upper, " IN (")
	if inIdx == -1 {
		inIdx = strings.Index(upper, " IN(")
	}
	if inIdx == -1 {
		return nil
	}

	start := strings.Index(checkSQL[inIdx:], "(")
	if start == -1 {
		return nil
	}
	start += inIdx + 1

	end := strings.Index(checkSQL[start:], ")")
	if end == -1 {
		return nil
	}

	valuesPart := checkSQL[start : start+end]
	parts := strings.Split(valuesPart, ",")

	var values []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		// Remove quotes
		p = strings.Trim(p, "'\"")
		if p != "" {
			values = append(values, p)
		}
	}

	return values
}
