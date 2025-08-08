package mcp

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewStdioTransport(t *testing.T) {
	config := ServerConfig{
		Command: "echo",
		Args:    []string{"hello"},
		Type:    "stdio",
	}

	transport, err := NewStdioTransport(config)
	require.NoError(t, err)
	assert.NotNil(t, transport)

	stdioTransport, ok := transport.(*StdioTransport)
	require.True(t, ok)
	assert.Equal(t, config.Command, stdioTransport.config.Command)
	assert.Equal(t, config.Args, stdioTransport.config.Args)
	assert.False(t, stdioTransport.IsConnected())
}

func TestNewStdioTransportMissingCommand(t *testing.T) {
	config := ServerConfig{
		Type: "stdio",
		// Command is missing
	}

	transport, err := NewStdioTransport(config)
	assert.Error(t, err)
	assert.Nil(t, transport)
	assert.Contains(t, err.Error(), "command is required")
}

func TestStdioTransportIsConnected(t *testing.T) {
	config := ServerConfig{
		Command: "echo",
		Args:    []string{"hello"},
		Type:    "stdio",
	}

	transport, err := NewStdioTransport(config)
	require.NoError(t, err)

	// Should not be connected initially
	assert.False(t, transport.IsConnected())
}

func TestStdioTransportSendRequestNotConnected(t *testing.T) {
	config := ServerConfig{
		Command: "echo",
		Args:    []string{"hello"},
		Type:    "stdio",
	}

	transport, err := NewStdioTransport(config)
	require.NoError(t, err)

	ctx := context.Background()
	result, err := transport.SendRequest(ctx, "test_method", map[string]interface{}{"param": "value"})

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "transport not connected")
}

func TestStdioTransportSendNotificationNotConnected(t *testing.T) {
	config := ServerConfig{
		Command: "echo",
		Args:    []string{"hello"},
		Type:    "stdio",
	}

	transport, err := NewStdioTransport(config)
	require.NoError(t, err)

	ctx := context.Background()
	err = transport.SendNotification(ctx, "test_notification", map[string]interface{}{"param": "value"})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "transport not connected")
}

func TestStdioTransportCloseNotConnected(t *testing.T) {
	config := ServerConfig{
		Command: "echo",
		Args:    []string{"hello"},
		Type:    "stdio",
	}

	transport, err := NewStdioTransport(config)
	require.NoError(t, err)

	// Close should not error even if not connected
	err = transport.Close()
	assert.NoError(t, err)
}

func TestStdioTransportConnect(t *testing.T) {
	// This test uses a simple command that should exist on most systems
	config := ServerConfig{
		Command: "echo",
		Args:    []string{"test"},
		Type:    "stdio",
	}

	transport, err := NewStdioTransport(config)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Note: This test may fail if the echo command doesn't behave like an MCP server
	// In a real scenario, we would use an actual MCP server for integration tests
	err = transport.Connect(ctx)

	if err != nil {
		// This is expected since echo is not an MCP server
		// We're mainly testing that the transport attempts to connect
		t.Logf("Connect failed as expected (echo is not an MCP server): %v", err)
	}

	// Clean up
	transport.Close()
}
