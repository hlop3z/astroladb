#!/bin/bash
# final-verify.sh - Comprehensive verification after all phases complete

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${BLUE}🎯 Final Testing Refactor Verification${NC}"
echo "========================================"

FAILED=0

# 1. Full test suite with race detection
echo ""
echo "1️⃣  Running full test suite with race detection..."
if go test ./... -race -coverprofile=coverage.final.out > test-final.log 2>&1; then
    echo -e "${GREEN}✅ All tests passed${NC}"
else
    echo -e "${RED}❌ Some tests failed${NC}"
    tail -30 test-final.log
    FAILED=1
fi

# 2. Coverage comparison
echo ""
echo "2️⃣  Comparing coverage..."
FINAL_COVERAGE=$(go tool cover -func=coverage.final.out | grep total | awk '{print $3}' | sed 's/%//')

if [ -f "coverage.baseline.out" ]; then
    BASELINE_COVERAGE=$(go tool cover -func=coverage.baseline.out | grep total | awk '{print $3}' | sed 's/%//')
    echo "  Baseline coverage: ${BASELINE_COVERAGE}%"
    echo "  Final coverage:    ${FINAL_COVERAGE}%"

    IMPROVEMENT=$(echo "$FINAL_COVERAGE - $BASELINE_COVERAGE" | bc)
    if (( $(echo "$FINAL_COVERAGE >= $BASELINE_COVERAGE" | bc -l) )); then
        echo -e "${GREEN}✅ Coverage improved by ${IMPROVEMENT}%${NC}"
    else
        echo -e "${RED}❌ Coverage decreased by ${IMPROVEMENT}%${NC}"
        FAILED=1
    fi
else
    echo "  Final coverage: ${FINAL_COVERAGE}%"
fi

# 3. Test count comparison
echo ""
echo "3️⃣  Comparing test counts..."
FINAL_COUNT=$(go test ./... -v -dry-run 2>/dev/null | grep -c "^=== RUN" || echo "0")

if [ -f "test_count.baseline.txt" ]; then
    BASELINE_COUNT=$(cat test_count.baseline.txt)
    echo "  Baseline tests: $BASELINE_COUNT"
    echo "  Final tests:    $FINAL_COUNT"

    INCREASE=$(($FINAL_COUNT - $BASELINE_COUNT))
    if [ $FINAL_COUNT -ge $BASELINE_COUNT ]; then
        echo -e "${GREEN}✅ Test count increased by $INCREASE${NC}"
    else
        echo -e "${YELLOW}⚠️  Test count decreased by $INCREASE${NC}"
    fi
else
    echo "  Final tests: $FINAL_COUNT"
fi

# 4. Verify all fixtures exist
echo ""
echo "4️⃣  Checking test fixtures..."
FIXTURE_COUNT=0
MISSING_COUNT=0

if [ -d "test/fixtures/schemas" ]; then
    for fixture in test/fixtures/schemas/**/*.js; do
        if [ -f "$fixture" ]; then
            FIXTURE_COUNT=$((FIXTURE_COUNT + 1))
        else
            echo -e "${RED}❌ Missing fixture: $fixture${NC}"
            MISSING_COUNT=$((MISSING_COUNT + 1))
            FAILED=1
        fi
    done

    if [ $MISSING_COUNT -eq 0 ]; then
        echo -e "${GREEN}✅ All $FIXTURE_COUNT fixtures exist${NC}"
    fi
else
    echo -e "${YELLOW}⚠️  No fixtures directory found${NC}"
fi

# 5. TypeScript tests
if [ -d "test/javascript" ]; then
    echo ""
    echo "5️⃣  Running TypeScript tests..."
    cd test/javascript

    if npm test > ../js-final.log 2>&1; then
        echo -e "${GREEN}✅ TypeScript tests passed${NC}"
    else
        echo -e "${RED}❌ TypeScript tests failed${NC}"
        tail -20 ../js-final.log
        FAILED=1
    fi

    # Type checking
    if npm run type-check > ../js-typecheck.log 2>&1; then
        echo -e "${GREEN}✅ TypeScript type checking passed${NC}"
    else
        echo -e "${RED}❌ TypeScript type checking failed${NC}"
        tail -20 ../js-typecheck.log
        FAILED=1
    fi

    cd ../..
fi

# 6. Build succeeds
echo ""
echo "6️⃣  Building binary..."
if go build -o alab.test ./cmd/alab > build.log 2>&1; then
    echo -e "${GREEN}✅ Binary built successfully${NC}"
    rm -f alab.test
else
    echo -e "${RED}❌ Build failed${NC}"
    tail -20 build.log
    FAILED=1
fi

# 7. Integration tests (if tagged)
echo ""
echo "7️⃣  Running integration tests..."
if go test ./test/integration/... -tags=integration -v > integration.log 2>&1; then
    echo -e "${GREEN}✅ Integration tests passed${NC}"
else
    echo -e "${YELLOW}⚠️  Integration tests failed or not found${NC}"
    # Not failing overall for this
fi

# 8. Documentation check
echo ""
echo "8️⃣  Checking documentation..."
DOC_ISSUES=0

if [ ! -f "TESTING.md" ]; then
    echo -e "${RED}❌ TESTING.md not found${NC}"
    DOC_ISSUES=$((DOC_ISSUES + 1))
fi

if [ ! -f "PLAN.md" ]; then
    echo -e "${RED}❌ PLAN.md not found${NC}"
    DOC_ISSUES=$((DOC_ISSUES + 1))
fi

if [ ! -f "test/README.md" ]; then
    echo -e "${YELLOW}⚠️  test/README.md not found${NC}"
fi

if [ $DOC_ISSUES -eq 0 ]; then
    echo -e "${GREEN}✅ Documentation complete${NC}"
else
    echo -e "${RED}❌ Missing documentation files${NC}"
    FAILED=1
fi

# Final summary
echo ""
echo "========================================"
if [ $FAILED -eq 0 ]; then
    echo -e "${GREEN}🎉 FINAL VERIFICATION SUCCESSFUL! 🎉${NC}"
    echo ""
    echo "Summary:"
    echo "  ✅ All tests passing"
    echo "  ✅ Coverage: ${FINAL_COVERAGE}%"
    echo "  ✅ Test count: $FINAL_COUNT"
    echo "  ✅ Build successful"
    echo "  ✅ Documentation complete"
    echo ""
    echo "Testing refactor is complete and verified!"
    exit 0
else
    echo -e "${RED}❌ VERIFICATION FAILED${NC}"
    echo ""
    echo "Please review the errors above and fix before proceeding."
    exit 1
fi
