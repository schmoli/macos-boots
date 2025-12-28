package installer

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"syscall"

	"github.com/charmbracelet/lipgloss"
	"github.com/schmoli/macos-setup/internal/config"
	"github.com/schmoli/macos-setup/internal/state"
)

// Styled output helpers
var (
	progressStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("39"))
	successStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("82"))
	failStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	warnStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	dimStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
)

func LogProgress(msg string) {
	fmt.Println(progressStyle.Render("⏳ " + msg))
}

func LogSuccess(msg string) {
	fmt.Println(successStyle.Render("✅ " + msg))
}

func LogFail(msg string) {
	fmt.Println(failStyle.Render("❌ " + msg))
}

func LogWarn(msg string) {
	fmt.Println(warnStyle.Render("⚡ " + msg))
}

func LogDim(msg string) {
	fmt.Println(dimStyle.Render("   " + msg))
}

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

	tmpFile := "/tmp/boots-Brewfile"
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
		if err := installBrewApps(brewApps, result, verbose); err != nil {
			return result, err
		}
	}

	// Install npm apps sequentially
	if len(npmApps) > 0 {
		for name, app := range npmApps {
			if err := installNpmApp(name, app, result, verbose); err != nil {
				result.Failed = append(result.Failed, name)
			}
		}
	}

	// Install mas apps (interactive)
	for name, app := range masApps {
		if err := installMasApp(name, app, result, verbose); err != nil {
			result.Failed = append(result.Failed, name)
		}
	}

	// Post-install: run post_install hooks
	for name, app := range apps {
		if contains(result.Installed, name) {
			configureApp(name, app)
		}
	}

	// Ensure shell integration is set up
	if len(result.Installed) > 0 {
		EnsureShellIntegration()
	}

	return result, nil
}

func isNpmInstalled(pkg string) bool {
	cmd := exec.Command("npm", "list", "-g", pkg)
	return cmd.Run() == nil
}

func installBrewApps(apps map[string]config.App, result *Result, verbose bool) error {
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
	LogProgress(fmt.Sprintf("Installing %d packages...", len(names)))

	cmd := exec.Command("/opt/homebrew/bin/brew", "bundle", "--file="+brewfile)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		// Some may have failed, but continue
		LogWarn("brew bundle had errors")
		if verbose {
			LogFail(fmt.Sprintf("Command: brew bundle --file=%s", brewfile))
			if exitErr, ok := err.(*exec.ExitError); ok {
				LogFail(fmt.Sprintf("Exit code: %d", exitErr.ExitCode()))
			}
		}
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

func installNpmApp(name string, app config.App, result *Result, verbose bool) error {
	pkg := name
	if app.Package != "" {
		pkg = app.Package
	}

	LogProgress(fmt.Sprintf("Installing %s...", name))
	cmd := exec.Command("npm", "install", "-g", pkg)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		if verbose {
			LogFail(fmt.Sprintf("Command: npm install -g %s", pkg))
			if exitErr, ok := err.(*exec.ExitError); ok {
				LogFail(fmt.Sprintf("Exit code: %d", exitErr.ExitCode()))
			}
		}
		return err
	}

	result.Installed = append(result.Installed, name)
	trackInstalled(name)
	return nil
}

func installMasApp(name string, app config.App, result *Result, verbose bool) error {
	LogProgress(fmt.Sprintf("Installing %s from App Store...", name))
	cmd := exec.Command("mas", "install", fmt.Sprintf("%d", app.ID))
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		if verbose {
			LogFail(fmt.Sprintf("Command: mas install %d", app.ID))
			if exitErr, ok := err.(*exec.ExitError); ok {
				LogFail(fmt.Sprintf("Exit code: %d", exitErr.ExitCode()))
			}
		}
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

	// Run post_install commands
	if len(app.PostInstall) > 0 {
		// Build preamble: brew shellenv + app's init.zsh if exists
		preamble := `eval "$(/opt/homebrew/bin/brew shellenv)" && `
		initZsh := filepath.Join(home, ".config", "boots", "repo", "apps", app.Category, name, "init.zsh")
		if _, err := os.Stat(initZsh); err == nil {
			preamble += fmt.Sprintf("source %s && ", initZsh)
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

// EnsureShellIntegration ensures ~/.zshrc sources the repo init files
func EnsureShellIntegration() error {
	home, _ := os.UserHomeDir()
	baseDir := filepath.Join(home, ".config", "boots")

	// Write init.zsh that sources from repo
	initPath := filepath.Join(baseDir, "init.zsh")
	initContent := `# boots shell integration (auto-generated)
# Ensure compinit is loaded for completions
autoload -Uz compinit && compinit -C

for f in ~/.config/boots/repo/apps/*/*/init.zsh(N); do
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
	sourceLine := "source ~/.config/boots/init.zsh"
	if !strings.Contains(string(existing), sourceLine) {
		f, err := os.OpenFile(zshrcPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return err
		}
		f.WriteString(fmt.Sprintf("\n# boots\n%s\n", sourceLine))
		f.Close()

		// Mark zshrc modified
		markerPath := filepath.Join(baseDir, ".zshrc-modified")
		os.WriteFile(markerPath, []byte{}, 0644)
	}

	return nil
}

// CheckZshrcModified returns true if zshrc was modified, clears the marker
func CheckZshrcModified() bool {
	home, _ := os.UserHomeDir()
	markerPath := filepath.Join(home, ".config", "boots", ".zshrc-modified")
	if _, err := os.Stat(markerPath); err == nil {
		os.Remove(markerPath)
		return true
	}
	return false
}

// AutoPull fetches and pulls from origin if behind, returns true if pulled
func AutoPull() bool {
	home, _ := os.UserHomeDir()
	repoDir := filepath.Join(home, ".config", "boots", "repo")

	// Reset go files to avoid pull conflicts from go mod tidy
	resetCmd := exec.Command("git", "checkout", "go.mod", "go.sum")
	resetCmd.Dir = repoDir
	resetCmd.Run()

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

	// Get changelog before pull
	logCmd := exec.Command("git", "log", "HEAD..origin/main", "--oneline", "--format=%s")
	logCmd.Dir = repoDir
	logOutput, _ := logCmd.Output()

	// Save old HEAD for diff check
	oldHead := strings.TrimSpace(string(localHash))

	// Pull silently
	LogProgress("Pulling updates...")
	cmd = exec.Command("git", "pull", "--rebase", "-q")
	cmd.Dir = repoDir
	cmd.Run()

	// Display changelog
	commits := strings.Split(strings.TrimSpace(string(logOutput)), "\n")
	LogSuccess(fmt.Sprintf("Updated (%d commits)", len(commits)))
	for _, msg := range commits {
		if msg != "" {
			fmt.Println(warnStyle.Render("   • " + msg))
		}
	}

	// Check if Go files changed
	diffCmd := exec.Command("git", "diff", "--name-only", oldHead, "HEAD")
	diffCmd.Dir = repoDir
	diffOutput, _ := diffCmd.Output()

	needsRebuild := false
	for _, file := range strings.Split(string(diffOutput), "\n") {
		if strings.HasSuffix(file, ".go") || file == "go.mod" || file == "go.sum" {
			needsRebuild = true
			break
		}
	}

	if !needsRebuild {
		return true
	}

	// Rebuild binary
	LogProgress("Rebuilding...")
	binary := filepath.Join(repoDir, "bin", "boots")
	buildCmd := exec.Command("go", "build", "-o", binary, "./cmd/macos-setup/")
	buildCmd.Dir = repoDir
	if err := buildCmd.Run(); err != nil {
		LogFail("Rebuild failed: " + err.Error())
		return true
	}
	LogSuccess("Rebuilt")

	// Re-exec with new binary
	syscall.Exec(binary, os.Args, os.Environ())
	return true
}

// Upgrade upgrades all tracked apps
func Upgrade(cfg *config.Config) error {
	s, err := state.Load()
	if err != nil {
		return err
	}

	if len(s.Installed) == 0 {
		LogDim("No tracked apps to upgrade")
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
		LogProgress(fmt.Sprintf("Upgrading %d brew packages...", len(brewPkgs)))
		args := append([]string{"upgrade"}, brewPkgs...)
		cmd := exec.Command("/opt/homebrew/bin/brew", args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Run()
	}

	// Upgrade npm packages
	if len(npmPkgs) > 0 {
		LogProgress(fmt.Sprintf("Upgrading %d npm packages...", len(npmPkgs)))
		for _, pkg := range npmPkgs {
			cmd := exec.Command("npm", "update", "-g", pkg)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			cmd.Run()
		}
	}

	LogSuccess("Upgrade complete")
	return nil
}

// Status prints installed apps in a styled table
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

	// Collect installed apps by category
	type appInfo struct {
		name string
		desc string
	}
	byCategory := make(map[string][]appInfo)

	for name, app := range cfg.Apps {
		pkg := name
		if app.Package != "" {
			pkg = app.Package
		}

		isInst := installed[pkg] || npmInstalled[name]
		if isInst {
			byCategory[app.Category] = append(byCategory[app.Category], appInfo{name, app.Description})
		}
	}

	// Sort apps within each category
	for cat := range byCategory {
		sort.Slice(byCategory[cat], func(i, j int) bool {
			return byCategory[cat][i].name < byCategory[cat][j].name
		})
	}

	// Styles
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#00D9FF")).
		PaddingLeft(1)

	nameStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#0088FF")).
		Width(14).
		PaddingLeft(2)

	descStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("245"))

	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#0066FF")).
		Padding(0, 1)

	categories := []string{"cli", "apps", "mas"}
	var sections []string

	for _, cat := range categories {
		apps := byCategory[cat]
		if len(apps) == 0 {
			continue
		}

		// Header
		header := headerStyle.Render(fmt.Sprintf("%s (%d)", cat, len(apps)))

		// Rows
		var rows []string
		for _, app := range apps {
			row := nameStyle.Render(app.name) + descStyle.Render(app.desc)
			rows = append(rows, row)
		}

		content := header + "\n" + strings.Join(rows, "\n")
		sections = append(sections, borderStyle.Render(content))
	}

	if len(sections) == 0 {
		LogDim("No apps installed")
		return
	}

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00D9FF"))
	fmt.Println()
	fmt.Println(titleStyle.Render("Status"))
	fmt.Println(strings.Join(sections, "\n\n"))
	fmt.Println()
}
