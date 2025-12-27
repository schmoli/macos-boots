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

	// Post-install: zsh integrations and hooks (will be added in Task 4)
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
