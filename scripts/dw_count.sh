#!/usr/bin/env bash
set -euo pipefail

# Downloads Count
gh api repos/hlop3z/astroladb/releases --jq '[.[].assets[].download_count] | add'
