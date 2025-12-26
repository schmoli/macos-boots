# macos-setup

TUI app for setting up a fresh macOS developer machine (Apple Silicon only).

## Install

```zsh
curl -fsSL https://raw.githubusercontent.com/schmoli/macos-setup/main/install.sh | zsh
```

Then add to PATH (if not already):
```zsh
export PATH="$HOME/.local/bin:$PATH"
```

## Run

```zsh
macos-setup
```

**First run** will install prerequisites:
- Xcode Command Line Tools (may show dialog)
- Homebrew
- Go

Then launches the TUI.

**Subsequent runs** go straight to TUI.

## Update

```zsh
curl -fsSL https://raw.githubusercontent.com/schmoli/macos-setup/main/install.sh | zsh
```

Or manually:
```zsh
cd ~/.config/macos-setup/repo && git pull
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
