package ui

import (
	"testing"

	"github.com/caneppelevitor/herdr-tmux-session-navigator/internal/herdr"
)

func fixture() ([]herdr.Workspace, map[string][]herdr.Tab) {
	ws := []herdr.Workspace{
		{ID: "w1", Label: "alpha", TabCount: 2},
		{ID: "w2", Label: "beta", TabCount: 1},
	}
	tabs := map[string][]herdr.Tab{
		"w1": {{ID: "w1:t1", WorkspaceID: "w1", Number: 1}, {ID: "w1:t2", WorkspaceID: "w1", Number: 2}},
		"w2": {{ID: "w2:t1", WorkspaceID: "w2", Number: 1}},
	}
	return ws, tabs
}

func TestBuildRowsExpanded(t *testing.T) {
	ws, tabs := fixture()
	rows := BuildRows(ws, tabs, map[string]bool{})
	if len(rows) != 5 {
		t.Fatalf("want 5 rows (2 ws + 3 tabs), got %d", len(rows))
	}
	if rows[0].Kind != RowWorkspace || rows[1].Kind != RowTab {
		t.Fatal("expected workspace row followed by tab row")
	}
	if !rows[2].Last {
		t.Error("last tab of w1 should be marked Last for the elbow connector")
	}
	if rows[1].Last {
		t.Error("first of two tabs must not be Last")
	}
}

func TestBuildRowsCollapsed(t *testing.T) {
	ws, tabs := fixture()
	rows := BuildRows(ws, tabs, map[string]bool{"w1": true})
	if len(rows) != 3 {
		t.Fatalf("collapsing w1 should hide its 2 tabs: want 3 rows, got %d", len(rows))
	}
	for _, r := range rows {
		if r.Kind == RowTab && r.Tab.WorkspaceID == "w1" {
			t.Error("collapsed workspace must contribute no tab rows")
		}
	}
}

func TestShortcutOverflowUsesMeta(t *testing.T) {
	var ws []herdr.Workspace
	for i := 0; i < 12; i++ {
		ws = append(ws, herdr.Workspace{ID: string(rune('a' + i))})
	}
	rows := BuildRows(ws, map[string][]herdr.Tab{}, map[string]bool{})
	if rows[9].Shortcut != "9" {
		t.Errorf("row 10 shortcut = %q, want 9", rows[9].Shortcut)
	}
	if rows[10].Shortcut != "M-a" || rows[10].Bind != "alt+a" {
		t.Errorf("row 11 = %q/%q, want M-a/alt+a", rows[10].Shortcut, rows[10].Bind)
	}
}

func TestRowTargetAndWorkspaceID(t *testing.T) {
	ws, tabs := fixture()
	rows := BuildRows(ws, tabs, map[string]bool{})
	if k, id := rows[0].Target(); k != "ws" || id != "w1" {
		t.Errorf("workspace target = %s/%s", k, id)
	}
	if k, id := rows[1].Target(); k != "tab" || id != "w1:t1" {
		t.Errorf("tab target = %s/%s", k, id)
	}
	// A tab row must report its parent so preview/collapse act on the workspace.
	if got := rows[1].WorkspaceID(); got != "w1" {
		t.Errorf("tab row WorkspaceID = %s, want w1", got)
	}
}
