package cmd

import (
	"fmt"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/pthm/cclint/internal/agent"
	"github.com/pthm/cclint/internal/analyzer"
	"github.com/pthm/cclint/internal/ui"
	"github.com/spf13/cobra"
)

var graphPrint bool

var graphCmd = &cobra.Command{
	Use:   "graph [path]",
	Short: "Interactive visualization of configuration graph and scopes",
	Long: `Displays an interactive tree view of your Claude Code configuration.

Features:
  - Navigate the configuration tree with arrow keys or vim bindings
  - Expand/collapse nodes to explore the hierarchy
  - View scopes (main agent and subagents) separately
  - Toggle reference visibility
  - See file details and reference counts

Controls:
  ↑/k, ↓/j    Navigate up/down
  ←/h, →/l    Collapse/expand nodes
  Enter/Space Toggle expand/collapse
  r           Toggle reference display
  s           Toggle scope grouping
  q           Quit

Examples:
  cclint graph .
  cclint graph /path/to/project
  cclint graph --print .`,
	Args: cobra.MaximumNArgs(1),
	RunE: runGraph,
}

func init() {
	graphCmd.Flags().BoolVarP(&graphPrint, "print", "p", false, "Print tree to stdout instead of interactive mode")
	RootCmd.AddCommand(graphCmd)
}

func runGraph(cmd *cobra.Command, args []string) error {
	path := "."
	if len(args) > 0 {
		path = args[0]
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}

	// Get the global UI
	u := GetUI()

	// Check if interactive mode is available (unless --print is used)
	if !graphPrint && !u.IsInteractive() {
		return fmt.Errorf("graph command requires an interactive terminal (TTY). Use --print for non-interactive output")
	}

	// Start spinner while building tree (only in interactive mode)
	var spinner *ui.SimpleSpinner
	if !graphPrint {
		spinner = u.StartSimpleSpinner(u.ErrWriter, "Building configuration tree...")
	}

	// Load agent configuration
	agentConfig, err := agent.Load(agentType)
	if err != nil {
		if spinner != nil {
			spinner.Stop()
		}
		return fmt.Errorf("failed to load agent config: %w", err)
	}

	// Build reference tree
	tree, err := analyzer.BuildTree(absPath, agentConfig)
	if err != nil {
		if spinner != nil {
			spinner.Stop()
		}
		return fmt.Errorf("failed to build reference tree: %w", err)
	}

	// Discover scopes
	scopes, err := tree.DiscoverScopes(agentConfig, absPath)
	if err != nil {
		if spinner != nil {
			spinner.Stop()
		}
		return fmt.Errorf("failed to discover scopes: %w", err)
	}

	// Stop spinner before output
	if spinner != nil {
		spinner.Stop()
	}

	// Print mode - output tree to stdout
	if graphPrint {
		printScopedTree(scopes, tree, absPath)
		return nil
	}

	// Create and run the graph model
	model := ui.NewGraphModel(tree, scopes, absPath)
	p := tea.NewProgram(model, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		return fmt.Errorf("error running graph viewer: %w", err)
	}

	return nil
}

// printScopedTree prints the configuration tree with scope information to stdout
func printScopedTree(scopes []*analyzer.ContextScope, tree *analyzer.Tree, rootPath string) {
	for _, scope := range scopes {
		printScope(scope, tree, rootPath, "", true)
	}
}

func printScope(scope *analyzer.ContextScope, tree *analyzer.Tree, rootPath string, prefix string, isLast bool) {
	// Determine scope icon and label
	var icon, label string
	switch scope.Type {
	case analyzer.ScopeTypeMain:
		icon = "📦"
		label = fmt.Sprintf("[%s] %s", scope.Type.String(), scope.Name)
	case analyzer.ScopeTypeSubagent:
		icon = "🤖"
		label = fmt.Sprintf("[%s] %s", scope.Type.String(), scope.Name)
	case analyzer.ScopeTypeCommand:
		icon = "⚡"
		label = fmt.Sprintf("[%s] %s", scope.Type.String(), scope.Name)
	case analyzer.ScopeTypeSkill:
		icon = "🔧"
		label = fmt.Sprintf("[%s] %s", scope.Type.String(), scope.Name)
	}

	// Print scope header
	connector := "├─"
	if isLast {
		connector = "└─"
	}
	if prefix == "" {
		// Root scope
		fmt.Printf("%s %s", icon, label)
	} else {
		fmt.Printf("%s%s %s %s", prefix, connector, icon, label)
	}

	// Add entrypoint path for non-main scopes
	if scope.Entrypoint != "" && scope.Type != analyzer.ScopeTypeMain {
		relPath, _ := filepath.Rel(rootPath, scope.Entrypoint)
		fmt.Printf(" (%s)", relPath)
	} else if scope.Type == analyzer.ScopeTypeMain {
		fmt.Printf(" (.)")
	}
	fmt.Println()

	// Calculate child prefix
	childPrefix := prefix
	if prefix != "" {
		if isLast {
			childPrefix += "  "
		} else {
			childPrefix += "│ "
		}
	}

	// Print child scopes (commands/skills)
	for i, childScope := range scope.Children {
		isLastChild := i == len(scope.Children)-1 && len(scope.Nodes) == 0
		printScope(childScope, tree, rootPath, childPrefix, isLastChild)
	}

	// For non-main scopes with an entrypoint, print the file tree starting from entrypoint
	if scope.Type != analyzer.ScopeTypeMain && scope.Entrypoint != "" {
		if entryNode, exists := tree.Nodes[scope.Entrypoint]; exists {
			printFileNode(entryNode, rootPath, childPrefix, true)
		}
	}

	// For main scope, print direct file nodes
	if scope.Type == analyzer.ScopeTypeMain {
		for i, node := range scope.Nodes {
			isLastNode := i == len(scope.Nodes)-1
			printFileNode(node, rootPath, childPrefix, isLastNode)
		}
	}
}

func printFileNode(node *analyzer.ConfigNode, rootPath string, prefix string, isLast bool) {
	// Get relative path
	relPath, _ := filepath.Rel(rootPath, node.Path)

	// Determine icon based on file type
	icon := "📖"
	base := filepath.Base(node.Path)
	switch {
	case strings.HasPrefix(base, "CLAUDE"):
		icon = "📋"
	case strings.HasSuffix(base, ".json"):
		icon = "⚙️"
	case strings.HasSuffix(base, ".yaml") || strings.HasSuffix(base, ".yml"):
		icon = "📝"
	}

	// Print connector and file
	connector := "├─"
	if isLast {
		connector = "└─"
	}

	// Check if node has children
	hasChildren := len(node.Children) > 0
	expandIcon := "  "
	if hasChildren {
		expandIcon = "▼ "
	}

	fmt.Printf("%s%s %s%s %s\n", prefix, connector, expandIcon, icon, relPath)

	// Calculate child prefix
	childPrefix := prefix
	if isLast {
		childPrefix += "  "
	} else {
		childPrefix += "│ "
	}

	// Print children
	for i, child := range node.Children {
		isLastChild := i == len(node.Children)-1
		printFileNode(child, rootPath, childPrefix, isLastChild)
	}
}
