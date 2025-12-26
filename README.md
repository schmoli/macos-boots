# macos-setup

TUI app for setting up a fresh macOS developer machine (Apple Silicon only).

## Install

**First, install Xcode Command Line Tools** (if not already):
```zsh
xcode-select --install
```

**Then run:**
```zsh
curl -fsSL https://raw.githubusercontent.com/schmoli/macos-setup/main/install.sh | zsh
source ~/.zshrc
macos-setup
```

This installs Homebrew, Go, clones the repo, builds the binary, and adds to PATH.

## Update

```zsh
curl -fsSL https://raw.githubusercontent.com/schmoli/macos-setup/main/install.sh | zsh
```

## Development

```zsh
git clone https://github.com/schmoli/macos-setup.git
cd macos-setup
go build -o bin/macos-setup ./cmd/macos-setup/
./bin/macos-setup
```

## Requirements

- macOS on Apple Silicon (M1/M2/M3/M4)
- Xcode Command Line Tools
