#!/bin/bash
# Build Types Script
# Generates type definitions from schema using alab's native exports

set -e

ALAB="../../alab.exe"

echo "=== Type Generation ==="
echo ""

# Check if alab.exe exists
if [ ! -f "$ALAB" ]; then
    echo "Error: alab.exe not found at $ALAB"
    echo "Build it first with: go build -o alab.exe ./cmd/alab"
    exit 1
fi

# Clean up previous runs
rm -rf exports/

# Generate all formats at once
# OpenAPI and GraphQL go to root, typed exports split by namespace
$ALAB export --format all
