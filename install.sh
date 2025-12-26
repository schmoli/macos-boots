#!/bin/zsh
# Bootstrap script for macos-setup
# Usage: curl -fsSL https://raw.githubusercontent.com/schmoli/macos-setup/main/install.sh | zsh

set -e

REPO="schmoli/macos-setup"
REPO_DIR="$HOME/.config/macos-setup/repo"
BINARY_DIR="$HOME/.local/bin"
BINARY="macos-setup"

echo "macos-setup bootstrap"
echo "====================="

# Require Apple Silicon
if [[ "$(uname -m)" != "arm64" ]]; then
  echo "Error: This tool only supports Apple Silicon (arm64)"
  exit 1
fi
echo "Detected: Apple Silicon"

# Install Xcode CLI tools if needed
if ! xcode-select -p &>/dev/null; then
  echo "Installing Xcode Command Line Tools..."
  xcode-select --install
  echo "Press Enter after Xcode tools finish installing..."
  read
fi

# Install Homebrew if needed
if ! command -v brew &>/dev/null; then
  echo "Installing Homebrew..."
  /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"

  # Add to PATH for this session
  eval "$(/opt/homebrew/bin/brew shellenv)"
fi
echo "Homebrew: $(brew --version | head -1)"

# Install Go if needed (for building from source)
if ! command -v go &>/dev/null; then
  echo "Installing Go..."
  brew install go
fi
echo "Go: $(go version)"

# Clone or update repo
mkdir -p "$(dirname "$REPO_DIR")"
if [[ -d "$REPO_DIR/.git" ]]; then
  echo "Updating repo..."
  git -C "$REPO_DIR" pull --rebase
else
  echo "Cloning repo..."
  git clone "https://github.com/$REPO.git" "$REPO_DIR"
fi

# Build binary
echo "Building macos-setup..."
mkdir -p "$BINARY_DIR"
(cd "$REPO_DIR" && go build -o "$BINARY_DIR/$BINARY" ./cmd/macos-setup/)
echo "Installed to: $BINARY_DIR/$BINARY"

# Add to PATH hint
if [[ ":$PATH:" != *":$BINARY_DIR:"* ]]; then
  echo ""
  echo "Add to your PATH (add to ~/.zshrc):"
  echo "  export PATH=\"$BINARY_DIR:\$PATH\""
fi

echo ""
echo "Run 'macos-setup' to start"
echo "Run this script again to update"
