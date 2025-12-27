# App Folder Structure Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Migrate from single `apps.yaml` to per-app folders with `app.yaml` and optional `init.zsh` files.

**Architecture:** Each app gets its own folder under `apps/<category>/<name>/`. Config loader scans these folders. Shell integration sources `init.zsh` files directly from repo.

**Tech Stack:** Go, YAML, zsh

---

### Task 1: Create folder structure and migrate apps

**Files:**
- Create: `apps/cli/<name>/app.yaml` for each CLI app
- Create: `apps/apps/<name>/app.yaml` for each GUI app
- Create: `apps/<category>/<name>/init.zsh` for apps with zsh field

**Step 1: Create directory structure**

```bash
mkdir -p apps/cli apps/apps
```

**Step 2: Create app folders and files**

Create these files (showing a few examples, do all apps):

`apps/cli/jq/app.yaml`:
```yaml
install: brew
description: JSON processor
```

`apps/cli/zoxide/app.yaml`:
```yaml
install: brew
description: Smarter cd command
```

`apps/cli/zoxide/init.zsh`:
```zsh
eval "$(zoxide init zsh)"
```

`apps/cli/yazi/app.yaml`:
```yaml
install: brew
description: Terminal file manager
```

`apps/cli/yazi/init.zsh`:
```zsh
alias ya=yazi
```

`apps/cli/fnm/app.yaml`:
```yaml
install: brew
tier: required
description: Fast Node.js version manager
post_install:
  - fnm install 24
  - fnm default 24
```

`apps/cli/fnm/init.zsh`:
```zsh
# fnm - fast node manager
eval "$(fnm env --use-on-cd)"
```

`apps/cli/kind/app.yaml`:
```yaml
install: brew
description: Run local Kubernetes cluster in Docker
depends:
  - docker
```

`apps/apps/rectangle/app.yaml`:
```yaml
install: cask
description: Move and resize windows using keyboard shortcuts or snap areas
```

**Step 3: Verify structure**

```bash
find apps -name "*.yaml" -o -name "*.zsh" | head -20
```

**Step 4: Commit**

```bash
git add apps/
git commit -m "feat: create app folder structure"
```

---

### Task 2: Update config loader to scan folders

**Files:**
- Modify: `internal/config/config.go`

**Step 1: Update Load function**

Replace the entire `config.go` with:

```go
package config

import (
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Apps map[string]App
}

type App struct {
	Install     string     `yaml:"install"`
	Category    string     `yaml:"-"` // inferred from path
	Tier        string     `yaml:"tier"`
	Description string     `yaml:"description"`
	Package     string     `yaml:"package"`
	ID          int        `yaml:"id"`
	Config      *AppConfig `yaml:"config"`
	PostInstall []string   `yaml:"post_install"`
	Depends     []string   `yaml:"depends"`
}

type AppConfig struct {
	Source string `yaml:"source"`
	Dest   string `yaml:"dest"`
}

// Load scans apps/<category>/<name>/app.yaml files
func Load(appsDir string) (*Config, error) {
	cfg := &Config{Apps: make(map[string]App)}

	// Scan category directories
	categories, err := os.ReadDir(appsDir)
	if err != nil {
		return nil, err
	}

	for _, cat := range categories {
		if !cat.IsDir() {
			continue
		}
		catName := cat.Name()
		catPath := filepath.Join(appsDir, catName)

		// Scan app directories within category
		apps, err := os.ReadDir(catPath)
		if err != nil {
			continue
		}

		for _, appDir := range apps {
			if !appDir.IsDir() {
				continue
			}
			appName := appDir.Name()
			appYaml := filepath.Join(catPath, appName, "app.yaml")

			data, err := os.ReadFile(appYaml)
			if err != nil {
				continue // skip if no app.yaml
			}

			var app App
			if err := yaml.Unmarshal(data, &app); err != nil {
				continue
			}

			app.Category = catName
			cfg.Apps[appName] = app
		}
	}

	return cfg, nil
}

// LoadLegacy loads from single apps.yaml (for migration)
func LoadLegacy(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var wrapper struct {
		Apps map[string]App `yaml:"apps"`
	}
	if err := yaml.Unmarshal(data, &wrapper); err != nil {
		return nil, err
	}

	return &Config{Apps: wrapper.Apps}, nil
}

// HasInitZsh checks if app has init.zsh in repo
func HasInitZsh(appsDir, category, name string) bool {
	initPath := filepath.Join(appsDir, category, name, "init.zsh")
	_, err := os.Stat(initPath)
	return err == nil
}

// AppsByCategory returns apps grouped by category
func (c *Config) AppsByCategory() map[string][]string {
	result := make(map[string][]string)
	for name, app := range c.Apps {
		result[app.Category] = append(result[app.Category], name)
	}
	return result
}

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

// FilterByInstallType returns apps matching install types
func (c *Config) FilterByInstallType(types ...string) map[string]App {
	typeSet := make(map[string]bool)
	for _, t := range types {
		typeSet[strings.ToLower(t)] = true
	}
	result := make(map[string]App)
	for name, app := range c.Apps {
		if typeSet[strings.ToLower(app.Install)] {
			result[name] = app
		}
	}
	return result
}
```

**Step 2: Build to verify**

```bash
go build ./...
```

**Step 3: Commit**

```bash
git add internal/config/config.go
git commit -m "feat(config): scan app folders instead of single yaml"
```

---

### Task 3: Update main.go to use new config path

**Files:**
- Modify: `cmd/macos-setup/main.go`

**Step 1: Update loadConfig function**

Change `loadConfig()` from:

```go
func loadConfig() (*config.Config, error) {
	home, _ := os.UserHomeDir()
	configPath := filepath.Join(home, ".config", "macos-setup", "repo", "apps.yaml")
	return config.Load(configPath)
}
```

To:

```go
func loadConfig() (*config.Config, error) {
	home, _ := os.UserHomeDir()
	appsDir := filepath.Join(home, ".config", "macos-setup", "repo", "apps")
	return config.Load(appsDir)
}
```

**Step 2: Build and test**

```bash
go build ./cmd/macos-setup/
./macos-setup status
```

**Step 3: Commit**

```bash
git add cmd/macos-setup/main.go
git commit -m "feat(main): load config from apps/ folder"
```

---

### Task 4: Update init.zsh to source from repo

**Files:**
- Modify: `internal/installer/installer.go`

**Step 1: Update addZshIntegration function**

Replace `addZshIntegration` with a simpler function that just ensures init.zsh exists and sources from repo:

Find this code block (around line 306-345):
```go
func addZshIntegration(name, zshContent string) error {
	...
}
```

Replace with:

```go
// EnsureShellIntegration ensures ~/.zshrc sources the repo init files
func EnsureShellIntegration() error {
	home, _ := os.UserHomeDir()
	baseDir := filepath.Join(home, ".config", "macos-setup")

	// Write init.zsh that sources from repo
	initPath := filepath.Join(baseDir, "init.zsh")
	initContent := `# macos-setup shell integration (auto-generated)
for f in ~/.config/macos-setup/repo/apps/*/*/init.zsh(N); do
  source "$f"
done
`
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return err
	}
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

	return nil
}
```

**Step 2: Update configureApp to remove zsh content handling**

Find this in `configureApp` (around line 275-304):
```go
func configureApp(name string, app config.App) {
	home, _ := os.UserHomeDir()

	// Add zsh integration if defined
	if app.Zsh != "" {
		if err := addZshIntegration(name, app.Zsh); err != nil {
			LogFail(fmt.Sprintf("Configuring %s: %v", name, err))
		} else {
			LogSuccess(fmt.Sprintf("Configured %s", name))
		}
	}
	...
```

Replace entire function with:

```go
func configureApp(name string, app config.App) {
	home, _ := os.UserHomeDir()

	// Run post_install commands
	if len(app.PostInstall) > 0 {
		// Source app's init.zsh if it exists
		preamble := ""
		initZsh := filepath.Join(home, ".config", "macos-setup", "repo", "apps", app.Category, name, "init.zsh")
		if _, err := os.Stat(initZsh); err == nil {
			preamble = fmt.Sprintf("source %s && ", initZsh)
		}

		for _, cmdStr := range app.PostInstall {
			LogDim(cmdStr)
			fullCmd := preamble + cmdStr
			cmd := exec.Command("zsh", "-c", fullCmd)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			cmd.Run()
		}
	}
}
```

**Step 3: Call EnsureShellIntegration on first install**

In `Install` function, after the installs complete (around line 169), add:

```go
	// Ensure shell integration is set up
	if len(result.Installed) > 0 {
		EnsureShellIntegration()
	}
```

**Step 4: Build and verify**

```bash
go build ./cmd/macos-setup/
```

**Step 5: Commit**

```bash
git add internal/installer/installer.go
git commit -m "feat(installer): source init.zsh from repo directly"
```

---

### Task 5: Delete old apps.yaml and cleanup

**Files:**
- Delete: `apps.yaml`
- Modify: `internal/config/config.go` (remove Zsh field)

**Step 1: Remove Zsh field from App struct**

The `Zsh` field is already removed in Task 2. Verify it's gone.

**Step 2: Delete apps.yaml**

```bash
rm apps.yaml
```

**Step 3: Build and test full flow**

```bash
go build ./cmd/macos-setup/
./macos-setup status
./macos-setup cli
```

**Step 4: Commit**

```bash
git add -A
git commit -m "chore: remove legacy apps.yaml"
```

---

### Task 6: Update /add-app skill

**Files:**
- Modify: `docs/skills/add-app/README.md`
- Modify: `.claude/commands/add-app.md`

**Step 1: Update skill to create folder structure**

Update the skill docs to:
1. Create `apps/<category>/<name>/app.yaml` instead of editing `apps.yaml`
2. If app has zsh content, create `apps/<category>/<name>/init.zsh`
3. Update commit paths

**Step 2: Test skill manually**

```bash
# Test with a new app
claude "/add-app tree to cli"
```

**Step 3: Commit**

```bash
git add docs/skills/add-app/ .claude/commands/
git commit -m "feat(add-app): update skill for folder structure"
git push
```

---

## Summary

| Task | Description |
|------|-------------|
| 1 | Create folder structure, migrate all apps |
| 2 | Update config loader to scan folders |
| 3 | Update main.go config path |
| 4 | Update installer for new init.zsh sourcing |
| 5 | Delete old apps.yaml |
| 6 | Update /add-app skill |
