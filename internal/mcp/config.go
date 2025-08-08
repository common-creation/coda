package mcp

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// ConfigLoader handles loading and parsing of MCP configuration files
type ConfigLoader struct {
	envVarPattern *regexp.Regexp
}

// NewConfigLoader creates a new ConfigLoader instance
func NewConfigLoader() *ConfigLoader {
	return &ConfigLoader{
		envVarPattern: regexp.MustCompile(`\${([^}]+)}`),
	}
}

// LoadConfigFromPaths attempts to load MCP configuration from the given paths in order
// Returns the first successfully loaded configuration
func (cl *ConfigLoader) LoadConfigFromPaths(paths []string) (*Config, string, error) {
	for _, path := range paths {
		if config, err := cl.LoadConfigFromPath(path); err == nil {
			return config, path, nil
		}
	}

	return nil, "", fmt.Errorf("no valid MCP configuration found in any of the provided paths: %v", paths)
}

// LoadConfigFromPath loads MCP configuration from a specific file path
func (cl *ConfigLoader) LoadConfigFromPath(path string) (*Config, error) {
	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("configuration file not found: %s", path)
	}

	// Open and read the file
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open configuration file %s: %w", path, err)
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read configuration file %s: %w", path, err)
	}

	// Expand environment variables
	expandedData := cl.expandEnvironmentVariables(string(data))

	// Parse JSON
	var config Config
	if err := json.Unmarshal([]byte(expandedData), &config); err != nil {
		return nil, fmt.Errorf("failed to parse configuration file %s: %w", path, err)
	}

	// Validate configuration
	if err := cl.validateConfig(&config); err != nil {
		return nil, fmt.Errorf("invalid configuration in %s: %w", path, err)
	}

	return &config, nil
}

// GetDefaultConfigPaths returns the default paths to search for MCP configuration files
func (cl *ConfigLoader) GetDefaultConfigPaths() []string {
	paths := []string{}

	// 1. Current directory (project-local)
	if cwd, err := os.Getwd(); err == nil {
		paths = append(paths, filepath.Join(cwd, ".mcp.json"))
	}

	// 2. User's CODA config directory
	if homeDir, err := os.UserHomeDir(); err == nil {
		paths = append(paths, filepath.Join(homeDir, ".coda", "mcp.json"))
	}

	// 3. Environment variable specified directory
	if codaConfigDir := os.Getenv("CODA_CONFIG_DIR"); codaConfigDir != "" {
		paths = append(paths, filepath.Join(codaConfigDir, "mcp.json"))
	}

	return paths
}

// expandEnvironmentVariables expands ${VAR_NAME} patterns with environment variables
func (cl *ConfigLoader) expandEnvironmentVariables(input string) string {
	return cl.envVarPattern.ReplaceAllStringFunc(input, func(match string) string {
		// Extract variable name (remove ${ and })
		varName := match[2 : len(match)-1]

		// Get environment variable value
		if value := os.Getenv(varName); value != "" {
			return value
		}

		// Return original match if environment variable is not set
		return match
	})
}

// validateConfig validates the MCP configuration
func (cl *ConfigLoader) validateConfig(config *Config) error {
	if config.Servers == nil {
		return fmt.Errorf("mcpServers section is required")
	}

	if len(config.Servers) == 0 {
		return fmt.Errorf("at least one MCP server must be configured")
	}

	for name, serverConfig := range config.Servers {
		if err := cl.validateServerConfig(name, &serverConfig); err != nil {
			return fmt.Errorf("invalid server configuration for %s: %w", name, err)
		}
	}

	return nil
}

// validateServerConfig validates a single server configuration
func (cl *ConfigLoader) validateServerConfig(name string, config *ServerConfig) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("server name cannot be empty")
	}

	// Set default transport type if not specified
	if config.Type == "" {
		config.Type = "stdio"
	}

	switch config.Type {
	case "stdio":
		if config.Command == "" {
			return fmt.Errorf("command is required for stdio transport")
		}
	case "http", "sse":
		if config.URL == "" {
			return fmt.Errorf("URL is required for %s transport", config.Type)
		}
	default:
		return fmt.Errorf("unsupported transport type: %s (supported: stdio, http, sse)", config.Type)
	}

	return nil
}
