package cmd

import (
	"github.com/spf13/cobra"
)

var (
	// Global flags
	verbose   bool
	format    string
	agentType string
)

var rootCmd = &cobra.Command{
	Use:   "cclint",
	Short: "A linter for Claude Code configurations",
	Long: `cclint analyzes Claude Code configurations and related files
to identify issues, suggest improvements, and ensure best practices.

It builds a reference tree of your agent configurations, analyzes
documentation quality, and checks for common problems like broken
references, circular dependencies, and unclear instructions.`,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")
	rootCmd.PersistentFlags().StringVarP(&format, "format", "f", "terminal", "Output format (terminal, json)")
	rootCmd.PersistentFlags().StringVarP(&agentType, "agent", "a", "claude-code", "Agent type to lint for")
}
