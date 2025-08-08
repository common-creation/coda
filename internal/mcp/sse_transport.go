package mcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// SSETransport implements Transport interface for Server-Sent Events based MCP communication
type SSETransport struct {
	mu           sync.RWMutex
	config       ServerConfig
	client       *http.Client
	baseURL      string
	headers      map[string]string
	connected    bool
	eventStream  *http.Response
	eventReader  *bufio.Scanner
	eventCh      chan SSEEvent
	stopCh       chan struct{}
	responseCh   map[string]chan interface{}
	responseMu   sync.RWMutex
	requestCount int64
}

// SSEEvent represents a Server-Sent Event
type SSEEvent struct {
	ID    string
	Event string
	Data  string
}

// NewSSETransport creates a new SSE transport instance
func NewSSETransport(config ServerConfig) (Transport, error) {
	if config.URL == "" {
		return nil, NewTransportError("SSE transport requires URL", "sse", nil)
	}

	transport := &SSETransport{
		config:     config,
		baseURL:    config.URL,
		headers:    make(map[string]string),
		eventCh:    make(chan SSEEvent, 100),
		stopCh:     make(chan struct{}),
		responseCh: make(map[string]chan interface{}),
		client: &http.Client{
			Timeout: 0, // No timeout for SSE connections
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

	// Set SSE-specific headers
	transport.headers["Accept"] = "text/event-stream"
	transport.headers["Cache-Control"] = "no-cache"

	return transport, nil
}

// Connect establishes connection to the SSE-based MCP server
func (s *SSETransport) Connect(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.connected {
		return nil
	}

	// Create SSE connection request
	url := fmt.Sprintf("%s/events", s.baseURL)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return NewTransportError("failed to create SSE request", "sse", err)
	}

	// Add headers
	for k, v := range s.headers {
		req.Header.Set(k, v)
	}

	// Establish connection
	resp, err := s.client.Do(req)
	if err != nil {
		return NewTransportError("failed to connect to SSE endpoint", "sse", err)
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return NewTransportError(
			fmt.Sprintf("SSE connection failed with status %d", resp.StatusCode),
			"sse", nil,
		)
	}

	s.eventStream = resp
	s.eventReader = bufio.NewScanner(resp.Body)
	s.connected = true

	// Start event processing goroutines
	go s.readEvents()
	go s.processEvents()

	return nil
}

// Close terminates the SSE connection
func (s *SSETransport) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.connected {
		return nil
	}

	// Signal stop
	close(s.stopCh)

	// Close event stream
	if s.eventStream != nil {
		s.eventStream.Body.Close()
		s.eventStream = nil
	}

	// Clean up response channels
	s.responseMu.Lock()
	for _, ch := range s.responseCh {
		close(ch)
	}
	s.responseCh = make(map[string]chan interface{})
	s.responseMu.Unlock()

	s.connected = false
	return nil
}

// IsConnected returns whether the transport is currently connected
func (s *SSETransport) IsConnected() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.connected
}

// SendRequest sends a request to the server and returns the response
func (s *SSETransport) SendRequest(ctx context.Context, method string, params interface{}) (interface{}, error) {
	s.mu.RLock()
	if !s.connected {
		s.mu.RUnlock()
		return nil, NewTransportError("not connected", "sse", nil)
	}
	s.mu.RUnlock()

	// Generate request ID
	s.responseMu.Lock()
	s.requestCount++
	requestID := fmt.Sprintf("req_%d", s.requestCount)
	responseCh := make(chan interface{}, 1)
	s.responseCh[requestID] = responseCh
	s.responseMu.Unlock()

	// Send request via HTTP POST
	err := s.sendHTTPRequest(ctx, method, params, requestID)
	if err != nil {
		s.responseMu.Lock()
		delete(s.responseCh, requestID)
		s.responseMu.Unlock()
		return nil, err
	}

	// Wait for response
	select {
	case response := <-responseCh:
		s.responseMu.Lock()
		delete(s.responseCh, requestID)
		s.responseMu.Unlock()
		return response, nil
	case <-ctx.Done():
		s.responseMu.Lock()
		delete(s.responseCh, requestID)
		s.responseMu.Unlock()
		return nil, NewTransportError("request timeout", "sse", ctx.Err())
	case <-s.stopCh:
		return nil, NewTransportError("transport stopped", "sse", nil)
	}
}

// SendNotification sends a notification to the server (no response expected)
func (s *SSETransport) SendNotification(ctx context.Context, method string, params interface{}) error {
	s.mu.RLock()
	if !s.connected {
		s.mu.RUnlock()
		return NewTransportError("not connected", "sse", nil)
	}
	s.mu.RUnlock()

	return s.sendHTTPRequest(ctx, method, params, "")
}

// sendHTTPRequest sends an HTTP request to the MCP server
func (s *SSETransport) sendHTTPRequest(ctx context.Context, method string, params interface{}, requestID string) error {
	// Construct the JSON-RPC 2.0 request
	request := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  method,
	}

	if requestID != "" {
		request["id"] = requestID
	}

	if params != nil {
		request["params"] = params
	}

	// Serialize request to JSON
	requestBody, err := json.Marshal(request)
	if err != nil {
		return NewTransportError("failed to marshal request", "sse", err)
	}

	// Create HTTP request
	url := fmt.Sprintf("%s/%s", s.baseURL, method)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(requestBody))
	if err != nil {
		return NewTransportError("failed to create HTTP request", "sse", err)
	}

	// Set headers for JSON request
	httpReq.Header.Set("Content-Type", "application/json")
	for k, v := range s.headers {
		if k != "Accept" && k != "Cache-Control" { // Skip SSE-specific headers for POST requests
			httpReq.Header.Set(k, v)
		}
	}

	// Send request
	resp, err := s.client.Do(httpReq)
	if err != nil {
		return NewTransportError("HTTP request failed", "sse", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return NewTransportError(
			fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(body)),
			"sse",
			nil,
		)
	}

	return nil
}

// readEvents reads events from the SSE stream
func (s *SSETransport) readEvents() {
	defer func() {
		if r := recover(); r != nil {
			// Log panic but don't crash
		}
	}()

	var currentEvent SSEEvent

	for s.eventReader.Scan() {
		line := s.eventReader.Text()

		// Empty line indicates end of event
		if line == "" {
			if currentEvent.Data != "" || currentEvent.Event != "" {
				select {
				case s.eventCh <- currentEvent:
				case <-s.stopCh:
					return
				}
				currentEvent = SSEEvent{}
			}
			continue
		}

		// Parse SSE fields
		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")
			if currentEvent.Data != "" {
				currentEvent.Data += "\n" + data
			} else {
				currentEvent.Data = data
			}
		} else if strings.HasPrefix(line, "event: ") {
			currentEvent.Event = strings.TrimPrefix(line, "event: ")
		} else if strings.HasPrefix(line, "id: ") {
			currentEvent.ID = strings.TrimPrefix(line, "id: ")
		}
	}

	// Scanner stopped, check for error
	if err := s.eventReader.Err(); err != nil {
		// Connection closed or error occurred
		s.mu.Lock()
		s.connected = false
		s.mu.Unlock()
	}
}

// processEvents processes incoming SSE events
func (s *SSETransport) processEvents() {
	defer func() {
		if r := recover(); r != nil {
			// Log panic but don't crash
		}
	}()

	for {
		select {
		case event := <-s.eventCh:
			s.handleEvent(event)
		case <-s.stopCh:
			return
		}
	}
}

// handleEvent handles a specific SSE event
func (s *SSETransport) handleEvent(event SSEEvent) {
	// Try to parse as JSON-RPC response
	var jsonRPCResponse map[string]interface{}
	if err := json.Unmarshal([]byte(event.Data), &jsonRPCResponse); err != nil {
		// Not a JSON-RPC response, ignore or log
		return
	}

	// Check if this is a response to a pending request
	if id, exists := jsonRPCResponse["id"]; exists {
		requestID := fmt.Sprintf("%v", id)

		s.responseMu.RLock()
		responseCh, exists := s.responseCh[requestID]
		s.responseMu.RUnlock()

		if exists {
			// Extract result or error
			var response interface{}
			if result, hasResult := jsonRPCResponse["result"]; hasResult {
				response = result
			} else if errorObj, hasError := jsonRPCResponse["error"]; hasError {
				// Convert error to Go error
				errorMsg := "unknown error"
				if errMap, ok := errorObj.(map[string]interface{}); ok {
					if msg, ok := errMap["message"].(string); ok {
						errorMsg = msg
					}
				}
				response = fmt.Errorf("JSON-RPC error: %s", errorMsg)
			}

			select {
			case responseCh <- response:
			default:
				// Channel full or closed
			}
		}
	}
}

// GetStats returns connection statistics
func (s *SSETransport) GetStats() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	s.responseMu.RLock()
	pendingRequests := len(s.responseCh)
	s.responseMu.RUnlock()

	return map[string]interface{}{
		"connected":        s.connected,
		"url":              s.baseURL,
		"pending_requests": pendingRequests,
		"total_requests":   s.requestCount,
	}
}
