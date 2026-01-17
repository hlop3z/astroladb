#!/usr/bin/env bash
# Scenario: Live OpenAPI export as schema evolves

type_cmd() {
    local cmd="$1"
    printf "demo$ "
    for ((i=0; i<${#cmd}; i++)); do
        printf "%s" "${cmd:$i:1}"
        sleep 0.04
    done
    printf "\n"
    sleep 0.2
}

type_file() {
    local content="$1"
    for ((i=0; i<${#content}; i++)); do
        printf "%s" "${content:$i:1}"
        sleep 0.02
    done
}

run() {
    eval "$1"
    sleep 0.3
}

# Setup
cd /workspace/project || exit 1
rm -rf ./* ./.* 2>/dev/null || true
alab init >/dev/null 2>&1
mkdir -p schemas/blog

# Demo starts
sleep 0.5

echo "═══════════════════════════════════════════════════════════════════════════════"
echo "  Live Schema → OpenAPI Export Demo"
echo "═══════════════════════════════════════════════════════════════════════════════"
echo ""
sleep 1

# Step 1: Create initial schema
type_cmd "cat > schemas/blog/post.js << 'EOF'"
cat > schemas/blog/post.js << 'SCHEMA'
export default table({
  id: col.id(),
  title: col.text(200),
});
SCHEMA
cat schemas/blog/post.js
echo "EOF"
sleep 0.5

# Export OpenAPI
echo ""
type_cmd "alab export -f openapi"
run "alab export -f openapi"
sleep 2

# Step 2: Add more fields
echo ""
echo "─────────────────────────────────────────────────────────────────────────────"
echo "  Adding more fields..."
echo "─────────────────────────────────────────────────────────────────────────────"
sleep 0.5

type_cmd "cat > schemas/blog/post.js << 'EOF'"
cat > schemas/blog/post.js << 'SCHEMA'
export default table({
  id: col.id(),
  title: col.text(200),
  content: col.text(),
  author_id: col.fk("auth.user"),
  is_published: col.flag(false),
}).timestamps();
SCHEMA
cat schemas/blog/post.js
echo "EOF"
sleep 0.5

# Export again
echo ""
type_cmd "alab export -f openapi"
run "alab export -f openapi"
sleep 2

# Step 3: Show GraphQL too
echo ""
echo "─────────────────────────────────────────────────────────────────────────────"
echo "  Now export as GraphQL..."
echo "─────────────────────────────────────────────────────────────────────────────"
sleep 0.5

type_cmd "alab export -f graphql"
run "alab export -f graphql"
sleep 2

printf "demo$ "
sleep 1
