# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

cclint is a linter for Claude Code configurations. It analyzes CLAUDE.md files, `.claude/` directories, and related configuration files to identify issues like broken references, circular dependencies, unclear instructions, and overly broad permissions.

## Build and Development Commands

```bash
just build          # Build binary to bin/cclint
just install        # Install to $GOPATH/bin
just test           # Run all tests
just test-coverage  # Run tests with coverage report
just lint           # Run golangci-lint
just fmt            # Format code
just vet            # Run go vet
just tidy           # Tidy go.mod dependencies
just run .          # Build and run linter on current directory
just run-deep       # Run with --deep flag (uses Claude API)
just run-report     # Generate a detailed report
just setup          # Setup dev environment (mise install + go mod download)
just release        # Build release binaries for all platforms
```

## CLI Commands

- `cclint lint [path]` - Lint configurations (flags: `--deep`, `--offline`, `--format`, `--agent`)
- `cclint report [path]` - Generate detailed report with tree, metrics, and references
- `cclint fix [path]` - Auto-fix issues (flags: `--ai`, `--dry-run`)

## Architecture

The linter follows a pipeline architecture:

1. **Agent Configs** (`internal/agent/configs/`) - Embedded YAML agent profiles (e.g., `claude-code.yaml`) specify entrypoints, reference patterns, and priority markers
2. **Parser** (`internal/parser/`) - Parses config files (Markdown, JSON, YAML) into `ParsedFile` with sections and `FileCategory` (config, instructions, commands, documentation)
3. **Analyzer** (`internal/analyzer/`) - Builds a `Tree` of `ConfigNode`s by following file references up to 5 levels deep
4. **Rules** (`internal/rules/`) - Each rule implements `Rule` interface with `Run(*AnalysisContext) ([]Issue, error)`
5. **Classifier** (`internal/classifier/`) - Quality assessment via heuristics or LLM (with `--deep` flag)
6. **Reporter** (`internal/reporter/`) - Output formatting (terminal or JSON)
7. **Fixer** (`internal/fixer/`) - Applies auto-fixes for issues that have `Fix` defined

## Adding New Rules

1. Create a new file in `internal/rules/` implementing the `Rule` interface
2. Register it in `DefaultRegistry()` in `internal/rules/registry.go`

## Key Types

- `rules.Issue` - Linting issue with severity, location, message, and optional auto-fix
- `rules.AnalysisContext` - Context passed to rules with Tree, AgentConfig, and helper methods
- `analyzer.Tree` - Graph of config nodes with references between them
- `analyzer.Reference` - A reference (file, URL, tool, subagent, MCP server, skill) with source location and priority
- `parser.FileCategory` - Semantic file type: config, instructions, commands, documentation
- `agent.Config` - Agent profile with entrypoints, reference patterns, and priority markers
