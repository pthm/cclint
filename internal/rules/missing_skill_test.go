package rules

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pthm/cclint/internal/agent"
	"github.com/pthm/cclint/internal/analyzer"
)

func TestMissingSkillRule_Name(t *testing.T) {
	r := &MissingSkillRule{}
	if r.Name() != "missing-skill" {
		t.Errorf("Name() = %q, want %q", r.Name(), "missing-skill")
	}
}

func TestMissingSkillRule_Run_MissingSkill(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "missing-skill-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dirs := []string{
		".claude",
		".claude/agents",
		".claude/skills",
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(filepath.Join(tmpDir, dir), 0o755); err != nil {
			t.Fatalf("Failed to create dir %s: %v", dir, err)
		}
	}

	files := map[string]string{
		"CLAUDE.md": `# Main Agent
Simple configuration.`,
		".claude/skills/existing-skill.md": `---
name: existing-skill
---
# Existing Skill`,
		".claude/agents/coder.md": `---
name: coder
skills: existing-skill, nonexistent-skill
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

	rule := &MissingSkillRule{}
	issues, err := rule.Run(ctx)
	if err != nil {
		t.Fatalf("Run() returned error: %v", err)
	}

	// Should have 1 issue for nonexistent-skill
	foundMissing := false
	for _, issue := range issues {
		if issue.Rule == "missing-skill" {
			if issue.Message == "Skill 'nonexistent-skill' not found in .claude/skills/" {
				foundMissing = true
			}
		}
	}

	if !foundMissing {
		t.Error("Expected issue for 'nonexistent-skill', got none")
	}
}

func TestMissingSkillRule_Run_AllSkillsExist(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "valid-skill-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dirs := []string{
		".claude",
		".claude/agents",
		".claude/skills",
		".claude/skills/ck-search",
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(filepath.Join(tmpDir, dir), 0o755); err != nil {
			t.Fatalf("Failed to create dir %s: %v", dir, err)
		}
	}

	files := map[string]string{
		"CLAUDE.md": `# Main Agent
Simple configuration.`,
		".claude/skills/ck-search/SKILL.md": `---
name: ck-search
---
# CK Search Skill`,
		".claude/skills/linear.md": `---
name: linear
---
# Linear Skill`,
		".claude/agents/coder.md": `---
name: coder
skills: ck-search, linear
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

	rule := &MissingSkillRule{}
	issues, err := rule.Run(ctx)
	if err != nil {
		t.Fatalf("Run() returned error: %v", err)
	}

	if len(issues) > 0 {
		t.Errorf("Expected no issues for valid skills, got %d: %v", len(issues), issues)
	}
}

func TestMissingSkillRule_Run_NoSkillsDeclared(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "no-skill-test-*")
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
---
# Coder Agent without skills`,
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

	rule := &MissingSkillRule{}
	issues, err := rule.Run(ctx)
	if err != nil {
		t.Fatalf("Run() returned error: %v", err)
	}

	if len(issues) > 0 {
		t.Errorf("Expected no issues when no skills declared, got %d", len(issues))
	}
}
