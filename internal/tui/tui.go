package tui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

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
)

type category struct {
	name  string
	key   string
	apps  []appItem
}

type appItem struct {
	name        string
	description string
	installed   bool
}

type model struct {
	config     *config.Config
	categories []category
	cursor     int
	inCategory bool
	appCursor  int
	quitting   bool
}

func initialModel() model {
	home, _ := os.UserHomeDir()
	configPath := filepath.Join(home, ".config", "macos-setup", "repo", "apps.yaml")

	cfg, err := config.Load(configPath)
	if err != nil {
		// Return empty model if config fails to load
		return model{
			categories: []category{
				{name: "CLI Tools", key: "cli"},
			},
		}
	}

	// Build categories from config
	cats := buildCategories(cfg)

	return model{
		config:     cfg,
		categories: cats,
		cursor:     0,
	}
}

func buildCategories(cfg *config.Config) []category {
	// Group apps by category
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
		}
		catMap[app.Category] = append(catMap[app.Category], item)
	}

	// Build category list (only include non-empty categories)
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

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.inCategory {
			return m.updateInCategory(msg)
		}
		return m.updateMain(msg)
	}
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
	case key.Matches(msg, keys.Quit), key.Matches(msg, keys.Back):
		m.inCategory = false
	case key.Matches(msg, keys.Up):
		if m.appCursor > 0 {
			m.appCursor--
		}
	case key.Matches(msg, keys.Down):
		if m.appCursor < len(cat.apps)-1 {
			m.appCursor++
		}
	case key.Matches(msg, keys.Enter), key.Matches(msg, keys.InstallAll):
		// Install selected app
		app := cat.apps[m.appCursor]
		if !app.installed {
			installApp(app.name)
			cat.apps[m.appCursor].installed = true
		}
	}
	return m, nil
}

func installApp(name string) {
	cmd := exec.Command("/opt/homebrew/bin/brew", "install", "-q", name)
	cmd.Run()
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
		indicator := "○"
		if app.installed {
			indicator = "✓"
		}

		line := fmt.Sprintf("%s %-15s %s", indicator, app.name, statusStyle.Render(app.description))

		if i == m.appCursor {
			s += appSelectedStyle.Render(line) + "\n"
		} else {
			s += appStyle.Render(line) + "\n"
		}
	}

	s += helpStyle.Render("\n↑/↓ navigate • enter install • esc back • q quit")

	return s
}

type keyMap struct {
	Up         key.Binding
	Down       key.Binding
	Enter      key.Binding
	Back       key.Binding
	InstallAll key.Binding
	Quit       key.Binding
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
	InstallAll: key.NewBinding(
		key.WithKeys("i"),
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
