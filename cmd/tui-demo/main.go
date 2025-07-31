package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Styles
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#7D56F4")).
			PaddingTop(1).
			PaddingBottom(1).
			PaddingLeft(4).
			PaddingRight(4)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#999999"))
)

type model struct {
	messages []string
	input    string
}

func initialModel() model {
	return model{
		messages: []string{"Welcome to CODA TUI Demo!"},
		input:    "",
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "enter":
			if m.input != "" {
				m.messages = append(m.messages, fmt.Sprintf("> %s", m.input))
				m.messages = append(m.messages, "AI: This is a demo response. The real AI integration is pending.")
				m.input = ""
			}
		case "backspace":
			if len(m.input) > 0 {
				m.input = m.input[:len(m.input)-1]
			}
		default:
			m.input += msg.String()
		}
	}
	return m, nil
}

func (m model) View() string {
	s := titleStyle.Render("CODA - Coding Agent (TUI Demo)") + "\n\n"
	
	// Show messages
	for _, msg := range m.messages {
		s += msg + "\n"
	}
	
	// Input area
	s += "\n> " + m.input
	
	// Help
	s += "\n\n" + helpStyle.Render("Press 'q' or Ctrl+C to quit, Enter to send message")
	
	return s
}

func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if err := p.Start(); err != nil {
		fmt.Printf("Error: %v", err)
		os.Exit(1)
	}
}