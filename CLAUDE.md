# macos-setup Development

## Workflow

Work happens here (local dev machine). Testing on Parallels VM with fresh macOS Tahoe 26.3 + Homebrew preinstalled. Never test locally - always push and pull on VM.

1. Make changes locally
2. Commit and push to origin
3. On Parallels VM: `macos-setup update` (auto-pulls) or fresh install

## Project Structure

```
apps/
├── cli/                    # brew install
└── apps/                   # brew install --cask
```

Each app folder:

```yaml
# app.yaml (required)
install: brew|cask|mas
description: Short desc
post_install:               # Optional: one-time commands
  - command here
depends_on: other-app       # Optional
```

```zsh
# init.zsh (optional) - sourced in shell
eval "$(tool init zsh)"
alias x='tool'
```

## Conventions

- Branches: `feat/`, `fix/`, `chore/` prefixes
- Commits: conventional format (`feat(apps):`, `fix(installer):`, etc.)
- Prefer rebase for PRs/merges
- Always check origin before branching

## Skills

### /add-app

Adds apps with fuzzy brew matching, category inference, auto-commit.

```
/add-app <name> [to <category>] [depends on <dep>] [wip]
```

Files:
- Skill definition: `.claude/commands/add-app.md`
- Skill docs: `docs/skills/add-app/README.md`
- History log: `docs/skills/add-app/history.md`

To update the skill, edit `.claude/commands/add-app.md`.
