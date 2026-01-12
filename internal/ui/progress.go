package ui

import (
	"fmt"
	"io"

	tea "github.com/charmbracelet/bubbletea"
)

// ProgressController manages the bubbletea program for progress display
type ProgressController struct {
	ui      *UI
	program *tea.Program
}

// StartProgress starts the progress display if in interactive mode
// Returns nil if not in interactive mode
func (ui *UI) StartProgress() *ProgressController {
	if ui.Mode != OutputModeInteractive {
		return nil
	}

	m := NewModel()
	p := tea.NewProgram(m, tea.WithOutput(ui.ErrWriter))

	ctrl := &ProgressController{
		ui:      ui,
		program: p,
	}

	// Run the program in a goroutine
	go func() {
		if _, err := p.Run(); err != nil {
			// Silently handle program errors
			_ = err
		}
	}()

	return ctrl
}

// SetStage updates the current stage
func (pc *ProgressController) SetStage(stage Stage) {
	if pc != nil && pc.program != nil {
		pc.program.Send(StageMsg(stage))
	}
}

// SetOperation updates the current operation description
func (pc *ProgressController) SetOperation(op string) {
	if pc != nil && pc.program != nil {
		pc.program.Send(OperationMsg(op))
	}
}

// SetRuleCount sets the total number of rules to run
func (pc *ProgressController) SetRuleCount(count int) {
	if pc != nil && pc.program != nil {
		pc.program.Send(RuleCountMsg(count))
	}
}

// RuleStart indicates a rule has started
func (pc *ProgressController) RuleStart(name string) {
	if pc != nil && pc.program != nil {
		pc.program.Send(RuleStartMsg(fmt.Sprintf("Running %s...", name)))
	}
}

// RuleDone indicates a rule has completed
func (pc *ProgressController) RuleDone() {
	if pc != nil && pc.program != nil {
		pc.program.Send(RuleDoneMsg{})
	}
}

// Done signals that all work is complete
func (pc *ProgressController) Done(err error) {
	if pc != nil && pc.program != nil {
		pc.program.Send(DoneMsg{Err: err})
		pc.program.Wait()
	}
}

// SimpleSpinner provides a simple spinner for short operations
// without the full progress tracking
type SimpleSpinner struct {
	ui      *UI
	program *tea.Program
	done    chan struct{}
}

// simpleSpinnerModel is a minimal model for just showing a spinner
type simpleSpinnerModel struct {
	message  string
	quitting bool
}

func (m simpleSpinnerModel) Init() tea.Cmd {
	return nil
}

func (m simpleSpinnerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			m.quitting = true
			return m, tea.Quit
		}
	case DoneMsg:
		m.quitting = true
		return m, tea.Quit
	}
	return m, nil
}

func (m simpleSpinnerModel) View() string {
	if m.quitting {
		return ""
	}
	return fmt.Sprintf("  %s", m.message)
}

// StartSimpleSpinner starts a simple spinner with a message
func (ui *UI) StartSimpleSpinner(w io.Writer, message string) *SimpleSpinner {
	if ui.Mode != OutputModeInteractive {
		// In non-interactive mode, just print the message
		fmt.Fprintf(w, "%s\n", message)
		return nil
	}

	m := simpleSpinnerModel{message: message}
	p := tea.NewProgram(m, tea.WithOutput(w))

	ss := &SimpleSpinner{
		ui:      ui,
		program: p,
		done:    make(chan struct{}),
	}

	go func() {
		if _, err := p.Run(); err != nil {
			_ = err
		}
		close(ss.done)
	}()

	return ss
}

// Stop stops the simple spinner
func (ss *SimpleSpinner) Stop() {
	if ss != nil && ss.program != nil {
		ss.program.Send(DoneMsg{})
		<-ss.done
	}
}
