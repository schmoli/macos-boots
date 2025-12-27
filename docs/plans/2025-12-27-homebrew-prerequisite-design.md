# Homebrew as Hard Prerequisite

## Overview

Simplify macos-setup by requiring Homebrew be installed before running the tool. Removes all Homebrew/CLT installation logic from install.sh and setup.sh.

## Current State

Three installation phases:
1. **install.sh** - Downloads repo, checks CLT, provides next steps
2. **setup.sh** - Checks CLT, installs Homebrew if missing, installs Go, builds binary
3. **Go binary** - Handles app installation via brew/cask/npm/mas

## Changes

### Removed Functionality

- setup.sh:16-26 - CLT check and error message
- setup.sh:49-64 - Homebrew installation and zshrc modification
- install.sh:52-66 - CLT check and conditional instructions

### New Behavior

- install.sh checks for homebrew at line 25 (after Apple Silicon check)
- setup.sh checks for homebrew at line 16 (replaces CLT check)
- Both fail immediately if homebrew missing
- Error message format:
  ```bash
  echo ""
  echo "Error: Homebrew required"
  echo ""
  echo "Install Homebrew: https://brew.sh"
  echo ""
  exit 1
  ```

### No Changes

- installer.go - already hardcodes `/opt/homebrew/bin/brew`
- README.md - already lists Homebrew as requirement
- CLAUDE.md - already mentions Homebrew preinstalled in VM

## User Experience

**Before:**
```
install.sh → checks CLT → gives instructions
source ~/.zshrc
macos-setup → checks CLT → installs homebrew → installs Go → builds → runs
```

**After:**
```
install.sh → checks homebrew → fails if missing with brew.sh link
(user installs homebrew manually)
install.sh → downloads repo, updates PATH
source ~/.zshrc
macos-setup → checks homebrew → installs Go → builds → runs
```

## Testing

VM test scenarios:

1. **Fresh macOS + Homebrew installed**
   - Run install.sh → should succeed
   - Run macos-setup → should install Go, build, run

2. **Fresh macOS without Homebrew**
   - Run install.sh → should fail with brew.sh link

3. **Existing installation**
   - Pull latest changes
   - Run macos-setup update → should work normally

## Documentation

No updates needed:
- README.md already lists Homebrew as requirement
- CLAUDE.md already mentions Homebrew preinstalled in VM
