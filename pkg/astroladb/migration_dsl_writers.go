package astroladb

import (
	"fmt"
	"strings"

	"github.com/hlop3z/astroladb/internal/ast"
)

// Pure DSL writers for migration content generation.
// These functions convert AST operations into JavaScript DSL calls.

func (c *Client) writeCreateTable(sb *strings.Builder, op *ast.CreateTable) {
	ref := op.Name
	if op.Namespace != "" {
		ref = op.Namespace + "." + op.Name
	}

	// Add table comment
	sb.WriteString(fmt.Sprintf("\n    // Table: %s\n", ref))
	sb.WriteString(fmt.Sprintf("    m.create_table(\"%s\", t => {\n", ref))

	// Sort columns: id first, timestamps last, others alphabetically
	sortedColumns := c.sortColumnsForDatabase(op.Columns)

	// Track which columns to skip (for timestamps detection)
	skip := make(map[int]bool)
	timestampIndex := -1

	// Detect timestamps() pattern: created_at and updated_at datetime columns
	for i := 0; i < len(sortedColumns)-1; i++ {
		col1 := sortedColumns[i]
		col2 := sortedColumns[i+1]
		if col1.Name == "created_at" && col1.Type == "datetime" &&
			col2.Name == "updated_at" && col2.Type == "datetime" {
			skip[i] = true
			skip[i+1] = true
			timestampIndex = i
		}
	}

	// Write columns
	for i, col := range sortedColumns {
		if skip[i] {
			continue
		}
		c.writeColumn(sb, col)
	}

	// Write timestamps() at the end if detected
	if timestampIndex >= 0 {
		sb.WriteString("    t.timestamps()\n")
	}

	sb.WriteString("  })\n")

	// Output embedded indexes as separate create_index calls
	for _, idx := range op.Indexes {
		c.writeCreateIndex(sb, &ast.CreateIndex{
			TableRef: ast.TableRef{
				Namespace: op.Namespace,
				Table_:    op.Name,
			},
			Name:        idx.Name,
			Columns:     idx.Columns,
			Unique:      idx.Unique,
			IfNotExists: true,
		})
	}
}

// typeToDSLMethod converts internal type names to DSL method names.
// Some internal types have different names than their DSL methods.
func typeToDSLMethod(internalType string) string {
	switch internalType {
	case "datetime":
		return "datetime"
	default:
		return internalType
	}
}

// formatDSLValue formats a value for DSL output (default/backfill).
// Handles internal Go types from Goja's JavaScript engine (int64, float64)
// and converts them back to JavaScript literals.
func formatDSLValue(method string, value any) string {
	switch v := value.(type) {
	case bool:
		return fmt.Sprintf(".%s(%t)", method, v)
	case int:
		return fmt.Sprintf(".%s(%d)", method, v)
	case int64: // Internal: JavaScript numbers from Goja
		return fmt.Sprintf(".%s(%d)", method, v)
	case float64: // Internal: JavaScript numbers from Goja
		// Goja represents all JavaScript numbers as float64 in Go.
		// Check if the value is mathematically an integer (no fractional part).
		// If so, format as integer without decimal point for cleaner DSL output.
		// Example: 42.0 → ".default(42)" instead of ".default(42.0)"
		if v == float64(int64(v)) {
			return fmt.Sprintf(".%s(%d)", method, int64(v))
		}
		return fmt.Sprintf(".%s(%v)", method, v)
	case string:
		// Check if it's a sql() expression (for backfill)
		if strings.HasPrefix(v, "sql(") {
			return fmt.Sprintf(".%s(%s)", method, v)
		}
		return fmt.Sprintf(".%s(\"%s\")", method, v)
	case *ast.SQLExpr:
		return fmt.Sprintf(".%s(sql(\"%s\"))", method, v.Expr)
	default:
		return ""
	}
}

// writeColumn writes a column definition.
func (c *Client) writeColumn(sb *strings.Builder, col *ast.ColumnDef) {
	// Handle special column types

	// Detect t.id() pattern: uuid primary key named "id"
	if col.Type == "uuid" && col.Name == "id" && col.PrimaryKey {
		sb.WriteString("    t.id()\n")
		return
	}

	// Detect belongs_to pattern: uuid column with a Reference
	if col.Reference != nil {
		c.writeBelongsTo(sb, col)
		return
	}

	// Build the column call
	var call strings.Builder
	dslMethod := typeToDSLMethod(col.Type)
	call.WriteString(fmt.Sprintf("    t.%s(\"%s\"", dslMethod, col.Name))

	// Add type arguments
	for _, arg := range col.TypeArgs {
		switch v := arg.(type) {
		case int:
			call.WriteString(fmt.Sprintf(", %d", v))
		case float64:
			call.WriteString(fmt.Sprintf(", %v", v))
		case string:
			call.WriteString(fmt.Sprintf(", \"%s\"", v))
		case []string:
			// Enum values array
			quoted := make([]string, len(v))
			for i, s := range v {
				quoted[i] = fmt.Sprintf("\"%s\"", s)
			}
			call.WriteString(fmt.Sprintf(", [%s]", strings.Join(quoted, ", ")))
		case []any:
			// Generic array (e.g., enum values passed as []any)
			quoted := make([]string, len(v))
			for i, s := range v {
				quoted[i] = fmt.Sprintf("\"%v\"", s)
			}
			call.WriteString(fmt.Sprintf(", [%s]", strings.Join(quoted, ", ")))
		}
	}
	call.WriteString(")")

	// Add modifiers
	if col.Nullable {
		call.WriteString(".optional()")
	}
	if col.Unique {
		call.WriteString(".unique()")
	}
	// Handle default values
	if col.DefaultSet {
		call.WriteString(formatDSLValue("default", col.Default))
	}

	call.WriteString("\n")
	sb.WriteString(call.String())
}

// writeBelongsTo writes a belongs_to relationship column.
func (c *Client) writeBelongsTo(sb *strings.Builder, col *ast.ColumnDef) {
	var call strings.Builder
	call.WriteString(fmt.Sprintf("    t.belongs_to(\"%s\")", col.Reference.Table))

	// Determine if an alias is needed by checking if column name differs from default
	// Default column name for belongs_to("ns.table") is "table_id"
	expectedColName := extractTableName(col.Reference.Table) + "_id"
	if col.Name != expectedColName {
		// Extract alias from column name (remove _id suffix)
		alias := strings.TrimSuffix(col.Name, "_id")
		call.WriteString(fmt.Sprintf(".as(\"%s\")", alias))
	}

	// Add modifiers
	if col.Nullable {
		call.WriteString(".optional()")
	}
	if col.Reference.OnDelete != "" {
		call.WriteString(fmt.Sprintf(".on_delete(\"%s\")", strings.ToLower(col.Reference.OnDelete)))
	}
	if col.Reference.OnUpdate != "" {
		call.WriteString(fmt.Sprintf(".on_update(\"%s\")", strings.ToLower(col.Reference.OnUpdate)))
	}

	call.WriteString("\n")
	sb.WriteString(call.String())
}

// extractTableName extracts the table name from a reference like "ns.table" or "table".
func extractTableName(ref string) string {
	parts := strings.Split(ref, ".")
	return parts[len(parts)-1]
}

// writeDropTable writes a drop_table DSL call.
func (c *Client) writeDropTable(sb *strings.Builder, op *ast.DropTable) {
	ref := op.Name
	if op.Namespace != "" {
		ref = op.Namespace + "." + op.Name
	}
	sb.WriteString(fmt.Sprintf("    m.drop_table(\"%s\")\n", ref))
}

// writeRenameTable writes a rename_table DSL call.
func (c *Client) writeRenameTable(sb *strings.Builder, op *ast.RenameTable) {
	oldRef := op.OldName
	newRef := op.NewName
	if op.Namespace != "" {
		oldRef = op.Namespace + "." + op.OldName
		newRef = op.Namespace + "." + op.NewName
	}
	sb.WriteString(fmt.Sprintf("    m.rename_table(\"%s\", \"%s\")\n", oldRef, newRef))
}

// writeRenameColumn writes a rename_column DSL call.
func (c *Client) writeRenameColumn(sb *strings.Builder, op *ast.RenameColumn) {
	ref := op.Table_
	if op.Namespace != "" {
		ref = op.Namespace + "." + op.Table_
	}
	sb.WriteString(fmt.Sprintf("    m.rename_column(\"%s\", \"%s\", \"%s\")\n", ref, op.OldName, op.NewName))
}

// writeAddColumn writes an add_column DSL call.
func (c *Client) writeAddColumn(sb *strings.Builder, op *ast.AddColumn) {
	ref := op.Table_
	if op.Namespace != "" {
		ref = op.Namespace + "." + op.Table_
	}

	dslMethod := typeToDSLMethod(op.Column.Type)
	sb.WriteString(fmt.Sprintf("    m.add_column(\"%s\", c => c.%s(\"%s\"", ref, dslMethod, op.Column.Name))

	// Add type args
	for _, arg := range op.Column.TypeArgs {
		switch v := arg.(type) {
		case int:
			sb.WriteString(fmt.Sprintf(", %d", v))
		case float64:
			sb.WriteString(fmt.Sprintf(", %v", v))
		case string:
			sb.WriteString(fmt.Sprintf(", \"%s\"", v))
		}
	}
	sb.WriteString(")")

	// Add modifiers
	if op.Column.Nullable {
		sb.WriteString(".optional()")
	}
	if op.Column.Unique {
		sb.WriteString(".unique()")
	}
	if op.Column.DefaultSet {
		sb.WriteString(formatDSLValue("default", op.Column.Default))
	}
	// Add backfill for existing rows
	if op.Column.BackfillSet {
		sb.WriteString(formatDSLValue("backfill", op.Column.Backfill))
	}

	sb.WriteString(")\n")
}

// writeDropColumn writes a drop_column DSL call.
func (c *Client) writeDropColumn(sb *strings.Builder, op *ast.DropColumn) {
	ref := op.Table_
	if op.Namespace != "" {
		ref = op.Namespace + "." + op.Table_
	}
	sb.WriteString(fmt.Sprintf("    m.drop_column(\"%s\", \"%s\")\n", ref, op.Name))
}

// writeCreateIndex writes a create_index DSL call.
func (c *Client) writeCreateIndex(sb *strings.Builder, op *ast.CreateIndex) {
	ref := op.Table_
	if op.Namespace != "" {
		ref = op.Namespace + "." + op.Table_
	}

	cols := make([]string, len(op.Columns))
	for i, col := range op.Columns {
		cols[i] = fmt.Sprintf("\"%s\"", col)
	}

	sb.WriteString(fmt.Sprintf("    // Index: %s\n", ref))
	sb.WriteString(fmt.Sprintf("    m.create_index(\"%s\", [%s]", ref, strings.Join(cols, ", ")))

	if op.Unique || op.Name != "" {
		sb.WriteString(", {")
		parts := []string{}
		if op.Unique {
			parts = append(parts, "unique: true")
		}
		if op.Name != "" {
			parts = append(parts, fmt.Sprintf("name: \"%s\"", op.Name))
		}
		sb.WriteString(strings.Join(parts, ", "))
		sb.WriteString("}")
	}

	sb.WriteString(")\n")
}

// writeDropIndex writes a drop_index DSL call.
func (c *Client) writeDropIndex(sb *strings.Builder, op *ast.DropIndex) {
	sb.WriteString(fmt.Sprintf("    m.drop_index(\"%s\")\n", op.Name))
}

// writeAlterColumn writes an alter_column DSL call.
func (c *Client) writeAlterColumn(sb *strings.Builder, op *ast.AlterColumn) {
	ref := op.Table_
	if op.Namespace != "" {
		ref = op.Namespace + "." + op.Table_
	}

	sb.WriteString(fmt.Sprintf("    m.alter_column(\"%s\", \"%s\", c => c", ref, op.Name))

	if op.NewType != "" {
		dslMethod := typeToDSLMethod(op.NewType)
		sb.WriteString(fmt.Sprintf(".set_type(\"%s\"", dslMethod))
		for _, arg := range op.NewTypeArgs {
			switch v := arg.(type) {
			case int:
				sb.WriteString(fmt.Sprintf(", %d", v))
			case float64:
				sb.WriteString(fmt.Sprintf(", %v", v))
			}
		}
		sb.WriteString(")")
	}
	if op.SetNullable != nil {
		if *op.SetNullable {
			sb.WriteString(".set_nullable()")
		} else {
			sb.WriteString(".set_not_null()")
		}
	}
	if op.DropDefault {
		sb.WriteString(".drop_default()")
	} else if op.SetDefault != nil {
		switch v := op.SetDefault.(type) {
		case bool:
			sb.WriteString(fmt.Sprintf(".set_default(%t)", v))
		case int:
			sb.WriteString(fmt.Sprintf(".set_default(%d)", v))
		case float64:
			sb.WriteString(fmt.Sprintf(".set_default(%v)", v))
		case string:
			sb.WriteString(fmt.Sprintf(".set_default(\"%s\")", v))
		}
	}
	if op.ServerDefault != "" {
		sb.WriteString(fmt.Sprintf(".set_server_default(\"%s\")", op.ServerDefault))
	}

	sb.WriteString(")\n")
}

// writeAddForeignKey writes an add_foreign_key DSL call.
func (c *Client) writeAddForeignKey(sb *strings.Builder, op *ast.AddForeignKey) {
	ref := op.Table_
	if op.Namespace != "" {
		ref = op.Namespace + "." + op.Table_
	}

	// Format columns
	cols := make([]string, len(op.Columns))
	for i, col := range op.Columns {
		cols[i] = fmt.Sprintf("\"%s\"", col)
	}

	// Format ref columns
	refCols := make([]string, len(op.RefColumns))
	for i, col := range op.RefColumns {
		refCols[i] = fmt.Sprintf("\"%s\"", col)
	}

	sb.WriteString(fmt.Sprintf("    m.add_foreign_key(\"%s\", [%s], \"%s\", [%s]",
		ref, strings.Join(cols, ", "), op.RefTable, strings.Join(refCols, ", ")))

	// Add options if present
	opts := []string{}
	if op.Name != "" {
		opts = append(opts, fmt.Sprintf("name: \"%s\"", op.Name))
	}
	if op.OnDelete != "" {
		opts = append(opts, fmt.Sprintf("on_delete: \"%s\"", strings.ToLower(op.OnDelete)))
	}
	if op.OnUpdate != "" {
		opts = append(opts, fmt.Sprintf("on_update: \"%s\"", strings.ToLower(op.OnUpdate)))
	}
	if len(opts) > 0 {
		sb.WriteString(fmt.Sprintf(", {%s}", strings.Join(opts, ", ")))
	}

	sb.WriteString(")\n")
}

// writeDropForeignKey writes a drop_foreign_key DSL call.
func (c *Client) writeDropForeignKey(sb *strings.Builder, op *ast.DropForeignKey) {
	ref := op.Table_
	if op.Namespace != "" {
		ref = op.Namespace + "." + op.Table_
	}
	sb.WriteString(fmt.Sprintf("    m.drop_foreign_key(\"%s\", \"%s\")\n", ref, op.Name))
}
