#!/bin/zsh
# Main setup script - run interactively after install.sh
# This has proper stdin so can prompt for sudo

set -e

REPO_DIR="$HOME/.config/macos-setup/repo"
BINARY="$REPO_DIR/bin/macos-setup"

# Just exec the binary if it exists
if [[ -x "$BINARY" ]]; then
  exec "$BINARY" "$@"
fi

# Binary missing - installation incomplete
echo ""
echo "Error: macos-setup not installed"
echo ""
echo "Install: curl -fsSL https://raw.githubusercontent.com/schmoli/macos-setup/main/install.sh | zsh"
echo ""
exit 1
