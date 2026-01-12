package rules

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/pthm/cclint/internal/analyzer"
	"github.com/pthm/cclint/internal/parser"
)

// LLMContradictionsRule detects contradicting instructions using LLM analysis
type LLMContradictionsRule struct {
	base *LLMRuleBase
}

// NewLLMContradictionsRule creates a new LLM contradictions rule.
// Returns nil if Claude Code CLI is not available.
func NewLLMContradictionsRule() *LLMContradictionsRule {
	base := NewLLMRuleBase()
	if base == nil {
		return nil
	}
	return &LLMContradictionsRule{base: base}
}

func (r *LLMContradictionsRule) Name() string {
	return "llm-contradictions"
}

func (r *LLMContradictionsRule) Description() string {
	return "Uses AI to detect contradicting instructions within a context scope"
}

func (r *LLMContradictionsRule) Config() RuleConfig {
	return RuleConfig{
		FileCategories: []parser.FileCategory{
			parser.FileCategoryInstructions,
			parser.FileCategoryDocumentation,
		},
		RequiresAI: true,
	}
}

func (r *LLMContradictionsRule) Run(ctx *AnalysisContext) ([]Issue, error) {
	if r == nil || r.base == nil {
		return nil, fmt.Errorf("LLM rule not initialized")
	}

	return RunPerScope(ctx, r.Config().FileCategories, r.analyzeScope)
}

func (r *LLMContradictionsRule) analyzeScope(scope *analyzer.ContextScope, files []*analyzer.ConfigNode) ([]Issue, error) {
	paths := FilteredFilePaths(files, 100)
	if len(paths) == 0 {
		return nil, nil
	}

	cwd := filepath.Dir(paths[0])

	prompt := fmt.Sprintf(`Analyze the following AI agent configuration files for CONTRADICTING INSTRUCTIONS.

Context: %s
Files in scope:
%s
Read each file and identify instructions that contradict each other:
1. Direct contradictions (e.g., "always do X" vs "never do X")
2. Logical conflicts (e.g., "use tabs" vs "use spaces")
3. Conflicting priorities (e.g., "prioritize speed" vs "prioritize quality" without resolution)
4. Inconsistent behaviors (e.g., different error handling for similar cases)

Focus on contradictions that would confuse an AI agent about what to do.
Consider the context - some apparent contradictions may be valid for different situations.

Return ONLY valid JSON with this structure:
{
  "issues": [
    {
      "file": "/path/to/file.md",
      "line": 42,
      "severity": "error",
      "message": "Contradicts instruction at file2.md:15 - one says X, the other says Y",
      "suggestion": "Clarify which instruction takes precedence or under what conditions each applies"
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
