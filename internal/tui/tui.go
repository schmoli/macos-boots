package tui

import (
	"fmt"
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
			Foreground(lipgloss.Color("205")).
			MarginBottom(1)

	categoryStyle = lipgloss.NewStyle().
			PaddingLeft(2)

	selectedStyle = lipgloss.NewStyle().
			PaddingLeft(2).
			Foreground(lipgloss.Color("170")).
			Bold(true)

	statusStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			MarginTop(1)

	appStyle = lipgloss.NewStyle().
			PaddingLeft(4)

	appSelectedStyle = lipgloss.NewStyle().
				PaddingLeft(4).
				Foreground(lipgloss.Color("170")).
				Bold(true)

	progressStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("39"))
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
}

func initialModel() model {
	home, _ := os.UserHomeDir()
	configPath := filepath.Join(home, ".config", "macos-setup", "repo", "apps.yaml")

	cfg, err := config.Load(configPath)
	if err != nil {
		return model{
			categories: []category{
				{name: "CLI Tools", key: "cli"},
			},
		}
	}

	cats := buildCategories(cfg)

	return model{
		config:     cfg,
		categories: cats,
		cursor:     0,
	}
}

func buildCategories(cfg *config.Config) []category {
	catMap := make(map[string][]appItem)
	catNames := map[string]string{
		"cli":  "CLI Tools",
		"apps": "Desktop Apps",
		"mas":  "App Store",
	}

	for name, app := range cfg.Apps {
		installed := isInstalled(name, app.Install)
		item := appItem{
			name:        name,
			description: app.Description,
			installed:   installed,
			selected:    false,
		}
		catMap[app.Category] = append(catMap[app.Category], item)
	}

	var cats []category
	for key, displayName := range catNames {
		if apps, ok := catMap[key]; ok && len(apps) > 0 {
			cats = append(cats, category{
				name: displayName,
				key:  key,
				apps: apps,
			})
		}
	}

	return cats
}

func isInstalled(name string, installType string) bool {
	switch installType {
	case "brew":
		cmd := exec.Command("/opt/homebrew/bin/brew", "list", name)
		return cmd.Run() == nil
	case "cask":
		cmd := exec.Command("/opt/homebrew/bin/brew", "list", "--cask", name)
		return cmd.Run() == nil
	default:
		return false
	}
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

func (c *category) getSelectedNames() []string {
	var names []string
	for _, app := range c.apps {
		if app.selected {
			names = append(names, app.name)
		}
	}
	return names
}

// Messages for async operations
type installCompleteMsg struct {
	name    string
	success bool
}

type removeCompleteMsg struct {
	name    string
	success bool
}

func installAppCmd(name string) tea.Cmd {
	return func() tea.Msg {
		cmd := exec.Command("/opt/homebrew/bin/brew", "install", "-q", name)
		err := cmd.Run()
		return installCompleteMsg{name: name, success: err == nil}
	}
}

func removeAppCmd(name string) tea.Cmd {
	return func() tea.Msg {
		cmd := exec.Command("/opt/homebrew/bin/brew", "uninstall", "-q", name)
		err := cmd.Run()
		return removeCompleteMsg{name: name, success: err == nil}
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case installCompleteMsg:
		return m.handleInstallComplete(msg)
	case removeCompleteMsg:
		return m.handleRemoveComplete(msg)
	case tea.KeyMsg:
		if m.state != stateIdle {
			return m, nil // Ignore keys during install/remove
		}
		if m.inCategory {
			return m.updateInCategory(msg)
		}
		return m.updateMain(msg)
	}
	return m, nil
}

func (m model) handleInstallComplete(msg installCompleteMsg) (tea.Model, tea.Cmd) {
	cat := &m.categories[m.cursor]

	// Mark as installed and deselect
	for i := range cat.apps {
		if cat.apps[i].name == msg.name {
			cat.apps[i].installed = msg.success
			cat.apps[i].selected = false
			break
		}
	}

	m.progressIdx++

	// Continue with next app or finish
	if m.progressIdx < len(m.progressApps) {
		nextApp := m.progressApps[m.progressIdx]
		m.progressMsg = fmt.Sprintf("Installing %s... (%d/%d)", nextApp, m.progressIdx+1, len(m.progressApps))
		return m, installAppCmd(nextApp)
	}

	// Done
	m.state = stateIdle
	m.progressMsg = ""
	m.progressApps = nil
	return m, nil
}

func (m model) handleRemoveComplete(msg removeCompleteMsg) (tea.Model, tea.Cmd) {
	cat := &m.categories[m.cursor]

	// Mark as uninstalled and deselect
	for i := range cat.apps {
		if cat.apps[i].name == msg.name {
			if msg.success {
				cat.apps[i].installed = false
			}
			cat.apps[i].selected = false
			break
		}
	}

	m.progressIdx++

	// Continue with next app or finish
	if m.progressIdx < len(m.progressApps) {
		nextApp := m.progressApps[m.progressIdx]
		m.progressMsg = fmt.Sprintf("Removing %s... (%d/%d)", nextApp, m.progressIdx+1, len(m.progressApps))
		return m, removeAppCmd(nextApp)
	}

	// Done
	m.state = stateIdle
	m.progressMsg = ""
	m.progressApps = nil
	return m, nil
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
		// Toggle selection
		cat.apps[m.appCursor].selected = !cat.apps[m.appCursor].selected
	case key.Matches(msg, keys.SelectAll):
		// Select all uninstalled
		for i := range cat.apps {
			if !cat.apps[i].installed {
				cat.apps[i].selected = true
			}
		}
	case key.Matches(msg, keys.Install):
		// Install selected
		selected := cat.getSelectedNames()
		if len(selected) > 0 {
			// Filter to only uninstalled
			var toInstall []string
			for _, name := range selected {
				for _, app := range cat.apps {
					if app.name == name && !app.installed {
						toInstall = append(toInstall, name)
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
		}
	case key.Matches(msg, keys.Remove):
		// Remove selected
		selected := cat.getSelectedNames()
		if len(selected) > 0 {
			// Filter to only installed
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
	}
	return m, nil
}

func (m model) View() string {
	if m.quitting {
		return ""
	}

	if m.inCategory {
		return m.viewCategory()
	}

	return m.viewMain()
}

func (m model) viewMain() string {
	s := titleStyle.Render("macos-setup") + "\n\n"

	for i, cat := range m.categories {
		installed := cat.installedCount()
		total := len(cat.apps)
		status := fmt.Sprintf("[%d/%d]", installed, total)

		indicator := "○"
		if installed > 0 {
			indicator = "●"
		}
		if installed == total && total > 0 {
			indicator = "✓"
		}

		line := fmt.Sprintf("%s %-20s %s  ▸", indicator, cat.name, statusStyle.Render(status))

		if i == m.cursor {
			s += selectedStyle.Render(line) + "\n"
		} else {
			s += categoryStyle.Render(line) + "\n"
		}
	}

	s += helpStyle.Render("\n↑/↓ navigate • enter select • q quit")

	return s
}

func (m model) viewCategory() string {
	cat := m.categories[m.cursor]
	s := titleStyle.Render(cat.name) + "\n\n"

	for i, app := range cat.apps {
		// Checkbox
		checkbox := "[ ]"
		if app.selected {
			checkbox = "[x]"
		}

		// Install status
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

	// Progress message
	if m.progressMsg != "" {
		s += "\n" + progressStyle.Render(m.progressMsg)
	}

	// Help
	var helpParts []string
	helpParts = append(helpParts, "↑/↓ navigate", "space select", "a select all")

	selectedCount := cat.selectedCount()
	if selectedCount > 0 {
		helpParts = append(helpParts, fmt.Sprintf("i install(%d)", selectedCount))
		helpParts = append(helpParts, fmt.Sprintf("r remove(%d)", selectedCount))
	}
	helpParts = append(helpParts, "esc back", "q quit")

	s += helpStyle.Render("\n" + strings.Join(helpParts, " • "))

	return s
}

type keyMap struct {
	Up        key.Binding
	Down      key.Binding
	Enter     key.Binding
	Back      key.Binding
	Space     key.Binding
	SelectAll key.Binding
	Install   key.Binding
	Remove    key.Binding
	Quit      key.Binding
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
	Remove: key.NewBinding(
		key.WithKeys("r"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
	),
}

func Run() error {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	_, err := p.Run()
	return err
}
