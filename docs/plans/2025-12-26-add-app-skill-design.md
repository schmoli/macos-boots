# /add-app Skill Design

## Overview

Project-local Claude Code skill to streamline adding apps to `apps.yaml` with AI-assisted fuzzy matching, dependency inference, and auto-commit.

## Components

- **Skill file**: `.claude/skills/add-app.md` - thin, process-focused
- **Knowledge docs**: `docs/skills/add-app/` - growing reference
  - `README.md` - schema, install types, categories, patterns
  - `patterns.md` - learned patterns (grows over time)
  - `gotchas.md` - things that went wrong
  - `history.md` - log of additions

## Syntax

```
/add-app <name> [to <category>] [depends on <dep>] [wip]
```

- Keywords: `to`, `depends on`, `wip`
- Order flexible, missing parts get prompted
- `wip` skips commit for batching

## Workflow

1. **Parse input** - extract app, category, deps, wip flag
2. **Read docs** - glob `docs/skills/add-app/*.md`, README first
3. **Resolve app** - fuzzy match, verify via `brew search`, ask if ambiguous
4. **Resolve category** - infer or ask (cli/dev/ai/apps/utils)
5. **Resolve deps** - infer obvious (npm→fnm, docker-*→docker), ask if uncertain
6. **Generate description** - from brew info or context
7. **Show preview** - structured YAML, get confirmation
8. **Edit apps.yaml** - insert in correct category section
9. **Commit** (unless wip) - `feat(apps): add <app> to <category>`, push
10. **Log** - append to history.md

## YAML Fields

| Field | Handling |
|-------|----------|
| install | Required - infer from source (brew/cask/npm/mas/shell) |
| category | Required - from `to <cat>` or ask |
| description | Auto-generate from brew info |
| tier | Omit unless needed (defaults to auto) |
| package | Only if name differs from key |
| depends | Infer obvious, ask if uncertain |
| config, zsh, post_install | Prompt if relevant |

## Example Interaction

```
> /add-app vscode to dev

Reading docs/skills/add-app/...

Resolving "vscode"...
  → brew search: visual-studio-code (cask)

Preview:
  visual-studio-code:
    install: cask
    category: dev
    description: Code editor by Microsoft

Add this to apps.yaml? (y/n)

> y

✓ Added to apps.yaml
✓ Committed: feat(apps): add visual-studio-code to dev
✓ Pushed to origin/main
✓ Logged to history.md
```

## Commit Format

Conventional commits: `feat(apps): add <app> to <category>`
