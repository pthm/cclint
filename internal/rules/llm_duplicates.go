package rules

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/pthm/cclint/internal/analyzer"
	"github.com/pthm/cclint/internal/parser"
)

// LLMDuplicatesRule detects duplicate or near-duplicate instructions using LLM analysis
type LLMDuplicatesRule struct {
	base *LLMRuleBase
}

// NewLLMDuplicatesRule creates a new LLM duplicates rule.
// Returns nil if Claude Code CLI is not available.
func NewLLMDuplicatesRule() *LLMDuplicatesRule {
	base := NewLLMRuleBase()
	if base == nil {
		return nil
	}
	return &LLMDuplicatesRule{base: base}
}

func (r *LLMDuplicatesRule) Name() string {
	return "llm-duplicates"
}

func (r *LLMDuplicatesRule) Description() string {
	return "Uses AI to detect duplicate or semantically similar instructions within a context scope"
}

func (r *LLMDuplicatesRule) Config() RuleConfig {
	return RuleConfig{
		FileCategories: []parser.FileCategory{
			parser.FileCategoryInstructions,
			parser.FileCategoryDocumentation,
		},
		RequiresAI: true,
	}
}

func (r *LLMDuplicatesRule) Run(ctx *AnalysisContext) ([]Issue, error) {
	if r == nil || r.base == nil {
		return nil, fmt.Errorf("LLM rule not initialized")
	}

	return RunPerScope(ctx, r.Config().FileCategories, r.analyzeScope)
}

func (r *LLMDuplicatesRule) analyzeScope(scope *analyzer.ContextScope, files []*analyzer.ConfigNode) ([]Issue, error) {
	paths := FilteredFilePaths(files, 100) // Skip very small files
	if len(paths) == 0 {
		return nil, nil
	}

	// Use the first file's directory as cwd
	cwd := filepath.Dir(paths[0])

	prompt := fmt.Sprintf(`Analyze the following AI agent configuration files for DUPLICATE INSTRUCTIONS.

Context: %s
Files in scope:
%s
Read each file and identify instructions that are:
1. Exact duplicates (same text appearing multiple times)
2. Near-duplicates (same meaning with slightly different wording)
3. Redundant instructions (one instruction makes another unnecessary)

Focus on meaningful duplicates that waste context or could cause confusion.
Ignore intentional repetition for emphasis if clearly marked.

Return ONLY valid JSON with this structure:
{
  "issues": [
    {
      "file": "/path/to/file.md",
      "line": 42,
      "severity": "warning",
      "message": "Duplicate instruction found (also at file2.md:15)",
      "suggestion": "Remove one occurrence or consolidate"
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
