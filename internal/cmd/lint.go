package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/fatih/color"
	"github.com/pthm-cable/cclint/internal/agent"
	"github.com/pthm-cable/cclint/internal/analyzer"
	"github.com/pthm-cable/cclint/internal/reporter"
	"github.com/pthm-cable/cclint/internal/rules"
	"github.com/spf13/cobra"
)

var (
	deep    bool
	offline bool
)

var lintCmd = &cobra.Command{
	Use:   "lint [path]",
	Short: "Lint Claude Code configurations",
	Long: `Analyze Claude Code configurations for issues and improvements.

Examples:
  cclint lint .
  cclint lint --deep .
  cclint lint --format json . > report.json`,
	Args:         cobra.MaximumNArgs(1),
	RunE:         runLint,
	SilenceUsage: true,
}

func init() {
	lintCmd.Flags().BoolVar(&deep, "deep", false, "Enable deep analysis using Claude API")
	lintCmd.Flags().BoolVar(&offline, "offline", false, "Run in offline mode (heuristics only)")
	rootCmd.AddCommand(lintCmd)
}

func runLint(cmd *cobra.Command, args []string) error {
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

	if verbose {
		fmt.Printf("Linting with agent: %s\n", agentConfig.Name)
		fmt.Printf("Path: %s\n\n", absPath)
	}

	// Build reference tree
	tree, err := analyzer.BuildTree(absPath, agentConfig)
	if err != nil {
		return fmt.Errorf("failed to build reference tree: %w", err)
	}

	if verbose {
		fmt.Printf("Found %d config files\n", tree.NodeCount())
	}

	// Run rules
	registry := rules.DefaultRegistry()
	var allIssues []rules.Issue

	ctx := &rules.AnalysisContext{
		Tree:        tree,
		AgentConfig: agentConfig,
		RootPath:    absPath,
	}

	// Include AI rules only when --deep is set and not offline
	includeAI := deep && !offline
	for _, rule := range registry.Rules(includeAI) {
		issues, err := rule.Run(ctx)
		if err != nil {
			color.Yellow("Warning: rule %s failed: %v\n", rule.Name(), err)
			continue
		}
		allIssues = append(allIssues, issues...)
	}

	// Report results
	var rep reporter.Reporter
	switch format {
	case "json":
		rep = reporter.NewJSONReporter(os.Stdout)
	default:
		rep = reporter.NewTerminalReporter(os.Stdout)
	}

	return rep.Report(allIssues)
}
