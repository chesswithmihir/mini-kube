# Makefile for my-runc project

# Default target
all: test

# Run all tests
test:
	go test -v

# Build the project
build:
	go build -o my-runc .

# Clean build artifacts
clean:
	rm -f my-runc

# Run tests with coverage
test-cover:
	go test -coverprofile=coverage.txt -covermode=atomic
	go tool cover -html=coverage.txt

# Run vet check
vet:
	go vet

# Run all checks
check: vet test

# Help target
help:
	@echo "Available targets:"
	@echo "  all         - Run all checks (default)"
	@echo "  test        - Run tests"
	@echo "  build       - Build the project"
	@echo "  clean       - Clean build artifacts"
	@echo "  test-cover  - Run tests with coverage report"
	@echo "  vet         - Run vet check"
	@echo "  check       - Run all checks"
	@echo "  help        - Show this help"

.PHONY: all test build clean test-cover vet check help