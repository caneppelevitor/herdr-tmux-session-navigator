# herdr session popup

tmux-style fuzzy workspace switcher for [herdr](https://herdr.dev). Opens a modal
pane with an fzf list of workspaces and a live preview of the panes inside each.

    prefix + shift + f      (POC binding — see "Taking over prefix+s")

    enter   switch to workspace
    esc     cancel

## Requirements

- herdr >= 0.7.0
- fzf, python3 (both already required by herdr workflows here)
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

    herdr-plugin.toml    manifest: action + pane entrypoint
    bin/open             action — opens the picker pane at the configured placement
    bin/picker           the fzf UI; calls `herdr workspace focus` on select
    bin/preview          fzf preview — lists panes + agent status per workspace

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

## Status

POC. Workspaces only. Not yet: agent rows, zoxide dirs, create-on-miss,
close-workspace binding.
