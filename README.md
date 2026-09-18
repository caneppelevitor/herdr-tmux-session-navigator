# herdr session popup

tmux `choose-tree` for [herdr](https://herdr.dev). Collapsible workspace/tab tree
with live panel previews, driven by the herdr event stream.

Go + Bubble Tea. Talks the herdr unix socket directly — no CLI subprocesses, no
polling: it subscribes to `events.subscribe` and re-renders when the server says
something changed.

    prefix + shift + f      (POC binding -- see "Taking over prefix+s")

    enter        switch to workspace / tab
    space, tab   collapse or expand the workspace
    left/right   collapse / expand
    ctrl+a       collapse or expand all
    0-9, M-a..   jump straight to a row (tmux choose-tree keys)
    /            filter (matches workspace and tab labels)
    ctrl+x       close the selected workspace
    esc, q       cancel

## Requirements

- herdr >= 0.7.0
- Go 1.24+ to build (runtime needs nothing -- static binary)
- Linux or macOS

## Install

    herdr plugin link /path/to/herdr-session-popup

Then bind it in `~/.config/herdr/config.toml`:

    [[keys.command]]
    key = "prefix+shift+f"
    type = "plugin_action"
    command = "vitor.session-popup.open"

    herdr server reload-config

## Placement

herdr caps which placements exist:

| version | placements |
|---------|-----------------------------------------|
| 0.7.x   | `overlay`, `split`, `tab`, `zoomed`     |
| 0.9.x   | adds `popup` (centered float, w/h args) |

This plugin resolves placement at runtime, not from the manifest, so no code
change is needed after upgrading. Precedence: env > config file > default.

    # after `herdr update` to >= 0.9
    echo popup > "$(herdr plugin config-dir vitor.session-popup)/placement"

Or per-invocation: `SESSION_POPUP_PLACEMENT=zoomed`.

Default is `overlay` — a modal over the active pane that restores your layout on
exit. Closest thing 0.7.3 has to tmux `display-popup -E`.

## Taking over prefix+s

`prefix+s` is currently herdr's built-in `workspace_picker`. To hand it to this
plugin, rebind the built-in first or the two will collide:

    [keys]
    workspace_picker = ""            # or move it elsewhere

    [[keys.command]]
    key = "prefix+s"
    type = "plugin_action"
    command = "vitor.session-popup.open"

## Layout

    herdr-plugin.toml       manifest: build + action + pane entrypoint
    bin/open                action -- opens the picker pane at the configured placement
    cmd/picker/main.go      entrypoint: subscribe, run TUI, act on selection
    internal/herdr/         socket client (request/response + event stream)
    internal/ui/tree.go     row model, collapse state, tmux shortcut order
    internal/ui/preview.go  panel strip: tabs for a workspace row, panes for a tab row
    internal/ui/model.go    Bubble Tea state machine
    internal/ui/view.go     rendering

## Notes from building this against 0.7.3

Things that cost time and are not obvious from the docs:

- Manifest `[[keys.command]]` blocks are **silently ignored**. Keybindings must
  live in the user's `config.toml`. `herdr plugin list --json` shows no `keys`.
- Config keybinding `type` accepts `shell`, `pane`, `plugin_action` — **not**
  `plugin_pane`, despite that string existing in the binary as an API method.
  Hence the `bin/open` action wrapper.
- **Action** commands resolve relative to the plugin root. **Pane** commands
  resolve against `PATH`. Pane commands therefore go through
  `$HERDR_PLUGIN_ROOT`.
- Pane commands need a **login** shell (`bash -lc`). A non-login shell gets
  `/usr/gnu/bin:/usr/local/bin:/bin:/usr/bin:.` with no Homebrew, so `fzf` is
  unresolvable and the pane dies instantly at startup.

## Preview panels

The strip under the tree mirrors tmux's preview row:

- cursor on a **workspace** row -> one panel per tab, showing that tab's first pane
- cursor on a **tab** row -> one panel per pane in that tab

Panels are read with `pane.read` at `source: "visible"`, `format: "ansi"`, so each
keeps its own colours. Columns are capped at a minimum width; overflow is
reported as `+N more` rather than silently dropped.

## Why a persistent socket

`events.subscribe` pushes 24 event types (`workspace.*`, `tab.*`, `pane.*`,
`layout.updated`). The earlier bash+fzf version could not consume them -- fzf owns
the event loop and blocks -- so its tree was a snapshot that went stale the moment
anything changed. Bubble Tea owns the loop here and treats an event as a cue to
refetch, so the tree tracks the server.

Per-pane subscriptions (`pane.agent_status_changed`, `pane.scroll_changed`,
`pane.output_matched`) require a specific `pane_id` and are excluded; the
parameter-free set covers structural change.

## Status

Workspaces and tabs. Not yet: agent rows as a third tree level, zoxide dirs,
create-on-miss, rename in place.
