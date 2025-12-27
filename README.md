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
macos-setup update   # Upgrade installed apps
```

## Project Structure

```
apps/
├── cli/             # Terminal tools (brew)
│   └── <name>/
│       ├── app.yaml     # install, description, post_install
│       └── init.zsh     # Shell setup (optional)
└── apps/            # GUI apps (cask)
    └── <name>/
        └── app.yaml
```

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
