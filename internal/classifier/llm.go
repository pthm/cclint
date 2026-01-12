package classifier

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/pthm/cclint/internal/rules"
)

// LLMClassifier uses Claude for deep configuration analysis
type LLMClassifier struct {
	client anthropic.Client
}

// NewLLMClassifier creates a new LLM-based classifier
func NewLLMClassifier() *LLMClassifier {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		return nil
	}

	client := anthropic.NewClient(option.WithAPIKey(apiKey))
	return &LLMClassifier{client: client}
}

// Analyze runs LLM-based analysis on the configuration
func (c *LLMClassifier) Analyze(ctx *rules.AnalysisContext) ([]rules.Issue, error) {
	if c == nil {
		return nil, fmt.Errorf("LLM classifier not initialized (missing ANTHROPIC_API_KEY)")
	}

	var issues []rules.Issue

	for _, node := range ctx.Tree.Nodes {
		if node.Content == nil || len(node.Content) < 100 {
			continue
		}

		nodeIssues, err := c.analyzeWithClaude(node.Path, string(node.Content))
		if err != nil {
			// Log error but continue with other files
			continue
		}
		issues = append(issues, nodeIssues...)
	}

	return issues, nil
}

func (c *LLMClassifier) analyzeWithClaude(path, content string) ([]rules.Issue, error) {
	prompt := fmt.Sprintf(`Analyze this AI agent configuration file for quality and potential issues.

File: %s

Content:
%s

Provide a JSON response with the following structure:
{
  "issues": [
    {
      "severity": "info|suggestion|warning|error",
      "message": "description of the issue",
      "line": 0,
      "suggestion": "how to fix it"
    }
  ],
  "metrics": {
    "clarity": 0.0-1.0,
    "specificity": 0.0-1.0,
    "consistency": 0.0-1.0,
    "completeness": 0.0-1.0,
    "verbosity": "concise|moderate|verbose"
  }
}

Focus on:
1. Contradictory instructions
2. Vague or unclear guidance
3. Missing important context
4. Overly verbose sections
5. Potentially dangerous permissions or instructions
6. Best practices for AI agent configuration

Return ONLY the JSON, no other text.`, path, truncateContent(content, 8000))

	resp, err := c.client.Messages.New(context.Background(), anthropic.MessageNewParams{
		Model:     anthropic.ModelClaude3_5Haiku20241022,
		MaxTokens: 2000,
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("Claude API error: %w", err)
	}

	// Extract text from response
	var responseText string
	for _, block := range resp.Content {
		if block.Type == "text" {
			responseText = block.Text
			break
		}
	}

	// Parse response
	var result struct {
		Issues []struct {
			Severity   string `json:"severity"`
			Message    string `json:"message"`
			Line       int    `json:"line"`
			Suggestion string `json:"suggestion"`
		} `json:"issues"`
		Metrics QualityMetrics `json:"metrics"`
	}

	if err := json.Unmarshal([]byte(responseText), &result); err != nil {
		return nil, fmt.Errorf("failed to parse Claude response: %w", err)
	}

	// Convert to rules.Issue
	var issues []rules.Issue
	for _, issue := range result.Issues {
		severity := rules.Info
		switch issue.Severity {
		case "suggestion":
			severity = rules.Suggestion
		case "warning":
			severity = rules.Warning
		case "error":
			severity = rules.Error
		}

		msg := issue.Message
		if issue.Suggestion != "" {
			msg = fmt.Sprintf("%s (Suggestion: %s)", issue.Message, issue.Suggestion)
		}

		issues = append(issues, rules.Issue{
			Rule:     "classifier/llm",
			Severity: severity,
			Message:  msg,
			File:     path,
			Line:     issue.Line,
		})
	}

	// Add overall metrics as info
	if result.Metrics.OverallScore > 0 && result.Metrics.OverallScore < 0.6 {
		issues = append(issues, rules.Issue{
			Rule:     "classifier/llm-quality",
			Severity: rules.Suggestion,
			Message: fmt.Sprintf(
				"Overall quality score: %.0f%% (clarity: %.0f%%, specificity: %.0f%%, verbosity: %s)",
				result.Metrics.OverallScore*100,
				result.Metrics.Clarity*100,
				result.Metrics.Specificity*100,
				result.Metrics.Verbosity,
			),
			File: path,
			Line: 1,
		})
	}

	return issues, nil
}

// truncateContent truncates content to a maximum length
func truncateContent(content string, maxLen int) string {
	if len(content) <= maxLen {
		return content
	}
	return content[:maxLen] + "\n...[truncated]..."
}
