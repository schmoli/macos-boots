# CLI Redesign: TUI → Simple Commands

## Goal

Replace slow TUI with fast CLI commands. Batch brew/cask installs via Brewfile for performance + cleaner output.

## Commands

```
macos-setup              # install all categories
macos-setup cli          # install cli category
macos-setup apps         # install apps category
macos-setup mas          # install mas apps (interactive, needs terminal)
macos-setup update       # upgrade installed tracked apps
macos-setup status       # show installed vs available
```

All commands auto-pull from origin first (silent if up to date).

## Install Flow

1. Auto-pull from origin if behind
2. Load `apps.yaml`, filter by category if specified
3. Check installed state (single `brew list` call, cached)
4. Split: `to_install` vs `already_installed`
5. Generate temp Brewfile for brew + cask apps
6. Run `brew bundle --file=/tmp/macos-setup-Brewfile`, stream output
7. Post-install: zsh integrations + post_install hooks per app
8. Handle npm apps sequentially
9. Print summary

## Output Style

```
$ macos-setup cli
Installing 3 packages...
==> Installing ripgrep
==> Installing fzf
==> Installing bat
Configuring zoxide... done
Configuring fnm... done
  → fnm install 24
  → fnm default 24

Installed: ripgrep, fzf, bat
Skipped (already installed): jq, yq, htop
```

## File Changes

### Remove
- `internal/tui/tui.go` - entire TUI

### Modify
- `cmd/macos-setup/main.go` - CLI arg parsing, command dispatch
- `internal/config/config.go` - helper methods (filter by category)

### New
- `internal/installer/installer.go` - Brewfile generation + install logic

### Keep As-Is
- `apps.yaml`
- `internal/state/state.go`
- `install.sh`
- `setup.sh` (first-install bootstrap, don't change without approval)

## Deferred

- Remove commands (uninstall apps)
- Interactive selection

## Notes

- Minimize zshrc source prompts - only when actually modified
- mas apps require terminal for password, run last/separately
- Dependencies handled by brew bundle automatically
