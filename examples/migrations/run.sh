#!/bin/bash
# Migration Operations Example
# Demonstrates all available migration operations

set -e

ALAB="../../alab.exe"

echo "=== Migration Operations Example ==="
echo ""

# Check if alab.exe exists
if [ ! -f "$ALAB" ]; then
    echo "Error: alab.exe not found at $ALAB"
    echo "Build it first with: go build -o alab.exe ./cmd/alab"
    exit 1
fi

# Clean up previous runs
rm -f example.db
rm -rf migrations/
rm -rf .alab/

echo "1. Initializing project..."
$ALAB init
echo ""

echo "2. Checking schemas..."
$ALAB schema:check
echo ""

echo "3. Dumping schema as SQL..."
echo "----------------------------------------"
$ALAB schema:dump
echo "----------------------------------------"
echo ""

echo "4. Generating initial migration..."
$ALAB migration:generate initial_schema
echo ""

echo "5. Running migrations..."
$ALAB migration:run
echo ""

echo "6. Migration status..."
$ALAB migration:status
echo ""

echo "=== Example Complete ==="
echo ""
echo "Available migration operations in JS DSL:"
echo "  - create_table(ref, fn)      - Create a new table"
echo "  - drop_table(ref)            - Drop a table"
echo "  - rename_table(oldRef, new)  - Rename a table"
echo "  - add_column(ref, fn)        - Add a column"
echo "  - drop_column(ref, column)   - Drop a column"
echo "  - rename_column(ref, old, new) - Rename a column"
echo "  - alter_column(ref, col, fn) - Modify column properties"
echo "  - create_index(ref, cols)    - Create an index"
echo "  - drop_index(name)           - Drop an index"
echo "  - add_foreign_key(...)       - Add FK constraint"
echo "  - drop_foreign_key(...)      - Drop FK constraint"
echo "  - sql(query)                 - Execute raw SQL"
