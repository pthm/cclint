package analyzer

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/pthm/cclint/internal/agent"
)

// ScopeType represents the type of context scope
type ScopeType int

const (
	// ScopeTypeMain represents the main agent context
	ScopeTypeMain ScopeType = iota
	// ScopeTypeSubagent represents a subagent context
	ScopeTypeSubagent
	// ScopeTypeCommand represents a slash command context
	ScopeTypeCommand
	// ScopeTypeSkill represents a skill context
	ScopeTypeSkill
)

func (st ScopeType) String() string {
	switch st {
	case ScopeTypeMain:
		return "main"
	case ScopeTypeSubagent:
		return "subagent"
	case ScopeTypeCommand:
		return "command"
	case ScopeTypeSkill:
		return "skill"
	default:
		return "unknown"
	}
}

// ContextScope represents an isolated scope for analysis.
// Each scope contains a subset of files that form a coherent context,
// either the main agent configuration or a subagent's configuration.
type ContextScope struct {
	// Type indicates whether this is a main or subagent scope
	Type ScopeType

	// Name identifies the scope ("main" or the subagent name)
	Name string

	// Entrypoint is the root file path for this scope
	Entrypoint string

	// Nodes contains all ConfigNodes that belong to this scope
	Nodes []*ConfigNode

	// FilePaths is a convenience list of all file paths in this scope
	FilePaths []string

	// Children contains nested scopes (commands/skills for main, skills for subagents)
	Children []*ContextScope
}

// DiscoverScopes finds all context scopes in the tree.
// It identifies the main scope and any subagent scopes from:
// 1. RefTypeSubagent references in parsed files
// 2. Files in well-known paths like .claude/agents/
func (t *Tree) DiscoverScopes(agentConfig *agent.Config, rootPath string) ([]*ContextScope, error) {
	var scopes []*ContextScope

	// Collect subagent entrypoints from both sources
	subagentEntrypoints := make(map[string]string) // path -> name

	// 1. Find subagents from RefTypeSubagent references
	for _, node := range t.Nodes {
		for _, ref := range node.References {
			if ref.Type == RefTypeSubagent {
				// The value is the subagent name, try to find its config file
				subagentName := ref.Value
				possiblePaths := []string{
					filepath.Join(rootPath, ".claude", "agents", subagentName+".md"),
					filepath.Join(rootPath, ".claude", "agents", subagentName, "CLAUDE.md"),
					filepath.Join(rootPath, ".claude", "agents", subagentName, "instructions.md"),
				}
				for _, path := range possiblePaths {
					if _, err := os.Stat(path); err == nil {
						subagentEntrypoints[path] = subagentName
						break
					}
				}
			}
		}
	}

	// 2. Find subagents from well-known paths
	agentsDir := filepath.Join(rootPath, ".claude", "agents")
	if entries, err := os.ReadDir(agentsDir); err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				// Check for CLAUDE.md or instructions.md inside directory
				dirPath := filepath.Join(agentsDir, entry.Name())
				for _, filename := range []string{"CLAUDE.md", "instructions.md"} {
					path := filepath.Join(dirPath, filename)
					if _, err := os.Stat(path); err == nil {
						if _, exists := subagentEntrypoints[path]; !exists {
							subagentEntrypoints[path] = entry.Name()
						}
						break
					}
				}
			} else if strings.HasSuffix(entry.Name(), ".md") {
				// Direct .md file in agents directory
				path := filepath.Join(agentsDir, entry.Name())
				name := strings.TrimSuffix(entry.Name(), ".md")
				if _, exists := subagentEntrypoints[path]; !exists {
					subagentEntrypoints[path] = name
				}
			}
		}
	}

	// Create subagent scopes
	for entrypoint, name := range subagentEntrypoints {
		scope := &ContextScope{
			Type:       ScopeTypeSubagent,
			Name:       name,
			Entrypoint: entrypoint,
		}

		// If entrypoint is not in tree, process it to walk its references
		if _, exists := t.Nodes[entrypoint]; !exists {
			// Process the subagent file and its references, adding them to the tree
			_, _ = t.processFile(entrypoint, agentConfig, nil, 1)
		}

		// Now collect all reachable nodes from the entrypoint
		if _, exists := t.Nodes[entrypoint]; exists {
			scope.Nodes = t.collectReachableNodes(entrypoint)
			scope.FilePaths = make([]string, 0, len(scope.Nodes))
			for _, n := range scope.Nodes {
				scope.FilePaths = append(scope.FilePaths, n.Path)
			}
		}

		if len(scope.Nodes) > 0 {
			scopes = append(scopes, scope)
		}
	}

	// Create main scope by walking from main entrypoints
	mainScope := &ContextScope{
		Type:       ScopeTypeMain,
		Name:       "main",
		Entrypoint: rootPath,
	}

	// Collect all nodes reachable from main entrypoints (children of root)
	mainVisited := make(map[string]bool)
	for _, child := range t.Root.Children {
		for _, node := range t.collectReachableNodes(child.Path) {
			if !mainVisited[node.Path] {
				mainVisited[node.Path] = true
				mainScope.Nodes = append(mainScope.Nodes, node)
				mainScope.FilePaths = append(mainScope.FilePaths, node.Path)
			}
		}
	}

	// Discover commands and skills
	commands, _ := t.DiscoverCommands(agentConfig, rootPath)
	skills, _ := t.DiscoverSkills(agentConfig, rootPath)

	// Add commands and skills as children of main scope
	mainScope.Children = append(mainScope.Children, commands...)
	mainScope.Children = append(mainScope.Children, skills...)

	// Main scope first, then subagents
	if len(mainScope.Nodes) > 0 || len(mainScope.Children) > 0 {
		scopes = append([]*ContextScope{mainScope}, scopes...)
	}

	// For each subagent, find and attach declared skills from frontmatter
	for _, subagentScope := range scopes {
		if subagentScope.Type != ScopeTypeSubagent {
			continue
		}

		// Get the subagent's parsed file to read frontmatter
		if node, exists := t.Nodes[subagentScope.Entrypoint]; exists && node.Parsed != nil {
			declaredSkills := extractSkillsFromFrontmatter(node.Parsed.Frontmatter)
			for _, skillName := range declaredSkills {
				// Find matching skill scope and add as child
				for _, skill := range skills {
					if skill.Name == skillName {
						subagentScope.Children = append(subagentScope.Children, skill)
						break
					}
				}
			}
		}
	}

	return scopes, nil
}

// extractSkillsFromFrontmatter extracts skill names from frontmatter
// Handles both comma-separated string and list formats:
// - skills: skill1, skill2
// - skills: [skill1, skill2]
// - skills:
//   - skill1
//   - skill2
func extractSkillsFromFrontmatter(frontmatter map[string]interface{}) []string {
	if frontmatter == nil {
		return nil
	}

	skillsVal, ok := frontmatter["skills"]
	if !ok {
		return nil
	}

	var skills []string

	switch v := skillsVal.(type) {
	case string:
		// Comma-separated: "skill1, skill2"
		for _, s := range strings.Split(v, ",") {
			s = strings.TrimSpace(s)
			if s != "" {
				skills = append(skills, s)
			}
		}
	case []interface{}:
		// List: [skill1, skill2]
		for _, item := range v {
			if s, ok := item.(string); ok {
				skills = append(skills, s)
			}
		}
	case []string:
		skills = v
	}

	return skills
}

// collectReachableNodes returns all nodes reachable from the given entrypoint
// by following file references.
func (t *Tree) collectReachableNodes(entrypoint string) []*ConfigNode {
	visited := make(map[string]bool)
	var nodes []*ConfigNode

	var visit func(path string)
	visit = func(path string) {
		if visited[path] {
			return
		}
		visited[path] = true

		node, exists := t.Nodes[path]
		if !exists {
			return
		}

		nodes = append(nodes, node)

		// Follow file references
		for _, ref := range node.References {
			if ref.Type == RefTypeFile && ref.Resolved {
				visit(ref.Target)
			}
		}

		// Also visit children (which are file refs already resolved)
		for _, child := range node.Children {
			visit(child.Path)
		}
	}

	visit(entrypoint)
	return nodes
}

// AllPaths returns all file paths in the tree
func (t *Tree) AllPaths() []string {
	paths := make([]string, 0, len(t.Nodes))
	for path := range t.Nodes {
		paths = append(paths, path)
	}
	return paths
}

// DiscoverSkills finds all skills in .claude/skills/ and builds scopes for them.
// Each skill directory becomes its own context scope that can be analyzed independently.
func (t *Tree) DiscoverSkills(agentConfig *agent.Config, rootPath string) ([]*ContextScope, error) {
	var skills []*ContextScope

	skillsDir := filepath.Join(rootPath, ".claude", "skills")
	if _, err := os.Stat(skillsDir); os.IsNotExist(err) {
		return skills, nil // No skills directory, return empty
	}

	// Walk the skills directory looking for SKILL.md files or direct .md files
	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		var skillPath string
		var skillName string

		if entry.IsDir() {
			// Look for SKILL.md inside the directory
			skillPath = filepath.Join(skillsDir, entry.Name(), "SKILL.md")
			if _, err := os.Stat(skillPath); os.IsNotExist(err) {
				// Try lowercase
				skillPath = filepath.Join(skillsDir, entry.Name(), "skill.md")
				if _, err := os.Stat(skillPath); os.IsNotExist(err) {
					continue
				}
			}
			skillName = entry.Name()
		} else if strings.HasSuffix(entry.Name(), ".md") {
			// Direct .md file in skills directory
			skillPath = filepath.Join(skillsDir, entry.Name())
			skillName = strings.TrimSuffix(entry.Name(), ".md")
		} else {
			continue
		}

		// Build scope for this skill
		scope := &ContextScope{
			Type:       ScopeTypeSkill,
			Name:       skillName,
			Entrypoint: skillPath,
		}

		// Process the skill file to follow its references
		if _, exists := t.Nodes[skillPath]; !exists {
			_, _ = t.processFile(skillPath, agentConfig, nil, 1)
		}

		// Collect reachable nodes from skill entrypoint
		if _, exists := t.Nodes[skillPath]; exists {
			scope.Nodes = t.collectReachableNodes(skillPath)
			scope.FilePaths = make([]string, 0, len(scope.Nodes))
			for _, n := range scope.Nodes {
				scope.FilePaths = append(scope.FilePaths, n.Path)
			}
		}

		if len(scope.Nodes) > 0 {
			skills = append(skills, scope)
		}
	}

	return skills, nil
}

// DiscoverCommands finds all slash commands in .claude/commands/ and builds scopes for them.
// Each command file becomes its own context scope that can be analyzed independently.
func (t *Tree) DiscoverCommands(agentConfig *agent.Config, rootPath string) ([]*ContextScope, error) {
	var commands []*ContextScope

	commandsDir := filepath.Join(rootPath, ".claude", "commands")
	if _, err := os.Stat(commandsDir); os.IsNotExist(err) {
		return commands, nil // No commands directory, return empty
	}

	// Walk the commands directory
	err := filepath.WalkDir(commandsDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // Skip errors, continue walking
		}

		if d.IsDir() {
			return nil // Continue into directories
		}

		// Only process markdown files
		if !strings.HasSuffix(d.Name(), ".md") {
			return nil
		}

		// Derive command name from path
		// .claude/commands/commit.md -> "commit"
		// .claude/commands/git/push.md -> "git/push"
		relPath, _ := filepath.Rel(commandsDir, path)
		commandName := strings.TrimSuffix(relPath, ".md")
		commandName = strings.ReplaceAll(commandName, string(filepath.Separator), "/")

		// Build scope for this command
		scope := &ContextScope{
			Type:       ScopeTypeCommand,
			Name:       commandName,
			Entrypoint: path,
		}

		// Process the command file to follow its references
		if _, exists := t.Nodes[path]; !exists {
			_, _ = t.processFile(path, agentConfig, nil, 1)
		}

		// Collect reachable nodes from command entrypoint
		if _, exists := t.Nodes[path]; exists {
			scope.Nodes = t.collectReachableNodes(path)
			scope.FilePaths = make([]string, 0, len(scope.Nodes))
			for _, n := range scope.Nodes {
				scope.FilePaths = append(scope.FilePaths, n.Path)
			}
		}

		if len(scope.Nodes) > 0 {
			commands = append(commands, scope)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return commands, nil
}
