.PHONY: build test lint clean install help

# Build the binaries
build:
	go build -o vga-events ./cmd/vga-events
	go build -o vga-events-telegram ./cmd/vga-events-telegram

# Run tests
test:
	go test -v -race -coverprofile=coverage.out ./...

# Run tests with coverage report
test-coverage: test
	go tool cover -html=coverage.out

# Run linter
lint:
	golangci-lint run

# Clean build artifacts
clean:
	rm -f vga-events vga-events-telegram coverage.out
	rm -rf bin/

# Install the binary to $GOPATH/bin
install:
	go install ./cmd/vga-events

# Run all checks (test + lint)
check: test lint

# Display help
help:
	@echo "Available targets:"
	@echo "  build         - Build the binary"
	@echo "  test          - Run tests"
	@echo "  test-coverage - Run tests and show coverage report"
	@echo "  lint          - Run golangci-lint"
	@echo "  clean         - Remove build artifacts"
	@echo "  install       - Install binary to GOPATH/bin"
	@echo "  check         - Run tests and linting"
	@echo "  help          - Show this help message"

# Default target
.DEFAULT_GOAL := help
