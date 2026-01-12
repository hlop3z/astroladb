package introspect

import (
	"context"
	"database/sql"
	"strings"

	"github.com/hlop3z/astroladb/internal/alerr"
	"github.com/hlop3z/astroladb/internal/ast"
	"github.com/hlop3z/astroladb/internal/dialect"
	"github.com/hlop3z/astroladb/internal/engine"
)

type postgresIntrospector struct {
	db      *sql.DB
	dialect dialect.Dialect
}

func (p *postgresIntrospector) IntrospectSchema(ctx context.Context) (*engine.Schema, error) {
	return introspectSchemaCommon(ctx, p, p)
}

func (p *postgresIntrospector) IntrospectSchemaWithMapping(ctx context.Context, mapping TableNameMapping) (*engine.Schema, error) {
	return introspectSchemaCommonWithMapping(ctx, p, p, mapping)
}

func (p *postgresIntrospector) introspectTableWithMapping(ctx context.Context, tableName string, mapping TableNameMapping) (*ast.TableDef, error) {
	return introspectTableCommonWithMapping(ctx, tableName, p, p, p, mapping)
}

func (p *postgresIntrospector) listTables(ctx context.Context) ([]string, error) {
	query := `
		SELECT tablename FROM pg_tables
		WHERE schemaname = current_schema()
		ORDER BY tablename
	`

	rows, err := p.db.QueryContext(ctx, query)
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

func (p *postgresIntrospector) IntrospectTable(ctx context.Context, tableName string) (*ast.TableDef, error) {
	return introspectTableCommon(ctx, tableName, p, p, p)
}

func (p *postgresIntrospector) introspectColumns(ctx context.Context, tableName string) ([]*ast.ColumnDef, error) {
	query := `
		SELECT
			c.column_name,
			c.data_type,
			c.is_nullable,
			c.column_default,
			c.character_maximum_length,
			c.numeric_precision,
			c.numeric_scale,
			COALESCE(pk.is_pk, FALSE) as is_primary_key
		FROM information_schema.columns c
		LEFT JOIN (
			SELECT kcu.column_name, TRUE as is_pk
			FROM information_schema.table_constraints tc
			JOIN information_schema.key_column_usage kcu
				ON tc.constraint_name = kcu.constraint_name
				AND tc.table_schema = kcu.table_schema
			WHERE tc.table_name = $1
				AND tc.constraint_type = 'PRIMARY KEY'
				AND tc.table_schema = current_schema()
		) pk ON c.column_name = pk.column_name
		WHERE c.table_schema = current_schema()
			AND c.table_name = $1
		ORDER BY c.ordinal_position
	`

	rows, err := p.db.QueryContext(ctx, query, tableName)
	if err != nil {
		return nil, alerr.WrapSQL(err, "introspect columns", tableName)
	}
	defer rows.Close()

	var columns []*ast.ColumnDef
	for rows.Next() {
		var raw RawColumn
		var isNullable string
		var isPK bool

		err := rows.Scan(
			&raw.Name,
			&raw.DataType,
			&isNullable,
			&raw.Default,
			&raw.MaxLength,
			&raw.Precision,
			&raw.Scale,
			&isPK,
		)
		if err != nil {
			return nil, alerr.WrapSQL(err, "scan column", tableName)
		}

		raw.IsNullable = isNullable == "YES"
		raw.IsPrimaryKey = isPK

		// Map SQL type to Alab type (deterministic, no heuristics)
		typeMapping := MapPostgresType(raw.DataType, raw.MaxLength, raw.Precision, raw.Scale)

		col := &ast.ColumnDef{
			Name:       raw.Name,
			Type:       typeMapping.AlabType,
			TypeArgs:   typeMapping.TypeArgs,
			Nullable:   raw.IsNullable && !raw.IsPrimaryKey, // PK columns are never nullable
			PrimaryKey: raw.IsPrimaryKey,
		}

		// Handle default values
		if raw.Default.Valid {
			col.DefaultSet = true
			col.ServerDefault = raw.Default.String
		}

		columns = append(columns, col)
	}

	return columns, rows.Err()
}

func (p *postgresIntrospector) introspectIndexes(ctx context.Context, tableName string) ([]*ast.IndexDef, error) {
	query := `
		SELECT
			i.relname as index_name,
			ix.indisunique as is_unique,
			array_to_string(array_agg(a.attname ORDER BY x.n), ',') as columns
		FROM pg_index ix
		JOIN pg_class t ON t.oid = ix.indrelid
		JOIN pg_class i ON i.oid = ix.indexrelid
		JOIN LATERAL unnest(ix.indkey) WITH ORDINALITY AS x(attnum, n) ON TRUE
		JOIN pg_attribute a ON a.attrelid = t.oid AND a.attnum = x.attnum
		WHERE t.relname = $1
			AND t.relnamespace = (SELECT oid FROM pg_namespace WHERE nspname = current_schema())
			AND NOT ix.indisprimary
		GROUP BY i.relname, ix.indisunique
		ORDER BY i.relname
	`

	rows, err := p.db.QueryContext(ctx, query, tableName)
	if err != nil {
		return nil, alerr.WrapSQL(err, "introspect indexes", tableName)
	}
	defer rows.Close()

	var indexes []*ast.IndexDef
	for rows.Next() {
		var name string
		var unique bool
		var columnsStr string

		err := rows.Scan(&name, &unique, &columnsStr)
		if err != nil {
			return nil, alerr.WrapSQL(err, "scan index", tableName)
		}

		indexes = append(indexes, &ast.IndexDef{
			Name:    name,
			Columns: strings.Split(columnsStr, ","),
			Unique:  unique,
		})
	}

	return indexes, rows.Err()
}

func (p *postgresIntrospector) introspectForeignKeys(ctx context.Context, tableName string) ([]*ast.ForeignKeyDef, error) {
	query := `
		SELECT
			tc.constraint_name,
			kcu.column_name,
			ccu.table_name AS foreign_table_name,
			ccu.column_name AS foreign_column_name,
			rc.delete_rule,
			rc.update_rule
		FROM information_schema.table_constraints AS tc
		JOIN information_schema.key_column_usage AS kcu
			ON tc.constraint_name = kcu.constraint_name
			AND tc.table_schema = kcu.table_schema
		JOIN information_schema.constraint_column_usage AS ccu
			ON ccu.constraint_name = tc.constraint_name
			AND ccu.table_schema = tc.table_schema
		JOIN information_schema.referential_constraints AS rc
			ON rc.constraint_name = tc.constraint_name
			AND rc.constraint_schema = tc.table_schema
		WHERE tc.constraint_type = 'FOREIGN KEY'
			AND tc.table_name = $1
			AND tc.table_schema = current_schema()
		ORDER BY tc.constraint_name, kcu.ordinal_position
	`

	rows, err := p.db.QueryContext(ctx, query, tableName)
	if err != nil {
		return nil, alerr.WrapSQL(err, "introspect foreign keys", tableName)
	}
	defer rows.Close()

	acc := NewFKAccumulator()
	for rows.Next() {
		var name, column, refTable, refColumn, onDelete, onUpdate string

		err := rows.Scan(&name, &column, &refTable, &refColumn, &onDelete, &onUpdate)
		if err != nil {
			return nil, alerr.WrapSQL(err, "scan foreign key", tableName)
		}

		acc.Add(name, column, refTable, refColumn, onDelete, onUpdate)
	}

	return acc.Values(), rows.Err()
}

func (p *postgresIntrospector) TableExists(ctx context.Context, tableName string) (bool, error) {
	var exists bool
	err := p.db.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM pg_tables
			WHERE schemaname = current_schema() AND tablename = $1
		)
	`, tableName).Scan(&exists)

	if err != nil {
		return false, alerr.WrapSQL(err, "check table existence", tableName)
	}
	return exists, nil
}
