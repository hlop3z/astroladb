#!/bin/bash
# Build Types Script
# Generates type definitions for multiple languages from JSON Schema using quicktype

set -e

ALAB="../../alab.exe"

echo "=== Multi-Language Type Generation ==="
echo ""

# Check if alab.exe exists
if [ ! -f "$ALAB" ]; then
    echo "Error: alab.exe not found at $ALAB"
    echo "Build it first with: go build -o alab.exe ./cmd/alab"
    exit 1
fi

# Clean up previous runs
rm -rf lang/
rm -rf exports/
mkdir -p lang/typescript lang/python lang/go lang/rust

echo "1. Generating JSON Schema..."
$ALAB schema:export --format jsonschema
echo ""

# Create a wrapper schema that references all definitions
# quicktype needs a root type to generate from
cat > exports/schema-wrapped.json << 'WRAPPER'
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "title": "Schema",
  "type": "object",
  "definitions": DEFINITIONS_PLACEHOLDER,
  "properties": {
    "BlogUser": { "$ref": "#/definitions/BlogUser" },
    "BlogPost": { "$ref": "#/definitions/BlogPost" },
    "BlogComment": { "$ref": "#/definitions/BlogComment" }
  }
}
WRAPPER

# Extract definitions from schema and inject into wrapper
node -e "
const fs = require('fs');
const schema = JSON.parse(fs.readFileSync('exports/schema.json', 'utf8'));
const wrapper = fs.readFileSync('exports/schema-wrapped.json', 'utf8');
const result = wrapper.replace('DEFINITIONS_PLACEHOLDER', JSON.stringify(schema.definitions, null, 2));
fs.writeFileSync('exports/schema-wrapped.json', result);
"

echo "2. Generating TypeScript types..."
npx quicktype \
  --src exports/schema-wrapped.json \
  --src-lang schema \
  --lang typescript \
  --out lang/typescript/types.ts \
  --just-types \
  --acronym-style original
echo "   -> lang/typescript/types.ts"

echo "3. Generating Python types (dataclasses)..."
npx quicktype \
  --src exports/schema-wrapped.json \
  --src-lang schema \
  --lang python \
  --out lang/python/types.py \
  --just-types \
  --python-version 3.7
echo "   -> lang/python/types.py"

echo "4. Generating Go types..."
npx quicktype \
  --src exports/schema-wrapped.json \
  --src-lang schema \
  --lang go \
  --out lang/go/types.go \
  --just-types \
  --package models
echo "   -> lang/go/types.go"

echo "5. Generating Rust types..."
npx quicktype \
  --src exports/schema-wrapped.json \
  --src-lang schema \
  --lang rust \
  --out lang/rust/types.rs
echo "   -> lang/rust/types.rs"

echo ""
echo "=== Generation Complete ==="
echo ""
echo "Generated files:"
find lang -type f -name "*.*" | while read f; do
  echo "  $f ($(wc -l < "$f") lines)"
done

echo ""
echo "=== TypeScript Preview ==="
cat lang/typescript/types.ts

echo ""
echo "=== Python Preview ==="
cat lang/python/types.py

echo ""
echo "=== Go Preview ==="
cat lang/go/types.go

echo ""
echo "=== Rust Preview ==="
cat lang/rust/types.rs
