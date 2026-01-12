package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pthm-cable/cclint/internal/agent"
	"github.com/pthm-cable/cclint/internal/analyzer"
	"github.com/pthm-cable/cclint/internal/reporter"
	"github.com/pthm-cable/cclint/internal/rules"
	"github.com/pthm-cable/cclint/internal/ui"
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
	RootCmd.AddCommand(lintCmd)
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

	// Get the global UI
	u := GetUI()

	// Start progress tracking if in interactive mode
	progress := u.StartProgress()
	defer func() {
		if progress != nil {
			progress.Done(nil)
		}
	}()

	// Stage 1: Load agent configuration
	if progress != nil {
		progress.SetStage(ui.StageLoadConfig)
	}

	agentConfig, err := agent.Load(agentType)
	if err != nil {
		return fmt.Errorf("failed to load agent config: %w", err)
	}

	if verbose {
		fmt.Printf("Linting with agent: %s\n", agentConfig.Name)
		fmt.Printf("Path: %s\n\n", absPath)
	}

	// Stage 2: Build reference tree
	if progress != nil {
		progress.SetStage(ui.StageBuildTree)
	}

	tree, err := analyzer.BuildTree(absPath, agentConfig)
	if err != nil {
		return fmt.Errorf("failed to build reference tree: %w", err)
	}

	if verbose {
		fmt.Printf("Found %d config files\n", tree.NodeCount())
	}

	// Stage 3: Run rules
	if progress != nil {
		progress.SetStage(ui.StageRunRules)
	}

	registry := rules.DefaultRegistry()
	var allIssues []rules.Issue

	ctx := &rules.AnalysisContext{
		Tree:        tree,
		AgentConfig: agentConfig,
		RootPath:    absPath,
	}

	// Include AI rules only when --deep is set and not offline
	includeAI := deep && !offline
	ruleList := registry.Rules(includeAI)

	if progress != nil {
		progress.SetRuleCount(len(ruleList))
	}

	for _, rule := range ruleList {
		if progress != nil {
			progress.RuleStart(rule.Name())
		}

		issues, err := rule.Run(ctx)
		if err != nil {
			// Use styled warning output
			fmt.Fprintln(os.Stderr, u.Styles.Warning.Render(
				fmt.Sprintf("%s Warning: rule %s failed: %v", u.Styles.IconWarning, rule.Name(), err),
			))
			if progress != nil {
				progress.RuleDone()
			}
			continue
		}
		allIssues = append(allIssues, issues...)

		if progress != nil {
			progress.RuleDone()
		}
	}

	// Stop progress before reporting
	if progress != nil {
		progress.Done(nil)
		progress = nil // Prevent double-done in defer
	}

	// Stage 4: Report results
	var rep reporter.Reporter
	switch format {
	case "json":
		rep = reporter.NewJSONReporter(os.Stdout)
	default:
		rep = reporter.NewTerminalReporter(os.Stdout, u)
	}

	return rep.Report(allIssues)
}
