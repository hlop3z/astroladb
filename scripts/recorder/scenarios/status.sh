#!/usr/bin/env bash
# Scenario: Show migration status (text mode)

type_cmd() {
    local cmd="$1"
    printf "demo$ "
    for ((i=0; i<${#cmd}; i++)); do
        printf "%s" "${cmd:$i:1}"
        sleep 0.05
    done
    printf "\n"
    sleep 0.2
}

run() {
    eval "$1"
    sleep 0.3
}

# Setup: init project with a table and migration (silently)
cd /workspace/project || exit 1
rm -rf ./* ./.* 2>/dev/null || true
alab init >/dev/null 2>&1
alab table core users >/dev/null 2>&1
alab new add-users-table >/dev/null 2>&1

# Demo starts
sleep 0.5

# Show status (TUI mode) - cycle through tabs then quit
type_cmd "alab status"

# Run TUI with unbuffer to preserve PTY, pipe keystrokes
{
    sleep 1
    printf "1"
    sleep 1.5
    printf "2"
    sleep 1.5
    printf "3"
    sleep 1.5
    printf "4"
    sleep 1.5
    printf "q"
} | unbuffer -p alab status

sleep 0.5

printf "demo$ "
sleep 1
