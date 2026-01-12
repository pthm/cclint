package analyzer

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pthm/cclint/internal/agent"
)

func TestScopeTypeString(t *testing.T) {
	tests := []struct {
		scopeType ScopeType
		expected  string
	}{
		{ScopeTypeMain, "main"},
		{ScopeTypeSubagent, "subagent"},
		{ScopeTypeCommand, "command"},
		{ScopeTypeSkill, "skill"},
		{ScopeType(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.scopeType.String(); got != tt.expected {
				t.Errorf("ScopeType.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestDiscoverScopes(t *testing.T) {
	// Create a temporary directory structure for testing
	tmpDir, err := os.MkdirTemp("", "scope-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create directory structure
	dirs := []string{
		".claude",
		".claude/agents",
		".claude/instructions",
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(filepath.Join(tmpDir, dir), 0o755); err != nil {
			t.Fatalf("Failed to create dir %s: %v", dir, err)
		}
	}

	// Create main config files
	files := map[string]string{
		"CLAUDE.md": `# Main Agent
This is the main agent configuration.
See @.claude/instructions/style.md for style guide.`,

		".claude/CLAUDE.md": `# Claude Config
Additional configuration.`,

		".claude/instructions/style.md": `# Style Guide
Use consistent formatting.`,

		".claude/agents/reviewer.md": `# Reviewer Agent
This agent reviews code.`,

		".claude/agents/coder.md": `# Coder Agent
This agent writes code.`,
	}

	for name, content := range files {
		path := filepath.Join(tmpDir, name)
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("Failed to write file %s: %v", name, err)
		}
	}

	// Load agent config
	agentConfig, err := agent.Load("claude-code")
	if err != nil {
		t.Fatalf("Failed to load agent config: %v", err)
	}

	// Build tree
	tree, err := BuildTree(tmpDir, agentConfig)
	if err != nil {
		t.Fatalf("Failed to build tree: %v", err)
	}

	// Discover scopes
	scopes, err := tree.DiscoverScopes(agentConfig, tmpDir)
	if err != nil {
		t.Fatalf("DiscoverScopes failed: %v", err)
	}

	// Should have main scope + 2 subagent scopes
	if len(scopes) < 1 {
		t.Fatalf("Expected at least 1 scope, got %d", len(scopes))
	}

	// First scope should be main
	mainScope := scopes[0]
	if mainScope.Type != ScopeTypeMain {
		t.Errorf("First scope should be main, got %s", mainScope.Type)
	}
	if mainScope.Name != "main" {
		t.Errorf("Main scope name should be 'main', got %s", mainScope.Name)
	}

	// Check subagent scopes
	subagentNames := make(map[string]bool)
	for _, scope := range scopes {
		if scope.Type == ScopeTypeSubagent {
			subagentNames[scope.Name] = true
		}
	}

	if !subagentNames["reviewer"] {
		t.Error("Expected 'reviewer' subagent scope")
	}
	if !subagentNames["coder"] {
		t.Error("Expected 'coder' subagent scope")
	}
}

func TestDiscoverScopesWithSubagentReference(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "scope-ref-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create structure with subagent reference in main config
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
Use subagent_type: "tester" for testing tasks.`,

		".claude/agents/tester.md": `# Tester Agent
This agent runs tests.`,
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

	tree, err := BuildTree(tmpDir, agentConfig)
	if err != nil {
		t.Fatalf("Failed to build tree: %v", err)
	}

	scopes, err := tree.DiscoverScopes(agentConfig, tmpDir)
	if err != nil {
		t.Fatalf("DiscoverScopes failed: %v", err)
	}

	// Should find tester subagent from both reference and file existence
	hasMainScope := false
	hasTesterSubagent := false

	for _, scope := range scopes {
		if scope.Type == ScopeTypeMain {
			hasMainScope = true
		}
		if scope.Type == ScopeTypeSubagent && scope.Name == "tester" {
			hasTesterSubagent = true
		}
	}

	if !hasMainScope {
		t.Error("Expected main scope")
	}
	if !hasTesterSubagent {
		t.Error("Expected 'tester' subagent scope")
	}
}

func TestDiscoverScopesNoSubagents(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "scope-nosub-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	if err := os.MkdirAll(filepath.Join(tmpDir, ".claude"), 0o755); err != nil {
		t.Fatalf("Failed to create .claude dir: %v", err)
	}

	files := map[string]string{
		"CLAUDE.md": `# Main Agent
Simple configuration.`,
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

	tree, err := BuildTree(tmpDir, agentConfig)
	if err != nil {
		t.Fatalf("Failed to build tree: %v", err)
	}

	scopes, err := tree.DiscoverScopes(agentConfig, tmpDir)
	if err != nil {
		t.Fatalf("DiscoverScopes failed: %v", err)
	}

	// Should have only main scope
	if len(scopes) != 1 {
		t.Fatalf("Expected 1 scope, got %d", len(scopes))
	}

	if scopes[0].Type != ScopeTypeMain {
		t.Errorf("Expected main scope, got %s", scopes[0].Type)
	}
}

func TestCollectReachableNodes(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "reachable-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	if err := os.MkdirAll(filepath.Join(tmpDir, ".claude/instructions"), 0o755); err != nil {
		t.Fatalf("Failed to create dirs: %v", err)
	}

	// Create files with references
	// Note: @ references are resolved relative to the source file's directory
	files := map[string]string{
		"CLAUDE.md": `# Main
See @.claude/instructions/a.md`,
		".claude/instructions/a.md": `# A
See @./b.md`,
		".claude/instructions/b.md": `# B
Leaf node.`,
		".claude/instructions/isolated.md": `# Isolated
Not referenced by anyone.`,
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

	tree, err := BuildTree(tmpDir, agentConfig)
	if err != nil {
		t.Fatalf("Failed to build tree: %v", err)
	}

	// Collect reachable from CLAUDE.md
	mainPath := filepath.Join(tmpDir, "CLAUDE.md")
	reachable := tree.collectReachableNodes(mainPath)

	// Should include CLAUDE.md, a.md, b.md but NOT isolated.md
	reachablePaths := make(map[string]bool)
	for _, node := range reachable {
		reachablePaths[filepath.Base(node.Path)] = true
	}

	if !reachablePaths["CLAUDE.md"] {
		t.Error("Expected CLAUDE.md in reachable nodes")
	}
	if !reachablePaths["a.md"] {
		t.Error("Expected a.md in reachable nodes")
	}
	if !reachablePaths["b.md"] {
		t.Error("Expected b.md in reachable nodes")
	}
}

func TestDiscoverScopesSharedFiles(t *testing.T) {
	// Test that files referenced by both main and subagent appear in both scopes
	tmpDir, err := os.MkdirTemp("", "scope-shared-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dirs := []string{
		".claude",
		".claude/agents",
		".claude/instructions",
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(filepath.Join(tmpDir, dir), 0o755); err != nil {
			t.Fatalf("Failed to create dir %s: %v", dir, err)
		}
	}

	files := map[string]string{
		"CLAUDE.md": `# Main Agent
See @.claude/instructions/code-style.md for style guide.`,

		".claude/instructions/code-style.md": `# Code Style Guide
Use consistent formatting.`,

		".claude/agents/reviewer.md": `# Reviewer Agent
See @../instructions/code-style.md for style guide.`,
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

	tree, err := BuildTree(tmpDir, agentConfig)
	if err != nil {
		t.Fatalf("Failed to build tree: %v", err)
	}

	scopes, err := tree.DiscoverScopes(agentConfig, tmpDir)
	if err != nil {
		t.Fatalf("DiscoverScopes failed: %v", err)
	}

	// Find main and reviewer scopes
	var mainScope, reviewerScope *ContextScope
	for _, scope := range scopes {
		if scope.Type == ScopeTypeMain {
			mainScope = scope
		}
		if scope.Type == ScopeTypeSubagent && scope.Name == "reviewer" {
			reviewerScope = scope
		}
	}

	if mainScope == nil {
		t.Fatal("Expected main scope")
	}
	if reviewerScope == nil {
		t.Fatal("Expected reviewer subagent scope")
	}

	// Check that code-style.md appears in main scope
	mainHasStyle := false
	for _, path := range mainScope.FilePaths {
		if filepath.Base(path) == "code-style.md" {
			mainHasStyle = true
			break
		}
	}
	if !mainHasStyle {
		t.Error("Expected code-style.md in main scope")
	}

	// Check that code-style.md appears in reviewer scope
	reviewerHasStyle := false
	for _, path := range reviewerScope.FilePaths {
		if filepath.Base(path) == "code-style.md" {
			reviewerHasStyle = true
			break
		}
	}
	if !reviewerHasStyle {
		t.Error("Expected code-style.md in reviewer scope")
	}
}

func TestAllPaths(t *testing.T) {
	tree := &Tree{
		Nodes: map[string]*ConfigNode{
			"/a": {Path: "/a"},
			"/b": {Path: "/b"},
			"/c": {Path: "/c"},
		},
	}

	paths := tree.AllPaths()
	if len(paths) != 3 {
		t.Errorf("Expected 3 paths, got %d", len(paths))
	}

	pathSet := make(map[string]bool)
	for _, p := range paths {
		pathSet[p] = true
	}

	for _, expected := range []string{"/a", "/b", "/c"} {
		if !pathSet[expected] {
			t.Errorf("Expected path %s in AllPaths()", expected)
		}
	}
}

func TestDiscoverCommands(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "commands-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create directory structure with commands
	dirs := []string{
		".claude",
		".claude/commands",
		".claude/commands/git",
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(filepath.Join(tmpDir, dir), 0o755); err != nil {
			t.Fatalf("Failed to create dir %s: %v", dir, err)
		}
	}

	files := map[string]string{
		"CLAUDE.md": `# Main Agent
Simple configuration.`,
		".claude/commands/deploy.md": `# Deploy Command
Deploys the application.`,
		".claude/commands/git/push.md": `# Git Push Command
Pushes changes to remote.`,
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

	tree, err := BuildTree(tmpDir, agentConfig)
	if err != nil {
		t.Fatalf("Failed to build tree: %v", err)
	}

	// Discover scopes (commands are nested under main scope)
	scopes, err := tree.DiscoverScopes(agentConfig, tmpDir)
	if err != nil {
		t.Fatalf("DiscoverScopes failed: %v", err)
	}

	// Find main scope
	var mainScope *ContextScope
	for _, scope := range scopes {
		if scope.Type == ScopeTypeMain {
			mainScope = scope
			break
		}
	}

	if mainScope == nil {
		t.Fatal("Expected main scope")
	}

	// Find command scopes in main scope's children
	commandNames := make(map[string]bool)
	for _, child := range mainScope.Children {
		if child.Type == ScopeTypeCommand {
			commandNames[child.Name] = true
		}
	}

	// Should have both commands
	if !commandNames["deploy"] {
		t.Error("Expected 'deploy' command scope in main.Children")
	}
	if !commandNames["git/push"] {
		t.Error("Expected 'git/push' command scope in main.Children")
	}

	// Check that command scopes have correct entrypoints
	for _, child := range mainScope.Children {
		if child.Type == ScopeTypeCommand {
			if child.Name == "deploy" && !filepath.IsAbs(child.Entrypoint) {
				t.Errorf("Expected absolute entrypoint for deploy, got %s", child.Entrypoint)
			}
			if child.Name == "git/push" && !filepath.IsAbs(child.Entrypoint) {
				t.Errorf("Expected absolute entrypoint for git/push, got %s", child.Entrypoint)
			}
		}
	}
}

func TestDiscoverCommandsEmpty(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "commands-empty-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create structure without commands directory
	if err := os.MkdirAll(filepath.Join(tmpDir, ".claude"), 0o755); err != nil {
		t.Fatalf("Failed to create .claude dir: %v", err)
	}

	files := map[string]string{
		"CLAUDE.md": `# Main Agent
Simple configuration.`,
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

	tree, err := BuildTree(tmpDir, agentConfig)
	if err != nil {
		t.Fatalf("Failed to build tree: %v", err)
	}

	// Discover scopes
	scopes, err := tree.DiscoverScopes(agentConfig, tmpDir)
	if err != nil {
		t.Fatalf("DiscoverScopes failed: %v", err)
	}

	// Find main scope and verify no command children
	var mainScope *ContextScope
	for _, scope := range scopes {
		if scope.Type == ScopeTypeMain {
			mainScope = scope
			break
		}
	}

	if mainScope != nil {
		for _, child := range mainScope.Children {
			if child.Type == ScopeTypeCommand {
				t.Errorf("Expected no command scopes, got %s", child.Name)
			}
		}
	}
}

func TestDiscoverSkills(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "skills-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create directory structure with skills
	dirs := []string{
		".claude",
		".claude/skills",
		".claude/skills/ck-search",
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
		".claude/skills/ck-search/SKILL.md": `---
name: ck-search
description: Semantic code search
---
# CK Search Skill
Use this skill to search code semantically.`,
		".claude/skills/linear.md": `---
name: linear
description: Linear integration
---
# Linear Skill`,
		".claude/agents/reviewer.md": `---
name: reviewer
skills: ck-search, linear
---
# Reviewer Agent`,
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

	tree, err := BuildTree(tmpDir, agentConfig)
	if err != nil {
		t.Fatalf("Failed to build tree: %v", err)
	}

	scopes, err := tree.DiscoverScopes(agentConfig, tmpDir)
	if err != nil {
		t.Fatalf("DiscoverScopes failed: %v", err)
	}

	// Find main scope
	var mainScope *ContextScope
	var reviewerScope *ContextScope
	for _, scope := range scopes {
		if scope.Type == ScopeTypeMain {
			mainScope = scope
		}
		if scope.Type == ScopeTypeSubagent && scope.Name == "reviewer" {
			reviewerScope = scope
		}
	}

	if mainScope == nil {
		t.Fatal("Expected main scope")
	}

	// Check skills are in main scope's children
	skillNames := make(map[string]bool)
	for _, child := range mainScope.Children {
		if child.Type == ScopeTypeSkill {
			skillNames[child.Name] = true
		}
	}

	if !skillNames["ck-search"] {
		t.Error("Expected 'ck-search' skill in main.Children")
	}
	if !skillNames["linear"] {
		t.Error("Expected 'linear' skill in main.Children")
	}

	// Check reviewer subagent has its declared skills
	if reviewerScope == nil {
		t.Fatal("Expected reviewer subagent scope")
	}

	reviewerSkills := make(map[string]bool)
	for _, child := range reviewerScope.Children {
		if child.Type == ScopeTypeSkill {
			reviewerSkills[child.Name] = true
		}
	}

	if !reviewerSkills["ck-search"] {
		t.Error("Expected 'ck-search' skill in reviewer.Children")
	}
	if !reviewerSkills["linear"] {
		t.Error("Expected 'linear' skill in reviewer.Children")
	}
}

func TestExtractToolsFromFrontmatter(t *testing.T) {
	tests := []struct {
		name        string
		frontmatter map[string]interface{}
		expected    []string
	}{
		{
			name:        "nil frontmatter",
			frontmatter: nil,
			expected:    nil,
		},
		{
			name:        "no tools key",
			frontmatter: map[string]interface{}{"name": "test"},
			expected:    nil,
		},
		{
			name:        "comma-separated string",
			frontmatter: map[string]interface{}{"tools": "rg, jq, git"},
			expected:    []string{"rg", "jq", "git"},
		},
		{
			name:        "interface slice",
			frontmatter: map[string]interface{}{"tools": []interface{}{"Read", "Write", "Bash"}},
			expected:    []string{"Read", "Write", "Bash"},
		},
		{
			name:        "string slice",
			frontmatter: map[string]interface{}{"tools": []string{"npm", "node"}},
			expected:    []string{"npm", "node"},
		},
		{
			name:        "empty string",
			frontmatter: map[string]interface{}{"tools": ""},
			expected:    nil,
		},
		{
			name:        "whitespace handling",
			frontmatter: map[string]interface{}{"tools": "  rg  ,  jq  "},
			expected:    []string{"rg", "jq"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractToolsFromFrontmatter(tt.frontmatter)

			if len(got) != len(tt.expected) {
				t.Errorf("extractToolsFromFrontmatter() returned %d tools, want %d", len(got), len(tt.expected))
				return
			}

			for i, tool := range got {
				if tool != tt.expected[i] {
					t.Errorf("extractToolsFromFrontmatter()[%d] = %q, want %q", i, tool, tt.expected[i])
				}
			}
		})
	}
}

func TestDiscoverScopesWithDeclaredTools(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "tools-test-*")
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
tools: Read, Write, Bash, rg, jq
skills: ck-search
---
# Coder Agent
Uses tools for coding tasks.`,
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

	tree, err := BuildTree(tmpDir, agentConfig)
	if err != nil {
		t.Fatalf("Failed to build tree: %v", err)
	}

	scopes, err := tree.DiscoverScopes(agentConfig, tmpDir)
	if err != nil {
		t.Fatalf("DiscoverScopes failed: %v", err)
	}

	// Find coder subagent scope
	var coderScope *ContextScope
	for _, scope := range scopes {
		if scope.Type == ScopeTypeSubagent && scope.Name == "coder" {
			coderScope = scope
			break
		}
	}

	if coderScope == nil {
		t.Fatal("Expected coder subagent scope")
	}

	// Check DeclaredTools
	expectedTools := []string{"Read", "Write", "Bash", "rg", "jq"}
	if len(coderScope.DeclaredTools) != len(expectedTools) {
		t.Errorf("DeclaredTools has %d items, want %d", len(coderScope.DeclaredTools), len(expectedTools))
	}

	for i, tool := range expectedTools {
		if i >= len(coderScope.DeclaredTools) {
			break
		}
		if coderScope.DeclaredTools[i] != tool {
			t.Errorf("DeclaredTools[%d] = %q, want %q", i, coderScope.DeclaredTools[i], tool)
		}
	}

	// Check DeclaredSkills
	if len(coderScope.DeclaredSkills) != 1 || coderScope.DeclaredSkills[0] != "ck-search" {
		t.Errorf("DeclaredSkills = %v, want [ck-search]", coderScope.DeclaredSkills)
	}
}
