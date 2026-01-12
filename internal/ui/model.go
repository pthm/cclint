package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Stage represents the current stage of analysis
type Stage int

const (
	StageLoadConfig Stage = iota
	StageBuildTree
	StageRunRules
	StageDone
)

// Message types for updating the model
type (
	StageMsg     Stage
	OperationMsg string
	RuleStartMsg string
	RuleDoneMsg  struct{}
	DoneMsg      struct{ Err error }
	RuleCountMsg int
)

// Model is the Bubbletea model for progress display
type Model struct {
	stage     Stage
	spinner   spinner.Model
	progress  progress.Model
	currentOp string
	ruleCount int
	rulesDone int
	width     int
	quitting  bool
	err       error
}

// NewModel creates a new progress model
func NewModel() Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	p := progress.New(progress.WithDefaultGradient())

	return Model{
		stage:    StageLoadConfig,
		spinner:  s,
		progress: p,
	}
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return m.spinner.Tick
}

// Update handles messages and updates the model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			m.quitting = true
			return m, tea.Quit
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.progress.Width = msg.Width - 4
		if m.progress.Width > 60 {
			m.progress.Width = 60
		}
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case StageMsg:
		m.stage = Stage(msg)
		return m, nil

	case OperationMsg:
		m.currentOp = string(msg)
		return m, nil

	case RuleStartMsg:
		m.currentOp = string(msg)
		return m, nil

	case RuleCountMsg:
		m.ruleCount = int(msg)
		return m, nil

	case RuleDoneMsg:
		m.rulesDone++
		return m, nil

	case DoneMsg:
		m.err = msg.Err
		m.quitting = true
		return m, tea.Quit
	}

	return m, nil
}

// View renders the model
func (m Model) View() string {
	if m.quitting {
		return ""
	}

	var sb strings.Builder

	switch m.stage {
	case StageLoadConfig:
		sb.WriteString(m.spinner.View())
		sb.WriteString(" Loading agent configuration...")

	case StageBuildTree:
		sb.WriteString(m.spinner.View())
		sb.WriteString(" Building reference tree")
		if m.currentOp != "" {
			sb.WriteString(fmt.Sprintf(" (%s)", m.currentOp))
		}

	case StageRunRules:
		if m.ruleCount > 0 {
			pct := float64(m.rulesDone) / float64(m.ruleCount)
			sb.WriteString(m.progress.ViewAs(pct))
			sb.WriteString("\n")
		}
		sb.WriteString(m.spinner.View())
		sb.WriteString(" ")
		if m.currentOp != "" {
			sb.WriteString(m.currentOp)
		} else {
			sb.WriteString("Running rules...")
		}
	}

	return sb.String()
}
