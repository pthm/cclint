package rules

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/pthm/cclint/internal/analyzer"
)

// BrokenRefsRule checks for broken file and URL references
type BrokenRefsRule struct{}

func (r *BrokenRefsRule) Name() string {
	return "broken-refs"
}

func (r *BrokenRefsRule) Description() string {
	return "Checks for references to files or URLs that don't exist or are invalid"
}

func (r *BrokenRefsRule) Config() RuleConfig {
	return RuleConfig{} // Applies to all file types
}

func (r *BrokenRefsRule) Run(ctx *AnalysisContext) ([]Issue, error) {
	var issues []Issue

	for _, node := range ctx.AllFiles() {
		for _, ref := range node.References {
			switch ref.Type {
			case analyzer.RefTypeFile:
				if issue := r.checkFileRef(ref, ctx.RootPath); issue != nil {
					issues = append(issues, *issue)
				}
			case analyzer.RefTypeURL:
				if issue := r.checkURLRef(ref); issue != nil {
					issues = append(issues, *issue)
				}
			}
		}
	}

	return issues, nil
}

func (r *BrokenRefsRule) checkFileRef(ref analyzer.Reference, rootPath string) *Issue {
	// Try to resolve the file path
	refPath := strings.TrimPrefix(ref.Value, "@")

	// Try relative to source file
	var resolvedPath string
	if filepath.IsAbs(refPath) {
		resolvedPath = refPath
	} else {
		// Try relative to source file directory
		sourceDir := filepath.Dir(ref.Source.File)
		resolvedPath = filepath.Join(sourceDir, refPath)
	}

	// Check if file exists
	if _, err := os.Stat(resolvedPath); err != nil {
		// Try relative to root
		rootRelative := filepath.Join(rootPath, refPath)
		if _, err := os.Stat(rootRelative); err != nil {
			return &Issue{
				Rule:     r.Name() + "/file-not-found",
				Severity: Error,
				Message:  fmt.Sprintf("Referenced file not found: %s", ref.Value),
				File:     ref.Source.File,
				Line:     ref.Source.Line,
				Column:   ref.Source.Column,
				Context:  ref.Context,
				Fix: &Fix{
					Description: "Remove or update the broken reference",
				},
			}
		}
	}

	return nil
}

func (r *BrokenRefsRule) checkURLRef(ref analyzer.Reference) *Issue {
	// Validate URL format
	_, err := url.ParseRequestURI(ref.Value)
	if err != nil {
		return &Issue{
			Rule:     r.Name() + "/invalid-url-format",
			Severity: Error,
			Message:  fmt.Sprintf("Invalid URL format: %s", ref.Value),
			File:     ref.Source.File,
			Line:     ref.Source.Line,
			Column:   ref.Source.Column,
			Context:  ref.Context,
		}
	}

	// Note: We don't check if URL is reachable as that would slow down linting
	// and URLs can be temporarily unavailable

	return nil
}
