# macos-setup

TUI app for setting up a fresh macOS developer machine (Apple Silicon only).

## Install

```zsh
curl -fsSL https://raw.githubusercontent.com/schmoli/macos-setup/main/install.sh | zsh
```

If Xcode CLT not installed, it will tell you to run `xcode-select --install` first.

Then:
```zsh
source ~/.zshrc
macos-setup
```

First run installs Homebrew + Go, builds the TUI, and launches it.

## Update

```zsh
cd ~/.config/macos-setup/repo && git pull
rm -f bin/macos-setup  # force rebuild
macos-setup
```

## Requirements

- macOS on Apple Silicon (M1/M2/M3/M4)
- Xcode Command Line Tools
