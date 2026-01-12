"""
AstrolaDB Test Framework
Provides base classes and utilities for testing migrations.
"""
import os
import shutil
import subprocess
import tempfile
from pathlib import Path
from typing import Optional, List, Dict, Any
from dataclasses import dataclass
from enum import Enum


class TestResult(Enum):
    PASSED = "passed"
    FAILED = "failed"
    SKIPPED = "skipped"


@dataclass
class TestCase:
    """Represents a single test case"""
    name: str
    result: TestResult
    error: Optional[str] = None
    duration_ms: int = 0


class TestEnvironment:
    """Manages an isolated test environment with its own database and schemas"""

    def __init__(self, name: str, dialect: str = "sqlite"):
        self.name = name
        self.dialect = dialect
        self.tmpdir: Optional[Path] = None
        self.config_path: Optional[Path] = None

    def __enter__(self):
        """Setup test environment"""
        self.tmpdir = Path(tempfile.mkdtemp(prefix=f"alab_test_{self.name}_"))

        # Create directory structure
        (self.tmpdir / "schemas").mkdir()
        (self.tmpdir / "migrations").mkdir()

        # Create config file
        db_url = f"./{self.name}.db" if self.dialect == "sqlite" else "postgresql://..."
        config_content = f"""database:
  dialect: {self.dialect}
  url: {db_url}
schemas: ./schemas
migrations: ./migrations
"""
        self.config_path = self.tmpdir / "alab.yaml"
        self.config_path.write_text(config_content)

        return self

    def __exit__(self, exc_type, exc_val, exc_tb):
        """Cleanup test environment"""
        if self.tmpdir and self.tmpdir.exists():
            shutil.rmtree(self.tmpdir)

    def write_schema(self, namespace: str, table: str, content: str):
        """Write a schema file"""
        schema_dir = self.tmpdir / "schemas" / namespace
        schema_dir.mkdir(parents=True, exist_ok=True)
        (schema_dir / f"{table}.js").write_text(content)

    def read_migration(self, revision: str) -> str:
        """Read a migration file"""
        migrations = list((self.tmpdir / "migrations").glob(f"{revision}_*.js"))
        if not migrations:
            raise FileNotFoundError(f"Migration {revision} not found")
        return migrations[0].read_text()

    def run_command(self, cmd: List[str], input_text: str = "") -> subprocess.CompletedProcess:
        """Run an alab command in the test environment"""
        # Use the alab binary from the project root
        import os
        project_root = Path(__file__).parent.parent
        alab_path = project_root / "alab.exe" if os.name == "nt" else project_root / "alab"
        full_cmd = [str(alab_path)] + cmd
        result = subprocess.run(
            full_cmd,
            cwd=self.tmpdir,
            input=input_text,
            capture_output=True,
            text=True
        )
        return result

    def new_migration(self, name: str, auto_yes: bool = True) -> subprocess.CompletedProcess:
        """Generate a new migration"""
        input_text = "Y\n" if auto_yes else ""
        return self.run_command(["new", name], input_text)

    def migrate(self, auto_yes: bool = True) -> subprocess.CompletedProcess:
        """Apply pending migrations"""
        input_text = "Y\n" if auto_yes else ""
        return self.run_command(["migrate"], input_text)

    def rollback(self, steps: int = 1, auto_yes: bool = True) -> subprocess.CompletedProcess:
        """Rollback migrations"""
        input_text = "y\n" if auto_yes else ""
        cmd = ["rollback", str(steps)] if steps > 1 else ["rollback"]
        return self.run_command(cmd, input_text)

    def status(self) -> subprocess.CompletedProcess:
        """Get migration status"""
        return self.run_command(["status"])


class TestRunner:
    """Orchestrates test execution and reporting"""

    def __init__(self, verbose: bool = False):
        self.verbose = verbose
        self.results: List[TestCase] = []

    def run_test(self, test_func, name: str) -> TestCase:
        """Run a single test function"""
        import time

        print(f"Running: {name}...", end=" ")
        start_time = time.time()

        try:
            test_func()
            duration_ms = int((time.time() - start_time) * 1000)
            result = TestCase(name=name, result=TestResult.PASSED, duration_ms=duration_ms)
            print(f"[PASS] ({duration_ms}ms)")
        except AssertionError as e:
            duration_ms = int((time.time() - start_time) * 1000)
            result = TestCase(name=name, result=TestResult.FAILED, error=str(e), duration_ms=duration_ms)
            print(f"[FAIL] {e}")
        except Exception as e:
            duration_ms = int((time.time() - start_time) * 1000)
            result = TestCase(name=name, result=TestResult.FAILED, error=f"Exception: {e}", duration_ms=duration_ms)
            print(f"[ERROR] {e}")

        self.results.append(result)
        return result

    def print_summary(self):
        """Print test execution summary"""
        total = len(self.results)
        passed = sum(1 for r in self.results if r.result == TestResult.PASSED)
        failed = sum(1 for r in self.results if r.result == TestResult.FAILED)
        total_time = sum(r.duration_ms for r in self.results)

        print("\n" + "="*60)
        print("TEST SUMMARY")
        print("="*60)
        print(f"Total:  {total}")
        print(f"Passed: {passed}")
        print(f"Failed: {failed}")
        print(f"Time:   {total_time}ms")

        if failed > 0:
            print("\nFailed tests:")
            for result in self.results:
                if result.result == TestResult.FAILED:
                    print(f"  - {result.name}")
                    if result.error:
                        print(f"    {result.error}")

        return failed == 0


def assert_migration_contains(migration_content: str, expected: str, message: str = ""):
    """Assert that migration contains expected content"""
    if expected not in migration_content:
        raise AssertionError(message or f"Migration does not contain: {expected}")


def assert_command_success(result: subprocess.CompletedProcess, message: str = ""):
    """Assert that a command completed successfully"""
    if result.returncode != 0:
        error_msg = message or f"Command failed with code {result.returncode}"
        error_msg += f"\nStdout: {result.stdout}\nStderr: {result.stderr}"
        raise AssertionError(error_msg)


def assert_command_output_contains(result: subprocess.CompletedProcess, expected: str, message: str = ""):
    """Assert that command output contains expected text"""
    output = result.stdout + result.stderr
    if expected not in output:
        raise AssertionError(message or f"Output does not contain: {expected}")
