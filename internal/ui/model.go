package ui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/caneppelevitor/herdr-tmux-session-navigator/internal/herdr"
)

// herdr emits events in bursts (a single split produces pane.created,
// layout.updated, tab.focused...). Refetching per event would mean an N+1 call
// storm, so events set a dirty flag and one refresh runs after this window.
const coalesceWindow = 120 * time.Millisecond

// resubscribeDelay backs off before reconnecting a dropped event stream.
const resubscribeDelay = 2 * time.Second

type snapshotMsg struct {
	workspaces []herdr.Workspace
	tabs       map[string][]herdr.Tab
	panes      map[string][]herdr.Pane
	err        error
}

type panelsMsg struct {
	key    string
	panels []Panel
}

type eventMsg struct{ name string }
type errMsg struct{ err error }
type refreshMsg struct{}
type resubscribeMsg struct {
	events <-chan herdr.Event
	err    error
}

// Model is the picker state. It refreshes from herdr events rather than
// polling, so the tree reflects the server as it changes.
type Model struct {
	client *herdr.Client
	events <-chan herdr.Event

	workspaces []herdr.Workspace
	tabs       map[string][]herdr.Tab
	panes      map[string][]herdr.Pane

	rows      []Row
	cursor    int
	collapsed map[string]bool

	panels    []Panel
	panelKey  string
	filter    string
	filtering bool

	width, height int
	status        string
	dirty         bool // an event arrived; a coalesced refresh is pending
	pendingClose  string
	quitting      bool
	Chosen        Row
	HasChosen     bool
}

func NewModel(c *herdr.Client, events <-chan herdr.Event) Model {
	return Model{
		client:    c,
		events:    events,
		tabs:      map[string][]herdr.Tab{},
		panes:     map[string][]herdr.Pane{},
		collapsed: map[string]bool{},
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.loadSnapshot(), m.waitEvent())
}

// loadSnapshot pulls the full tree. Cheap enough (one call per workspace) that
// refetching on any event is simpler and safer than patching state per event.
func (m Model) loadSnapshot() tea.Cmd {
	c := m.client
	return func() tea.Msg {
		wss, err := c.Workspaces()
		if err != nil {
			return snapshotMsg{err: err}
		}
		tabs := make(map[string][]herdr.Tab, len(wss))
		panes := make(map[string][]herdr.Pane, len(wss))
		for _, w := range wss {
			if t, err := c.Tabs(w.ID); err == nil {
				tabs[w.ID] = t
			}
			if p, err := c.Panes(w.ID); err == nil {
				panes[w.ID] = p
			}
		}
		return snapshotMsg{workspaces: wss, tabs: tabs, panes: panes}
	}
}

func (m Model) waitEvent() tea.Cmd {
	ch := m.events
	return func() tea.Msg {
		ev, ok := <-ch
		if !ok {
			return errMsg{fmt.Errorf("event stream closed")}
		}
		return eventMsg{ev.Name}
	}
}

// loadPanels fetches preview content off the UI goroutine.
func (m Model) loadPanels() tea.Cmd {
	if len(m.rows) == 0 || m.cursor >= len(m.rows) {
		return nil
	}
	row := m.rows[m.cursor]
	kind, id := row.Target()
	key := kind + ":" + id
	if key == m.panelKey {
		return nil
	}

	c, tabs, panes := m.client, m.tabs, m.panes
	lines := m.previewHeight()
	return func() tea.Msg {
		return panelsMsg{key: key, panels: PanelsFor(c, row, tabs, panes, lines)}
	}
}

func (m Model) previewHeight() int {
	h := m.height*62/100 - 3
	if h < 3 {
		h = 3
	}
	return h
}

func (m *Model) rebuild() {
	m.rows = BuildRows(m.filtered(), m.tabs, m.collapsed)
	if m.cursor >= len(m.rows) {
		m.cursor = len(m.rows) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
}

// filtered narrows workspaces by the live filter, matching workspace labels and
// their tab labels so typing a tab name still reveals its parent.
func (m Model) filtered() []herdr.Workspace {
	if m.filter == "" {
		return m.workspaces
	}
	q := strings.ToLower(m.filter)
	var out []herdr.Workspace
	for _, w := range m.workspaces {
		if strings.Contains(strings.ToLower(w.Label), q) {
			out = append(out, w)
			continue
		}
		for _, t := range m.tabs[w.ID] {
			if strings.Contains(strings.ToLower(t.Label), q) {
				out = append(out, w)
				break
			}
		}
	}
	return out
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.panelKey = "" // force preview refit
		return m, m.loadPanels()

	case snapshotMsg:
		if msg.err != nil {
			m.status = msg.err.Error()
			return m, nil
		}
		m.workspaces, m.tabs, m.panes = msg.workspaces, msg.tabs, msg.panes
		m.rebuild()
		m.panelKey = ""
		return m, m.loadPanels()

	case eventMsg:
		// Coalesce bursts: mark dirty and schedule one refresh, rather than
		// refetching the whole tree per event.
		cmds := []tea.Cmd{m.waitEvent()}
		if !m.dirty {
			m.dirty = true
			cmds = append(cmds, tea.Tick(coalesceWindow, func(time.Time) tea.Msg {
				return refreshMsg{}
			}))
		}
		return m, tea.Batch(cmds...)

	case refreshMsg:
		m.dirty = false
		return m, m.loadSnapshot()

	case panelsMsg:
		m.panelKey, m.panels = msg.key, msg.panels
		return m, nil

	case errMsg:
		// Most commonly the event stream closed. Report it and try to reconnect
		// so the picker does not sit silently stale.
		m.status = msg.err.Error()
		return m, tea.Tick(resubscribeDelay, func(time.Time) tea.Msg {
			ch := make(chan herdr.Event, 64)
			if _, err := m.client.Subscribe(herdr.DefaultSubscriptions, ch); err != nil {
				return resubscribeMsg{err: err}
			}
			return resubscribeMsg{events: ch}
		})

	case resubscribeMsg:
		if msg.err != nil {
			m.status = "reconnect failed: " + msg.err.Error()
			return m, tea.Tick(resubscribeDelay, func(time.Time) tea.Msg {
				return errMsg{fmt.Errorf("retrying")}
			})
		}
		m.status = ""
		m.events = msg.events
		return m, tea.Batch(m.loadSnapshot(), m.waitEvent())

	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	// Filter mode captures printable keys so labels containing digits are
	// searchable; jump keys stay available outside it.
	if m.filtering {
		switch key {
		case "esc":
			m.filtering, m.filter = false, ""
			m.rebuild()
			return m, m.loadPanels()
		case "enter":
			m.filtering = false
			return m, nil
		case "backspace":
			if m.filter != "" {
				m.filter = m.filter[:len(m.filter)-1]
				m.rebuild()
				return m, m.loadPanels()
			}
			return m, nil
		default:
			if len(key) == 1 {
				m.filter += key
				m.rebuild()
				return m, m.loadPanels()
			}
			return m, nil
		}
	}

	// Any key other than a confirming ctrl+x aborts a pending close.
	if m.pendingClose != "" && key != "ctrl+x" {
		m.pendingClose = ""
		m.status = ""
	}

	switch key {
	case "ctrl+c", "esc", "q":
		m.quitting = true
		return m, tea.Quit

	case "/":
		m.filtering = true
		return m, nil

	case "up", "k", "ctrl+p":
		if m.cursor > 0 {
			m.cursor--
		}
		return m, m.loadPanels()

	case "down", "j", "ctrl+n":
		if m.cursor < len(m.rows)-1 {
			m.cursor++
		}
		return m, m.loadPanels()

	case "g", "home":
		m.cursor = 0
		return m, m.loadPanels()

	case "G", "end":
		m.cursor = len(m.rows) - 1
		return m, m.loadPanels()

	case "left", "h", "-":
		return m.collapse(true)

	case "right", "l", "+":
		return m.collapse(false)

	case " ", "tab":
		// Toggle collapse on the row's workspace.
		if len(m.rows) == 0 {
			return m, nil
		}
		id := m.rows[m.cursor].WorkspaceID()
		m.collapsed[id] = !m.collapsed[id]
		m.rebuild()
		return m, m.loadPanels()

	case "ctrl+a":
		// Collapse or expand everything, whichever is the minority.
		anyOpen := false
		for _, w := range m.workspaces {
			if !m.collapsed[w.ID] {
				anyOpen = true
				break
			}
		}
		for _, w := range m.workspaces {
			m.collapsed[w.ID] = anyOpen
		}
		m.rebuild()
		return m, m.loadPanels()

	case "enter":
		if len(m.rows) == 0 {
			return m, nil
		}
		m.Chosen, m.HasChosen = m.rows[m.cursor], true
		m.quitting = true
		return m, tea.Quit

	case "ctrl+x":
		// Closing a workspace kills its panes and cannot be undone, so require
		// a second confirming keypress.
		if len(m.rows) == 0 {
			return m, nil
		}
		row := m.rows[m.cursor]
		if row.Kind != RowWorkspace {
			m.status = "select a workspace row to close"
			return m, nil
		}
		if m.pendingClose != row.Workspace.ID {
			m.pendingClose = row.Workspace.ID
			m.status = fmt.Sprintf("close %q and all its panes? ctrl+x again to confirm, any other key cancels",
				row.Workspace.Label)
			return m, nil
		}
		m.pendingClose = ""
		m.status = ""
		if err := m.client.CloseWorkspace(row.Workspace.ID); err != nil {
			m.status = err.Error()
		}
		return m, m.loadSnapshot()
	}

	// tmux jump keys: 0-9 then M-a.. select that row directly.
	for i, r := range m.rows {
		if r.Bind != "" && r.Bind == key {
			m.Chosen, m.HasChosen = m.rows[i], true
			m.quitting = true
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m Model) collapse(want bool) (tea.Model, tea.Cmd) {
	if len(m.rows) == 0 {
		return m, nil
	}
	row := m.rows[m.cursor]
	id := row.WorkspaceID()
	// Collapsing from a tab row jumps the cursor up to its workspace header so
	// the selection stays visible.
	if want && row.Kind == RowTab {
		for i, r := range m.rows {
			if r.Kind == RowWorkspace && r.Workspace.ID == id {
				m.cursor = i
				break
			}
		}
	}
	m.collapsed[id] = want
	m.rebuild()
	return m, m.loadPanels()
}
