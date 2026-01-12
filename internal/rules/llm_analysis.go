package rules

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/pthm-cable/cclint/internal/parser"
)

// llmQualityMetrics holds quality assessment metrics from LLM analysis
type llmQualityMetrics struct {
	Clarity      float64 `json:"clarity"`
	Specificity  float64 `json:"specificity"`
	Consistency  float64 `json:"consistency"`
	Completeness float64 `json:"completeness"`
	Verbosity    string  `json:"verbosity"`
	OverallScore float64 `json:"overall_score"`
}

// LLMAnalysisRule uses Claude for deep configuration analysis
type LLMAnalysisRule struct {
	client anthropic.Client
}

// NewLLMAnalysisRule creates a new LLM analysis rule.
// Returns nil if ANTHROPIC_API_KEY is not set.
func NewLLMAnalysisRule() *LLMAnalysisRule {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		return nil
	}

	client := anthropic.NewClient(option.WithAPIKey(apiKey))
	return &LLMAnalysisRule{client: client}
}

func (r *LLMAnalysisRule) Name() string {
	return "llm-analysis"
}

func (r *LLMAnalysisRule) Description() string {
	return "Uses Claude AI for deep analysis of configuration quality"
}

func (r *LLMAnalysisRule) Config() RuleConfig {
	return RuleConfig{
		FileCategories: []parser.FileCategory{
			parser.FileCategoryInstructions,
			parser.FileCategoryDocumentation,
		},
		RequiresAI: true,
	}
}

func (r *LLMAnalysisRule) Run(ctx *AnalysisContext) ([]Issue, error) {
	if r == nil {
		return nil, fmt.Errorf("LLM rule not initialized (missing ANTHROPIC_API_KEY)")
	}

	var issues []Issue

	for _, node := range ctx.FilesOfType(
		parser.FileCategoryInstructions,
		parser.FileCategoryDocumentation,
	) {
		if len(node.Content) < 100 {
			continue
		}

		nodeIssues, err := r.analyzeWithClaude(node.Path, string(node.Content))
		if err != nil {
			// Log error but continue with other files
			continue
		}
		issues = append(issues, nodeIssues...)
	}

	return issues, nil
}

func (r *LLMAnalysisRule) analyzeWithClaude(path, content string) ([]Issue, error) {
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

	resp, err := r.client.Messages.New(context.Background(), anthropic.MessageNewParams{
		Model:     anthropic.ModelClaude3_5Haiku20241022,
		MaxTokens: 2000,
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("claude API error: %w", err)
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
		Metrics llmQualityMetrics `json:"metrics"`
	}

	if err := json.Unmarshal([]byte(responseText), &result); err != nil {
		return nil, fmt.Errorf("failed to parse Claude response: %w", err)
	}

	// Convert to rules.Issue
	var issues []Issue
	for _, issue := range result.Issues {
		severity := Info
		switch issue.Severity {
		case "suggestion":
			severity = Suggestion
		case "warning":
			severity = Warning
		case "error":
			severity = Error
		}

		msg := issue.Message
		if issue.Suggestion != "" {
			msg = fmt.Sprintf("%s (Suggestion: %s)", issue.Message, issue.Suggestion)
		}

		issues = append(issues, Issue{
			Rule:     r.Name(),
			Severity: severity,
			Message:  msg,
			File:     path,
			Line:     issue.Line,
		})
	}

	// Add overall metrics as info if quality is low
	if result.Metrics.OverallScore > 0 && result.Metrics.OverallScore < 0.6 {
		issues = append(issues, Issue{
			Rule:     r.Name() + "/quality",
			Severity: Suggestion,
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
