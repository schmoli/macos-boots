package tui

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/schmoli/macos-setup/internal/config"
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("205"))

	categoryStyle = lipgloss.NewStyle().
			PaddingLeft(2)

	selectedStyle = lipgloss.NewStyle().
			PaddingLeft(2).
			Foreground(lipgloss.Color("170")).
			Bold(true)

	statusStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	appStyle = lipgloss.NewStyle().
			PaddingLeft(4)

	appSelectedStyle = lipgloss.NewStyle().
				PaddingLeft(4).
				Foreground(lipgloss.Color("170")).
				Bold(true)

	progressStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("39"))

	borderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")).
			Padding(1, 2)
)

type category struct {
	name string
	key  string
	apps []appItem
}

type appItem struct {
	name        string
	description string
	installed   bool
	selected    bool
}

type installState int

const (
	stateIdle installState = iota
	stateInstalling
	stateRemoving
)

type layoutMode int

const (
	layoutVertical   layoutMode = iota // top-bottom (default)
	layoutHorizontal                   // side-by-side
)

type model struct {
	config       *config.Config
	categories   []category
	cursor       int
	inCategory   bool
	appCursor    int
	quitting     bool
	state        installState
	progressMsg  string
	progressIdx  int
	progressApps []string
	width        int
	height       int
	layout       layoutMode
	logLines     []string
	logScroll    int
	requiredDeps map[string][]string // dep name -> apps requiring it
}

func initialModel() model {
	home, _ := os.UserHomeDir()
	configPath := filepath.Join(home, ".config", "macos-setup", "repo", "apps.yaml")

	cfg, err := config.Load(configPath)
	if err != nil {
		return model{
			categories:   []category{{name: "CLI Tools", key: "cli"}},
			requiredDeps: make(map[string][]string),
		}
	}

	cats := buildCategories(cfg)

	return model{
		config:       cfg,
		categories:   cats,
		cursor:       0,
		requiredDeps: make(map[string][]string),
	}
}

func buildCategories(cfg *config.Config) []category {
	catMap := make(map[string][]appItem)
	// Ordered list of categories
	catOrder := []struct {
		key  string
		name string
	}{
		{"cli", "CLI Tools"},
		{"dev", "Developer Tools"},
		{"ai", "AI Tools"},
		{"apps", "Desktop Apps"},
		{"mas", "App Store"},
	}

	for name, app := range cfg.Apps {
		installed := isInstalledWithPkg(name, app.Install, app.Package)
		item := appItem{
			name:        name,
			description: app.Description,
			installed:   installed,
			selected:    false,
		}
		catMap[app.Category] = append(catMap[app.Category], item)
	}

	var cats []category
	for _, c := range catOrder {
		if apps, ok := catMap[c.key]; ok && len(apps) > 0 {
			cats = append(cats, category{
				name: c.name,
				key:  c.key,
				apps: apps,
			})
		}
	}

	return cats
}

func isInstalledWithPkg(name, installType, pkgName string) bool {
	if pkgName == "" {
		pkgName = name
	}
	switch installType {
	case "brew":
		cmd := exec.Command("/opt/homebrew/bin/brew", "list", pkgName)
		return cmd.Run() == nil
	case "cask":
		cmd := exec.Command("/opt/homebrew/bin/brew", "list", "--cask", pkgName)
		return cmd.Run() == nil
	case "npm":
		cmd := exec.Command("npm", "list", "-g", pkgName)
		return cmd.Run() == nil
	case "shell":
		// Shell-only apps check if their config exists
		home, _ := os.UserHomeDir()
		appDir := filepath.Join(home, ".config", "macos-setup", "apps", name)
		_, err := os.Stat(appDir)
		return err == nil
	default:
		return false
	}
}

func isInstalled(name string, installType string) bool {
	pkgName := name
	if appConfig != nil {
		if app, ok := appConfig.Apps[name]; ok && app.Package != "" {
			pkgName = app.Package
		}
	}
	return isInstalledWithPkg(name, installType, pkgName)
}

func (c *category) installedCount() int {
	count := 0
	for _, app := range c.apps {
		if app.installed {
			count++
		}
	}
	return count
}

func (c *category) selectedCount() int {
	count := 0
	for _, app := range c.apps {
		if app.selected {
			count++
		}
	}
	return count
}

func (c *category) selectedUninstalledCount() int {
	count := 0
	for _, app := range c.apps {
		if app.selected && !app.installed {
			count++
		}
	}
	return count
}

func (c *category) selectedInstalledCount() int {
	count := 0
	for _, app := range c.apps {
		if app.selected && app.installed {
			count++
		}
	}
	return count
}

func (c *category) getSelectedNames() []string {
	var names []string
	for _, app := range c.apps {
		if app.selected {
			names = append(names, app.name)
		}
	}
	return names
}

type installCompleteMsg struct {
	name    string
	success bool
}

type removeCompleteMsg struct {
	name    string
	success bool
}

type logLineMsg string

var program *tea.Program
var appConfig *config.Config

func streamToLog(r io.Reader) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		if program != nil {
			program.Send(logLineMsg(scanner.Text()))
		}
	}
}

func installAppCmd(name string) tea.Cmd {
	return func() tea.Msg {
		var result tea.Msg
		success := true

		// Check install type and package name
		installType := "brew"
		pkgName := name
		if appConfig != nil {
			if app, ok := appConfig.Apps[name]; ok {
				installType = app.Install
				if app.Package != "" {
					pkgName = app.Package
				}
			}
		}

		switch installType {
		case "shell":
			// Shell-only: just add integration, no install
			if program != nil {
				program.Send(logLineMsg(fmt.Sprintf("Setting up %s...", name)))
			}
		case "npm":
			cmd := exec.Command("npm", "install", "-g", pkgName)
			r := runCmdWithOutput(cmd, name, true)
			if msg, ok := r.(installCompleteMsg); ok {
				success = msg.success
			}
		default:
			// brew, cask
			cmd := exec.Command("/opt/homebrew/bin/brew", "install", pkgName)
			r := runCmdWithOutput(cmd, name, true)
			if msg, ok := r.(installCompleteMsg); ok {
				success = msg.success
			}
		}

		// Handle zsh integration if defined
		if success && appConfig != nil {
			if app, ok := appConfig.Apps[name]; ok && app.Zsh != "" {
				if err := addZshIntegration(name, app.Zsh); err != nil {
					if program != nil {
						program.Send(logLineMsg(fmt.Sprintf("Warning: zsh setup failed: %v", err)))
					}
				} else {
					if program != nil {
						program.Send(logLineMsg("Added shell integration"))
					}
				}
			}
		}

		// Run post_install commands if defined
		if success && appConfig != nil {
			if app, ok := appConfig.Apps[name]; ok && len(app.PostInstall) > 0 {
				for _, cmdStr := range app.PostInstall {
					if program != nil {
						program.Send(logLineMsg(fmt.Sprintf("Running: %s", cmdStr)))
					}
					cmd := exec.Command("zsh", "-c", cmdStr)
					stdout, _ := cmd.StdoutPipe()
					stderr, _ := cmd.StderrPipe()
					cmd.Start()
					go streamToLog(stdout)
					go streamToLog(stderr)
					if err := cmd.Wait(); err != nil {
						if program != nil {
							program.Send(logLineMsg(fmt.Sprintf("Warning: %s failed: %v", cmdStr, err)))
						}
					}
				}
			}
		}

		result = installCompleteMsg{name: name, success: success}
		return result
	}
}

func addZshIntegration(name, zshContent string) error {
	home, _ := os.UserHomeDir()
	baseDir := filepath.Join(home, ".config", "macos-setup")
	appDir := filepath.Join(baseDir, "apps", name)

	// Ensure app directory exists
	if err := os.MkdirAll(appDir, 0755); err != nil {
		return err
	}

	// Always write init.zsh (auto-generated, safe to overwrite)
	initPath := filepath.Join(baseDir, "init.zsh")
	initContent := `# macos-setup shell integration (auto-generated)
# Sources all *.zsh files from ~/.config/macos-setup/apps/*/
for f in $HOME/.config/macos-setup/apps/*/*.zsh(N); do
  source "$f"
done
`
	if err := os.WriteFile(initPath, []byte(initContent), 0644); err != nil {
		return err
	}

	// Ensure .zshrc sources our init.zsh
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

		// Mark zshrc as modified
		markerPath := filepath.Join(baseDir, ".zshrc-modified")
		os.WriteFile(markerPath, []byte{}, 0644)
	}

	// Write app-specific zshrc
	appZshPath := filepath.Join(appDir, "zshrc.zsh")
	return os.WriteFile(appZshPath, []byte(strings.TrimSpace(zshContent)+"\n"), 0644)
}

func removeAppCmd(name string) tea.Cmd {
	return func() tea.Msg {
		var result tea.Msg

		// Check install type and package name
		installType := "brew"
		pkgName := name
		if appConfig != nil {
			if app, ok := appConfig.Apps[name]; ok {
				installType = app.Install
				if app.Package != "" {
					pkgName = app.Package
				}
			}
		}

		switch installType {
		case "shell":
			// Shell-only: just remove config
			if program != nil {
				program.Send(logLineMsg(fmt.Sprintf("Removing %s config...", name)))
			}
			result = removeCompleteMsg{name: name, success: true}
		case "npm":
			cmd := exec.Command("npm", "uninstall", "-g", pkgName)
			result = runCmdWithOutput(cmd, name, false)
		default:
			cmd := exec.Command("/opt/homebrew/bin/brew", "uninstall", pkgName)
			result = runCmdWithOutput(cmd, name, false)
		}

		// Remove app config directory if exists
		home, _ := os.UserHomeDir()
		appDir := filepath.Join(home, ".config", "macos-setup", "apps", name)
		if _, err := os.Stat(appDir); err == nil {
			os.RemoveAll(appDir)
			if program != nil {
				program.Send(logLineMsg("Removed shell integration"))
			}
		}

		return result
	}
}

func reinstallAppCmd(name string) tea.Cmd {
	return func() tea.Msg {
		var result tea.Msg
		success := true

		// Check install type and package name
		installType := "brew"
		pkgName := name
		if appConfig != nil {
			if app, ok := appConfig.Apps[name]; ok {
				installType = app.Install
				if app.Package != "" {
					pkgName = app.Package
				}
			}
		}

		switch installType {
		case "shell":
			// Shell-only: just update integration
			if program != nil {
				program.Send(logLineMsg(fmt.Sprintf("Updating %s config...", name)))
			}
		case "npm":
			// npm reinstall = uninstall + install
			cmd := exec.Command("npm", "install", "-g", pkgName)
			r := runCmdWithOutput(cmd, name, true)
			if msg, ok := r.(installCompleteMsg); ok {
				success = msg.success
			}
		default:
			cmd := exec.Command("/opt/homebrew/bin/brew", "reinstall", pkgName)
			r := runCmdWithOutput(cmd, name, true)
			if msg, ok := r.(installCompleteMsg); ok {
				success = msg.success
			}
		}

		result = installCompleteMsg{name: name, success: success}

		// Re-apply zsh integration if defined
		if appConfig != nil {
			if app, ok := appConfig.Apps[name]; ok && app.Zsh != "" {
				if err := updateZshIntegration(name, app.Zsh); err != nil {
					if program != nil {
						program.Send(logLineMsg(fmt.Sprintf("Warning: zsh setup failed: %v", err)))
					}
				} else {
					if program != nil {
						program.Send(logLineMsg("Updated shell integration"))
					}
				}
			}
		}

		return result
	}
}

func updateZshIntegration(name, zshContent string) error {
	// Just overwrite the app's config file
	return addZshIntegration(name, zshContent)
}

func runCmdWithOutput(cmd *exec.Cmd, name string, isInstall bool) tea.Msg {
	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()

	if err := cmd.Start(); err != nil {
		if program != nil {
			program.Send(logLineMsg(fmt.Sprintf("Error starting: %v", err)))
		}
		if isInstall {
			return installCompleteMsg{name: name, success: false}
		}
		return removeCompleteMsg{name: name, success: false}
	}

	// Stream output in goroutines
	streamOutput := func(r io.Reader) {
		scanner := bufio.NewScanner(r)
		for scanner.Scan() {
			if program != nil {
				program.Send(logLineMsg(scanner.Text()))
			}
		}
	}

	go streamOutput(stdout)
	go streamOutput(stderr)

	err := cmd.Wait()

	if isInstall {
		return installCompleteMsg{name: name, success: err == nil}
	}
	return removeCompleteMsg{name: name, success: err == nil}
}

func (m model) Init() tea.Cmd {
	return tea.SetWindowTitle("macos-setup")
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case logLineMsg:
		m.logLines = append(m.logLines, string(msg))
		// Auto-scroll to bottom
		maxVisible := m.getLogHeight() - 2 // account for title + border
		if maxVisible < 1 {
			maxVisible = 1
		}
		if len(m.logLines) > maxVisible {
			m.logScroll = len(m.logLines) - maxVisible
		}
		return m, tea.SetWindowTitle("macos-setup")
	case installCompleteMsg:
		return m.handleInstallComplete(msg)
	case removeCompleteMsg:
		return m.handleRemoveComplete(msg)
	case tea.KeyMsg:
		if m.state != stateIdle {
			return m, nil
		}
		if m.inCategory {
			return m.updateInCategory(msg)
		}
		return m.updateMain(msg)
	}
	return m, nil
}

func (m model) getLogHeight() int {
	if m.height == 0 {
		return 10
	}
	if m.layout == layoutVertical {
		return (m.height - 6) / 2 // half height minus borders/footer
	}
	return m.height - 6 // full height minus borders/footer
}

// addWithDeps adds an app and its dependencies to the install list
func (m model) addWithDeps(name string, list []string, seen map[string]bool) []string {
	if seen[name] {
		return list
	}
	seen[name] = true

	// Check if already installed
	if m.config != nil {
		if app, ok := m.config.Apps[name]; ok {
			// Add dependencies first (recursively)
			for _, dep := range app.Depends {
				if !isInstalled(dep, "brew") {
					list = m.addWithDeps(dep, list, seen)
				}
			}
		}
	}

	// Add the app itself
	list = append(list, name)
	return list
}

// addDeps adds dependencies for an app to requiredDeps
func (m *model) addDeps(appName string) {
	if m.config == nil {
		return
	}
	app, ok := m.config.Apps[appName]
	if !ok {
		return
	}

	for _, dep := range app.Depends {
		// Add this app to the list of apps requiring this dep
		found := false
		for _, existing := range m.requiredDeps[dep] {
			if existing == appName {
				found = true
				break
			}
		}
		if !found {
			m.requiredDeps[dep] = append(m.requiredDeps[dep], appName)
		}
	}
}

// removeDeps removes dependencies for an app from requiredDeps
func (m *model) removeDeps(appName string) {
	if m.config == nil {
		return
	}
	app, ok := m.config.Apps[appName]
	if !ok {
		return
	}

	for _, dep := range app.Depends {
		// Remove this app from the list of apps requiring this dep
		newList := []string{}
		for _, existing := range m.requiredDeps[dep] {
			if existing != appName {
				newList = append(newList, existing)
			}
		}
		if len(newList) == 0 {
			delete(m.requiredDeps, dep)
		} else {
			m.requiredDeps[dep] = newList
		}
	}
}

func (m model) handleInstallComplete(msg installCompleteMsg) (tea.Model, tea.Cmd) {
	// Search all categories for the app
	for catIdx := range m.categories {
		for appIdx := range m.categories[catIdx].apps {
			if m.categories[catIdx].apps[appIdx].name == msg.name {
				m.categories[catIdx].apps[appIdx].installed = msg.success
				m.categories[catIdx].apps[appIdx].selected = false
				break
			}
		}
	}

	m.progressIdx++

	if m.progressIdx < len(m.progressApps) {
		nextApp := m.progressApps[m.progressIdx]
		m.progressMsg = fmt.Sprintf("Installing %s... (%d/%d)", nextApp, m.progressIdx+1, len(m.progressApps))
		return m, tea.Batch(tea.SetWindowTitle("macos-setup"), installAppCmd(nextApp))
	}

	m.state = stateIdle
	m.progressMsg = ""
	m.progressApps = nil
	// Clear all selections when done
	for catIdx := range m.categories {
		for appIdx := range m.categories[catIdx].apps {
			m.categories[catIdx].apps[appIdx].selected = false
		}
	}
	m.requiredDeps = make(map[string][]string)
	return m, tea.SetWindowTitle("macos-setup")
}

func (m model) handleRemoveComplete(msg removeCompleteMsg) (tea.Model, tea.Cmd) {
	// Search all categories for the app
	for catIdx := range m.categories {
		for appIdx := range m.categories[catIdx].apps {
			if m.categories[catIdx].apps[appIdx].name == msg.name {
				if msg.success {
					m.categories[catIdx].apps[appIdx].installed = false
				}
				m.categories[catIdx].apps[appIdx].selected = false
				break
			}
		}
	}

	m.progressIdx++

	if m.progressIdx < len(m.progressApps) {
		nextApp := m.progressApps[m.progressIdx]
		m.progressMsg = fmt.Sprintf("Removing %s... (%d/%d)", nextApp, m.progressIdx+1, len(m.progressApps))
		return m, tea.Batch(tea.SetWindowTitle("macos-setup"), removeAppCmd(nextApp))
	}

	m.state = stateIdle
	m.progressMsg = ""
	m.progressApps = nil
	return m, tea.SetWindowTitle("macos-setup")
}

func (m model) updateMain(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, keys.Quit):
		m.quitting = true
		return m, tea.Quit
	case key.Matches(msg, keys.Up):
		if m.cursor > 0 {
			m.cursor--
		}
	case key.Matches(msg, keys.Down):
		if m.cursor < len(m.categories)-1 {
			m.cursor++
		}
	case key.Matches(msg, keys.Enter):
		m.inCategory = true
		m.appCursor = 0
	case key.Matches(msg, keys.Space):
		// Toggle all apps in current category
		cat := &m.categories[m.cursor]
		allSelected := cat.selectedCount() == len(cat.apps)
		for i := range cat.apps {
			if allSelected {
				cat.apps[i].selected = false
				m.removeDeps(cat.apps[i].name)
			} else {
				if !cat.apps[i].selected {
					cat.apps[i].selected = true
					m.addDeps(cat.apps[i].name)
				}
			}
		}
	case key.Matches(msg, keys.Layout):
		if m.layout == layoutHorizontal {
			m.layout = layoutVertical
		} else {
			m.layout = layoutHorizontal
		}
	case key.Matches(msg, keys.SelectAll):
		// Select all apps in all categories
		for catIdx := range m.categories {
			for appIdx := range m.categories[catIdx].apps {
				if !m.categories[catIdx].apps[appIdx].selected {
					m.categories[catIdx].apps[appIdx].selected = true
					m.addDeps(m.categories[catIdx].apps[appIdx].name)
				}
			}
		}
	case key.Matches(msg, keys.Install):
		// Install all selected apps across all categories
		var toInstall []string
		seen := make(map[string]bool)

		// Add dependencies first
		for dep := range m.requiredDeps {
			if !seen[dep] && !isInstalled(dep, "brew") {
				toInstall = append(toInstall, dep)
				seen[dep] = true
			}
		}

		// Add selected apps from all categories
		for catIdx := range m.categories {
			for _, app := range m.categories[catIdx].apps {
				if app.selected && !app.installed && !seen[app.name] {
					toInstall = append(toInstall, app.name)
					seen[app.name] = true
				}
			}
		}

		if len(toInstall) > 0 {
			m.state = stateInstalling
			m.progressApps = toInstall
			m.progressIdx = 0
			m.progressMsg = fmt.Sprintf("Installing %s... (1/%d)", toInstall[0], len(toInstall))
			return m, installAppCmd(toInstall[0])
		}
	case key.Matches(msg, keys.Remove):
		// Remove all selected installed apps across all categories
		var toRemove []string
		seen := make(map[string]bool)

		for catIdx := range m.categories {
			for _, app := range m.categories[catIdx].apps {
				if app.selected && app.installed && !seen[app.name] {
					toRemove = append(toRemove, app.name)
					seen[app.name] = true
				}
			}
		}

		if len(toRemove) > 0 {
			m.state = stateRemoving
			m.progressApps = toRemove
			m.progressIdx = 0
			m.progressMsg = fmt.Sprintf("Removing %s... (1/%d)", toRemove[0], len(toRemove))
			return m, removeAppCmd(toRemove[0])
		}
	case key.Matches(msg, keys.InstallForce):
		// Reinstall all selected installed apps across all categories
		var toReinstall []string
		seen := make(map[string]bool)

		for catIdx := range m.categories {
			for _, app := range m.categories[catIdx].apps {
				if app.selected && app.installed && !seen[app.name] {
					toReinstall = append(toReinstall, app.name)
					seen[app.name] = true
				}
			}
		}

		if len(toReinstall) > 0 {
			m.state = stateInstalling
			m.progressApps = toReinstall
			m.progressIdx = 0
			m.progressMsg = fmt.Sprintf("Reinstalling %s... (1/%d)", toReinstall[0], len(toReinstall))
			return m, reinstallAppCmd(toReinstall[0])
		}
	}
	return m, nil
}

func (m model) updateInCategory(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	cat := &m.categories[m.cursor]
	switch {
	case key.Matches(msg, keys.Quit):
		m.quitting = true
		return m, tea.Quit
	case key.Matches(msg, keys.Back):
		m.inCategory = false
	case key.Matches(msg, keys.Up):
		if m.appCursor > 0 {
			m.appCursor--
		}
	case key.Matches(msg, keys.Down):
		if m.appCursor < len(cat.apps)-1 {
			m.appCursor++
		}
	case key.Matches(msg, keys.Space):
		app := &m.categories[m.cursor].apps[m.appCursor]
		app.selected = !app.selected
		if app.selected {
			m.addDeps(app.name)
		} else {
			m.removeDeps(app.name)
		}
	case key.Matches(msg, keys.SelectAll):
		// Toggle: if any unselected, select all; otherwise deselect all
		anyUnselected := false
		for i := range m.categories[m.cursor].apps {
			if !m.categories[m.cursor].apps[i].selected {
				anyUnselected = true
				break
			}
		}
		for i := range m.categories[m.cursor].apps {
			wasSelected := m.categories[m.cursor].apps[i].selected
			m.categories[m.cursor].apps[i].selected = anyUnselected
			appName := m.categories[m.cursor].apps[i].name
			if anyUnselected && !wasSelected {
				m.addDeps(appName)
			} else if !anyUnselected && wasSelected {
				m.removeDeps(appName)
			}
		}
	case key.Matches(msg, keys.Install):
		var toInstall []string
		seen := make(map[string]bool)

		// Add dependencies first (not already installed)
		for dep := range m.requiredDeps {
			if !seen[dep] && !isInstalled(dep, "brew") {
				toInstall = append(toInstall, dep)
				seen[dep] = true
			}
		}

		// Add selected apps
		selected := cat.getSelectedNames()
		for _, name := range selected {
			for _, app := range cat.apps {
				if app.name == name && !app.installed && !seen[name] {
					toInstall = append(toInstall, name)
					seen[name] = true
					break
				}
			}
		}

		if len(toInstall) > 0 {
			m.state = stateInstalling
			m.progressApps = toInstall
			m.progressIdx = 0
			m.progressMsg = fmt.Sprintf("Installing %s... (1/%d)", toInstall[0], len(toInstall))
			return m, installAppCmd(toInstall[0])
		}
	case key.Matches(msg, keys.Remove):
		selected := cat.getSelectedNames()
		if len(selected) > 0 {
			var toRemove []string
			for _, name := range selected {
				for _, app := range cat.apps {
					if app.name == name && app.installed {
						toRemove = append(toRemove, name)
						break
					}
				}
			}
			if len(toRemove) > 0 {
				m.state = stateRemoving
				m.progressApps = toRemove
				m.progressIdx = 0
				m.progressMsg = fmt.Sprintf("Removing %s... (1/%d)", toRemove[0], len(toRemove))
				return m, removeAppCmd(toRemove[0])
			}
		}
	case key.Matches(msg, keys.InstallForce):
		selected := cat.getSelectedNames()
		if len(selected) > 0 {
			var toReinstall []string
			for _, name := range selected {
				for _, app := range cat.apps {
					if app.name == name && app.installed {
						toReinstall = append(toReinstall, name)
						break
					}
				}
			}
			if len(toReinstall) > 0 {
				m.state = stateInstalling
				m.progressApps = toReinstall
				m.progressIdx = 0
				m.progressMsg = fmt.Sprintf("Reinstalling %s... (1/%d)", toReinstall[0], len(toReinstall))
				return m, reinstallAppCmd(toReinstall[0])
			}
		}
	case key.Matches(msg, keys.Layout):
		if m.layout == layoutHorizontal {
			m.layout = layoutVertical
		} else {
			m.layout = layoutHorizontal
		}
	}
	return m, nil
}

var (
	logTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("39"))

	logStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("245"))

	paneStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")).
			Padding(0, 1)

	footerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Padding(0, 1)
)

func (m model) View() string {
	if m.quitting {
		return ""
	}

	if m.width == 0 || m.height == 0 {
		return "Loading..."
	}

	// Build footer
	footer := m.buildFooter()
	footerHeight := 1

	// Available space for panes
	availableHeight := m.height - footerHeight - 2 // -2 for margins
	availableWidth := m.width - 2

	var mainPane, logPane string
	var mainWidth, logWidth, mainHeight, logHeight int

	if m.layout == layoutHorizontal {
		// Side-by-side
		mainWidth = availableWidth/2 - 2
		logWidth = availableWidth - mainWidth - 4
		mainHeight = availableHeight - 2
		logHeight = availableHeight - 2
	} else {
		// Top-bottom
		mainWidth = availableWidth - 2
		logWidth = availableWidth - 2
		mainHeight = availableHeight/2 - 2
		logHeight = availableHeight - mainHeight - 4
	}

	// Render main content pane
	var content string
	if m.inCategory {
		content = m.viewCategoryContent()
	} else {
		content = m.viewMainContent()
	}
	paneFooter := ""
	if m.progressMsg != "" {
		paneFooter = progressStyle.Render(m.progressMsg)
	}
	mainPane = m.renderPane("", content, paneFooter, mainWidth, mainHeight)

	// Render log pane (subtract for title + border)
	logContent := m.viewLog(logHeight - 2)
	logPane = m.renderPane("Log", logContent, "", logWidth, logHeight)

	// Combine panes based on layout
	var combined string
	if m.layout == layoutHorizontal {
		combined = lipgloss.JoinHorizontal(lipgloss.Top, mainPane, " ", logPane)
	} else {
		combined = lipgloss.JoinVertical(lipgloss.Left, mainPane, logPane)
	}

	// Add footer
	return lipgloss.JoinVertical(lipgloss.Left, combined, footer)
}

func (m model) renderPane(title string, content string, footer string, width, height int) string {
	if width < 10 {
		width = 10
	}
	if height < 3 {
		height = 3
	}

	// Add title if present
	var s string
	contentHeight := height
	if title != "" {
		s = logTitleStyle.Render(title) + "\n"
		contentHeight--
	}

	// Reserve space for footer if present
	footerHeight := 0
	if footer != "" {
		footerHeight = 2 // blank line + footer
		contentHeight -= footerHeight
	}

	// Pad/truncate content to fit
	lines := strings.Split(content, "\n")
	for len(lines) < contentHeight {
		lines = append(lines, "")
	}
	if len(lines) > contentHeight {
		lines = lines[:contentHeight]
	}
	// Truncate long lines
	for i, line := range lines {
		if len(line) > width-2 {
			lines[i] = line[:width-5] + "..."
		}
	}
	s += strings.Join(lines, "\n")

	// Add footer at bottom
	if footer != "" {
		s += "\n\n" + footer
	}

	return paneStyle.Width(width).Height(height).Render(s)
}

func (m model) viewLog(height int) string {
	if len(m.logLines) == 0 {
		return logStyle.Render("No output yet...")
	}

	start := m.logScroll
	end := start + height
	if end > len(m.logLines) {
		end = len(m.logLines)
	}
	if start > len(m.logLines) {
		start = 0
		if end > len(m.logLines) {
			end = len(m.logLines)
		}
	}

	visible := m.logLines[start:end]
	return logStyle.Render(strings.Join(visible, "\n"))
}

func (m model) buildFooter() string {
	var parts []string
	parts = append(parts, "↑/↓ nav")

	if m.inCategory {
		parts = append(parts, "space sel", "a all")
		cat := m.categories[m.cursor]
		toInstall := cat.selectedUninstalledCount()
		toReinstall := cat.selectedInstalledCount()
		if toInstall > 0 {
			parts = append(parts, fmt.Sprintf("i inst(%d)", toInstall))
		}
		if toReinstall > 0 {
			parts = append(parts, fmt.Sprintf("I reinstall(%d)", toReinstall))
			parts = append(parts, fmt.Sprintf("r rem(%d)", toReinstall))
		}
		parts = append(parts, "esc back")
	} else {
		parts = append(parts, "space sel", "enter open", "a all")
		totalToInstall := m.totalSelectedCount()
		totalToReinstall := m.totalSelectedInstalledCount()
		if totalToInstall > 0 {
			parts = append(parts, fmt.Sprintf("i inst(%d)", totalToInstall))
		}
		if totalToReinstall > 0 {
			parts = append(parts, fmt.Sprintf("I reinstall(%d)", totalToReinstall))
			parts = append(parts, fmt.Sprintf("r rem(%d)", totalToReinstall))
		}
	}

	layoutName := "horiz"
	if m.layout == layoutVertical {
		layoutName = "vert"
	}
	parts = append(parts, fmt.Sprintf("tab layout(%s)", layoutName))
	parts = append(parts, "q quit")

	return footerStyle.Render(strings.Join(parts, " • "))
}

func (m model) totalSelectedCount() int {
	count := 0
	for _, cat := range m.categories {
		for _, app := range cat.apps {
			if app.selected && !app.installed {
				count++
			}
		}
	}
	return count
}

func (m model) totalSelectedInstalledCount() int {
	count := 0
	for _, cat := range m.categories {
		for _, app := range cat.apps {
			if app.selected && app.installed {
				count++
			}
		}
	}
	return count
}

func (m model) viewMainContent() string {
	s := titleStyle.Render("macos-setup") + "\n\n"

	for i, cat := range m.categories {
		installed := cat.installedCount()
		selected := cat.selectedCount()
		total := len(cat.apps)
		status := fmt.Sprintf("[%d/%d]", installed, total)

		// Selection checkbox: [ ] none, [~] some, [x] all
		checkbox := "[ ]"
		if selected == total && total > 0 {
			checkbox = "[x]"
		} else if selected > 0 {
			checkbox = "[~]"
		}

		// Install indicator
		indicator := "○"
		if installed > 0 {
			indicator = "●"
		}
		if installed == total && total > 0 {
			indicator = "✓"
		}

		line := fmt.Sprintf("%s %s %-18s %s  ▸", checkbox, indicator, cat.name, statusStyle.Render(status))

		if i == m.cursor {
			s += selectedStyle.Render(line) + "\n"
		} else {
			s += categoryStyle.Render(line) + "\n"
		}
	}

	// Show dependencies section if any
	if len(m.requiredDeps) > 0 {
		s += "\n" + statusStyle.Render("Dependencies:") + "\n"
		for dep, apps := range m.requiredDeps {
			status := "○"
			installing := false
			for i, name := range m.progressApps {
				if name == dep && i >= m.progressIdx {
					installing = true
					break
				}
			}
			if installing {
				status = "◐"
			} else if isInstalled(dep, "brew") {
				status = "✓"
			}
			appList := strings.Join(apps, ", ")
			line := fmt.Sprintf("     %s %-12s %s", status, dep, statusStyle.Render("["+appList+"]"))
			s += appStyle.Render(line) + "\n"
		}
	}

	return s
}

func (m model) viewCategoryContent() string {
	cat := m.categories[m.cursor]
	s := titleStyle.Render(cat.name) + "\n\n"

	for i, app := range cat.apps {
		checkbox := "[ ]"
		if app.selected {
			checkbox = "[x]"
		}

		status := "○"
		if app.installed {
			status = "✓"
		}

		line := fmt.Sprintf("%s %s %-12s %s", checkbox, status, app.name, statusStyle.Render(app.description))

		if i == m.appCursor {
			s += appSelectedStyle.Render(line) + "\n"
		} else {
			s += appStyle.Render(line) + "\n"
		}
	}

	// Show dependencies section if any
	if len(m.requiredDeps) > 0 {
		s += "\n" + statusStyle.Render("Dependencies:") + "\n"
		for dep, apps := range m.requiredDeps {
			status := "○"
			// Check if currently installing
			installing := false
			for i, name := range m.progressApps {
				if name == dep && i >= m.progressIdx {
					installing = true
					break
				}
			}
			if installing {
				status = "◐" // in progress
			} else if isInstalled(dep, "brew") {
				status = "✓"
			}
			appList := strings.Join(apps, ", ")
			line := fmt.Sprintf("     %s %-12s %s", status, dep, statusStyle.Render("["+appList+"]"))
			s += appStyle.Render(line) + "\n"
		}
	}

	return s
}

type keyMap struct {
	Up           key.Binding
	Down         key.Binding
	Enter        key.Binding
	Back         key.Binding
	Space        key.Binding
	SelectAll    key.Binding
	Install      key.Binding
	InstallForce key.Binding
	Remove       key.Binding
	Layout       key.Binding
	Quit         key.Binding
}

var keys = keyMap{
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
	),
	Enter: key.NewBinding(
		key.WithKeys("enter", "l"),
	),
	Back: key.NewBinding(
		key.WithKeys("esc", "h"),
	),
	Space: key.NewBinding(
		key.WithKeys(" "),
	),
	SelectAll: key.NewBinding(
		key.WithKeys("a"),
	),
	Install: key.NewBinding(
		key.WithKeys("i"),
	),
	InstallForce: key.NewBinding(
		key.WithKeys("I"),
	),
	Remove: key.NewBinding(
		key.WithKeys("r"),
	),
	Layout: key.NewBinding(
		key.WithKeys("tab"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
	),
}

func Run() error {
	m := initialModel()
	appConfig = m.config

	// Bootstrap core shell aliases on first run
	home, _ := os.UserHomeDir()
	coreApps := map[string]string{
		"macos-setup": "alias macos=macos-setup\nalias refreshenv='source ~/.zshrc'",
		"brew":        "alias brewup='brew update && brew upgrade'",
	}
	for name, zsh := range coreApps {
		appDir := filepath.Join(home, ".config", "macos-setup", "apps", name)
		if _, err := os.Stat(appDir); os.IsNotExist(err) {
			addZshIntegration(name, zsh)
		}
	}

	p := tea.NewProgram(m, tea.WithAltScreen())
	program = p
	_, err := p.Run()

	// Check if .zshrc was modified and notify user
	markerPath := filepath.Join(home, ".config", "macos-setup", ".zshrc-modified")
	if _, statErr := os.Stat(markerPath); statErr == nil {
		fmt.Println()
		fmt.Println("\033[1;33m⚠\033[0m  ~/.zshrc was modified. Run: \033[1msource ~/.zshrc\033[0m")
		fmt.Println()
		os.Remove(markerPath)
	}

	return err
}
