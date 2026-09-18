package herdr

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestRemapMissingKeyIsAnError(t *testing.T) {
	// If herdr renames a response field, an empty tree with no diagnostic is the
	// worst outcome. A missing key must surface as an error.
	drifted := map[string]any{"items": []any{map[string]any{"workspace_id": "w1"}}}
	got, err := remap[[]Workspace](drifted, "workspaces")
	if err == nil {
		t.Fatalf("missing key returned no error (got %v)", got)
	}
	if !strings.Contains(err.Error(), "workspaces") {
		t.Errorf("error should name the missing field, got: %v", err)
	}
}

func TestRemapDecodesPresentKey(t *testing.T) {
	raw := map[string]any{"workspaces": []any{
		map[string]any{"workspace_id": "w1", "label": "alpha", "tab_count": float64(2)},
	}}
	got, err := remap[[]Workspace](raw, "workspaces")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 || got[0].ID != "w1" || got[0].Label != "alpha" || got[0].TabCount != 2 {
		t.Errorf("decoded %+v", got)
	}
}

func TestRemapTypeMismatchIsAnError(t *testing.T) {
	raw := map[string]any{"workspaces": "not-a-list"}
	if _, err := remap[[]Workspace](raw, "workspaces"); err == nil {
		t.Error("type mismatch should error")
	}
}

func TestEventUnmarshalsServerEnvelope(t *testing.T) {
	// Envelope observed on the wire: {"event":"...","data":{...}}
	line := `{"data":{"type":"tab_focused","tab_id":"wA:tJ","workspace_id":"wA"},"event":"tab_focused"}`
	var ev Event
	if err := json.Unmarshal([]byte(line), &ev); err != nil {
		t.Fatal(err)
	}
	if ev.Name != "tab_focused" {
		t.Errorf("Name = %q", ev.Name)
	}
	if ev.Data["tab_id"] != "wA:tJ" {
		t.Errorf("Data = %v", ev.Data)
	}
}

func TestWireErrorMessage(t *testing.T) {
	e := &wireError{Code: "invalid_request", Message: "missing field"}
	if got := e.Error(); got != "invalid_request: missing field" {
		t.Errorf("Error() = %q", got)
	}
}
