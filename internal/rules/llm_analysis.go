// Deprecated: This file contains the original monolithic LLM analysis rule.
// It has been replaced by focused per-scope rules:
// - llm_duplicates.go
// - llm_contradictions.go
// - llm_clarity.go
// - llm_actionability.go
//
// This file is kept for reference but is no longer registered in the default registry.
// It will be removed in a future version.

package rules

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	claudecode "github.com/severity1/claude-agent-sdk-go"
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

// LLMAnalysisRule uses Claude Code for deep configuration analysis
type LLMAnalysisRule struct {
	available bool
}

// NewLLMAnalysisRule creates a new LLM analysis rule.
// Returns nil if Claude Code CLI is not available.
func NewLLMAnalysisRule() *LLMAnalysisRule {
	// Test if Claude Code is available by attempting a simple query
	ctx := context.Background()
	_, err := claudecode.Query(ctx, "echo test",
		claudecode.WithMaxTurns(1),
	)
	if err != nil {
		// Check if it's a CLI not found error
		if claudecode.IsCLINotFoundError(err) {
			return nil
		}
		// Other errors might be temporary, allow the rule to be created
	}

	return &LLMAnalysisRule{available: true}
}

func (r *LLMAnalysisRule) Name() string {
	return "llm-analysis"
}

func (r *LLMAnalysisRule) Description() string {
	return "Uses Claude Code for deep analysis of configuration quality"
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
	if r == nil || !r.available {
		return nil, fmt.Errorf("LLM rule not initialized (Claude Code CLI not available)")
	}

	var issues []Issue

	for _, node := range ctx.FilesOfType(
		parser.FileCategoryInstructions,
		parser.FileCategoryDocumentation,
	) {
		if len(node.Content) < 100 {
			continue
		}

		nodeIssues, err := r.analyzeWithClaudeCode(context.Background(), node.Path)
		if err != nil {
			// Log error but continue with other files
			continue
		}
		issues = append(issues, nodeIssues...)
	}

	return issues, nil
}

// llmAnalysisResponse is the expected JSON response from Claude Code
type llmAnalysisResponse struct {
	Issues []struct {
		Severity   string `json:"severity"`
		Message    string `json:"message"`
		Line       int    `json:"line"`
		Suggestion string `json:"suggestion"`
	} `json:"issues"`
	Metrics llmQualityMetrics `json:"metrics"`
}

func (r *LLMAnalysisRule) analyzeWithClaudeCode(ctx context.Context, path string) ([]Issue, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	prompt := fmt.Sprintf(`Analyze the AI agent configuration file at %s for quality and potential issues.

Read the file and provide a JSON response with this exact structure (no other text, just JSON):
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
    "verbosity": "concise|moderate|verbose",
    "overall_score": 0.0-1.0
  }
}

Focus on:
1. Contradictory instructions
2. Vague or unclear guidance
3. Missing important context
4. Overly verbose sections
5. Potentially dangerous permissions or instructions
6. Best practices for AI agent configuration

Return ONLY valid JSON, no markdown code blocks, no explanatory text.`, absPath)

	iterator, err := claudecode.Query(ctx, prompt,
		claudecode.WithCwd(filepath.Dir(absPath)),
		claudecode.WithMaxTurns(3), // Allow reading files
		claudecode.WithAllowedTools("Read"), // Only allow reading
	)
	if err != nil {
		return nil, fmt.Errorf("claude code error: %w", err)
	}
	defer iterator.Close()

	// Collect all text from the response
	var responseBuilder strings.Builder
	for {
		message, err := iterator.Next(ctx)
		if err != nil {
			if errors.Is(err, claudecode.ErrNoMoreMessages) {
				break
			}
			return nil, fmt.Errorf("error reading claude response: %w", err)
		}

		// Extract text from assistant messages
		if assistantMsg, ok := message.(*claudecode.AssistantMessage); ok {
			for _, block := range assistantMsg.Content {
				if textBlock, ok := block.(*claudecode.TextBlock); ok {
					responseBuilder.WriteString(textBlock.Text)
				}
			}
		}
	}

	responseText := responseBuilder.String()
	if responseText == "" {
		return nil, fmt.Errorf("empty response from claude code")
	}

	// Try to extract JSON from the response
	responseText = extractJSON(responseText)

	var response llmAnalysisResponse
	if err := json.Unmarshal([]byte(responseText), &response); err != nil {
		return nil, fmt.Errorf("failed to parse Claude Code response: %w (response: %s)", err, truncateForError(responseText))
	}

	// Convert to rules.Issue
	var issues []Issue
	for _, issue := range response.Issues {
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
	if response.Metrics.OverallScore > 0 && response.Metrics.OverallScore < 0.6 {
		issues = append(issues, Issue{
			Rule:     r.Name() + "/quality",
			Severity: Suggestion,
			Message: fmt.Sprintf(
				"Overall quality score: %.0f%% (clarity: %.0f%%, specificity: %.0f%%, verbosity: %s)",
				response.Metrics.OverallScore*100,
				response.Metrics.Clarity*100,
				response.Metrics.Specificity*100,
				response.Metrics.Verbosity,
			),
			File: path,
			Line: 1,
		})
	}

	return issues, nil
}

// extractJSON attempts to extract JSON from a response that might be wrapped in markdown
func extractJSON(s string) string {
	s = strings.TrimSpace(s)

	// If it starts with {, assume it's already JSON
	if strings.HasPrefix(s, "{") {
		return s
	}

	// Try to find JSON block in markdown
	if idx := strings.Index(s, "```json"); idx != -1 {
		start := idx + 7
		if end := strings.Index(s[start:], "```"); end != -1 {
			return strings.TrimSpace(s[start : start+end])
		}
	}

	// Try to find raw JSON block
	if idx := strings.Index(s, "```"); idx != -1 {
		start := idx + 3
		// Skip any language identifier
		if nlIdx := strings.Index(s[start:], "\n"); nlIdx != -1 {
			start += nlIdx + 1
		}
		if end := strings.Index(s[start:], "```"); end != -1 {
			return strings.TrimSpace(s[start : start+end])
		}
	}

	// Try to find { ... } pattern
	if start := strings.Index(s, "{"); start != -1 {
		if end := strings.LastIndex(s, "}"); end > start {
			return s[start : end+1]
		}
	}

	return s
}

// truncateForError truncates a string for inclusion in error messages
func truncateForError(s string) string {
	if len(s) > 200 {
		return s[:200] + "..."
	}
	return s
}
