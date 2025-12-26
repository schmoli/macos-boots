package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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
)

type category struct {
	name      string
	installed int
	total     int
}

type model struct {
	categories []category
	cursor     int
	quitting   bool
}

func initialModel() model {
	return model{
		categories: []category{
			{name: "CLI Tools", installed: 0, total: 5},
			{name: "Desktop Apps", installed: 0, total: 8},
			{name: "App Store", installed: 0, total: 3},
			{name: "Configs", installed: 0, total: 4},
		},
		cursor: 0,
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
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
			// TODO: drill into category
		case key.Matches(msg, keys.InstallAll):
			// TODO: install all auto-tier apps
		}
	}
	return m, nil
}

func (m model) View() string {
	if m.quitting {
		return ""
	}

	s := titleStyle.Render("macos-setup") + "\n\n"

	for i, cat := range m.categories {
		status := fmt.Sprintf("[%d/%d]", cat.installed, cat.total)
		indicator := "○"
		if cat.installed > 0 {
			indicator = "●"
		}
		if cat.installed == cat.total {
			indicator = "✓"
		}

		line := fmt.Sprintf("%s %-20s %s  ▸", indicator, cat.name, statusStyle.Render(status))

		if i == m.cursor {
			s += selectedStyle.Render(line) + "\n"
		} else {
			s += categoryStyle.Render(line) + "\n"
		}
	}

	s += helpStyle.Render("\n↑/↓ navigate • enter select • i install all • q quit")

	return s
}

type keyMap struct {
	Up         key.Binding
	Down       key.Binding
	Enter      key.Binding
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
