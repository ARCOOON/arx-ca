package tui

import "github.com/charmbracelet/lipgloss"

var (
	colorBg      = lipgloss.Color("#1a1b26")
	colorSurface = lipgloss.Color("#24283b")
	colorBorder  = lipgloss.Color("#414868")
	colorMuted   = lipgloss.Color("#565f89")
	colorText    = lipgloss.Color("#c0caf5")
	colorAccent  = lipgloss.Color("#7aa2f7")
	colorOk      = lipgloss.Color("#9ece6a")
	colorWarn    = lipgloss.Color("#e0af68")
	colorErr     = lipgloss.Color("#f7768e")

	styleApp = lipgloss.NewStyle().
			Background(colorBg).
			Foreground(colorText).
			Padding(0, 1)

	styleTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorAccent).
			MarginBottom(1)

	styleTab = lipgloss.NewStyle().
			Foreground(colorMuted).
			Padding(0, 2)

	styleTabActive = styleTab.Copy().
			Foreground(colorText).
			Bold(true).
			Background(colorSurface).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorAccent)

	stylePanel = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorBorder).
			Padding(1, 2).
			Width(72)

	styleLabel = lipgloss.NewStyle().
			Foreground(colorMuted).
			Width(16)

	styleValue = lipgloss.NewStyle().
			Foreground(colorText)

	styleStatusOK   = lipgloss.NewStyle().Foreground(colorOk).Bold(true)
	styleStatusWarn = lipgloss.NewStyle().Foreground(colorWarn).Bold(true)
	styleStatusErr  = lipgloss.NewStyle().Foreground(colorErr).Bold(true)

	styleRow         = lipgloss.NewStyle().Padding(0, 1)
	styleRowSelected = styleRow.Copy().
				Background(colorSurface).
				Foreground(colorAccent).
				Bold(true)

	styleHelp = lipgloss.NewStyle().
			Foreground(colorMuted).
			MarginTop(1)

	styleFlashOK  = lipgloss.NewStyle().Foreground(colorOk).Bold(true)
	styleFlashErr = lipgloss.NewStyle().Foreground(colorErr).Bold(true)
)
