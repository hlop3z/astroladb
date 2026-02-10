# Testing Refactor - Baseline Metrics

**Date**: 2026-02-09
**Status**: ✅ Baseline Established

## Summary

| Metric             | Value           |
| ------------------ | --------------- |
| **Total Coverage** | **45.8%**       |
| **Test Count**     | **2,683 tests** |
| **All Tests**      | ✅ PASSING      |

## Coverage by Package

| Package               | Coverage | Status        |
| --------------------- | -------- | ------------- |
| `internal/sqlgen`     | 100.0%   | 🟢 Excellent  |
| `internal/registry`   | 99.2%    | 🟢 Excellent  |
| `internal/cli`        | 97.4%    | 🟢 Excellent  |
| `internal/types`      | 97.7%    | 🟢 Excellent  |
| `internal/validate`   | 94.2%    | 🟢 Excellent  |
| `internal/strutil`    | 90.0%    | 🟢 Good       |
| `internal/metadata`   | 89.6%    | 🟢 Good       |
| `internal/dialect`    | 62.0%    | 🟡 Needs work |
| `internal/runtime`    | 60.5%    | 🟡 Needs work |
| `internal/dsl`        | 59.3%    | 🟡 Needs work |
| `internal/engine`     | 57.8%    | 🟡 Needs work |
| `internal/lockfile`   | 57.0%    | 🟡 Needs work |
| `internal/jsutil`     | 53.1%    | 🟡 Needs work |
| `internal/git`        | 51.1%    | 🟡 Needs work |
| `internal/testutil`   | 46.3%    | 🔴 Low        |
| `pkg/astroladb`       | 42.1%    | 🔴 Low        |
| `internal/drift`      | 40.7%    | 🔴 Low        |
| `internal/introspect` | 32.4%    | 🔴 Low        |
| `internal/devdb`      | 21.4%    | 🔴 Very Low   |
| `internal/ui`         | 11.0%    | 🔴 Very Low   |

## Priority Areas for Improvement

### High Priority (Target: 80%+)

1. `internal/runtime` (60.5% → 85%) - Core functionality
2. `internal/engine` (57.8% → 85%) - Migration engine
3. `internal/introspect` (32.4% → 75%) - Database introspection

### Medium Priority (Target: 70%+)

4. `internal/drift` (40.7% → 70%) - Schema drift detection
5. `internal/git` (51.1% → 70%) - Git operations
6. `pkg/astroladb` (42.1% → 70%) - Public API

### Low Priority (Target: 60%+)

7. `internal/devdb` (21.4% → 60%) - Dev database utilities
8. `internal/ui` (11.0% → 50%) - UI components (TUI)

## Goals

### Phase 5 Targets

- **Overall Coverage**: 45.8% → **80%+**
- **Test Count**: 2,683 → **3,200+** (20% increase)
- **Integration Tests**: Move to `test/integration/`
- **JS Fixtures**: Extract to `test/fixtures/`
- **TypeScript Tests**: Add to `test/javascript/`

## Next Steps

1. ✅ **Phase 0 Complete** - Baseline established
2. 🔵 **Phase 1 Starting** - Create test utilities
3. ⚪ **Phase 2** - Extract JS fixtures
4. ⚪ **Phase 3** - Add TypeScript tests
5. ⚪ **Phase 4** - Move integration tests
6. ⚪ **Phase 5** - Add new tests
7. ⚪ **Phase 6** - Documentation

---

**Files Created:**

- `coverage.baseline.out` - Coverage profile
- `coverage.baseline.txt` - Coverage report
- `test_count.baseline.txt` - Test count

**Git Checkpoint:**

```bash
git tag testing-refactor-start
```
