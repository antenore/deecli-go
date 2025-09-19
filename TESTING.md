# Testing Guide

This document describes the comprehensive testing infrastructure for DeeCLI.

## Overview

DeeCLI uses a multi-layered testing approach with:
- **Unit Tests** - Fast, isolated tests for individual components
- **Integration Tests** - End-to-end tests with real systems
- **Benchmark Tests** - Performance testing for critical paths
- **Mock Infrastructure** - Comprehensive mocking for external dependencies

## Test Structure

```
test/
├── unit/           # Unit tests using testify framework
├── integration/    # Integration tests with build tags
├── mocks/          # Centralized mock implementations
├── testdata/       # Test fixtures and sample data
└── utils/          # Test utilities and helpers
```

## Running Tests

### Quick Commands

```bash
# Run all tests
make test

# Run unit tests only
make test-unit

# Run integration tests only
make test-integration

# Run tests with coverage report
make test-coverage

# Run benchmark tests
make test-bench

# Run comprehensive test suite
make test-all
```

### Detailed Test Commands

```bash
# Unit tests with coverage
make test-unit-coverage

# Show coverage percentages
make test-coverage-func

# Run with race detection
make test-race

# Verbose test output
make test-verbose

# Clean test artifacts
make test-clean
```

## Test Categories

### Unit Tests (`test/unit/`)

Fast, isolated tests that use mocks for external dependencies:

- **API Client Tests** - HTTP client and streaming functionality
- **Chat Commands Tests** - Command parsing and execution
- **Input Manager Tests** - History navigation and tab completion
- **File Tracker Tests** - File extraction and suggestion tracking
- **Benchmark Tests** - Performance testing for critical operations

### Integration Tests (`test/integration/`)

End-to-end tests that interact with real systems:

- **API Integration** - Real DeepSeek API calls (requires API key)
- **File Operations** - Actual file system operations
- **Configuration** - Config file loading and validation

**Note**: Integration tests require the `integration` build tag and will skip gracefully if external dependencies are unavailable.

### Mock Infrastructure (`test/mocks/`)

Comprehensive mocks for all external dependencies:

- **API Client Mock** - Simulates DeepSeek API responses
- **File System Mock** - In-memory file system for testing
- **Config Manager Mock** - Configuration management simulation
- **Session Manager Mock** - Session storage simulation

## Test Utilities (`test/utils/`)

Helper functions and builders for common test scenarios:

- **TestEnvironment** - Complete test setup with cleanup
- **FileBuilder** - Easy test file creation
- **ConfigBuilder** - Test configuration building
- **TimeHelper** - Time-based testing utilities
- **AssertionHelper** - Enhanced assertion utilities

## Coverage Targets

### Current Coverage Goals

- **Unit Tests**: 85% minimum coverage
- **Integration Tests**: Critical path coverage
- **Combined**: 80% overall coverage

### Coverage Reports

Coverage reports are generated in multiple formats:

```bash
# HTML report (opens in browser)
make test-coverage
open coverage/coverage.html

# Function-level coverage
make test-coverage-func

# Unit test specific coverage
make test-unit-coverage
open coverage/unit-coverage.html
```

## Writing Tests

### Unit Test Example

```go
func (suite *MyTestSuite) TestSomeFunction() {
    // Setup
    env := utils.NewTestEnvironment(suite.T())
    defer env.Cleanup()
    
    // Test
    result := functionUnderTest(input)
    
    // Assert
    assert.Equal(suite.T(), expected, result)
}
```

### Integration Test Example

```go
//go:build integration

func (suite *IntegrationSuite) TestRealAPICall() {
    if !suite.hasAPIKey {
        suite.T().Skip("API key required")
    }
    
    response, err := suite.client.SendRequest(ctx, message)
    require.NoError(suite.T(), err)
    assert.NotEmpty(suite.T(), response.Content)
}
```

### Benchmark Test Example

```go
func BenchmarkCriticalOperation(b *testing.B) {
    b.ResetTimer()
    b.ReportAllocs()
    
    for i := 0; i < b.N; i++ {
        result := criticalOperation(input)
        _ = result // Prevent optimization
    }
}
```

## Best Practices

### Test Organization

1. **Use testify suites** for complex test setups
2. **Group related tests** in the same file
3. **Use descriptive test names** that explain the scenario
4. **Include both positive and negative test cases**

### Mock Usage

1. **Use centralized mocks** from `test/mocks/`
2. **Set up common scenarios** with helper functions
3. **Verify mock interactions** when relevant
4. **Reset mocks** between tests

### Performance Testing

1. **Benchmark critical paths** regularly
2. **Use `b.ReportAllocs()`** to track memory usage
3. **Test with realistic data sizes**
4. **Monitor performance regressions**

### Integration Testing

1. **Make tests resilient** to external failures
2. **Use environment variables** for configuration
3. **Skip gracefully** when dependencies unavailable
4. **Test error conditions** as well as success paths

## Continuous Integration

Tests run automatically on:
- **Push to main branches**
- **Pull requests**
- **Scheduled nightly runs**

### CI Configuration

The GitHub Actions workflow (`.github/workflows/test.yml`) runs:

1. Unit tests
2. Integration tests (with optional API key)
3. Coverage generation
4. Benchmark tests
5. Race condition detection
6. Coverage upload to Codecov

### Environment Variables

- `DEEPSEEK_API_KEY` - For integration tests (optional)
- `CI` - Automatically set in CI environments

## Troubleshooting

### Common Issues

**Tests fail with "API key required"**
- Set `DEEPSEEK_API_KEY` environment variable
- Or run `make test-unit` to skip integration tests

**Coverage reports not generated**
- Ensure `coverage/` directory exists
- Run `make test-clean` and retry

**Benchmark tests too slow**
- Use `-short` flag: `go test -short -bench=.`
- Or set `BENCHMARK_TIME=1s` environment variable

**Race condition failures**
- Fix data races in concurrent code
- Use proper synchronization primitives
- Test with `make test-race` locally

### Getting Help

1. Check this documentation first
2. Look at existing test examples
3. Review test utilities in `test/utils/`
4. Check CI logs for detailed error information

## Maintenance

### Regular Tasks

- **Review coverage reports** monthly
- **Update benchmark baselines** when performance improves
- **Clean up obsolete tests** during refactoring
- **Update mocks** when APIs change

### Adding New Tests

1. **Determine test type** (unit/integration/benchmark)
2. **Use appropriate directory** and naming conventions
3. **Include in relevant test suites**
4. **Update documentation** if needed

---

*For more information about the project's testing philosophy, see DEVELOPMENT.md*