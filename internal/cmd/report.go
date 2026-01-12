package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/fatih/color"
	"github.com/pthm-cable/cclint/internal/agent"
	"github.com/pthm-cable/cclint/internal/analyzer"
	"github.com/spf13/cobra"
)

var reportCmd = &cobra.Command{
	Use:   "report [path]",
	Short: "Generate a detailed report of Claude Code configurations",
	Long: `Generate a comprehensive report of your Claude Code setup.

This includes:
  - Configuration file tree
  - Reference map
  - Token usage estimates
  - Quality metrics

Examples:
  cclint report .
  cclint report --format json . > report.json`,
	Args: cobra.MaximumNArgs(1),
	RunE: runReport,
}

func init() {
	rootCmd.AddCommand(reportCmd)
}

func runReport(cmd *cobra.Command, args []string) error {
	path := "."
	if len(args) > 0 {
		path = args[0]
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}

	// Load agent configuration
	agentConfig, err := agent.Load(agentType)
	if err != nil {
		return fmt.Errorf("failed to load agent config: %w", err)
	}

	// Build reference tree
	tree, err := analyzer.BuildTree(absPath, agentConfig)
	if err != nil {
		return fmt.Errorf("failed to build reference tree: %w", err)
	}

	// Print report header
	color.Cyan("Claude Code Configuration Report\n")
	color.Cyan("================================\n\n")

	fmt.Printf("Agent: %s\n", agentConfig.Name)
	fmt.Printf("Root:  %s\n\n", absPath)

	// Print tree structure
	color.Yellow("Configuration Tree:\n")
	tree.PrintTree()
	fmt.Println()

	// Print metrics
	color.Yellow("Metrics:\n")
	metrics := analyzer.ComputeMetrics(tree)
	fmt.Printf("  Total files:      %d\n", metrics.TotalFiles)
	fmt.Printf("  Total references: %d\n", metrics.TotalReferences)
	fmt.Printf("  Estimated tokens: %d\n", metrics.EstimatedTokens)
	fmt.Printf("  Total bytes:      %d\n", metrics.TotalBytes)
	fmt.Println()

	// Print reference summary
	color.Yellow("References by Type:\n")
	for refType, count := range metrics.ReferencesByType {
		fmt.Printf("  %s: %d\n", refType, count)
	}

	return nil
}
