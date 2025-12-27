# Add App Skill Reference

Reference docs for the `/add-app` skill. Skill reads all `.md` files in this directory.

## apps.yaml Schema

```yaml
apps:
  <app-key>:                    # lowercase, hyphenated (e.g., visual-studio-code)
    install: brew|cask|npm|mas|shell
    category: cli|dev|ai|apps|utils
    description: string         # short, no period
    tier: required|auto         # optional, omit for auto
    package: string             # optional, if pkg name differs from key
    id: number                  # mas only, App Store ID
    depends: [string]           # optional, list of app keys
    config:                     # optional, dotfile symlink
      source: string            # relative to repo
      dest: string              # absolute path
    zsh: |                      # optional, shell integration
      eval "$(tool init zsh)"
    post_install:               # optional, commands after install
      - command1
      - command2
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
| cli | Command-line tools, terminal utilities |
| dev | Development tools, IDEs, language runtimes |
| ai | AI tools, assistants |
| apps | General macOS applications |
| utils | System utilities |

## Tiers

| Tier | Behavior |
|------|----------|
| required | Auto-installs before TUI launches |
| auto | Normal install, no special handling (default) |
| interactive | Needs user input (e.g., App Store sign-in) |

Omit tier field for auto behavior.

## Common Dependency Patterns

| App Type | Depends On |
|----------|------------|
| npm packages | fnm |
| docker-compose, docker-* | docker |
| language-specific tools | their runtime (e.g., go tools → go) |

## Verification Commands

```bash
# Check if brew package exists
brew search <name>

# Check if cask exists
brew search --cask <name>

# Get package info
brew info <name>
brew info --cask <name>

# Check npm package
npm search <name>
```

## Example Entries

### Basic brew tool
```yaml
jq:
  install: brew
  category: cli
  description: JSON processor
```

### Cask with description
```yaml
visual-studio-code:
  install: cask
  category: dev
  description: Code editor by Microsoft
```

### npm with package name
```yaml
claude-code:
  install: npm
  package: "@anthropic-ai/claude-code"
  category: ai
  description: Claude Code CLI
```

### Tool with zsh integration
```yaml
zoxide:
  install: brew
  category: cli
  description: Smarter cd command
  zsh: |
    eval "$(zoxide init zsh)"
```

### Tool with dependencies and post_install
```yaml
fnm:
  install: brew
  category: dev
  tier: required
  description: Fast Node.js version manager
  zsh: |
    eval "$(fnm env --use-on-cd)"
  post_install:
    - fnm install 24
    - fnm default 24
```

### Mac App Store app
```yaml
amphetamine:
  install: mas
  id: 937984704
  category: utils
  description: Keep Mac awake
```
