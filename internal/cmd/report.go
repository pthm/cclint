package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/pthm/cclint/internal/agent"
	"github.com/pthm/cclint/internal/analyzer"
	"github.com/spf13/cobra"
)

var reportCmd = &cobra.Command{
	Use:   "report [path]",
	Short: "Generate a detailed report of Claude Code configurations",
	Long: `Generate a comprehensive report of your Claude Code setup.

This includes:
  - Reference map
  - Token usage estimates
  - Quality metrics

Use 'cclint graph --print' to see the configuration tree.

Examples:
  cclint report .
  cclint report --format json . > report.json`,
	Args: cobra.MaximumNArgs(1),
	RunE: runReport,
}

func init() {
	RootCmd.AddCommand(reportCmd)
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

	// Get the global UI
	u := GetUI()

	// Start spinner for tree building
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

	// Stop spinner
	if spinner != nil {
		spinner.Stop()
	}

	// Print report header
	fmt.Println(u.Styles.Suggestion.Render("Claude Code Configuration Report"))
	fmt.Println(u.Styles.Suggestion.Render("================================"))
	fmt.Println()

	fmt.Printf("Agent: %s\n", agentConfig.Name)
	fmt.Printf("Root:  %s\n\n", absPath)

	// Print metrics
	fmt.Println(u.Styles.Warning.Render("Metrics:"))
	metrics := analyzer.ComputeMetrics(tree)
	fmt.Printf("  Total files:      %d\n", metrics.TotalFiles)
	fmt.Printf("  Total references: %d\n", metrics.TotalReferences)
	fmt.Printf("  Estimated tokens: %d\n", metrics.EstimatedTokens)
	fmt.Printf("  Total bytes:      %d\n", metrics.TotalBytes)
	fmt.Println()

	// Print reference summary
	fmt.Println(u.Styles.Warning.Render("References by Type:"))
	for refType, count := range metrics.ReferencesByType {
		fmt.Printf("  %s: %d\n", refType, count)
	}

	return nil
}
