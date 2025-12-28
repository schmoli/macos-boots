```
 ____              _
|  _ \            | |
| |_) | ___   ___ | |_ ___
|  _ < / _ \ / _ \| __/ __|
| |_) | (_) | (_) | |_\__ \
|____/ \___/ \___/ \__|___/
```

**macOS Bootstrapper** - Get your dev machine up and running in minutes.

## Requirements

- macOS Tahoe 26.3+ (Apple Silicon)
- Homebrew installed
- Xcode Command Line Tools

## Install

```zsh
curl -fsSL https://raw.githubusercontent.com/schmoli/macos-boots/main/install.sh | zsh
source ~/.zshrc
boots cli      # Install CLI tools
```

First-run bootstrap (Go, fnm, Node, build) happens during curl | sh.

## Commands

```zsh
boots          # Show install status
boots all      # Install everything
boots cli      # Install CLI tools only
boots apps     # Install desktop apps only
boots mas      # Install App Store apps only
boots update   # Upgrade installed apps
boots status   # Show install status (same as no args)
boots help     # Show help

# Flags
boots cli -v   # Verbose mode (show command details on failure)
```

## Project Structure

```
apps/
├── cli/             # Terminal tools (brew/npm)
│   └── <name>/
│       ├── app.yaml     # Config (see below)
│       └── init.zsh     # Shell setup (optional)
└── apps/            # GUI apps (cask)
    └── <name>/
        └── app.yaml
```

### app.yaml Fields

```yaml
install: brew|cask|npm|mas    # Required: install method
description: Tool description  # Required: short description
package: npm-package-name      # Optional: if different from folder name
depends:                       # Optional: dependencies (must install first)
  - docker
post_install:                  # Optional: commands to run after install
  - command here
```

### init.zsh (Optional)

Shell integration file sourced automatically at shell startup. Use for:
- Tool initialization: `eval "$(tool init zsh)"`
- Aliases and functions
- Completions

All `apps/*/*/init.zsh` files are auto-sourced via `~/.config/boots/init.zsh`.

## Adding Apps

Use `/add-app` in Claude Code:

```
/add-app htop to cli
/add-app vscode to apps
/add-app docker-compose to cli depends on docker
```

## Uninstall

```zsh
rm -rf ~/.config/boots
rm -f ~/.local/bin/boots
```

Remove the PATH line from `~/.zshrc` if desired.
