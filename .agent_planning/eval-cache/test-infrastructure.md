# Test Infrastructure Cache
**Confidence**: FRESH
**Date**: 2025-12-18

## Test Execution

### Primary Commands
```bash
just test              # Run all tests
just test-coverage     # Run with race detector + coverage report
just build             # Verify compilation
```

### Direct Go Commands
```bash
go test ./...                              # All tests
go test -race ./...                        # With race detector
go test -cover ./internal/...              # With coverage
go test -coverprofile=coverage.out ./...   # Generate coverage profile
go tool cover -func=coverage.out           # View coverage details
```

## Test Organization

### Unit Tests
- Located in same package as code (`*_test.go`)
- Test specific functions in isolation
- Use table-driven tests for parsing functions

### Integration Tests
- `internal/git/integration_test.go`
- Create real git repos in temp directories
- Test actual git command execution

### Test Utilities

#### Git Test Helpers
```go
setupTestRepo(t) string          // Creates temp git repo with initial commit
runGit(t, dir, args...)          // Execute git command
writeFile(t, path, content)      // Write test file
pathsEqual(t, a, b) bool         // Compare paths resolving symlinks
```

#### Concurrency Helpers
- sync.WaitGroup for parallel tests
- Mutex for shared state
- Goroutines for concurrent lock testing

## Coverage Analysis

### Current Coverage (Phase 1)
- internal/git: 87.8%
- internal/metadata: 76.8%
- internal/lock: 66.7%
- internal/config: 82.0%
- internal/core: 22.2% (errors.go fully tested)

### What's Tested
- All happy paths
- Common error conditions (invalid input, not found, dirty state)
- Concurrent access patterns
- Atomic operation behavior

### What's Not Tested
- File system error injection (mkdir fails, rename fails)
- Some defensive error handling paths
- Debug utility methods (String())

## Race Detection

All tests pass with `-race` flag:
- Lock concurrent acquisition test specifically exercises race conditions
- No data races detected in any package

## Test Patterns

### Table-Driven Tests
Used for parsing functions:
- `TestParseWorktreeList` - 5 scenarios
- `TestParseStatusV2` - 6 scenarios
- Config validation tests

### Error Verification
```go
coreErr, ok := err.(*core.Error)
if !ok || coreErr.Code != core.ErrExpected {
    t.Fatalf("Expected specific error, got %T %v", err, err)
}
```

### Symlink Handling (macOS)
```go
// Resolve symlinks for path comparison (handles /var vs /private/var)
resolved, _ := filepath.EvalSymlinks(path)
```

## Test Data

### Fixtures
- Located in `testdata/` directory (currently minimal)
- Most tests use dynamically created data

### Temp Directories
- Use `t.TempDir()` for automatic cleanup
- Create isolated test repos per test
