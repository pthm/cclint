package cmd

import (
	"os"

	"github.com/pthm/cclint/internal/ui"
	"github.com/spf13/cobra"
)

var (
	// Global flags
	verbose   bool
	format    string
	agentType string

	// Global UI instance
	globalUI *ui.UI
)

var RootCmd = &cobra.Command{
	Use:   "cclint",
	Short: "A linter for Claude Code configurations",
	Long: `cclint analyzes Claude Code configurations and related files
to identify issues, suggest improvements, and ensure best practices.

It builds a reference tree of your agent configurations, analyzes
documentation quality, and checks for common problems like broken
references, circular dependencies, and unclear instructions.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Initialize global UI with TTY detection
		globalUI = ui.New(os.Stdout, os.Stderr, format)
	},
}

func init() {
	RootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")
	RootCmd.PersistentFlags().StringVarP(&format, "format", "f", "terminal", "Output format (terminal, json)")
	RootCmd.PersistentFlags().StringVarP(&agentType, "agent", "a", "claude-code", "Agent type to lint for")
}

// GetUI returns the global UI instance for use by subcommands
func GetUI() *ui.UI {
	return globalUI
}
