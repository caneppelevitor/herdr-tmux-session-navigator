package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/caneppelevitor/herdr-session-popup/internal/herdr"
	"github.com/caneppelevitor/herdr-session-popup/internal/ui"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "session-popup: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
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

	p := tea.NewProgram(ui.NewModel(c, ch), tea.WithAltScreen())
	out, err := p.Run()
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
