package parser

import (
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// ParsedFile represents a parsed configuration file
type ParsedFile struct {
	Path        string
	Content     []byte
	FileType    FileType
	Category    FileCategory
	Sections    []Section
	Frontmatter map[string]interface{} // YAML frontmatter from markdown files
}

// FileType represents the type of configuration file
type FileType int

const (
	FileTypeUnknown FileType = iota
	FileTypeMarkdown
	FileTypeJSON
	FileTypeYAML
)

// FileCategory represents the semantic purpose of a file
type FileCategory int

const (
	// FileCategoryUnknown is for files that don't match any known category
	FileCategoryUnknown FileCategory = iota
	// FileCategoryConfig is for settings and configuration files (settings.json, mcp.json)
	FileCategoryConfig
	// FileCategoryInstructions is for directive content (CLAUDE.md files)
	FileCategoryInstructions
	// FileCategoryCommands is for custom slash commands (.claude/commands/)
	FileCategoryCommands
	// FileCategoryDocumentation is for descriptive content (README.md, docs)
	FileCategoryDocumentation
)

func (c FileCategory) String() string {
	switch c {
	case FileCategoryConfig:
		return "config"
	case FileCategoryInstructions:
		return "instructions"
	case FileCategoryCommands:
		return "commands"
	case FileCategoryDocumentation:
		return "documentation"
	default:
		return "unknown"
	}
}

// Section represents a section within a parsed file
type Section struct {
	Title      string
	Level      int
	StartLine  int
	EndLine    int
	Content    string
	Subsections []Section
}

// Parser defines the interface for parsing configuration files
type Parser interface {
	Parse(path string, content []byte) (*ParsedFile, error)
	CanParse(path string) bool
}

// Parse parses a file using the appropriate parser
func Parse(path string) (*ParsedFile, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	parser := getParser(path)
	parsed, err := parser.Parse(path, content)
	if err != nil {
		return nil, err
	}

	// Set the semantic category
	parsed.Category = GetFileCategory(path)
	return parsed, nil
}

// getParser returns the appropriate parser for a file
func getParser(path string) Parser {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".md", ".markdown":
		return &MarkdownParser{}
	case ".json":
		return &JSONParser{}
	case ".yaml", ".yml":
		return &YAMLParser{}
	default:
		// Try to detect by filename
		base := filepath.Base(path)
		if strings.Contains(strings.ToUpper(base), "CLAUDE") && ext == "" {
			return &MarkdownParser{}
		}
		return &PlainParser{}
	}
}

// GetFileType returns the FileType for a given path
func GetFileType(path string) FileType {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".md", ".markdown":
		return FileTypeMarkdown
	case ".json":
		return FileTypeJSON
	case ".yaml", ".yml":
		return FileTypeYAML
	default:
		return FileTypeUnknown
	}
}

// GetFileCategory returns the semantic FileCategory for a given path
func GetFileCategory(path string) FileCategory {
	base := filepath.Base(path)
	baseLower := strings.ToLower(base)
	dir := filepath.Dir(path)

	// Config files - settings and MCP configuration
	configFiles := []string{
		"settings.json",
		"settings.local.json",
		"mcp.json",
		".mcp.json",
	}
	for _, cf := range configFiles {
		if baseLower == cf {
			return FileCategoryConfig
		}
	}

	// Commands - files in .claude/commands/ directory
	if strings.Contains(dir, ".claude/commands") || strings.Contains(dir, ".claude\\commands") {
		return FileCategoryCommands
	}

	// Instructions - CLAUDE.md files
	if strings.Contains(strings.ToUpper(base), "CLAUDE") {
		return FileCategoryInstructions
	}

	// Documentation - README files and docs directories
	if strings.HasPrefix(baseLower, "readme") {
		return FileCategoryDocumentation
	}
	if strings.Contains(dir, "/docs/") || strings.Contains(dir, "\\docs\\") ||
		strings.HasSuffix(dir, "/docs") || strings.HasSuffix(dir, "\\docs") {
		return FileCategoryDocumentation
	}

	return FileCategoryUnknown
}

// ParseFrontmatter extracts YAML frontmatter from content between --- delimiters
// Returns the parsed frontmatter and the remaining content without frontmatter
func ParseFrontmatter(content []byte) (map[string]interface{}, []byte) {
	s := string(content)

	// Must start with ---
	if !strings.HasPrefix(s, "---") {
		return nil, content
	}

	// Find the closing ---
	rest := s[3:]
	endIdx := strings.Index(rest, "\n---")
	if endIdx == -1 {
		return nil, content
	}

	// Extract frontmatter YAML
	frontmatterStr := strings.TrimSpace(rest[:endIdx])

	var frontmatter map[string]interface{}
	if err := yaml.Unmarshal([]byte(frontmatterStr), &frontmatter); err != nil {
		return nil, content
	}

	// Return remaining content after frontmatter
	remaining := rest[endIdx+4:] // +4 for "\n---"
	if strings.HasPrefix(remaining, "\n") {
		remaining = remaining[1:]
	}

	return frontmatter, []byte(remaining)
}
