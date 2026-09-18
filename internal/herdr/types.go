package herdr

// Wire types for the herdr socket API. Field sets are trimmed to what the
// picker renders; the server sends more.

type Workspace struct {
	ID          string `json:"workspace_id"`
	Label       string `json:"label"`
	Number      int    `json:"number"`
	TabCount    int    `json:"tab_count"`
	PaneCount   int    `json:"pane_count"`
	Focused     bool   `json:"focused"`
	AgentStatus string `json:"agent_status"`
	ActiveTabID string `json:"active_tab_id"`
}

type Tab struct {
	ID          string `json:"tab_id"`
	WorkspaceID string `json:"workspace_id"`
	Label       string `json:"label"`
	Number      int    `json:"number"`
	PaneCount   int    `json:"pane_count"`
	Focused     bool   `json:"focused"`
	AgentStatus string `json:"agent_status"`
}

type Pane struct {
	ID          string `json:"pane_id"`
	WorkspaceID string `json:"workspace_id"`
	TabID       string `json:"tab_id"`
	Label       string `json:"label"`
	Focused     bool   `json:"focused"`
	AgentStatus string `json:"agent_status"`
	Cwd         string `json:"cwd"`
}

type request struct {
	ID     string `json:"id"`
	Method string `json:"method"`
	Params any    `json:"params"`
}

type response struct {
	ID     string         `json:"id"`
	Result map[string]any `json:"result"`
	Error  *wireError     `json:"error"`
}

type wireError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *wireError) Error() string { return e.Code + ": " + e.Message }

// Event is one pushed message from events.subscribe.
type Event struct {
	Name string         `json:"event"`
	Data map[string]any `json:"data"`
}
