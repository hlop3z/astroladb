#!/bin/bash
# OpenAPI Export Example
# Demonstrates schema export to OpenAPI, TypeScript, and Go

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
$ALAB check
echo ""

echo "2. Exporting to OpenAPI 3.0..."
echo "----------------------------------------"
$ALAB export --format openapi
cat exports/openapi.json | head -50
echo "..."
echo "----------------------------------------"
echo ""

echo "3. Exporting to TypeScript..."
echo "----------------------------------------"
$ALAB export --format typescript
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
