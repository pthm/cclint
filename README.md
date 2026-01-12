# cclint

A comprehensive linter and analyzer for [Claude Code](https://claude.com/code) configurations. Keep your agent instructions clean, consistent, and effective by catching issues before they impact your workflow.

## Why cclint?

As your Claude Code configurations grow—spanning `CLAUDE.md` files, `.claude/` directories, nested commands, skills, and subagents—it becomes increasingly difficult to:
- Track which files are actually being loaded into context
- Spot broken references, circular dependencies, or duplicate instructions
- Ensure instructions are clear, actionable, and conflict-free
- Estimate token usage and optimize context size
- Understand the full scope of what each agent sees

**cclint solves this** by analyzing your entire configuration graph, providing automated checks, interactive visualization, and intelligent insights.

## Features

### 1. Catch Configuration Issues Early

Automated linting rules detect common problems across structural, content quality, and AI-powered categories:

**Structural Issues**
- Broken file references and URLs
- Circular dependencies
- Missing entrypoints for commands and skills
- Overly broad file permissions
- Missing tool or skill declarations

**Content Quality**
- Vague or unclear instructions
- Contradictory guidance
- Missing critical context
- Excessive verbosity

**AI-Powered Analysis** (via `--deep` flag)
- Duplicate instructions across scopes
- Subtle contradictions in guidance
- Clarity and actionability scoring

<img width="1664" height="720" alt="CleanShot 2026-01-12 at 11 56 23@2x" src="https://github.com/user-attachments/assets/75566cdb-7ebb-41a8-acc2-a6cde0acecde" />

### 2. Visualize Configuration Hierarchies

The interactive graph browser provides a complete view of your configuration structure:
- Navigate through main agent, subagents, commands, and skills
- See which files are loaded in each scope
- Track references between files
- Understand inheritance and nesting
- Preview file contents inline

<img width="3302" height="1874" alt="CleanShot 2026-01-12 at 11 54 02@2x" src="https://github.com/user-attachments/assets/b8c08482-8915-43f0-8eac-5ccad650a495" />

### 3. Understand Context Usage

Generate detailed reports showing:
- Reference maps across all configuration files
- Token usage estimates per scope
- File category breakdown
- Reference types (files, URLs, tools, MCP servers, skills)

### 4. Auto-Fix Common Issues

Automatically resolve fixable problems with:
- Standard auto-fixes for structural issues
- AI-assisted fixes (via `--ai` flag) for content improvements
- Dry-run mode to preview changes


## Installation

### Homebrew

```bash
brew install pthm/tap/cclint
```

### Go Install

```bash
go install github.com/pthm/cclint/cmd/cclint@latest
```

## Quick Start

```bash
# Lint your configuration
cclint lint .

# View the interactive configuration graph
cclint graph .

# Generate a detailed report
cclint report .
```

## Usage

### Lint Command

Check your configurations for issues:

```bash
# Basic linting (structural + content quality rules)
cclint lint

# Deep analysis with AI-powered rules (requires Claude API)
cclint lint --deep

# Lint specific directory
cclint lint /path/to/project

# Output as JSON
cclint lint --format json

# Specify agent type (default: claude-code)
cclint lint --agent claude-code
```

### Graph Command

Explore your configuration hierarchy interactively:

```bash
# Launch interactive graph browser
cclint graph

# Print tree to stdout (non-interactive)
cclint graph --print

# Specify directory
cclint graph /path/to/project
```

**Interactive Controls:**
- `↑/k`, `↓/j` - Navigate up/down
- `←/h`, `→/l` - Collapse/expand nodes
- `Enter`/`Space` - Toggle expand/collapse
- `r` - Toggle reference display
- `s` - Toggle scope grouping
- `q` - Quit

### Report Command

Generate comprehensive configuration reports:

```bash
# Generate report with metrics and references
cclint report

# Output as JSON for programmatic use
cclint report --format json > report.json
```

### Fix Command

Automatically fix issues:

```bash
# Auto-fix standard issues
cclint fix

# Preview fixes without applying (dry-run)
cclint fix --dry-run

# Enable AI-assisted fixes for content issues
cclint fix --ai

# Combine dry-run with AI fixes
cclint fix --ai --dry-run
```

### Version Command

```bash
cclint version
```

## How It Works

cclint understands the full context hierarchy of Claude Code configurations:

1. **Agent-Aware Parsing** - Loads agent profiles (like `claude-code.yaml`) that define entrypoints, reference patterns, and priority markers
2. **Scope Discovery** - Identifies distinct scopes: main agent, subagents, commands, and skills—each with their own context boundaries
3. **Reference Tracking** - Follows file references, URLs, tool declarations, MCP server connections, and skill invocations up to 5 levels deep
4. **Intelligent Analysis** - Runs both heuristic and LLM-based rules that understand the semantic meaning of your instructions
5. **Context-Aware Reporting** - Provides insights specific to each scope, helping you understand what each agent actually sees

## Common Use Cases

- **Pre-commit checks** - Catch issues before they're committed
- **Debugging context** - Understand why an agent isn't seeing certain instructions
- **Token optimization** - Identify and reduce bloated configurations
- **Team collaboration** - Ensure consistent, high-quality agent configurations
- **Configuration refactoring** - Safely reorganize complex setups with confidence

## Development

See [CLAUDE.md](CLAUDE.md) for:
- Build and development commands
- Architecture overview
- How to add custom linting rules
- Testing and contribution guidelines

## Roadmap

This is an active project. Planned features include:
- Additional linting rules
- Performance optimizations for large codebases
- CI/CD integration examples
- Configuration migration tools

## Contributing

Contributions are welcome! Please feel free to submit issues or pull requests.

## License

MIT License - see [LICENSE](LICENSE) for details.
