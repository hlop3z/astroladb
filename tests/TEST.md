# AstrolaDB Test Suite

Clean Python-based test suite with proper separation of concerns.

## Architecture

```
tests/
├── test_framework.py           # Base classes & utilities (DRY)
├── test_basic_migrations.py    # Basic CRUD operations
├── test_default_values.py      # Default value handling (regression tests)
├── test_relationships.py       # Foreign keys & belongs_to
└── run_all.py                  # Test orchestrator
```

## Design Principles

### Separation of Concerns (SOC)

- **test_framework.py**: Infrastructure (TestEnvironment, TestRunner, assertions)
- **test\_\*.py**: Domain-specific tests (basic, defaults, relationships)
- **run_all.py**: Orchestration (running suites, aggregate reporting)

### Don't Repeat Yourself (DRY)

- `TestEnvironment`: Manages temp directories, config files, schema creation
- `TestRunner`: Handles test execution, timing, error handling, reporting
- Common assertions: `assert_command_success`, `assert_migration_contains`, etc.

### Clean Test Structure

Each test follows the pattern:

1. Setup isolated environment (`with TestEnvironment(...)`)
2. Perform actions (write schemas, run migrations)
3. Assert expected outcomes
4. Auto-cleanup on exit

## Running Tests

```bash
# Run all tests
uv run tests/run_all.py

# Run specific test suite
uv run tests/test_basic_migrations.py
uv run tests/test_default_values.py
uv run tests/test_relationships.py
```

## Adding New Tests

1. **Create new test module**: `tests/test_<domain>.py`
2. **Import framework**: `from test_framework import TestEnvironment, TestRunner, ...`
3. **Write test functions**: Each test uses `TestEnvironment` context manager
4. **Create test runner**:

```python
def run_tests():
    runner = TestRunner(verbose=True)
    runner.run_test(test_func1, "Description 1")
    runner.run_test(test_func2, "Description 2")
    return runner.print_summary()
```

5. **Register in run_all.py**: Add module name to `test_modules` list

## Test Categories

### Basic Migrations (`test_basic_migrations.py`)

- Create initial migration
- Apply migrations
- Rollback migrations
- Check status
- Add/drop columns

### Default Values (`test_default_values.py`)

- Default values in create_table
- Default values in add_column
- Regression: no spurious alter_column
- String/integer/boolean defaults
- Multiple defaults per table

### Relationships (`test_relationships.py`)

- belongs_to relationships
- Optional foreign keys
- Circular references
- ON DELETE/UPDATE actions

## Key Features

- **Isolated environments**: Each test runs in temp directory
- **Automatic cleanup**: Context managers handle setup/teardown
- **Rich assertions**: Domain-specific assertion helpers
- **Detailed reporting**: Per-test timing, error messages, summary
- **Regression protection**: Tests for bugs that were fixed
