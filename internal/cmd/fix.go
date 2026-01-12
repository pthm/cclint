package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/pthm/cclint/internal/agent"
	"github.com/pthm/cclint/internal/analyzer"
	"github.com/pthm/cclint/internal/fixer"
	"github.com/pthm/cclint/internal/rules"
	"github.com/pthm/cclint/internal/ui"
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
	RootCmd.AddCommand(fixCmd)
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

	// Get the global UI
	u := GetUI()

	// Start progress tracking
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

	// Stage 2: Build reference tree
	if progress != nil {
		progress.SetStage(ui.StageBuildTree)
	}

	tree, err := analyzer.BuildTree(absPath, agentConfig)
	if err != nil {
		return fmt.Errorf("failed to build reference tree: %w", err)
	}

	// Stage 3: Run rules to find issues
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

	ruleList := registry.Rules(false)
	if progress != nil {
		progress.SetRuleCount(len(ruleList))
	}

	for _, rule := range ruleList {
		if progress != nil {
			progress.RuleStart(rule.Name())
		}

		issues, err := rule.Run(ctx)
		if err != nil {
			if progress != nil {
				progress.RuleDone()
			}
			continue // Skip rules that fail
		}
		allIssues = append(allIssues, issues...)

		if progress != nil {
			progress.RuleDone()
		}
	}

	// Stop progress before output
	if progress != nil {
		progress.Done(nil)
		progress = nil
	}

	// Filter to fixable issues
	var fixableIssues []rules.Issue
	for _, issue := range allIssues {
		if issue.Fix != nil {
			fixableIssues = append(fixableIssues, issue)
		}
	}

	if len(fixableIssues) == 0 {
		fmt.Println(u.Styles.Success.Render(u.Styles.IconSuccess + " No fixable issues found!"))
		return nil
	}

	fmt.Printf("Found %d fixable issues\n\n", len(fixableIssues))

	// Create fixer
	f := fixer.New(fixer.Options{
		DryRun:     dryRun,
		AIAssisted: aiAssisted,
	}, u)

	// Apply fixes
	for _, issue := range fixableIssues {
		if err := f.ApplyFix(issue); err != nil {
			fmt.Println(u.Styles.Error.Render(
				fmt.Sprintf("%s Failed to fix %s: %v", u.Styles.IconError, issue.Rule, err),
			))
		}
	}

	return nil
}
