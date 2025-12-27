# Add App Skill Reference

Reference docs for the `/add-app` skill. Skill reads all `.md` files in this directory.

## Folder Structure

Each app lives in `apps/<category>/<name>/`:

```
apps/
  cli/
    jq/
      app.yaml
    zoxide/
      app.yaml
      init.zsh
  apps/
    rectangle/
      app.yaml
```

## app.yaml Schema

```yaml
install: brew|cask|npm|mas|shell
description: string         # short, no period
tier: required|auto         # optional, omit for auto
package: string             # optional, if pkg name differs from key
id: number                  # mas only, App Store ID
depends: [string]           # optional, list of app keys
post_install:               # optional, commands after install
  - command1
  - command2
```

Note: `category` is inferred from folder path. `zsh` content goes in separate `init.zsh` file.

## init.zsh

Optional file for shell integration (aliases, eval commands). Sourced directly from repo.

```zsh
# apps/cli/zoxide/init.zsh
eval "$(zoxide init zsh)"
```

## Install Types

| Type | Source | Example |
|------|--------|---------|
| brew | Homebrew formula | jq, ripgrep, fzf |
| cask | Homebrew cask (GUI apps) | visual-studio-code, docker |
| npm | npm global package | @anthropic-ai/claude-code |
| mas | Mac App Store | amphetamine (needs id field) |
| shell | No binary, zsh integration only | custom aliases |

## Categories

| Category | Description |
|----------|-------------|
| cli | Command-line tools, terminal utilities, dev tools |
| apps | GUI applications (.app bundles, casks, App Store) |

**Inference**: cask/mas → apps, brew/npm/shell → cli

## Tiers

| Tier | Behavior |
|------|----------|
| required | Auto-installs on first run (fnm, mas) |
| auto | Normal install (default) |

Omit tier field for auto behavior.

## Common Dependency Patterns

| App Type | Depends On |
|----------|------------|
| npm packages | fnm |
| docker-compose, docker-* | docker |
| language-specific tools | their runtime (e.g., go tools → go) |

## Verification Commands

```bash
# Primary: use brew info (more reliable for exact matches)
brew info <name>
brew info --cask <name>

# Fallback: search for suggestions if info fails
brew search <name>

# Check npm package
npm search <name>
```

## Example Entries

### Basic brew tool
```
apps/cli/jq/app.yaml:
```
```yaml
install: brew
description: JSON processor
```

### Cask (GUI app)
```
apps/apps/visual-studio-code/app.yaml:
```
```yaml
install: cask
description: Code editor by Microsoft
```

### npm with package name
```
apps/cli/claude-code/app.yaml:
```
```yaml
install: npm
package: "@anthropic-ai/claude-code"
description: Claude Code CLI
```

### Tool with zsh integration
```
apps/cli/zoxide/app.yaml:
```
```yaml
install: brew
description: Smarter cd command
```
```
apps/cli/zoxide/init.zsh:
```
```zsh
eval "$(zoxide init zsh)"
```

### Tool with dependencies and post_install
```
apps/cli/fnm/app.yaml:
```
```yaml
install: brew
tier: required
description: Fast Node.js version manager
post_install:
  - fnm install 24
  - fnm default 24
```
```
apps/cli/fnm/init.zsh:
```
```zsh
eval "$(fnm env --use-on-cd)"
```

### Mac App Store app
```
apps/apps/amphetamine/app.yaml:
```
```yaml
install: mas
id: 937984704
description: Keep Mac awake
```
