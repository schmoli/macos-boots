# App Folder Structure Design

## Problem

Zsh configs defined inline in `apps.yaml` are only written to disk at install time. If an app is already installed, or if the zsh config is updated, users don't get the changes until they reinstall.

## Solution

Move from single `apps.yaml` to per-app folders. Each app gets its own directory with:
- `app.yaml` - install config, description, post_install, etc.
- `init.zsh` - optional shell config (sourced directly from repo)

## Structure

```
apps/
  cli/
    yazi/
      app.yaml
      init.zsh
    zoxide/
      app.yaml
      init.zsh
    fnm/
      app.yaml
      init.zsh
    jq/
      app.yaml       # no init.zsh needed
  apps/
    rectangle/
      app.yaml
    claude/
      app.yaml
```

## app.yaml Schema

```yaml
install: brew|cask|npm|mas|shell
description: string
tier: required|auto          # optional, defaults to auto
package: string              # optional, if differs from folder name
id: number                   # mas only
depends: [string]            # optional
post_install:                # optional
  - command1
  - command2
```

Note: `zsh` field removed - use `init.zsh` file instead.

## init.zsh

Real zsh file sourced by shell. Lives in repo, updated via git pull.

```zsh
# apps/cli/yazi/init.zsh
alias ya=yazi
```

```zsh
# apps/cli/zoxide/init.zsh
eval "$(zoxide init zsh)"
```

## Shell Integration

`~/.config/macos-setup/init.zsh` becomes:

```zsh
# macos-setup shell integration
for f in ~/.config/macos-setup/repo/apps/*/*/init.zsh(N); do
  source "$f"
done
```

No file generation needed. Git pull = instant config updates.

## Config Loading

Update `internal/config/config.go`:

1. Scan `apps/*/` for category directories
2. Scan `apps/<category>/*/app.yaml` for app configs
3. Infer category from path
4. Infer app name from folder name
5. Merge into same `Config` struct

```go
func Load(baseDir string) (*Config, error) {
    // baseDir = ~/.config/macos-setup/repo/apps
    // scan apps/cli/*/app.yaml, apps/apps/*/app.yaml
}
```

## Migration Steps

1. Create `apps/cli/` and `apps/apps/` directories
2. For each app in current `apps.yaml`:
   - Create `apps/<category>/<name>/app.yaml`
   - If has `zsh:` field, create `apps/<category>/<name>/init.zsh`
3. Update config loader to scan folders
4. Update shell init.zsh path
5. Delete old `apps.yaml`
6. Update `/add-app` skill for new structure

## Benefits

- All app config in one place
- Zsh configs update on git pull
- Extensible: could add `post-install.sh`, `update.sh` later
- Better organization as app count grows
- Real `.zsh` files get syntax highlighting

## Non-Goals

- Separate script files for update/uninstall (keep in app.yaml)
- Custom ordering (alphabetical by folder name)
