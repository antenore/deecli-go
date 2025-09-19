# DeeCLI Makefile

BINARY_NAME=deecli
MAIN_PACKAGE=./main.go

# Build for current platform
build:
	go build -o $(BINARY_NAME) $(MAIN_PACKAGE)

# Install dependencies
deps:
	go mod tidy
	go mod download

# Run in development mode
dev:
	go run $(MAIN_PACKAGE) chat

# Test configuration
COVERAGE_DIR=coverage
COVERAGE_PROFILE=$(COVERAGE_DIR)/coverage.out
COVERAGE_HTML=$(COVERAGE_DIR)/coverage.html

# Run all tests
test:
	go test ./...

# Run unit tests only
test-unit:
	go test ./test/unit/... ./internal/...

# Run integration tests only (requires build tag)
test-integration:
	go test -tags=integration ./test/integration/...

# Run tests with coverage
test-coverage:
	@mkdir -p $(COVERAGE_DIR)
	go test -coverprofile=$(COVERAGE_PROFILE) ./...
	go tool cover -html=$(COVERAGE_PROFILE) -o $(COVERAGE_HTML)
	@echo "Coverage report generated: $(COVERAGE_HTML)"

# Run unit tests with coverage
test-unit-coverage:
	@mkdir -p $(COVERAGE_DIR)
	go test -coverprofile=$(COVERAGE_DIR)/unit-coverage.out ./test/unit/... ./internal/...
	go tool cover -html=$(COVERAGE_DIR)/unit-coverage.out -o $(COVERAGE_DIR)/unit-coverage.html
	@echo "Unit test coverage report: $(COVERAGE_DIR)/unit-coverage.html"

# Run benchmark tests
test-bench:
	go test -bench=. -benchmem ./...

# Run tests with race detection
test-race:
	go test -race ./...

# Run tests with verbose output
test-verbose:
	go test -v ./...

# Run tests and show coverage percentage
test-coverage-func:
	@mkdir -p $(COVERAGE_DIR)
	go test -coverprofile=$(COVERAGE_PROFILE) ./...
	go tool cover -func=$(COVERAGE_PROFILE)

# Run all test variants (comprehensive testing)
test-all: test-unit test-integration test-coverage test-race test-bench

# Clean test artifacts
test-clean:
	rm -rf $(COVERAGE_DIR)
	go clean -testcache

# Build for all platforms
build-all: build-linux build-darwin build-windows

build-linux:
	GOOS=linux GOARCH=amd64 go build -o $(BINARY_NAME)-linux $(MAIN_PACKAGE)

build-darwin:
	GOOS=darwin GOARCH=amd64 go build -o $(BINARY_NAME)-macos $(MAIN_PACKAGE)

build-windows:
	GOOS=windows GOARCH=amd64 go build -o $(BINARY_NAME)-windows.exe $(MAIN_PACKAGE)

# Clean build artifacts
clean:
	rm -f $(BINARY_NAME)*

# Clean everything (build artifacts and test artifacts)
clean-all: clean test-clean

# Format code
fmt:
	go fmt ./...

# Lint code
lint:
	golangci-lint run

# Show help
help:
	@echo "Available commands:"
	@echo ""
	@echo "Build Commands:"
	@echo "  build          - Build binary for current platform"
	@echo "  build-all      - Build for all platforms (Linux, macOS, Windows)"
	@echo "  build-linux    - Build for Linux"
	@echo "  build-darwin   - Build for macOS"
	@echo "  build-windows  - Build for Windows"
	@echo ""
	@echo "Development Commands:"
	@echo "  deps           - Install and tidy dependencies"
	@echo "  dev            - Run in development mode"
	@echo "  fmt            - Format code"
	@echo "  lint           - Lint code with golangci-lint"
	@echo ""
	@echo "Test Commands:"
	@echo "  test           - Run all tests"
	@echo "  test-unit      - Run unit tests only"
	@echo "  test-integration - Run integration tests only"
	@echo "  test-coverage  - Run tests with HTML coverage report"
	@echo "  test-unit-coverage - Run unit tests with coverage"
	@echo "  test-coverage-func - Show coverage percentages by function"
	@echo "  test-bench     - Run benchmark tests"
	@echo "  test-race      - Run tests with race detection"
	@echo "  test-verbose   - Run tests with verbose output"
	@echo "  test-all       - Run comprehensive test suite"
	@echo ""
	@echo "Cleanup Commands:"
	@echo "  clean          - Remove build artifacts"
	@echo "  test-clean     - Remove test artifacts and cache"
	@echo "  clean-all      - Remove all artifacts"

.PHONY: build deps dev test test-unit test-integration test-coverage test-unit-coverage test-coverage-func test-bench test-race test-verbose test-all test-clean build-all build-linux build-darwin build-windows clean clean-all fmt lint help