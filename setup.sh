#!/bin/zsh
# Main setup script - run interactively after install.sh
# This has proper stdin so can prompt for sudo

set -e

REPO_DIR="$HOME/.config/macos-setup/repo"
BINARY="$REPO_DIR/bin/macos-setup"

GREEN='\033[0;32m'
CYAN='\033[0;36m'
YELLOW='\033[1;33m'
DIM='\033[0;90m'
NC='\033[0m'

# Check Homebrew
if [[ ! -x "/opt/homebrew/bin/brew" ]]; then
  echo ""
  echo "Error: Homebrew required"
  echo ""
  echo "Install Homebrew: https://brew.sh"
  echo ""
  exit 1
fi

# If binary exists, just run it
if [[ -x "$BINARY" ]]; then
  exec "$BINARY" "$@"
fi

# If arguments passed but no binary, need first time setup
if [[ $# -gt 0 ]]; then
  echo ""
  echo "${YELLOW}⚡ First time setup required${NC}"
  echo ""
  echo "Run:  macos-setup"
  echo ""
  exit 1
fi

# First time setup - install dependencies and build
echo ""
echo "${GREEN}macos-setup${NC} - First Time Setup"
echo "================================"
echo ""

# Set up Homebrew environment
eval "$(/opt/homebrew/bin/brew shellenv)"

# Go
if [[ ! -x "$(brew --prefix)/bin/go" ]]; then
  echo "${CYAN}⏳ Installing Go...${NC}"
  brew install -q go
fi
echo "${GREEN}✅ Go${NC}"

# Convert tarball to git repo if needed
if [[ ! -d "$REPO_DIR/.git" ]]; then
  echo "${CYAN}⏳ Setting up repo...${NC}"
  tmp_dir=$(mktemp -d)
  git clone --quiet "https://github.com/schmoli/macos-setup.git" "$tmp_dir"
  rm -rf "$REPO_DIR"
  mv "$tmp_dir" "$REPO_DIR"
fi
echo "${GREEN}✅ Repo${NC}"

# Build
echo "${CYAN}⏳ Building...${NC}"
mkdir -p "$(dirname "$BINARY")"
(cd "$REPO_DIR" && go mod tidy >/dev/null 2>&1 && go build -o "$BINARY" ./cmd/macos-setup/)
echo "${GREEN}✅ Built${NC}"

echo ""
echo "${GREEN}✅ Setup complete!${NC}"
echo ""

# Run the CLI
exec "$BINARY" "$@"
