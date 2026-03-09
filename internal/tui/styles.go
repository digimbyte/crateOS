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

	headerTitle = lipgloss.NewStyle().
			Foreground(colorPrimary).
			Bold(true)

	headerMeta = lipgloss.NewStyle().
			Foreground(colorSecondary).
			Bold(true)

	headerSubmeta = lipgloss.NewStyle().
			Foreground(colorDim)

	viewBox = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorPrimary).
		Padding(0, 1)

	panelBox = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorPrimary).
			Padding(0, 1).
			MarginRight(1)

	panelActive = lipgloss.NewStyle().
			Border(lipgloss.DoubleBorder()).
			BorderForeground(colorPrimary).
			Foreground(colorBright).
			Padding(0, 1)

	menuItem = lipgloss.NewStyle().
			Foreground(colorDim).
			PaddingLeft(4)

	menuActive = lipgloss.NewStyle().
			Foreground(colorPrimary).
			Bold(true).
			PaddingLeft(2)

	highlight = lipgloss.NewStyle().
			Foreground(colorBright).
			Background(colorPrimary).
			Bold(true)

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

	panelTitle = lipgloss.NewStyle().
			Foreground(colorSecondary).
			Bold(true)

	railTitleBar = lipgloss.NewStyle().
			Foreground(colorPrimary).
			Bold(true)

	railTitleMeta = lipgloss.NewStyle().
			Foreground(colorDim).
			Bold(true)

	selectorIdle = lipgloss.NewStyle().
			Foreground(colorDim).
			PaddingLeft(1)

	selectorActive = lipgloss.NewStyle().
			Foreground(colorBright).
			Background(colorPrimary).
			Bold(true).
			Padding(0, 1)

	panelTitleBar = lipgloss.NewStyle().
			Foreground(colorBright).
			Background(colorPrimary).
			Bold(true).
			Padding(0, 1)

	panelTitleMeta = lipgloss.NewStyle().
			Foreground(colorDim).
			Bold(true)
	subsectionCard = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorDim).
			Padding(0, 1).
			MarginTop(1)

	subsectionCardSummary = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(colorPrimary).
				Padding(0, 1).
				MarginTop(1)

	subsectionCardWarning = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(colorDanger).
				Padding(0, 1).
				MarginTop(1)

	subsectionCardAction = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(colorSecondary).
				Padding(0, 1).
				MarginTop(1)

	subsectionTitle = lipgloss.NewStyle().
			Foreground(colorSecondary).
			Bold(true)

	commandStrip = lipgloss.NewStyle().
			Foreground(colorBright).
			Border(lipgloss.NormalBorder()).
			BorderForeground(colorSecondary).
			Padding(0, 1).
			MarginTop(1)

	shellFrame = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(colorDim).
			Padding(0, 1)

	shellLabel = lipgloss.NewStyle().
			Foreground(colorSecondary).
			Bold(true)

	commandBusLabel = lipgloss.NewStyle().
			Foreground(colorSecondary).
			Bold(true)

	commandBusKey = lipgloss.NewStyle().
			Foreground(colorPrimary).
			Bold(true)

	commandBusSep = lipgloss.NewStyle().
			Foreground(colorDim)

	commandPromptLabel = lipgloss.NewStyle().
				Foreground(colorSecondary).
				Bold(true)

	commandPromptSep = lipgloss.NewStyle().
				Foreground(colorDim)

	commandPromptInput = lipgloss.NewStyle().
				Foreground(colorBright)

	commandPromptCursor = lipgloss.NewStyle().
				Foreground(colorPrimary).
				Bold(true)

	commandPromptHint = lipgloss.NewStyle().
				Foreground(colorDim)

	commandStatusInfo = lipgloss.NewStyle().
				Foreground(colorSecondary)

	commandStatusOK = lipgloss.NewStyle().
			Foreground(colorPrimary)

	commandStatusWarn = lipgloss.NewStyle().
				Foreground(colorSecondary).
				Bold(true)

	commandStatusError = lipgloss.NewStyle().
				Foreground(colorDanger).
				Bold(true)

	statStrip = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(colorDim).
			Padding(0, 1).
			MarginTop(1)
)
