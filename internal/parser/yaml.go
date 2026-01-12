package parser

import (
	"gopkg.in/yaml.v3"
)

// YAMLParser parses YAML configuration files
type YAMLParser struct{}

// CanParse returns true if this parser can handle the file
func (p *YAMLParser) CanParse(path string) bool {
	ft := GetFileType(path)
	return ft == FileTypeYAML
}

// Parse parses a YAML file
func (p *YAMLParser) Parse(path string, content []byte) (*ParsedFile, error) {
	// Validate YAML
	var data interface{}
	if err := yaml.Unmarshal(content, &data); err != nil {
		return nil, err
	}

	// Extract top-level keys as sections
	sections := p.extractSections(data)

	return &ParsedFile{
		Path:     path,
		Content:  content,
		FileType: FileTypeYAML,
		Sections: sections,
	}, nil
}

// extractSections extracts top-level keys as sections
func (p *YAMLParser) extractSections(data interface{}) []Section {
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
