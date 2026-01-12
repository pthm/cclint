package classifier

import (
	"github.com/pthm-cable/cclint/internal/rules"
)

// Classifier defines the interface for configuration quality classification
type Classifier interface {
	// Analyze runs classification on the analysis context
	Analyze(ctx *rules.AnalysisContext) ([]rules.Issue, error)
}

// QualityMetrics holds quality assessment metrics
type QualityMetrics struct {
	Clarity       float64 `json:"clarity"`       // 0-1: How clear are the instructions
	Specificity   float64 `json:"specificity"`   // 0-1: How specific vs vague
	Consistency   float64 `json:"consistency"`   // 0-1: Internal consistency
	Completeness  float64 `json:"completeness"`  // 0-1: Coverage of expected topics
	Verbosity     string  `json:"verbosity"`     // "concise", "moderate", "verbose"
	OverallScore  float64 `json:"overall_score"` // 0-1: Overall quality
}

// SectionAnalysis holds analysis results for a section
type SectionAnalysis struct {
	Title           string
	File            string
	Metrics         QualityMetrics
	Contradictions  []string
	Suggestions     []string
}
