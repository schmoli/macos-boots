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
	layoutHorizontal layoutMode = iota // side-by-side
	layoutVertical                     // top-bottom
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

func installAppCmd(name string) tea.Cmd {
	return func() tea.Msg {
		cmd := exec.Command("/opt/homebrew/bin/brew", "install", name)
		return runCmdWithOutput(cmd, name, true)
	}
}

func removeAppCmd(name string) tea.Cmd {
	return func() tea.Msg {
		cmd := exec.Command("/opt/homebrew/bin/brew", "uninstall", name)
		return runCmdWithOutput(cmd, name, false)
	}
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
	return nil
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
		maxVisible := m.getLogHeight()
		if len(m.logLines) > maxVisible {
			m.logScroll = len(m.logLines) - maxVisible
		}
		return m, nil
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

func (m model) handleInstallComplete(msg installCompleteMsg) (tea.Model, tea.Cmd) {
	cat := &m.categories[m.cursor]

	for i := range cat.apps {
		if cat.apps[i].name == msg.name {
			cat.apps[i].installed = msg.success
			cat.apps[i].selected = false
			break
		}
	}

	m.progressIdx++

	if m.progressIdx < len(m.progressApps) {
		nextApp := m.progressApps[m.progressIdx]
		m.progressMsg = fmt.Sprintf("Installing %s... (%d/%d)", nextApp, m.progressIdx+1, len(m.progressApps))
		return m, installAppCmd(nextApp)
	}

	m.state = stateIdle
	m.progressMsg = ""
	m.progressApps = nil
	return m, nil
}

func (m model) handleRemoveComplete(msg removeCompleteMsg) (tea.Model, tea.Cmd) {
	cat := &m.categories[m.cursor]

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

	if m.progressIdx < len(m.progressApps) {
		nextApp := m.progressApps[m.progressIdx]
		m.progressMsg = fmt.Sprintf("Removing %s... (%d/%d)", nextApp, m.progressIdx+1, len(m.progressApps))
		return m, removeAppCmd(nextApp)
	}

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
	case key.Matches(msg, keys.Layout):
		if m.layout == layoutHorizontal {
			m.layout = layoutVertical
		} else {
			m.layout = layoutHorizontal
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
		m.categories[m.cursor].apps[m.appCursor].selected = !m.categories[m.cursor].apps[m.appCursor].selected
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
			m.categories[m.cursor].apps[i].selected = anyUnselected
		}
	case key.Matches(msg, keys.Install):
		selected := cat.getSelectedNames()
		if len(selected) > 0 {
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
	mainPane = m.renderPane("", content, mainWidth, mainHeight)

	// Render log pane
	logContent := m.viewLog(logHeight)
	logPane = m.renderPane("Log", logContent, logWidth, logHeight)

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

func (m model) renderPane(title string, content string, width, height int) string {
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
		if cat.selectedCount() > 0 {
			parts = append(parts, fmt.Sprintf("i inst(%d)", cat.selectedCount()))
			parts = append(parts, fmt.Sprintf("r rem(%d)", cat.selectedCount()))
		}
		parts = append(parts, "esc back")
	} else {
		parts = append(parts, "enter sel")
	}

	layoutName := "horiz"
	if m.layout == layoutVertical {
		layoutName = "vert"
	}
	parts = append(parts, fmt.Sprintf("tab layout(%s)", layoutName))
	parts = append(parts, "q quit")

	return footerStyle.Render(strings.Join(parts, " • "))
}

func (m model) viewMainContent() string {
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

	if m.progressMsg != "" {
		s += "\n" + progressStyle.Render(m.progressMsg)
	}

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
	Layout    key.Binding
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
	Layout: key.NewBinding(
		key.WithKeys("tab"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
	),
}

func Run() error {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	program = p
	_, err := p.Run()
	return err
}
