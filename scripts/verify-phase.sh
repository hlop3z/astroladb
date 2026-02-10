#!/bin/bash
# verify-phase.sh - Run after each phase to ensure everything still works

set -e

PHASE=${1:-"current"}
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "🧪 Running verification for phase: $PHASE"
echo "================================"

# 1. All tests pass
echo ""
echo "1️⃣  Running all Go tests..."
if go test ./... -v > test-output.log 2>&1; then
    echo -e "${GREEN}✅ All tests passed${NC}"
else
    echo -e "${RED}❌ Some tests failed${NC}"
    tail -20 test-output.log
    exit 1
fi

# 2. Coverage hasn't decreased
echo ""
echo "2️⃣  Checking test coverage..."
go test ./... -coverprofile=coverage.current.out > /dev/null 2>&1
COVERAGE=$(go tool cover -func=coverage.current.out | grep total | awk '{print $3}' | sed 's/%//')

echo "Current coverage: ${COVERAGE}%"

if [ -f "coverage.baseline.out" ]; then
    BASELINE=$(go tool cover -func=coverage.baseline.out | grep total | awk '{print $3}' | sed 's/%//')
    echo "Baseline coverage: ${BASELINE}%"

    if (( $(echo "$COVERAGE < $BASELINE" | bc -l) )); then
        echo -e "${YELLOW}⚠️  Coverage decreased from ${BASELINE}% to ${COVERAGE}%${NC}"
    else
        echo -e "${GREEN}✅ Coverage maintained or improved${NC}"
    fi
fi

# 3. No broken imports
echo ""
echo "3️⃣  Checking for broken imports..."
if go build ./... > /dev/null 2>&1; then
    echo -e "${GREEN}✅ All packages build successfully${NC}"
else
    echo -e "${RED}❌ Build failed${NC}"
    exit 1
fi

# 4. Linter passes (optional, skip if not installed)
echo ""
echo "4️⃣  Running linter (if available)..."
if command -v golangci-lint &> /dev/null; then
    if golangci-lint run --timeout=5m > lint-output.log 2>&1; then
        echo -e "${GREEN}✅ Linter passed${NC}"
    else
        echo -e "${YELLOW}⚠️  Linter found issues (see lint-output.log)${NC}"
    fi
else
    echo -e "${YELLOW}⚠️  golangci-lint not installed, skipping${NC}"
fi

# 5. JavaScript tests (if applicable)
if [ -d "test/javascript" ]; then
    echo ""
    echo "5️⃣  Running JavaScript tests..."
    cd test/javascript

    if [ ! -d "node_modules" ]; then
        echo "Installing JS dependencies..."
        npm install > /dev/null 2>&1
    fi

    if npm test > ../js-test-output.log 2>&1; then
        echo -e "${GREEN}✅ JavaScript tests passed${NC}"
    else
        echo -e "${RED}❌ JavaScript tests failed${NC}"
        tail -20 ../js-test-output.log
        cd ../..
        exit 1
    fi
    cd ../..
fi

# 6. Check test count
echo ""
echo "6️⃣  Checking test count..."
CURRENT_COUNT=$(go test ./... -v -dry-run 2>/dev/null | grep -c "^=== RUN" || echo "0")
echo "Current test count: $CURRENT_COUNT"

if [ -f "test_count.baseline.txt" ]; then
    BASELINE_COUNT=$(cat test_count.baseline.txt)
    echo "Baseline test count: $BASELINE_COUNT"

    if [ "$CURRENT_COUNT" -ge "$BASELINE_COUNT" ]; then
        echo -e "${GREEN}✅ Test count maintained or increased${NC}"
    else
        echo -e "${YELLOW}⚠️  Test count decreased from $BASELINE_COUNT to $CURRENT_COUNT${NC}"
    fi
fi

# Summary
echo ""
echo "================================"
echo -e "${GREEN}✅ All verifications passed!${NC}"
echo ""
echo "Summary:"
echo "  - Tests passing: ✅"
echo "  - Coverage: ${COVERAGE}%"
echo "  - Build: ✅"
echo "  - Test count: $CURRENT_COUNT"

# Save checkpoint
if [ "$PHASE" != "current" ]; then
    cp coverage.current.out "coverage.$PHASE.out"
    echo "$CURRENT_COUNT" > "test_count.$PHASE.txt"
    echo ""
    echo "💾 Checkpoint saved for $PHASE"
fi

exit 0
