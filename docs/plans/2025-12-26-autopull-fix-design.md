# AutoPull Fix Design

## Problem

`AutoPull()` fails when go.mod/go.sum are modified locally by `go mod tidy`. This happens because different Go versions update these files differently, causing uncommitted changes that block `git pull --rebase`.

## Solution

Reset go.mod/go.sum before fetching:

```go
resetCmd := exec.Command("git", "checkout", "go.mod", "go.sum")
resetCmd.Dir = repoDir
resetCmd.Run()
```

## Rationale

- Repo is source of truth for dependencies
- `go mod tidy` runs again after pull during build
- Errors ignored (files may not be modified)
- Dependency updates from repo flow through naturally
