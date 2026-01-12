package rules

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/pthm/cclint/internal/analyzer"
	"github.com/pthm/cclint/internal/parser"
)

// LLMClarityRule detects unclear or vague instructions using LLM analysis
type LLMClarityRule struct {
	base *LLMRuleBase
}

// NewLLMClarityRule creates a new LLM clarity rule.
// Returns nil if Claude Code CLI is not available.
func NewLLMClarityRule() *LLMClarityRule {
	base := NewLLMRuleBase()
	if base == nil {
		return nil
	}
	return &LLMClarityRule{base: base}
}

func (r *LLMClarityRule) Name() string {
	return "llm-clarity"
}

func (r *LLMClarityRule) Description() string {
	return "Uses AI to detect unclear, vague, or ambiguous instructions within a context scope"
}

func (r *LLMClarityRule) Config() RuleConfig {
	return RuleConfig{
		FileCategories: []parser.FileCategory{
			parser.FileCategoryInstructions,
			parser.FileCategoryDocumentation,
		},
		RequiresAI: true,
	}
}

func (r *LLMClarityRule) Run(ctx *AnalysisContext) ([]Issue, error) {
	if r == nil || r.base == nil {
		return nil, fmt.Errorf("LLM rule not initialized")
	}

	return RunPerScope(ctx, r.Config().FileCategories, r.analyzeScope)
}

func (r *LLMClarityRule) analyzeScope(scope *analyzer.ContextScope, files []*analyzer.ConfigNode) ([]Issue, error) {
	paths := FilteredFilePaths(files, 100)
	if len(paths) == 0 {
		return nil, nil
	}

	cwd := filepath.Dir(paths[0])

	prompt := fmt.Sprintf(`Analyze the following AI agent configuration files for CLARITY ISSUES.

Context: %s
Files in scope:
%s
Read each file and identify instructions that are unclear or vague:
1. Ambiguous language (e.g., "use appropriate methods", "be careful")
2. Missing specifics (e.g., "format properly" without defining the format)
3. Unclear scope (e.g., "sometimes" without defining when)
4. Jargon or unexplained terms that an AI might misinterpret
5. Instructions that could be interpreted multiple ways

Focus on instructions that an AI agent would struggle to follow consistently.
Consider the AI's perspective - what would be confusing without additional context?

Return ONLY valid JSON with this structure:
{
  "issues": [
    {
      "file": "/path/to/file.md",
      "line": 42,
      "severity": "warning",
      "message": "Vague instruction: 'format appropriately' - unclear what format is expected",
      "suggestion": "Specify the exact format, e.g., 'use ISO 8601 date format (YYYY-MM-DD)'"
    }
  ]
}

Return ONLY valid JSON, no markdown, no explanatory text.`,
		ScopeContextDescription(scope),
		BuildFileList(paths))

	responseText, err := r.base.ExecuteQuery(context.Background(), prompt, cwd)
	if err != nil {
		return nil, err
	}

	jsonStr := ExtractJSON(responseText)
	var response LLMResponse
	if err := json.Unmarshal([]byte(jsonStr), &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w (response: %s)", err, TruncateForError(jsonStr))
	}

	return response.ToRuleIssues(r.Name(), scope.Entrypoint), nil
}
