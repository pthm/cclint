package analyzer

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pthm/cclint/internal/agent"
	"github.com/pthm/cclint/internal/parser"
)

// RefType represents the type of reference
type RefType int

const (
	RefTypeFile RefType = iota
	RefTypeURL
	RefTypeTool
	RefTypeSubagent
	RefTypeSkill
	RefTypeMCPServer
	RefTypeUnknown
)

func (rt RefType) String() string {
	switch rt {
	case RefTypeFile:
		return "file"
	case RefTypeURL:
		return "url"
	case RefTypeTool:
		return "tool"
	case RefTypeSubagent:
		return "subagent"
	case RefTypeSkill:
		return "skill"
	case RefTypeMCPServer:
		return "mcp_server"
	default:
		return "unknown"
	}
}

// ParseRefType converts a string to RefType
func ParseRefType(s string) RefType {
	switch s {
	case "file":
		return RefTypeFile
	case "url":
		return RefTypeURL
	case "tool":
		return RefTypeTool
	case "subagent":
		return RefTypeSubagent
	case "skill":
		return RefTypeSkill
	case "mcp_server":
		return RefTypeMCPServer
	default:
		return RefTypeUnknown
	}
}

// Location represents a position in a file
type Location struct {
	File   string
	Line   int
	Column int
}

func (l Location) String() string {
	return fmt.Sprintf("%s:%d:%d", l.File, l.Line, l.Column)
}

// Reference represents a reference found in a configuration file
type Reference struct {
	Type     RefType
	Value    string
	Source   Location
	Priority int    // Based on surrounding markers
	Context  string // Surrounding text for context
	Resolved bool   // Whether the reference was resolved
	Target   string // Resolved path/URL
}

// ConfigNode represents a node in the configuration tree
type ConfigNode struct {
	Path       string
	Content    []byte
	Parsed     *parser.ParsedFile
	References []Reference
	Children   []*ConfigNode
	Parent     *ConfigNode
	Depth      int
}

// Tree represents the complete configuration tree
type Tree struct {
	Root     *ConfigNode
	RootPath string                   // Absolute path to project root
	Nodes    map[string]*ConfigNode // Path -> Node
}

// BuildTree builds a reference tree starting from the given path
func BuildTree(rootPath string, agentConfig *agent.Config) (*Tree, error) {
	absRoot, err := filepath.Abs(rootPath)
	if err != nil {
		return nil, err
	}

	tree := &Tree{
		RootPath: absRoot,
		Nodes:    make(map[string]*ConfigNode),
	}

	// Find entrypoints
	var entrypoints []string
	for _, pattern := range agentConfig.Entrypoints {
		matches, err := filepath.Glob(filepath.Join(rootPath, pattern))
		if err != nil {
			continue
		}
		entrypoints = append(entrypoints, matches...)
	}

	if len(entrypoints) == 0 {
		return nil, fmt.Errorf("no configuration files found")
	}

	// Create virtual root node
	tree.Root = &ConfigNode{
		Path:  rootPath,
		Depth: 0,
	}

	// Process each entrypoint
	for _, entry := range entrypoints {
		node, err := tree.processFile(entry, agentConfig, tree.Root, 1)
		if err != nil {
			continue
		}
		tree.Root.Children = append(tree.Root.Children, node)
	}

	return tree, nil
}

// processFile processes a single file and its references
func (t *Tree) processFile(path string, agentConfig *agent.Config, parent *ConfigNode, depth int) (*ConfigNode, error) {
	// Avoid cycles
	if _, exists := t.Nodes[path]; exists {
		return t.Nodes[path], nil
	}

	// Read and parse file
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	parsed, err := parser.Parse(path)
	if err != nil {
		// Continue with unparsed content
		parsed = &parser.ParsedFile{
			Path:    path,
			Content: content,
		}
	}

	node := &ConfigNode{
		Path:    path,
		Content: content,
		Parsed:  parsed,
		Parent:  parent,
		Depth:   depth,
	}

	t.Nodes[path] = node

	// Extract references
	node.References = extractReferences(content, path, agentConfig)

	// Process file references (don't go too deep)
	if depth < 5 {
		seenChildren := make(map[string]bool)
		for i, ref := range node.References {
			if ref.Type == RefTypeFile {
				resolvedPath := resolveFilePath(t.RootPath, path, ref.Value)
				node.References[i].Target = resolvedPath

				if _, err := os.Stat(resolvedPath); err == nil {
					node.References[i].Resolved = true

					// Avoid duplicate children from multiple references to the same file
					if seenChildren[resolvedPath] {
						continue
					}
					seenChildren[resolvedPath] = true

					// Process child file
					child, err := t.processFile(resolvedPath, agentConfig, node, depth+1)
					if err == nil {
						node.Children = append(node.Children, child)
					}
				}
			}
		}
	}

	return node, nil
}

// extractReferences extracts all references from content
func extractReferences(content []byte, path string, agentConfig *agent.Config) []Reference {
	var refs []Reference
	lines := strings.Split(string(content), "\n")

	for _, pattern := range agentConfig.ReferencePatterns {
		re := pattern.CompiledRegex()
		if re == nil {
			continue
		}

		refType := ParseRefType(pattern.Type)

		for lineNum, line := range lines {
			matches := re.FindAllStringSubmatchIndex(line, -1)
			for _, match := range matches {
				if len(match) >= 4 {
					value := line[match[2]:match[3]]

					// Calculate priority based on markers
					priority := calculatePriority(lines, lineNum, agentConfig)

					// Get context
					context := getContext(lines, lineNum, 2)

					refs = append(refs, Reference{
						Type:  refType,
						Value: value,
						Source: Location{
							File:   path,
							Line:   lineNum + 1,
							Column: match[2] + 1,
						},
						Priority: priority,
						Context:  context,
					})
				}
			}
		}
	}

	return refs
}

// calculatePriority calculates the priority of a reference based on nearby markers
func calculatePriority(lines []string, lineNum int, agentConfig *agent.Config) int {
	priority := 0

	// Check surrounding lines (5 before and 5 after)
	start := lineNum - 5
	if start < 0 {
		start = 0
	}
	end := lineNum + 5
	if end > len(lines) {
		end = len(lines)
	}

	context := strings.Join(lines[start:end], " ")
	contextUpper := strings.ToUpper(context)

	for _, marker := range agentConfig.Markers.HighPriority {
		if strings.Contains(contextUpper, strings.ToUpper(marker)) {
			priority += 3
		}
	}

	for _, marker := range agentConfig.Markers.MediumPriority {
		if strings.Contains(contextUpper, strings.ToUpper(marker)) {
			priority += 2
		}
	}

	for _, marker := range agentConfig.Markers.LowPriority {
		if strings.Contains(contextUpper, strings.ToUpper(marker)) {
			priority += 1
		}
	}

	return priority
}

// getContext returns surrounding lines for context
func getContext(lines []string, lineNum int, radius int) string {
	start := lineNum - radius
	if start < 0 {
		start = 0
	}
	end := lineNum + radius + 1
	if end > len(lines) {
		end = len(lines)
	}

	return strings.Join(lines[start:end], "\n")
}

// resolveFilePath resolves a relative file path
// Claude configs typically use project-root-relative paths, so we try that first
func resolveFilePath(rootPath, sourcePath, refPath string) string {
	// Remove @ prefix if present
	refPath = strings.TrimPrefix(refPath, "@")

	// Check if it's a project-root-relative path (starts with /)
	// e.g., `/CABLE.md` means rootPath/CABLE.md
	if strings.HasPrefix(refPath, "/") {
		// Try resolving relative to project root first
		rootRelative := filepath.Join(rootPath, refPath)
		if _, err := os.Stat(rootRelative); err == nil {
			return filepath.Clean(rootRelative)
		}

		// If not found at root, check if it's actually an absolute path that exists
		if filepath.IsAbs(refPath) {
			if _, err := os.Stat(refPath); err == nil {
				return refPath
			}
		}

		// Default to root-relative even if not found (for error reporting)
		return filepath.Clean(rootRelative)
	}

	// For relative paths (no leading /), try project root first since Claude configs
	// typically document paths relative to the project root
	rootRelative := filepath.Join(rootPath, refPath)
	if _, err := os.Stat(rootRelative); err == nil {
		return filepath.Clean(rootRelative)
	}

	// Fall back to resolving relative to source file's directory
	sourceDir := filepath.Dir(sourcePath)
	sourceRelative := filepath.Clean(filepath.Join(sourceDir, refPath))
	if _, err := os.Stat(sourceRelative); err == nil {
		return sourceRelative
	}

	// Neither exists - prefer project-root-relative for error reporting
	// since that's the expected convention
	return rootRelative
}

// NodeCount returns the total number of nodes in the tree
func (t *Tree) NodeCount() int {
	return len(t.Nodes)
}

// PrintTree prints the tree structure
func (t *Tree) PrintTree() {
	t.printNode(t.Root, "")
}

func (t *Tree) printNode(node *ConfigNode, prefix string) {
	if node.Path != "" {
		name := filepath.Base(node.Path)
		if node.Parent == nil || node.Parent.Path == "" {
			name = node.Path
		}
		fmt.Printf("%s%s\n", prefix, name)
	}

	for i, child := range node.Children {
		isLast := i == len(node.Children)-1
		childPrefix := prefix
		if node.Path != "" {
			if isLast {
				childPrefix += "  "
			} else {
				childPrefix += "│ "
			}
		}

		connector := "├─"
		if isLast {
			connector = "└─"
		}

		if node.Path != "" {
			fmt.Printf("%s%s ", prefix, connector)
			t.printNode(child, childPrefix)
		} else {
			t.printNode(child, prefix)
		}
	}
}

// AllReferences returns all references from all nodes
func (t *Tree) AllReferences() []Reference {
	var refs []Reference
	for _, node := range t.Nodes {
		refs = append(refs, node.References...)
	}
	return refs
}
