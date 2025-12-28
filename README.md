# macos-setup

CLI tool for setting up a fresh macOS developer machine.

## Requirements

- macOS Tahoe 26.3+ (Apple Silicon)
- Homebrew installed
- Xcode Command Line Tools

## Install

```zsh
curl -fsSL https://raw.githubusercontent.com/schmoli/macos-setup/main/install.sh | zsh
source ~/.zshrc
macos-setup cli      # Install CLI tools
```

First-run bootstrap (Go, fnm, Node, build) happens during curl | sh.

## Commands

```zsh
macos-setup          # Show install status
macos-setup all      # Install everything
macos-setup cli      # Install CLI tools only
macos-setup apps     # Install desktop apps only
macos-setup mas      # Install App Store apps only
macos-setup update   # Upgrade installed apps
macos-setup status   # Show install status (same as no args)
macos-setup help     # Show help

# Flags
macos-setup cli -v   # Verbose mode (show command details on failure)
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

All `apps/*/*/init.zsh` files are auto-sourced via `~/.config/macos-setup/init.zsh`.

## Adding Apps

Use `/add-app` in Claude Code:

```
/add-app htop to cli
/add-app vscode to apps
/add-app docker-compose to cli depends on docker
```

## Uninstall

```zsh
rm -rf ~/.config/macos-setup
rm -f ~/.local/bin/macos-setup
```

Remove the PATH line from `~/.zshrc` if desired.
