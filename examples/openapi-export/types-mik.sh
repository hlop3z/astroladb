#!/bin/sh

# Usage: types-mik.sh file.rs

file="$1"

if [ -z "$file" ]; then
  echo "Usage: $0 <file.rs>" >&2
  exit 1
fi

# 1) Replace two-line derive + serde(rename_all)
sed -i '
  /^[[:space:]]*#\[derive(Serialize, Deserialize)\][[:space:]]*$/{
    N
    s/#[[:space:]]*derive(Serialize, Deserialize)[[:space:]]*\n#[[:space:]]*serde(rename_all = "PascalCase")[[:space:]]*/#[derive(Type)]/
  }
' "$file"

# 2) Replace remaining single-line derive
sed -i '
  s/^[[:space:]]*#\[derive(Serialize, Deserialize)\][[:space:]]*$/#[derive(Type)]/
' "$file"

# 3) Remove any leftover serde(rename_all) lines
sed -i '
  /^[[:space:]]*#\[serde(rename_all = "PascalCase")\][[:space:]]*$/d
' "$file"

# 4) Replace use serde::{Serialize, Deserialize}; with miniserde
# Use literal braces instead of \{ \} escape
sed -i '
  s/^[[:space:]]*use[[:space:]]\+serde::{Serialize,[[:space:]]*Deserialize};[[:space:]]*$/use miniserde::{json, Value};/
' "$file"

# 5) Replace serde_json::Value with Value
sed -i '
  s/serde_json::Value/Value/g
' "$file"
