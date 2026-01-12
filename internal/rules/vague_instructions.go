package rules

import (
	"regexp"
	"strings"

	"github.com/pthm/cclint/internal/parser"
)

// VagueInstructionsRule checks for vague or unclear instructions
type VagueInstructionsRule struct{}

func (r *VagueInstructionsRule) Name() string {
	return "vague-instructions"
}

func (r *VagueInstructionsRule) Description() string {
	return "Checks for vague or unclear instructions that could be more specific"
}

func (r *VagueInstructionsRule) Config() RuleConfig {
	return RuleConfig{
		FileCategories: []parser.FileCategory{
			parser.FileCategoryInstructions,
			parser.FileCategoryDocumentation,
			parser.FileCategoryCommands,
		},
	}
}

// vaguePattern defines a pattern to match and its sub-rule name
type vaguePattern struct {
	pattern *regexp.Regexp
	subRule string
	message string
}

var vaguePatterns = []vaguePattern{
	{
		regexp.MustCompile(`(?i)\b(appropriate|proper|correct|good)\s+(way|manner|approach)\b`),
		"unclear-criteria",
		"Vague instruction: specify what makes something 'appropriate' or 'proper'",
	},
	{
		regexp.MustCompile(`(?i)\b(as needed|when necessary|if appropriate|as appropriate)\b`),
		"vague-condition",
		"Vague condition: specify concrete criteria",
	},
	{
		regexp.MustCompile(`(?i)\b(etc\.?|and so on|and more)\b`),
		"incomplete-list",
		"Incomplete list: consider being explicit about all options",
	},
	{
		regexp.MustCompile(`(?i)\b(be careful|take care|be mindful)\b`),
		"vague-guidance",
		"Vague guidance: specify what to watch out for",
	},
}

func (r *VagueInstructionsRule) Run(ctx *AnalysisContext) ([]Issue, error) {
	var issues []Issue

	for _, node := range ctx.FilesOfType(
		parser.FileCategoryInstructions,
		parser.FileCategoryDocumentation,
		parser.FileCategoryCommands,
	) {
		if node.Content == nil {
			continue
		}

		content := string(node.Content)
		lines := strings.Split(content, "\n")

		for lineNum, line := range lines {
			for _, vp := range vaguePatterns {
				if vp.pattern.MatchString(line) {
					issues = append(issues, Issue{
						Rule:     r.Name() + "/" + vp.subRule,
						Severity: Suggestion,
						Message:  vp.message,
						File:     node.Path,
						Line:     lineNum + 1,
						Context:  line,
					})
				}
			}
		}
	}

	return issues, nil
}
