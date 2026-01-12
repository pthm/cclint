package rules

import (
	"strings"

	"github.com/pthm/cclint/internal/parser"
)

// MissingContextRule checks for instructions that lack supporting context
type MissingContextRule struct{}

func (r *MissingContextRule) Name() string {
	return "missing-context"
}

func (r *MissingContextRule) Description() string {
	return "Checks for important instructions that lack examples or supporting context"
}

func (r *MissingContextRule) Config() RuleConfig {
	return RuleConfig{
		FileCategories: []parser.FileCategory{
			parser.FileCategoryInstructions,
		},
	}
}

func (r *MissingContextRule) Run(ctx *AnalysisContext) ([]Issue, error) {
	var issues []Issue

	for _, node := range ctx.FilesOfType(parser.FileCategoryInstructions) {
		if node.Content == nil {
			continue
		}

		content := string(node.Content)
		contentLower := strings.ToLower(content)

		// Check for instructions without examples
		hasImportantInstructions := strings.Contains(contentLower, "important") ||
			strings.Contains(contentLower, "must")

		if hasImportantInstructions {
			hasExamples := strings.Contains(contentLower, "example") ||
				strings.Contains(content, "```") ||
				strings.Contains(contentLower, "for instance")

			if !hasExamples {
				issues = append(issues, Issue{
					Rule:     r.Name() + "/no-examples",
					Severity: Suggestion,
					Message:  "Document contains important instructions but no examples - consider adding concrete examples",
					File:     node.Path,
					Line:     1,
				})
			}
		}
	}

	return issues, nil
}
