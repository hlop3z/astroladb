#!/bin/bash

# Integration Test Runner for 1M Row Backfill Test
# Automatically detects PostgreSQL, sets up database, and runs test

set -e

echo "=========================================="
echo "AstrolaDB Integration Test Runner"
echo "Test: 1M Row Batched Backfill"
echo "=========================================="
echo ""

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Step 1: Detect PostgreSQL container
echo -e "${YELLOW}Step 1: Detecting PostgreSQL container...${NC}"

# Try to find running PostgreSQL container
POSTGRES_CONTAINER=$(docker ps --filter "ancestor=postgres" --format "{{.Names}}" | head -n 1)

if [ -z "$POSTGRES_CONTAINER" ]; then
    # Try common container names
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

# Get port mapping
POSTGRES_PORT=$(docker port $POSTGRES_CONTAINER 5432 | cut -d ':' -f 2)
if [ -z "$POSTGRES_PORT" ]; then
    POSTGRES_PORT=5432
fi

# Try to get password from container environment
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

# Step 5: Run integration test
echo ""
echo -e "${YELLOW}Step 5: Running integration test...${NC}"
echo "  Test: TestBackfillLargeDataset_1MillionRows"
echo "  Expected duration: 6-10 minutes"
echo "  Rows: 1,000,000"
echo "  Batch size: 5,000"
echo ""
echo "=========================================="
echo ""

# Run the test with proper tags and timeout
go test \
    -tags=integration \
    -run TestBackfillLargeDataset_1MillionRows \
    -v \
    -timeout 30m \
    ./internal/engine/runner

TEST_EXIT_CODE=$?

# Step 6: Report results
echo ""
echo "=========================================="
if [ $TEST_EXIT_CODE -eq 0 ]; then
    echo -e "${GREEN}✓ Integration test PASSED${NC}"
    echo ""
    echo "Next steps:"
    echo "  - Review test output above for performance metrics"
    echo "  - Continue with remaining Week 8-9 tasks:"
    echo "    • Test concurrent index creation"
    echo "    • Test phase failure recovery"
    echo "    • Performance benchmarks"
else
    echo -e "${RED}✗ Integration test FAILED (exit code: $TEST_EXIT_CODE)${NC}"
    echo ""
    echo "Troubleshooting:"
    echo "  - Check test output above for error details"
    echo "  - Verify PostgreSQL has enough disk space (2GB+)"
    echo "  - Check PostgreSQL logs: docker logs $POSTGRES_CONTAINER"
fi
echo "=========================================="

exit $TEST_EXIT_CODE
