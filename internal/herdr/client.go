package herdr

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sync"
)

// SocketPath resolves the server socket, preferring the env var herdr injects
// into plugin processes.
func SocketPath() string {
	if p := os.Getenv("HERDR_SOCKET_PATH"); p != "" {
		return p
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "herdr", "herdr.sock")
}

// Client issues request/response calls. The server closes a connection after
// each request, so every call dials fresh. Events use a separate long-lived
// connection (see Subscribe).
type Client struct {
	path string
	mu   sync.Mutex
	seq  int
}

func New() *Client { return &Client{path: SocketPath()} }

func (c *Client) nextID() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.seq++
	return fmt.Sprintf("picker-%d", c.seq)
}

// Call sends one request and decodes the result object.
func (c *Client) Call(method string, params any) (map[string]any, error) {
	if params == nil {
		params = map[string]any{}
	}
	conn, err := net.Dial("unix", c.path)
	if err != nil {
		return nil, fmt.Errorf("dial herdr socket: %w", err)
	}
	defer conn.Close()

	enc := json.NewEncoder(conn)
	if err := enc.Encode(request{ID: c.nextID(), Method: method, Params: params}); err != nil {
		return nil, fmt.Errorf("%s: %w", method, err)
	}

	var resp response
	if err := json.NewDecoder(bufio.NewReader(conn)).Decode(&resp); err != nil {
		return nil, fmt.Errorf("%s: decode: %w", method, err)
	}
	if resp.Error != nil {
		return nil, fmt.Errorf("%s: %w", method, resp.Error)
	}
	return resp.Result, nil
}

// remap round-trips a result field through JSON into a typed value.
func remap[T any](result map[string]any, key string) (T, error) {
	var out T
	raw, err := json.Marshal(result[key])
	if err != nil {
		return out, err
	}
	err = json.Unmarshal(raw, &out)
	return out, err
}

func (c *Client) Workspaces() ([]Workspace, error) {
	r, err := c.Call("workspace.list", nil)
	if err != nil {
		return nil, err
	}
	return remap[[]Workspace](r, "workspaces")
}

func (c *Client) Tabs(workspaceID string) ([]Tab, error) {
	r, err := c.Call("tab.list", map[string]any{"workspace_id": workspaceID})
	if err != nil {
		return nil, err
	}
	return remap[[]Tab](r, "tabs")
}

func (c *Client) Panes(workspaceID string) ([]Pane, error) {
	r, err := c.Call("pane.list", map[string]any{"workspace_id": workspaceID})
	if err != nil {
		return nil, err
	}
	return remap[[]Pane](r, "panes")
}

// ReadPane returns what is currently on the pane's screen. source "visible"
// matches tmux preview semantics; keeping ANSI preserves the pane's own colors.
func (c *Client) ReadPane(paneID string, lines int) (string, error) {
	params := map[string]any{
		"pane_id":    paneID,
		"source":     "visible",
		"format":     "ansi",
		"strip_ansi": false,
	}
	if lines > 0 {
		params["lines"] = lines
	}
	r, err := c.Call("pane.read", params)
	if err != nil {
		return "", err
	}
	// The payload is nested: {"type":"pane_read","read":{...,"text":"..."}}
	read, ok := r["read"].(map[string]any)
	if !ok {
		return "", nil
	}
	text, _ := read["text"].(string)
	return text, nil
}

func (c *Client) FocusWorkspace(id string) error {
	_, err := c.Call("workspace.focus", map[string]any{"workspace_id": id})
	return err
}

func (c *Client) FocusTab(id string) error {
	_, err := c.Call("tab.focus", map[string]any{"tab_id": id})
	return err
}

func (c *Client) CloseWorkspace(id string) error {
	_, err := c.Call("workspace.close", map[string]any{"workspace_id": id})
	return err
}

// DefaultSubscriptions are the parameter-free event types. The per-pane ones
// (pane.agent_status_changed, pane.scroll_changed, pane.output_matched) require
// a specific pane_id and are deliberately excluded.
var DefaultSubscriptions = []string{
	"workspace.created", "workspace.updated", "workspace.renamed",
	"workspace.closed", "workspace.focused", "workspace.moved",
	"tab.created", "tab.closed", "tab.focused", "tab.renamed", "tab.moved",
	"pane.created", "pane.closed", "pane.exited", "pane.agent_detected",
	"layout.updated",
}

// Subscribe opens a dedicated connection and streams events until the returned
// stop func is called or the server closes. Subscribe must be the first request
// on its connection.
func (c *Client) Subscribe(types []string, out chan<- Event) (stop func(), err error) {
	conn, err := net.Dial("unix", c.path)
	if err != nil {
		return nil, fmt.Errorf("dial herdr socket: %w", err)
	}

	subs := make([]map[string]string, 0, len(types))
	for _, t := range types {
		subs = append(subs, map[string]string{"type": t})
	}
	req := request{ID: "picker-events", Method: "events.subscribe",
		Params: map[string]any{"subscriptions": subs}}
	if err := json.NewEncoder(conn).Encode(req); err != nil {
		conn.Close()
		return nil, err
	}

	rd := bufio.NewReader(conn)
	var ack response
	if err := json.NewDecoder(rd).Decode(&ack); err != nil {
		conn.Close()
		return nil, fmt.Errorf("subscribe ack: %w", err)
	}
	if ack.Error != nil {
		conn.Close()
		return nil, ack.Error
	}

	go func() {
		defer conn.Close()
		dec := json.NewDecoder(rd)
		for {
			var ev Event
			if err := dec.Decode(&ev); err != nil {
				close(out)
				return
			}
			out <- ev
		}
	}()

	return func() { conn.Close() }, nil
}
