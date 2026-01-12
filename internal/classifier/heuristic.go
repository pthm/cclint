package classifier

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/pthm/cclint/internal/parser"
	"github.com/pthm/cclint/internal/rules"
)

// HeuristicClassifier uses rule-based heuristics for classification
type HeuristicClassifier struct{}

// NewHeuristicClassifier creates a new heuristic classifier
func NewHeuristicClassifier() *HeuristicClassifier {
	return &HeuristicClassifier{}
}

// Analyze runs heuristic analysis on the configuration
func (c *HeuristicClassifier) Analyze(ctx *rules.AnalysisContext) ([]rules.Issue, error) {
	var issues []rules.Issue

	for _, node := range ctx.Tree.Nodes {
		if node.Content == nil {
			continue
		}

		// Get the file category
		category := parser.FileCategoryUnknown
		if node.Parsed != nil {
			category = node.Parsed.Category
		}

		content := string(node.Content)
		nodeIssues := c.analyzeContent(node.Path, content, category)
		issues = append(issues, nodeIssues...)
	}

	return issues, nil
}

func (c *HeuristicClassifier) analyzeContent(path, content string, category parser.FileCategory) []rules.Issue {
	var issues []rules.Issue

	// Skip instruction-quality checks for config files
	// Config files are structural settings, not directive content
	if category == parser.FileCategoryConfig {
		return issues
	}

	// Check for vague instructions
	issues = append(issues, c.checkVagueInstructions(path, content)...)

	// Check for contradictions
	issues = append(issues, c.checkContradictions(path, content)...)

	// Check for missing context
	issues = append(issues, c.checkMissingContext(path, content)...)

	// Check for verbosity
	issues = append(issues, c.checkVerbosity(path, content)...)

	return issues
}

func (c *HeuristicClassifier) checkVagueInstructions(path, content string) []rules.Issue {
	var issues []rules.Issue

	vaguePatterns := []struct {
		pattern *regexp.Regexp
		message string
	}{
		{
			regexp.MustCompile(`(?i)\b(appropriate|proper|correct|good)\s+(way|manner|approach)\b`),
			"Vague instruction: specify what makes something 'appropriate' or 'proper'",
		},
		{
			regexp.MustCompile(`(?i)\b(as needed|when necessary|if appropriate|as appropriate)\b`),
			"Vague condition: specify concrete criteria",
		},
		{
			regexp.MustCompile(`(?i)\b(etc\.?|and so on|and more)\b`),
			"Incomplete list: consider being explicit about all options",
		},
		{
			regexp.MustCompile(`(?i)\b(be careful|take care|be mindful)\b`),
			"Vague guidance: specify what to watch out for",
		},
	}

	lines := strings.Split(content, "\n")
	for lineNum, line := range lines {
		for _, vp := range vaguePatterns {
			if vp.pattern.MatchString(line) {
				issues = append(issues, rules.Issue{
					Rule:     "classifier/vague-instruction",
					Severity: rules.Suggestion,
					Message:  vp.message,
					File:     path,
					Line:     lineNum + 1,
					Context:  line,
				})
			}
		}
	}

	return issues
}

func (c *HeuristicClassifier) checkContradictions(path, content string) []rules.Issue {
	var issues []rules.Issue

	contentLower := strings.ToLower(content)

	// Check for common contradiction patterns
	contradictions := []struct {
		pattern1 string
		pattern2 string
		message  string
	}{
		{
			"always",
			"never",
			"Document contains both 'always' and 'never' for potentially conflicting rules",
		},
		{
			"must not",
			"must",
			"Document contains both 'must' and 'must not' - verify no conflicts",
		},
	}

	for _, c := range contradictions {
		if strings.Contains(contentLower, c.pattern1) && strings.Contains(contentLower, c.pattern2) {
			// This is a very basic check - just flags for review
			issues = append(issues, rules.Issue{
				Rule:     "classifier/potential-contradiction",
				Severity: rules.Info,
				Message:  c.message,
				File:     path,
				Line:     1,
			})
		}
	}

	return issues
}

func (c *HeuristicClassifier) checkMissingContext(path, content string) []rules.Issue {
	var issues []rules.Issue

	// Check for instructions without examples
	if strings.Contains(strings.ToLower(content), "important") ||
		strings.Contains(strings.ToLower(content), "must") {

		hasExamples := strings.Contains(strings.ToLower(content), "example") ||
			strings.Contains(content, "```") ||
			strings.Contains(strings.ToLower(content), "for instance")

		if !hasExamples {
			issues = append(issues, rules.Issue{
				Rule:     "classifier/missing-examples",
				Severity: rules.Suggestion,
				Message:  "Document contains important instructions but no examples - consider adding concrete examples",
				File:     path,
				Line:     1,
			})
		}
	}

	return issues
}

func (c *HeuristicClassifier) checkVerbosity(path, content string) []rules.Issue {
	var issues []rules.Issue

	lines := strings.Split(content, "\n")
	words := strings.Fields(content)

	// Check for overly long sentences
	longSentences := 0
	for _, line := range lines {
		lineWords := strings.Fields(line)
		if len(lineWords) > 40 {
			longSentences++
		}
	}

	if longSentences > 5 {
		issues = append(issues, rules.Issue{
			Rule:     "classifier/verbose",
			Severity: rules.Suggestion,
			Message:  fmt.Sprintf("Document has %d overly long sentences - consider breaking them up for clarity", longSentences),
			File:     path,
			Line:     1,
		})
	}

	// Check word-to-instruction ratio (rough heuristic)
	instructionWords := []string{"must", "should", "always", "never", "do", "don't", "use", "avoid"}
	instructionCount := 0
	for _, word := range words {
		for _, iw := range instructionWords {
			if strings.ToLower(word) == iw {
				instructionCount++
				break
			}
		}
	}

	if len(words) > 500 && instructionCount < 10 {
		issues = append(issues, rules.Issue{
			Rule:     "classifier/low-instruction-density",
			Severity: rules.Info,
			Message:  "Document is lengthy but has few actionable instructions - consider being more directive",
			File:     path,
			Line:     1,
		})
	}

	return issues
}
