package ui

import (
	"errors"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	// Base styles
	baseStyle = lipgloss.NewStyle().
			PaddingLeft(2)

	// Header section styles
	headerStyle = baseStyle.Copy().
			Foreground(lipgloss.Color("#FF69B4")).
			Bold(true)

	searchStyle = baseStyle.Copy().
			Foreground(lipgloss.Color("#00FFFF"))

	// Item styles
	normalItemStyle = baseStyle.Copy().
			Foreground(lipgloss.Color("#FFFFFF"))

	selectedItemStyle = baseStyle.Copy().
				Background(lipgloss.Color("#304878")).
				Foreground(lipgloss.Color("#FFFFFF"))

	// Footer style
	footerStyle = baseStyle.Copy().
			Foreground(lipgloss.Color("#666666")).
			Italic(true)

	// Divider
	dividerStyle = baseStyle.Copy().
			Foreground(lipgloss.Color("#304878"))
)

type model struct {
	items    []Pair
	cursor   int
	selected string
	search   string
	filtered []Pair
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit
		case "enter":
			if len(m.filtered) > 0 {
				m.selected = m.filtered[m.cursor].Value
				return m, tea.Quit
			}
			return m, nil
		case "up":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down":
			if m.cursor < len(m.filtered)-1 {
				m.cursor++
			}
		case "backspace":
			if len(m.search) > 0 {
				m.search = m.search[:len(m.search)-1]
				m.filterItems()
				m.cursor = 0
			}
		default:
			if len(msg.String()) == 1 {
				m.search += msg.String()
				m.filterItems()
				m.cursor = 0
			}
		}
	}

	return m, nil
}

func (m *model) filterItems() {
	if m.search == "" {
		m.filtered = m.items
		return
	}

	m.filtered = []Pair{}
	searchLower := strings.ToLower(m.search)
	for _, item := range m.items {
		if strings.Contains(strings.ToLower(item.Label), searchLower) {
			m.filtered = append(m.filtered, item)
		}
	}
}

func (m model) View() string {
	var s strings.Builder

	// Header section with consistent spacing
	header := lipgloss.JoinHorizontal(
		lipgloss.Left,
		headerStyle.Render("Select an item"),
		"    ", // spacing between elements
		searchStyle.Render("Search: "+m.search),
	)
	s.WriteString(header + "\n\n")

	// Divider
	s.WriteString(dividerStyle.Render(strings.Repeat("─", 50)) + "\n")

	// Items list with full line highlighting
	if len(m.filtered) == 0 {
		s.WriteString(baseStyle.Render("No matches found"))
	} else {
		for i, item := range m.filtered {
			if i == m.cursor {
				s.WriteString(selectedItemStyle.Render(item.Label))
			} else {
				s.WriteString(normalItemStyle.Render(item.Label))
			}
			s.WriteString("\n")
		}
	}

	// Footer
	s.WriteString("\n")
	s.WriteString(footerStyle.Render("↑/↓ navigate • enter select • esc quit"))

	return s.String()
}

// ShowCLIMenu displays a CLI menu using Bubble Tea and returns the selected value
func ShowCLIMenu(menuType MenuType, items []Pair) (string, error) {
	var err error
	if len(items) == 0 {
		return "", errors.New("no items to show")
	}

	initialModel := model{
		items:    items,
		filtered: items,
		search:   "",
	}

	p := tea.NewProgram(initialModel)
	m, err := p.Run()
	if err != nil {
		return "", err
	}

	return m.(model).selected, nil
}
