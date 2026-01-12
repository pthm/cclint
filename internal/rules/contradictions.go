package rules

import (
	"strings"

	"github.com/pthm-cable/cclint/internal/parser"
)

// ContradictionsRule checks for potential contradictions in instructions
type ContradictionsRule struct{}

func (r *ContradictionsRule) Name() string {
	return "contradictions"
}

func (r *ContradictionsRule) Description() string {
	return "Checks for potential contradictions or conflicting instructions"
}

func (r *ContradictionsRule) Config() RuleConfig {
	return RuleConfig{
		FileCategories: []parser.FileCategory{
			parser.FileCategoryInstructions,
			parser.FileCategoryDocumentation,
		},
	}
}

// contradictionCheck defines a pair of patterns that might indicate contradiction
type contradictionCheck struct {
	pattern1 string
	pattern2 string
	subRule  string
	message  string
}

var contradictionChecks = []contradictionCheck{
	{
		"always",
		"never",
		"always-never",
		"Document contains both 'always' and 'never' for potentially conflicting rules",
	},
	{
		"must not",
		"must",
		"must-conflict",
		"Document contains both 'must' and 'must not' - verify no conflicts",
	},
}

func (r *ContradictionsRule) Run(ctx *AnalysisContext) ([]Issue, error) {
	var issues []Issue

	for _, node := range ctx.FilesOfType(
		parser.FileCategoryInstructions,
		parser.FileCategoryDocumentation,
	) {
		if node.Content == nil {
			continue
		}

		contentLower := strings.ToLower(string(node.Content))

		for _, check := range contradictionChecks {
			if strings.Contains(contentLower, check.pattern1) && strings.Contains(contentLower, check.pattern2) {
				issues = append(issues, Issue{
					Rule:     r.Name() + "/" + check.subRule,
					Severity: Info,
					Message:  check.message,
					File:     node.Path,
					Line:     1,
				})
			}
		}
	}

	return issues, nil
}
