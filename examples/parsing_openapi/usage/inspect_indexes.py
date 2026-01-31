"""Inspect indexes across all tables."""

import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent.parent))

from openapi_to_code import Schema

schema = Schema(Path(__file__).resolve().parent.parent / "openapi.json")

for table in schema:
    if not table.indexes:
        continue
    print(f"{table.table}:")
    for idx in table.indexes:
        flags = []
        if idx.primary:
            flags.append("PK")
        if idx.unique and not idx.primary:
            flags.append("UNIQUE")
        label = f" [{', '.join(flags)}]" if flags else ""
        print(f"  {idx.name}: ({', '.join(idx.columns)}){label}")
    print()
