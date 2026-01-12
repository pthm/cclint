package analyzer

import (
	"github.com/pthm/cclint/internal/parser"
)

// Metrics contains computed metrics about the configuration
type Metrics struct {
	TotalFiles       int
	TotalReferences  int
	EstimatedTokens  int
	TotalBytes       int
	ReferencesByType map[string]int
	FilesByType      map[string]int
	MaxDepth         int
	UnresolvedRefs   int
}

// ComputeMetrics computes metrics for a configuration tree
func ComputeMetrics(tree *Tree) *Metrics {
	m := &Metrics{
		ReferencesByType: make(map[string]int),
		FilesByType:      make(map[string]int),
	}

	for _, node := range tree.Nodes {
		if node.Content == nil {
			continue
		}

		m.TotalFiles++
		m.TotalBytes += len(node.Content)

		// Estimate tokens (rough: ~4 chars per token)
		m.EstimatedTokens += len(node.Content) / 4

		// Track max depth
		if node.Depth > m.MaxDepth {
			m.MaxDepth = node.Depth
		}

		// Count file types
		if node.Parsed != nil {
			fileType := fileTypeToString(node.Parsed.FileType)
			m.FilesByType[fileType]++
		}

		// Count references
		for _, ref := range node.References {
			m.TotalReferences++
			m.ReferencesByType[ref.Type.String()]++

			if !ref.Resolved && ref.Type == RefTypeFile {
				m.UnresolvedRefs++
			}
		}
	}

	return m
}

// fileTypeToString converts a parser.FileType to string
func fileTypeToString(ft parser.FileType) string {
	switch ft {
	case parser.FileTypeMarkdown:
		return "markdown"
	case parser.FileTypeJSON:
		return "json"
	case parser.FileTypeYAML:
		return "yaml"
	default:
		return "unknown"
	}
}
