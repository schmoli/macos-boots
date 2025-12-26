#!/bin/zsh
# Bootstrap: downloads files, sets up PATH, checks prerequisites
# Usage: curl -fsSL https://raw.githubusercontent.com/schmoli/macos-setup/main/install.sh | zsh

set -e

REPO="schmoli/macos-setup"
REPO_DIR="$HOME/.config/macos-setup/repo"
BINARY_DIR="$HOME/.local/bin"

GREEN='\033[0;32m'
BLUE='\033[0;36m'
NC='\033[0m'

echo ""
echo "${GREEN}macos-setup${NC}"
echo "==========="
echo ""

# Require Apple Silicon
if [[ "$(uname -m)" != "arm64" ]]; then
  echo "Error: Apple Silicon required"
  exit 1
fi

# Download repo (use curl+tar since git might not exist yet)
mkdir -p "$REPO_DIR"
echo "Downloading..."
curl -fsSL "https://github.com/$REPO/archive/refs/heads/main.tar.gz" | tar -xz -C "$REPO_DIR" --strip-components=1
echo "${BLUE}✓${NC} Downloaded to ~/.config/macos-setup/repo"

# Create wrapper script
mkdir -p "$BINARY_DIR"
cat > "$BINARY_DIR/macos-setup" << 'WRAPPER'
#!/bin/zsh
exec "$HOME/.config/macos-setup/repo/setup.sh" "$@"
WRAPPER
chmod +x "$BINARY_DIR/macos-setup"
echo "${BLUE}✓${NC} Created ~/.local/bin/macos-setup"

# Add to PATH if needed
if ! grep -q "$BINARY_DIR" ~/.zshrc 2>/dev/null; then
  echo "" >> ~/.zshrc
  echo "# macos-setup" >> ~/.zshrc
  echo "export PATH=\"$BINARY_DIR:\$PATH\"" >> ~/.zshrc
  echo "${BLUE}✓${NC} Added to PATH in ~/.zshrc"
fi

echo ""

# Check CLT and give appropriate instructions
if ! xcode-select -p &>/dev/null; then
  echo "Almost ready! First install Xcode Command Line Tools:"
  echo ""
  echo "    xcode-select --install"
  echo ""
  echo "Then:"
  echo ""
  echo "    source ~/.zshrc"
  echo "    macos-setup"
else
  echo "Ready! Run:"
  echo ""
  echo "    source ~/.zshrc"
  echo "    macos-setup"
fi
echo ""
