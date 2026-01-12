package rules

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// MissingEntrypointRule checks for missing primary configuration files
type MissingEntrypointRule struct{}

func (r *MissingEntrypointRule) Name() string {
	return "missing-entrypoint"
}

func (r *MissingEntrypointRule) Description() string {
	return "Checks if primary configuration files (like CLAUDE.md) are missing"
}

func (r *MissingEntrypointRule) Config() RuleConfig {
	return RuleConfig{} // Applies to all file types
}

// alternativeEntrypoints defines groups of entrypoints that are mutually exclusive.
// If any one in a group exists, the others in that group are not required.
// If multiple in a group exist, a warning is issued.
var alternativeEntrypoints = [][]string{
	{"CLAUDE.md", ".claude/CLAUDE.md"},
}

func (r *MissingEntrypointRule) Run(ctx *AnalysisContext) ([]Issue, error) {
	var issues []Issue

	// Track which entrypoints are in alternative groups
	inAlternativeGroup := make(map[string]bool)
	for _, group := range alternativeEntrypoints {
		for _, ep := range group {
			inAlternativeGroup[ep] = true
		}
	}

	// Check alternative entrypoint groups
	for _, group := range alternativeEntrypoints {
		var existing []string
		for _, entrypoint := range group {
			fullPath := filepath.Join(ctx.RootPath, entrypoint)
			if _, err := os.Stat(fullPath); err == nil {
				existing = append(existing, entrypoint)
			}
		}

		if len(existing) == 0 {
			// None exist - recommend creating one
			issues = append(issues, Issue{
				Rule:     r.Name(),
				Severity: Info,
				Message:  fmt.Sprintf("Recommended configuration file not found: %s (or %s)", group[0], group[1]),
				File:     filepath.Join(ctx.RootPath, group[0]),
				Line:     0,
			})
		} else if len(existing) > 1 {
			// Multiple exist - warn about ambiguity
			issues = append(issues, Issue{
				Rule:     r.Name(),
				Severity: Warning,
				Message:  fmt.Sprintf("Both %s exist - Claude Code treats these as alternatives and may only read one", strings.Join(existing, " and ")),
				File:     filepath.Join(ctx.RootPath, existing[0]),
				Line:     0,
			})
		}
	}

	// Check non-alternative required entrypoints
	for _, entrypoint := range ctx.AgentConfig.Entrypoints {
		// Skip if this entrypoint is part of an alternative group
		if inAlternativeGroup[entrypoint] {
			continue
		}

		fullPath := filepath.Join(ctx.RootPath, entrypoint)
		isRequired := isRequiredEntrypoint(entrypoint)

		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			if isRequired {
				issues = append(issues, Issue{
					Rule:     r.Name(),
					Severity: Info,
					Message:  fmt.Sprintf("Recommended configuration file not found: %s", entrypoint),
					File:     fullPath,
					Line:     0,
				})
			}
		}
	}

	// Also check if there are NO config files at all
	if len(ctx.Tree.Nodes) == 0 {
		issues = append(issues, Issue{
			Rule:     r.Name(),
			Severity: Warning,
			Message:  "No configuration files found in this project",
			File:     ctx.RootPath,
			Line:     0,
		})
	}

	return issues, nil
}

// isRequiredEntrypoint returns true for commonly expected files
// that are NOT part of alternative groups
func isRequiredEntrypoint(path string) bool {
	// Currently no standalone required entrypoints
	// CLAUDE.md is handled via alternativeEntrypoints
	return false
}
