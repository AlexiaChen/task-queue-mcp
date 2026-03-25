package tui

import "github.com/charmbracelet/lipgloss"

var (
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#5B21B6")).
			PaddingLeft(2).
			PaddingRight(2).
			PaddingTop(1).
			PaddingBottom(1)

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF"))

	selectedItemStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("#3730A3")).
				Foreground(lipgloss.Color("#FFFFFF")).
				Bold(true)

	normalItemStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#D1D5DB"))

	pendingBadge = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FCD34D")).
			Bold(true)

	doingBadge = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#67E8F9")).
			Bold(true)

	finishedBadge = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6EE7B7")).
			Bold(true)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F87171")).
			Bold(true)

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#34D399")).
			Bold(true)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6B7280"))

	borderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#4B5563")).
			Padding(0, 1)

	labelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#9CA3AF"))

	dimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#4B5563"))
)
