package rules

import (
	"context"
	"errors"
	"fmt"
	"strings"

	claudecode "github.com/severity1/claude-agent-sdk-go"
)

// LLMRuleBase provides shared functionality for LLM-based rules
type LLMRuleBase struct {
	available bool
}

// NewLLMRuleBase creates a new LLMRuleBase, checking if Claude Code is available.
// Returns nil if the CLI is not found.
func NewLLMRuleBase() *LLMRuleBase {
	ctx := context.Background()
	_, err := claudecode.Query(ctx, "echo test",
		claudecode.WithModel("sonnet"),
		claudecode.WithMaxTurns(1),
	)
	if err != nil {
		if claudecode.IsCLINotFoundError(err) {
			return nil
		}
		// Other errors might be temporary, allow creation
	}
	return &LLMRuleBase{available: true}
}

// IsAvailable returns whether the LLM rule can run
func (b *LLMRuleBase) IsAvailable() bool {
	return b != nil && b.available
}

// CheckAvailable returns an error if the rule cannot run
func (b *LLMRuleBase) CheckAvailable() error {
	if !b.IsAvailable() {
		return fmt.Errorf("LLM rule not initialized (Claude Code CLI not available)")
	}
	return nil
}

// ExecuteQuery runs a query against Claude Code and returns the response text
func (b *LLMRuleBase) ExecuteQuery(ctx context.Context, prompt string, cwd string) (string, error) {
	if err := b.CheckAvailable(); err != nil {
		return "", err
	}

	var iterator claudecode.MessageIterator
	var err error

	if cwd != "" {
		iterator, err = claudecode.Query(ctx, prompt,
			claudecode.WithModel("sonnet"),
			claudecode.WithMaxTurns(3),
			claudecode.WithAllowedTools("Read"),
			claudecode.WithCwd(cwd),
		)
	} else {
		iterator, err = claudecode.Query(ctx, prompt,
			claudecode.WithModel("sonnet"),
			claudecode.WithMaxTurns(3),
			claudecode.WithAllowedTools("Read"),
		)
	}
	if err != nil {
		return "", fmt.Errorf("claude code error: %w", err)
	}
	defer iterator.Close()

	var responseBuilder strings.Builder
	for {
		message, err := iterator.Next(ctx)
		if err != nil {
			if errors.Is(err, claudecode.ErrNoMoreMessages) {
				break
			}
			return "", fmt.Errorf("error reading claude response: %w", err)
		}

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
		return "", fmt.Errorf("empty response from claude code")
	}

	return responseText, nil
}

// ExtractJSON attempts to extract JSON from a response that might be wrapped in markdown
func ExtractJSON(s string) string {
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

// TruncateForError truncates a string for inclusion in error messages
func TruncateForError(s string) string {
	if len(s) > 200 {
		return s[:200] + "..."
	}
	return s
}

// LLMIssue represents an issue from LLM analysis
type LLMIssue struct {
	File       string `json:"file"`
	Line       int    `json:"line"`
	Severity   string `json:"severity"`
	Message    string `json:"message"`
	Suggestion string `json:"suggestion,omitempty"`
}

// LLMResponse is the common response structure from LLM rules
type LLMResponse struct {
	Issues []LLMIssue `json:"issues"`
}

// ToRuleIssues converts LLM issues to rule issues
func (r *LLMResponse) ToRuleIssues(ruleName string, defaultFile string) []Issue {
	var issues []Issue
	for _, llmIssue := range r.Issues {
		severity := Info
		switch llmIssue.Severity {
		case "suggestion":
			severity = Suggestion
		case "warning":
			severity = Warning
		case "error":
			severity = Error
		}

		msg := llmIssue.Message
		if llmIssue.Suggestion != "" {
			msg = fmt.Sprintf("%s (Suggestion: %s)", llmIssue.Message, llmIssue.Suggestion)
		}

		file := llmIssue.File
		if file == "" {
			file = defaultFile
		}

		issues = append(issues, Issue{
			Rule:     ruleName,
			Severity: severity,
			Message:  msg,
			File:     file,
			Line:     llmIssue.Line,
		})
	}
	return issues
}
