#!/bin/zsh
# Main setup script - run interactively after install.sh
# This has proper stdin so can prompt for sudo

set -e

REPO_DIR="$HOME/.config/macos-setup/repo"
BINARY="$REPO_DIR/bin/macos-setup"

GREEN='\033[0;32m'
BLUE='\033[0;36m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Check CLT first
if ! xcode-select -p &>/dev/null; then
  echo ""
  echo "${YELLOW}Xcode Command Line Tools required${NC}"
  echo ""
  echo "Run:  xcode-select --install"
  echo ""
  echo "Then re-run:  macos-setup"
  echo ""
  exit 1
fi

# If binary exists and is up to date, just run it
if [[ -x "$BINARY" ]]; then
  exec "$BINARY" "$@"
fi

# First time setup - install dependencies and build
echo ""
echo "${GREEN}macos-setup${NC} - First Time Setup"
echo "================================"
echo ""

# Homebrew
if [[ ! -x "/opt/homebrew/bin/brew" ]]; then
  echo "${BLUE}→${NC} Installing Homebrew..."
  /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
fi
eval "$(/opt/homebrew/bin/brew shellenv)"
echo "${BLUE}✓${NC} Homebrew"

# Go
if [[ ! -x "$(brew --prefix)/bin/go" ]]; then
  echo "${BLUE}→${NC} Installing Go..."
  brew install go
fi
echo "${BLUE}✓${NC} Go"

# Convert tarball to git repo if needed
if [[ ! -d "$REPO_DIR/.git" ]]; then
  echo "${BLUE}→${NC} Setting up git repo..."
  local tmp_dir=$(mktemp -d)
  git clone --quiet "https://github.com/schmoli/macos-setup.git" "$tmp_dir"
  rm -rf "$REPO_DIR"
  mv "$tmp_dir" "$REPO_DIR"
fi
echo "${BLUE}✓${NC} Repo"

# Build
echo "${BLUE}→${NC} Building..."
mkdir -p "$(dirname "$BINARY")"
(cd "$REPO_DIR" && go build -o "$BINARY" ./cmd/macos-setup/)
echo "${BLUE}✓${NC} Built"

echo ""
echo "${GREEN}Setup complete!${NC}"
echo ""

# Run the TUI
exec "$BINARY" "$@"
