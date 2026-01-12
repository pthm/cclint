package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/pthm/cclint/internal/ui"
	"github.com/pthm/cclint/internal/update"
	"github.com/pthm/cclint/internal/version"
	"github.com/spf13/cobra"
)

var (
	// Global flags
	verbose       bool
	format        string
	agentType     string
	noUpdateCheck bool

	// Global UI instance
	globalUI *ui.UI

	// Update check result channel
	updateResult chan *update.Info
)

var RootCmd = &cobra.Command{
	Use:     "cclint",
	Short:   "A linter for Claude Code configurations",
	Version: version.Short(),
	Long: `cclint analyzes Claude Code configurations and related files
to identify issues, suggest improvements, and ensure best practices.

It builds a reference tree of your agent configurations, analyzes
documentation quality, and checks for common problems like broken
references, circular dependencies, and unclear instructions.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Initialize global UI with TTY detection
		globalUI = ui.New(os.Stdout, os.Stderr, format)

		// Start background update check (unless disabled)
		if !noUpdateCheck {
			updateResult = make(chan *update.Info, 1)
			go func() {
				info, _ := update.CheckWithCache(context.Background())
				updateResult <- info
			}()
		}
	},
}

func init() {
	RootCmd.SetVersionTemplate(fmt.Sprintf("%s\n", version.Info()))
	RootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")
	RootCmd.PersistentFlags().StringVarP(&format, "format", "f", "terminal", "Output format (terminal, json)")
	RootCmd.PersistentFlags().StringVarP(&agentType, "agent", "a", "claude-code", "Agent type to lint for")
	RootCmd.PersistentFlags().BoolVar(&noUpdateCheck, "no-update-check", false, "Disable update check")
}

// showUpdateNotice displays the update available notice
func showUpdateNotice(info *update.Info) {
	fmt.Fprintln(os.Stderr)
	if globalUI.IsInteractive() {
		fmt.Fprintf(os.Stderr, "%s A new version of cclint is available: %s (current: %s)\n",
			globalUI.Styles.Info.Render("*"),
			globalUI.Styles.Success.Render("v"+info.LatestVersion),
			info.CurrentVersion)
		fmt.Fprintf(os.Stderr, "  %s  or  %s\n",
			globalUI.Styles.Subheader.Render("brew upgrade cclint"),
			globalUI.Styles.Subheader.Render("go install github.com/pthm/cclint@latest"))
	} else {
		fmt.Fprintf(os.Stderr, "* A new version of cclint is available: v%s (current: %s)\n",
			info.LatestVersion, info.CurrentVersion)
		fmt.Fprintln(os.Stderr, "  brew upgrade cclint  or  go install github.com/pthm/cclint@latest")
	}
}

// GetUI returns the global UI instance for use by subcommands
func GetUI() *ui.UI {
	return globalUI
}

// ShowUpdateNoticeIfAvailable checks for pending update results and displays a notice
// This should be called after command execution (from main.go) since PersistentPostRun
// doesn't run when commands return errors.
func ShowUpdateNoticeIfAvailable() {
	if updateResult == nil || globalUI == nil || globalUI.IsJSON() {
		return
	}

	// Wait briefly for cached results (fast), skip if network check is slow
	select {
	case info := <-updateResult:
		if info != nil && info.UpdateAvailable {
			showUpdateNotice(info)
		}
	case <-time.After(100 * time.Millisecond):
		// Check not finished in time, skip notice
	}
}
