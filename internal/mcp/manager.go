package mcp

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/charmbracelet/log"
	"github.com/common-creation/coda/internal/tools"
)

// MCPManager implements the Manager interface
type MCPManager struct {
	mu           sync.RWMutex
	config       *Config
	configPath   string
	servers      map[string]*ServerInstance
	logger       *log.Logger
	toolRegistry *tools.MCPRegistry
}

// ServerInstance represents a running MCP server instance
type ServerInstance struct {
	Name      string
	Config    ServerConfig
	Status    ServerStatus
	Transport Transport
	cancel    context.CancelFunc
	mu        sync.RWMutex
}

// NewManager creates a new MCP Manager instance
func NewManager(logger *log.Logger) *MCPManager {
	if logger == nil {
		logger = log.New(os.Stderr)
	}

	return &MCPManager{
		servers: make(map[string]*ServerInstance),
		logger:  logger,
	}
}

// LoadConfig loads MCP configuration from the provided paths
func (m *MCPManager) LoadConfig(paths []string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	loader := NewConfigLoader()

	// Use default paths if none provided
	if len(paths) == 0 {
		paths = loader.GetDefaultConfigPaths()
	}

	config, configPath, err := loader.LoadConfigFromPaths(paths)
	if err != nil {
		return fmt.Errorf("failed to load MCP configuration: %w", err)
	}

	m.config = config
	m.configPath = configPath

	m.logger.Info("Loaded MCP configuration", "path", configPath, "servers", len(config.Servers))

	return nil
}

// StartServer starts a specific MCP server
func (m *MCPManager) StartServer(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.config == nil {
		return fmt.Errorf("no configuration loaded")
	}

	serverConfig, exists := m.config.Servers[name]
	if !exists {
		return fmt.Errorf("server %s not found in configuration", name)
	}

	// Check if server is already running
	if instance, exists := m.servers[name]; exists {
		instance.mu.RLock()
		state := instance.Status.State
		instance.mu.RUnlock()

		if state == StateRunning || state == StateStarting {
			return fmt.Errorf("server %s is already running or starting", name)
		}
	}

	// Create new server instance
	ctx, cancel := context.WithCancel(context.Background())

	instance := &ServerInstance{
		Name:   name,
		Config: serverConfig,
		Status: ServerStatus{
			Name:      name,
			State:     StateStarting,
			StartedAt: time.Now(),
			Transport: serverConfig.Type,
		},
		cancel: cancel,
	}

	m.servers[name] = instance

	// Start server in background
	go m.startServerAsync(ctx, instance)

	m.logger.Info("Starting MCP server", "name", name, "transport", serverConfig.Type)

	return nil
}

// StopServer stops a specific MCP server
func (m *MCPManager) StopServer(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	instance, exists := m.servers[name]
	if !exists {
		return fmt.Errorf("server %s not found", name)
	}

	// Cancel the server context
	if instance.cancel != nil {
		instance.cancel()
	}

	// Update status
	instance.mu.Lock()
	oldState := instance.Status.State
	instance.Status.State = StateStopped
	instance.mu.Unlock()

	// Notify tool registry of state change
	m.notifyServerStateChange(name, oldState, StateStopped)

	// Stop transport if available
	if instance.Transport != nil {
		if err := instance.Transport.Close(); err != nil {
			m.logger.Error("Error closing transport", "server", name, "error", err)
		}
	}

	delete(m.servers, name)

	m.logger.Info("Stopped MCP server", "name", name)

	return nil
}

// RestartServer restarts a specific MCP server
func (m *MCPManager) RestartServer(name string) error {
	if err := m.StopServer(name); err != nil {
		// Log error but continue with start
		m.logger.Warn("Error stopping server during restart", "server", name, "error", err)
	}

	// Give it a moment to fully stop
	time.Sleep(100 * time.Millisecond)

	return m.StartServer(name)
}

// StartAll starts all configured MCP servers
func (m *MCPManager) StartAll() error {
	if m.config == nil {
		return fmt.Errorf("no configuration loaded")
	}

	var errors []error
	for name := range m.config.Servers {
		if err := m.StartServer(name); err != nil {
			errors = append(errors, fmt.Errorf("failed to start server %s: %w", name, err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("failed to start some servers: %v", errors)
	}

	return nil
}

// StopAll stops all running MCP servers
func (m *MCPManager) StopAll() error {
	m.mu.RLock()
	serverNames := make([]string, 0, len(m.servers))
	for name := range m.servers {
		serverNames = append(serverNames, name)
	}
	m.mu.RUnlock()

	var errors []error
	for _, name := range serverNames {
		if err := m.StopServer(name); err != nil {
			errors = append(errors, fmt.Errorf("failed to stop server %s: %w", name, err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("failed to stop some servers: %v", errors)
	}

	return nil
}

// GetServerStatus returns the status of a specific server
func (m *MCPManager) GetServerStatus(name string) ServerStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if instance, exists := m.servers[name]; exists {
		instance.mu.RLock()
		defer instance.mu.RUnlock()
		return instance.Status
	}

	// Return stopped status for unknown servers
	return ServerStatus{
		Name:  name,
		State: StateStopped,
	}
}

// GetAllStatuses returns the status of all servers
func (m *MCPManager) GetAllStatuses() map[string]ServerStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()

	statuses := make(map[string]ServerStatus)

	// Add status for all running servers
	for name, instance := range m.servers {
		instance.mu.RLock()
		statuses[name] = instance.Status
		instance.mu.RUnlock()
	}

	// Add status for configured but not running servers
	if m.config != nil {
		for name := range m.config.Servers {
			if _, exists := statuses[name]; !exists {
				statuses[name] = ServerStatus{
					Name:  name,
					State: StateStopped,
				}
			}
		}
	}

	return statuses
}

// ListTools returns all available tools from all servers
func (m *MCPManager) ListTools() ([]ToolInfo, error) {
	// TODO: Implement when transport layer is ready
	m.logger.Debug("ListTools not yet implemented")
	return []ToolInfo{}, nil
}

// ListResources returns all available resources from all servers
func (m *MCPManager) ListResources() ([]ResourceInfo, error) {
	// TODO: Implement when transport layer is ready
	m.logger.Debug("ListResources not yet implemented")
	return []ResourceInfo{}, nil
}

// ListPrompts returns all available prompts from all servers
func (m *MCPManager) ListPrompts() ([]PromptInfo, error) {
	// TODO: Implement when transport layer is ready
	m.logger.Debug("ListPrompts not yet implemented")
	return []PromptInfo{}, nil
}

// ExecuteTool executes a tool on the specified server
func (m *MCPManager) ExecuteTool(serverName, toolName string, params map[string]interface{}) (interface{}, error) {
	// TODO: Implement when transport layer is ready
	m.logger.Debug("ExecuteTool not yet implemented", "server", serverName, "tool", toolName)
	return nil, fmt.Errorf("tool execution not yet implemented")
}

// startServerAsync starts a server instance asynchronously
func (m *MCPManager) startServerAsync(ctx context.Context, instance *ServerInstance) {
	defer func() {
		if r := recover(); r != nil {
			instance.mu.Lock()
			instance.Status.State = StateError
			instance.Status.Error = fmt.Errorf("server panic: %v", r)
			instance.mu.Unlock()
			m.logger.Error("MCP server panicked", "server", instance.Name, "error", r)
			m.notifyServerStateChange(instance.Name, StateStarting, StateError)
		}
	}()

	// Create transport
	transport, err := m.createTransport(instance.Config)
	if err != nil {
		instance.mu.Lock()
		instance.Status.State = StateError
		instance.Status.Error = fmt.Errorf("failed to create transport: %w", err)
		instance.mu.Unlock()
		m.logger.Error("Failed to create transport", "server", instance.Name, "error", err)
		m.notifyServerStateChange(instance.Name, StateStarting, StateError)
		return
	}

	instance.mu.Lock()
	instance.Transport = transport
	instance.mu.Unlock()

	// Initialize connection
	if err := transport.Connect(ctx); err != nil {
		instance.mu.Lock()
		instance.Status.State = StateError
		instance.Status.Error = fmt.Errorf("failed to connect: %w", err)
		instance.mu.Unlock()
		m.logger.Error("Failed to connect to MCP server", "server", instance.Name, "error", err)
		m.notifyServerStateChange(instance.Name, StateStarting, StateError)
		return
	}

	// Update status to running
	instance.mu.Lock()
	oldState := instance.Status.State
	instance.Status.State = StateRunning
	instance.Status.Error = nil
	instance.mu.Unlock()

	m.logger.Info("MCP server started successfully", "server", instance.Name)

	// Notify tool registry of state change
	m.notifyServerStateChange(instance.Name, oldState, StateRunning)

	// Keep server running until context is cancelled
	<-ctx.Done()

	// Clean shutdown
	if err := transport.Close(); err != nil {
		m.logger.Error("Error during server shutdown", "server", instance.Name, "error", err)
	}
}

// SetToolRegistry sets the tool registry for dynamic tool management
func (m *MCPManager) SetToolRegistry(registry *tools.MCPRegistry) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.toolRegistry = registry
}

// GetToolRegistry returns the current tool registry
func (m *MCPManager) GetToolRegistry() *tools.MCPRegistry {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.toolRegistry
}

// notifyServerStateChange notifies the tool registry of server state changes
func (m *MCPManager) notifyServerStateChange(serverName string, oldState, newState State) {
	if m.toolRegistry != nil {
		go m.toolRegistry.HandleServerStateChange(serverName, tools.State(oldState), tools.State(newState))
	}
}

// createTransport creates the appropriate transport for the server configuration
func (m *MCPManager) createTransport(config ServerConfig) (Transport, error) {
	factory := NewTransportFactory()
	return factory.CreateTransport(config)
}
