# Makefile for School Finder TUI

.PHONY: help test test-verbose test-coverage test-race build clean install lint fmt vet

# Default target
help:
	@echo "School Finder TUI - Available targets:"
	@echo ""
	@echo "  make test           - Run all tests"
	@echo "  make test-verbose   - Run tests with verbose output"
	@echo "  make test-coverage  - Run tests with coverage report"
	@echo "  make test-race      - Run tests with race detector"
	@echo "  make build          - Build the application"
	@echo "  make install        - Install the application"
	@echo "  make clean          - Clean build artifacts"
	@echo "  make lint           - Run golangci-lint"
	@echo "  make fmt            - Format code"
	@echo "  make vet            - Run go vet"
	@echo ""

# Run all tests
test:
	go test ./...

# Run tests with verbose output
test-verbose:
	go test -v ./...

# Run tests with coverage
test-coverage:
	go test -v -coverprofile=coverage.out -covermode=atomic ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run tests with race detector
test-race:
	go test -v -race ./...

# Run specific test
test-db:
	go test -v -run "TestNewDB|TestSearchSchools|TestGetSchoolByID"

test-tui:
	go test -v -run "TestInitialModel|TestSearchView|TestDetailView"

test-naep:
	go test -v -run "TestDetermineGrades|TestMatchDistrict|TestSortNAEPScores"

# Build the application
build:
	go build -v -o schoolfinder

# Build with optimizations
build-release:
	go build -v -ldflags="-s -w" -o schoolfinder

# Install the application
install:
	go install

# Clean build artifacts
clean:
	rm -f schoolfinder
	rm -f coverage.out coverage.html
	rm -rf testdata/data.duckdb*
	go clean

# Run linter
lint:
	golangci-lint run ./...

# Format code
fmt:
	go fmt ./...
	gofmt -s -w .

# Run go vet
vet:
	go vet ./...

# Run all checks (fmt, vet, lint, test)
check: fmt vet lint test

# Quick test (no race detector for speed)
quick-test:
	go test ./...

# Continuous testing (watch for changes)
watch:
	@echo "Watching for changes..."
	@while true; do \
		inotifywait -qre close_write .; \
		clear; \
		make quick-test; \
	done

# Download and verify dependencies
deps:
	go mod download
	go mod verify
	go mod tidy
