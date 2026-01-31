"""Generate CREATE TABLE DDL from the parsed OpenAPI schema."""

import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent.parent))

from openapi_to_code import Schema, Table

schema = Schema(Path(__file__).resolve().parent.parent / "openapi.json")


def generate_create_table(table: Table, dialect: str = "postgres") -> str:
    lines = []
    for col in table.columns.values():
        if col.is_app_only:
            continue
        sql_type = col.sql_type.get(dialect, "TEXT")
        parts = [f"  {col.name} {sql_type}"]
        if not col.nullable and col.name not in table.pk:
            parts.append("NOT NULL")
        if col.default is not None:
            if isinstance(col.default, dict) and "expr" in col.default:
                parts.append(f"DEFAULT {col.default['expr']}")
            elif isinstance(col.default, bool):
                parts.append(f"DEFAULT {'TRUE' if col.default else 'FALSE'}")
            elif isinstance(col.default, str):
                parts.append(f"DEFAULT '{col.default}'")
            else:
                parts.append(f"DEFAULT {col.default}")
        lines.append(" ".join(parts))

    if table.pk:
        pk = ", ".join(table.pk)
        lines.append(f"  PRIMARY KEY ({pk})")

    # FK constraints
    for col in table.columns.values():
        if col.fk:
            ref_parts = col.fk.rsplit(".", 1)
            ref_table = ref_parts[0].replace(".", "_")
            ref_col = ref_parts[1] if len(ref_parts) > 1 else "id"
            constraint = f"  FOREIGN KEY ({col.name}) REFERENCES {ref_table}({ref_col})"
            if col.on_delete:
                constraint += f" ON DELETE {col.on_delete}"
            lines.append(constraint)

    body = ",\n".join(lines)
    return f"CREATE TABLE {table.table} (\n{body}\n);\n"


# Generate for all tables
for table in schema:
    print(generate_create_table(table))
