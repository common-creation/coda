package mcp

import (
	"context"
)

// Transport defines the interface for MCP server communication
type Transport interface {
	// Connect establishes connection to the MCP server
	Connect(ctx context.Context) error

	// Close terminates the connection to the MCP server
	Close() error

	// IsConnected returns whether the transport is currently connected
	IsConnected() bool

	// SendRequest sends a request to the server and returns the response
	SendRequest(ctx context.Context, method string, params interface{}) (interface{}, error)

	// SendNotification sends a notification to the server (no response expected)
	SendNotification(ctx context.Context, method string, params interface{}) error
}

// TransportFactory creates Transport instances based on server configuration
type TransportFactory interface {
	CreateTransport(config ServerConfig) (Transport, error)
}

// DefaultTransportFactory implements TransportFactory
type DefaultTransportFactory struct{}

// NewTransportFactory creates a new DefaultTransportFactory
func NewTransportFactory() *DefaultTransportFactory {
	return &DefaultTransportFactory{}
}

// CreateTransport creates the appropriate transport based on the configuration
func (f *DefaultTransportFactory) CreateTransport(config ServerConfig) (Transport, error) {
	switch config.Type {
	case "stdio":
		return NewStdioTransport(config)
	case "http":
		return NewHTTPTransport(config)
	case "sse":
		return NewSSETransport(config)
	default:
		return nil, NewTransportError("unsupported transport type", config.Type, nil)
	}
}

// TransportError represents transport-specific errors
type TransportError struct {
	Message   string
	Transport string
	Cause     error
}

// NewTransportError creates a new TransportError
func NewTransportError(message, transport string, cause error) *TransportError {
	return &TransportError{
		Message:   message,
		Transport: transport,
		Cause:     cause,
	}
}

// Error implements the error interface
func (e *TransportError) Error() string {
	if e.Cause != nil {
		return e.Message + " (" + e.Transport + "): " + e.Cause.Error()
	}
	return e.Message + " (" + e.Transport + ")"
}

// Unwrap returns the underlying cause error
func (e *TransportError) Unwrap() error {
	return e.Cause
}

// DummyTransport is a placeholder implementation for testing
type DummyTransport struct {
	connected bool
}

// Connect implements Transport.Connect
func (dt *DummyTransport) Connect(ctx context.Context) error {
	dt.connected = true
	return nil
}

// Close implements Transport.Close
func (dt *DummyTransport) Close() error {
	dt.connected = false
	return nil
}

// IsConnected implements Transport.IsConnected
func (dt *DummyTransport) IsConnected() bool {
	return dt.connected
}

// SendRequest implements Transport.SendRequest
func (dt *DummyTransport) SendRequest(ctx context.Context, method string, params interface{}) (interface{}, error) {
	return map[string]interface{}{"result": "dummy response"}, nil
}

// SendNotification implements Transport.SendNotification
func (dt *DummyTransport) SendNotification(ctx context.Context, method string, params interface{}) error {
	return nil
}

// Placeholder functions for transport implementations that will be implemented later

// Note: NewStdioTransport is implemented in stdio_transport.go

// Note: NewHTTPTransport is implemented in http_transport.go
// Note: NewSSETransport is implemented in sse_transport.go
