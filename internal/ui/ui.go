package ui

import (
	"io"
	"os"

	"golang.org/x/term"
)

// OutputMode determines how output should be formatted
type OutputMode int

const (
	// OutputModeInteractive enables full colors, spinners, and progress bars
	OutputModeInteractive OutputMode = iota
	// OutputModePlain disables colors and progress (for piped output)
	OutputModePlain
	// OutputModeJSON outputs raw JSON only
	OutputModeJSON
)

// UI provides a unified interface for terminal output with TTY detection
type UI struct {
	Mode      OutputMode
	Writer    io.Writer
	ErrWriter io.Writer
	Styles    *Styles
}

// New creates a new UI instance with automatic TTY detection
func New(w, errW io.Writer, format string) *UI {
	mode := detectMode(w, format)
	return &UI{
		Mode:      mode,
		Writer:    w,
		ErrWriter: errW,
		Styles:    NewStyles(mode == OutputModeInteractive),
	}
}

// detectMode determines the output mode based on TTY and format flags
func detectMode(w io.Writer, format string) OutputMode {
	if format == "json" {
		return OutputModeJSON
	}

	// Check if stdout is a terminal
	if f, ok := w.(*os.File); ok {
		if term.IsTerminal(int(f.Fd())) {
			return OutputModeInteractive
		}
	}

	return OutputModePlain
}

// IsInteractive returns true if the output is interactive (TTY)
func (ui *UI) IsInteractive() bool {
	return ui.Mode == OutputModeInteractive
}

// IsJSON returns true if JSON output mode is enabled
func (ui *UI) IsJSON() bool {
	return ui.Mode == OutputModeJSON
}
