package rules

import (
	"github.com/pthm-cable/cclint/internal/analyzer"
	"github.com/pthm-cable/cclint/internal/parser"
)

// ScopeAnalyzeFunc is a function that analyzes a single context scope
type ScopeAnalyzeFunc func(scope *analyzer.ContextScope, files []*analyzer.ConfigNode) ([]Issue, error)

// RunPerScope discovers all context scopes and runs the analysis function for each.
// This allows LLM rules to operate on isolated scopes without needing to know about
// scope boundaries themselves.
func RunPerScope(ctx *AnalysisContext, categories []parser.FileCategory, analyze ScopeAnalyzeFunc) ([]Issue, error) {
	scopes, err := ctx.Tree.DiscoverScopes(ctx.AgentConfig, ctx.RootPath)
	if err != nil {
		return nil, err
	}

	var allIssues []Issue

	for _, scope := range scopes {
		// Filter nodes by category if specified
		var files []*analyzer.ConfigNode
		if len(categories) == 0 {
			files = scope.Nodes
		} else {
			categorySet := make(map[parser.FileCategory]bool)
			for _, cat := range categories {
				categorySet[cat] = true
			}
			for _, node := range scope.Nodes {
				if node.Parsed != nil && categorySet[node.Parsed.Category] {
					files = append(files, node)
				}
			}
		}

		// Skip empty scopes
		if len(files) == 0 {
			continue
		}

		// Run analysis for this scope
		issues, err := analyze(scope, files)
		if err != nil {
			// Log error but continue with other scopes
			continue
		}

		allIssues = append(allIssues, issues...)
	}

	return allIssues, nil
}

// FilteredFilePaths returns file paths from nodes, optionally filtering by minimum content size
func FilteredFilePaths(nodes []*analyzer.ConfigNode, minContentSize int) []string {
	var paths []string
	for _, node := range nodes {
		if len(node.Content) >= minContentSize {
			paths = append(paths, node.Path)
		}
	}
	return paths
}

// BuildFileList formats a list of file paths for inclusion in an LLM prompt
func BuildFileList(paths []string) string {
	if len(paths) == 0 {
		return "(no files)"
	}
	result := ""
	for _, path := range paths {
		result += "- " + path + "\n"
	}
	return result
}

// ScopeContextDescription returns a human-readable description of the scope
func ScopeContextDescription(scope *analyzer.ContextScope) string {
	if scope.Type == analyzer.ScopeTypeMain {
		return "main agent"
	}
	return "subagent: " + scope.Name
}
