# Move fnm/node to Bootstrap Phase

## Overview

Move fnm and node installation from app.yaml to setup.sh bootstrap phase. This ensures npm is available before the Go binary runs, eliminating npm package deferral logic.

## Current Flow

1. `setup.sh` → installs Go → builds binary
2. Binary runs → installs fnm via brew → runs post_install (fnm install 24, fnm default 24)
3. Binary defers npm packages (claude-code) until later runs when node is in PATH

## Proposed Flow

1. `setup.sh` → installs Go → **installs fnm → installs node 24** → builds binary
2. Binary runs → npm already available → installs all packages immediately

## Changes

### Remove fnm from apps

- Delete `apps/cli/fnm/` directory entirely (app.yaml and init.zsh)
- fnm becomes part of bootstrap, not a managed app

### Update setup.sh

Add after Go installation, before build:

```bash
# fnm (Fast Node Manager)
if [[ ! -x "$(brew --prefix)/bin/fnm" ]]; then
  echo "${CYAN}⏳ Installing fnm...${NC}"
  brew install -q fnm
fi
eval "$(fnm env --use-on-cd)"
echo "${GREEN}✅ fnm${NC}"

# Node.js
if ! fnm list | grep -q "v24"; then
  echo "${CYAN}⏳ Installing Node.js 24...${NC}"
  fnm install 24
  fnm default 24
fi
echo "${GREEN}✅ Node.js${NC}"

# Add fnm to zshrc if not there
if ! grep -q 'fnm env --use-on-cd' ~/.zshrc 2>/dev/null; then
  echo "" >> ~/.zshrc
  echo "# fnm" >> ~/.zshrc
  echo 'eval "$(fnm env --use-on-cd)"' >> ~/.zshrc
fi
```

### Update installer.go

Remove npm deferral logic (lines 153-157):

```go
// Before (with deferral):
if len(npmApps) > 0 {
  if !isNpmAvailable() {
    for name := range npmApps {
      result.Deferred = append(result.Deferred, name)
    }
  } else {
    // install npm apps
  }
}

// After (no deferral):
if len(npmApps) > 0 {
  for name, app := range npmApps {
    if err := installNpmApp(name, app, result, verbose); err != nil {
      result.Failed = append(result.Failed, name)
    }
  }
}
```

Remove `Deferred` field from `Result` struct (line 109) if no longer used elsewhere.

## Benefits

1. **Simpler logic** - No npm deferral tracking
2. **Immediate installs** - All packages install on first run
3. **Consistent with Go** - Both Go and node are bootstrap dependencies
4. **Cleaner separation** - Build-time deps vs user apps

## Edge Cases

### npm not in PATH after fnm install
- Solution: `eval "$(fnm env --use-on-cd)"` in setup.sh before node install
- Binary execution gets npm via user's shell (zshrc sources fnm)

### User already has fnm
- Check prevents reinstall: `if [[ ! -x "$(brew --prefix)/bin/fnm" ]]`
- Check prevents re-installing node: `if ! fnm list | grep -q "v24"`

### Upgrade path
- N/A - unreleased, no backward compatibility needed
- Existing `apps/cli/fnm/` deleted, users won't have conflicts

## Testing

1. **Fresh install** - No fnm, no node → setup.sh installs both → binary runs, installs npm packages
2. **Existing fnm** - fnm installed → setup.sh skips brew install → ensures node 24 → binary runs normally
3. **Existing node** - Different version → fnm installs 24, sets default → binary uses node 24
