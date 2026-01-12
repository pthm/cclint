package rules

import (
	"fmt"
	"os/exec"

	"github.com/pthm/cclint/internal/analyzer"
)

// ClaudeBuiltinTools is the set of known Claude Code built-in tools
var ClaudeBuiltinTools = map[string]bool{
	"Read":            true,
	"Write":           true,
	"Edit":            true,
	"Bash":            true,
	"Glob":            true,
	"Grep":            true,
	"Task":            true,
	"WebFetch":        true,
	"WebSearch":       true,
	"TodoWrite":       true,
	"NotebookEdit":    true,
	"AskUserQuestion": true,
	"Skill":           true,
	"EnterPlanMode":   true,
	"ExitPlanMode":    true,
	"KillShell":       true,
	"TaskOutput":      true,
}

// MissingToolRule checks that tools declared in frontmatter exist
type MissingToolRule struct{}

func (r *MissingToolRule) Name() string {
	return "missing-tool"
}

func (r *MissingToolRule) Description() string {
	return "Checks that tools declared in frontmatter exist (Claude built-in or OS command)"
}

func (r *MissingToolRule) Config() RuleConfig {
	return RuleConfig{} // Applies to all file types
}

func (r *MissingToolRule) Run(ctx *AnalysisContext) ([]Issue, error) {
	var issues []Issue

	scopes, err := ctx.Scopes()
	if err != nil {
		return nil, err
	}

	for _, scope := range scopes {
		if scope.Type != analyzer.ScopeTypeSubagent {
			continue
		}

		for _, tool := range scope.DeclaredTools {
			if !r.toolExists(tool) {
				issues = append(issues, Issue{
					Rule:     r.Name(),
					Severity: Warning,
					Message:  fmt.Sprintf("Tool '%s' not found (not a Claude built-in tool and not found on PATH)", tool),
					File:     scope.Entrypoint,
					Line:     1, // Frontmatter is at the top
					Context:  fmt.Sprintf("Declared in frontmatter of subagent '%s'", scope.Name),
				})
			}
		}
	}

	return issues, nil
}

// toolExists checks if a tool is either a Claude built-in or exists on the OS
func (r *MissingToolRule) toolExists(tool string) bool {
	// Check if it's a Claude built-in tool
	if ClaudeBuiltinTools[tool] {
		return true
	}

	// Check if it exists on the OS PATH
	_, err := exec.LookPath(tool)
	return err == nil
}
