---
name: add-app
description: Add a new app to macos-setup with fuzzy matching, dep inference, auto-commit
---

## Syntax

```
/add-app <name> [to <category>] [depends on <dep>] [wip]
```

Keywords: `to`, `depends on`, `wip` (order flexible)

## Process

1. **Read docs** - Glob and read all `docs/skills/add-app/*.md` (README.md first)

2. **Parse input** - Extract:
   - App name (required)
   - Category via `to <cat>` (optional)
   - Dependencies via `depends on <dep>` (optional)
   - `wip` flag (skip commit if present)

3. **Resolve app**
   - First try `brew info <name>` (more reliable for exact matches)
   - If not found, try `brew info --cask <name>`
   - If neither works, fall back to `brew search` for suggestions
   - If multiple/ambiguous, ask user to clarify
   - Infer install type from source (formula → brew, cask → cask)

4. **Resolve category** - If not provided:
   - Infer from install type: cask/mas → apps, brew/npm/shell → cli
   - If uncertain, ask with choices: cli / apps

5. **Resolve dependencies**
   - npm packages → depends on fnm
   - docker-* → depends on docker
   - If uncertain or explicit `depends on` without target, ask

6. **Generate description** - From `brew info` or infer from purpose

7. **Show preview** - Display YAML block:
   ```yaml
   app-key:
     install: type
     category: cat
     description: desc
   ```
   Ask: "Add this to apps.yaml?"

8. **Apply** - Edit apps.yaml, insert in alphabetical order within category section

9. **Commit** (unless `wip`):
   - First check `git diff apps.yaml` for ALL uncommitted app additions
   - Parse diff to find all new app keys (lines starting with `+  <name>:`)
   - Chain multiple apps in commit message: `feat(apps): add app1, app2 to <category>`
   - If apps span multiple categories, group by category or use generic message
   ```bash
   git add apps.yaml docs/skills/add-app/history.md
   git commit -m "feat(apps): add app1, app2 to cli"
   git push
   ```

10. **Log** - Append to `docs/skills/add-app/history.md`:
    ```
    YYYY-MM-DD | app-key | category | any notes
    ```

## Examples

```
/add-app vscode to apps
/add-app docker-compose to cli depends on docker
/add-app something for markdown editing
```

### Batching with wip

```
/add-app wip jq to cli       # adds jq, no commit
/add-app wip yq to cli       # adds yq, no commit
/add-app bat to cli          # adds bat, commits all three
                             # → "feat(apps): add jq, yq, bat to cli"
```
