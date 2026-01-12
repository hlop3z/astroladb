"""
Relationship Tests
Tests foreign keys, belongs_to, and relationship handling.
"""
from test_framework import (
    TestEnvironment,
    TestRunner,
    assert_command_success,
    assert_migration_contains
)


def test_belongs_to_relationship():
    """Test belongs_to generates correct foreign key"""
    with TestEnvironment("rel_belongs_to") as env:
        env.write_schema("auth", "user", """
export default table({
  id: col.id(),
  name: col.string(100),
}).timestamps()
""")

        env.write_schema("app", "post", """
export default table({
  id: col.id(),
  title: col.string(200),
  user_id: col.belongs_to(".auth.user"),
}).timestamps()
""")

        result = env.new_migration("initial")
        assert_command_success(result)

        migration = env.read_migration("001")
        assert_migration_contains(migration, 'belongs_to(".auth.user")')


def test_optional_belongs_to():
    """Test optional belongs_to relationship"""
    with TestEnvironment("rel_optional") as env:
        env.write_schema("core", "category", """
export default table({
  id: col.id(),
  name: col.string(100),
}).timestamps()
""")

        env.write_schema("core", "product", """
export default table({
  id: col.id(),
  name: col.string(200),
  category_id: col.belongs_to(".core.category").optional(),
}).timestamps()
""")

        result = env.new_migration("initial")
        assert_command_success(result)

        migration = env.read_migration("001")
        assert_migration_contains(migration, 'optional()')


def test_circular_reference():
    """Test circular references (tree structure)"""
    with TestEnvironment("rel_circular") as env:
        env.write_schema("core", "node", """
export default table({
  id: col.id(),
  name: col.string(100),
  parent_id: col.belongs_to(".core.node").optional(),
}).timestamps()
""")

        result = env.new_migration("initial")
        assert_command_success(result)

        result = env.migrate()
        assert_command_success(result, "Circular reference should not fail")


def test_on_delete_cascade():
    """Test on_delete cascade option"""
    with TestEnvironment("rel_cascade") as env:
        env.write_schema("auth", "user", """
export default table({
  id: col.id(),
  email: col.string(255),
}).timestamps()
""")

        env.write_schema("app", "session", """
export default table({
  id: col.id(),
  token: col.string(64),
  user_id: col.belongs_to(".auth.user").on_delete("cascade"),
}).timestamps()
""")

        result = env.new_migration("initial")
        assert_command_success(result)

        migration = env.read_migration("001")
        assert_migration_contains(migration, 'on_delete("cascade")')


def run_tests():
    """Run all relationship tests"""
    runner = TestRunner(verbose=True)

    runner.run_test(test_belongs_to_relationship, "belongs_to relationship")
    runner.run_test(test_optional_belongs_to, "Optional belongs_to")
    runner.run_test(test_circular_reference, "Circular reference")
    runner.run_test(test_on_delete_cascade, "ON DELETE CASCADE")

    return runner.print_summary()


if __name__ == "__main__":
    import sys
    success = run_tests()
    sys.exit(0 if success else 1)
