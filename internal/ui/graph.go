package ui

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/pthm/cclint/internal/analyzer"
)

// RefSource stores metadata about where a file was referenced from
type RefSource struct {
	Line         int    // Line number in parent file
	Column       int    // Column number
	OriginalText string // The markdown text that created this reference
	Context      string // Surrounding lines
	Priority     int    // Priority based on markers
}

// GraphNode represents a displayable node in the graph
type GraphNode struct {
	Node       *analyzer.ConfigNode
	Scope      *analyzer.ContextScope
	Depth      int
	Expanded   bool
	IsScope    bool // True if this is a scope header
	Children   []*GraphNode
	Parent     *GraphNode
	RefType    analyzer.RefType // For reference nodes
	IsRef      bool             // True if this is a reference (not a file)
	RefValue   string           // The reference value
	RefContext string           // Context around the reference
	SourceRefs []RefSource      // Reference sources (merged from parent refs)
}

// GraphModel is the bubbletea model for graph visualization
type GraphModel struct {
	tree          *analyzer.Tree
	scopes        []*analyzer.ContextScope
	rootPath      string
	nodes         []*GraphNode // Flattened list of visible nodes
	allNodes      []*GraphNode // All nodes including collapsed ones
	cursor        int          // Currently selected node
	viewport      viewport.Model
	ready         bool
	width         int
	height        int
	showRefs      bool // Toggle to show/hide references
	showScopes    bool // Toggle to show scope grouping
	showPreview   bool // Toggle preview pane
	previewScroll int  // Scroll offset in preview
	keys          graphKeyMap
	styles        graphStyles
}

type graphKeyMap struct {
	Up           key.Binding
	Down         key.Binding
	Left         key.Binding
	Right        key.Binding
	Toggle       key.Binding
	ToggleRefs   key.Binding
	ToggleScopes key.Binding
	TogglePreview key.Binding
	PreviewUp    key.Binding
	PreviewDown  key.Binding
	Quit         key.Binding
	Help         key.Binding
}

type graphStyles struct {
	selected      lipgloss.Style
	file          lipgloss.Style
	scope         lipgloss.Style
	scopeMain     lipgloss.Style
	scopeSub      lipgloss.Style
	refFile       lipgloss.Style
	refURL        lipgloss.Style
	refTool       lipgloss.Style
	refSubagent   lipgloss.Style
	refSkill      lipgloss.Style
	refMCP        lipgloss.Style
	tree          lipgloss.Style
	dim           lipgloss.Style
	detail        lipgloss.Style
	header        lipgloss.Style
	help          lipgloss.Style
	statusBar     lipgloss.Style
	helpBar       lipgloss.Style
	lineNum       lipgloss.Style
	highlightLine lipgloss.Style
	separator     lipgloss.Style
	previewHeader lipgloss.Style
}

func defaultGraphKeyMap() graphKeyMap {
	return graphKeyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		Left: key.NewBinding(
			key.WithKeys("left", "h"),
			key.WithHelp("←/h", "collapse"),
		),
		Right: key.NewBinding(
			key.WithKeys("right", "l"),
			key.WithHelp("→/l", "expand"),
		),
		Toggle: key.NewBinding(
			key.WithKeys("enter", " "),
			key.WithHelp("enter/space", "toggle"),
		),
		ToggleRefs: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "toggle refs"),
		),
		ToggleScopes: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "toggle scopes"),
		),
		TogglePreview: key.NewBinding(
			key.WithKeys("p"),
			key.WithHelp("p", "preview"),
		),
		PreviewUp: key.NewBinding(
			key.WithKeys("ctrl+u"),
			key.WithHelp("ctrl+u", "preview up"),
		),
		PreviewDown: key.NewBinding(
			key.WithKeys("ctrl+d"),
			key.WithHelp("ctrl+d", "preview down"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
	}
}

func defaultGraphStyles() graphStyles {
	return graphStyles{
		selected:      lipgloss.NewStyle().Background(lipgloss.Color("237")).Bold(true),
		file:          lipgloss.NewStyle().Foreground(lipgloss.Color("15")),
		scope:         lipgloss.NewStyle().Foreground(lipgloss.Color("14")).Bold(true),
		scopeMain:     lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true),
		scopeSub:      lipgloss.NewStyle().Foreground(lipgloss.Color("13")).Bold(true),
		refFile:       lipgloss.NewStyle().Foreground(lipgloss.Color("12")),
		refURL:        lipgloss.NewStyle().Foreground(lipgloss.Color("11")),
		refTool:       lipgloss.NewStyle().Foreground(lipgloss.Color("9")),
		refSubagent:   lipgloss.NewStyle().Foreground(lipgloss.Color("13")),
		refSkill:      lipgloss.NewStyle().Foreground(lipgloss.Color("14")),
		refMCP:        lipgloss.NewStyle().Foreground(lipgloss.Color("208")),
		tree:          lipgloss.NewStyle().Foreground(lipgloss.Color("240")),
		dim:           lipgloss.NewStyle().Foreground(lipgloss.Color("242")),
		detail:        lipgloss.NewStyle().Foreground(lipgloss.Color("250")),
		header:        lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("15")).Background(lipgloss.Color("236")).Padding(0, 1),
		help:          lipgloss.NewStyle().Foreground(lipgloss.Color("241")),
		statusBar:     lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Background(lipgloss.Color("236")).Padding(0, 1),
		helpBar:       lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Background(lipgloss.Color("235")).Padding(0, 0),
		lineNum:       lipgloss.NewStyle().Foreground(lipgloss.Color("240")),
		highlightLine: lipgloss.NewStyle().Background(lipgloss.Color("58")).Foreground(lipgloss.Color("230")),
		separator:     lipgloss.NewStyle().Foreground(lipgloss.Color("238")),
		previewHeader: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("14")).Background(lipgloss.Color("236")).Padding(0, 1),
	}
}

// NewGraphModel creates a new graph visualization model
func NewGraphModel(tree *analyzer.Tree, scopes []*analyzer.ContextScope, rootPath string) GraphModel {
	m := GraphModel{
		tree:       tree,
		scopes:     scopes,
		rootPath:   rootPath,
		showRefs:   true,
		showScopes: true,
		keys:       defaultGraphKeyMap(),
		styles:     defaultGraphStyles(),
	}

	m.buildNodes()
	return m
}

// buildNodes constructs the flattened node list from the tree
func (m *GraphModel) buildNodes() {
	m.allNodes = nil

	if m.showScopes && len(m.scopes) > 0 {
		// Group by scopes
		for _, scope := range m.scopes {
			scopeNode := &GraphNode{
				Scope:    scope,
				IsScope:  true,
				Expanded: true,
				Depth:    0,
			}
			m.allNodes = append(m.allNodes, scopeNode)

			// Add files in this scope
			for _, configNode := range scope.Nodes {
				fileNode := m.buildFileNode(configNode, scopeNode, 1)
				scopeNode.Children = append(scopeNode.Children, fileNode)
				m.addVisibleNodes(fileNode)
			}
		}
	} else {
		// Show tree structure without scope grouping
		for _, child := range m.tree.Root.Children {
			fileNode := m.buildFileNode(child, nil, 0)
			m.allNodes = append(m.allNodes, fileNode)
			m.addVisibleNodes(fileNode)
		}
	}

	m.updateVisibleNodes()
}

func (m *GraphModel) buildFileNode(configNode *analyzer.ConfigNode, parent *GraphNode, depth int) *GraphNode {
	node := &GraphNode{
		Node:     configNode,
		Depth:    depth,
		Expanded: depth < 2, // Auto-expand first two levels
		Parent:   parent,
	}

	// Build map of resolved paths -> references (for file refs that resolved)
	refsByTarget := make(map[string][]analyzer.Reference)
	for _, ref := range configNode.References {
		if ref.Type == analyzer.RefTypeFile && ref.Resolved {
			refsByTarget[ref.Target] = append(refsByTarget[ref.Target], ref)
		}
	}

	// Add non-file references as children (URLs, tools, etc.) or unresolved file refs
	if m.showRefs {
		for _, ref := range configNode.References {
			// Skip resolved file refs - they'll be merged with the child node
			if ref.Type == analyzer.RefTypeFile && ref.Resolved {
				continue
			}
			refNode := &GraphNode{
				IsRef:      true,
				RefType:    ref.Type,
				RefValue:   ref.Value,
				RefContext: ref.Context,
				Depth:      depth + 1,
				Parent:     node,
			}
			node.Children = append(node.Children, refNode)
		}
	}

	// Add child files with merged reference metadata
	for _, childConfig := range configNode.Children {
		childNode := m.buildFileNode(childConfig, node, depth+1)

		// Attach reference sources from parent's refs that pointed to this child
		if refs, ok := refsByTarget[childConfig.Path]; ok {
			for _, ref := range refs {
				childNode.SourceRefs = append(childNode.SourceRefs, RefSource{
					Line:         ref.Source.Line,
					Column:       ref.Source.Column,
					OriginalText: ref.Value,
					Context:      ref.Context,
					Priority:     ref.Priority,
				})
			}
		}

		node.Children = append(node.Children, childNode)
	}

	return node
}

func (m *GraphModel) addVisibleNodes(node *GraphNode) {
	// This is called during initial build for non-scope view
}

func (m *GraphModel) updateVisibleNodes() {
	m.nodes = nil
	for _, node := range m.allNodes {
		m.collectVisible(node)
	}

	// Clamp cursor
	if m.cursor >= len(m.nodes) {
		m.cursor = len(m.nodes) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
}

func (m *GraphModel) collectVisible(node *GraphNode) {
	m.nodes = append(m.nodes, node)

	if node.Expanded {
		for _, child := range node.Children {
			m.collectVisible(child)
		}
	}
}

// Init initializes the model
func (m GraphModel) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (m GraphModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Quit):
			return m, tea.Quit

		case key.Matches(msg, m.keys.Up):
			if m.cursor > 0 {
				m.cursor--
			}

		case key.Matches(msg, m.keys.Down):
			if m.cursor < len(m.nodes)-1 {
				m.cursor++
			}

		case key.Matches(msg, m.keys.Left):
			if len(m.nodes) > 0 {
				m.nodes[m.cursor].Expanded = false
				m.updateVisibleNodes()
			}

		case key.Matches(msg, m.keys.Right), key.Matches(msg, m.keys.Toggle):
			if len(m.nodes) > 0 {
				m.nodes[m.cursor].Expanded = !m.nodes[m.cursor].Expanded
				m.updateVisibleNodes()
			}

		case key.Matches(msg, m.keys.ToggleRefs):
			m.showRefs = !m.showRefs
			m.buildNodes()

		case key.Matches(msg, m.keys.ToggleScopes):
			m.showScopes = !m.showScopes
			m.buildNodes()

		case key.Matches(msg, m.keys.TogglePreview):
			m.showPreview = !m.showPreview
			m.previewScroll = 0 // Reset scroll when toggling

		case key.Matches(msg, m.keys.PreviewUp):
			if m.showPreview && m.previewScroll > 0 {
				m.previewScroll -= 10
				if m.previewScroll < 0 {
					m.previewScroll = 0
				}
			}

		case key.Matches(msg, m.keys.PreviewDown):
			if m.showPreview {
				m.previewScroll += 10
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if !m.ready {
			m.viewport = viewport.New(msg.Width, msg.Height-4)
			m.viewport.YPosition = 2
			m.ready = true
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - 4
		}
	}

	return m, nil
}

// View renders the graph
func (m GraphModel) View() string {
	if !m.ready {
		return "Initializing..."
	}

	// Reserve space for footer (detail + help + padding)
	footerHeight := 4
	contentHeight := m.height - footerHeight
	if contentHeight < 5 {
		contentHeight = 5
	}

	var mainContent string

	if m.showPreview {
		// Split view: 50% tree, 50% preview
		treeWidth := m.width * 50 / 100
		previewWidth := m.width - treeWidth - 1 // -1 for separator

		treeView := m.renderTreePane(treeWidth, contentHeight)
		previewView := m.renderPreviewPanes(previewWidth, contentHeight)

		// Build separator column
		var sepLines []string
		for i := 0; i < contentHeight; i++ {
			sepLines = append(sepLines, m.styles.separator.Render("│"))
		}
		separator := strings.Join(sepLines, "\n")

		mainContent = lipgloss.JoinHorizontal(lipgloss.Top, treeView, separator, previewView)
	} else {
		// Single pane view
		mainContent = m.renderTreePane(m.width, contentHeight)
	}

	var sb strings.Builder
	sb.WriteString(mainContent)

	// Footer section
	sb.WriteString("\n")

	// Status bar with detail info (single line)
	if len(m.nodes) > 0 && m.cursor < len(m.nodes) {
		detail := m.renderDetailLine(m.nodes[m.cursor])
		sb.WriteString(m.styles.statusBar.Width(m.width).Render(detail))
	} else {
		sb.WriteString(m.styles.statusBar.Width(m.width).Render(""))
	}
	sb.WriteString("\n")

	// Help bar
	help := fmt.Sprintf(" ↑↓ navigate  ←→ collapse/expand  p preview(%s)  r refs(%s)  s scopes(%s)  q quit",
		boolToOnOff(m.showPreview),
		boolToOnOff(m.showRefs),
		boolToOnOff(m.showScopes),
	)
	sb.WriteString(m.styles.helpBar.Width(m.width).Render(help))

	return sb.String()
}

// renderTreePane renders the tree with a specific width constraint
func (m *GraphModel) renderTreePane(width, height int) string {
	treeContent := m.renderTree()
	lines := strings.Split(strings.TrimSuffix(treeContent, "\n"), "\n")

	// Scroll to keep cursor visible
	startIdx := 0
	if m.cursor >= height {
		startIdx = m.cursor - height + 1
	}

	endIdx := startIdx + height
	if endIdx > len(lines) {
		endIdx = len(lines)
	}

	// Truncate lines to width
	var visibleLines []string
	for i := startIdx; i < endIdx; i++ {
		line := lines[i]
		// Truncate if needed (accounting for ANSI codes is tricky, so just limit raw length)
		if len(line) > width {
			line = line[:width-1] + "…"
		}
		visibleLines = append(visibleLines, line)
	}

	// Pad to fill height
	for i := len(visibleLines); i < height; i++ {
		visibleLines = append(visibleLines, "")
	}

	return strings.Join(visibleLines, "\n")
}

// renderPreview renders the file preview pane
func (m *GraphModel) renderPreview(width, height int) string {
	if len(m.nodes) == 0 || m.cursor >= len(m.nodes) {
		return m.padLines([]string{m.styles.dim.Render("No selection")}, height)
	}

	node := m.nodes[m.cursor]

	// Can't preview scopes or refs
	if node.IsScope || node.IsRef || node.Node == nil {
		return m.padLines([]string{m.styles.dim.Render("No preview available")}, height)
	}

	// Get file content
	content := string(node.Node.Content)
	if content == "" {
		return m.padLines([]string{m.styles.dim.Render("Empty file")}, height)
	}

	lines := strings.Split(content, "\n")

	// Determine highlight line (from SourceRefs if available)
	highlightLine := -1
	if len(node.SourceRefs) > 0 {
		highlightLine = node.SourceRefs[0].Line - 1 // 0-indexed
	}

	// Calculate start line: auto-scroll to center highlighted line, or use manual scroll
	startLine := m.previewScroll
	if highlightLine >= 0 && m.previewScroll == 0 {
		// Auto-scroll to center the highlighted line
		startLine = highlightLine - height/2
		if startLine < 0 {
			startLine = 0
		}
	}

	// Clamp scroll
	maxScroll := len(lines) - height
	if maxScroll < 0 {
		maxScroll = 0
	}
	if startLine > maxScroll {
		startLine = maxScroll
	}

	// Render preview lines with line numbers
	var previewLines []string

	// Header showing filename
	relPath, _ := filepath.Rel(m.rootPath, node.Node.Path)
	header := m.styles.previewHeader.Width(width).Render(fmt.Sprintf(" %s ", relPath))
	previewLines = append(previewLines, header)

	// File content
	for i := startLine; i < startLine+height-1 && i < len(lines); i++ {
		lineNum := fmt.Sprintf("%4d ", i+1)
		lineContent := lines[i]

		// Truncate line if needed
		maxContentWidth := width - 6 // Account for line number
		if len(lineContent) > maxContentWidth {
			lineContent = lineContent[:maxContentWidth-1] + "…"
		}

		var renderedLine string
		if i == highlightLine {
			// Highlight this line
			renderedLine = m.styles.highlightLine.Render(lineNum + lineContent)
		} else {
			renderedLine = m.styles.lineNum.Render(lineNum) + lineContent
		}

		previewLines = append(previewLines, renderedLine)
	}

	return m.padLines(previewLines, height)
}

// renderPreviewPanes renders a two-pane preview: reference context (top) and file content (bottom)
func (m *GraphModel) renderPreviewPanes(width, height int) string {
	if len(m.nodes) == 0 || m.cursor >= len(m.nodes) {
		return m.padLines([]string{m.styles.dim.Render("No selection")}, height)
	}

	node := m.nodes[m.cursor]

	// Can't preview scopes or refs
	if node.IsScope || node.IsRef || node.Node == nil {
		return m.padLines([]string{m.styles.dim.Render("No preview available")}, height)
	}

	// Calculate pane heights: reference context gets ~25% (min 5 lines), file content gets rest
	refPaneHeight := height / 4
	if refPaneHeight < 5 {
		refPaneHeight = 5
	}
	if refPaneHeight > 10 {
		refPaneHeight = 10 // Cap it so file content has enough space
	}
	filePaneHeight := height - refPaneHeight - 1 // -1 for separator

	// Render reference context pane (top)
	refPane := m.renderRefContextPane(node, width, refPaneHeight)

	// Horizontal separator
	separator := m.styles.separator.Render(strings.Repeat("─", width))

	// Render file content pane (bottom)
	filePane := m.renderFileContentPane(node, width, filePaneHeight)

	return refPane + "\n" + separator + "\n" + filePane
}

// renderRefContextPane renders the reference context from the parent file
func (m *GraphModel) renderRefContextPane(node *GraphNode, width, height int) string {
	var lines []string

	// Header
	if len(node.SourceRefs) > 0 && node.Parent != nil && node.Parent.Node != nil {
		parentPath, _ := filepath.Rel(m.rootPath, node.Parent.Node.Path)
		header := m.styles.previewHeader.Width(width).Render(fmt.Sprintf(" Referenced from: %s ", parentPath))
		lines = append(lines, header)

		// Show the context around the reference
		ref := node.SourceRefs[0]
		if ref.Context != "" {
			contextLines := strings.Split(ref.Context, "\n")
			refLineInContext := 2 // Context is typically 2 lines before, ref line, 2 lines after

			for i, ctxLine := range contextLines {
				if i >= height-1 {
					break
				}

				// Calculate actual line number
				actualLineNum := ref.Line - refLineInContext + i
				if actualLineNum < 1 {
					actualLineNum = i + 1
				}

				lineNum := fmt.Sprintf("%4d ", actualLineNum)

				// Truncate if needed
				maxContentWidth := width - 6
				displayLine := ctxLine
				if len(displayLine) > maxContentWidth {
					displayLine = displayLine[:maxContentWidth-1] + "…"
				}

				// Highlight the reference line
				if i == refLineInContext {
					lines = append(lines, m.styles.highlightLine.Render(lineNum+displayLine))
				} else {
					lines = append(lines, m.styles.lineNum.Render(lineNum)+displayLine)
				}
			}
		} else {
			// No context, just show the reference text
			lines = append(lines, m.styles.dim.Render(fmt.Sprintf("  L:%d  %s", ref.Line, ref.OriginalText)))
		}
	} else {
		// No reference info (this is a root file)
		header := m.styles.previewHeader.Width(width).Render(" Entrypoint (no parent reference) ")
		lines = append(lines, header)
		lines = append(lines, m.styles.dim.Render("  This file is a root configuration entrypoint"))
	}

	return m.padLines(lines, height)
}

// renderFileContentPane renders the actual file content
func (m *GraphModel) renderFileContentPane(node *GraphNode, width, height int) string {
	var lines []string

	// Header showing filename
	relPath, _ := filepath.Rel(m.rootPath, node.Node.Path)
	header := m.styles.previewHeader.Width(width).Render(fmt.Sprintf(" %s ", relPath))
	lines = append(lines, header)

	// Get file content
	content := string(node.Node.Content)
	if content == "" {
		lines = append(lines, m.styles.dim.Render("  Empty file"))
		return m.padLines(lines, height)
	}

	fileLines := strings.Split(content, "\n")

	// Calculate start line based on scroll
	startLine := m.previewScroll
	maxScroll := len(fileLines) - (height - 1)
	if maxScroll < 0 {
		maxScroll = 0
	}
	if startLine > maxScroll {
		startLine = maxScroll
	}

	// Render file content with line numbers
	for i := startLine; i < startLine+height-1 && i < len(fileLines); i++ {
		lineNum := fmt.Sprintf("%4d ", i+1)
		lineContent := fileLines[i]

		// Truncate if needed
		maxContentWidth := width - 6
		if len(lineContent) > maxContentWidth {
			lineContent = lineContent[:maxContentWidth-1] + "…"
		}

		lines = append(lines, m.styles.lineNum.Render(lineNum)+lineContent)
	}

	return m.padLines(lines, height)
}

// padLines pads a slice of lines to fill the given height
func (m *GraphModel) padLines(lines []string, height int) string {
	for len(lines) < height {
		lines = append(lines, "")
	}
	return strings.Join(lines, "\n")
}

func (m *GraphModel) renderTree() string {
	var sb strings.Builder

	for i, node := range m.nodes {
		line := m.renderNode(node, i == m.cursor)
		sb.WriteString(line)
		sb.WriteString("\n")
	}

	return sb.String()
}

func (m *GraphModel) renderNode(node *GraphNode, selected bool) string {
	var sb strings.Builder

	// Indentation
	indent := strings.Repeat("  ", node.Depth)
	sb.WriteString(m.styles.tree.Render(indent))

	// Tree connector
	connector := "├─ "
	if node.Parent != nil {
		isLast := true
		for i, sibling := range node.Parent.Children {
			if sibling == node && i < len(node.Parent.Children)-1 {
				isLast = false
				break
			}
		}
		if isLast {
			connector = "└─ "
		}
	} else if node.IsScope {
		connector = ""
	}
	sb.WriteString(m.styles.tree.Render(connector))

	// Expand/collapse indicator
	if len(node.Children) > 0 {
		if node.Expanded {
			sb.WriteString(m.styles.dim.Render("▼ "))
		} else {
			sb.WriteString(m.styles.dim.Render("▶ "))
		}
	} else {
		sb.WriteString("  ")
	}

	// Content
	var content string
	if node.IsScope {
		icon := "📦 "
		style := m.styles.scopeMain
		if node.Scope.Type == analyzer.ScopeTypeSubagent {
			icon = "🤖 "
			style = m.styles.scopeSub
		}
		content = icon + style.Render(fmt.Sprintf("[%s] %s", node.Scope.Type.String(), node.Scope.Name))
		if node.Scope.Entrypoint != "" {
			relPath, _ := filepath.Rel(m.rootPath, node.Scope.Entrypoint)
			content += m.styles.dim.Render(fmt.Sprintf(" (%s)", relPath))
		}
	} else if node.IsRef {
		icon := m.refIcon(node.RefType)
		style := m.refStyle(node.RefType)
		content = icon + " " + style.Render(node.RefValue)
	} else if node.Node != nil {
		relPath, err := filepath.Rel(m.rootPath, node.Node.Path)
		if err != nil {
			relPath = node.Node.Path
		}
		icon := m.fileIcon(node.Node)
		content = icon + " " + m.styles.file.Render(relPath)

		// Show line number where this file was referenced (from SourceRefs)
		if len(node.SourceRefs) > 0 {
			content += m.styles.dim.Render(fmt.Sprintf(" (L:%d)", node.SourceRefs[0].Line))
		}

		// Show reference count if refs are hidden
		if len(node.Node.References) > 0 && !m.showRefs {
			content += m.styles.dim.Render(fmt.Sprintf(" [%d refs]", len(node.Node.References)))
		}
	}

	if selected {
		content = m.styles.selected.Render(content)
	}
	sb.WriteString(content)

	return sb.String()
}

func (m *GraphModel) refIcon(rt analyzer.RefType) string {
	switch rt {
	case analyzer.RefTypeFile:
		return "📄"
	case analyzer.RefTypeURL:
		return "🔗"
	case analyzer.RefTypeTool:
		return "🔧"
	case analyzer.RefTypeSubagent:
		return "🤖"
	case analyzer.RefTypeSkill:
		return "⚡"
	case analyzer.RefTypeMCPServer:
		return "🔌"
	default:
		return "❓"
	}
}

func (m *GraphModel) refStyle(rt analyzer.RefType) lipgloss.Style {
	switch rt {
	case analyzer.RefTypeFile:
		return m.styles.refFile
	case analyzer.RefTypeURL:
		return m.styles.refURL
	case analyzer.RefTypeTool:
		return m.styles.refTool
	case analyzer.RefTypeSubagent:
		return m.styles.refSubagent
	case analyzer.RefTypeSkill:
		return m.styles.refSkill
	case analyzer.RefTypeMCPServer:
		return m.styles.refMCP
	default:
		return m.styles.dim
	}
}

func (m *GraphModel) fileIcon(node *analyzer.ConfigNode) string {
	if node.Parsed == nil {
		return "📄"
	}

	base := filepath.Base(node.Path)
	switch {
	case strings.HasPrefix(base, "CLAUDE"):
		return "📋"
	case strings.HasSuffix(base, ".json"):
		return "⚙️"
	case strings.HasSuffix(base, ".yaml") || strings.HasSuffix(base, ".yml"):
		return "📝"
	case strings.HasSuffix(base, ".md"):
		return "📖"
	default:
		return "📄"
	}
}

func (m *GraphModel) renderDetailLine(node *GraphNode) string {
	if node.IsScope {
		return fmt.Sprintf(" Scope: %s (%s)  Files: %d",
			node.Scope.Name, node.Scope.Type.String(), len(node.Scope.FilePaths))
	} else if node.IsRef {
		return fmt.Sprintf(" %s  Type: %s", node.RefValue, node.RefType.String())
	} else if node.Node != nil {
		category := "unknown"
		if node.Node.Parsed != nil {
			category = node.Node.Parsed.Category.String()
		}

		// Basic file info
		info := fmt.Sprintf(" %s  Category: %s", node.Node.Path, category)

		// Add reference source info if available
		if len(node.SourceRefs) > 0 {
			ref := node.SourceRefs[0]
			info += fmt.Sprintf("  Referenced at L:%d as \"%s\"", ref.Line, ref.OriginalText)
		}

		return info
	}
	return ""
}

func boolToOnOff(b bool) string {
	if b {
		return "on"
	}
	return "off"
}
