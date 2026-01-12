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

# Build and sign the binary (macOS only)
build-signed: build
    ./scripts/sign-macos.sh bin/cclint

# Build, sign, and notarize the binary (macOS only, requires 1Password)
build-notarized: build
    ./scripts/sign-macos.sh bin/cclint --notarize

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
    GITHUB_TOKEN=$(gh auth token) goreleaser release --snapshot --clean

# Create and publish a release (e.g., just release 0.1.0)
release VERSION:
    #!/usr/bin/env bash
    set -euo pipefail

    VERSION="{{VERSION}}"

    # Strip 'v' prefix if provided
    VERSION="${VERSION#v}"
    TAG="v${VERSION}"

    # Validate version format (semver)
    if ! [[ "$VERSION" =~ ^[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9.]+)?$ ]]; then
        echo "Error: Invalid version format. Use semver (e.g., 0.1.0, 1.0.0-beta.1)"
        exit 1
    fi

    # Check for uncommitted changes
    if ! git diff --quiet || ! git diff --cached --quiet; then
        echo "Error: You have uncommitted changes. Commit or stash them first."
        exit 1
    fi

    # Check if tag already exists
    if git rev-parse "$TAG" >/dev/null 2>&1; then
        echo "Error: Tag $TAG already exists"
        exit 1
    fi

    # Verify gh auth
    if ! gh auth token >/dev/null 2>&1; then
        echo "Error: Not authenticated with gh. Run 'gh auth login' first."
        exit 1
    fi

    # Show what will be released
    echo "Releasing $TAG"
    echo ""
    echo "Recent commits:"
    git log --oneline -5
    echo ""

    # Confirm
    read -p "Continue with release? [y/N] " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo "Aborted."
        exit 1
    fi

    # Create and push tag
    git tag -a "$TAG" -m "Release $TAG"
    git push origin "$TAG"

    # Run GoReleaser
    GITHUB_TOKEN=$(gh auth token) HOMEBREW_TAP_GITHUB_TOKEN=$(gh auth token) goreleaser release --clean

    echo ""
    echo "Release $TAG published!"

# Delete a release tag (local and remote)
release-delete VERSION:
    #!/usr/bin/env bash
    set -euo pipefail

    VERSION="{{VERSION}}"
    VERSION="${VERSION#v}"
    TAG="v${VERSION}"

    read -p "Delete tag $TAG locally and remotely? [y/N] " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo "Aborted."
        exit 1
    fi

    git tag -d "$TAG" 2>/dev/null || true
    git push origin --delete "$TAG" 2>/dev/null || true
    echo "Tag $TAG deleted."

# Build release locally with GoReleaser (without publishing)
release-local:
    GITHUB_TOKEN=$(gh auth token) HOMEBREW_TAP_GITHUB_TOKEN=$(gh auth token) goreleaser release --clean --skip=publish

# Setup development environment
setup:
    mise install
    go mod download
