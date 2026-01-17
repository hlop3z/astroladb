#!/usr/bin/env bash
# Scenario: Initialize a new alab project

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

# Setup clean directory
cd /workspace/project
rm -rf * 2>/dev/null || true

# Demo starts
sleep 0.5

type_cmd "ls -la"
run "ls -la"
sleep 1

type_cmd "alab init"
run "alab init"
sleep 1

type_cmd "tree"
run "tree"
sleep 2

printf "demo$ "
sleep 1
