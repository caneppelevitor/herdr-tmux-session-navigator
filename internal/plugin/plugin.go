// Package plugin holds the plugin identity and the herdr handshake that opens
// the picker pane. Keeping the ID here means the manifest and the open action
// cannot drift apart.
package plugin

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ID must match the `id` field in herdr-plugin.toml.
const ID = "vitor.tmux-session-navigator"

// PaneEntrypoint must match the [[panes]] id in herdr-plugin.toml.
const PaneEntrypoint = "picker"

// DefaultPlacement is the widest-compatible choice. herdr 0.7.x supports
// overlay|split|tab|zoomed; 0.9+ adds "popup" (a centred float). Placement is
// resolved at run time so upgrading herdr needs no code change.
const DefaultPlacement = "overlay"

// Placement resolves the pane placement: env var, then the plugin config dir,
// then the default.
func Placement() string {
	if p := strings.TrimSpace(os.Getenv("HERDR_NAVIGATOR_PLACEMENT")); p != "" {
		return p
	}
	if dir := os.Getenv("HERDR_PLUGIN_CONFIG_DIR"); dir != "" {
		if b, err := os.ReadFile(filepath.Join(dir, "placement")); err == nil {
			if p := strings.TrimSpace(string(b)); p != "" {
				return p
			}
		}
	}
	return DefaultPlacement
}

// Open asks herdr to spawn the picker pane. It runs as the plugin action, so
// the TUI itself starts in a real pane with a TTY.
func Open() error {
	bin := os.Getenv("HERDR_BIN_PATH")
	if bin == "" {
		bin = "herdr"
	}
	id := os.Getenv("HERDR_PLUGIN_ID")
	if id == "" {
		id = ID
	}

	cmd := exec.Command(bin, "plugin", "pane", "open",
		"--plugin", id,
		"--entrypoint", PaneEntrypoint,
		"--placement", Placement(),
		"--focus")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("open picker pane: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}
