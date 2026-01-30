package ui

import "github.com/charmbracelet/lipgloss/v2"

var (
	// Title style for section headers.
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#7C3AED"))

	// Subtle style for secondary information.
	SubtleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6B7280"))

	// Success style for positive values / success messages.
	SuccessStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#10B981"))

	// Error style for negative values / error messages.
	ErrorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#EF4444"))

	// Info style for informational text.
	InfoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#3B82F6"))
)
