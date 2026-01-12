package parser

import (
	"encoding/json"
)

// JSONParser parses JSON configuration files
type JSONParser struct{}

// CanParse returns true if this parser can handle the file
func (p *JSONParser) CanParse(path string) bool {
	return GetFileType(path) == FileTypeJSON
}

// Parse parses a JSON file
func (p *JSONParser) Parse(path string, content []byte) (*ParsedFile, error) {
	// Validate JSON
	var data interface{}
	if err := json.Unmarshal(content, &data); err != nil {
		return nil, err
	}

	// Extract top-level keys as sections
	sections := p.extractSections(data, content)

	return &ParsedFile{
		Path:     path,
		Content:  content,
		FileType: FileTypeJSON,
		Sections: sections,
	}, nil
}

// extractSections extracts top-level keys as sections
func (p *JSONParser) extractSections(data interface{}, source []byte) []Section {
	var sections []Section

	if obj, ok := data.(map[string]interface{}); ok {
		line := 1
		for key := range obj {
			sections = append(sections, Section{
				Title:     key,
				Level:     1,
				StartLine: line,
			})
			line++
		}
	}

	return sections
}
