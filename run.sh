#!/bin/zsh
# Runner script for macos-setup
# Handles first-time setup (CLT, Homebrew, Go) then launches TUI

set -e

REPO_DIR="$HOME/.config/macos-setup/repo"
BINARY="$REPO_DIR/bin/macos-setup"
STATE_FILE="$HOME/.config/macos-setup/state.json"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;36m'
NC='\033[0m'

print_step() { echo "${BLUE}==>${NC} $1"; }
print_ok() { echo "${GREEN}✓${NC} $1"; }
print_warn() { echo "${YELLOW}!${NC} $1"; }

# Check if first-time setup is needed
needs_setup() {
  ! command -v brew &>/dev/null || \
  ! command -v go &>/dev/null || \
  [[ ! -f "$BINARY" ]]
}

# First-time setup
first_time_setup() {
  echo ""
  echo "${GREEN}macos-setup${NC} - First Time Setup"
  echo "================================"
  echo ""

  # Require Apple Silicon
  if [[ "$(uname -m)" != "arm64" ]]; then
    echo "${RED}Error: This tool only supports Apple Silicon (arm64)${NC}"
    exit 1
  fi

  # 1. Xcode Command Line Tools
  if ! xcode-select -p &>/dev/null; then
    print_step "Installing Xcode Command Line Tools..."
    echo "    This may open a dialog - click Install and wait for it to complete."
    echo ""
    xcode-select --install
    echo ""
    echo "Press Enter when installation is complete..."
    read
  fi
  print_ok "Xcode Command Line Tools"

  # 2. Homebrew
  if ! command -v brew &>/dev/null; then
    print_step "Installing Homebrew..."
    /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
    eval "$(/opt/homebrew/bin/brew shellenv)"
  fi
  print_ok "Homebrew $(brew --version | head -1 | cut -d' ' -f2)"

  # 3. Go
  if ! command -v go &>/dev/null; then
    print_step "Installing Go..."
    brew install go
  fi
  print_ok "Go $(go version | cut -d' ' -f3)"

  # 4. Update repo if we only had tarball before
  if [[ ! -d "$REPO_DIR/.git" ]] && command -v git &>/dev/null; then
    print_step "Converting to git repo..."
    rm -rf "$REPO_DIR"
    git clone "https://github.com/schmoli/macos-setup.git" "$REPO_DIR"
  fi

  # 5. Build binary
  print_step "Building macos-setup..."
  mkdir -p "$(dirname "$BINARY")"
  (cd "$REPO_DIR" && go build -o "$BINARY" ./cmd/macos-setup/)
  print_ok "Built $(basename "$BINARY")"

  echo ""
  echo "${GREEN}Setup complete!${NC} Launching TUI..."
  echo ""
  sleep 1
}

# Main
if needs_setup; then
  first_time_setup
fi

# Ensure Homebrew is in PATH (for this session)
if [[ -f "/opt/homebrew/bin/brew" ]]; then
  eval "$(/opt/homebrew/bin/brew shellenv)"
fi

# Run the TUI
exec "$BINARY" "$@"
