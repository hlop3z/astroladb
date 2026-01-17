#!/usr/bin/env bash
set -euo pipefail

# CLI Recording Script
# Produces clean GIFs from terminal demos

SCENARIO="${1:-help}"
OUTPUT_DIR="${OUTPUT_DIR:-/output}"
CAST_FILE="/tmp/${SCENARIO}.cast"
GIF_FILE="${OUTPUT_DIR}/${SCENARIO}.gif"

# Recording settings
COLS="${COLS:-140}"
ROWS="${ROWS:-30}"
IDLE_LIMIT="${IDLE_LIMIT:-2}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m'

log() { echo -e "${BLUE}[recorder]${NC} $*" >&2; }
success() { echo -e "${GREEN}[recorder]${NC} $*" >&2; }
error() { echo -e "${RED}[recorder]${NC} $*" >&2; }

# Check if scenario exists
SCENARIO_FILE="/scenarios/${SCENARIO}.sh"
if [[ ! -f "$SCENARIO_FILE" ]]; then
    error "Scenario not found: $SCENARIO"
    echo "Available scenarios:" >&2
    ls -1 /scenarios/*.sh 2>/dev/null | xargs -I{} basename {} .sh | sed 's/^/  - /' >&2
    exit 1
fi

log "Recording scenario: $SCENARIO"
log "Output: $GIF_FILE"
log "Terminal: ${COLS}x${ROWS}"

# Record with asciinema using bash script directly
log "Starting recording..."
asciinema rec \
    --quiet \
    --cols "$COLS" \
    --rows "$ROWS" \
    --idle-time-limit "$IDLE_LIMIT" \
    --command "bash $SCENARIO_FILE" \
    "$CAST_FILE"

# Convert to GIF
log "Converting to GIF..."
agg \
    --cols "$COLS" \
    --rows "$ROWS" \
    --speed 1.2 \
    --theme monokai \
    --font-size 14 \
    --font-family "DejaVu Sans Mono" \
    "$CAST_FILE" \
    "$GIF_FILE"

success "Recording complete: $GIF_FILE"
ls -lh "$GIF_FILE" | awk '{print "Size: " $5}'
