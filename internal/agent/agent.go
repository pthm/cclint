package agent

import "regexp"

// Config represents an agent configuration that defines how to
// parse and analyze configuration files for a specific AI coding agent.
type Config struct {
	// Name is the identifier for this agent (e.g., "claude-code", "cursor")
	Name string `yaml:"name"`

	// Entrypoints are the files that serve as starting points for analysis
	Entrypoints []string `yaml:"entrypoints"`

	// ReferencePatterns define how to extract references from config files
	ReferencePatterns []ReferencePattern `yaml:"reference_patterns"`

	// Markers define keywords that affect priority scoring
	Markers Markers `yaml:"markers"`

	// FilePatterns define additional files to include in analysis
	FilePatterns []string `yaml:"file_patterns"`
}

// ReferencePattern defines a pattern for extracting references from text
type ReferencePattern struct {
	// Regex is the pattern to match
	Regex string `yaml:"regex"`

	// Type categorizes the reference (file, url, tool, etc.)
	Type string `yaml:"type"`

	// compiled is the compiled regex
	compiled *regexp.Regexp
}

// Compile compiles the regex pattern
func (rp *ReferencePattern) Compile() error {
	re, err := regexp.Compile(rp.Regex)
	if err != nil {
		return err
	}
	rp.compiled = re
	return nil
}

// CompiledRegex returns the compiled regex
func (rp *ReferencePattern) CompiledRegex() *regexp.Regexp {
	return rp.compiled
}

// Markers define keywords that affect priority scoring
type Markers struct {
	// HighPriority keywords indicate critical instructions
	HighPriority []string `yaml:"high_priority"`

	// MediumPriority keywords indicate important instructions
	MediumPriority []string `yaml:"medium_priority"`

	// LowPriority keywords indicate suggestions
	LowPriority []string `yaml:"low_priority"`

	// Sections define expected section headers
	Sections []string `yaml:"sections"`
}
