#!/usr/bin/env bash
# Scenario: Show alab help

# Typing simulation function
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

# Run command and show output
run() {
    eval "$1"
    sleep 0.3
}

# Demo starts here
sleep 0.5
type_cmd "alab --help"
run "alab --help"
sleep 2
printf "demo$ "
sleep 1
