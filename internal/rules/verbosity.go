package rules

import (
	"fmt"
	"strings"

	"github.com/pthm/cclint/internal/parser"
)

// VerbosityRule checks for overly verbose or low-density instructions
type VerbosityRule struct{}

func (r *VerbosityRule) Name() string {
	return "verbosity"
}

func (r *VerbosityRule) Description() string {
	return "Checks for overly verbose content or low instruction density"
}

func (r *VerbosityRule) Config() RuleConfig {
	return RuleConfig{
		FileCategories: []parser.FileCategory{
			parser.FileCategoryInstructions,
			parser.FileCategoryDocumentation,
		},
	}
}

func (r *VerbosityRule) Run(ctx *AnalysisContext) ([]Issue, error) {
	var issues []Issue

	for _, node := range ctx.FilesOfType(
		parser.FileCategoryInstructions,
		parser.FileCategoryDocumentation,
	) {
		if node.Content == nil {
			continue
		}

		content := string(node.Content)
		issues = append(issues, r.checkLongSentences(node.Path, content)...)
		issues = append(issues, r.checkInstructionDensity(node.Path, content)...)
	}

	return issues, nil
}

func (r *VerbosityRule) checkLongSentences(path, content string) []Issue {
	var issues []Issue

	lines := strings.Split(content, "\n")
	longSentences := 0

	for _, line := range lines {
		lineWords := strings.Fields(line)
		if len(lineWords) > 40 {
			longSentences++
		}
	}

	if longSentences > 5 {
		issues = append(issues, Issue{
			Rule:     r.Name() + "/long-sentences",
			Severity: Suggestion,
			Message:  fmt.Sprintf("Document has %d overly long sentences - consider breaking them up for clarity", longSentences),
			File:     path,
			Line:     1,
		})
	}

	return issues
}

func (r *VerbosityRule) checkInstructionDensity(path, content string) []Issue {
	var issues []Issue

	words := strings.Fields(content)
	instructionWords := []string{"must", "should", "always", "never", "do", "don't", "use", "avoid"}

	instructionCount := 0
	for _, word := range words {
		wordLower := strings.ToLower(word)
		for _, iw := range instructionWords {
			if wordLower == iw {
				instructionCount++
				break
			}
		}
	}

	if len(words) > 500 && instructionCount < 10 {
		issues = append(issues, Issue{
			Rule:     r.Name() + "/low-instruction-density",
			Severity: Info,
			Message:  "Document is lengthy but has few actionable instructions - consider being more directive",
			File:     path,
			Line:     1,
		})
	}

	return issues
}
