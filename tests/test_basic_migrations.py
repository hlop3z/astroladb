"""
Basic Migration Tests
Tests fundamental migration operations: create, apply, rollback.
"""
from test_framework import (
    TestEnvironment,
    TestRunner,
    assert_command_success,
    assert_command_output_contains,
    assert_migration_contains
)


def test_create_initial_migration():
    """Test creating an initial migration"""
    with TestEnvironment("basic_initial") as env:
        # Create a simple schema
        env.write_schema("auth", "user", """
export default table({
  id: col.id(),
  name: col.string(100),
  email: col.string(255),
}).timestamps()
""")

        # Generate migration
        result = env.new_migration("initial")
        assert_command_success(result, "Failed to generate initial migration")

        # Verify migration was created
        migration = env.read_migration("001")
        assert_migration_contains(migration, "create_table")
        assert_migration_contains(migration, "auth.user")


def test_apply_migration():
    """Test applying a migration"""
    with TestEnvironment("basic_apply") as env:
        env.write_schema("auth", "user", """
export default table({
  id: col.id(),
  username: col.string(50),
}).timestamps()
""")

        # Generate and apply
        result = env.new_migration("initial")
        assert_command_success(result)

        result = env.migrate()
        assert_command_success(result, "Failed to apply migration")
        assert_command_output_contains(result, "Applied 1 migration")


def test_rollback_migration():
    """Test rolling back a migration"""
    with TestEnvironment("basic_rollback") as env:
        env.write_schema("app", "task", """
export default table({
  id: col.id(),
  title: col.string(200),
}).timestamps()
""")

        # Generate, apply, then rollback
        env.new_migration("initial")
        env.migrate()

        result = env.rollback()
        assert_command_success(result, "Failed to rollback migration")
        assert_command_output_contains(result, "Rolled back 1 migration")


def test_migration_status():
    """Test checking migration status"""
    with TestEnvironment("basic_status") as env:
        env.write_schema("core", "item", """
export default table({
  id: col.id(),
  name: col.string(100),
}).timestamps()
""")

        env.new_migration("create_items")
        env.migrate()

        result = env.status()
        assert_command_success(result)
        assert_command_output_contains(result, "Applied")


def test_add_column_migration():
    """Test adding a column to an existing table"""
    with TestEnvironment("basic_add_col") as env:
        # Initial schema
        env.write_schema("app", "post", """
export default table({
  id: col.id(),
  title: col.string(200),
}).timestamps()
""")

        env.new_migration("initial")
        env.migrate()

        # Add a column
        env.write_schema("app", "post", """
export default table({
  id: col.id(),
  title: col.string(200),
  content: col.text().optional(),
}).timestamps()
""")

        result = env.new_migration("add_content")
        assert_command_success(result)

        migration = env.read_migration("002")
        assert_migration_contains(migration, "add_column")
        assert_migration_contains(migration, "content")

        result = env.migrate()
        assert_command_success(result, "Failed to apply add_column migration")


def run_tests():
    """Run all basic migration tests"""
    runner = TestRunner(verbose=True)

    runner.run_test(test_create_initial_migration, "Create initial migration")
    runner.run_test(test_apply_migration, "Apply migration")
    runner.run_test(test_rollback_migration, "Rollback migration")
    runner.run_test(test_migration_status, "Migration status")
    runner.run_test(test_add_column_migration, "Add column migration")

    return runner.print_summary()


if __name__ == "__main__":
    import sys
    success = run_tests()
    sys.exit(0 if success else 1)
