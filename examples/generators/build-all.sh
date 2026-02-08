#!/bin/bash
# Build all framework generators from alab schema
# Usage: ./build-all.sh [--check]

set -e  # Exit on error

CHECK_MODE=false
if [[ "$1" == "--check" ]]; then
    CHECK_MODE=true
fi

echo "🚀 Building all framework generators..."
echo ""

# Base directories
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
OUTPUT_DIR="${SCRIPT_DIR}/generated"

# Clean previous builds
if [ -d "$OUTPUT_DIR" ]; then
    echo "🧹 Cleaning previous builds..."
    rm -rf "$OUTPUT_DIR"
fi

# Create output directory
mkdir -p "$OUTPUT_DIR"

# Python - FastAPI
echo "📦 Generating Python (FastAPI)..."
alab gen run generators/fastapi -o "$OUTPUT_DIR/python-fastapi"
echo "✓ Python (FastAPI) generated"
echo ""

# TypeScript - tRPC
echo "📦 Generating TypeScript (tRPC)..."
alab gen run generators/typescript-trpc -o "$OUTPUT_DIR/typescript-trpc"
echo "✓ TypeScript (tRPC) generated"
echo ""

# Rust - Axum
echo "📦 Generating Rust (Axum)..."
alab gen run generators/rust-axum -o "$OUTPUT_DIR/rust-axum"
echo "✓ Rust (Axum) generated"
echo ""

# Go - Chi
echo "📦 Generating Go (Chi)..."
alab gen run generators/go-chi -o "$OUTPUT_DIR/go-chi"
echo "✓ Go (Chi) generated"
echo ""

# Summary
echo "✨ All generators completed successfully!"
echo ""
echo "Generated APIs:"
echo "  📁 $OUTPUT_DIR/python-fastapi"
echo "  📁 $OUTPUT_DIR/typescript-trpc"
echo "  📁 $OUTPUT_DIR/rust-axum"
echo "  📁 $OUTPUT_DIR/go-chi"
echo ""
# Check validity if requested
if [ "$CHECK_MODE" = true ]; then
    echo "🔍 Checking generated code..."
    "${SCRIPT_DIR}/check-all.sh" || true
    echo ""
fi

echo "Next steps:"
echo "  • Check:      ./check-all.sh (verify syntax/compilation)"
echo "  • Test:       ./test-all.sh --auto (start servers & test endpoints)"
echo ""
echo "Or run servers manually:"
echo "  • Python:     cd $OUTPUT_DIR/python-fastapi && uv run fastapi dev main.py"
echo "  • TypeScript: cd $OUTPUT_DIR/typescript-trpc && npm install && npx tsx server.ts"
echo "  • Rust:       cd $OUTPUT_DIR/rust-axum && cargo run"
echo "  • Go:         cd $OUTPUT_DIR/go-chi && go run main.go"
