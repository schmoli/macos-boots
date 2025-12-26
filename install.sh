#!/bin/zsh
# Bootstrap script for macos-setup
# Usage: curl -fsSL https://raw.githubusercontent.com/schmoli/macos-setup/main/install.sh | zsh

set -e

REPO="schmoli/macos-setup"
REPO_DIR="$HOME/.config/macos-setup/repo"
BINARY_DIR="$HOME/.local/bin"
BINARY="$REPO_DIR/bin/macos-setup"

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;36m'
NC='\033[0m'

echo ""
echo "${GREEN}macos-setup${NC}"
echo "==========="

# Require Apple Silicon
if [[ "$(uname -m)" != "arm64" ]]; then
  echo "Error: Apple Silicon only"
  exit 1
fi

# Require Xcode CLI Tools - if missing, tell user and exit
if ! xcode-select -p &>/dev/null; then
  echo ""
  echo "${YELLOW}Xcode Command Line Tools required${NC}"
  echo ""
  echo "Run this command and complete the installation:"
  echo ""
  echo "    xcode-select --install"
  echo ""
  echo "Then re-run this script."
  exit 1
fi
echo "${BLUE}✓${NC} Xcode Command Line Tools"

# Install Homebrew if needed
if ! command -v brew &>/dev/null; then
  echo "${BLUE}→${NC} Installing Homebrew..."
  /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
  eval "$(/opt/homebrew/bin/brew shellenv)"
fi
echo "${BLUE}✓${NC} Homebrew"

# Install Go if needed
if ! command -v go &>/dev/null; then
  echo "${BLUE}→${NC} Installing Go..."
  brew install go
fi
echo "${BLUE}✓${NC} Go"

# Clone or update repo
mkdir -p "$(dirname "$REPO_DIR")"
if [[ -d "$REPO_DIR/.git" ]]; then
  echo "${BLUE}→${NC} Updating repo..."
  git -C "$REPO_DIR" pull --rebase --quiet
else
  echo "${BLUE}→${NC} Cloning repo..."
  git clone --quiet "https://github.com/$REPO.git" "$REPO_DIR"
fi
echo "${BLUE}✓${NC} Repo"

# Build binary
echo "${BLUE}→${NC} Building..."
mkdir -p "$(dirname "$BINARY")"
(cd "$REPO_DIR" && go build -o "$BINARY" ./cmd/macos-setup/)
echo "${BLUE}✓${NC} Built"

# Create wrapper in PATH
mkdir -p "$BINARY_DIR"
cat > "$BINARY_DIR/macos-setup" << WRAPPER
#!/bin/zsh
eval "\$(/opt/homebrew/bin/brew shellenv)"
exec "$BINARY" "\$@"
WRAPPER
chmod +x "$BINARY_DIR/macos-setup"

# Add to PATH if needed
if ! grep -q "$BINARY_DIR" ~/.zshrc 2>/dev/null; then
  echo "" >> ~/.zshrc
  echo "# macos-setup" >> ~/.zshrc
  echo "export PATH=\"$BINARY_DIR:\$PATH\"" >> ~/.zshrc
fi

echo ""
echo "${GREEN}Ready!${NC}"
echo ""
echo "Run:  source ~/.zshrc && macos-setup"
