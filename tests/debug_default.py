#!/usr/bin/env python3
"""
Debug test for default value bug.
"""
import sys
import subprocess
from pathlib import Path

sys.path.insert(0, str(Path(__file__).parent))
from test_framework import TestEnvironment

with TestEnvironment("debug_default") as env:
    # Create initial schema
    env.write_schema("app", "post", """
export default table({
  id: col.id(),
  title: col.string(200),
}).timestamps()
""")

    # Generate initial migration
    env.new_migration("initial")
    env.migrate()

    # Add column with default - run with debug logging
    env.write_schema("app", "post", """
export default table({
  id: col.id(),
  title: col.string(200),
  views: col.integer().default(0),
}).timestamps()
""")

    # Run with debug logging
    result = env.run_command(["--log", "debug", "new", "add_views"], input_text="Y\n")

    print("=== DEBUG OUTPUT ===")
    print(result.stderr.decode() if isinstance(result.stderr, bytes) else result.stderr)
    print("\n=== MIGRATION CONTENT ===")
    migration = env.read_migration("002")
    print(migration)

    if ".default(0)" in migration:
        print("\nSUCCESS: default(0) found!")
    else:
        print("\nBUG: default(0) NOT found in migration!")
