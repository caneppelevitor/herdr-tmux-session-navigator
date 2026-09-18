package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) View() string {
	if m.quitting {
		return ""
	}
	if m.width == 0 {
		return "loading..."
	}

	previewH := m.previewHeight()
	treeH := m.height - previewH - 3 // header + rule + status
	if treeH < 3 {
		treeH = 3
	}

	var b strings.Builder
	b.WriteString(m.renderHeader())
	b.WriteString("\n")
	b.WriteString(m.renderTree(treeH))
	b.WriteString("\n")
	b.WriteString(m.renderRule())
	b.WriteString("\n")
	b.WriteString(RenderPanels(m.panels, m.width, previewH))
	return b.String()
}

func (m Model) renderHeader() string {
	if m.filtering || m.filter != "" {
		cur := ""
		if m.filtering {
			cur = lipgloss.NewStyle().Foreground(cSelBg).Render("█")
		}
		return lipgloss.NewStyle().Foreground(cAccent).Render("  /") +
			lipgloss.NewStyle().Foreground(cFg).Render(m.filter) + cur +
			dimf("    esc clear")
	}
	keys := []string{
		"enter switch", "space collapse", "^a all", "/ search",
		"^x close", "0-9 jump", "esc quit",
	}
	return dimf("  %s", strings.Join(keys, "    "))
}

func (m Model) renderRule() string {
	label := " preview "
	if len(m.rows) > 0 && m.cursor < len(m.rows) {
		row := m.rows[m.cursor]
		if row.Kind == RowWorkspace {
			label = fmt.Sprintf(" %s — tabs ", row.Workspace.Label)
		} else {
			label = fmt.Sprintf(" %s — panes ", row.Tab.Label)
		}
	}
	styled := lipgloss.NewStyle().Foreground(cDim).Render(label)
	w := m.width - len(label) - 4
	if w < 0 {
		w = 0
	}
	line := lipgloss.NewStyle().Foreground(cBorder)
	return line.Render("──") + styled + line.Render(strings.Repeat("─", w))
}

// renderTree draws the row list with a scroll window that keeps the cursor
// visible, and paints the selected row as a full-width bar.
func (m Model) renderTree(height int) string {
	if len(m.rows) == 0 {
		msg := "  no workspaces"
		if m.filter != "" {
			msg = "  no match for " + m.filter
		}
		return dimf("%s", msg)
	}

	start := 0
	if m.cursor >= height {
		start = m.cursor - height + 1
	}
	end := start + height
	if end > len(m.rows) {
		end = len(m.rows)
	}

	lines := make([]string, 0, height)
	for i := start; i < end; i++ {
		lines = append(lines, m.renderRow(m.rows[i], i == m.cursor))
	}
	for len(lines) < height {
		lines = append(lines, "")
	}
	return strings.Join(lines, "\n")
}

func (m Model) renderRow(r Row, selected bool) string {
	sc := fmt.Sprintf("(%s)", r.Shortcut)
	if r.Shortcut == "" {
		sc = ""
	}

	var body string
	if r.Kind == RowWorkspace {
		marker := "-"
		if m.collapsed[r.Workspace.ID] {
			marker = "+"
		}
		n := r.Workspace.TabCount
		attached := ""
		if r.Workspace.Focused {
			attached = "  (attached)"
		}
		body = fmt.Sprintf("%-6s %s %s %s: %d tab%s%s",
			sc, marker, statusDot(r.Workspace.AgentStatus),
			r.Workspace.Label, n, plural(n), attached)
		if !selected {
			body = fmt.Sprintf("%s %s %s %s: %s%s",
				dimf("%-6s", sc), dimf("%s", marker),
				statusDot(r.Workspace.AgentStatus),
				lipgloss.NewStyle().Foreground(cName).Bold(true).Render(r.Workspace.Label),
				lipgloss.NewStyle().Foreground(cFg).Render(fmt.Sprintf("%d tab%s", n, plural(n))),
				lipgloss.NewStyle().Foreground(cAccent).Render(attached))
		}
	} else {
		elbow := "├─>"
		if r.Last {
			elbow = "└─>"
		}
		mark := ""
		if r.Tab.Focused {
			mark = "*"
		}
		label := r.Tab.Label
		if label == "" {
			label = fmt.Sprintf("%d", r.Tab.Number)
		}
		pc := r.Tab.PaneCount
		body = fmt.Sprintf("%-6s %s %d: %s%s  (%d pane%s)",
			sc, elbow, r.Tab.Number, label, mark, pc, plural(pc))
		if !selected {
			body = fmt.Sprintf("%s %s %s %s%s  %s",
				dimf("%-6s", sc), dimf("%s", elbow),
				lipgloss.NewStyle().Foreground(cFg).Render(fmt.Sprintf("%d:", r.Tab.Number)),
				label,
				lipgloss.NewStyle().Foreground(cAccent).Render(mark),
				dimf("(%d pane%s)", pc, plural(pc)))
		}
	}

	if selected {
		return lipgloss.NewStyle().
			Background(cSelBg).Foreground(cSelFg).Bold(true).
			Render(padTo(" "+body, m.width))
	}
	return " " + body
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}
