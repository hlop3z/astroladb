"""Example: Recreate all types and constraints from OpenAPI export.

This demonstrates how the new type_args field makes it easy to
programmatically recreate exact type definitions without parsing SQL strings.
"""

import sys
from pathlib import Path

# Add parent directory to path
sys.path.insert(0, str(Path(__file__).parent.parent))

from openapi_to_code import Schema


def recreate_column_definition(schema: Schema, table_name: str, column_name: str) -> str:
    """Recreate the original column definition from OpenAPI metadata."""
    table = schema.by_table(table_name)
    if not table:
        return f"Table '{table_name}' not found"

    col = table[column_name]

    # Determine the actual column type (use format for special types like decimal)
    col_type = col.semantic or col.format or col.type

    # Build the column definition
    parts = [f"col.{col_type}"]

    # Add type arguments
    if col.type_args:
        if col.format == "decimal" or col_type == "decimal":
            # col.decimal(19, 4)
            parts.append(f"({col.decimal_precision}, {col.decimal_scale})")
        elif col.enum:
            # col.enum(['active', 'pending', 'suspended'])
            values = ", ".join(f"'{v}'" for v in col.enum)
            parts.append(f"([{values}])")
        elif col.type == "string":
            # col.string(255)
            parts.append(f"({col.string_length})")

    # Add constraints
    constraints = []
    if col.nullable:
        constraints.append(".nullable()")
    # Note: unique is stored in table indexes, not on the column itself
    if col.default is not None and not col.auto_managed:
        if isinstance(col.default, bool):
            constraints.append(f".default({str(col.default).lower()})")
        elif isinstance(col.default, str):
            constraints.append(f".default('{col.default}')")
        elif isinstance(col.default, (int, float)):
            constraints.append(f".default({col.default})")

    return "".join(parts) + "".join(constraints)


def main():
    # Load the OpenAPI schema from the basic example (which has more columns)
    # Use relative path from parsing_openapi directory
    openapi_path = Path(__file__).parent.parent.parent / "basic" / "exports" / "openapi.json"
    if not openapi_path.exists():
        # Fallback: try relative from current directory
        openapi_path = Path("../basic/exports/openapi.json")
    schema = Schema(str(openapi_path))

    print("=" * 70)
    print("Recreating Column Definitions from OpenAPI")
    print("=" * 70)

    # Example: Recreate various column types
    examples = [
        ("auth_user", "username", "String with length"),
        ("auth_user", "balance", "Decimal with precision/scale"),
        ("auth_user", "role", "Enum with values"),
        ("auth_user", "is_active", "Boolean with default"),
        ("auth_user", "age", "Nullable integer"),
    ]

    for table_name, col_name, description in examples:
        table = schema.by_table(table_name)
        if table and col_name in table.columns:
            col = table[col_name]
            definition = recreate_column_definition(schema, table_name, col_name)

            print(f"\n{description}:")
            print(f"  Column: {col_name}")
            print(f"  Type: {col.type}")
            if col.type_args:
                print(f"  Type Args: {col.type_args}")
            if col.sql_type:
                print(f"  SQL (PostgreSQL): {col.pg_type}")
            print(f"  Recreated: {col_name}: {definition}")

    # Demonstrate direct access to type constraints
    print("\n" + "=" * 70)
    print("Direct Access to Type Constraints")
    print("=" * 70)

    balance = schema.by_table("auth_user")["balance"]
    print(f"\nDecimal field 'balance':")
    print(f"  Precision: {balance.decimal_precision}")
    print(f"  Scale: {balance.decimal_scale}")
    print(f"  -> NUMERIC({balance.decimal_precision},{balance.decimal_scale})")

    username = schema.by_table("auth_user")["username"]
    print(f"\nString field 'username':")
    print(f"  Max Length: {username.string_length}")
    print(f"  -> VARCHAR({username.string_length})")

    role = schema.by_table("auth_user")["role"]
    print(f"\nEnum field 'role':")
    print(f"  Values: {role.enum_values}")
    print(f"  -> ENUM({', '.join(repr(v) for v in role.enum_values)})")


if __name__ == "__main__":
    main()
