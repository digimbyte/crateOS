package tui

import "github.com/charmbracelet/lipgloss"

// ── Pitboy/DOS color palette ────────────────────────────────────────

var (
	colorPrimary   = lipgloss.Color("#00FF41") // terminal green
	colorSecondary = lipgloss.Color("#FFB000") // amber
	colorDim       = lipgloss.Color("#555555")
	colorBright    = lipgloss.Color("#EEEEEE")
	colorDanger    = lipgloss.Color("#FF3333")
)

// ── Reusable styles ─────────────────────────────────────────────────

var (
	headerBox = lipgloss.NewStyle().
			Border(lipgloss.DoubleBorder()).
			BorderForeground(colorPrimary).
			Foreground(colorPrimary).
			Bold(true).
			Padding(0, 2).
			Align(lipgloss.Center)

	viewBox = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorPrimary).
		Padding(0, 1)

	menuItem = lipgloss.NewStyle().
			Foreground(colorDim).
			PaddingLeft(4)

	menuActive = lipgloss.NewStyle().
			Foreground(colorPrimary).
			Bold(true).
			PaddingLeft(2)

	label = lipgloss.NewStyle().
		Foreground(colorSecondary).
		Width(16)

	value = lipgloss.NewStyle().
		Foreground(colorBright)

	dim = lipgloss.NewStyle().
		Foreground(colorDim)

	ok = lipgloss.NewStyle().
		Foreground(colorPrimary)

	warn = lipgloss.NewStyle().
		Foreground(colorSecondary)

	danger = lipgloss.NewStyle().
		Foreground(colorDanger)

	section = lipgloss.NewStyle().
		Foreground(colorSecondary).
		Bold(true).
		MarginTop(1)

	footer = lipgloss.NewStyle().
		Foreground(colorDim).
		MarginTop(1)
)
