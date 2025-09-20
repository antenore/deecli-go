# Testing Guide

DeeCLI follows Go's standard testing practices. Tests are located alongside source code.

## Running Tests

```bash
# Run all tests
make test

# Run with coverage report
make test-coverage

# Run unit tests only
make test-unit

# Run with race detection
make test-race

# Run benchmark tests
make test-bench

# Clean test artifacts
make test-clean
```

## Test Structure

Tests follow Go conventions:
```
internal/
├── api/
│   ├── client.go
│   └── client_test.go           # Tests for client.go
├── chat/
│   ├── model.go
│   ├── model_test.go            # Tests for model.go
│   └── commands/
│       ├── commands.go
│       └── commands_test.go     # Tests for commands.go
└── ...
```

## Current Tests

- **API Client** (`internal/api/client_streaming_test.go`) - HTTP streaming tests
- **Chat Model** (`internal/chat/model_test.go`) - Core chat functionality
- **Chat Commands** (`internal/chat/commands/*_test.go`) - Command handling
- **Input Manager** (`internal/chat/input/manager_test.go`) - User input handling
- **File Tracker** (`internal/chat/tracker/filetracker_test.go`) - File tracking
- **UI Components** (`internal/chat/ui/spinner_test.go`) - UI elements
- **Configuration** (`internal/config/config_test.go`) - Config validation
- **File Operations** (`internal/files/*_test.go`) - File handling

## Writing Tests

Follow standard Go testing patterns:

```go
package mypackage

import "testing"

func TestSomeFunction(t *testing.T) {
    input := "test input"
    expected := "expected output"

    result := functionUnderTest(input)

    if result != expected {
        t.Errorf("got %v, want %v", result, expected)
    }
}
```

### Table-Driven Tests

For multiple test cases:

```go
func TestMultipleCases(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected string
    }{
        {"case 1", "input1", "output1"},
        {"case 2", "input2", "output2"},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := functionUnderTest(tt.input)
            if result != tt.expected {
                t.Errorf("got %v, want %v", result, tt.expected)
            }
        })
    }
}
```

## Coverage

View coverage reports:
```bash
make test-coverage
open coverage/coverage.html
```

Current coverage varies by module. Add tests incrementally when:
- Implementing new features
- Fixing bugs
- Refactoring code

## Best Practices

1. **Test alongside code** - Keep `*_test.go` files with source code
2. **Test both success and error paths** - Include negative test cases
3. **Use descriptive test names** - Explain what's being tested
4. **Keep tests simple** - One assertion per test when possible
5. **Test in real environments** - Verify functionality in actual terminals

## Continuous Integration

Tests run automatically on:
- Push to main branches
- Pull requests

The CI pipeline runs all test commands and uploads coverage reports.

---

*Simple is better. Follow Go conventions. Add tests incrementally.*