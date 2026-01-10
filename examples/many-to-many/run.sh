#!/bin/bash
# Many-to-Many Example Runner
# Uses alab.exe from parent directory

set -e

ALAB="../../alab.exe"

echo "=== Many-to-Many Relationship Example ==="
echo ""

# Check if alab.exe exists
if [ ! -f "$ALAB" ]; then
    echo "Error: alab.exe not found at $ALAB"
    echo "Build it first with: go build -o alab.exe ./cmd/alab"
    exit 1
fi

# Clean up previous runs
rm -f example.db
rm -rf migrations/

echo "1. Initializing project..."
$ALAB init
echo ""

echo "2. Checking schemas..."
$ALAB schema:check
echo ""

echo "3. Dumping schema as SQL..."
echo "----------------------------------------"
$ALAB schema:dump
echo "----------------------------------------"
echo ""

echo "4. Saving metadata (many-to-many info)..."
$ALAB schema:metadata --save
echo ""

echo "5. Showing metadata file..."
if [ -f ".alab/metadata.json" ]; then
    echo "----------------------------------------"
    cat .alab/metadata.json
    echo ""
    echo "----------------------------------------"
fi
echo ""

echo "6. Generating migration from schema..."
$ALAB migration:generate initial_schema
echo ""

echo "7. Running migrations..."
$ALAB migration:run
echo ""

echo "8. Migration status..."
$ALAB migration:status
echo ""

echo "9. Checking database tables..."
echo "----------------------------------------"
if command -v sqlite3 &> /dev/null; then
    sqlite3 example.db ".tables"
    echo ""
    echo "Schema for auth_users:"
    sqlite3 example.db ".schema auth_users"
    echo ""
    echo "Schema for auth_roles:"
    sqlite3 example.db ".schema auth_roles"
    echo ""
    echo "Schema for join table (auth_roles_auth_users):"
    sqlite3 example.db ".schema auth_roles_auth_users"
else
    echo "(sqlite3 CLI not found - skipping table inspection)"
    echo "Tables created: auth_users, auth_roles, auth_roles_auth_users"
fi
echo "----------------------------------------"

echo ""
echo "=== Example Complete ==="
echo ""
echo "The many-to-many relationship between users and roles"
echo "automatically generated the join table: auth_roles_auth_users"
