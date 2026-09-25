package tui

import "github.com/charmbracelet/lipgloss"

// Slack-inspired palette. Every colour is adaptive so the UI stays readable on
// both light and dark terminals.
var (
	colInk    = lipgloss.AdaptiveColor{Light: "#1D1C1D", Dark: "#E8E8E8"}
	colGrey   = lipgloss.AdaptiveColor{Light: "#616061", Dark: "#A8A8A8"}
	colBorder = lipgloss.AdaptiveColor{Light: "#C9C9C9", Dark: "#4A4A4A"}
	colLabel  = lipgloss.AdaptiveColor{Light: "#4A154B", Dark: "#E3B0E8"}
	colBlue   = lipgloss.AdaptiveColor{Light: "#1264A3", Dark: "#79B8E8"}
	colGreen  = lipgloss.AdaptiveColor{Light: "#1F8A5B", Dark: "#5BD69A"}
	colYellow = lipgloss.AdaptiveColor{Light: "#8A6D00", Dark: "#ECB22E"}
	colRed    = lipgloss.AdaptiveColor{Light: "#C0392B", Dark: "#FF7A96"}
	colFaint  = lipgloss.AdaptiveColor{Light: "#DDDDDD", Dark: "#555555"}

	// Solid background colours (used with white text, so fixed).
	colTitleBg    = lipgloss.Color("#4A154B")
	colSelectedBg = lipgloss.Color("#1264A3")
)

var (
	styleTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(colTitleBg).
			Padding(0, 1)

	styleSubtle = lipgloss.NewStyle().Foreground(colGrey)

	styleLabel = lipgloss.NewStyle().Foreground(colLabel).Bold(true)

	styleValue = lipgloss.NewStyle().Foreground(colInk)

	styleSelected = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(colSelectedBg).
			Padding(0, 1)

	styleCursor = lipgloss.NewStyle().Foreground(colInk)

	styleRowSelected = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FFFFFF")).
				Background(colSelectedBg)

	styleSuccess = lipgloss.NewStyle().Foreground(colGreen)
	styleWarning = lipgloss.NewStyle().Foreground(colYellow)
	styleError   = lipgloss.NewStyle().Foreground(colRed)
	styleAccent  = lipgloss.NewStyle().Foreground(colBlue)
	styleFaint   = lipgloss.NewStyle().Foreground(colFaint)

	styleCard = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colBorder).
			Padding(1, 2)
)
