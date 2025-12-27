# CLI Redesign Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Replace TUI with fast CLI commands, batch brew/cask via Brewfile.

**Architecture:** Single `installer` package handles Brewfile generation and install orchestration. Main dispatches commands. Auto-pull on startup. State tracking preserved.

**Tech Stack:** Go, brew bundle, gopkg.in/yaml.v3

---

### Task 1: Add FilterByCategory to config

**Files:**
- Modify: `internal/config/config.go`

**Step 1: Add FilterByCategory method**

```go
// FilterByCategory returns apps matching the given category
func (c *Config) FilterByCategory(category string) map[string]App {
	result := make(map[string]App)
	for name, app := range c.Apps {
		if app.Category == category {
			result[name] = app
		}
	}
	return result
}

// FilterByInstallType returns apps matching brew or cask
func (c *Config) FilterByInstallType(types ...string) map[string]App {
	typeSet := make(map[string]bool)
	for _, t := range types {
		typeSet[t] = true
	}
	result := make(map[string]App)
	for name, app := range c.Apps {
		if typeSet[app.Install] {
			result[name] = app
		}
	}
	return result
}
```

**Step 2: Verify it compiles**

Run: `cd /Users/toli/code/tools/macos-setup && go build ./...`
Expected: No errors

**Step 3: Commit**

```bash
git add internal/config/config.go
git commit -m "feat(config): add filter methods for category and install type"
```

---

### Task 2: Create installer package - Brewfile generation

**Files:**
- Create: `internal/installer/installer.go`

**Step 1: Create installer package with Brewfile generation**

```go
package installer

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/schmoli/macos-setup/internal/config"
	"github.com/schmoli/macos-setup/internal/state"
)

// InstalledBrewPackages returns set of installed brew/cask packages
func InstalledBrewPackages() map[string]bool {
	installed := make(map[string]bool)

	// Get brew formulae
	cmd := exec.Command("/opt/homebrew/bin/brew", "list", "--formula", "-1")
	if out, err := cmd.Output(); err == nil {
		for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
			if line != "" {
				installed[line] = true
			}
		}
	}

	// Get casks
	cmd = exec.Command("/opt/homebrew/bin/brew", "list", "--cask", "-1")
	if out, err := cmd.Output(); err == nil {
		for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
			if line != "" {
				installed[line] = true
			}
		}
	}

	return installed
}

// GenerateBrewfile creates a temp Brewfile for the given apps
func GenerateBrewfile(apps map[string]config.App) (string, error) {
	var lines []string

	for name, app := range apps {
		pkg := name
		if app.Package != "" {
			pkg = app.Package
		}

		switch app.Install {
		case "brew":
			lines = append(lines, fmt.Sprintf("brew \"%s\"", pkg))
		case "cask":
			lines = append(lines, fmt.Sprintf("cask \"%s\"", pkg))
		}
	}

	if len(lines) == 0 {
		return "", nil
	}

	tmpFile := "/tmp/macos-setup-Brewfile"
	content := strings.Join(lines, "\n") + "\n"
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		return "", err
	}

	return tmpFile, nil
}
```

**Step 2: Verify it compiles**

Run: `cd /Users/toli/code/tools/macos-setup && go build ./...`
Expected: No errors

**Step 3: Commit**

```bash
git add internal/installer/installer.go
git commit -m "feat(installer): add Brewfile generation and installed package detection"
```

---

### Task 3: Add Install function to installer

**Files:**
- Modify: `internal/installer/installer.go`

**Step 1: Add Install function**

Append to `internal/installer/installer.go`:

```go
// Result tracks install outcomes
type Result struct {
	Installed []string
	Skipped   []string
	Failed    []string
}

// Install installs apps from the given map, using Brewfile for brew/cask
func Install(apps map[string]config.App, verbose bool) (*Result, error) {
	result := &Result{}
	installed := InstalledBrewPackages()

	// Separate by install type
	brewApps := make(map[string]config.App)
	npmApps := make(map[string]config.App)
	masApps := make(map[string]config.App)

	for name, app := range apps {
		// Skip required tier
		if app.Tier == "required" {
			continue
		}

		pkg := name
		if app.Package != "" {
			pkg = app.Package
		}

		// Check if already installed
		if installed[pkg] || isNpmInstalled(pkg) {
			result.Skipped = append(result.Skipped, name)
			continue
		}

		switch app.Install {
		case "brew", "cask":
			brewApps[name] = app
		case "npm":
			npmApps[name] = app
		case "mas":
			masApps[name] = app
		}
	}

	// Install brew/cask via Brewfile
	if len(brewApps) > 0 {
		if err := installBrewApps(brewApps, result); err != nil {
			return result, err
		}
	}

	// Install npm apps sequentially
	for name, app := range npmApps {
		if err := installNpmApp(name, app, result); err != nil {
			result.Failed = append(result.Failed, name)
		}
	}

	// Install mas apps (interactive)
	for name, app := range masApps {
		if err := installMasApp(name, app, result); err != nil {
			result.Failed = append(result.Failed, name)
		}
	}

	// Post-install: zsh integrations and hooks
	for name, app := range apps {
		if contains(result.Installed, name) {
			configureApp(name, app)
		}
	}

	return result, nil
}

func isNpmInstalled(pkg string) bool {
	cmd := exec.Command("npm", "list", "-g", pkg)
	return cmd.Run() == nil
}

func installBrewApps(apps map[string]config.App, result *Result) error {
	brewfile, err := GenerateBrewfile(apps)
	if err != nil {
		return err
	}
	if brewfile == "" {
		return nil
	}
	defer os.Remove(brewfile)

	var names []string
	for name := range apps {
		names = append(names, name)
	}
	fmt.Printf("Installing %d packages...\n", len(names))

	cmd := exec.Command("/opt/homebrew/bin/brew", "bundle", "--file="+brewfile)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		// Some may have failed, but continue
		fmt.Printf("Warning: brew bundle had errors\n")
	}

	// Check what actually got installed
	nowInstalled := InstalledBrewPackages()
	for name, app := range apps {
		pkg := name
		if app.Package != "" {
			pkg = app.Package
		}
		if nowInstalled[pkg] {
			result.Installed = append(result.Installed, name)
			trackInstalled(name)
		} else {
			result.Failed = append(result.Failed, name)
		}
	}

	return nil
}

func installNpmApp(name string, app config.App, result *Result) error {
	pkg := name
	if app.Package != "" {
		pkg = app.Package
	}

	fmt.Printf("Installing %s...\n", name)
	cmd := exec.Command("npm", "install", "-g", pkg)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}

	result.Installed = append(result.Installed, name)
	trackInstalled(name)
	return nil
}

func installMasApp(name string, app config.App, result *Result) error {
	fmt.Printf("Installing %s from App Store...\n", name)
	cmd := exec.Command("mas", "install", fmt.Sprintf("%d", app.ID))
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}

	result.Installed = append(result.Installed, name)
	trackInstalled(name)
	return nil
}

func trackInstalled(name string) {
	if s, err := state.Load(); err == nil {
		s.MarkInstalled(name)
	}
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
```

**Step 2: Verify it compiles**

Run: `cd /Users/toli/code/tools/macos-setup && go build ./...`
Expected: No errors

**Step 3: Commit**

```bash
git add internal/installer/installer.go
git commit -m "feat(installer): add Install function with Brewfile batching"
```

---

### Task 4: Add zsh integration to installer

**Files:**
- Modify: `internal/installer/installer.go`

**Step 1: Add configureApp function**

Append to `internal/installer/installer.go`:

```go
func configureApp(name string, app config.App) {
	home, _ := os.UserHomeDir()

	// Add zsh integration if defined
	if app.Zsh != "" {
		fmt.Printf("Configuring %s...", name)
		if err := addZshIntegration(name, app.Zsh); err != nil {
			fmt.Printf(" failed: %v\n", err)
		} else {
			fmt.Printf(" done\n")
		}
	}

	// Run post_install commands
	if len(app.PostInstall) > 0 {
		preamble := ""
		if app.Zsh != "" {
			zshFile := filepath.Join(home, ".config", "macos-setup", "apps", name, "zshrc.zsh")
			preamble = fmt.Sprintf("source %s && ", zshFile)
		}

		for _, cmdStr := range app.PostInstall {
			fmt.Printf("  → %s\n", cmdStr)
			fullCmd := preamble + cmdStr
			cmd := exec.Command("zsh", "-c", fullCmd)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			cmd.Run()
		}
	}
}

func addZshIntegration(name, zshContent string) error {
	home, _ := os.UserHomeDir()
	baseDir := filepath.Join(home, ".config", "macos-setup")
	appDir := filepath.Join(baseDir, "apps", name)

	if err := os.MkdirAll(appDir, 0755); err != nil {
		return err
	}

	// Write init.zsh (sources all app zsh files)
	initPath := filepath.Join(baseDir, "init.zsh")
	initContent := `# macos-setup shell integration (auto-generated)
for f in $HOME/.config/macos-setup/apps/*/*.zsh(N); do
  source "$f"
done
`
	if err := os.WriteFile(initPath, []byte(initContent), 0644); err != nil {
		return err
	}

	// Ensure .zshrc sources init.zsh
	zshrcPath := filepath.Join(home, ".zshrc")
	existing, _ := os.ReadFile(zshrcPath)
	sourceLine := "source ~/.config/macos-setup/init.zsh"
	if !strings.Contains(string(existing), sourceLine) {
		f, err := os.OpenFile(zshrcPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return err
		}
		f.WriteString(fmt.Sprintf("\n# macos-setup\n%s\n", sourceLine))
		f.Close()

		// Mark zshrc modified
		markerPath := filepath.Join(baseDir, ".zshrc-modified")
		os.WriteFile(markerPath, []byte{}, 0644)
	}

	// Write app zshrc
	appZshPath := filepath.Join(appDir, "zshrc.zsh")
	return os.WriteFile(appZshPath, []byte(strings.TrimSpace(zshContent)+"\n"), 0644)
}

// CheckZshrcModified returns true if zshrc was modified, clears the marker
func CheckZshrcModified() bool {
	home, _ := os.UserHomeDir()
	markerPath := filepath.Join(home, ".config", "macos-setup", ".zshrc-modified")
	if _, err := os.Stat(markerPath); err == nil {
		os.Remove(markerPath)
		return true
	}
	return false
}
```

**Step 2: Verify it compiles**

Run: `cd /Users/toli/code/tools/macos-setup && go build ./...`
Expected: No errors

**Step 3: Commit**

```bash
git add internal/installer/installer.go
git commit -m "feat(installer): add zsh integration and post_install hooks"
```

---

### Task 5: Add auto-pull and upgrade functions

**Files:**
- Modify: `internal/installer/installer.go`

**Step 1: Add AutoPull and Upgrade functions**

Append to `internal/installer/installer.go`:

```go
// AutoPull fetches and pulls from origin if behind, returns true if pulled
func AutoPull() bool {
	home, _ := os.UserHomeDir()
	repoDir := filepath.Join(home, ".config", "macos-setup", "repo")

	// Fetch
	cmd := exec.Command("git", "fetch", "origin")
	cmd.Dir = repoDir
	cmd.Run()

	// Compare
	localCmd := exec.Command("git", "rev-parse", "HEAD")
	localCmd.Dir = repoDir
	localHash, _ := localCmd.Output()

	remoteCmd := exec.Command("git", "rev-parse", "origin/main")
	remoteCmd.Dir = repoDir
	remoteHash, _ := remoteCmd.Output()

	if string(localHash) == string(remoteHash) {
		return false
	}

	// Pull
	fmt.Println("Pulling updates...")
	cmd = exec.Command("git", "pull", "--rebase")
	cmd.Dir = repoDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Run()

	return true
}

// Upgrade upgrades all tracked apps
func Upgrade(cfg *config.Config) error {
	s, err := state.Load()
	if err != nil {
		return err
	}

	if len(s.Installed) == 0 {
		fmt.Println("No tracked apps to upgrade.")
		return nil
	}

	// Collect tracked brew/cask apps
	var brewPkgs []string
	var npmPkgs []string

	for name := range s.Installed {
		app, ok := cfg.Apps[name]
		if !ok {
			continue
		}

		pkg := name
		if app.Package != "" {
			pkg = app.Package
		}

		switch app.Install {
		case "brew", "cask":
			brewPkgs = append(brewPkgs, pkg)
		case "npm":
			npmPkgs = append(npmPkgs, pkg)
		}
	}

	// Upgrade brew packages
	if len(brewPkgs) > 0 {
		fmt.Printf("Upgrading %d brew packages...\n", len(brewPkgs))
		args := append([]string{"upgrade"}, brewPkgs...)
		cmd := exec.Command("/opt/homebrew/bin/brew", args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Run()
	}

	// Upgrade npm packages
	if len(npmPkgs) > 0 {
		fmt.Printf("Upgrading %d npm packages...\n", len(npmPkgs))
		for _, pkg := range npmPkgs {
			cmd := exec.Command("npm", "update", "-g", pkg)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			cmd.Run()
		}
	}

	fmt.Println("✓ Upgrade complete")
	return nil
}
```

**Step 2: Verify it compiles**

Run: `cd /Users/toli/code/tools/macos-setup && go build ./...`
Expected: No errors

**Step 3: Commit**

```bash
git add internal/installer/installer.go
git commit -m "feat(installer): add AutoPull and Upgrade functions"
```

---

### Task 6: Add Status function

**Files:**
- Modify: `internal/installer/installer.go`

**Step 1: Add Status function**

Append to `internal/installer/installer.go`:

```go
// Status prints installed vs available apps
func Status(cfg *config.Config) {
	installed := InstalledBrewPackages()

	// Also check npm
	npmInstalled := make(map[string]bool)
	for name, app := range cfg.Apps {
		if app.Install == "npm" {
			pkg := name
			if app.Package != "" {
				pkg = app.Package
			}
			if isNpmInstalled(pkg) {
				npmInstalled[name] = true
			}
		}
	}

	byCategory := make(map[string][]string)
	byStatus := make(map[string]map[string][]string) // category -> status -> names

	for name, app := range cfg.Apps {
		if app.Tier == "required" {
			continue
		}

		pkg := name
		if app.Package != "" {
			pkg = app.Package
		}

		isInstalled := installed[pkg] || npmInstalled[name]
		status := "available"
		if isInstalled {
			status = "installed"
		}

		if byStatus[app.Category] == nil {
			byStatus[app.Category] = make(map[string][]string)
		}
		byStatus[app.Category][status] = append(byStatus[app.Category][status], name)
		byCategory[app.Category] = append(byCategory[app.Category], name)
	}

	categories := []string{"cli", "apps", "mas"}
	for _, cat := range categories {
		if len(byCategory[cat]) == 0 {
			continue
		}

		fmt.Printf("\n%s:\n", cat)
		if len(byStatus[cat]["installed"]) > 0 {
			fmt.Printf("  ✓ %s\n", strings.Join(byStatus[cat]["installed"], ", "))
		}
		if len(byStatus[cat]["available"]) > 0 {
			fmt.Printf("  ○ %s\n", strings.Join(byStatus[cat]["available"], ", "))
		}
	}
	fmt.Println()
}
```

**Step 2: Verify it compiles**

Run: `cd /Users/toli/code/tools/macos-setup && go build ./...`
Expected: No errors

**Step 3: Commit**

```bash
git add internal/installer/installer.go
git commit -m "feat(installer): add Status function"
```

---

### Task 7: Rewrite main.go with new CLI commands

**Files:**
- Modify: `cmd/macos-setup/main.go`

**Step 1: Rewrite main.go**

```go
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/schmoli/macos-setup/internal/config"
	"github.com/schmoli/macos-setup/internal/installer"
)

func main() {
	// Auto-pull on any command
	if installer.AutoPull() {
		fmt.Println()
	}

	cfg, err := loadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	cmd := ""
	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}

	var runErr error
	switch cmd {
	case "", "all":
		runErr = runInstall(cfg, "")
	case "cli":
		runErr = runInstall(cfg, "cli")
	case "apps":
		runErr = runInstall(cfg, "apps")
	case "mas":
		runErr = runInstall(cfg, "mas")
	case "update":
		runErr = installer.Upgrade(cfg)
	case "status":
		installer.Status(cfg)
	case "help", "--help", "-h":
		printHelp()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", cmd)
		printHelp()
		os.Exit(1)
	}

	if runErr != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", runErr)
		os.Exit(1)
	}

	// Check if zshrc was modified
	if installer.CheckZshrcModified() {
		fmt.Println()
		fmt.Println("⚠  Run: source ~/.zshrc")
	}
}

func loadConfig() (*config.Config, error) {
	home, _ := os.UserHomeDir()
	configPath := filepath.Join(home, ".config", "macos-setup", "repo", "apps.yaml")
	return config.Load(configPath)
}

func runInstall(cfg *config.Config, category string) error {
	apps := cfg.Apps
	if category != "" {
		apps = cfg.FilterByCategory(category)
	}

	if len(apps) == 0 {
		fmt.Printf("No apps found for category: %s\n", category)
		return nil
	}

	result, err := installer.Install(apps, true)
	if err != nil {
		return err
	}

	// Print summary
	fmt.Println()
	if len(result.Installed) > 0 {
		fmt.Printf("✓ Installed: %v\n", result.Installed)
	}
	if len(result.Skipped) > 0 {
		fmt.Printf("Skipped (already installed): %v\n", result.Skipped)
	}
	if len(result.Failed) > 0 {
		fmt.Printf("✗ Failed: %v\n", result.Failed)
	}

	return nil
}

func printHelp() {
	fmt.Println("macos-setup - fast macOS setup")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  macos-setup          Install all apps")
	fmt.Println("  macos-setup cli      Install CLI tools")
	fmt.Println("  macos-setup apps     Install desktop apps")
	fmt.Println("  macos-setup mas      Install App Store apps")
	fmt.Println("  macos-setup update   Upgrade tracked apps")
	fmt.Println("  macos-setup status   Show installed vs available")
	fmt.Println("  macos-setup help     Show this help")
}
```

**Step 2: Verify it compiles**

Run: `cd /Users/toli/code/tools/macos-setup && go build ./...`
Expected: No errors

**Step 3: Commit**

```bash
git add cmd/macos-setup/main.go
git commit -m "feat(cli): rewrite main with new CLI commands"
```

---

### Task 8: Remove TUI code and dependencies

**Files:**
- Remove: `internal/tui/tui.go`
- Modify: `go.mod`

**Step 1: Remove TUI file**

Run: `rm /Users/toli/code/tools/macos-setup/internal/tui/tui.go && rmdir /Users/toli/code/tools/macos-setup/internal/tui`

**Step 2: Clean up go.mod**

Run: `cd /Users/toli/code/tools/macos-setup && go mod tidy`

**Step 3: Verify it builds**

Run: `cd /Users/toli/code/tools/macos-setup && go build ./...`
Expected: No errors, TUI deps removed from go.mod

**Step 4: Commit**

```bash
git add -A
git commit -m "chore: remove TUI code and dependencies"
```

---

### Task 9: Build and test

**Step 1: Build binary**

Run: `cd /Users/toli/code/tools/macos-setup && go build -o bin/macos-setup ./cmd/macos-setup/`
Expected: Binary created

**Step 2: Test help**

Run: `./bin/macos-setup help`
Expected: Shows new help output

**Step 3: Test status**

Run: `./bin/macos-setup status`
Expected: Shows installed vs available by category

**Step 4: Test dry run (already installed)**

Run: `./bin/macos-setup cli`
Expected: Shows "Skipped (already installed)" for most, only installs new

**Step 5: Commit any fixes**

If needed, commit fixes found during testing.

---

### Task 10: Update add-app skill docs

**Files:**
- Modify: `docs/skills/add-app/README.md`

**Step 1: Update Tiers section**

Change lines 48-54 from:

```markdown
## Tiers

| Tier | Behavior |
|------|----------|
| required | Auto-installs before TUI launches |
| auto | Normal install, no special handling (default) |
| interactive | Needs user input (e.g., App Store sign-in) |

Omit tier field for auto behavior.
```

To:

```markdown
## Tiers

| Tier | Behavior |
|------|----------|
| required | Auto-installs on first run (fnm, mas) |
| auto | Normal install (default) |

Omit tier field for auto behavior.
```

**Step 2: Commit**

```bash
git add docs/skills/add-app/README.md
git commit -m "docs(add-app): update tiers for CLI-first design"
```

---

## Summary

Tasks:
1. Add FilterByCategory to config
2. Create installer package - Brewfile generation
3. Add Install function with batching
4. Add zsh integration to installer
5. Add AutoPull and Upgrade functions
6. Add Status function
7. Rewrite main.go with new CLI
8. Remove TUI code and dependencies
9. Build and test
10. Update add-app skill docs
