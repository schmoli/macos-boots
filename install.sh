#!/bin/zsh
# Bootstrap script for macos-setup
# Usage: curl -fsSL https://raw.githubusercontent.com/schmoli/macos-setup/main/install.sh | zsh
#
# This just pulls the repo and sets up the wrapper - no installs yet.
# Run `macos-setup` after to start the setup process.

set -e

REPO="schmoli/macos-setup"
REPO_DIR="$HOME/.config/macos-setup/repo"
BINARY_DIR="$HOME/.local/bin"

echo "macos-setup"
echo "==========="

# Clone or update repo
mkdir -p "$(dirname "$REPO_DIR")"
if [[ -d "$REPO_DIR/.git" ]]; then
  echo "Updating repo..."
  git -C "$REPO_DIR" pull --rebase 2>/dev/null || {
    echo "Git not available yet - will update on first run"
  }
else
  # Try git first, fall back to curl + tar if git not available
  if command -v git &>/dev/null; then
    echo "Cloning repo..."
    git clone "https://github.com/$REPO.git" "$REPO_DIR"
  else
    echo "Downloading repo..."
    mkdir -p "$REPO_DIR"
    curl -fsSL "https://github.com/$REPO/archive/refs/heads/main.tar.gz" | tar -xz -C "$REPO_DIR" --strip-components=1
  fi
fi

# Create wrapper script
mkdir -p "$BINARY_DIR"
cat > "$BINARY_DIR/macos-setup" << 'WRAPPER'
#!/bin/zsh
exec "$HOME/.config/macos-setup/repo/run.sh" "$@"
WRAPPER
chmod +x "$BINARY_DIR/macos-setup"

echo "Installed to: $BINARY_DIR/macos-setup"

# Add to PATH if not already there
if ! grep -q "$BINARY_DIR" ~/.zshrc 2>/dev/null; then
  echo "" >> ~/.zshrc
  echo "# macos-setup" >> ~/.zshrc
  echo "export PATH=\"$BINARY_DIR:\$PATH\"" >> ~/.zshrc
  echo "Added $BINARY_DIR to PATH in ~/.zshrc"
fi

echo ""
echo "Run: macos-setup"
echo "(You may need to restart your terminal or run: source ~/.zshrc)"
