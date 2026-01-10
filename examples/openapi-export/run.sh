#!/bin/bash
# OpenAPI Export Example
# Demonstrates schema export to OpenAPI, JSON Schema, and TypeScript

set -e

ALAB="../../alab.exe"

echo "=== OpenAPI Export Example ==="
echo ""

# Check if alab.exe exists
if [ ! -f "$ALAB" ]; then
    echo "Error: alab.exe not found at $ALAB"
    echo "Build it first with: go build -o alab.exe ./cmd/alab"
    exit 1
fi

# Clean up previous runs
rm -f blog.db
rm -rf migrations/
rm -rf .alab/
rm -rf exports/

echo "1. Checking schemas..."
$ALAB schema:check
echo ""

echo "2. Exporting to OpenAPI 3.0..."
echo "----------------------------------------"
$ALAB schema:export --format openapi
cat exports/openapi.json | head -50
echo "..."
echo "----------------------------------------"
echo ""

echo "3. Exporting to JSON Schema..."
echo "----------------------------------------"
$ALAB schema:export --format jsonschema
echo "----------------------------------------"
echo ""

echo "4. Exporting to TypeScript..."
echo "----------------------------------------"
$ALAB schema:export --format typescript
cat exports/types.ts
echo "----------------------------------------"
echo ""

echo "=== Example Complete ==="
echo ""
echo "Generated files in exports/:"
ls -la exports/
echo ""
echo "OpenAPI extensions for tooling:"
echo "  Schema-level:"
echo "    x-table      - SQL table name"
echo "    x-namespace  - Schema namespace"
echo "  Property-level (foreign keys):"
echo "    x-ref        - Logical reference (e.g., 'blog.user')"
echo "    x-fk         - Full FK path (e.g., 'blog.user.id')"


echo "=== LANGS Types ==="
echo "----------------------------------------"
sh types-build.sh
echo "Creating Types..."
echo "----------------------------------------"
echo ""

echo "=== Mik (Rust) ==="
echo "----------------------------------------"
sh types-mik.sh ./lang/rust/types.rs
echo "Editing (Rust) Types..."
echo "----------------------------------------"
