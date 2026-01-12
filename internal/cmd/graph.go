package cmd

import (
	"fmt"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/pthm/cclint/internal/agent"
	"github.com/pthm/cclint/internal/analyzer"
	"github.com/pthm/cclint/internal/ui"
	"github.com/spf13/cobra"
)

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
  cclint graph /path/to/project`,
	Args: cobra.MaximumNArgs(1),
	RunE: runGraph,
}

func init() {
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

	// Check if interactive mode is available
	if !u.IsInteractive() {
		return fmt.Errorf("graph command requires an interactive terminal (TTY)")
	}

	// Start spinner while building tree
	spinner := u.StartSimpleSpinner(u.ErrWriter, "Building configuration tree...")

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

	// Stop spinner before launching TUI
	if spinner != nil {
		spinner.Stop()
	}

	// Create and run the graph model
	model := ui.NewGraphModel(tree, scopes, absPath)
	p := tea.NewProgram(model, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		return fmt.Errorf("error running graph viewer: %w", err)
	}

	return nil
}
