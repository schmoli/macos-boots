# Bootstrap Simplification Design

**Date:** 2025-12-27
**Goal:** Reduce installation steps by moving first-run bootstrap into curl | sh

## Problem

Current flow requires too many steps:
```zsh
curl | sh
source ~/.zshrc
macos-setup              # First run: installs fnm, node, repo, builds
source ~/.zshrc          # Again to get node/npm in PATH
macos-setup cli          # Actually installs packages
```

## Solution

Move all first-run bootstrap into install.sh:
```zsh
curl | sh                # Installs everything, builds binary
source ~/.zshrc          # Get node/npm in PATH
macos-setup cli          # Works immediately
```

## Design

### install.sh Changes

**New responsibilities:**
1. Check prerequisites (Apple Silicon, Homebrew)
2. Set up Homebrew environment in script
3. Install Go (brew install go)
4. Install fnm (brew install fnm)
5. Set up fnm in script context (`eval "$(fnm env)"`)
6. Install Node.js 24 (fnm install 24)
7. Clone full git repo (upgrade from tarball download)
8. Build binary (go build)
9. Create wrapper script
10. Update zshrc (PATH + fnm)

**Key implementation details:**
- Use `brew --prefix` for paths
- `eval "$(fnm env --use-on-cd)"` within script enables fnm commands
- Check before installing (idempotent)
- `brew install -q` works in non-interactive piped context
- Build binary before script exits

**Error handling:**
```zsh
set -e
trap 'echo ""; echo "Installation failed. Check output above."; echo ""' ERR
```

**Verification before build:**
```zsh
command -v go >/dev/null || { echo "Error: Go install failed"; exit 1; }
command -v fnm >/dev/null || { echo "Error: fnm install failed"; exit 1; }
command -v node >/dev/null || { echo "Error: Node install failed"; exit 1; }
```

### setup.sh Simplification

**New setup.sh (minimal wrapper):**
```zsh
#!/bin/zsh
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

**What we remove:**
- All first-run setup logic (Go, fnm, Node, build)
- First-run detection checks
- Homebrew checks (done in install.sh)
- zshrc modifications (done in install.sh)

**Why this works:**
- Binary always exists after install.sh completes
- No need for "run macos-setup first" special case
- Much simpler logic

## Testing

**Test scenarios:**

1. **Fresh install (primary):**
   - Parallels VM with Homebrew pre-installed
   - Run curl | sh
   - Verify: binary built, zshrc updated, no errors
   - Source zshrc, run `macos-setup cli`

2. **Re-run install.sh (idempotency):**
   - After successful install, run curl | sh again
   - Should skip already-installed deps
   - Should handle existing git repo gracefully

3. **Partial failure recovery:**
   - Kill install.sh midway (Ctrl+C)
   - Re-run should pick up where it left off

4. **Parallel testing:**
   - Keep VM snapshot with current flow
   - Test new flow on fresh VM
   - Compare final state

**Testing checklist:**
- [ ] Fresh install completes without errors
- [ ] Binary exists at ~/.config/macos-setup/repo/bin/macos-setup
- [ ] zshrc has PATH and fnm entries
- [ ] After source, `which macos-setup` works
- [ ] After source, `which node` works
- [ ] `macos-setup cli` runs immediately
- [ ] Re-running install.sh is safe

## Benefits

- Simpler UX: 3 steps → 2 steps
- One less source command
- No confusing "run macos-setup first" step
- Faster dev iteration on Parallels VM
- More straightforward error path
