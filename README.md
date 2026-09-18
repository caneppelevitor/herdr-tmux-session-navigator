# herdr-tmux-session-navigator

A tmux `choose-tree` style workspace navigator for [herdr](https://herdr.dev).

Press a key, get a collapsible tree of every workspace and its tabs, with live
previews of what is actually running in each pane. Pick one, land there.

<img width="1512" height="982" alt="image" src="https://github.com/user-attachments/assets/779dbfb4-c086-4ef2-9ff1-0fad6df65335" />


It is not a fuzzy launcher. It is the tmux session/window tree, for herdr.

## What it does

- **Tree of workspaces and tabs**, with tmux's `├─>` / `└─>` connectors,
  `(attached)` on the focused workspace, and `*` on the focused tab
- **Live panel previews** under the tree:
  - on a **workspace** row, one panel per tab
  - on a **tab** row, one panel per pane
  - panels keep each pane's own colours
- **Collapsible** workspaces, so a machine with many tabs stays readable
- **tmux jump keys** — `0`-`9` then `M-a`, `M-b`, ... select a row directly
- **Live**, not a snapshot: it subscribes to herdr's event stream and re-renders
  when workspaces, tabs, or panes change underneath it
- **Agent status** dots, since herdr knows whether an agent is working, idle, or
  blocked

## Requirements

- herdr 0.7.3 or newer
- Go 1.24+ to build
- Linux or macOS — the client speaks herdr's unix socket directly; Windows uses
  a named pipe instead, which is not implemented

Nothing at run time — it is a single static binary talking to herdr's unix
socket.

## Install

```sh
herdr plugin install caneppelevitor/herdr-tmux-session-navigator
```

That runs the build step in the manifest for you.

From a local checkout instead:

```sh
git clone https://github.com/caneppelevitor/herdr-tmux-session-navigator
cd herdr-tmux-session-navigator
make link          # builds, then `herdr plugin link .`
```

`herdr plugin link` does **not** run the manifest build step, so build first —
that is what `make link` does.

## Updating

herdr has no `plugin update`; reinstall to pull a newer version:

```sh
herdr plugin install caneppelevitor/herdr-tmux-session-navigator
```

## Add the keybinding

herdr does not take keybindings from plugin manifests, so add it to your own
`~/.config/herdr/config.toml`:

```toml
[[keys.command]]
key = "prefix+s"
type = "plugin_action"
command = "vitor.tmux-session-navigator.open"
```

Then reload, no restart needed:

```sh
herdr server reload-config
```

`prefix+s` is the tmux `choose-tree` muscle memory. It is herdr's built-in
`workspace_picker` by default, so move that out of the way first — either unset
it or give it another key:

```toml
[keys]
workspace_picker = "prefix+ctrl+w"   # or "" to unset
```

## Keys

| key | action |
|-----|--------|
| `enter` | switch to the selected workspace or tab |
| `space`, `tab` | collapse / expand the workspace |
| `←` `→`, `h` `l` | collapse / expand |
| `ctrl+a` | collapse or expand everything |
| `↑` `↓`, `j` `k` | move |
| `g` / `G` | first / last row |
| `0`-`9`, `M-a`.. | jump straight to a row |
| `/` | filter by workspace or tab label |
| `ctrl+x` | close the selected workspace (press twice to confirm) |
| `esc`, `q` | cancel |

Digits are jump keys, as in tmux. Press `/` first if you want to search for
something containing a digit.

## Where it opens

herdr caps which pane placements exist:

| herdr | placements |
|-------|------------|
| 0.7.x | `overlay`, `split`, `tab`, `zoomed` |
| 0.9+  | adds `popup` — a centred float |

Default is `overlay`: a modal over the active pane that restores your layout on
exit. Placement is resolved at run time, so on herdr 0.9+ you can switch without
rebuilding:

```sh
echo popup > "$(herdr plugin config-dir vitor.tmux-session-navigator)/placement"
```

Or per invocation: `HERDR_NAVIGATOR_PLACEMENT=zoomed`.

## Development

```sh
make build    # build to bin/
make test     # run tests
make vet      # go vet
make link     # build + link into herdr
```

After changing Go code, rebuild and re-invoke — a relink is only needed if you
edit `herdr-plugin.toml`.

Layout:

```
cmd/navigator/        entrypoint; --open spawns the pane, bare runs the TUI
internal/herdr/       socket client: request/response + event subscription
internal/plugin/      plugin identity and the pane-open handshake
internal/ui/tree.go   row model, collapse state, tmux shortcut ordering
internal/ui/preview.go  panel strip
internal/ui/model.go    Bubble Tea state machine
internal/ui/view.go     rendering
```

## License

MIT
