#!/bin/bash

# Comprehensive Integration Test Runner for AstrolaDB
# Runs all Week 8-9 integration tests with PostgreSQL

set -e

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo "=========================================="
echo "AstrolaDB Integration Test Suite"
echo "Week 8-9: Integration Testing & Validation"
echo "=========================================="
echo ""

# Step 1: Detect PostgreSQL container
echo -e "${YELLOW}Step 1: Detecting PostgreSQL container...${NC}"

POSTGRES_CONTAINER=$(docker ps --filter "ancestor=postgres" --format "{{.Names}}" | head -n 1)

if [ -z "$POSTGRES_CONTAINER" ]; then
    if docker ps | grep -q "postgres"; then
        POSTGRES_CONTAINER=$(docker ps --filter "name=postgres" --format "{{.Names}}" | head -n 1)
    fi
fi

if [ -z "$POSTGRES_CONTAINER" ]; then
    echo -e "${RED}✗ No running PostgreSQL container found${NC}"
    echo ""
    echo "Please start PostgreSQL with:"
    echo "  docker run -d --name astroladb-test-postgres \\"
    echo "    -e POSTGRES_PASSWORD=postgres \\"
    echo "    -e POSTGRES_DB=astroladb_test \\"
    echo "    -p 5432:5432 \\"
    echo "    postgres:15"
    exit 1
fi

echo -e "${GREEN}✓ Found PostgreSQL container: $POSTGRES_CONTAINER${NC}"

# Step 2: Get PostgreSQL connection details
echo ""
echo -e "${YELLOW}Step 2: Getting PostgreSQL connection details...${NC}"

POSTGRES_PORT=$(docker port $POSTGRES_CONTAINER 5432 | cut -d ':' -f 2)
if [ -z "$POSTGRES_PORT" ]; then
    POSTGRES_PORT=5432
fi

POSTGRES_PASSWORD=$(docker exec $POSTGRES_CONTAINER printenv POSTGRES_PASSWORD 2>/dev/null || echo "postgres")
POSTGRES_USER=$(docker exec $POSTGRES_CONTAINER printenv POSTGRES_USER 2>/dev/null || echo "postgres")

echo -e "${GREEN}✓ PostgreSQL running on port: $POSTGRES_PORT${NC}"
echo -e "${GREEN}✓ User: $POSTGRES_USER${NC}"

# Step 3: Set DATABASE_URL
export DATABASE_URL="postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@localhost:${POSTGRES_PORT}/postgres?sslmode=disable"

echo ""
echo -e "${YELLOW}Step 3: Setting DATABASE_URL...${NC}"
echo -e "${GREEN}✓ DATABASE_URL=$DATABASE_URL${NC}"

# Step 4: Verify database connection
echo ""
echo -e "${YELLOW}Step 4: Verifying database connection...${NC}"

if docker exec $POSTGRES_CONTAINER psql -U $POSTGRES_USER -d postgres -c "SELECT 1" > /dev/null 2>&1; then
    echo -e "${GREEN}✓ Database connection successful${NC}"
else
    echo -e "${RED}✗ Cannot connect to database${NC}"
    exit 1
fi

# Step 5: Run integration tests
echo ""
echo -e "${YELLOW}Step 5: Running integration tests...${NC}"
echo ""

# Parse command line arguments
TEST_FILTER=""
if [ "$1" != "" ]; then
    TEST_FILTER="-run $1"
    echo -e "${BLUE}Running specific test: $1${NC}"
else
    echo -e "${BLUE}Running all integration tests${NC}"
fi

echo "=========================================="
echo ""

# Array to track test results
declare -a RESULTS

# Test 1: Large Dataset Backfill (1M rows)
echo -e "${BLUE}Test 1: Large Dataset Backfill (1M rows)${NC}"
echo "Expected duration: 6-10 minutes"
echo ""

if go test -tags=integration -run TestBackfillLargeDataset_1MillionRows -v -timeout 30m ./internal/engine/runner 2>&1; then
    RESULTS+=("✓ Large Dataset Backfill")
    echo ""
    echo -e "${GREEN}✓ Test 1 PASSED${NC}"
else
    RESULTS+=("✗ Large Dataset Backfill")
    echo ""
    echo -e "${RED}✗ Test 1 FAILED${NC}"
fi

echo ""
echo "=========================================="
echo ""

# Test 2: Concurrent Index Creation
echo -e "${BLUE}Test 2: Concurrent Index Creation (No Write Locks)${NC}"
echo "Expected duration: 2-3 minutes"
echo ""

if go test -tags=integration -run TestConcurrentIndexCreation_NoWriteLocks -v -timeout 10m ./internal/engine/runner 2>&1; then
    RESULTS+=("✓ Concurrent Index Creation")
    echo ""
    echo -e "${GREEN}✓ Test 2 PASSED${NC}"
else
    RESULTS+=("✗ Concurrent Index Creation")
    echo ""
    echo -e "${RED}✗ Test 2 FAILED${NC}"
fi

echo ""
echo "=========================================="
echo ""

# Test 3: Phase Failure Recovery
echo -e "${BLUE}Test 3: Phase Failure Recovery (Backfill Resumability)${NC}"
echo "Expected duration: 1-2 minutes"
echo ""

if go test -tags=integration -run TestPhaseFailureRecovery_BackfillResumability -v -timeout 10m ./internal/engine/runner 2>&1; then
    RESULTS+=("✓ Phase Failure Recovery")
    echo ""
    echo -e "${GREEN}✓ Test 3 PASSED${NC}"
else
    RESULTS+=("✗ Phase Failure Recovery")
    echo ""
    echo -e "${RED}✗ Test 3 FAILED${NC}"
fi

echo ""
echo "=========================================="
echo ""

# Test 4: Index Recovery
echo -e "${BLUE}Test 4: Index Creation Recovery${NC}"
echo "Expected duration: < 1 minute"
echo ""

if go test -tags=integration -run TestPhaseFailureRecovery_IndexCreation -v -timeout 5m ./internal/engine/runner 2>&1; then
    RESULTS+=("✓ Index Recovery")
    echo ""
    echo -e "${GREEN}✓ Test 4 PASSED${NC}"
else
    RESULTS+=("✗ Index Recovery")
    echo ""
    echo -e "${RED}✗ Test 4 FAILED${NC}"
fi

echo ""
echo "=========================================="
echo ""

# Test 5: Performance Benchmarks - Backfill
echo -e "${BLUE}Test 5: Performance Benchmarks - Backfill${NC}"
echo "Expected duration: 10-15 minutes"
echo ""

if go test -tags=integration -run TestBenchmarkBackfillPerformance -v -timeout 30m ./internal/engine/runner 2>&1; then
    RESULTS+=("✓ Backfill Performance")
    echo ""
    echo -e "${GREEN}✓ Test 5 PASSED${NC}"
else
    RESULTS+=("✗ Backfill Performance")
    echo ""
    echo -e "${RED}✗ Test 5 FAILED${NC}"
fi

echo ""
echo "=========================================="
echo ""

# Test 6: Performance Benchmarks - Index Creation
echo -e "${BLUE}Test 6: Performance Benchmarks - Index Creation${NC}"
echo "Expected duration: 10-15 minutes"
echo ""

if go test -tags=integration -run TestBenchmarkIndexCreation -v -timeout 30m ./internal/engine/runner 2>&1; then
    RESULTS+=("✓ Index Performance")
    echo ""
    echo -e "${GREEN}✓ Test 6 PASSED${NC}"
else
    RESULTS+=("✗ Index Performance")
    echo ""
    echo -e "${RED}✗ Test 6 FAILED${NC}"
fi

echo ""
echo "=========================================="
echo ""

# Test 7: Performance Benchmarks - DDL Operations
echo -e "${BLUE}Test 7: Performance Benchmarks - DDL Operations${NC}"
echo "Expected duration: 2-3 minutes"
echo ""

if go test -tags=integration -run TestBenchmarkDDLOperations -v -timeout 10m ./internal/engine/runner 2>&1; then
    RESULTS+=("✓ DDL Performance")
    echo ""
    echo -e "${GREEN}✓ Test 7 PASSED${NC}"
else
    RESULTS+=("✗ DDL Performance")
    echo ""
    echo -e "${RED}✗ Test 7 FAILED${NC}"
fi

echo ""
echo "=========================================="
echo ""

# Print final summary
echo -e "${YELLOW}Integration Test Suite Summary${NC}"
echo "=========================================="

PASSED=0
FAILED=0

for result in "${RESULTS[@]}"; do
    if [[ $result == ✓* ]]; then
        echo -e "${GREEN}$result${NC}"
        ((PASSED++))
    else
        echo -e "${RED}$result${NC}"
        ((FAILED++))
    fi
done

echo ""
echo "Total: $PASSED passed, $FAILED failed"
echo "=========================================="

if [ $FAILED -eq 0 ]; then
    echo ""
    echo -e "${GREEN}✓ All integration tests PASSED!${NC}"
    echo ""
    echo "Next steps:"
    echo "  - Review performance metrics above"
    echo "  - Update ENTERPRISE_PLAN.md to mark Week 8-9 complete"
    echo "  - Move to Week 10: Documentation & Release v0.0.9"
    echo ""
    exit 0
else
    echo ""
    echo -e "${RED}✗ Some tests FAILED${NC}"
    echo ""
    echo "Troubleshooting:"
    echo "  - Check test output above for error details"
    echo "  - Verify PostgreSQL has enough disk space"
    echo "  - Check PostgreSQL logs: docker logs $POSTGRES_CONTAINER"
    echo ""
    exit 1
fi
