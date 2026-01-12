# cclint - Claude Code Linter

# Build variables
version := `git describe --tags --always --dirty 2>/dev/null || echo "dev"`
build_time := `date -u +"%Y-%m-%dT%H:%M:%SZ"`
ldflags := "-ldflags \"-X main.version=" + version + " -X main.buildTime=" + build_time + "\""

# Default recipe - show help
default:
    @just --list

# Build the binary
build:
    go build {{ldflags}} -o bin/cclint ./cmd/cclint

# Install to $GOPATH/bin (or $HOME/go/bin if GOPATH unset)
install:
    go build {{ldflags}} -o $(go env GOPATH)/bin/cclint ./cmd/cclint

# Clean build artifacts
clean:
    rm -rf bin/
    go clean

# Run tests
test:
    go test -v ./...

# Run tests with coverage
test-coverage:
    go test -v -coverprofile=coverage.out ./...
    go tool cover -html=coverage.out -o coverage.html

# Run linter (golangci-lint)
lint:
    golangci-lint run ./...

# Format code
fmt:
    go fmt ./...

# Run go vet
vet:
    go vet ./...

# Tidy dependencies
tidy:
    go mod tidy

# Download dependencies
deps:
    go mod download

# Run cclint on current directory
run *ARGS: build
    ./bin/cclint lint {{ARGS}} .

# Run with deep analysis
run-deep: build
    ./bin/cclint lint --deep .

# Generate a report
run-report: build
    ./bin/cclint report .

# Build release binaries using GoReleaser (snapshot for local testing)
release-snapshot:
    goreleaser release --snapshot --clean

# Create a release (requires git tag)
release:
    goreleaser release --clean

# Setup development environment
setup:
    mise install
    go mod download
