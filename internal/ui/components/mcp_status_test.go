package components

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/common-creation/coda/internal/mcp"
	"github.com/stretchr/testify/assert"
)

func TestNewMCPStatusDisplay(t *testing.T) {
	display := NewMCPStatusDisplay()
	assert.NotNil(t, display)
	assert.Equal(t, 80, display.width)
	assert.Equal(t, 24, display.height)
	assert.NotNil(t, display.statuses)
	assert.Len(t, display.statuses, 0)
}

func TestMCPStatusDisplay_SetSize(t *testing.T) {
	display := NewMCPStatusDisplay()

	display.SetSize(120, 30)
	assert.Equal(t, 120, display.width)
	assert.Equal(t, 30, display.height)
}

func TestMCPStatusDisplay_UpdateStatuses(t *testing.T) {
	display := NewMCPStatusDisplay()

	statuses := map[string]mcp.ServerStatus{
		"server1": {
			Name:      "server1",
			State:     mcp.StateRunning,
			StartedAt: time.Now(),
			Transport: "stdio",
		},
		"server2": {
			Name:      "server2",
			State:     mcp.StateError,
			Error:     errors.New("connection failed"),
			Transport: "http",
		},
	}

	display.UpdateStatuses(statuses)
	assert.Len(t, display.statuses, 2)
	assert.Equal(t, statuses, display.statuses)
}

func TestMCPStatusDisplay_GetRunningCount(t *testing.T) {
	display := NewMCPStatusDisplay()

	statuses := map[string]mcp.ServerStatus{
		"server1": {State: mcp.StateRunning},
		"server2": {State: mcp.StateError},
		"server3": {State: mcp.StateRunning},
		"server4": {State: mcp.StateStopped},
	}

	display.UpdateStatuses(statuses)
	assert.Equal(t, 2, display.GetRunningCount())
}

func TestMCPStatusDisplay_GetErrorCount(t *testing.T) {
	display := NewMCPStatusDisplay()

	statuses := map[string]mcp.ServerStatus{
		"server1": {State: mcp.StateRunning},
		"server2": {State: mcp.StateError},
		"server3": {State: mcp.StateError},
		"server4": {State: mcp.StateStopped},
	}

	display.UpdateStatuses(statuses)
	assert.Equal(t, 2, display.GetErrorCount())
}

func TestMCPStatusDisplay_GetSummary(t *testing.T) {
	display := NewMCPStatusDisplay()

	tests := []struct {
		name     string
		statuses map[string]mcp.ServerStatus
		expected string
	}{
		{
			name:     "no servers",
			statuses: map[string]mcp.ServerStatus{},
			expected: "No servers",
		},
		{
			name: "all running",
			statuses: map[string]mcp.ServerStatus{
				"server1": {State: mcp.StateRunning},
				"server2": {State: mcp.StateRunning},
			},
			expected: "2/2 running",
		},
		{
			name: "some errors",
			statuses: map[string]mcp.ServerStatus{
				"server1": {State: mcp.StateRunning},
				"server2": {State: mcp.StateError},
				"server3": {State: mcp.StateStopped},
			},
			expected: "1/3 running (1 errors)",
		},
		{
			name: "mixed states",
			statuses: map[string]mcp.ServerStatus{
				"server1": {State: mcp.StateRunning},
				"server2": {State: mcp.StateRunning},
				"server3": {State: mcp.StateStarting},
				"server4": {State: mcp.StateStopped},
			},
			expected: "2/4 running",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			display.UpdateStatuses(tt.statuses)
			summary := display.GetSummary()
			assert.Equal(t, tt.expected, summary)
		})
	}
}

func TestMCPStatusDisplay_formatDuration(t *testing.T) {
	display := NewMCPStatusDisplay()

	tests := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		{
			name:     "seconds",
			duration: 30 * time.Second,
			expected: "30s",
		},
		{
			name:     "minutes",
			duration: 5 * time.Minute,
			expected: "5m",
		},
		{
			name:     "hours",
			duration: 2*time.Hour + 30*time.Minute,
			expected: "2.5h",
		},
		{
			name:     "days",
			duration: 25 * time.Hour,
			expected: "1.0d",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := display.formatDuration(tt.duration)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMCPStatusDisplay_hasCapabilities(t *testing.T) {
	display := NewMCPStatusDisplay()

	tests := []struct {
		name         string
		capabilities mcp.ServerCapabilities
		expected     bool
	}{
		{
			name:         "no capabilities",
			capabilities: mcp.ServerCapabilities{},
			expected:     false,
		},
		{
			name: "tools capability",
			capabilities: mcp.ServerCapabilities{
				Tools: &mcp.ToolsCapability{},
			},
			expected: true,
		},
		{
			name: "resources capability",
			capabilities: mcp.ServerCapabilities{
				Resources: &mcp.ResourcesCapability{},
			},
			expected: true,
		},
		{
			name: "prompts capability",
			capabilities: mcp.ServerCapabilities{
				Prompts: &mcp.PromptsCapability{},
			},
			expected: true,
		},
		{
			name: "all capabilities",
			capabilities: mcp.ServerCapabilities{
				Tools:     &mcp.ToolsCapability{},
				Resources: &mcp.ResourcesCapability{},
				Prompts:   &mcp.PromptsCapability{},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := display.hasCapabilities(tt.capabilities)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMCPStatusDisplay_formatCapabilities(t *testing.T) {
	display := NewMCPStatusDisplay()

	tests := []struct {
		name         string
		capabilities mcp.ServerCapabilities
		expected     string
	}{
		{
			name:         "no capabilities",
			capabilities: mcp.ServerCapabilities{},
			expected:     "none",
		},
		{
			name: "tools only",
			capabilities: mcp.ServerCapabilities{
				Tools: &mcp.ToolsCapability{},
			},
			expected: "tools",
		},
		{
			name: "tools and resources",
			capabilities: mcp.ServerCapabilities{
				Tools:     &mcp.ToolsCapability{},
				Resources: &mcp.ResourcesCapability{},
			},
			expected: "tools, resources",
		},
		{
			name: "all capabilities",
			capabilities: mcp.ServerCapabilities{
				Tools:     &mcp.ToolsCapability{},
				Resources: &mcp.ResourcesCapability{},
				Prompts:   &mcp.PromptsCapability{},
			},
			expected: "tools, resources, prompts",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := display.formatCapabilities(tt.capabilities)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMCPStatusDisplay_Render(t *testing.T) {
	display := NewMCPStatusDisplay()

	t.Run("empty state", func(t *testing.T) {
		result := display.Render()
		assert.Contains(t, result, "No MCP servers configured or loaded")
	})

	t.Run("with servers", func(t *testing.T) {
		now := time.Now()
		statuses := map[string]mcp.ServerStatus{
			"test-server": {
				Name:      "test-server",
				State:     mcp.StateRunning,
				StartedAt: now,
				Transport: "stdio",
				Capabilities: mcp.ServerCapabilities{
					Tools: &mcp.ToolsCapability{},
				},
			},
		}

		display.UpdateStatuses(statuses)
		result := display.Render()

		assert.Contains(t, result, "MCP Server Status")
		assert.Contains(t, result, "Total servers: 1")
		assert.Contains(t, result, "test-server")
		assert.Contains(t, result, "[Running]")
		assert.Contains(t, result, "Transport: stdio")
		assert.Contains(t, result, "Started:")
		assert.Contains(t, result, "Capabilities: tools")
		assert.Contains(t, result, "Press ESC to close")
	})

	t.Run("with error server", func(t *testing.T) {
		statuses := map[string]mcp.ServerStatus{
			"error-server": {
				Name:      "error-server",
				State:     mcp.StateError,
				Error:     errors.New("connection timeout"),
				Transport: "http",
			},
		}

		display.UpdateStatuses(statuses)
		result := display.Render()

		assert.Contains(t, result, "error-server")
		assert.Contains(t, result, "[Error]")
		assert.Contains(t, result, "Error: connection timeout")
		assert.Contains(t, result, "Transport: http")
	})
}

func TestMCPStatusDisplay_StateStyles(t *testing.T) {
	display := NewMCPStatusDisplay()

	// Test that state styles are different for different states
	runningStyle := display.getStateStyle(mcp.StateRunning)
	errorStyle := display.getStateStyle(mcp.StateError)
	stoppedStyle := display.getStateStyle(mcp.StateStopped)
	startingStyle := display.getStateStyle(mcp.StateStarting)

	// These should all be different (we can't easily test the actual colors
	// but we can ensure they're distinct style objects)
	assert.NotEqual(t, runningStyle, errorStyle)
	assert.NotEqual(t, runningStyle, stoppedStyle)
	assert.NotEqual(t, errorStyle, startingStyle)
}

func TestMCPStatusDisplay_RenderStructure(t *testing.T) {
	display := NewMCPStatusDisplay()
	display.SetSize(100, 30)

	statuses := map[string]mcp.ServerStatus{
		"server1": {
			Name:      "server1",
			State:     mcp.StateRunning,
			StartedAt: time.Now(),
			Transport: "stdio",
		},
	}

	display.UpdateStatuses(statuses)
	result := display.Render()

	// Check that the rendered output has the expected structure
	lines := strings.Split(result, "\n")
	assert.Greater(t, len(lines), 5, "Should have multiple lines of output")

	// Should contain borders (from lipgloss)
	found := false
	for _, line := range lines {
		if strings.Contains(line, "─") || strings.Contains(line, "│") {
			found = true
			break
		}
	}
	assert.True(t, found, "Should contain border characters")
}
