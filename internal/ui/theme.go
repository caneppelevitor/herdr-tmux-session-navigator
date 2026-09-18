package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// Palette follows the tmux choose-tree reference: dim slate chrome, amber
// selection bar, muted greens/yellows for status.
var (
	cFg      = lipgloss.Color("#d4d4d4")
	cDim     = lipgloss.Color("#7c8377")
	cBorder  = lipgloss.Color("#4a5057")
	cAccent  = lipgloss.Color("#89b482")
	cName    = lipgloss.Color("#e8d4a0")
	cSelBg   = lipgloss.Color("#d8a657")
	cSelFg   = lipgloss.Color("#1d2021")
	cWorking = lipgloss.Color("#d8a657")
	cIdle    = lipgloss.Color("#89b482")
	cBlocked = lipgloss.Color("#ea6962")
)

func dimf(format string, a ...any) string {
	return lipgloss.NewStyle().Foreground(cDim).Render(fmt.Sprintf(format, a...))
}

// statusDot maps herdr agent status to a coloured marker.
func statusDot(status string) string {
	switch status {
	case "working":
		return lipgloss.NewStyle().Foreground(cWorking).Render("*")
	case "idle":
		return lipgloss.NewStyle().Foreground(cIdle).Render("*")
	case "blocked":
		return lipgloss.NewStyle().Foreground(cBlocked).Render("*")
	default:
		return lipgloss.NewStyle().Foreground(cDim).Render("-")
	}
}

// truncate cuts to a printable-cell width, preserving ANSI styling. Pane bodies
// carry their own escape sequences, so a byte-wise cut would corrupt them.
func truncate(s string, w int) string {
	if w <= 0 {
		return ""
	}
	s = strings.ReplaceAll(s, "\t", "    ")
	if ansi.StringWidth(s) <= w {
		return s
	}
	return ansi.Truncate(s, w, "")
}

// padTo right-pads to an exact cell width, ANSI-aware.
func padTo(s string, w int) string {
	cur := ansi.StringWidth(s)
	if cur >= w {
		return truncate(s, w)
	}
	return s + strings.Repeat(" ", w-cur)
}
