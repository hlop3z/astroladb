"""Quick demo: load openapi.json and print a summary."""

from pathlib import Path
from openapi_to_code import Schema

schema = Schema(Path(__file__).resolve().parent / "openapi.json")

for table in schema:
    prefix = f"{table.namespace}." if table.namespace else ""
    print(f"\n{prefix}{table.table} ({table.schema_name})")
    print(f"  PK: {table.pk}  timestamps={table.timestamps}  soft_delete={table.soft_delete}")

    for col in table.columns.values():
        tags: list[str] = []
        if col.fk:
            tags.append(f"FK->{col.fk}")
        if col.semantic:
            tags.append(col.semantic)
        if col.default is not None:
            tags.append(f"default={col.default}")
        print(f"  {col.name}: {col.pg_type} {' '.join(tags)}")

    for rel in table.relationships.values():
        print(f"  rel {rel.name}: {rel.type} -> {rel.target}")
