package reporter

import (
	"fmt"
	"io"
	"path/filepath"
	"sort"

	"github.com/fatih/color"
	"github.com/pthm-cable/cclint/internal/rules"
)

// TerminalReporter outputs results to the terminal with colors
type TerminalReporter struct {
	w io.Writer
}

// NewTerminalReporter creates a new terminal reporter
func NewTerminalReporter(w io.Writer) *TerminalReporter {
	return &TerminalReporter{w: w}
}

// Report outputs issues to the terminal
func (r *TerminalReporter) Report(issues []rules.Issue) error {
	if len(issues) == 0 {
		color.New(color.FgGreen).Fprintln(r.w, "✓ No issues found")
		return nil
	}

	// Group issues by file
	byFile := make(map[string][]rules.Issue)
	for _, issue := range issues {
		byFile[issue.File] = append(byFile[issue.File], issue)
	}

	// Sort files
	var files []string
	for f := range byFile {
		files = append(files, f)
	}
	sort.Strings(files)

	// Print issues grouped by file
	for _, file := range files {
		fileIssues := byFile[file]

		// Sort issues by line
		sort.Slice(fileIssues, func(i, j int) bool {
			return fileIssues[i].Line < fileIssues[j].Line
		})

		// Print file header
		fmt.Fprintln(r.w)
		color.New(color.FgWhite, color.Bold).Fprintf(r.w, "%s\n", filepath.Base(file))
		color.New(color.FgHiBlack).Fprintf(r.w, "  %s\n", file)

		// Print each issue
		for _, issue := range fileIssues {
			r.printIssue(issue)
		}
	}

	// Print summary
	r.printSummary(issues)

	// Return error if there are errors
	for _, issue := range issues {
		if issue.Severity == rules.Error {
			return fmt.Errorf("lint errors found")
		}
	}

	return nil
}

func (r *TerminalReporter) printIssue(issue rules.Issue) {
	var severityColor *color.Color
	var icon string

	switch issue.Severity {
	case rules.Error:
		severityColor = color.New(color.FgRed)
		icon = "✗"
	case rules.Warning:
		severityColor = color.New(color.FgYellow)
		icon = "⚠"
	case rules.Suggestion:
		severityColor = color.New(color.FgCyan)
		icon = "💡"
	case rules.Info:
		severityColor = color.New(color.FgBlue)
		icon = "ℹ"
	}

	// Line info
	lineInfo := ""
	if issue.Line > 0 {
		lineInfo = fmt.Sprintf(":%d", issue.Line)
		if issue.Column > 0 {
			lineInfo = fmt.Sprintf(":%d:%d", issue.Line, issue.Column)
		}
	}

	severityColor.Fprintf(r.w, "  %s ", icon)
	fmt.Fprintf(r.w, "%s%s", filepath.Base(issue.File), lineInfo)
	color.New(color.FgHiBlack).Fprintf(r.w, " [%s]", issue.Rule)
	fmt.Fprintln(r.w)
	fmt.Fprintf(r.w, "    %s\n", issue.Message)

	// Print context if available
	if issue.Context != "" && len(issue.Context) < 200 {
		color.New(color.FgHiBlack).Fprintf(r.w, "    > %s\n", issue.Context)
	}
}

func (r *TerminalReporter) printSummary(issues []rules.Issue) {
	summary := ComputeSummary(issues)

	fmt.Fprintln(r.w)
	fmt.Fprintln(r.w, "─────────────────────────────────────")

	parts := []string{}

	if summary.Errors > 0 {
		parts = append(parts, color.RedString("%d errors", summary.Errors))
	}
	if summary.Warnings > 0 {
		parts = append(parts, color.YellowString("%d warnings", summary.Warnings))
	}
	if summary.Suggestions > 0 {
		parts = append(parts, color.CyanString("%d suggestions", summary.Suggestions))
	}
	if summary.Info > 0 {
		parts = append(parts, color.BlueString("%d info", summary.Info))
	}

	fmt.Fprintf(r.w, "Found %d issues in %d files: ", summary.TotalIssues, summary.Files)
	for i, part := range parts {
		if i > 0 {
			fmt.Fprint(r.w, ", ")
		}
		fmt.Fprint(r.w, part)
	}
	fmt.Fprintln(r.w)
}
