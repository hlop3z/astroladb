#!/usr/bin/env bash
# Scenario: Full workflow - init, create table, generate migration

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

run() {
    eval "$1"
    sleep 0.3
}

# Setup clean directory
cd /workspace/project
rm -rf * .* 2>/dev/null || true

# Demo starts
sleep 0.5

type_cmd "alab init"
run "alab init"
sleep 0.8

type_cmd "alab table core users"
run "alab table core users"
sleep 0.8

type_cmd "cat schemas/core/users.js"
run "cat schemas/core/users.js"
sleep 1.5

type_cmd "alab new add-users-table"
run "alab new add-users-table"
sleep 1

type_cmd "ls migrations/"
run "ls migrations/"
sleep 2

printf "demo$ "
sleep 1
