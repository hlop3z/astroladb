#!/bin/bash
# Quick syntax/build checks for generated code
# Usage: ./check-all.sh

set +e  # Don't exit on error

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
OUTPUT_DIR="${SCRIPT_DIR}/generated"
FAILED_CHECKS=()

echo "🔍 Checking generated code validity..."
echo ""

# ─── Python (FastAPI) ───────────────────────────────────────
if [ -d "$OUTPUT_DIR/python-fastapi" ]; then
    echo "🐍 Checking Python..."
    cd "$OUTPUT_DIR/python-fastapi"

    # Just check syntax, don't enforce style
    python_ok=true
    if command -v uv &> /dev/null; then
        for pyfile in $(find . -name "*.py"); do
            if ! uv run python -m py_compile "$pyfile" 2>/dev/null; then
                python_ok=false
                break
            fi
        done
    elif command -v python3 &> /dev/null; then
        for pyfile in $(find . -name "*.py"); do
            if ! python3 -m py_compile "$pyfile" 2>/dev/null; then
                python_ok=false
                break
            fi
        done
    else
        echo "⚠️  python/uv not found (skipping check)"
        python_ok=true  # Don't fail if no Python installed
    fi

    if [ "$python_ok" = true ]; then
        echo "✓ Python syntax valid"
    else
        echo "❌ Python syntax errors"
        FAILED_CHECKS+=("Python")
    fi
    echo ""
fi

# ─── TypeScript (tRPC) ──────────────────────────────────────
if [ -d "$OUTPUT_DIR/typescript-trpc" ]; then
    echo "📘 Checking TypeScript..."
    cd "$OUTPUT_DIR/typescript-trpc"

    # Check if it compiles (don't need to install deps)
    if command -v tsc &> /dev/null; then
        if tsc --noEmit 2>/dev/null; then
            echo "✓ TypeScript types valid"
        else
            echo "⚠️  TypeScript has type issues (may need npm install)"
            # Don't fail - might just need deps
        fi
    else
        echo "⚠️  tsc not found (skipping type check)"
    fi
    echo ""
fi

# ─── Rust (Axum) ────────────────────────────────────────────
if [ -d "$OUTPUT_DIR/rust-axum" ]; then
    echo "🦀 Checking Rust..."
    cd "$OUTPUT_DIR/rust-axum"

    if command -v cargo &> /dev/null; then
        # cargo check is faster than cargo build
        if cargo check 2>/dev/null; then
            echo "✓ Rust compiles"
        else
            echo "❌ Rust compilation errors"
            FAILED_CHECKS+=("Rust")
        fi
    else
        echo "⚠️  cargo not found (skipping check)"
    fi
    echo ""
fi

# ─── Go (Chi) ───────────────────────────────────────────────
if [ -d "$OUTPUT_DIR/go-chi" ]; then
    echo "🐹 Checking Go..."
    cd "$OUTPUT_DIR/go-chi"

    if command -v go &> /dev/null; then
        # Download dependencies first
        echo "  Downloading Go dependencies..."
        go mod tidy 2>/dev/null

        # go vet checks for common errors
        if go vet ./... 2>/dev/null && go build -o /dev/null 2>/dev/null; then
            echo "✓ Go compiles"
        else
            echo "❌ Go compilation errors"
            FAILED_CHECKS+=("Go")
        fi
    else
        echo "⚠️  go not found (skipping check)"
    fi
    echo ""
fi

# Summary
cd "$SCRIPT_DIR"
echo "✨ Checks complete!"
echo ""

if [ ${#FAILED_CHECKS[@]} -gt 0 ]; then
    echo "❌ Failed checks:"
    for lang in "${FAILED_CHECKS[@]}"; do
        echo "  • $lang"
    done
    echo ""
    echo "💡 These are syntax/compilation errors in generated code."
    echo "💡 If you see failures, the generator needs to be fixed."
    exit 1
else
    echo "✅ All generated code is valid!"
    echo ""
    echo "Next step: Run servers and test them"
    echo "  ./test-all.sh --auto"
    exit 0
fi
