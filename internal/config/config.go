package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/common-creation/coda/internal/logging"
)

// Config represents the complete configuration for CODA
type Config struct {
	// AI configuration
	AI AIConfig `yaml:"ai" json:"ai"`

	// Tools configuration
	Tools ToolsConfig `yaml:"tools" json:"tools"`

	// UI configuration
	UI UIConfig `yaml:"ui" json:"ui"`

	// Logging configuration
	Logging logging.LoggingConfig `yaml:"logging" json:"logging"`

	// Session configuration
	Session SessionConfig `yaml:"session" json:"session"`
}

// AIConfig contains AI provider specific configuration
type AIConfig struct {
	// Provider can be "openai" or "azure"
	Provider string `yaml:"provider" json:"provider"`

	// API key for authentication
	APIKey string `yaml:"api_key" json:"api_key"`

	// Model name to use
	Model string `yaml:"model" json:"model"`

	// Temperature for response generation (0-2)
	Temperature float32 `yaml:"temperature" json:"temperature"`

	// Maximum tokens for response
	MaxTokens int `yaml:"max_tokens" json:"max_tokens"`

	// OpenAI specific settings
	OpenAI OpenAIConfig `yaml:"openai" json:"openai"`

	// Azure specific settings
	Azure AzureConfig `yaml:"azure" json:"azure"`
	
	// Reasoning effort for GPT-5 models (optional)
	// Valid values: "minimal", "low", "medium", "high"
	ReasoningEffort *string `yaml:"reasoning_effort,omitempty" json:"reasoning_effort,omitempty"`
	
	// Use Structured Outputs for tool calls (requires GPT-4o-2024-08-06 or later)
	UseStructuredOutputs bool `yaml:"use_structured_outputs" json:"use_structured_outputs"`
}

// OpenAIConfig contains OpenAI specific settings
type OpenAIConfig struct {
	// Base URL for OpenAI API (optional, for custom endpoints)
	BaseURL string `yaml:"base_url" json:"base_url"`

	// Organization ID (optional)
	Organization string `yaml:"organization" json:"organization"`
}

// AzureConfig contains Azure OpenAI specific settings
type AzureConfig struct {
	// Azure OpenAI endpoint
	Endpoint string `yaml:"endpoint" json:"endpoint"`

	// Deployment name
	DeploymentName string `yaml:"deployment_name" json:"deployment_name"`

	// API version
	APIVersion string `yaml:"api_version" json:"api_version"`
}

// ToolsConfig contains tools related configuration
type ToolsConfig struct {
	// Enable/disable specific tools
	Enabled []string `yaml:"enabled" json:"enabled"`

	// Workspace root for file operations
	WorkspaceRoot string `yaml:"workspace_root" json:"workspace_root"`

	// File access restrictions
	FileAccess FileAccessConfig `yaml:"file_access" json:"file_access"`

	// Auto-approval for certain operations
	AutoApprove bool `yaml:"auto_approve" json:"auto_approve"`
}

// FileAccessConfig contains file access restrictions
type FileAccessConfig struct {
	// Allowed paths (glob patterns)
	AllowedPaths []string `yaml:"allowed_paths" json:"allowed_paths"`

	// Denied paths (glob patterns)
	DeniedPaths []string `yaml:"denied_paths" json:"denied_paths"`

	// Maximum file size in bytes
	MaxFileSize int64 `yaml:"max_file_size" json:"max_file_size"`
}

// UIConfig contains UI related configuration
type UIConfig struct {
	// Theme name
	Theme string `yaml:"theme" json:"theme"`

	// Enable/disable syntax highlighting
	SyntaxHighlighting bool `yaml:"syntax_highlighting" json:"syntax_highlighting"`

	// Enable/disable markdown rendering
	MarkdownRendering bool `yaml:"markdown_rendering" json:"markdown_rendering"`

	// Key bindings preset
	KeyBindings string `yaml:"key_bindings" json:"key_bindings"`

	// Input display lines (0 for unlimited)
	InputDisplayLines int `yaml:"input_display_lines" json:"input_display_lines"`
}

// SessionConfig contains session related configuration
type SessionConfig struct {
	// History file path
	HistoryFile string `yaml:"history_file" json:"history_file"`

	// Maximum history entries
	MaxHistory int `yaml:"max_history" json:"max_history"`

	// Auto-save interval in seconds
	AutoSaveInterval int `yaml:"auto_save_interval" json:"auto_save_interval"`
}

// NewDefaultConfig creates a new configuration with default values
func NewDefaultConfig() *Config {
	homeDir, _ := os.UserHomeDir()
	configDir := filepath.Join(homeDir, ".config", "coda")

	return &Config{
		AI: AIConfig{
			Provider:    getEnvOrDefault("CODA_AI_PROVIDER", "openai"),
			APIKey:      os.Getenv("OPENAI_API_KEY"),
			Model:       getEnvOrDefault("CODA_MODEL", "gpt-4"),
			Temperature: 0.7,
			MaxTokens:   4096,
			OpenAI: OpenAIConfig{
				BaseURL:      os.Getenv("OPENAI_BASE_URL"),
				Organization: os.Getenv("OPENAI_ORGANIZATION"),
			},
			Azure: AzureConfig{
				Endpoint:       os.Getenv("AZURE_OPENAI_ENDPOINT"),
				DeploymentName: os.Getenv("AZURE_OPENAI_DEPLOYMENT"),
				APIVersion:     getEnvOrDefault("AZURE_OPENAI_API_VERSION", "2024-02-01"),
			},
		},
		Tools: ToolsConfig{
			Enabled:       []string{"read_file", "write_file", "edit_file", "list_files", "search_files"},
			WorkspaceRoot: getEnvOrDefault("CODA_WORKSPACE", "."),
			FileAccess: FileAccessConfig{
				AllowedPaths: []string{"**/*"},
				DeniedPaths: []string{
					"**/node_modules/**",
					"**/.git/**",
					"**/vendor/**",
					"**/*.exe",
					"**/*.dll",
					"**/*.so",
					"**/*.dylib",
				},
				MaxFileSize: 10 * 1024 * 1024, // 10MB
			},
			AutoApprove: false,
		},
		UI: UIConfig{
			Theme:              "default",
			SyntaxHighlighting: true,
			MarkdownRendering:  true,
			KeyBindings:        "default",
			InputDisplayLines:  0, // 0 = dynamic sizing up to half screen
		},
		Logging: logging.DefaultConfig(),
		Session: SessionConfig{
			HistoryFile:      filepath.Join(configDir, "history.json"),
			MaxHistory:       1000,
			AutoSaveInterval: 30,
		},
	}
}

// Validate validates the configuration
func (c *Config) Validate() error {
	// Validate AI configuration
	if err := c.AI.Validate(); err != nil {
		return fmt.Errorf("AI configuration error: %w", err)
	}

	// Validate Tools configuration
	if err := c.Tools.Validate(); err != nil {
		return fmt.Errorf("Tools configuration error: %w", err)
	}

	// Logging validation is handled by the logging package

	return nil
}

// Validate validates the AI configuration
func (ai *AIConfig) Validate() error {
	if ai.Provider == "" {
		return errors.New("provider is required")
	}

	if ai.Provider != "openai" && ai.Provider != "azure" {
		return fmt.Errorf("invalid provider: %s (must be 'openai' or 'azure')", ai.Provider)
	}

	if ai.APIKey == "" {
		return errors.New("API key is required")
	}

	if ai.Model == "" {
		return errors.New("model is required")
	}

	if ai.Temperature < 0 || ai.Temperature > 2 {
		return fmt.Errorf("temperature must be between 0 and 2, got %f", ai.Temperature)
	}

	if ai.MaxTokens <= 0 {
		return fmt.Errorf("max_tokens must be positive, got %d", ai.MaxTokens)
	}

	// Provider-specific validation
	switch ai.Provider {
	case "azure":
		if ai.Azure.Endpoint == "" {
			return errors.New("Azure endpoint is required")
		}
		if ai.Azure.DeploymentName == "" {
			return errors.New("Azure deployment name is required")
		}
	}
	
	// Validate reasoning effort if specified
	if ai.ReasoningEffort != nil {
		validEfforts := map[string]bool{
			"minimal": true,
			"low":     true,
			"medium":  true,
			"high":    true,
		}
		if !validEfforts[*ai.ReasoningEffort] {
			return fmt.Errorf("invalid reasoning_effort: %s (must be 'minimal', 'low', 'medium', or 'high')", *ai.ReasoningEffort)
		}
	}

	return nil
}

// Validate validates the Tools configuration
func (t *ToolsConfig) Validate() error {
	if t.WorkspaceRoot == "" {
		return errors.New("workspace root is required")
	}

	// Check if workspace root exists
	if _, err := os.Stat(t.WorkspaceRoot); os.IsNotExist(err) {
		return fmt.Errorf("workspace root does not exist: %s", t.WorkspaceRoot)
	}

	if t.FileAccess.MaxFileSize <= 0 {
		return errors.New("max file size must be positive")
	}

	return nil
}

// Helper functions

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
