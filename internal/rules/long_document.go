package rules

import (
	"fmt"

	"github.com/pthm-cable/cclint/internal/parser"
)

// LongDocumentRule checks for overly long configuration documents
type LongDocumentRule struct {
	MaxLines  int
	MaxTokens int
}

func (r *LongDocumentRule) Name() string {
	return "long-document"
}

func (r *LongDocumentRule) Description() string {
	return "Checks for configuration documents that are too long"
}

func (r *LongDocumentRule) Config() RuleConfig {
	return RuleConfig{
		FileCategories: []parser.FileCategory{
			parser.FileCategoryInstructions,
			parser.FileCategoryDocumentation,
			parser.FileCategoryCommands,
		},
	}
}

func (r *LongDocumentRule) Run(ctx *AnalysisContext) ([]Issue, error) {
	var issues []Issue

	maxLines := r.MaxLines
	if maxLines == 0 {
		maxLines = 500 // Default
	}

	maxTokens := r.MaxTokens
	if maxTokens == 0 {
		maxTokens = 4000 // Default
	}

	for _, node := range ctx.FilesOfType(
		parser.FileCategoryInstructions,
		parser.FileCategoryDocumentation,
		parser.FileCategoryCommands,
	) {
		if node.Content == nil {
			continue
		}

		// Count lines
		lines := 0
		for _, b := range node.Content {
			if b == '\n' {
				lines++
			}
		}

		// Estimate tokens (~4 chars per token)
		tokens := len(node.Content) / 4

		if lines > maxLines {
			issues = append(issues, Issue{
				Rule:     r.Name(),
				Severity: Warning,
				Message:  fmt.Sprintf("Document has %d lines, exceeds recommended maximum of %d", lines, maxLines),
				File:     node.Path,
				Line:     1,
			})
		}

		if tokens > maxTokens {
			issues = append(issues, Issue{
				Rule:     r.Name(),
				Severity: Warning,
				Message:  fmt.Sprintf("Document has ~%d tokens, exceeds recommended maximum of %d", tokens, maxTokens),
				File:     node.Path,
				Line:     1,
			})
		}
	}

	return issues, nil
}
