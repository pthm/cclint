package reporter

import (
	"fmt"
	"io"
	"path/filepath"
	"sort"

	"github.com/pthm/cclint/internal/rules"
	"github.com/pthm/cclint/internal/ui"
)

// TerminalReporter outputs results to the terminal with colors
type TerminalReporter struct {
	w  io.Writer
	ui *ui.UI
}

// NewTerminalReporter creates a new terminal reporter
func NewTerminalReporter(w io.Writer, u *ui.UI) *TerminalReporter {
	return &TerminalReporter{w: w, ui: u}
}

// Report outputs issues to the terminal
func (r *TerminalReporter) Report(issues []rules.Issue) error {
	if len(issues) == 0 {
		fmt.Fprintln(r.w, r.ui.Styles.Success.Render(r.ui.Styles.IconSuccess+" No issues found"))
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
		fmt.Fprintln(r.w, r.ui.Styles.Header.Render(filepath.Base(file)))
		fmt.Fprintln(r.w, r.ui.Styles.Path.Render("  "+file))

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
	var style *ui.Styles
	var icon string

	style = r.ui.Styles

	switch issue.Severity {
	case rules.Error:
		icon = style.Error.Render(style.IconError)
	case rules.Warning:
		icon = style.Warning.Render(style.IconWarning)
	case rules.Suggestion:
		icon = style.Suggestion.Render(style.IconSuggestion)
	case rules.Info:
		icon = style.Info.Render(style.IconInfo)
	default:
		icon = style.Info.Render(style.IconInfo)
	}

	// Line info
	lineInfo := ""
	if issue.Line > 0 {
		lineInfo = fmt.Sprintf(":%d", issue.Line)
		if issue.Column > 0 {
			lineInfo = fmt.Sprintf(":%d:%d", issue.Line, issue.Column)
		}
	}

	fmt.Fprintf(r.w, "  %s ", icon)
	fmt.Fprintf(r.w, "%s%s", filepath.Base(issue.File), lineInfo)
	fmt.Fprintf(r.w, " %s", style.Rule.Render("["+issue.Rule+"]"))
	fmt.Fprintln(r.w)
	fmt.Fprintf(r.w, "    %s\n", issue.Message)

	// Print context if available
	if issue.Context != "" && len(issue.Context) < 200 {
		fmt.Fprintf(r.w, "    %s\n", style.Subheader.Render("> "+issue.Context))
	}
}

func (r *TerminalReporter) printSummary(issues []rules.Issue) {
	summary := ComputeSummary(issues)

	fmt.Fprintln(r.w)
	fmt.Fprintln(r.w, r.ui.Styles.Separator.Render("─────────────────────────────────────"))

	parts := []string{}

	if summary.Errors > 0 {
		parts = append(parts, r.ui.Styles.Error.Render(fmt.Sprintf("%d errors", summary.Errors)))
	}
	if summary.Warnings > 0 {
		parts = append(parts, r.ui.Styles.Warning.Render(fmt.Sprintf("%d warnings", summary.Warnings)))
	}
	if summary.Suggestions > 0 {
		parts = append(parts, r.ui.Styles.Suggestion.Render(fmt.Sprintf("%d suggestions", summary.Suggestions)))
	}
	if summary.Info > 0 {
		parts = append(parts, r.ui.Styles.Info.Render(fmt.Sprintf("%d info", summary.Info)))
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
