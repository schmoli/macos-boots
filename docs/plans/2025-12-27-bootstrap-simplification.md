# Bootstrap Simplification Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Move first-run bootstrap from setup.sh into install.sh, reducing installation steps from 5 to 3.

**Architecture:** Enhance install.sh to handle all dependency installation (Go, fnm, Node) and binary building during curl | sh. Simplify setup.sh to just exec the binary.

**Tech Stack:** Bash/Zsh, Homebrew, fnm, Go

---

## Task 1: Simplify setup.sh

**Files:**
- Modify: `setup.sh`

**Step 1: Read current setup.sh**

Run: `cat setup.sh`
Expected: Current first-run logic present

**Step 2: Replace with minimal wrapper**

Edit `setup.sh` to:

```zsh
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
```

**Step 3: Verify syntax**

Run: `zsh -n setup.sh`
Expected: No syntax errors

**Step 4: Commit**

```bash
git add setup.sh
git commit -m "refactor(installer): simplify setup.sh to exec-only wrapper"
```

---

## Task 2: Enhance install.sh with bootstrap

**Files:**
- Modify: `install.sh`

**Step 1: Read current install.sh**

Run: `cat install.sh`
Expected: Current tarball download and wrapper creation logic

**Step 2: Add bootstrap logic after existing checks**

After line 34 (Homebrew check), add:

```zsh
# Set up Homebrew environment in this script
eval "$(/opt/homebrew/bin/brew shellenv)"

GREEN='\033[0;32m'
CYAN='\033[0;36m'
NC='\033[0m'

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
  fnm install ${NODE_VERSION}
  fnm default ${NODE_VERSION}
fi
echo "${GREEN}✅ Node.js ${NODE_VERSION}${NC}"

# Verify installations
command -v go >/dev/null || { echo "Error: Go install failed"; exit 1; }
command -v fnm >/dev/null || { echo "Error: fnm install failed"; exit 1; }
command -v node >/dev/null || { echo "Error: Node install failed"; exit 1; }
```

**Step 3: Replace tarball download with git clone**

Replace lines 36-40 (tarball download) with:

```zsh
# Clone repo
if [[ -d "$REPO_DIR/.git" ]]; then
  echo "${CYAN}⏳ Updating repo...${NC}"
  (cd "$REPO_DIR" && git pull --quiet origin main)
else
  echo "${CYAN}⏳ Cloning repo...${NC}"
  mkdir -p "$(dirname "$REPO_DIR")"
  git clone --quiet "https://github.com/$REPO/macos-setup.git" "$REPO_DIR"
fi
echo "${GREEN}✅ Repo${NC}"
```

**Step 4: Add binary build before wrapper creation**

After repo clone/update, before wrapper creation (before line 42), add:

```zsh
# Build binary
echo "${CYAN}⏳ Building...${NC}"
BINARY="$REPO_DIR/bin/macos-setup"
mkdir -p "$(dirname "$BINARY")"
(cd "$REPO_DIR" && go mod tidy >/dev/null 2>&1 && go build -o "$BINARY" ./cmd/macos-setup/)
echo "${GREEN}✅ Built${NC}"
```

**Step 5: Update wrapper script path**

The wrapper at line 44-47 should now point to the built binary.
Wrapper is already correct (points to setup.sh which execs the binary).

**Step 6: Add fnm to zshrc**

After PATH addition (line 52-57), add:

```zsh
# Add fnm to zshrc if needed
if ! grep -q 'fnm env --use-on-cd' ~/.zshrc 2>/dev/null; then
  echo "" >> ~/.zshrc
  echo "# fnm" >> ~/.zshrc
  echo 'eval "$(fnm env --use-on-cd)"' >> ~/.zshrc
  echo "${GREEN}✅ Added fnm to ~/.zshrc${NC}"
fi
```

**Step 7: Update final instructions**

Replace lines 59-64 with:

```zsh
echo ""
echo "${GREEN}✅ Installation complete!${NC}"
echo ""
echo "Next steps:"
echo ""
echo "    source ~/.zshrc"
echo "    macos-setup cli    # Install CLI tools"
echo ""
```

**Step 8: Add error trap at top**

After `set -e` (line 5), add:

```zsh
trap 'echo ""; echo "Installation failed. Check output above."; echo ""' ERR
```

**Step 9: Verify syntax**

Run: `zsh -n install.sh`
Expected: No syntax errors

**Step 10: Commit**

```bash
git add install.sh
git commit -m "feat(installer): move bootstrap into install.sh

- Install Go, fnm, Node during curl | sh
- Clone git repo instead of tarball
- Build binary immediately
- Add fnm to zshrc
- Reduces install steps from 5 to 3"
```

---

## Task 3: Update README

**Files:**
- Modify: `README.md`

**Step 1: Update install instructions**

Replace lines 13-19 with:

```markdown
```zsh
curl -fsSL https://raw.githubusercontent.com/schmoli/macos-setup/main/install.sh | zsh
source ~/.zshrc
macos-setup cli      # Install CLI tools
```

First-run bootstrap (Go, fnm, Node, build) happens during curl | sh.
```

**Step 2: Remove "first run" explanation**

Remove line 19: "First run installs Go, builds the CLI, then shows status."

**Step 3: Commit**

```bash
git add README.md
git commit -m "docs: update install instructions for simplified flow"
```

---

## Testing Checklist

After implementation, verify:

- [ ] `zsh -n install.sh` has no syntax errors
- [ ] `zsh -n setup.sh` has no syntax errors
- [ ] README reflects new 3-step flow
- [ ] All files committed

**Manual testing on Parallels VM:**
- [ ] Fresh install: curl | sh completes without errors
- [ ] Binary exists at ~/.config/macos-setup/repo/bin/macos-setup
- [ ] zshrc has both PATH and fnm entries
- [ ] After `source ~/.zshrc`, `which macos-setup` works
- [ ] After `source ~/.zshrc`, `which node` works
- [ ] `macos-setup cli` runs immediately without "first run setup" message
- [ ] Re-running curl | sh is idempotent (skips already-installed deps)
