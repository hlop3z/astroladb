#!/bin/bash
# OpenAPI Live Documentation Server
# Starts a local server with Swagger UI for live schema exploration

set -e

ALAB="../../alab.exe"
PORT=${1:-8080}

echo "=== Alab API Documentation Server ==="
echo ""

# Check if alab.exe exists
if [ ! -f "$ALAB" ]; then
    echo "Error: alab.exe not found at $ALAB"
    echo "Build it first with: go build -o alab.exe ./cmd/alab"
    exit 1
fi

$ALAB types
echo "Starting server on port $PORT..."
echo ""
echo "  Swagger UI:  http://localhost:$PORT/"
echo "  OpenAPI:     http://localhost:$PORT/openapi.json"
echo ""
echo "Edit schemas in ./schemas/ and refresh browser to see changes."
echo "Press Ctrl+C to stop."
echo ""

$ALAB http --port $PORT
