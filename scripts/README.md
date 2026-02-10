# Testing Refactor Scripts

Helper scripts for the testing infrastructure refactor.

## Usage

### Verify After Each Phase

Run after completing each phase to ensure nothing broke:

```bash
./scripts/verify-phase.sh phase1
./scripts/verify-phase.sh phase2
./scripts/verify-phase.sh phase3
# etc...
```

This will:
- ✅ Run all tests
- ✅ Check coverage
- ✅ Verify builds
- ✅ Run linter (if available)
- ✅ Run JavaScript tests (if applicable)
- 💾 Save checkpoint for the phase

### Final Verification

Run after completing all phases:

```bash
./scripts/final-verify.sh
```

This will:
- ✅ Run full test suite with race detection
- ✅ Compare coverage (baseline vs final)
- ✅ Compare test counts
- ✅ Verify all fixtures exist
- ✅ Run TypeScript tests
- ✅ Build binary
- ✅ Check documentation

## Scripts

| Script | Purpose | When to Run |
|--------|---------|-------------|
| `verify-phase.sh` | Quick verification | After each phase |
| `final-verify.sh` | Comprehensive check | After all phases complete |

## Checkpoints

Scripts save checkpoints in the root directory:

```
coverage.phase1.out
coverage.phase2.out
...
test_count.phase1.txt
test_count.phase2.txt
...
```

These can be used to track progress and compare metrics.

## Exit Codes

- `0` - All checks passed
- `1` - One or more checks failed

## Logs

Scripts create log files:

- `test-output.log` - Go test output
- `lint-output.log` - Linter output
- `js-test-output.log` - JavaScript test output
- `test-final.log` - Final test run
- `integration.log` - Integration test output

## CI/CD Integration

These scripts can be integrated into CI:

```yaml
- name: Verify Phase
  run: ./scripts/verify-phase.sh ${{ matrix.phase }}
```

## Troubleshooting

### Permission Denied

```bash
chmod +x scripts/*.sh
```

### Windows

Use Git Bash or WSL to run these scripts on Windows.
