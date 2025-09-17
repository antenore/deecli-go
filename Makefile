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

# Run tests
test:
	go test ./...

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

# Format code
fmt:
	go fmt ./...

# Lint code
lint:
	golangci-lint run

# Show help
help:
	@echo "Available commands:"
	@echo "  build      - Build binary for current platform"
	@echo "  build-all  - Build for all platforms"
	@echo "  deps       - Install dependencies"
	@echo "  dev        - Run in development mode"
	@echo "  test       - Run tests"
	@echo "  fmt        - Format code"
	@echo "  lint       - Lint code"
	@echo "  clean      - Remove build artifacts"

.PHONY: build deps dev test build-all build-linux build-darwin build-windows clean fmt lint help