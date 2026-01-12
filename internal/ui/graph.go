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
}

// GraphModel is the bubbletea model for graph visualization
type GraphModel struct {
	tree       *analyzer.Tree
	scopes     []*analyzer.ContextScope
	rootPath   string
	nodes      []*GraphNode    // Flattened list of visible nodes
	allNodes   []*GraphNode    // All nodes including collapsed ones
	cursor     int             // Currently selected node
	viewport   viewport.Model
	ready      bool
	width      int
	height     int
	showRefs   bool            // Toggle to show/hide references
	showScopes bool            // Toggle to show scope grouping
	keys       graphKeyMap
	styles     graphStyles
}

type graphKeyMap struct {
	Up       key.Binding
	Down     key.Binding
	Left     key.Binding
	Right    key.Binding
	Toggle   key.Binding
	ToggleRefs key.Binding
	ToggleScopes key.Binding
	Quit     key.Binding
	Help     key.Binding
}

type graphStyles struct {
	selected    lipgloss.Style
	file        lipgloss.Style
	scope       lipgloss.Style
	scopeMain   lipgloss.Style
	scopeSub    lipgloss.Style
	refFile     lipgloss.Style
	refURL      lipgloss.Style
	refTool     lipgloss.Style
	refSubagent lipgloss.Style
	refSkill    lipgloss.Style
	refMCP      lipgloss.Style
	tree        lipgloss.Style
	dim         lipgloss.Style
	detail      lipgloss.Style
	header      lipgloss.Style
	help        lipgloss.Style
	statusBar   lipgloss.Style
	helpBar     lipgloss.Style
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
		selected:    lipgloss.NewStyle().Background(lipgloss.Color("237")).Bold(true),
		file:        lipgloss.NewStyle().Foreground(lipgloss.Color("15")),
		scope:       lipgloss.NewStyle().Foreground(lipgloss.Color("14")).Bold(true),
		scopeMain:   lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true),
		scopeSub:    lipgloss.NewStyle().Foreground(lipgloss.Color("13")).Bold(true),
		refFile:     lipgloss.NewStyle().Foreground(lipgloss.Color("12")),
		refURL:      lipgloss.NewStyle().Foreground(lipgloss.Color("11")),
		refTool:     lipgloss.NewStyle().Foreground(lipgloss.Color("9")),
		refSubagent: lipgloss.NewStyle().Foreground(lipgloss.Color("13")),
		refSkill:    lipgloss.NewStyle().Foreground(lipgloss.Color("14")),
		refMCP:      lipgloss.NewStyle().Foreground(lipgloss.Color("208")),
		tree:        lipgloss.NewStyle().Foreground(lipgloss.Color("240")),
		dim:         lipgloss.NewStyle().Foreground(lipgloss.Color("242")),
		detail:      lipgloss.NewStyle().Foreground(lipgloss.Color("250")),
		header:      lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("15")).Background(lipgloss.Color("236")).Padding(0, 1),
		help:        lipgloss.NewStyle().Foreground(lipgloss.Color("241")),
		statusBar:   lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Background(lipgloss.Color("236")).Padding(0, 1),
		helpBar:     lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Background(lipgloss.Color("235")).Padding(0, 0),
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

	// Add references as children if enabled
	if m.showRefs {
		for _, ref := range configNode.References {
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

	// Add child files
	for _, childConfig := range configNode.Children {
		childNode := m.buildFileNode(childConfig, node, depth+1)
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
	treeHeight := m.height - footerHeight
	if treeHeight < 5 {
		treeHeight = 5
	}

	var sb strings.Builder

	// Tree view
	treeContent := m.renderTree()
	lines := strings.Split(strings.TrimSuffix(treeContent, "\n"), "\n")

	// Scroll to keep cursor visible
	startIdx := 0
	if m.cursor >= treeHeight {
		startIdx = m.cursor - treeHeight + 1
	}

	endIdx := startIdx + treeHeight
	if endIdx > len(lines) {
		endIdx = len(lines)
	}

	// Render visible tree lines
	if startIdx < len(lines) {
		visibleLines := lines[startIdx:endIdx]
		sb.WriteString(strings.Join(visibleLines, "\n"))
	}

	// Pad tree area to maintain consistent height
	renderedLines := 0
	if endIdx > startIdx {
		renderedLines = endIdx - startIdx
	}
	for i := renderedLines; i < treeHeight; i++ {
		sb.WriteString("\n")
	}

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
	help := fmt.Sprintf(" ↑↓ navigate  ←→ collapse/expand  r refs(%s)  s scopes(%s)  q quit",
		boolToOnOff(m.showRefs),
		boolToOnOff(m.showScopes),
	)
	sb.WriteString(m.styles.helpBar.Width(m.width).Render(help))

	return sb.String()
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

		// Show reference count
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
		return fmt.Sprintf(" %s  Refs: %d  Children: %d  Category: %s",
			node.Node.Path, len(node.Node.References), len(node.Node.Children), category)
	}
	return ""
}

func boolToOnOff(b bool) string {
	if b {
		return "on"
	}
	return "off"
}
