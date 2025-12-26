# macos-setup

TUI app for setting up a fresh macOS developer machine.

## Requirements

- **macOS on Apple Silicon** (M1/M2/M3/M4)
- **Xcode Command Line Tools** - must be installed manually before running

## What Gets Installed

The first run automatically installs:
- **Homebrew** - package manager
- **Go** - for building the TUI app

## Install

### 1. Install Xcode Command Line Tools

```zsh
xcode-select --install
```

Complete the installation dialog before proceeding.

### 2. Download macos-setup

```zsh
curl -fsSL https://raw.githubusercontent.com/schmoli/macos-setup/main/install.sh | zsh
```

This downloads the repo to `~/.config/macos-setup/repo` and adds the command to your PATH.

### 3. Run

```zsh
source ~/.zshrc
macos-setup
```

First run installs Homebrew and Go, builds the TUI, then launches it.

## Update

```zsh
macos-setup update
```

This pulls the latest code and rebuilds.

## Uninstall

```zsh
rm -rf ~/.config/macos-setup
rm -f ~/.local/bin/macos-setup
```

Remove the PATH line from `~/.zshrc` if desired.

## Development

```zsh
git clone https://github.com/schmoli/macos-setup.git
cd macos-setup
go mod tidy
go build -o bin/macos-setup ./cmd/macos-setup/
./bin/macos-setup
```
