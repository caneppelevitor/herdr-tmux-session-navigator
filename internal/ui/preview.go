package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/caneppelevitor/herdr-tmux-session-navigator/internal/herdr"
)

// Panel is one column in the preview strip.
type Panel struct {
	Title string
	Body  string
}

// PanelsFor decides what the preview shows for a row:
//   - workspace row -> one panel per tab (its active pane's screen)
//   - tab row       -> one panel per pane in that tab
func PanelsFor(c *herdr.Client, row Row, tabs map[string][]herdr.Tab,
	panes map[string][]herdr.Pane, lines int) []Panel {

	wsID := row.WorkspaceID()
	all := panes[wsID]

	pick := func(tabID string) []herdr.Pane {
		var out []herdr.Pane
		for _, p := range all {
			if p.TabID == tabID {
				out = append(out, p)
			}
		}
		return out
	}

	var panels []Panel

	if row.Kind == RowWorkspace {
		for _, t := range tabs[wsID] {
			tp := pick(t.ID)
			if len(tp) == 0 {
				continue
			}
			title := t.Label
			if title == "" {
				title = t.ID
			}
			body, _ := c.ReadPane(tp[0].ID, lines)
			if len(tp) > 1 {
				title += dimf(" +%d", len(tp)-1)
			}
			panels = append(panels, Panel{Title: title, Body: body})
		}
	} else {
		for _, p := range pick(row.Tab.ID) {
			title := p.Label
			if title == "" {
				title = p.ID
			}
			body, _ := c.ReadPane(p.ID, lines)
			panels = append(panels, Panel{Title: title, Body: body})
		}
	}
	return panels
}

// RenderPanels lays panels out side by side, separated by vertical rules —
// the tmux choose-tree preview strip.
func RenderPanels(panels []Panel, width, height int) string {
	if len(panels) == 0 {
		return lipgloss.NewStyle().Foreground(cDim).Render("  (nothing to preview)")
	}
	if height < 3 {
		return ""
	}

	// Cap columns so each stays readable.
	const minCol = 24
	max := width / minCol
	if max < 1 {
		max = 1
	}
	truncated := 0
	if len(panels) > max {
		truncated = len(panels) - max
		panels = panels[:max]
	}

	sepW := len(panels) - 1
	colW := (width - sepW) / len(panels)
	if colW < 1 {
		colW = 1
	}

	cols := make([]string, 0, len(panels))
	for _, p := range panels {
		title := truncate(p.Title, colW-1)
		head := lipgloss.NewStyle().Foreground(cAccent).Bold(true).Render(title)

		body := fitBody(p.Body, colW, height-1)
		cols = append(cols, lipgloss.JoinVertical(lipgloss.Left, head, body))
	}

	rule := lipgloss.NewStyle().Foreground(cBorder).
		Render(strings.TrimRight(strings.Repeat("│\n", height), "\n"))

	parts := make([]string, 0, len(cols)*2)
	for i, c := range cols {
		if i > 0 {
			parts = append(parts, rule)
		}
		parts = append(parts, c)
	}
	out := lipgloss.JoinHorizontal(lipgloss.Top, parts...)

	if truncated > 0 {
		out += dimf("\n  +%d more", truncated)
	}
	return out
}

// fitBody clips a pane's screen to the column box, keeping the last lines
// (where live output is) and hard-cutting width so ANSI cannot bleed across
// the vertical rule.
func fitBody(body string, w, h int) string {
	if h < 1 {
		return ""
	}
	lines := strings.Split(strings.ReplaceAll(body, "\r\n", "\n"), "\n")

	// Drop trailing blank lines so short output sits at the top of the column.
	for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
		lines = lines[:len(lines)-1]
	}
	if len(lines) > h {
		lines = lines[len(lines)-h:]
	}

	out := make([]string, 0, h)
	for _, l := range lines {
		out = append(out, truncate(l, w))
	}
	for len(out) < h {
		out = append(out, "")
	}
	return strings.Join(out, "\n")
}
