package config

import (
	_ "embed"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/common-creation/coda/internal/logging"
	"gopkg.in/yaml.v3"
)

//go:embed config.example.yaml
var embeddedConfigSample string

// Loader handles configuration loading and saving
type Loader struct {
	// Config file paths in priority order
	searchPaths []string
}

// NewLoader creates a new configuration loader
func NewLoader() *Loader {
	return &Loader{
		searchPaths: getDefaultSearchPaths(),
	}
}

// Load loads configuration from file and environment variables
func (l *Loader) Load(explicitPath string) (*Config, error) {
	// Start with default configuration
	cfg := NewDefaultConfig()

	// Find config file
	configPath := ""
	if explicitPath != "" {
		// Use explicitly provided path
		configPath = explicitPath
	} else {
		// Search for config file
		for _, path := range l.searchPaths {
			if fileExists(path) {
				configPath = path
				break
			}
		}
	}

	// Load from file if found
	if configPath != "" {
		fileCfg, err := l.loadFromFile(configPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load config from %s: %w", configPath, err)
		}
		// Merge file config into default config
		if err := mergeConfig(cfg, fileCfg); err != nil {
			return nil, fmt.Errorf("failed to merge config: %w", err)
		}
	} else {
		// No config file found, create one from embedded sample
		if err := l.createDefaultConfig(); err != nil {
			// Log warning but continue with default config
			fmt.Fprintf(os.Stderr, "Warning: Failed to create default config file: %v\n", err)
		} else {
			fmt.Fprintf(os.Stderr, "Created default config file at ~/.config/coda/config.yaml\n")
			fmt.Fprintf(os.Stderr, "Please edit it to add your API key.\n")
		}
	}

	// Apply environment variables override
	applyEnvironmentOverrides(cfg)

	// Validate final configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return cfg, nil
}

// Save saves configuration to file
func (l *Loader) Save(path string, cfg *Config) error {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Marshal config to YAML
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write to file
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// GetConfigPath returns the path where config would be loaded from
func (l *Loader) GetConfigPath(explicitPath string) string {
	if explicitPath != "" {
		return explicitPath
	}

	// Check environment variable
	if envPath := os.Getenv("CODA_CONFIG_PATH"); envPath != "" {
		return envPath
	}

	// Return first existing path or default path
	for _, path := range l.searchPaths {
		if fileExists(path) {
			return path
		}
	}

	// Return default path if none exists
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ".config", "coda", "config.yaml")
}

// loadFromFile loads configuration from YAML file
func (l *Loader) loadFromFile(path string) (*Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// getDefaultSearchPaths returns the default configuration search paths
func getDefaultSearchPaths() []string {
	paths := []string{}

	// Environment variable path
	if envPath := os.Getenv("CODA_CONFIG_PATH"); envPath != "" {
		paths = append(paths, envPath)
	}

	// Current directory - prioritized first
	paths = append(paths, "config.yaml")

	// User config directory
	if homeDir, err := os.UserHomeDir(); err == nil {
		paths = append(paths, filepath.Join(homeDir, ".config", "coda", "config.yaml"))
	}

	return paths
}

// mergeConfig merges source config into destination config
func mergeConfig(dst, src *Config) error {
	// Merge AI config
	if src.AI.Provider != "" {
		dst.AI.Provider = src.AI.Provider
	}
	if src.AI.APIKey != "" {
		dst.AI.APIKey = src.AI.APIKey
	}
	if src.AI.Model != "" {
		dst.AI.Model = src.AI.Model
	}
	if src.AI.Temperature != 0 {
		dst.AI.Temperature = src.AI.Temperature
	}
	if src.AI.MaxTokens != 0 {
		dst.AI.MaxTokens = src.AI.MaxTokens
	}

	// Merge OpenAI config
	if src.AI.OpenAI.BaseURL != "" {
		dst.AI.OpenAI.BaseURL = src.AI.OpenAI.BaseURL
	}
	if src.AI.OpenAI.Organization != "" {
		dst.AI.OpenAI.Organization = src.AI.OpenAI.Organization
	}

	// Merge Azure config
	if src.AI.Azure.Endpoint != "" {
		dst.AI.Azure.Endpoint = src.AI.Azure.Endpoint
	}
	if src.AI.Azure.DeploymentName != "" {
		dst.AI.Azure.DeploymentName = src.AI.Azure.DeploymentName
	}
	if src.AI.Azure.APIVersion != "" {
		dst.AI.Azure.APIVersion = src.AI.Azure.APIVersion
	}

	// Merge Tools config
	if len(src.Tools.Enabled) > 0 {
		dst.Tools.Enabled = src.Tools.Enabled
	}
	if src.Tools.WorkspaceRoot != "" {
		dst.Tools.WorkspaceRoot = src.Tools.WorkspaceRoot
	}
	dst.Tools.AutoApprove = src.Tools.AutoApprove

	// Merge FileAccess config
	if len(src.Tools.FileAccess.AllowedPaths) > 0 {
		dst.Tools.FileAccess.AllowedPaths = src.Tools.FileAccess.AllowedPaths
	}
	if len(src.Tools.FileAccess.DeniedPaths) > 0 {
		dst.Tools.FileAccess.DeniedPaths = src.Tools.FileAccess.DeniedPaths
	}
	if src.Tools.FileAccess.MaxFileSize != 0 {
		dst.Tools.FileAccess.MaxFileSize = src.Tools.FileAccess.MaxFileSize
	}

	// Merge UI config
	if src.UI.Theme != "" {
		dst.UI.Theme = src.UI.Theme
	}
	dst.UI.SyntaxHighlighting = src.UI.SyntaxHighlighting
	dst.UI.MarkdownRendering = src.UI.MarkdownRendering
	if src.UI.KeyBindings != "" {
		dst.UI.KeyBindings = src.UI.KeyBindings
	}

	// Merge Logging config - comprehensive merge for new logging system
	if src.Logging.Level != "" {
		dst.Logging.Level = src.Logging.Level
	}
	if len(src.Logging.Outputs) > 0 {
		dst.Logging.Outputs = src.Logging.Outputs
	}
	if src.Logging.Sampling.Enabled != dst.Logging.Sampling.Enabled {
		dst.Logging.Sampling = src.Logging.Sampling
	}
	if src.Logging.Privacy.Enabled != dst.Logging.Privacy.Enabled {
		dst.Logging.Privacy = src.Logging.Privacy
	}
	if src.Logging.Buffering.Enabled != dst.Logging.Buffering.Enabled {
		dst.Logging.Buffering = src.Logging.Buffering
	}
	if src.Logging.Rotation.MaxSize != 0 {
		dst.Logging.Rotation = src.Logging.Rotation
	}

	// Merge Session config
	if src.Session.HistoryFile != "" {
		dst.Session.HistoryFile = src.Session.HistoryFile
	}
	if src.Session.MaxHistory != 0 {
		dst.Session.MaxHistory = src.Session.MaxHistory
	}
	if src.Session.AutoSaveInterval != 0 {
		dst.Session.AutoSaveInterval = src.Session.AutoSaveInterval
	}

	return nil
}

// applyEnvironmentOverrides applies environment variable overrides to config
func applyEnvironmentOverrides(cfg *Config) {
	// AI overrides
	if provider := os.Getenv("CODA_AI_PROVIDER"); provider != "" {
		cfg.AI.Provider = provider
	}
	if apiKey := os.Getenv("CODA_AI_API_KEY"); apiKey != "" {
		cfg.AI.APIKey = apiKey
	}
	// Also check provider-specific API key env vars
	if cfg.AI.Provider == "openai" && cfg.AI.APIKey == "" {
		if apiKey := os.Getenv("OPENAI_API_KEY"); apiKey != "" {
			cfg.AI.APIKey = apiKey
		}
	}
	if cfg.AI.Provider == "azure" && cfg.AI.APIKey == "" {
		if apiKey := os.Getenv("AZURE_OPENAI_API_KEY"); apiKey != "" {
			cfg.AI.APIKey = apiKey
		}
	}

	if model := os.Getenv("CODA_MODEL"); model != "" {
		cfg.AI.Model = model
	}

	// OpenAI specific
	if baseURL := os.Getenv("OPENAI_BASE_URL"); baseURL != "" {
		cfg.AI.OpenAI.BaseURL = baseURL
	}
	if org := os.Getenv("OPENAI_ORGANIZATION"); org != "" {
		cfg.AI.OpenAI.Organization = org
	}

	// Azure specific
	if endpoint := os.Getenv("AZURE_OPENAI_ENDPOINT"); endpoint != "" {
		cfg.AI.Azure.Endpoint = endpoint
	}
	if deployment := os.Getenv("AZURE_OPENAI_DEPLOYMENT"); deployment != "" {
		cfg.AI.Azure.DeploymentName = deployment
	}
	if apiVersion := os.Getenv("AZURE_OPENAI_API_VERSION"); apiVersion != "" {
		cfg.AI.Azure.APIVersion = apiVersion
	}

	// Tools overrides
	if workspace := os.Getenv("CODA_WORKSPACE"); workspace != "" {
		cfg.Tools.WorkspaceRoot = workspace
	}
	if autoApprove := os.Getenv("CODA_AUTO_APPROVE"); autoApprove != "" {
		cfg.Tools.AutoApprove = strings.ToLower(autoApprove) == "true"
	}

	// Logging overrides - basic environment variable support
	if logLevel := os.Getenv("CODA_LOG_LEVEL"); logLevel != "" {
		cfg.Logging.Level = logLevel
	}
	// Environment can override to simple file logging
	if logFile := os.Getenv("CODA_LOG_FILE"); logFile != "" {
		// Add file output to existing outputs
		cfg.Logging.Outputs = append(cfg.Logging.Outputs,
			logging.OutputConfig{
				Type:   "file",
				Target: logFile,
				Format: "json",
			})
	}

	// UI overrides
	if theme := os.Getenv("CODA_THEME"); theme != "" {
		cfg.UI.Theme = theme
	}
}

// fileExists checks if a file exists
func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

// createDefaultConfig creates a default config file from embedded sample
func (l *Loader) createDefaultConfig() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	configPath := filepath.Join(homeDir, ".config", "coda", "config.yaml")
	configDir := filepath.Dir(configPath)

	// Create directory if it doesn't exist
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Check if file already exists
	if fileExists(configPath) {
		return nil // Already exists, don't overwrite
	}

	// Write embedded sample to file
	if err := os.WriteFile(configPath, []byte(embeddedConfigSample), 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// CreateSampleConfig creates a sample configuration file
func CreateSampleConfig(path string) error {
	sampleConfig := `# CODA Configuration File
# This is a sample configuration file for CODA

# AI Provider Configuration
ai:
  # Provider can be "openai" or "azure"
  provider: openai
  
  # API key (can also be set via OPENAI_API_KEY or AZURE_OPENAI_API_KEY env var)
  # api_key: your-api-key-here
  
  # Model to use
  model: gpt-4
  
  # Temperature (0-2, default: 0.7)
  temperature: 0.7
  
  # Maximum tokens for response
  max_tokens: 4096
  
  # OpenAI specific settings
  openai:
    # Custom base URL (optional)
    # base_url: https://api.openai.com/v1
    
    # Organization ID (optional)
    # organization: org-xxxxx
  
  # Azure OpenAI specific settings
  azure:
    # Azure endpoint (required for Azure)
    # endpoint: https://your-resource.openai.azure.com
    
    # Deployment name (required for Azure)
    # deployment_name: your-deployment
    
    # API version
    api_version: "2024-02-01"

# Tools Configuration
tools:
  # Enabled tools
  enabled:
    - read_file
    - write_file
    - edit_file
    - list_files
    - search_files
  
  # Workspace root directory
  workspace_root: "."
  
  # Auto-approve tool executions (use with caution!)
  auto_approve: false
  
  # File access restrictions
  file_access:
    # Allowed paths (glob patterns)
    allowed_paths:
      - "**/*"
    
    # Denied paths (glob patterns)
    denied_paths:
      - "**/node_modules/**"
      - "**/.git/**"
      - "**/vendor/**"
      - "**/*.exe"
      - "**/*.dll"
      - "**/*.so"
      - "**/*.dylib"
    
    # Maximum file size in bytes (10MB)
    max_file_size: 10485760

# UI Configuration
ui:
  # Theme name
  theme: default
  
  # Enable syntax highlighting
  syntax_highlighting: true
  
  # Enable markdown rendering
  markdown_rendering: true
  
  # Key bindings preset
  key_bindings: default

# Logging Configuration
logging:
  # Log level (debug, info, warn, error)
  level: info
  
  # Log format (text, json)
  format: text
  
  # Log file path (empty for stdout only)
  # file: /path/to/logfile.log
  
  # Enable timestamps
  timestamp: true

# Session Configuration
session:
  # History file path
  # history_file: ~/.config/coda/history.json
  
  # Maximum history entries
  max_history: 1000
  
  # Auto-save interval in seconds
  auto_save_interval: 30
`

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write sample config
	if err := os.WriteFile(path, []byte(sampleConfig), 0644); err != nil {
		return fmt.Errorf("failed to write sample config: %w", err)
	}

	return nil
}
