package analyzer

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/pthm-cable/cclint/internal/agent"
)

// ScopeType represents the type of context scope
type ScopeType int

const (
	// ScopeTypeMain represents the main agent context
	ScopeTypeMain ScopeType = iota
	// ScopeTypeSubagent represents a subagent context
	ScopeTypeSubagent
)

func (st ScopeType) String() string {
	switch st {
	case ScopeTypeMain:
		return "main"
	case ScopeTypeSubagent:
		return "subagent"
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

	// Main scope first, then subagents
	if len(mainScope.Nodes) > 0 {
		scopes = append([]*ContextScope{mainScope}, scopes...)
	}

	return scopes, nil
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
