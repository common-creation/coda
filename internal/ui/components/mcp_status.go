package components

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/common-creation/coda/internal/mcp"
	"github.com/common-creation/coda/internal/styles"
)

// MCPStatusDisplay handles the display of MCP server status information
type MCPStatusDisplay struct {
	statuses map[string]mcp.ServerStatus
	width    int
	height   int
	styles   styles.Styles
}

// NewMCPStatusDisplay creates a new MCP status display component
func NewMCPStatusDisplay() *MCPStatusDisplay {
	theme := styles.GetTheme("default")
	return &MCPStatusDisplay{
		statuses: make(map[string]mcp.ServerStatus),
		width:    80,
		height:   24,
		styles:   theme.GetStyles(),
	}
}

// SetSize sets the display dimensions
func (m *MCPStatusDisplay) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// UpdateStatuses updates the server statuses to display
func (m *MCPStatusDisplay) UpdateStatuses(statuses map[string]mcp.ServerStatus) {
	m.statuses = statuses
}

// Render renders the MCP status display
func (m *MCPStatusDisplay) Render() string {
	if len(m.statuses) == 0 {
		return m.renderEmpty()
	}

	var sections []string

	// Header
	sections = append(sections, m.renderHeader())
	sections = append(sections, "")

	// Server statuses
	for name, status := range m.statuses {
		sections = append(sections, m.renderServerStatus(name, status))
		sections = append(sections, "")
	}

	// Footer with help
	sections = append(sections, m.renderFooter())

	content := strings.Join(sections, "\n")

	// Apply container styling
	containerStyle := lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Padding(1).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.styles.Colors.Border)

	return containerStyle.Render(content)
}

// renderEmpty renders the empty state
func (m *MCPStatusDisplay) renderEmpty() string {
	emptyStyle := lipgloss.NewStyle().
		Width(m.width-4).
		Height(m.height-4).
		Align(lipgloss.Center, lipgloss.Center).
		Foreground(m.styles.Colors.Muted)

	return emptyStyle.Render("No MCP servers configured or loaded")
}

// renderHeader renders the header section
func (m *MCPStatusDisplay) renderHeader() string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(m.styles.Colors.Primary).
		Align(lipgloss.Center).
		Width(m.width - 4)

	title := "MCP Server Status"
	subtitle := fmt.Sprintf("Total servers: %d", len(m.statuses))

	return titleStyle.Render(title) + "\n" +
		lipgloss.NewStyle().
			Foreground(m.styles.Colors.Muted).
			Align(lipgloss.Center).
			Width(m.width-4).
			Render(subtitle)
}

// renderServerStatus renders the status for a single server
func (m *MCPStatusDisplay) renderServerStatus(name string, status mcp.ServerStatus) string {
	var lines []string

	// Server name and state
	nameStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(m.styles.Colors.Foreground)

	stateStyle := m.getStateStyle(status.State)

	headerLine := fmt.Sprintf("%s %s",
		nameStyle.Render(name),
		stateStyle.Render(fmt.Sprintf("[%s]", status.State.String())),
	)
	lines = append(lines, headerLine)

	// Transport information
	if status.Transport != "" {
		transportLine := fmt.Sprintf("  Transport: %s", status.Transport)
		lines = append(lines, lipgloss.NewStyle().
			Foreground(m.styles.Colors.Muted).
			Render(transportLine))
	}

	// Start time (if started)
	if !status.StartedAt.IsZero() {
		duration := time.Since(status.StartedAt)
		timeLine := fmt.Sprintf("  Started: %s (uptime: %s)",
			status.StartedAt.Format("15:04:05"),
			m.formatDuration(duration))
		lines = append(lines, lipgloss.NewStyle().
			Foreground(m.styles.Colors.Muted).
			Render(timeLine))
	}

	// Error information
	if status.Error != nil {
		errorLine := fmt.Sprintf("  Error: %s", status.Error.Error())
		lines = append(lines, lipgloss.NewStyle().
			Foreground(m.styles.Colors.Error).
			Render(errorLine))
	}

	// Capabilities
	if m.hasCapabilities(status.Capabilities) {
		capLine := fmt.Sprintf("  Capabilities: %s", m.formatCapabilities(status.Capabilities))
		lines = append(lines, lipgloss.NewStyle().
			Foreground(m.styles.Colors.Info).
			Render(capLine))
	}

	return strings.Join(lines, "\n")
}

// renderFooter renders the footer with help information
func (m *MCPStatusDisplay) renderFooter() string {
	footerStyle := lipgloss.NewStyle().
		Foreground(m.styles.Colors.Muted).
		Align(lipgloss.Center).
		Width(m.width - 4)

	return footerStyle.Render("Press ESC to close")
}

// getStateStyle returns the appropriate style for a server state
func (m *MCPStatusDisplay) getStateStyle(state mcp.State) lipgloss.Style {
	switch state {
	case mcp.StateRunning:
		return lipgloss.NewStyle().
			Foreground(m.styles.Colors.Success).
			Bold(true)
	case mcp.StateStarting:
		return lipgloss.NewStyle().
			Foreground(m.styles.Colors.Warning).
			Bold(true)
	case mcp.StateError:
		return lipgloss.NewStyle().
			Foreground(m.styles.Colors.Error).
			Bold(true)
	case mcp.StateStopped:
		return lipgloss.NewStyle().
			Foreground(m.styles.Colors.Muted).
			Bold(true)
	default:
		return lipgloss.NewStyle().
			Foreground(m.styles.Colors.Foreground).
			Bold(true)
	}
}

// formatDuration formats a duration for display
func (m *MCPStatusDisplay) formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.0fs", d.Seconds())
	} else if d < time.Hour {
		return fmt.Sprintf("%.0fm", d.Minutes())
	} else if d < 24*time.Hour {
		return fmt.Sprintf("%.1fh", d.Hours())
	} else {
		return fmt.Sprintf("%.1fd", d.Hours()/24)
	}
}

// hasCapabilities checks if the server has any capabilities
func (m *MCPStatusDisplay) hasCapabilities(caps mcp.ServerCapabilities) bool {
	return caps.Tools != nil || caps.Resources != nil || caps.Prompts != nil
}

// formatCapabilities formats capabilities for display
func (m *MCPStatusDisplay) formatCapabilities(caps mcp.ServerCapabilities) string {
	var capList []string

	if caps.Tools != nil {
		capList = append(capList, "tools")
	}
	if caps.Resources != nil {
		capList = append(capList, "resources")
	}
	if caps.Prompts != nil {
		capList = append(capList, "prompts")
	}

	if len(capList) == 0 {
		return "none"
	}

	return strings.Join(capList, ", ")
}

// GetRunningCount returns the number of running servers
func (m *MCPStatusDisplay) GetRunningCount() int {
	count := 0
	for _, status := range m.statuses {
		if status.State == mcp.StateRunning {
			count++
		}
	}
	return count
}

// GetErrorCount returns the number of servers in error state
func (m *MCPStatusDisplay) GetErrorCount() int {
	count := 0
	for _, status := range m.statuses {
		if status.State == mcp.StateError {
			count++
		}
	}
	return count
}

// GetSummary returns a brief summary of server statuses
func (m *MCPStatusDisplay) GetSummary() string {
	total := len(m.statuses)
	running := m.GetRunningCount()
	errors := m.GetErrorCount()

	if total == 0 {
		return "No servers"
	}

	if errors > 0 {
		return fmt.Sprintf("%d/%d running (%d errors)", running, total, errors)
	}

	return fmt.Sprintf("%d/%d running", running, total)
}
