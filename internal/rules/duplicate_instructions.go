package rules

import (
	"crypto/sha256"
	"fmt"
	"strings"

	"github.com/pthm-cable/cclint/internal/parser"
)

// DuplicateInstructionsRule checks for duplicate or near-duplicate instructions
type DuplicateInstructionsRule struct{}

func (r *DuplicateInstructionsRule) Name() string {
	return "duplicate-instructions"
}

func (r *DuplicateInstructionsRule) Description() string {
	return "Checks for duplicate or near-duplicate instructions across config files"
}

func (r *DuplicateInstructionsRule) Config() RuleConfig {
	return RuleConfig{
		FileCategories: []parser.FileCategory{
			parser.FileCategoryInstructions,
			parser.FileCategoryDocumentation,
		},
	}
}

func (r *DuplicateInstructionsRule) Run(ctx *AnalysisContext) ([]Issue, error) {
	var issues []Issue

	// Collect all sections with their hashes
	type sectionInfo struct {
		file    string
		title   string
		content string
		hash    string
	}

	var sections []sectionInfo

	for _, node := range ctx.FilesOfType(
		parser.FileCategoryInstructions,
		parser.FileCategoryDocumentation,
	) {
		if node.Parsed == nil {
			continue
		}

		for _, section := range node.Parsed.Sections {
			if len(section.Content) < 50 {
				continue // Skip very short sections
			}

			normalized := normalizeContent(section.Content)
			hash := hashContent(normalized)

			sections = append(sections, sectionInfo{
				file:    node.Path,
				title:   section.Title,
				content: normalized,
				hash:    hash,
			})
		}
	}

	// Find duplicates
	seen := make(map[string]sectionInfo)
	for _, section := range sections {
		if existing, found := seen[section.hash]; found {
			// Found duplicate
			if existing.file != section.file {
				issues = append(issues, Issue{
					Rule:     r.Name(),
					Severity: Suggestion,
					Message: fmt.Sprintf(
						"Section '%s' appears to be duplicated from '%s' in '%s'",
						section.title,
						existing.title,
						existing.file,
					),
					File: section.file,
					Line: 1,
				})
			}
		} else {
			seen[section.hash] = section
		}
	}

	// Check for similar content (not exact duplicates)
	// This is a simplified similarity check - could be enhanced with proper similarity metrics
	for i := 0; i < len(sections); i++ {
		for j := i + 1; j < len(sections); j++ {
			if sections[i].file == sections[j].file {
				continue
			}
			if sections[i].hash == sections[j].hash {
				continue // Already reported as duplicate
			}

			similarity := calculateSimilarity(sections[i].content, sections[j].content)
			if similarity > 0.8 {
				issues = append(issues, Issue{
					Rule:     r.Name(),
					Severity: Info,
					Message: fmt.Sprintf(
						"Section '%s' in '%s' is %.0f%% similar to '%s' in '%s'",
						sections[i].title,
						sections[i].file,
						similarity*100,
						sections[j].title,
						sections[j].file,
					),
					File: sections[i].file,
					Line: 1,
				})
			}
		}
	}

	return issues, nil
}

// normalizeContent normalizes content for comparison
func normalizeContent(content string) string {
	// Remove extra whitespace
	content = strings.TrimSpace(content)
	// Collapse multiple spaces/newlines
	for strings.Contains(content, "  ") {
		content = strings.ReplaceAll(content, "  ", " ")
	}
	for strings.Contains(content, "\n\n\n") {
		content = strings.ReplaceAll(content, "\n\n\n", "\n\n")
	}
	return strings.ToLower(content)
}

// hashContent creates a hash of normalized content
func hashContent(content string) string {
	h := sha256.Sum256([]byte(content))
	return fmt.Sprintf("%x", h[:8])
}

// calculateSimilarity calculates a simple similarity score between two strings
func calculateSimilarity(a, b string) float64 {
	if a == b {
		return 1.0
	}

	wordsA := strings.Fields(a)
	wordsB := strings.Fields(b)

	if len(wordsA) == 0 || len(wordsB) == 0 {
		return 0.0
	}

	// Count common words
	wordSetA := make(map[string]bool)
	for _, w := range wordsA {
		wordSetA[w] = true
	}

	common := 0
	for _, w := range wordsB {
		if wordSetA[w] {
			common++
		}
	}

	// Jaccard similarity
	total := len(wordsA) + len(wordsB) - common
	if total == 0 {
		return 0.0
	}

	return float64(common) / float64(total)
}
