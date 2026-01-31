"""Basic usage examples for openapi_to_code."""

import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent.parent))

from openapi_to_code import Schema

schema = Schema(Path(__file__).resolve().parent.parent / "openapi.json")

# --- List all tables ---
print("=== Tables ===")
for table in schema:
    prefix = f"{table.namespace}." if table.namespace else ""
    print(f"  {prefix}{table.table} (PK: {table.pk}, timestamps={table.timestamps})")

# --- Columns for each table ---
print("\n=== Columns ===")
for table in schema:
    print(f"\n  {table.table}:")
    for col in table.columns.values():
        tags = []
        if col.fk:
            tags.append(f"FK->{col.fk}")
        if col.semantic:
            tags.append(col.semantic)
        if col.nullable:
            tags.append("nullable")
        if col.default is not None:
            tags.append(f"default={col.default}")
        suffix = f"  ({', '.join(tags)})" if tags else ""
        print(f"    {col.name}: {col.pg_type}{suffix}")

# --- Foreign keys ---
print("\n=== Foreign Keys ===")
for table in schema:
    fks = table.fk_columns
    if fks:
        for name, col in fks.items():
            print(f"  {table.table}.{name} -> {col.ref} ({col.fk})")

# --- Unique columns ---
print("\n=== Unique Columns ===")
for table in schema:
    uniques = table.unique_columns
    if uniques:
        print(f"  {table.table}: {uniques}")

# --- Relationships ---
print("\n=== Relationships ===")
for table in schema:
    for rel in table.relationships.values():
        if rel.type == "many_to_many":
            print(f"  {table.table}.{rel.name}: M2M -> {rel.target} via {rel.through['table']}")
        else:
            print(f"  {table.table}.{rel.name}: {rel.type} -> {rel.target_table}")

# --- Write-only columns ---
print("\n=== Write-Only (hidden from reads) ===")
for table in schema:
    for col in table.columns.values():
        if col.write_only:
            print(f"  {table.table}.{col.name}")

# --- Defaults ---
print("\n=== Columns with Defaults ===")
for table in schema:
    for col in table.columns.values():
        if col.default is not None:
            print(f"  {table.table}.{col.name} = {col.default}")

# --- Soft-delete tables ---
print("\n=== Soft-Delete Tables ===")
for table in schema:
    if table.soft_delete:
        print(f"  {table.table}")

# --- Lookup helpers ---
print("\n=== Lookup by table name ===")
post = schema.by_table("blog_post")
if post:
    print(f"  Found: {post.schema_name} -> {post.table}")

print("\n=== Lookup by namespace ===")
auth_tables = schema.by_namespace("auth")
print(f"  auth namespace: {[t.table for t in auth_tables]}")
