package rules

import (
	"github.com/pthm/cclint/internal/agent"
	"github.com/pthm/cclint/internal/analyzer"
	"github.com/pthm/cclint/internal/parser"
)

// Severity represents the severity level of an issue
type Severity int

const (
	Info Severity = iota
	Suggestion
	Warning
	Error
)

func (s Severity) String() string {
	switch s {
	case Info:
		return "info"
	case Suggestion:
		return "suggestion"
	case Warning:
		return "warning"
	case Error:
		return "error"
	default:
		return "unknown"
	}
}

// Fix represents an auto-fix for an issue
type Fix struct {
	Description string
	Edits       []Edit
}

// Edit represents a single edit operation
type Edit struct {
	File       string
	StartLine  int
	EndLine    int
	NewContent string
}

// Issue represents a linting issue
type Issue struct {
	Rule     string
	Severity Severity
	Message  string
	File     string
	Line     int
	Column   int
	EndLine  int
	Context  string
	Fix      *Fix
}

// AnalysisContext provides context for rule analysis
type AnalysisContext struct {
	Tree        *analyzer.Tree
	AgentConfig *agent.Config
	RootPath    string
}

// AllFiles returns all ConfigNodes in the tree.
func (ctx *AnalysisContext) AllFiles() []*analyzer.ConfigNode {
	nodes := make([]*analyzer.ConfigNode, 0, len(ctx.Tree.Nodes))
	for _, node := range ctx.Tree.Nodes {
		nodes = append(nodes, node)
	}
	return nodes
}

// FilesOfType returns all ConfigNodes matching the given categories.
// If no categories are provided, returns all files.
func (ctx *AnalysisContext) FilesOfType(categories ...parser.FileCategory) []*analyzer.ConfigNode {
	if len(categories) == 0 {
		return ctx.AllFiles()
	}

	categorySet := make(map[parser.FileCategory]bool)
	for _, cat := range categories {
		categorySet[cat] = true
	}

	var nodes []*analyzer.ConfigNode
	for _, node := range ctx.Tree.Nodes {
		if node.Parsed == nil {
			continue
		}
		if categorySet[node.Parsed.Category] {
			nodes = append(nodes, node)
		}
	}
	return nodes
}

// FilesWithContent returns all ConfigNodes that have non-empty content.
func (ctx *AnalysisContext) FilesWithContent() []*analyzer.ConfigNode {
	var nodes []*analyzer.ConfigNode
	for _, node := range ctx.Tree.Nodes {
		if len(node.Content) > 0 {
			nodes = append(nodes, node)
		}
	}
	return nodes
}

// FilesMatching returns all ConfigNodes matching the predicate.
func (ctx *AnalysisContext) FilesMatching(predicate func(*analyzer.ConfigNode) bool) []*analyzer.ConfigNode {
	var nodes []*analyzer.ConfigNode
	for _, node := range ctx.Tree.Nodes {
		if predicate(node) {
			nodes = append(nodes, node)
		}
	}
	return nodes
}

// FileByPath returns the ConfigNode for a specific path, or nil if not found.
func (ctx *AnalysisContext) FileByPath(path string) *analyzer.ConfigNode {
	return ctx.Tree.Nodes[path]
}

// RuleConfig defines how a rule should be invoked
type RuleConfig struct {
	// FileCategories specifies which file types this rule applies to.
	// Empty slice means all file types.
	FileCategories []parser.FileCategory

	// RequiresAI indicates this rule needs AI/LLM for analysis.
	// AI rules only run when --deep flag is enabled.
	RequiresAI bool
}

// Rule defines the interface for lint rules
type Rule interface {
	// Name returns the unique identifier for this rule
	Name() string

	// Description returns a human-readable description
	Description() string

	// Config returns the rule's configuration
	Config() RuleConfig

	// Run executes the rule and returns any issues found.
	// AI rules may return errors for API failures; regular rules typically return nil error.
	Run(ctx *AnalysisContext) ([]Issue, error)
}
