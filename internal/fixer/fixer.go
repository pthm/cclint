package fixer

import (
	"fmt"
	"os"
	"strings"

	"github.com/pthm/cclint/internal/rules"
	"github.com/pthm/cclint/internal/ui"
)

// Options configures the fixer behavior
type Options struct {
	DryRun     bool
	AIAssisted bool
}

// Fixer applies fixes to configuration files
type Fixer struct {
	opts Options
	ui   *ui.UI
}

// New creates a new Fixer
func New(opts Options, u *ui.UI) *Fixer {
	return &Fixer{opts: opts, ui: u}
}

// ApplyFix applies a fix for an issue
func (f *Fixer) ApplyFix(issue rules.Issue) error {
	if issue.Fix == nil {
		return fmt.Errorf("no fix available for this issue")
	}

	fix := issue.Fix

	if f.opts.DryRun {
		f.printDryRun(issue, fix)
		return nil
	}

	// Apply each edit
	for _, edit := range fix.Edits {
		if err := f.applyEdit(edit); err != nil {
			return fmt.Errorf("failed to apply edit to %s: %w", edit.File, err)
		}
	}

	fmt.Println(f.ui.Styles.Success.Render(
		fmt.Sprintf("%s Fixed: %s", f.ui.Styles.IconSuccess, issue.Rule),
	))
	fmt.Printf("  %s\n", fix.Description)

	return nil
}

func (f *Fixer) printDryRun(issue rules.Issue, fix *rules.Fix) {
	fmt.Println(f.ui.Styles.Suggestion.Render(
		fmt.Sprintf("Would fix: %s", issue.Rule),
	))
	fmt.Printf("  File: %s\n", issue.File)
	fmt.Printf("  Line: %d\n", issue.Line)
	fmt.Printf("  Fix: %s\n", fix.Description)

	if len(fix.Edits) > 0 {
		fmt.Println("  Changes:")
		for _, edit := range fix.Edits {
			fmt.Printf("    %s:%d-%d\n", edit.File, edit.StartLine, edit.EndLine)
			if edit.NewContent != "" {
				// Show a preview of the new content
				preview := edit.NewContent
				if len(preview) > 100 {
					preview = preview[:100] + "..."
				}
				fmt.Println(f.ui.Styles.Success.Render(
					"    + " + strings.ReplaceAll(preview, "\n", "\n    + "),
				))
			}
		}
	}
	fmt.Println()
}

func (f *Fixer) applyEdit(edit rules.Edit) error {
	// Read file
	content, err := os.ReadFile(edit.File)
	if err != nil {
		return err
	}

	lines := strings.Split(string(content), "\n")

	// Validate line numbers
	if edit.StartLine < 1 || edit.StartLine > len(lines) {
		return fmt.Errorf("invalid start line: %d", edit.StartLine)
	}
	if edit.EndLine < edit.StartLine || edit.EndLine > len(lines) {
		edit.EndLine = edit.StartLine
	}

	// Apply edit
	newLines := make([]string, 0, len(lines))
	newLines = append(newLines, lines[:edit.StartLine-1]...)

	if edit.NewContent != "" {
		newLines = append(newLines, strings.Split(edit.NewContent, "\n")...)
	}

	newLines = append(newLines, lines[edit.EndLine:]...)

	// Write file
	newContent := strings.Join(newLines, "\n")
	return os.WriteFile(edit.File, []byte(newContent), 0644)
}

// AIFix generates an AI-assisted fix using Claude
func (f *Fixer) AIFix(issue rules.Issue) (*rules.Fix, error) {
	if !f.opts.AIAssisted {
		return nil, fmt.Errorf("AI-assisted fixes not enabled")
	}

	// TODO: Implement AI-assisted fix generation using Claude
	// This would:
	// 1. Send the issue context to Claude
	// 2. Ask for a suggested fix
	// 3. Return the fix for user approval

	return nil, fmt.Errorf("AI-assisted fixes not yet implemented")
}
