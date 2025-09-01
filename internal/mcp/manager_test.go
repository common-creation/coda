package mcp

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/charmbracelet/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewManager(t *testing.T) {
	manager := NewManager(nil)
	assert.NotNil(t, manager)
	assert.NotNil(t, manager.servers)
	assert.NotNil(t, manager.logger)
}

func TestManagerLoadConfig(t *testing.T) {
	// Create temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "mcp.json")

	configContent := `{
		"mcpServers": {
			"test-server": {
				"command": "echo",
				"args": ["hello"],
				"type": "stdio"
			}
		}
	}`

	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	manager := NewManager(log.New(os.Stderr))

	// Test loading configuration
	err = manager.LoadConfig([]string{configPath})
	require.NoError(t, err)

	assert.NotNil(t, manager.config)
	assert.Len(t, manager.config.Servers, 1)
	assert.Contains(t, manager.config.Servers, "test-server")
}

func TestManagerLoadConfigNotFound(t *testing.T) {
	manager := NewManager(log.New(os.Stderr))

	err := manager.LoadConfig([]string{"/nonexistent/path.json"})
	assert.Error(t, err)
}

func TestManagerServerLifecycle(t *testing.T) {
	t.Skip("Skipping test that requires real MCP server executable")
	// Create temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "mcp.json")

	configContent := `{
		"mcpServers": {
			"test-server": {
				"command": "echo",
				"args": ["hello"],
				"type": "stdio"
			}
		}
	}`

	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	manager := NewManager(log.New(os.Stderr))

	// Load configuration
	err = manager.LoadConfig([]string{configPath})
	require.NoError(t, err)

	// Test server status before starting
	status := manager.GetServerStatus("test-server")
	assert.Equal(t, StateStopped, status.State)

	// Start server
	err = manager.StartServer("test-server")
	assert.NoError(t, err)

	// Give it a moment to start
	time.Sleep(100 * time.Millisecond)

	// Check status after starting
	status = manager.GetServerStatus("test-server")
	assert.True(t, status.State == StateStarting || status.State == StateRunning)

	// Stop server
	err = manager.StopServer("test-server")
	assert.NoError(t, err)

	// Check status after stopping
	status = manager.GetServerStatus("test-server")
	assert.Equal(t, StateStopped, status.State)
}

func TestManagerGetAllStatuses(t *testing.T) {
	// Create temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "mcp.json")

	configContent := `{
		"mcpServers": {
			"server1": {
				"command": "echo",
				"args": ["hello"],
				"type": "stdio"
			},
			"server2": {
				"command": "echo",
				"args": ["world"],
				"type": "stdio"
			}
		}
	}`

	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	manager := NewManager(log.New(os.Stderr))

	// Load configuration
	err = manager.LoadConfig([]string{configPath})
	require.NoError(t, err)

	// Get all statuses
	statuses := manager.GetAllStatuses()
	assert.Len(t, statuses, 2)
	assert.Contains(t, statuses, "server1")
	assert.Contains(t, statuses, "server2")

	// Both should be stopped initially
	assert.Equal(t, StateStopped, statuses["server1"].State)
	assert.Equal(t, StateStopped, statuses["server2"].State)
}

func TestManagerStartNonExistentServer(t *testing.T) {
	manager := NewManager(log.New(os.Stderr))

	// Try to start server without configuration
	err := manager.StartServer("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no configuration loaded")

	// Load config with existing server but try to start different one
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "mcp.json")

	configContent := `{
		"mcpServers": {
			"existing-server": {
				"command": "echo",
				"args": ["test"],
				"type": "stdio"
			}
		}
	}`
	err = os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	err = manager.LoadConfig([]string{configPath})
	require.NoError(t, err)

	// Try to start non-existent server
	err = manager.StartServer("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "server nonexistent not found")
}
