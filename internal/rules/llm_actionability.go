package rules

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/pthm-cable/cclint/internal/analyzer"
	"github.com/pthm-cable/cclint/internal/parser"
)

// LLMActionabilityRule detects instructions with low actionability using LLM analysis
type LLMActionabilityRule struct {
	base *LLMRuleBase
}

// NewLLMActionabilityRule creates a new LLM actionability rule.
// Returns nil if Claude Code CLI is not available.
func NewLLMActionabilityRule() *LLMActionabilityRule {
	base := NewLLMRuleBase()
	if base == nil {
		return nil
	}
	return &LLMActionabilityRule{base: base}
}

func (r *LLMActionabilityRule) Name() string {
	return "llm-actionability"
}

func (r *LLMActionabilityRule) Description() string {
	return "Uses AI to detect instructions that lack clear actions or measurable outcomes"
}

func (r *LLMActionabilityRule) Config() RuleConfig {
	return RuleConfig{
		FileCategories: []parser.FileCategory{
			parser.FileCategoryInstructions,
			parser.FileCategoryDocumentation,
		},
		RequiresAI: true,
	}
}

func (r *LLMActionabilityRule) Run(ctx *AnalysisContext) ([]Issue, error) {
	if r == nil || r.base == nil {
		return nil, fmt.Errorf("LLM rule not initialized")
	}

	return RunPerScope(ctx, r.Config().FileCategories, r.analyzeScope)
}

func (r *LLMActionabilityRule) analyzeScope(scope *analyzer.ContextScope, files []*analyzer.ConfigNode) ([]Issue, error) {
	paths := FilteredFilePaths(files, 100)
	if len(paths) == 0 {
		return nil, nil
	}

	cwd := filepath.Dir(paths[0])

	prompt := fmt.Sprintf(`Analyze the following AI agent configuration files for LOW ACTIONABILITY.

Context: %s
Files in scope:
%s
Read each file and identify instructions that lack actionability:
1. No clear action (e.g., "keep in mind that..." without a resulting behavior)
2. No measurable outcome (e.g., "write good code" - how do you verify this?)
3. Missing triggers (e.g., "handle errors gracefully" - when and how?)
4. Passive observations (e.g., "users may want..." without an instruction)
5. Aspirational statements (e.g., "strive to be helpful") vs concrete guidance

Focus on instructions that an AI agent cannot reliably execute or verify.
Good instructions have: a trigger condition, an action to take, and a way to verify success.

Return ONLY valid JSON with this structure:
{
  "issues": [
    {
      "file": "/path/to/file.md",
      "line": 42,
      "severity": "suggestion",
      "message": "Low actionability: 'be mindful of performance' - no specific action defined",
      "suggestion": "Rephrase as: 'When writing loops, avoid O(n²) algorithms; prefer O(n log n) or better'"
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
