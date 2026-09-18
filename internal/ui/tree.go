package ui

import (
	"fmt"

	"github.com/caneppelevitor/herdr-tmux-session-navigator/internal/herdr"
)

type RowKind int

const (
	RowWorkspace RowKind = iota
	RowTab
)

// Row is one visible line in the tree. Collapsed workspaces contribute only
// their own row.
type Row struct {
	Kind      RowKind
	Shortcut  string // display form: "0".."9", then "M-a"..
	Bind      string // key that jumps here: "0".., "alt+a"..
	Workspace herdr.Workspace
	Tab       herdr.Tab
	Last      bool // last tab under its workspace (elbow vs tee)
}

// Target identifies what a row acts on.
func (r Row) Target() (kind, id string) {
	if r.Kind == RowWorkspace {
		return "ws", r.Workspace.ID
	}
	return "tab", r.Tab.ID
}

// WorkspaceID is the owning workspace for either row kind.
func (r Row) WorkspaceID() string {
	if r.Kind == RowWorkspace {
		return r.Workspace.ID
	}
	return r.Tab.WorkspaceID
}

// shortcutSeq yields tmux choose-tree's key order: 0-9, then M-a, M-b, ...
func shortcutSeq() func() (bind, disp string) {
	i := 0
	return func() (string, string) {
		defer func() { i++ }()
		if i < 10 {
			d := fmt.Sprintf("%d", i)
			return d, d
		}
		n := i - 10
		if n >= 26 {
			return "", ""
		}
		c := string(rune('a' + n))
		return "alt+" + c, "M-" + c
	}
}

// BuildRows flattens workspaces and their tabs into display rows, honouring
// per-workspace collapse state.
func BuildRows(wss []herdr.Workspace, tabs map[string][]herdr.Tab, collapsed map[string]bool) []Row {
	next := shortcutSeq()
	rows := make([]Row, 0, 32)

	for _, w := range wss {
		bind, disp := next()
		rows = append(rows, Row{
			Kind: RowWorkspace, Shortcut: disp, Bind: bind, Workspace: w,
		})
		if collapsed[w.ID] {
			continue
		}
		list := tabs[w.ID]
		for i, t := range list {
			bind, disp := next()
			rows = append(rows, Row{
				Kind: RowTab, Shortcut: disp, Bind: bind,
				Tab: t, Last: i == len(list)-1,
			})
		}
	}
	return rows
}
