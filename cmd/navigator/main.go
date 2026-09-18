// Command herdr-tmux-session-navigator is a tmux choose-tree style workspace
// switcher for herdr.
//
// With no arguments it runs the picker TUI (herdr starts it inside a plugin
// pane). With --open it asks herdr to spawn that pane; this is what the
// keybinding invokes.
package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/caneppelevitor/herdr-tmux-session-navigator/internal/herdr"
	"github.com/caneppelevitor/herdr-tmux-session-navigator/internal/plugin"
	"github.com/caneppelevitor/herdr-tmux-session-navigator/internal/ui"
)

// version is overridable at build time:
//
//	go build -ldflags "-X main.version=1.2.3"
var version = "dev"

func main() {
	open := flag.Bool("open", false, "ask herdr to open the picker pane, then exit")
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.Parse()

	switch {
	case *showVersion:
		fmt.Println(version)
	case *open:
		fail(plugin.Open())
	default:
		fail(runPicker())
	}
}

func fail(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "herdr-tmux-session-navigator: %v\n", err)
		os.Exit(1)
	}
}

func runPicker() error {
	c := herdr.New()

	// Fail fast with a readable message rather than dying inside the TUI.
	if _, err := c.Workspaces(); err != nil {
		return err
	}

	ch := make(chan herdr.Event, 64)
	stop, err := c.Subscribe(herdr.DefaultSubscriptions, ch)
	if err != nil {
		return fmt.Errorf("subscribe: %w", err)
	}
	defer stop()

	out, err := tea.NewProgram(ui.NewModel(c, ch), tea.WithAltScreen()).Run()
	if err != nil {
		return err
	}

	m, ok := out.(ui.Model)
	if !ok || !m.HasChosen {
		return nil
	}

	kind, id := m.Chosen.Target()
	switch kind {
	case "ws":
		return c.FocusWorkspace(id)
	case "tab":
		return c.FocusTab(id)
	}
	return nil
}
