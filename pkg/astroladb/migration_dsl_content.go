package astroladb

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/hlop3z/astroladb/internal/ast"
)

// Migration content generation (private implementation).
// Generates complete JavaScript migration file content.

func (c *Client) generateMigrationContent(name, upRevision, downRevision string, ops []ast.Operation) string {
	var sb strings.Builder

	// Header comments
	sb.WriteString("// Migration: ")
	sb.WriteString(name)
	sb.WriteString("\n")
	sb.WriteString("// Generated at: ")
	sb.WriteString(time.Now().Format("2006-01-02 15:04:05"))
	sb.WriteString("\n\n")

	// Write migration wrapper with metadata
	sb.WriteString("export default migration({\n")
	sb.WriteString("  description: null,\n")
	sb.WriteString(fmt.Sprintf("  up_revision: \"%s\",\n", upRevision))
	if downRevision == "" {
		sb.WriteString("  down_revision: null,\n\n")
	} else {
		sb.WriteString(fmt.Sprintf("  down_revision: \"%s\",\n\n", downRevision))
	}
	sb.WriteString("  up(m) {\n")

	// Track operation sections for organizing comments
	var lastOpType string
	hasIndexes := false

	// First pass: check if we have standalone indexes
	for _, op := range ops {
		if _, ok := op.(*ast.CreateIndex); ok {
			hasIndexes = true
			break
		}
	}

	for _, op := range ops {
		currentOpType := getOperationType(op)

		// Add section headers when switching operation types
		if currentOpType != lastOpType && currentOpType != "" {
			if lastOpType != "" {
				sb.WriteString("\n")
			}
			switch currentOpType {
			case "table":
				// Don't add a section header for tables, we'll comment each table individually
			case "index":
				if hasIndexes {
					sb.WriteString("    // Indexes\n")
				}
			case "column":
				sb.WriteString("    // Columns\n")
			case "foreignkey":
				sb.WriteString("    // Foreign Keys\n")
			}
			lastOpType = currentOpType
		}

		switch o := op.(type) {
		case *ast.CreateTable:
			c.writeCreateTable(&sb, o)
		case *ast.DropTable:
			c.writeDropTable(&sb, o)
		case *ast.RenameTable:
			c.writeRenameTable(&sb, o)
		case *ast.AddColumn:
			c.writeAddColumn(&sb, o)
		case *ast.DropColumn:
			c.writeDropColumn(&sb, o)
		case *ast.RenameColumn:
			c.writeRenameColumn(&sb, o)
		case *ast.AlterColumn:
			c.writeAlterColumn(&sb, o)
		case *ast.CreateIndex:
			c.writeCreateIndex(&sb, o)
		case *ast.DropIndex:
			c.writeDropIndex(&sb, o)
		case *ast.AddForeignKey:
			c.writeAddForeignKey(&sb, o)
		case *ast.DropForeignKey:
			c.writeDropForeignKey(&sb, o)
		default:
			sb.WriteString("    // Unsupported operation\n")
		}
	}

	sb.WriteString("  },\n\n")

	// Write down function (reverse operations)
	sb.WriteString("  down(m) {\n")
	for i := len(ops) - 1; i >= 0; i-- {
		switch o := ops[i].(type) {
		case *ast.CreateTable:
			ref := o.Name
			if o.Namespace != "" {
				ref = o.Namespace + "." + o.Name
			}
			sb.WriteString(fmt.Sprintf("    m.drop_table(\"%s\")\n", ref))
		case *ast.DropTable:
			sb.WriteString(fmt.Sprintf("    // Manual: Recreate table %s (data cannot be recovered automatically)\n", o.Name))
		case *ast.RenameTable:
			oldRef := o.OldName
			newRef := o.NewName
			if o.Namespace != "" {
				oldRef = o.Namespace + "." + o.OldName
				newRef = o.Namespace + "." + o.NewName
			}
			sb.WriteString(fmt.Sprintf("    m.rename_table(\"%s\", \"%s\")\n", newRef, oldRef))
		case *ast.AddColumn:
			ref := o.Table_
			if o.Namespace != "" {
				ref = o.Namespace + "." + o.Table_
			}
			sb.WriteString(fmt.Sprintf("    m.drop_column(\"%s\", \"%s\")\n", ref, o.Column.Name))
		case *ast.DropColumn:
			sb.WriteString(fmt.Sprintf("    // Manual: Recreate column %s (data cannot be recovered automatically)\n", o.Name))
		case *ast.RenameColumn:
			ref := o.Table_
			if o.Namespace != "" {
				ref = o.Namespace + "." + o.Table_
			}
			sb.WriteString(fmt.Sprintf("    m.rename_column(\"%s\", \"%s\", \"%s\")\n", ref, o.NewName, o.OldName))
		case *ast.AlterColumn:
			ref := o.Table_
			if o.Namespace != "" {
				ref = o.Namespace + "." + o.Table_
			}
			sb.WriteString(fmt.Sprintf("    // Manual: Reverse alter_column on %s.%s (original state unknown)\n", ref, o.Name))
		case *ast.CreateIndex:
			indexName := o.Name
			if indexName == "" {
				indexName = fmt.Sprintf("idx_%s_%s", o.Table_, strings.Join(o.Columns, "_"))
			}
			sb.WriteString(fmt.Sprintf("    m.drop_index(\"%s\")\n", indexName))
		case *ast.DropIndex:
			sb.WriteString(fmt.Sprintf("    // Manual: Recreate index %s (original definition unknown)\n", o.Name))
		case *ast.AddForeignKey:
			fkName := o.Name
			if fkName == "" {
				fkName = fmt.Sprintf("fk_%s_%s", o.Table_, strings.Join(o.Columns, "_"))
			}
			ref := o.Table_
			if o.Namespace != "" {
				ref = o.Namespace + "." + o.Table_
			}
			sb.WriteString(fmt.Sprintf("    m.drop_foreign_key(\"%s\", \"%s\")\n", ref, fkName))
		case *ast.DropForeignKey:
			sb.WriteString(fmt.Sprintf("    // Manual: Recreate foreign key %s (original definition unknown)\n", o.Name))
		}
	}
	sb.WriteString("  }\n")
	sb.WriteString("})\n")

	// Beautify the generated JavaScript code
	return beautifyJavaScript(sb.String())
}

// getOperationType returns the operation type category for section grouping.
func getOperationType(op ast.Operation) string {
	switch op.(type) {
	case *ast.CreateTable, *ast.DropTable, *ast.RenameTable:
		return "table"
	case *ast.CreateIndex, *ast.DropIndex:
		return "index"
	case *ast.AddColumn, *ast.DropColumn, *ast.RenameColumn, *ast.AlterColumn:
		return "column"
	case *ast.AddForeignKey, *ast.DropForeignKey:
		return "foreignkey"
	default:
		return ""
	}
}

// beautifyJavaScript formats JavaScript code using Prettier for professional-quality formatting.
// It tries to use npx prettier, and falls back to unformatted code if Prettier is unavailable.
func beautifyJavaScript(code string) string {
	// Try to format with prettier via npx
	cmd := exec.Command("npx", "--yes", "prettier@3.4.2",
		"--parser", "babel",
		"--tab-width", "2",
		"--single-quote=false",
		"--trailing-comma", "none",
		"--arrow-parens", "avoid",
		"--print-width", "80")

	var stdout, stderr bytes.Buffer
	cmd.Stdin = strings.NewReader(code)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		// Notify user that Prettier is not available
		fmt.Fprintln(os.Stderr, "Warning: Prettier not found. Migration generated without formatting.")
		fmt.Fprintln(os.Stderr, "  Install Node.js from https://nodejs.org or run: npm install -g prettier")

		// Return unformatted code - migration will still work
		return code
	}

	return stdout.String()
}
