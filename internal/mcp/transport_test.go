package mcp

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTransportFactory(t *testing.T) {
	factory := NewTransportFactory()
	assert.NotNil(t, factory)
	assert.IsType(t, &DefaultTransportFactory{}, factory)
}

func TestDefaultTransportFactory_CreateTransport(t *testing.T) {
	factory := NewTransportFactory()

	tests := []struct {
		name         string
		config       ServerConfig
		expectError  bool
		expectedType string
	}{
		{
			name: "stdio transport",
			config: ServerConfig{
				Type:    "stdio",
				Command: "test-command",
				Args:    []string{"--arg1", "--arg2"},
			},
			expectError:  false,
			expectedType: "*mcp.StdioTransport",
		},
		{
			name: "http transport",
			config: ServerConfig{
				Type: "http",
				URL:  "http://localhost:8080",
			},
			expectError:  false,
			expectedType: "*mcp.HTTPTransport",
		},
		{
			name: "sse transport",
			config: ServerConfig{
				Type: "sse",
				URL:  "http://localhost:8080",
			},
			expectError:  false,
			expectedType: "*mcp.SSETransport",
		},
		{
			name: "unsupported transport type",
			config: ServerConfig{
				Type: "websocket",
			},
			expectError:  true,
			expectedType: "",
		},
		{
			name: "empty transport type defaults to stdio",
			config: ServerConfig{
				Command: "test-command",
			},
			expectError:  false,
			expectedType: "*mcp.StdioTransport",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transport, err := factory.CreateTransport(tt.config)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, transport)

				// Check that it's a TransportError
				var transportErr *TransportError
				assert.ErrorAs(t, err, &transportErr)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, transport)

				// Check the transport type
				assert.Equal(t, tt.expectedType, getTypeName(transport))

				// All transports should initially be disconnected
				assert.False(t, transport.IsConnected())
			}
		})
	}
}

func TestTransportError(t *testing.T) {
	tests := []struct {
		name      string
		message   string
		transport string
		cause     error
		expected  string
	}{
		{
			name:      "error without cause",
			message:   "connection failed",
			transport: "http",
			cause:     nil,
			expected:  "connection failed (http)",
		},
		{
			name:      "error with cause",
			message:   "connection failed",
			transport: "http",
			cause:     assert.AnError,
			expected:  "connection failed (http): assert.AnError general error for testing",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewTransportError(tt.message, tt.transport, tt.cause)

			assert.Equal(t, tt.message, err.Message)
			assert.Equal(t, tt.transport, err.Transport)
			assert.Equal(t, tt.cause, err.Cause)
			assert.Equal(t, tt.expected, err.Error())

			// Test Unwrap
			assert.Equal(t, tt.cause, err.Unwrap())
		})
	}
}

func TestDummyTransport(t *testing.T) {
	transport := &DummyTransport{}
	ctx := context.Background()

	// Initially not connected
	assert.False(t, transport.IsConnected())

	// Test Connect
	err := transport.Connect(ctx)
	assert.NoError(t, err)
	assert.True(t, transport.IsConnected())

	// Test SendRequest
	result, err := transport.SendRequest(ctx, "test-method", map[string]interface{}{
		"param1": "value1",
	})
	assert.NoError(t, err)
	assert.NotNil(t, result)

	resultMap, ok := result.(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, "dummy response", resultMap["result"])

	// Test SendNotification
	err = transport.SendNotification(ctx, "test-notification", map[string]interface{}{
		"message": "test",
	})
	assert.NoError(t, err)

	// Test Close
	err = transport.Close()
	assert.NoError(t, err)
	assert.False(t, transport.IsConnected())
}

func TestTransportInterfaceCompliance(t *testing.T) {
	// Test that all transport implementations satisfy the Transport interface
	var _ Transport = &DummyTransport{}

	// Test HTTP transport (if we can create one)
	httpConfig := ServerConfig{
		Type: "http",
		URL:  "http://localhost:8080",
	}
	httpTransport, err := NewHTTPTransport(httpConfig)
	require.NoError(t, err)
	var _ Transport = httpTransport

	// Test SSE transport (if we can create one)
	sseConfig := ServerConfig{
		Type: "sse",
		URL:  "http://localhost:8080",
	}
	sseTransport, err := NewSSETransport(sseConfig)
	require.NoError(t, err)
	var _ Transport = sseTransport

	// Test Stdio transport (if we can create one)
	stdioConfig := ServerConfig{
		Type:    "stdio",
		Command: "echo",
		Args:    []string{"hello"},
	}
	stdioTransport, err := NewStdioTransport(stdioConfig)
	require.NoError(t, err)
	var _ Transport = stdioTransport
}

func TestTransportMethodsSignatures(t *testing.T) {
	// This test ensures all transport types have the correct method signatures
	// by calling them with the interface

	ctx := context.Background()
	dummy := &DummyTransport{}
	var transport Transport = dummy

	// Test all interface methods
	err := transport.Connect(ctx)
	assert.NoError(t, err)

	connected := transport.IsConnected()
	assert.True(t, connected)

	result, err := transport.SendRequest(ctx, "test", nil)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	err = transport.SendNotification(ctx, "test", nil)
	assert.NoError(t, err)

	err = transport.Close()
	assert.NoError(t, err)
}

// Helper function to get the type name of an interface value
func getTypeName(v interface{}) string {
	if v == nil {
		return "<nil>"
	}

	switch v.(type) {
	case *StdioTransport:
		return "*mcp.StdioTransport"
	case *HTTPTransport:
		return "*mcp.HTTPTransport"
	case *SSETransport:
		return "*mcp.SSETransport"
	case *DummyTransport:
		return "*mcp.DummyTransport"
	default:
		return "unknown"
	}
}
