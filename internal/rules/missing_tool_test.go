package rules

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pthm/cclint/internal/agent"
	"github.com/pthm/cclint/internal/analyzer"
)

func TestMissingToolRule_Name(t *testing.T) {
	r := &MissingToolRule{}
	if r.Name() != "missing-tool" {
		t.Errorf("Name() = %q, want %q", r.Name(), "missing-tool")
	}
}

func TestMissingToolRule_toolExists(t *testing.T) {
	r := &MissingToolRule{}

	// Claude built-in tools should exist
	builtins := []string{"Read", "Write", "Edit", "Bash", "Glob", "Grep", "Task"}
	for _, tool := range builtins {
		if !r.toolExists(tool) {
			t.Errorf("toolExists(%q) = false, want true (Claude built-in)", tool)
		}
	}

	// Common OS tools that should exist on most systems
	// We only check ones that are very likely to be present
	// Note: "ls" should exist on Unix-like systems, but we don't fail the test
	// if it doesn't since this test runs on various environments
	_ = r.toolExists("ls")

	// Non-existent tool
	if r.toolExists("definitely-not-a-real-tool-xyz123") {
		t.Error("toolExists(nonexistent) = true, want false")
	}
}

func TestMissingToolRule_Run(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "missing-tool-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dirs := []string{
		".claude",
		".claude/agents",
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(filepath.Join(tmpDir, dir), 0o755); err != nil {
			t.Fatalf("Failed to create dir %s: %v", dir, err)
		}
	}

	files := map[string]string{
		"CLAUDE.md": `# Main Agent
Simple configuration.`,
		".claude/agents/coder.md": `---
name: coder
tools: Read, Write, nonexistent-tool-xyz
---
# Coder Agent`,
	}

	for name, content := range files {
		path := filepath.Join(tmpDir, name)
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("Failed to write file %s: %v", name, err)
		}
	}

	agentConfig, err := agent.Load("claude-code")
	if err != nil {
		t.Fatalf("Failed to load agent config: %v", err)
	}

	tree, err := analyzer.BuildTree(tmpDir, agentConfig)
	if err != nil {
		t.Fatalf("Failed to build tree: %v", err)
	}

	ctx := &AnalysisContext{
		Tree:        tree,
		AgentConfig: agentConfig,
		RootPath:    tmpDir,
	}

	rule := &MissingToolRule{}
	issues, err := rule.Run(ctx)
	if err != nil {
		t.Fatalf("Run() returned error: %v", err)
	}

	// Should have 1 issue for nonexistent-tool-xyz
	foundNonexistent := false
	for _, issue := range issues {
		if issue.Rule == "missing-tool" {
			if issue.Message == "Tool 'nonexistent-tool-xyz' not found (not a Claude built-in tool and not found on PATH)" {
				foundNonexistent = true
			}
		}
	}

	if !foundNonexistent {
		t.Error("Expected issue for 'nonexistent-tool-xyz', got none")
	}
}

func TestMissingToolRule_NoIssuesForValidTools(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "valid-tool-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dirs := []string{
		".claude",
		".claude/agents",
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(filepath.Join(tmpDir, dir), 0o755); err != nil {
			t.Fatalf("Failed to create dir %s: %v", dir, err)
		}
	}

	// Only use Claude built-in tools to ensure test passes on all systems
	files := map[string]string{
		"CLAUDE.md": `# Main Agent
Simple configuration.`,
		".claude/agents/coder.md": `---
name: coder
tools: Read, Write, Bash
---
# Coder Agent`,
	}

	for name, content := range files {
		path := filepath.Join(tmpDir, name)
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("Failed to write file %s: %v", name, err)
		}
	}

	agentConfig, err := agent.Load("claude-code")
	if err != nil {
		t.Fatalf("Failed to load agent config: %v", err)
	}

	tree, err := analyzer.BuildTree(tmpDir, agentConfig)
	if err != nil {
		t.Fatalf("Failed to build tree: %v", err)
	}

	ctx := &AnalysisContext{
		Tree:        tree,
		AgentConfig: agentConfig,
		RootPath:    tmpDir,
	}

	rule := &MissingToolRule{}
	issues, err := rule.Run(ctx)
	if err != nil {
		t.Fatalf("Run() returned error: %v", err)
	}

	if len(issues) > 0 {
		t.Errorf("Expected no issues for valid tools, got %d: %v", len(issues), issues)
	}
}
