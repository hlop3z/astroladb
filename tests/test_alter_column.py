"""
Alter Column Tests
Tests modifying existing columns, including adding/removing defaults.
"""
from test_framework import (
    TestEnvironment,
    TestRunner,
    assert_command_success,
    assert_migration_contains
)


def test_add_default_to_existing_column():
    """Test adding a default to an existing column (proper alter_column usage)"""
    with TestEnvironment("alter_add_default") as env:
        # Initial table with column WITHOUT default
        env.write_schema("app", "task", """
export default table({
  id: col.id(),
  title: col.string(200),
  priority: col.integer(),
}).timestamps()
""")

        env.new_migration("initial")
        env.migrate()

        # Now add default to existing column
        env.write_schema("app", "task", """
export default table({
  id: col.id(),
  title: col.string(200),
  priority: col.integer().default(5),
}).timestamps()
""")

        result = env.new_migration("add_priority_default")
        assert_command_success(result)

        migration = env.read_migration("002")

        # Should have alter_column with set_default
        assert_migration_contains(migration, 'alter_column', "Should use alter_column to add default")
        assert_migration_contains(migration, 'priority', "Should modify priority column")

        # Apply successfully
        result = env.migrate()
        assert_command_success(result, "Adding default to existing column should work")


def test_remove_default_from_column():
    """Test removing a default from an existing column"""
    with TestEnvironment("alter_remove_default") as env:
        # Initial table with default
        env.write_schema("app", "post", """
export default table({
  id: col.id(),
  title: col.string(200),
  views: col.integer().default(0),
}).timestamps()
""")

        env.new_migration("initial")
        env.migrate()

        # Remove the default
        env.write_schema("app", "post", """
export default table({
  id: col.id(),
  title: col.string(200),
  views: col.integer(),
}).timestamps()
""")

        result = env.new_migration("remove_views_default")
        assert_command_success(result)

        migration = env.read_migration("002")

        # Should have alter_column
        assert_migration_contains(migration, 'alter_column')
        assert_migration_contains(migration, 'views')


def test_change_column_type():
    """Test changing column type"""
    with TestEnvironment("alter_type") as env:
        env.write_schema("core", "item", """
export default table({
  id: col.id(),
  code: col.string(10),
}).timestamps()
""")

        env.new_migration("initial")
        env.migrate()

        # Change string to integer
        env.write_schema("core", "item", """
export default table({
  id: col.id(),
  code: col.integer(),
}).timestamps()
""")

        result = env.new_migration("change_code_type")
        assert_command_success(result)

        migration = env.read_migration("002")
        assert_migration_contains(migration, 'alter_column')


def test_make_column_nullable():
    """Test making a NOT NULL column nullable"""
    with TestEnvironment("alter_nullable") as env:
        env.write_schema("app", "user", """
export default table({
  id: col.id(),
  email: col.string(255),
}).timestamps()
""")

        env.new_migration("initial")
        env.migrate()

        # Make email optional
        env.write_schema("app", "user", """
export default table({
  id: col.id(),
  email: col.string(255).optional(),
}).timestamps()
""")

        result = env.new_migration("make_email_optional")
        assert_command_success(result)

        migration = env.read_migration("002")
        assert_migration_contains(migration, 'alter_column')
        assert_migration_contains(migration, 'email')


def run_tests():
    """Run all alter column tests"""
    runner = TestRunner(verbose=True)

    runner.run_test(test_add_default_to_existing_column, "Add default to existing column")
    runner.run_test(test_remove_default_from_column, "Remove default from column")
    runner.run_test(test_change_column_type, "Change column type")
    runner.run_test(test_make_column_nullable, "Make column nullable")

    return runner.print_summary()


if __name__ == "__main__":
    import sys
    success = run_tests()
    sys.exit(0 if success else 1)
