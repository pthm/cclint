package parser

import (
	"bytes"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

// MarkdownParser parses markdown configuration files
type MarkdownParser struct{}

// CanParse returns true if this parser can handle the file
func (p *MarkdownParser) CanParse(path string) bool {
	return GetFileType(path) == FileTypeMarkdown
}

// Parse parses a markdown file into sections
func (p *MarkdownParser) Parse(path string, content []byte) (*ParsedFile, error) {
	// Extract frontmatter if present
	frontmatter, contentWithoutFrontmatter := ParseFrontmatter(content)

	md := goldmark.New()
	reader := text.NewReader(contentWithoutFrontmatter)
	doc := md.Parser().Parse(reader)

	sections := p.extractSections(doc, contentWithoutFrontmatter)

	return &ParsedFile{
		Path:        path,
		Content:     content, // Keep original content
		FileType:    FileTypeMarkdown,
		Sections:    sections,
		Frontmatter: frontmatter,
	}, nil
}

// extractSections walks the AST and extracts sections
func (p *MarkdownParser) extractSections(doc ast.Node, source []byte) []Section {
	var sections []Section
	var currentSection *Section
	var sectionStack []*Section

	ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		switch node := n.(type) {
		case *ast.Heading:
			level := node.Level
			title := string(node.Text(source))

			// Get line number from the node's lines
			line := 1
			if node.Lines().Len() > 0 {
				seg := node.Lines().At(0)
				line = bytes.Count(source[:seg.Start], []byte("\n")) + 1
			}

			newSection := Section{
				Title:     title,
				Level:     level,
				StartLine: line,
			}

			// Find parent section
			for len(sectionStack) > 0 && sectionStack[len(sectionStack)-1].Level >= level {
				sectionStack = sectionStack[:len(sectionStack)-1]
			}

			if len(sectionStack) > 0 {
				parent := sectionStack[len(sectionStack)-1]
				parent.Subsections = append(parent.Subsections, newSection)
				currentSection = &parent.Subsections[len(parent.Subsections)-1]
			} else {
				sections = append(sections, newSection)
				currentSection = &sections[len(sections)-1]
			}

			sectionStack = append(sectionStack, currentSection)
		}

		return ast.WalkContinue, nil
	})

	// Extract content for each section
	p.extractSectionContent(sections, source)

	return sections
}

// extractSectionContent extracts the text content for each section
func (p *MarkdownParser) extractSectionContent(sections []Section, source []byte) {
	lines := strings.Split(string(source), "\n")

	var extractContent func(sections []Section)
	extractContent = func(sections []Section) {
		for i := range sections {
			section := &sections[i]

			// Find end line (start of next section at same or higher level, or EOF)
			endLine := len(lines)
			if i+1 < len(sections) {
				endLine = sections[i+1].StartLine - 1
			}

			// Also check subsections
			if len(section.Subsections) > 0 {
				endLine = section.Subsections[0].StartLine - 1
			}

			section.EndLine = endLine

			// Extract content
			if section.StartLine > 0 && section.EndLine >= section.StartLine {
				start := section.StartLine - 1
				end := section.EndLine
				if end > len(lines) {
					end = len(lines)
				}
				section.Content = strings.Join(lines[start:end], "\n")
			}

			// Recurse into subsections
			if len(section.Subsections) > 0 {
				extractContent(section.Subsections)
			}
		}
	}

	extractContent(sections)
}
