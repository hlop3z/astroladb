"""
Default Values Tests
Tests that default values are properly preserved in migrations.
This is a regression test for the bug where .default() was not included in add_column.
"""
from test_framework import (
    TestEnvironment,
    TestRunner,
    assert_command_success,
    assert_migration_contains
)


def test_default_value_in_create_table():
    """Test that default values are included in create_table migrations"""
    with TestEnvironment("default_create") as env:
        env.write_schema("auth", "user", """
export default table({
  id: col.id(),
  name: col.string(100),
  count: col.integer().default(0),
  active: col.flag(true),
}).timestamps()
""")

        result = env.new_migration("initial")
        assert_command_success(result)

        migration = env.read_migration("001")
        assert_migration_contains(migration, '.default(0)', "Integer default not in migration")
        assert_migration_contains(migration, '.default(true)', "Boolean default not in migration")


def test_default_value_in_add_column():
    """Test that default values are included in add_column migrations"""
    with TestEnvironment("default_add") as env:
        # Initial schema without defaults
        env.write_schema("app", "post", """
export default table({
  id: col.id(),
  title: col.string(200),
}).timestamps()
""")

        env.new_migration("initial")
        env.migrate()

        # Add column with default
        env.write_schema("app", "post", """
export default table({
  id: col.id(),
  title: col.string(200),
  views: col.integer().default(0),
}).timestamps()
""")

        result = env.new_migration("add_views")
        assert_command_success(result)

        migration = env.read_migration("002")
        assert_migration_contains(migration, 'add_column', "Not an add_column migration")
        assert_migration_contains(migration, '.default(0)', "Default value missing in add_column")


def test_no_spurious_alter_column():
    """Test that adding a NEW column with default includes the default immediately"""
    with TestEnvironment("default_no_alter") as env:
        # Initial table
        env.write_schema("core", "item", """
export default table({
  id: col.id(),
  name: col.string(100),
}).timestamps()
""")

        env.new_migration("initial")
        env.migrate()

        # Add column WITH default - should have .default() in add_column
        env.write_schema("core", "item", """
export default table({
  id: col.id(),
  name: col.string(100),
  count: col.integer().default(0),
}).timestamps()
""")

        result = env.new_migration("add_count")
        assert_command_success(result)

        migration = env.read_migration("002")

        # Should have add_column with .default()
        assert_migration_contains(migration, 'add_column', "Missing add_column")
        assert_migration_contains(migration, 'count', "Missing count column")
        assert_migration_contains(migration, '.default(0)', "Missing default in add_column")

        # Apply successfully
        result = env.migrate()
        assert_command_success(result)

        # Add another column - should NOT trigger alter_column on count
        env.write_schema("core", "item", """
export default table({
  id: col.id(),
  name: col.string(100),
  count: col.integer().default(0),
  active: col.flag(true),
}).timestamps()
""")

        result = env.new_migration("add_active")
        assert_command_success(result)

        migration = env.read_migration("003")

        # Should only have add_column for active, no alter_column for count
        assert_migration_contains(migration, 'add_column')
        assert_migration_contains(migration, 'active')

        # Should NOT have alter_column for count (regression test)
        if 'alter_column' in migration and 'count' in migration:
            raise AssertionError("Bug regression: alter_column generated for column that already has default")

        result = env.migrate()
        assert_command_success(result)


def test_string_default_value():
    """Test string default values are properly quoted"""
    with TestEnvironment("default_string") as env:
        env.write_schema("app", "post", """
export default table({
  id: col.id(),
  title: col.string(200),
  status: col.string(20).default("draft"),
}).timestamps()
""")

        result = env.new_migration("initial")
        assert_command_success(result)

        migration = env.read_migration("001")
        assert_migration_contains(migration, '.default("draft")', "String default not properly quoted")


def test_multiple_defaults_same_table():
    """Test multiple columns with different default types"""
    with TestEnvironment("default_multiple") as env:
        # Start simple
        env.write_schema("analytics", "event", """
export default table({
  id: col.id(),
  name: col.string(100),
}).timestamps()
""")

        env.new_migration("initial")
        env.migrate()

        # Add multiple columns with defaults
        env.write_schema("analytics", "event", """
export default table({
  id: col.id(),
  name: col.string(100),
  count: col.integer().default(0),
  enabled: col.flag(false),
  status: col.string(20).default("pending"),
}).timestamps()
""")

        result = env.new_migration("add_defaults")
        assert_command_success(result)

        migration = env.read_migration("002")
        assert_migration_contains(migration, '.default(0)')
        assert_migration_contains(migration, '.default(false)')
        assert_migration_contains(migration, '.default("pending")')

        # Apply successfully
        result = env.migrate()
        assert_command_success(result)


def run_tests():
    """Run all default value tests"""
    runner = TestRunner(verbose=True)

    runner.run_test(test_default_value_in_create_table, "Default values in create_table")
    runner.run_test(test_default_value_in_add_column, "Default values in add_column")
    runner.run_test(test_no_spurious_alter_column, "No spurious alter_column (regression)")
    runner.run_test(test_string_default_value, "String default values")
    runner.run_test(test_multiple_defaults_same_table, "Multiple defaults same table")

    return runner.print_summary()


if __name__ == "__main__":
    import sys
    success = run_tests()
    sys.exit(0 if success else 1)
