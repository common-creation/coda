package mcp

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/charmbracelet/log"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// StdioTransport implements Transport interface using stdio communication
type StdioTransport struct {
	config  ServerConfig
	cmd     *exec.Cmd
	client  *mcpsdk.Client
	session *mcpsdk.ClientSession
	logger  *log.Logger

	mu        sync.RWMutex
	connected bool

	// For graceful shutdown
	cancel context.CancelFunc
	done   chan struct{}
}

// NewStdioTransport creates a new stdio transport instance
func NewStdioTransport(config ServerConfig) (Transport, error) {
	if config.Command == "" {
		return nil, NewTransportError("command is required for stdio transport", "stdio", nil)
	}

	logger := log.New(os.Stderr)

	return &StdioTransport{
		config: config,
		logger: logger,
		done:   make(chan struct{}),
	}, nil
}

// Connect establishes connection to the MCP server via stdio
func (st *StdioTransport) Connect(ctx context.Context) error {
	st.mu.Lock()
	defer st.mu.Unlock()

	if st.connected {
		return NewTransportError("transport already connected", "stdio", nil)
	}

	// Create context for the subprocess
	subCtx, cancel := context.WithCancel(ctx)
	st.cancel = cancel

	// Build command with arguments
	st.cmd = exec.CommandContext(subCtx, st.config.Command, st.config.Args...)

	// Set environment variables if provided
	if len(st.config.Env) > 0 {
		env := os.Environ()
		for key, value := range st.config.Env {
			env = append(env, fmt.Sprintf("%s=%s", key, value))
		}
		st.cmd.Env = env
	}

	st.logger.Debug("Starting MCP server process", "command", st.config.Command, "args", st.config.Args)

	// Start the command
	if err := st.cmd.Start(); err != nil {
		return NewTransportError("failed to start MCP server process", "stdio", err)
	}

	st.logger.Debug("Started MCP server process", "pid", st.cmd.Process.Pid, "command", st.config.Command)

	// Create command transport (which is the client counterpart to stdio transport)
	transport := mcpsdk.NewCommandTransport(st.cmd)

	// Create client implementation info
	impl := &mcpsdk.Implementation{
		Name:    "CODA",
		Version: "1.0.0",
		Title:   "CODA - CODing Agent",
	}

	// Create MCP client
	client := mcpsdk.NewClient(impl, &mcpsdk.ClientOptions{})

	// Connect to the server
	session, err := client.Connect(ctx, transport)
	if err != nil {
		st.cleanup()
		return NewTransportError("failed to connect to MCP server", "stdio", err)
	}

	st.client = client
	st.session = session
	st.connected = true

	// Start monitoring the process
	go st.monitorProcess(subCtx)

	st.logger.Info("Successfully connected to MCP server via stdio")
	return nil
}

// Close terminates the connection and cleans up resources
func (st *StdioTransport) Close() error {
	st.mu.Lock()
	defer st.mu.Unlock()

	if !st.connected {
		return nil
	}

	st.logger.Debug("Closing stdio transport")

	// Cancel the subprocess context
	if st.cancel != nil {
		st.cancel()
	}

	// Cleanup resources
	st.cleanup()

	// Wait for process monitor to finish
	select {
	case <-st.done:
		st.logger.Debug("Process monitor finished")
	case <-time.After(5 * time.Second):
		st.logger.Warn("Process monitor did not finish within timeout")
	}

	st.connected = false
	st.logger.Info("Stdio transport closed")

	return nil
}

// IsConnected returns whether the transport is currently connected
func (st *StdioTransport) IsConnected() bool {
	st.mu.RLock()
	defer st.mu.RUnlock()
	return st.connected
}

// SendRequest sends a request to the server and returns the response
func (st *StdioTransport) SendRequest(ctx context.Context, method string, params interface{}) (interface{}, error) {
	st.mu.RLock()
	session := st.session
	connected := st.connected
	st.mu.RUnlock()

	if !connected || session == nil {
		return nil, NewTransportError("transport not connected", "stdio", nil)
	}

	// MCP SDK handles specific methods, for now return a basic response
	// TODO: Implement proper method routing based on MCP protocol
	st.logger.Debug("SendRequest called but not fully implemented", "method", method)
	return map[string]interface{}{"result": "success"}, nil
}

// SendNotification sends a notification to the server (no response expected)
func (st *StdioTransport) SendNotification(ctx context.Context, method string, params interface{}) error {
	st.mu.RLock()
	session := st.session
	connected := st.connected
	st.mu.RUnlock()

	if !connected || session == nil {
		return NewTransportError("transport not connected", "stdio", nil)
	}

	// MCP SDK handles specific notifications, for now log and return success
	// TODO: Implement proper notification routing based on MCP protocol
	st.logger.Debug("SendNotification called but not fully implemented", "method", method)
	return nil
}

// Note: MCP initialization is handled automatically by the SDK when connecting

// monitorProcess monitors the subprocess and handles cleanup on termination
func (st *StdioTransport) monitorProcess(ctx context.Context) {
	defer close(st.done)

	if st.cmd == nil || st.cmd.Process == nil {
		return
	}

	// Wait for either context cancellation or process termination
	processErr := make(chan error, 1)
	go func() {
		processErr <- st.cmd.Wait()
	}()

	select {
	case <-ctx.Done():
		st.logger.Debug("Context cancelled, terminating MCP server process")
		// Give the process a chance to exit gracefully
		select {
		case <-processErr:
			st.logger.Debug("MCP server process exited gracefully")
		case <-time.After(5 * time.Second):
			st.logger.Warn("MCP server process did not exit gracefully, forcing termination")
			if err := st.cmd.Process.Kill(); err != nil {
				st.logger.Error("Failed to kill MCP server process", "error", err)
			}
		}
	case err := <-processErr:
		st.mu.Lock()
		st.connected = false
		st.mu.Unlock()

		if err != nil {
			st.logger.Error("MCP server process exited with error", "error", err)
		} else {
			st.logger.Info("MCP server process exited normally")
		}
	}
}

// cleanup cleans up transport resources
func (st *StdioTransport) cleanup() {
	// Close the MCP session
	if st.session != nil {
		if err := st.session.Close(); err != nil {
			st.logger.Error("Error closing MCP session", "error", err)
		}
		st.session = nil
	}

	// Note: MCP client doesn't have a Close method in this SDK version
	// Cleanup is handled through session closure
	st.client = nil

	// Terminate the process if it's still running
	if st.cmd != nil && st.cmd.Process != nil {
		if err := st.cmd.Process.Kill(); err != nil {
			st.logger.Debug("Process already terminated or error killing", "error", err)
		}
		st.cmd = nil
	}
}
