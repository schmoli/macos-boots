# macos-setup

TUI app for setting up a fresh macOS developer machine (Apple Silicon only).

## Install

```zsh
curl -fsSL https://raw.githubusercontent.com/schmoli/macos-setup/main/install.sh | zsh
```

This will:
1. Install Xcode Command Line Tools (if needed)
2. Install Homebrew (if needed)
3. Install Go (if needed)
4. Clone this repo to `~/.config/macos-setup/repo/`
5. Build and install binary to `~/.local/bin/macos-setup`

## Update

Run the install script again - it pulls latest and rebuilds.

## Usage

```zsh
macos-setup
```

## Development

```zsh
# Clone
git clone https://github.com/schmoli/macos-setup.git
cd macos-setup

# Build
go build -o macos-setup ./cmd/macos-setup/

# Run
./macos-setup
```

## Requirements

- macOS on Apple Silicon (M1/M2/M3/M4)
