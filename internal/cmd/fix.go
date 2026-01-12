package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/fatih/color"
	"github.com/pthm-cable/cclint/internal/agent"
	"github.com/pthm-cable/cclint/internal/analyzer"
	"github.com/pthm-cable/cclint/internal/fixer"
	"github.com/pthm-cable/cclint/internal/rules"
	"github.com/spf13/cobra"
)

var (
	aiAssisted bool
	dryRun     bool
)

var fixCmd = &cobra.Command{
	Use:   "fix [path]",
	Short: "Auto-fix Claude Code configuration issues",
	Long: `Automatically fix issues in Claude Code configurations.

Examples:
  cclint fix .
  cclint fix --ai .
  cclint fix --ai --dry-run .`,
	Args: cobra.MaximumNArgs(1),
	RunE: runFix,
}

func init() {
	fixCmd.Flags().BoolVar(&aiAssisted, "ai", false, "Enable AI-assisted fixes using Claude API")
	fixCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show fixes without applying them")
	rootCmd.AddCommand(fixCmd)
}

func runFix(cmd *cobra.Command, args []string) error {
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

	// Run rules to find issues
	registry := rules.DefaultRegistry()
	var allIssues []rules.Issue

	ctx := &rules.AnalysisContext{
		Tree:        tree,
		AgentConfig: agentConfig,
		RootPath:    absPath,
	}

	for _, rule := range registry.Rules(false) {
		issues, err := rule.Run(ctx)
		if err != nil {
			continue // Skip rules that fail
		}
		allIssues = append(allIssues, issues...)
	}

	// Filter to fixable issues
	var fixableIssues []rules.Issue
	for _, issue := range allIssues {
		if issue.Fix != nil {
			fixableIssues = append(fixableIssues, issue)
		}
	}

	if len(fixableIssues) == 0 {
		color.Green("No fixable issues found!")
		return nil
	}

	fmt.Printf("Found %d fixable issues\n\n", len(fixableIssues))

	// Create fixer
	f := fixer.New(fixer.Options{
		DryRun:     dryRun,
		AIAssisted: aiAssisted,
	})

	// Apply fixes
	for _, issue := range fixableIssues {
		if err := f.ApplyFix(issue); err != nil {
			color.Red("Failed to fix %s: %v\n", issue.Rule, err)
		}
	}

	return nil
}
