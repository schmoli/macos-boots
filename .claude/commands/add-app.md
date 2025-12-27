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
   - Infer from app type (CLI tools → cli, GUI → apps, etc.)
   - If uncertain, ask with choices: cli / dev / ai / apps / utils

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
   ```bash
   git add apps.yaml
   git commit -m "feat(apps): add <app> to <category>"
   git push
   ```

10. **Log** - Append to `docs/skills/add-app/history.md`:
    ```
    YYYY-MM-DD | app-key | category | any notes
    ```

## Examples

```
/add-app vscode to dev
/add-app wip jq to cli
/add-app docker-compose to dev depends on docker
/add-app something for markdown editing
```
