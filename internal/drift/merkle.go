// Package drift provides schema drift detection using merkle trees.
// It compares expected schema (from migrations) against actual database schema
// and efficiently identifies differences using hierarchical hashing.
package drift

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"

	"github.com/cbergoon/merkletree"

	"github.com/hlop3z/astroladb/internal/alerr"
	"github.com/hlop3z/astroladb/internal/ast"
	"github.com/hlop3z/astroladb/internal/engine"
)

// SchemaHash represents the merkle root hash of a schema.
type SchemaHash struct {
	Root   string                // Root hash of entire schema
	Tables map[string]*TableHash // Individual table hashes for drill-down
}

// TableHash represents the merkle hash of a single table.
type TableHash struct {
	Name    string            // Qualified table name (namespace.table)
	Hash    string            // Hash of entire table structure
	Columns map[string]string // Column name -> hash
	Indexes map[string]string // Index name -> hash
	FKs     map[string]string // FK name -> hash
}

// tableContent implements merkletree.Content for table-level hashing.
type tableContent struct {
	name string
	hash string
}

func (t tableContent) CalculateHash() ([]byte, error) {
	h := sha256.Sum256([]byte(t.hash))
	return h[:], nil
}

func (t tableContent) Equals(other merkletree.Content) (bool, error) {
	o, ok := other.(tableContent)
	if !ok {
		return false, nil
	}
	return t.hash == o.hash, nil
}

// ComputeSchemaHash computes the merkle tree hash for a schema.
// The hash is hierarchical: schema -> tables -> columns/indexes/fks
func ComputeSchemaHash(schema *engine.Schema) (*SchemaHash, error) {
	if schema == nil || len(schema.Tables) == 0 {
		return &SchemaHash{
			Root:   emptyHash(),
			Tables: make(map[string]*TableHash),
		}, nil
	}

	result := &SchemaHash{
		Tables: make(map[string]*TableHash),
	}

	// Compute hash for each table
	var tableContents []merkletree.Content
	tableNames := schema.TableNames() // Sorted for determinism

	for _, name := range tableNames {
		table := schema.Tables[name]
		tableHash := computeTableHash(table)
		result.Tables[name] = tableHash

		tableContents = append(tableContents, tableContent{
			name: name,
			hash: tableHash.Hash,
		})
	}

	// Build merkle tree from table hashes
	if len(tableContents) == 0 {
		result.Root = emptyHash()
		return result, nil
	}

	tree, err := merkletree.NewTree(tableContents)
	if err != nil {
		return nil, alerr.Wrap(alerr.ErrIntrospection, err, "failed to build merkle tree")
	}

	result.Root = hex.EncodeToString(tree.MerkleRoot())
	return result, nil
}

// computeTableHash computes the hash for a single table.
func computeTableHash(table *ast.TableDef) *TableHash {
	result := &TableHash{
		Name:    table.QualifiedName(),
		Columns: make(map[string]string),
		Indexes: make(map[string]string),
		FKs:     make(map[string]string),
	}

	// Hash columns (sorted for determinism)
	var columnHashes []string
	columnNames := make([]string, 0, len(table.Columns))
	for _, col := range table.Columns {
		columnNames = append(columnNames, col.Name)
	}
	sort.Strings(columnNames)

	for _, colName := range columnNames {
		col := table.GetColumn(colName)
		colHash := computeColumnHash(col)
		result.Columns[colName] = colHash
		columnHashes = append(columnHashes, colName+":"+colHash)
	}

	// Hash indexes (sorted for determinism)
	var indexHashes []string
	indexNames := make([]string, 0, len(table.Indexes))
	for _, idx := range table.Indexes {
		indexNames = append(indexNames, idx.Name)
	}
	sort.Strings(indexNames)

	for _, idxName := range indexNames {
		var idx *ast.IndexDef
		for _, i := range table.Indexes {
			if i.Name == idxName {
				idx = i
				break
			}
		}
		if idx != nil {
			idxHash := computeIndexHash(idx)
			result.Indexes[idxName] = idxHash
			indexHashes = append(indexHashes, idxName+":"+idxHash)
		}
	}

	// Hash foreign keys (sorted for determinism)
	var fkHashes []string
	fkNames := make([]string, 0, len(table.ForeignKeys))
	for _, fk := range table.ForeignKeys {
		fkNames = append(fkNames, fk.Name)
	}
	sort.Strings(fkNames)

	for _, fkName := range fkNames {
		var fk *ast.ForeignKeyDef
		for _, f := range table.ForeignKeys {
			if f.Name == fkName {
				fk = f
				break
			}
		}
		if fk != nil {
			fkHash := computeFKHash(fk)
			result.FKs[fkName] = fkHash
			fkHashes = append(fkHashes, fkName+":"+fkHash)
		}
	}

	// Combine all hashes for table hash
	tableData := fmt.Sprintf("table:%s|columns:[%s]|indexes:[%s]|fks:[%s]",
		table.QualifiedName(),
		strings.Join(columnHashes, ","),
		strings.Join(indexHashes, ","),
		strings.Join(fkHashes, ","),
	)
	result.Hash = hashString(tableData)

	return result
}

// computeColumnHash computes a deterministic hash for a column.
func computeColumnHash(col *ast.ColumnDef) string {
	// Include all relevant column properties
	data := fmt.Sprintf("name:%s|type:%s|nullable:%v|unique:%v|pk:%v",
		col.Name,
		normalizeType(col.Type, col.TypeArgs),
		col.Nullable,
		col.Unique,
		col.PrimaryKey,
	)

	// Include default if set (check both Default and ServerDefault for normalization compatibility)
	// After dev database normalization, defaults are stored in ServerDefault (introspected form)
	if col.DefaultSet {
		data += fmt.Sprintf("|default:%v", col.Default)
	}
	if col.ServerDefault != "" {
		data += fmt.Sprintf("|serverdefault:%s", col.ServerDefault)
	}

	return hashString(data)
}

// computeIndexHash computes a deterministic hash for an index.
func computeIndexHash(idx *ast.IndexDef) string {
	data := fmt.Sprintf("name:%s|columns:[%s]|unique:%v",
		idx.Name,
		strings.Join(idx.Columns, ","),
		idx.Unique,
	)

	if idx.Where != "" {
		data += fmt.Sprintf("|where:%s", idx.Where)
	}

	return hashString(data)
}

// computeFKHash computes a deterministic hash for a foreign key.
func computeFKHash(fk *ast.ForeignKeyDef) string {
	data := fmt.Sprintf("name:%s|columns:[%s]|ref_table:%s|ref_columns:[%s]|on_delete:%s|on_update:%s",
		fk.Name,
		strings.Join(fk.Columns, ","),
		fk.RefTable,
		strings.Join(fk.RefColumns, ","),
		fk.OnDelete,
		fk.OnUpdate,
	)

	return hashString(data)
}

// normalizeType creates a canonical string representation of a column type.
func normalizeType(typeName string, args []any) string {
	if len(args) == 0 {
		return typeName
	}

	argStrs := make([]string, len(args))
	for i, arg := range args {
		argStrs[i] = fmt.Sprintf("%v", arg)
	}
	return fmt.Sprintf("%s(%s)", typeName, strings.Join(argStrs, ","))
}

// hashString computes SHA256 hash of a string and returns hex encoding.
func hashString(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

// emptyHash returns a consistent hash for empty schemas.
func emptyHash() string {
	return hashString("empty_schema")
}

// CompareHashes compares two schema hashes and returns differences.
func CompareHashes(expected, actual *SchemaHash) *HashComparison {
	result := &HashComparison{
		Match:         expected.Root == actual.Root,
		ExpectedRoot:  expected.Root,
		ActualRoot:    actual.Root,
		TableDiffs:    make(map[string]*TableDiff),
		MissingTables: []string{},
		ExtraTables:   []string{},
	}

	if result.Match {
		return result
	}

	// Find missing tables (in expected but not in actual)
	for name := range expected.Tables {
		if _, exists := actual.Tables[name]; !exists {
			result.MissingTables = append(result.MissingTables, name)
		}
	}
	sort.Strings(result.MissingTables)

	// Find extra tables (in actual but not in expected)
	for name := range actual.Tables {
		if _, exists := expected.Tables[name]; !exists {
			result.ExtraTables = append(result.ExtraTables, name)
		}
	}
	sort.Strings(result.ExtraTables)

	// Compare tables that exist in both
	for name, expectedTable := range expected.Tables {
		actualTable, exists := actual.Tables[name]
		if !exists {
			continue // Already captured as missing
		}

		if expectedTable.Hash != actualTable.Hash {
			diff := compareTableHashes(expectedTable, actualTable)
			result.TableDiffs[name] = diff
		}
	}

	return result
}

// HashComparison represents the result of comparing two schema hashes.
type HashComparison struct {
	Match         bool                  // True if schemas are identical
	ExpectedRoot  string                // Expected schema root hash
	ActualRoot    string                // Actual schema root hash
	TableDiffs    map[string]*TableDiff // Tables with differences
	MissingTables []string              // Tables missing from actual
	ExtraTables   []string              // Extra tables in actual
}

// TableDiff represents differences within a table.
type TableDiff struct {
	Name            string   // Table name
	MissingColumns  []string // Columns missing from actual
	ExtraColumns    []string // Extra columns in actual
	ModifiedColumns []string // Columns with different definitions
	MissingIndexes  []string // Indexes missing from actual
	ExtraIndexes    []string // Extra indexes in actual
	ModifiedIndexes []string // Indexes with different definitions
	MissingFKs      []string // FKs missing from actual
	ExtraFKs        []string // Extra FKs in actual
	ModifiedFKs     []string // FKs with different definitions
}

// HasDifferences returns true if the table has any differences.
func (d *TableDiff) HasDifferences() bool {
	return len(d.MissingColumns) > 0 ||
		len(d.ExtraColumns) > 0 ||
		len(d.ModifiedColumns) > 0 ||
		len(d.MissingIndexes) > 0 ||
		len(d.ExtraIndexes) > 0 ||
		len(d.ModifiedIndexes) > 0 ||
		len(d.MissingFKs) > 0 ||
		len(d.ExtraFKs) > 0 ||
		len(d.ModifiedFKs) > 0
}

// compareTableHashes compares two table hashes and returns differences.
func compareTableHashes(expected, actual *TableHash) *TableDiff {
	diff := &TableDiff{Name: expected.Name}

	// Compare columns
	for name, hash := range expected.Columns {
		actualHash, exists := actual.Columns[name]
		if !exists {
			diff.MissingColumns = append(diff.MissingColumns, name)
		} else if hash != actualHash {
			diff.ModifiedColumns = append(diff.ModifiedColumns, name)
		}
	}
	for name := range actual.Columns {
		if _, exists := expected.Columns[name]; !exists {
			diff.ExtraColumns = append(diff.ExtraColumns, name)
		}
	}

	// Compare indexes
	for name, hash := range expected.Indexes {
		actualHash, exists := actual.Indexes[name]
		if !exists {
			diff.MissingIndexes = append(diff.MissingIndexes, name)
		} else if hash != actualHash {
			diff.ModifiedIndexes = append(diff.ModifiedIndexes, name)
		}
	}
	for name := range actual.Indexes {
		if _, exists := expected.Indexes[name]; !exists {
			diff.ExtraIndexes = append(diff.ExtraIndexes, name)
		}
	}

	// Compare foreign keys
	for name, hash := range expected.FKs {
		actualHash, exists := actual.FKs[name]
		if !exists {
			diff.MissingFKs = append(diff.MissingFKs, name)
		} else if hash != actualHash {
			diff.ModifiedFKs = append(diff.ModifiedFKs, name)
		}
	}
	for name := range actual.FKs {
		if _, exists := expected.FKs[name]; !exists {
			diff.ExtraFKs = append(diff.ExtraFKs, name)
		}
	}

	// Sort all slices for deterministic output
	sort.Strings(diff.MissingColumns)
	sort.Strings(diff.ExtraColumns)
	sort.Strings(diff.ModifiedColumns)
	sort.Strings(diff.MissingIndexes)
	sort.Strings(diff.ExtraIndexes)
	sort.Strings(diff.ModifiedIndexes)
	sort.Strings(diff.MissingFKs)
	sort.Strings(diff.ExtraFKs)
	sort.Strings(diff.ModifiedFKs)

	return diff
}
