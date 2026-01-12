#!/usr/bin/env python3
"""
AstrolaDB Test Suite Runner
Runs all test modules and reports aggregate results.
"""
import sys
import importlib
from pathlib import Path


def main():
    """Run all test suites"""
    print("="*60)
    print("AstrolaDB Test Suite")
    print("="*60)
    print()

    # Add current directory to path so imports work
    sys.path.insert(0, str(Path(__file__).parent))

    test_modules = [
        "test_basic_migrations",
        "test_default_values",
        "test_alter_column",
        "test_relationships",
    ]

    all_passed = True

    for module_name in test_modules:
        print(f"\n{'='*60}")
        print(f"Running: {module_name}")
        print('='*60)

        try:
            module = importlib.import_module(module_name)
            passed = module.run_tests()

            if not passed:
                all_passed = False

        except Exception as e:
            print(f"ERROR: Failed to run {module_name}: {e}")
            all_passed = False

    print("\n" + "="*60)
    print("FINAL SUMMARY")
    print("="*60)

    if all_passed:
        print("[PASS] All test suites PASSED")
        return 0
    else:
        print("[FAIL] Some test suites FAILED")
        return 1


if __name__ == "__main__":
    sys.exit(main())
