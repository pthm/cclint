package reporter

import (
	"github.com/pthm-cable/cclint/internal/rules"
)

// Reporter defines the interface for outputting lint results
type Reporter interface {
	// Report outputs the lint results
	Report(issues []rules.Issue) error
}

// Summary holds summary statistics for a lint run
type Summary struct {
	TotalIssues int
	Errors      int
	Warnings    int
	Suggestions int
	Info        int
	Files       int
}

// ComputeSummary computes summary statistics from issues
func ComputeSummary(issues []rules.Issue) Summary {
	s := Summary{
		TotalIssues: len(issues),
	}

	files := make(map[string]bool)
	for _, issue := range issues {
		files[issue.File] = true
		switch issue.Severity {
		case rules.Error:
			s.Errors++
		case rules.Warning:
			s.Warnings++
		case rules.Suggestion:
			s.Suggestions++
		case rules.Info:
			s.Info++
		}
	}
	s.Files = len(files)

	return s
}
