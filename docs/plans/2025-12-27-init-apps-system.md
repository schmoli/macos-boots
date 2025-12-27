# Init Apps System Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Auto-install base tools (fnm, node 24) on every CLI run, ensuring npm is always available before installing other tools.

**Architecture:** Add `init` field to app.yaml config, check/install init apps before every command execution (similar to existing AutoPull mechanism), remove npm deferral logic since npm will be guaranteed available.

**Tech Stack:** Go, YAML, Homebrew

---

## Task 1: Add init field to config

**Files:**
- Modify: `internal/config/config.go:15-24`

**Step 1: Add Init field to App struct**

In `internal/config/config.go`, add the `Init` field to the `App` struct:

```go
type App struct {
	Install     string     `yaml:"install"`
	Category    string     `yaml:"-"` // inferred from path
	Description string     `yaml:"description"`
	Package     string     `yaml:"package"`
	ID          int        `yaml:"id"`
	Config      *AppConfig `yaml:"config"`
	PostInstall []string   `yaml:"post_install"`
	Depends     []string   `yaml:"depends"`
	Init        bool       `yaml:"init"`  // NEW: marks app as init/base tool
}
```

**Step 2: Add InitApps method to Config**

Add this method after the `FilterByInstallType` method (after line 136):

```go
// InitApps returns all apps marked with init: true
func (c *Config) InitApps() map[string]App {
	result := make(map[string]App)
	for name, app := range c.Apps {
		if app.Init {
			result[name] = app
		}
	}
	return result
}
```

**Step 3: Verify changes compile**

Run: `go build ./cmd/macos-setup/`
Expected: Clean build with no errors

**Step 4: Commit**

```bash
git add internal/config/config.go
git commit -m "feat(config): add init field to mark base tools"
```

---

## Task 2: Mark fnm as init app

**Files:**
- Modify: `apps/cli/fnm/app.yaml`

**Step 1: Add init flag to fnm config**

Update `apps/cli/fnm/app.yaml`:

```yaml
install: brew
description: Fast Node.js version manager
init: true
post_install:
  - fnm install 24
  - fnm default 24
```

**Step 2: Commit**

```bash
git add apps/cli/fnm/app.yaml
git commit -m "feat(apps): mark fnm as init app"
```

---

## Task 3: Implement EnsureInitApps function

**Files:**
- Modify: `internal/installer/installer.go` (add after line 185)

**Step 1: Add EnsureInitApps function**

Add this function after the `Install` function (around line 185):

```go
// EnsureInitApps checks and installs any missing init apps silently
// Called at startup before every command to guarantee base tools are available
func EnsureInitApps(cfg *config.Config, verbose bool) {
	initApps := cfg.InitApps()
	if len(initApps) == 0 {
		return
	}

	// Check what's missing
	installed := InstalledBrewPackages()
	missing := make(map[string]config.App)

	for name, app := range initApps {
		pkg := name
		if app.Package != "" {
			pkg = app.Package
		}

		if !installed[pkg] {
			missing[name] = app
		}
	}

	if len(missing) == 0 {
		return // All init apps present, silent exit
	}

	// Install missing init apps
	var names []string
	for name := range missing {
		names = append(names, name)
	}
	LogProgress(fmt.Sprintf("Installing init apps: %v", names))

	result, _ := Install(missing, verbose)

	if len(result.Installed) > 0 {
		LogSuccess(fmt.Sprintf("Init apps ready: %v", result.Installed))
		fmt.Println()
	}
}
```

**Step 2: Verify changes compile**

Run: `go build ./cmd/macos-setup/`
Expected: Clean build with no errors

**Step 3: Commit**

```bash
git add internal/installer/installer.go
git commit -m "feat(installer): add EnsureInitApps function"
```

---

## Task 4: Call EnsureInitApps on every CLI run

**Files:**
- Modify: `cmd/macos-setup/main.go:36-45`

**Step 1: Add EnsureInitApps call before AutoPull**

In `cmd/macos-setup/main.go`, add the call after loading config and before AutoPull (around line 45):

```go
cfg, err := loadConfig()
if err != nil {
	fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
	os.Exit(1)
}

// Ensure init apps installed (NEW)
installer.EnsureInitApps(cfg, verbose)

// Auto-pull on any command (except help)
if installer.AutoPull() {
	fmt.Println()
}
```

**Step 2: Build and test**

Run: `go build -o bin/macos-setup ./cmd/macos-setup/`
Expected: Clean build

**Step 3: Manual test - verify init apps install**

```bash
# Remove fnm if installed
brew uninstall fnm || true

# Run macos-setup - should auto-install fnm
./bin/macos-setup status
```

Expected output should show:
```
⏳ Installing init apps: [fnm]
...
✅ Init apps ready: [fnm]
```

**Step 4: Commit**

```bash
git add cmd/macos-setup/main.go
git commit -m "feat(cli): auto-install init apps on every run"
```

---

## Task 5: Add explicit init command

**Files:**
- Modify: `cmd/macos-setup/main.go:48-67` (switch statement)
- Modify: `cmd/macos-setup/main.go:131-146` (printHelp function)

**Step 1: Add init case to switch statement**

In the switch statement (around line 48), add the init case after the empty case:

```go
var runErr error
switch cmd {
case "":
	installer.Status(cfg)
case "init":  // NEW
	runErr = runInit(cfg)
case "all":
	runErr = runInstall(cfg, "")
case "cli":
	runErr = runInstall(cfg, "cli")
// ... rest unchanged
```

**Step 2: Add runInit function**

Add this function after `runInstall` (around line 130):

```go
func runInit(cfg *config.Config) error {
	initApps := cfg.InitApps()
	if len(initApps) == 0 {
		installer.LogDim("No init apps defined")
		return nil
	}

	// Force reinstall of init apps
	result, err := installer.Install(initApps, verbose)
	if err != nil {
		return err
	}

	// Print summary
	fmt.Println()
	if len(result.Installed) > 0 {
		installer.LogSuccess(fmt.Sprintf("Installed: %v", result.Installed))
	}
	if len(result.Failed) > 0 {
		installer.LogFail(fmt.Sprintf("Failed: %v", result.Failed))
	}
	if len(result.Installed) == 0 && len(result.Failed) == 0 {
		installer.LogSuccess("All init apps installed")
	}

	return nil
}
```

**Step 3: Update help text**

In `printHelp()` function (around line 135), add the init command:

```go
func printHelp() {
	fmt.Println("macos-setup - fast macOS setup")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  macos-setup              Show status")
	fmt.Println("  macos-setup init         Install base tools")  // NEW
	fmt.Println("  macos-setup all          Install all apps")
	fmt.Println("  macos-setup cli          Install CLI tools")
	fmt.Println("  macos-setup apps         Install desktop apps")
	fmt.Println("  macos-setup mas          Install App Store apps")
	fmt.Println("  macos-setup update       Upgrade tracked apps")
	fmt.Println("  macos-setup help         Show this help")
	fmt.Println()
	fmt.Println("Flags:")
	fmt.Println("  -v, --verbose    Show command details on failure")
}
```

**Step 4: Build and test**

Run: `go build -o bin/macos-setup ./cmd/macos-setup/`

**Step 5: Test init command**

Run: `./bin/macos-setup init`
Expected: Installs or confirms init apps installed

**Step 6: Test help**

Run: `./bin/macos-setup help`
Expected: Shows init command in usage

**Step 7: Commit**

```bash
git add cmd/macos-setup/main.go
git commit -m "feat(cli): add explicit init command"
```

---

## Task 6: Remove npm deferral logic

**Files:**
- Modify: `internal/installer/installer.go:104-110` (Result struct)
- Modify: `internal/installer/installer.go:150-165` (npm install logic)
- Modify: `cmd/macos-setup/main.go:108-117` (deferred output)

**Step 1: Remove Deferred field from Result struct**

In `internal/installer/installer.go`, update the Result struct (around line 104):

```go
// Result tracks install outcomes
type Result struct {
	Installed []string
	Skipped   []string
	Failed    []string
	// Deferred field REMOVED - npm guaranteed available via init apps
}
```

**Step 2: Simplify npm install logic**

Replace the npm installation logic (around line 150-165) with:

```go
// Install npm apps sequentially (npm guaranteed available via init apps)
if len(npmApps) > 0 {
	for name, app := range npmApps {
		if err := installNpmApp(name, app, result, verbose); err != nil {
			result.Failed = append(result.Failed, name)
		}
	}
}
```

Remove the `isNpmAvailable()` function (lines 189-192) - no longer needed.

**Step 3: Remove deferred handling from main.go**

In `cmd/macos-setup/main.go`, in the `runInstall` function (around line 108-117), remove the deferred section:

```go
// Print summary
fmt.Println()
if len(result.Installed) > 0 {
	installer.LogSuccess(fmt.Sprintf("Installed: %v", result.Installed))
}
// REMOVED: Deferred section (lines 108-117)
if len(result.Failed) > 0 {
	installer.LogFail(fmt.Sprintf("Failed: %v", result.Failed))
}
if len(result.Installed) == 0 && len(result.Failed) == 0 {
	label := "tools"
	if category != "" {
		label = category + " tools"
	}
	installer.LogSuccess(fmt.Sprintf("All %s installed", label))
}
```

**Step 4: Verify changes compile**

Run: `go build ./cmd/macos-setup/`
Expected: Clean build with no errors

**Step 5: Commit**

```bash
git add internal/installer/installer.go cmd/macos-setup/main.go
git commit -m "feat(installer): remove npm deferral logic"
```

---

## Task 7: Integration testing

**Files:**
- None (testing only)

**Step 1: Build final binary**

Run: `go build -o bin/macos-setup ./cmd/macos-setup/`
Expected: Clean build

**Step 2: Test auto-install of init apps**

```bash
# Ensure fnm is not installed
brew uninstall fnm || true

# Run any command - should auto-install fnm
./bin/macos-setup status
```

Expected:
- fnm gets auto-installed
- Node 24 gets installed via post_install
- status shows normally

**Step 3: Test init command explicitly**

Run: `./bin/macos-setup init`
Expected: "All init apps installed" message

**Step 4: Test npm package installation**

Find an npm app and test install:

```bash
# Find an npm app
grep -r "install: npm" apps/

# Try installing it (if exists)
./bin/macos-setup cli
```

Expected: npm packages install without deferral

**Step 5: Verify no regression on other commands**

```bash
./bin/macos-setup help
./bin/macos-setup status
./bin/macos-setup all
```

Expected: All commands work normally

---

## Task 8: Update documentation

**Files:**
- Modify: `CLAUDE.md` (project conventions)

**Step 1: Document init apps in CLAUDE.md**

Add section about init apps to `CLAUDE.md`:

```markdown
## Init Apps

Apps marked with `init: true` in their `app.yaml` are installed automatically before every command. These are base tools required for other installations (e.g., fnm for npm packages).

Init apps are:
- Auto-installed if missing on every CLI run
- Installed before AutoPull check
- Can be explicitly reinstalled with `macos-setup init`

Current init apps:
- fnm (provides node/npm)
```

**Step 2: Commit**

```bash
git add CLAUDE.md
git commit -m "docs: document init apps system"
```

---

## Task 9: Final commit and summary

**Step 1: Verify all changes committed**

Run: `git status`
Expected: Clean working tree

**Step 2: View commit log**

Run: `git log --oneline -9`
Expected: See all commits from this implementation

**Step 3: Push changes (if ready)**

```bash
git push origin main
```

Or if working in a branch:

```bash
git push origin toli/init-apps-system
```

---

## Success Criteria

✅ fnm marked as init app in app.yaml
✅ Config struct has Init field
✅ EnsureInitApps auto-installs missing init apps
✅ Init apps installed before every command
✅ Explicit `macos-setup init` command works
✅ npm deferral logic removed
✅ All commands work without regression
✅ Documentation updated

## Notes

- Future: When shipping pre-built binary, move go installation from setup.sh to init apps
- Consider adding other base tools (mas, gh) to init apps if needed
- Init apps are self-healing - if removed, next run reinstalls them
