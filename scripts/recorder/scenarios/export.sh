#!/usr/bin/env bash
# Scenario: Export schema to different formats

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

# Setup: init project with a table (silently)
cd /workspace/project
rm -rf * .* 2>/dev/null || true
alab init >/dev/null 2>&1
alab table core users >/dev/null 2>&1

# Demo starts
sleep 0.5

type_cmd "alab export -f typescript"
run "alab export -f typescript"
sleep 1.2

type_cmd "alab export -f graphql"
run "alab export -f graphql"
sleep 1.2

type_cmd "alab export -f openapi"
run "alab export -f openapi"
sleep 2

printf "demo$ "
sleep 1
