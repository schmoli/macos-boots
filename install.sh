#!/bin/zsh
# Bootstrap: downloads files, sets up PATH, checks prerequisites
# Usage: curl -fsSL https://raw.githubusercontent.com/schmoli/macos-setup/main/install.sh | zsh

set -e
trap 'echo ""; echo "Installation failed. Check output above."; echo ""' ERR

REPO="schmoli/macos-setup"
REPO_DIR="$HOME/.config/macos-setup/repo"
BINARY_DIR="$HOME/.local/bin"

GREEN='\033[0;32m'
CYAN='\033[0;36m'
DIM='\033[0;90m'
NC='\033[0m'

# Require Apple Silicon
if [[ "$(uname -m)" != "arm64" ]]; then
  echo "Error: Apple Silicon required"
  exit 1
fi

# Require Homebrew
if [[ ! -x "/opt/homebrew/bin/brew" ]]; then
  echo ""
  echo "Error: Homebrew required"
  echo ""
  echo "Install Homebrew: https://brew.sh"
  echo ""
  exit 1
fi

# Set up Homebrew environment in this script
eval "$(/opt/homebrew/bin/brew shellenv)"

echo ""
echo "${GREEN}macos-setup${NC} - Installation"
echo "=========================="
echo ""

# Install Go
if [[ ! -x "$(brew --prefix)/bin/go" ]]; then
  echo "${CYAN}⏳ Installing Go...${NC}"
  brew install -q go
fi
echo "${GREEN}✅ Go${NC}"

# Install fnm
if [[ ! -x "$(brew --prefix)/bin/fnm" ]]; then
  echo "${CYAN}⏳ Installing fnm...${NC}"
  brew install -q fnm
fi
echo "${GREEN}✅ fnm${NC}"

# Set up fnm in this script
eval "$(fnm env --use-on-cd)"

# Install Node.js
NODE_VERSION=24
if ! fnm list 2>/dev/null | grep -q "v${NODE_VERSION}"; then
  echo "${CYAN}⏳ Installing Node.js ${NODE_VERSION}...${NC}"
  fnm install "${NODE_VERSION}"
  fnm default "${NODE_VERSION}"
fi
echo "${GREEN}✅ Node.js ${NODE_VERSION}${NC}"

# Verify installations
command -v go >/dev/null || { echo "Error: Go install failed"; exit 1; }
command -v fnm >/dev/null || { echo "Error: fnm install failed"; exit 1; }
command -v node >/dev/null || { echo "Error: Node install failed"; exit 1; }

# Clone repo
if [[ -d "$REPO_DIR/.git" ]]; then
  echo "${CYAN}⏳ Updating repo...${NC}"
  (cd "$REPO_DIR" && git checkout main --quiet && git pull --quiet origin main)
else
  echo "${CYAN}⏳ Cloning repo...${NC}"
  mkdir -p "$(dirname "$REPO_DIR")"
  git clone --quiet "https://github.com/$REPO/macos-setup.git" "$REPO_DIR"
fi
echo "${GREEN}✅ Repo${NC}"

# Build binary
echo "${CYAN}⏳ Building...${NC}"
BINARY="$REPO_DIR/bin/macos-setup"
mkdir -p "$(dirname "$BINARY")"
(cd "$REPO_DIR" && go mod tidy >/dev/null 2>&1 && go build -o "$BINARY" ./cmd/macos-setup/)
echo "${GREEN}✅ Built${NC}"

# Create wrapper script
mkdir -p "$BINARY_DIR"
cat > "$BINARY_DIR/macos-setup" << 'WRAPPER'
#!/bin/zsh
exec "$HOME/.config/macos-setup/repo/setup.sh" "$@"
WRAPPER
chmod +x "$BINARY_DIR/macos-setup"
echo "${GREEN}✅ Wrapper${NC}"

# Add to PATH if needed
if ! grep -q "$BINARY_DIR" ~/.zshrc 2>/dev/null; then
  echo "" >> ~/.zshrc
  echo "# macos-setup" >> ~/.zshrc
  echo "export PATH=\"$BINARY_DIR:\$PATH\"" >> ~/.zshrc
  echo "${GREEN}✅ Added macos-setup to ~/.zshrc${NC}"
fi

# Add fnm to zshrc if needed
if ! grep -q 'fnm env --use-on-cd' ~/.zshrc 2>/dev/null; then
  echo "" >> ~/.zshrc
  echo "# fnm" >> ~/.zshrc
  echo 'eval "$(fnm env --use-on-cd)"' >> ~/.zshrc
  echo "${GREEN}✅ Added fnm to ~/.zshrc${NC}"
fi

echo ""
echo "${GREEN}✅ Installation complete!${NC}"
INSTALLED_VERSION=$(git -C "$REPO_DIR" rev-parse --short HEAD 2>/dev/null || echo "unknown")
echo "${DIM}Installed version: ${INSTALLED_VERSION}${NC}"
echo ""
echo "Next steps:"
echo ""
echo "    source ~/.zshrc"
echo "    macos-setup cli    # Install CLI tools"
echo ""
