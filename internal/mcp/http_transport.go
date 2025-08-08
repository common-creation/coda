package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// HTTPTransport implements Transport interface for HTTP-based MCP communication
type HTTPTransport struct {
	mu        sync.RWMutex
	config    ServerConfig
	client    *http.Client
	baseURL   string
	headers   map[string]string
	connected bool
}

// NewHTTPTransport creates a new HTTP transport instance
func NewHTTPTransport(config ServerConfig) (Transport, error) {
	if config.URL == "" {
		return nil, NewTransportError("HTTP transport requires URL", "http", nil)
	}

	transport := &HTTPTransport{
		config:  config,
		baseURL: config.URL,
		headers: make(map[string]string),
		client: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        10,
				MaxIdleConnsPerHost: 5,
				IdleConnTimeout:     90 * time.Second,
			},
		},
	}

	// Copy headers from config
	for k, v := range config.Headers {
		transport.headers[k] = v
	}

	// Set default headers if not provided
	if _, exists := transport.headers["Content-Type"]; !exists {
		transport.headers["Content-Type"] = "application/json"
	}
	if _, exists := transport.headers["Accept"]; !exists {
		transport.headers["Accept"] = "application/json"
	}

	return transport, nil
}

// Connect establishes connection to the HTTP-based MCP server
func (h *HTTPTransport) Connect(ctx context.Context) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Test connection by sending an initialize request
	initParams := map[string]interface{}{
		"protocolVersion": "2024-11-05",
		"capabilities": map[string]interface{}{
			"tools":     map[string]interface{}{},
			"resources": map[string]interface{}{},
			"prompts":   map[string]interface{}{},
		},
		"clientInfo": map[string]interface{}{
			"name":    "coda",
			"version": "1.0.0",
		},
	}

	_, err := h.sendRequestInternal(ctx, "initialize", initParams)
	if err != nil {
		return NewTransportError("failed to initialize HTTP connection", "http", err)
	}

	h.connected = true
	return nil
}

// Close terminates the HTTP connection
func (h *HTTPTransport) Close() error {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Close idle connections
	h.client.CloseIdleConnections()
	h.connected = false
	return nil
}

// IsConnected returns whether the transport is currently connected
func (h *HTTPTransport) IsConnected() bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.connected
}

// SendRequest sends a request to the server and returns the response
func (h *HTTPTransport) SendRequest(ctx context.Context, method string, params interface{}) (interface{}, error) {
	h.mu.RLock()
	if !h.connected {
		h.mu.RUnlock()
		return nil, NewTransportError("not connected", "http", nil)
	}
	h.mu.RUnlock()

	return h.sendRequestInternal(ctx, method, params)
}

// SendNotification sends a notification to the server (no response expected)
func (h *HTTPTransport) SendNotification(ctx context.Context, method string, params interface{}) error {
	h.mu.RLock()
	if !h.connected {
		h.mu.RUnlock()
		return NewTransportError("not connected", "http", nil)
	}
	h.mu.RUnlock()

	_, err := h.sendRequestInternal(ctx, method, params)
	return err
}

// sendRequestInternal sends an HTTP request to the MCP server
func (h *HTTPTransport) sendRequestInternal(ctx context.Context, method string, params interface{}) (interface{}, error) {
	// Construct the JSON-RPC 2.0 request
	request := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      h.generateRequestID(),
		"method":  method,
	}

	if params != nil {
		request["params"] = params
	}

	// Serialize request to JSON
	requestBody, err := json.Marshal(request)
	if err != nil {
		return nil, NewTransportError("failed to marshal request", "http", err)
	}

	// Create HTTP request
	url := fmt.Sprintf("%s/%s", h.baseURL, method)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(requestBody))
	if err != nil {
		return nil, NewTransportError("failed to create HTTP request", "http", err)
	}

	// Add headers
	for k, v := range h.headers {
		httpReq.Header.Set(k, v)
	}

	// Send request
	resp, err := h.client.Do(httpReq)
	if err != nil {
		return nil, NewTransportError("HTTP request failed", "http", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, NewTransportError(
			fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(body)),
			"http",
			nil,
		)
	}

	// Read response body
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, NewTransportError("failed to read response body", "http", err)
	}

	// Parse JSON-RPC response
	var jsonRPCResponse map[string]interface{}
	if err := json.Unmarshal(responseBody, &jsonRPCResponse); err != nil {
		return nil, NewTransportError("failed to parse JSON response", "http", err)
	}

	// Check for JSON-RPC error
	if errorObj, exists := jsonRPCResponse["error"]; exists {
		errorMsg := "unknown error"
		if errMap, ok := errorObj.(map[string]interface{}); ok {
			if msg, ok := errMap["message"].(string); ok {
				errorMsg = msg
			}
		}
		return nil, NewTransportError(fmt.Sprintf("JSON-RPC error: %s", errorMsg), "http", nil)
	}

	// Return the result
	if result, exists := jsonRPCResponse["result"]; exists {
		return result, nil
	}

	return nil, nil
}

// generateRequestID generates a unique request ID for JSON-RPC
func (h *HTTPTransport) generateRequestID() string {
	return fmt.Sprintf("req_%d", time.Now().UnixNano())
}

// SetTimeout sets the HTTP client timeout
func (h *HTTPTransport) SetTimeout(timeout time.Duration) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.client.Timeout = timeout
}

// SetHeaders updates the transport headers
func (h *HTTPTransport) SetHeaders(headers map[string]string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for k, v := range headers {
		h.headers[k] = v
	}
}

// GetStats returns connection statistics
func (h *HTTPTransport) GetStats() map[string]interface{} {
	h.mu.RLock()
	defer h.mu.RUnlock()

	return map[string]interface{}{
		"connected": h.connected,
		"url":       h.baseURL,
		"timeout":   h.client.Timeout.String(),
	}
}
