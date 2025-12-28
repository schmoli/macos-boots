#!/bin/zsh
# Bootstrap: downloads files, sets up PATH, checks prerequisites
# Usage: curl -fsSL https://raw.githubusercontent.com/schmoli/macos-boots/main/install.sh | zsh

set -e
trap 'echo ""; echo "Installation failed. Check output above."; echo ""' ERR

REPO="schmoli/macos-boots"
REPO_DIR="$HOME/.config/boots/repo"
BINARY_DIR="$HOME/.config/boots/bin"

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
echo "${CYAN}❯ boots${NC}"
echo "${CYAN}  ┏┓ ┏━┓┏━┓╺┳╸┏━┓${NC}"
echo "${CYAN}  ┣┻┓┃ ┃┃ ┃ ┃ ┗━┓${NC}"
echo "${CYAN}  ┗━┛┗━┛┗━┛ ╹ ┗━┛${NC}"
echo ""
echo "${DIM}macOS bootstrapper - Installation${NC}"
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
  corepack enable
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
  git clone --quiet "https://github.com/$REPO.git" "$REPO_DIR"
fi
echo "${GREEN}✅ Repo${NC}"

# Build binary
echo "${CYAN}⏳ Building...${NC}"
BINARY="$BINARY_DIR/boots"
mkdir -p "$BINARY_DIR"
(cd "$REPO_DIR/go" && go mod tidy >/dev/null 2>&1 && go build -o "$BINARY" ./cmd/macos-setup/)
echo "${GREEN}✅ Built${NC}"

# Create initial init.zsh (will be updated by boots when apps installed)
INIT_ZSH="$HOME/.config/boots/init.zsh"
cat > "$INIT_ZSH" << 'INIT'
# boots shell integration (auto-generated)

# Add boots to PATH
export PATH="$HOME/.config/boots/bin:$PATH"

# fnm initialization
eval "$(fnm env --use-on-cd)"
INIT

# Add boots integration to zshrc if needed
BOOTS_LINE='[[ -f ~/.config/boots/init.zsh ]] && source ~/.config/boots/init.zsh'
if ! grep -q "boots/init.zsh" ~/.zshrc 2>/dev/null; then
  echo "" >> ~/.zshrc
  echo "$BOOTS_LINE" >> ~/.zshrc
  echo "${GREEN}✅ Added boots to ~/.zshrc${NC}"
fi

echo ""
echo "${GREEN}✅ Installation complete!${NC}"
INSTALLED_VERSION=$(git -C "$REPO_DIR" rev-parse --short HEAD 2>/dev/null || echo "unknown")
echo "${DIM}Installed version: ${INSTALLED_VERSION}${NC}"
echo ""
echo "Next steps:"
echo ""
echo "    source ~/.zshrc"
echo "    boots cli    # Install CLI tools"
echo ""
