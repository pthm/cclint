package ui

import (
	"github.com/charmbracelet/lipgloss"
)

// Styles contains all lipgloss styles for terminal output
type Styles struct {
	enabled bool

	// Severity styles
	Error      lipgloss.Style
	Warning    lipgloss.Style
	Suggestion lipgloss.Style
	Info       lipgloss.Style
	Success    lipgloss.Style

	// Structural styles
	Header    lipgloss.Style
	Subheader lipgloss.Style
	Path      lipgloss.Style
	Rule      lipgloss.Style
	Separator lipgloss.Style

	// Icons (degraded to ASCII when not interactive)
	IconError      string
	IconWarning    string
	IconSuggestion string
	IconInfo       string
	IconSuccess    string
}

// NewStyles creates a new Styles instance
// When enabled is false, styles return text unchanged (for non-TTY output)
func NewStyles(enabled bool) *Styles {
	s := &Styles{enabled: enabled}

	if enabled {
		// Severity styles
		s.Error = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))       // Red
		s.Warning = lipgloss.NewStyle().Foreground(lipgloss.Color("11"))    // Yellow
		s.Suggestion = lipgloss.NewStyle().Foreground(lipgloss.Color("14")) // Cyan
		s.Info = lipgloss.NewStyle().Foreground(lipgloss.Color("12"))       // Blue
		s.Success = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))    // Green

		// Structural styles
		s.Header = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("15"))    // White bold
		s.Subheader = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))             // Gray
		s.Path = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))                  // Gray
		s.Rule = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))                  // Gray
		s.Separator = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))             // Gray

		// Unicode icons
		s.IconError = "\u2717"      // ✗
		s.IconWarning = "\u26a0"    // ⚠
		s.IconSuggestion = "\U0001f4a1" // 💡
		s.IconInfo = "\u2139"       // ℹ
		s.IconSuccess = "\u2713"    // ✓
	} else {
		// No-op styles for non-TTY (plain text output)
		s.Error = lipgloss.NewStyle()
		s.Warning = lipgloss.NewStyle()
		s.Suggestion = lipgloss.NewStyle()
		s.Info = lipgloss.NewStyle()
		s.Success = lipgloss.NewStyle()

		s.Header = lipgloss.NewStyle()
		s.Subheader = lipgloss.NewStyle()
		s.Path = lipgloss.NewStyle()
		s.Rule = lipgloss.NewStyle()
		s.Separator = lipgloss.NewStyle()

		// ASCII fallback icons
		s.IconError = "ERROR:"
		s.IconWarning = "WARN:"
		s.IconSuggestion = "HINT:"
		s.IconInfo = "INFO:"
		s.IconSuccess = "OK:"
	}

	return s
}

// Enabled returns whether styling is enabled
func (s *Styles) Enabled() bool {
	return s.enabled
}
