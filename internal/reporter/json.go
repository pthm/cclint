package reporter

import (
	"encoding/json"
	"io"

	"github.com/pthm-cable/cclint/internal/rules"
)

// JSONReporter outputs results as JSON
type JSONReporter struct {
	w io.Writer
}

// NewJSONReporter creates a new JSON reporter
func NewJSONReporter(w io.Writer) *JSONReporter {
	return &JSONReporter{w: w}
}

// JSONOutput represents the JSON output format
type JSONOutput struct {
	Issues  []JSONIssue `json:"issues"`
	Summary Summary     `json:"summary"`
}

// JSONIssue represents an issue in JSON format
type JSONIssue struct {
	Rule     string `json:"rule"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
	File     string `json:"file"`
	Line     int    `json:"line,omitempty"`
	Column   int    `json:"column,omitempty"`
	EndLine  int    `json:"endLine,omitempty"`
	Context  string `json:"context,omitempty"`
	HasFix   bool   `json:"hasFix"`
}

// Report outputs issues as JSON
func (r *JSONReporter) Report(issues []rules.Issue) error {
	output := JSONOutput{
		Issues:  make([]JSONIssue, 0, len(issues)),
		Summary: ComputeSummary(issues),
	}

	for _, issue := range issues {
		output.Issues = append(output.Issues, JSONIssue{
			Rule:     issue.Rule,
			Severity: issue.Severity.String(),
			Message:  issue.Message,
			File:     issue.File,
			Line:     issue.Line,
			Column:   issue.Column,
			EndLine:  issue.EndLine,
			Context:  issue.Context,
			HasFix:   issue.Fix != nil,
		})
	}

	encoder := json.NewEncoder(r.w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}
