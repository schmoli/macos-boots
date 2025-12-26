# macOS Developer Box Setup Tool - Design

## Overview

A Go-based TUI application for setting up a fresh macOS developer machine. Downloads as a single binary with zero dependencies, then orchestrates installing apps, symlinking configs, and tracking progress across multiple runs.

## Goals

- **Idempotent** - Run multiple times safely, skips completed steps
- **Resumable** - State file tracks progress, pick up where you left off
- **Interactive when needed** - Auto installs run hands-off, interactive installs (Parallels, Office) prompt for password
- **Modular configs** - Per-app zsh modules, symlinked dotfiles
- **Single binary** - No runtime deps, curl-install before Homebrew exists

## Architecture

### Two Components

```
Binary (engine)              Repo (data)
─────────────────           ─────────────────
Go TUI application    +     apps.yaml
State management            configs/
Install orchestration       zsh modules
```

Binary fetches repo, reads config, executes installs, symlinks files.

### Repository Structure

```
github.com/toli/macos-setup
├── install.sh              # curl-able bootstrap (downloads binary)
├── cmd/                    # Go TUI source
│   └── macos-setup/
│       └── main.go
├── internal/               # Go packages
│   ├── tui/                # Bubbletea TUI
│   ├── state/              # State management
│   ├── installer/          # brew/cask/mas wrappers
│   └── config/             # YAML parsing
├── apps.yaml               # App recipes
├── configs/                # Dotfiles (symlinked to ~/.config/)
│   ├── yazi/
│   │   ├── yazi.toml
│   │   ├── theme.toml
│   │   └── flavors/
│   ├── starship.toml
│   └── zsh/
│       ├── init.zsh        # Module loader
│       └── modules/
│           ├── yazi.zsh
│           └── starship.zsh
├── releases/               # Pre-built binary (GitHub Actions)
│   └── macos-setup-darwin-arm64  # Apple Silicon only
└── docs/
```

### State File

Located at `~/.config/macos-setup/state.json`:

```json
{
  "version": "1.0.0",
  "last_run": "2025-12-26T10:30:00Z",
  "repo_path": "~/.config/macos-setup/repo",
  "bootstrap": "complete",
  "apps": {
    "yazi": {
      "install": "complete",
      "config": "complete",
      "zsh": "complete"
    },
    "parallels": {
      "install": "pending"
    }
  }
}
```

## Installation Flow

### Phase 1: Bootstrap (no auth, no deps)

```zsh
curl -fsSL https://raw.githubusercontent.com/toli/macos-setup/main/install.sh | zsh
```

`install.sh`:
```zsh
#!/bin/zsh
REPO="toli/macos-setup"
BINARY="macos-setup"
ARCH=$(uname -m)
[[ "$ARCH" == "arm64" ]] && ARCH="arm64" || ARCH="amd64"

curl -fsSL "https://github.com/$REPO/releases/latest/download/${BINARY}-darwin-${ARCH}" \
  -o /tmp/$BINARY
chmod +x /tmp/$BINARY
/tmp/$BINARY install
```

### Phase 2: Binary Takes Over

1. Prompts for xcode-select install (if needed)
2. Installs Homebrew (if needed)
3. Clones repo to `~/.config/macos-setup/repo/`
4. Reads `apps.yaml`
5. Launches TUI

### Phase 3: TUI-Driven Installs

User selects categories/apps, binary executes, updates state.

## App Recipe Format

`apps.yaml`:

```yaml
apps:
  # CLI tool with config and shell integration
  yazi:
    install: brew
    category: cli
    tier: auto
    description: "Terminal file manager"
    config:
      source: configs/yazi/
      dest: ~/.config/yazi/
    zsh: configs/zsh/modules/yazi.zsh

  # Simple cask, no config
  vscode:
    install: cask
    category: apps
    tier: auto
    description: "Code editor"

  # Interactive install (needs password)
  parallels:
    install: cask
    category: apps
    tier: interactive
    description: "VM software (requires password)"

  # Mac App Store app
  xcode:
    install: mas
    id: 497799835
    category: dev
    tier: auto
    post_install:
      - "sudo xcodebuild -license accept"
    description: "Apple's IDE"

  # Just config, no install (app installed separately)
  rectangle:
    install: cask
    category: apps
    tier: auto
    config:
      source: configs/rectangle/com.knollsoft.Rectangle.plist
      dest: ~/Library/Preferences/
```

### Recipe Fields

| Field | Required | Description |
|-------|----------|-------------|
| `install` | yes | `brew`, `cask`, `mas`, or `script` |
| `category` | yes | Grouping: `cli`, `apps`, `dev`, etc. |
| `tier` | yes | `auto` (hands-off) or `interactive` (needs password/input) |
| `description` | no | Shown in TUI |
| `id` | mas only | App Store ID |
| `config.source` | no | Path in repo to config file/dir |
| `config.dest` | no | Where to symlink in home |
| `zsh` | no | Zsh module to source (aliases, functions) |
| `post_install` | no | Commands to run after install |

### Sudo Handling

Before running any `tier: interactive` apps, the TUI:

1. Prompts user: "Interactive installs need admin password"
2. Runs `sudo -v` to cache credentials
3. Spawns background keep-alive: `while true; do sudo -n true; sleep 60; done &`
4. Proceeds with installs - no further password prompts needed

This way user types password once per interactive batch, not per-app.

## TUI Design

Built with Go + Bubbletea (Charm).

### Main Screen

```
┌─ macos-setup ─────────────────────────────────────────┐
│                                                       │
│  ● CLI Tools           [3/5]   ▸                      │
│  ○ Desktop Apps        [0/8]   ▸                      │
│  ○ App Store           [0/3]   ▸                      │
│  ○ Configs             [1/4]   ▸                      │
│                                                       │
│  ─────────────────────────────────────────────────    │
│  [i] Install all auto    [u] Update repo              │
│  [q] Quit                                             │
└───────────────────────────────────────────────────────┘
```

### Category View

```
┌─ CLI Tools ───────────────────────────────────────────┐
│                                                       │
│  [x] git              installed                       │
│  [x] yazi             installed + configured          │
│  [ ] starship         pending                         │
│  [ ] fzf              pending                         │
│  [x] ripgrep          installed                       │
│                                                       │
│  ─────────────────────────────────────────────────    │
│  [space] Toggle   [i] Install selected   [←] Back    │
└───────────────────────────────────────────────────────┘
```

### Install Progress

```
┌─ Installing ──────────────────────────────────────────┐
│                                                       │
│  ✓ starship          brew install complete            │
│  ● fzf               installing...                    │
│  ○ bat               queued                           │
│                                                       │
│  ─────────────────────────────────────────────────    │
│  [c] Cancel                                           │
└───────────────────────────────────────────────────────┘
```

## Zsh Module System

### Loader

Binary ensures `~/.zshrc` contains:

```zsh
source ~/.config/zsh/init.zsh
```

`configs/zsh/init.zsh`:

```zsh
# Load all zsh modules
for conf in ~/.config/zsh/modules/*.zsh; do
  [[ -f "$conf" ]] && source "$conf"
done

# Core setup
export PATH="/opt/homebrew/bin:$PATH"
autoload -Uz compinit && compinit
```

### Per-App Modules

`configs/zsh/modules/yazi.zsh`:

```zsh
# Yazi shell integration
function y() {
  local tmp="$(mktemp)"
  yazi "$@" --cwd-file="$tmp"
  if [[ -f "$tmp" ]]; then
    cd "$(cat "$tmp")"
    rm "$tmp"
  fi
}

alias ya="yazi"
```

### Symlink Structure

```
~/.config/zsh/           → repo/configs/zsh/
~/.config/yazi/          → repo/configs/yazi/
~/.config/starship.toml  → repo/configs/starship.toml
```

## CLI Commands

```zsh
macos-setup                 # Launch TUI
macos-setup install         # First-run setup
macos-setup --all           # Install all pending (auto tier)
macos-setup --category=cli  # Install specific category
macos-setup --app=yazi      # Install specific app
macos-setup --reset=yazi    # Mark app as pending, reinstall
macos-setup update          # Git pull repo, reapply configs
macos-setup status          # Show what's installed/pending
```

## Technology Stack

- **Language**: Go
- **TUI**: Bubbletea + Lipgloss + Bubbles (Charm ecosystem)
- **Config parsing**: gopkg.in/yaml.v3
- **State**: JSON file
- **Build**: GitHub Actions, releases for darwin/arm64 and darwin/amd64

## Future Considerations

- Private config repo support (post-auth clone)
- iCloud integration for secrets/personal configs
- Backup existing configs before overwriting
- Export current machine state to apps.yaml
- Profile support (work vs personal machine)

## Resolved Questions

1. **Rosetta apps** - Handle case-by-case when defining apps. Rosetta itself can be an app in the list if needed.
2. **post_install sudo** - Explicit in recipe. If command needs sudo, write `sudo` in the command. Binary runs what's written.
3. **Config conflicts** - Overwrite. If `~/.config/yazi` exists and isn't a symlink, remove and replace with symlink.
4. **Binary updates** - Re-run curl installer. No self-update mechanism.
