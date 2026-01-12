package parser

import (
	"strings"
)

// PlainParser parses plain text files with no special structure
type PlainParser struct{}

// CanParse returns true (fallback parser)
func (p *PlainParser) CanParse(path string) bool {
	return true
}

// Parse parses a plain text file
func (p *PlainParser) Parse(path string, content []byte) (*ParsedFile, error) {
	lines := strings.Split(string(content), "\n")

	return &ParsedFile{
		Path:     path,
		Content:  content,
		FileType: FileTypeUnknown,
		Sections: []Section{
			{
				Title:     "Content",
				Level:     1,
				StartLine: 1,
				EndLine:   len(lines),
				Content:   string(content),
			},
		},
	}, nil
}
