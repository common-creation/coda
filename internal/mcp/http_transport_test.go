package mcp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewHTTPTransport(t *testing.T) {
	tests := []struct {
		name        string
		config      ServerConfig
		expectError bool
	}{
		{
			name: "valid config",
			config: ServerConfig{
				URL:  "http://localhost:8080",
				Type: "http",
			},
			expectError: false,
		},
		{
			name: "missing URL",
			config: ServerConfig{
				Type: "http",
			},
			expectError: true,
		},
		{
			name: "with headers",
			config: ServerConfig{
				URL:  "http://localhost:8080",
				Type: "http",
				Headers: map[string]string{
					"Authorization": "Bearer token",
					"Custom-Header": "value",
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transport, err := NewHTTPTransport(tt.config)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, transport)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, transport)

				httpTransport := transport.(*HTTPTransport)
				assert.Equal(t, tt.config.URL, httpTransport.baseURL)
				assert.False(t, httpTransport.IsConnected())
			}
		})
	}
}

func TestHTTPTransport_Connect(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/initialize" {
			w.Header().Set("Content-Type", "application/json")
			response := map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      "test-id",
				"result": map[string]interface{}{
					"protocolVersion": "2024-11-05",
					"capabilities":    map[string]interface{}{},
					"serverInfo": map[string]interface{}{
						"name":    "test-server",
						"version": "1.0.0",
					},
				},
			}
			json.NewEncoder(w).Encode(response)
		} else {
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	config := ServerConfig{
		URL:  server.URL,
		Type: "http",
	}

	transport, err := NewHTTPTransport(config)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Test connection
	err = transport.Connect(ctx)
	assert.NoError(t, err)
	assert.True(t, transport.IsConnected())

	// Test close
	err = transport.Close()
	assert.NoError(t, err)
	assert.False(t, transport.IsConnected())
}

func TestHTTPTransport_SendRequest(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/initialize":
			response := map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      "test-id",
				"result":  map[string]interface{}{"status": "initialized"},
			}
			json.NewEncoder(w).Encode(response)
		case "/tools/list":
			response := map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      "test-id",
				"result": map[string]interface{}{
					"tools": []interface{}{
						map[string]interface{}{
							"name":        "test-tool",
							"description": "A test tool",
						},
					},
				},
			}
			json.NewEncoder(w).Encode(response)
		case "/error":
			response := map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      "test-id",
				"error": map[string]interface{}{
					"code":    -32000,
					"message": "Test error",
				},
			}
			json.NewEncoder(w).Encode(response)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	config := ServerConfig{
		URL:  server.URL,
		Type: "http",
	}

	transport, err := NewHTTPTransport(config)
	require.NoError(t, err)

	ctx := context.Background()

	// Connect first
	err = transport.Connect(ctx)
	require.NoError(t, err)

	t.Run("successful request", func(t *testing.T) {
		result, err := transport.SendRequest(ctx, "tools/list", nil)
		assert.NoError(t, err)
		assert.NotNil(t, result)

		resultMap, ok := result.(map[string]interface{})
		assert.True(t, ok)
		assert.Contains(t, resultMap, "tools")
	})

	t.Run("request with error response", func(t *testing.T) {
		result, err := transport.SendRequest(ctx, "error", nil)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "Test error")
	})

	t.Run("request when not connected", func(t *testing.T) {
		err := transport.Close()
		require.NoError(t, err)

		result, err := transport.SendRequest(ctx, "tools/list", nil)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "not connected")
	})
}

func TestHTTPTransport_SendNotification(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/initialize" {
			w.Header().Set("Content-Type", "application/json")
			response := map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      "test-id",
				"result":  map[string]interface{}{"status": "initialized"},
			}
			json.NewEncoder(w).Encode(response)
		} else if r.URL.Path == "/notifications/test" {
			w.Header().Set("Content-Type", "application/json")
			// Notifications typically don't return a response body, but we need valid JSON
			w.Write([]byte("{}"))
		} else {
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	config := ServerConfig{
		URL:  server.URL,
		Type: "http",
	}

	transport, err := NewHTTPTransport(config)
	require.NoError(t, err)

	ctx := context.Background()

	// Connect first
	err = transport.Connect(ctx)
	require.NoError(t, err)

	// Test notification
	err = transport.SendNotification(ctx, "notifications/test", map[string]interface{}{
		"message": "test notification",
	})
	assert.NoError(t, err)
}

func TestHTTPTransport_SetTimeout(t *testing.T) {
	config := ServerConfig{
		URL:  "http://localhost:8080",
		Type: "http",
	}

	transport, err := NewHTTPTransport(config)
	require.NoError(t, err)

	httpTransport := transport.(*HTTPTransport)

	// Test default timeout
	assert.Equal(t, 30*time.Second, httpTransport.client.Timeout)

	// Set custom timeout
	httpTransport.SetTimeout(10 * time.Second)
	assert.Equal(t, 10*time.Second, httpTransport.client.Timeout)
}

func TestHTTPTransport_SetHeaders(t *testing.T) {
	config := ServerConfig{
		URL:  "http://localhost:8080",
		Type: "http",
	}

	transport, err := NewHTTPTransport(config)
	require.NoError(t, err)

	httpTransport := transport.(*HTTPTransport)

	// Test initial headers
	assert.Equal(t, "application/json", httpTransport.headers["Content-Type"])
	assert.Equal(t, "application/json", httpTransport.headers["Accept"])

	// Set custom headers
	customHeaders := map[string]string{
		"Authorization": "Bearer token",
		"X-Custom":      "value",
	}
	httpTransport.SetHeaders(customHeaders)

	assert.Equal(t, "Bearer token", httpTransport.headers["Authorization"])
	assert.Equal(t, "value", httpTransport.headers["X-Custom"])
	// Original headers should still be there
	assert.Equal(t, "application/json", httpTransport.headers["Content-Type"])
}

func TestHTTPTransport_GetStats(t *testing.T) {
	config := ServerConfig{
		URL:  "http://localhost:8080",
		Type: "http",
	}

	transport, err := NewHTTPTransport(config)
	require.NoError(t, err)

	httpTransport := transport.(*HTTPTransport)
	stats := httpTransport.GetStats()

	assert.Contains(t, stats, "connected")
	assert.Contains(t, stats, "url")
	assert.Contains(t, stats, "timeout")

	assert.Equal(t, false, stats["connected"])
	assert.Equal(t, "http://localhost:8080", stats["url"])
	assert.Equal(t, "30s", stats["timeout"])
}

func TestHTTPTransport_ConnectionFailure(t *testing.T) {
	config := ServerConfig{
		URL:  "http://localhost:9999", // Non-existent server
		Type: "http",
	}

	transport, err := NewHTTPTransport(config)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// Test connection failure
	err = transport.Connect(ctx)
	assert.Error(t, err)
	assert.False(t, transport.IsConnected())
}
