package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDefaultConfig(t *testing.T) {
	// Save current env vars
	oldProvider := os.Getenv("CODA_AI_PROVIDER")
	oldAPIKey := os.Getenv("OPENAI_API_KEY")
	oldModel := os.Getenv("CODA_MODEL")
	oldLogLevel := os.Getenv("CODA_LOG_LEVEL")

	// Restore env vars after test
	defer func() {
		os.Setenv("CODA_AI_PROVIDER", oldProvider)
		os.Setenv("OPENAI_API_KEY", oldAPIKey)
		os.Setenv("CODA_MODEL", oldModel)
		os.Setenv("CODA_LOG_LEVEL", oldLogLevel)
	}()

	t.Run("default values", func(t *testing.T) {
		os.Unsetenv("CODA_AI_PROVIDER")
		os.Unsetenv("CODA_MODEL")
		os.Unsetenv("CODA_LOG_LEVEL")

		cfg := NewDefaultConfig()

		assert.Equal(t, "openai", cfg.AI.Provider)
		assert.Equal(t, "gpt-4", cfg.AI.Model)
		assert.Equal(t, float32(0.7), cfg.AI.Temperature)
		assert.Equal(t, 4096, cfg.AI.MaxTokens)
		assert.Equal(t, "info", cfg.Logging.Level)
		assert.Equal(t, "text", cfg.Logging.Format)
		assert.True(t, cfg.Logging.Timestamp)
		assert.Equal(t, "default", cfg.UI.Theme)
		assert.True(t, cfg.UI.SyntaxHighlighting)
		assert.True(t, cfg.UI.MarkdownRendering)
		assert.Equal(t, 1000, cfg.Session.MaxHistory)
		assert.Equal(t, 30, cfg.Session.AutoSaveInterval)
		assert.False(t, cfg.Tools.AutoApprove)
		assert.Equal(t, int64(10*1024*1024), cfg.Tools.FileAccess.MaxFileSize)
	})

	t.Run("environment overrides", func(t *testing.T) {
		os.Setenv("CODA_AI_PROVIDER", "azure")
		os.Setenv("OPENAI_API_KEY", "test-key")
		os.Setenv("CODA_MODEL", "gpt-4-turbo")
		os.Setenv("CODA_LOG_LEVEL", "debug")
		os.Setenv("AZURE_OPENAI_ENDPOINT", "https://test.openai.azure.com")
		os.Setenv("AZURE_OPENAI_DEPLOYMENT", "test-deployment")

		cfg := NewDefaultConfig()

		assert.Equal(t, "azure", cfg.AI.Provider)
		assert.Equal(t, "test-key", cfg.AI.APIKey)
		assert.Equal(t, "gpt-4-turbo", cfg.AI.Model)
		assert.Equal(t, "debug", cfg.Logging.Level)
		assert.Equal(t, "https://test.openai.azure.com", cfg.AI.Azure.Endpoint)
		assert.Equal(t, "test-deployment", cfg.AI.Azure.DeploymentName)
	})

	t.Run("default enabled tools", func(t *testing.T) {
		cfg := NewDefaultConfig()

		expectedTools := []string{"read_file", "write_file", "edit_file", "list_files", "search_files"}
		assert.Equal(t, expectedTools, cfg.Tools.Enabled)
	})

	t.Run("default denied paths", func(t *testing.T) {
		cfg := NewDefaultConfig()

		assert.Contains(t, cfg.Tools.FileAccess.DeniedPaths, "**/node_modules/**")
		assert.Contains(t, cfg.Tools.FileAccess.DeniedPaths, "**/.git/**")
		assert.Contains(t, cfg.Tools.FileAccess.DeniedPaths, "**/vendor/**")
		assert.Contains(t, cfg.Tools.FileAccess.DeniedPaths, "**/*.exe")
	})
}

func TestConfigValidate(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		cfg := &Config{
			AI: AIConfig{
				Provider:    "openai",
				APIKey:      "test-key",
				Model:       "gpt-4",
				Temperature: 0.7,
				MaxTokens:   4096,
			},
			Tools: ToolsConfig{
				WorkspaceRoot: ".",
				FileAccess: FileAccessConfig{
					MaxFileSize: 1024,
				},
			},
			Logging: LoggingConfig{
				Level:  "info",
				Format: "text",
			},
		}

		err := cfg.Validate()
		assert.NoError(t, err)
	})

	t.Run("invalid AI provider", func(t *testing.T) {
		cfg := NewDefaultConfig()
		cfg.AI.Provider = "invalid"
		cfg.AI.APIKey = "test-key"

		err := cfg.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid provider")
	})

	t.Run("missing API key", func(t *testing.T) {
		cfg := NewDefaultConfig()
		cfg.AI.APIKey = ""

		err := cfg.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "API key is required")
	})

	t.Run("invalid temperature", func(t *testing.T) {
		cfg := NewDefaultConfig()
		cfg.AI.APIKey = "test-key"
		cfg.AI.Temperature = 3.0

		err := cfg.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "temperature must be between 0 and 2")
	})

	t.Run("invalid max tokens", func(t *testing.T) {
		cfg := NewDefaultConfig()
		cfg.AI.APIKey = "test-key"
		cfg.AI.MaxTokens = 0

		err := cfg.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "max_tokens must be positive")
	})

	t.Run("azure missing endpoint", func(t *testing.T) {
		cfg := NewDefaultConfig()
		cfg.AI.Provider = "azure"
		cfg.AI.APIKey = "test-key"
		cfg.AI.Azure.Endpoint = ""

		err := cfg.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Azure endpoint is required")
	})

	t.Run("azure missing deployment", func(t *testing.T) {
		cfg := NewDefaultConfig()
		cfg.AI.Provider = "azure"
		cfg.AI.APIKey = "test-key"
		cfg.AI.Azure.Endpoint = "https://test.openai.azure.com"
		cfg.AI.Azure.DeploymentName = ""

		err := cfg.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Azure deployment name is required")
	})

	t.Run("invalid log level", func(t *testing.T) {
		cfg := NewDefaultConfig()
		cfg.AI.APIKey = "test-key"
		cfg.Logging.Level = "invalid"

		err := cfg.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid log level")
	})

	t.Run("invalid log format", func(t *testing.T) {
		cfg := NewDefaultConfig()
		cfg.AI.APIKey = "test-key"
		cfg.Logging.Format = "invalid"

		err := cfg.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid log format")
	})

	t.Run("non-existent workspace root", func(t *testing.T) {
		cfg := NewDefaultConfig()
		cfg.AI.APIKey = "test-key"
		cfg.Tools.WorkspaceRoot = "/non/existent/path"

		err := cfg.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "workspace root does not exist")
	})

	t.Run("invalid max file size", func(t *testing.T) {
		cfg := NewDefaultConfig()
		cfg.AI.APIKey = "test-key"
		cfg.Tools.FileAccess.MaxFileSize = 0

		err := cfg.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "max file size must be positive")
	})
}

func TestAIConfigValidate(t *testing.T) {
	t.Run("valid openai config", func(t *testing.T) {
		ai := AIConfig{
			Provider:    "openai",
			APIKey:      "test-key",
			Model:       "gpt-4",
			Temperature: 0.7,
			MaxTokens:   4096,
		}

		err := ai.Validate()
		assert.NoError(t, err)
	})

	t.Run("valid azure config", func(t *testing.T) {
		ai := AIConfig{
			Provider:    "azure",
			APIKey:      "test-key",
			Model:       "gpt-4",
			Temperature: 0.7,
			MaxTokens:   4096,
			Azure: AzureConfig{
				Endpoint:       "https://test.openai.azure.com",
				DeploymentName: "test-deployment",
			},
		}

		err := ai.Validate()
		assert.NoError(t, err)
	})

	t.Run("empty provider", func(t *testing.T) {
		ai := AIConfig{
			APIKey:    "test-key",
			Model:     "gpt-4",
			MaxTokens: 4096,
		}

		err := ai.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "provider is required")
	})

	t.Run("empty model", func(t *testing.T) {
		ai := AIConfig{
			Provider:  "openai",
			APIKey:    "test-key",
			MaxTokens: 4096,
		}

		err := ai.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "model is required")
	})
}

func TestToolsConfigValidate(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()

	t.Run("valid config", func(t *testing.T) {
		tools := ToolsConfig{
			WorkspaceRoot: tempDir,
			FileAccess: FileAccessConfig{
				MaxFileSize: 1024,
			},
		}

		err := tools.Validate()
		assert.NoError(t, err)
	})

	t.Run("empty workspace root", func(t *testing.T) {
		tools := ToolsConfig{
			WorkspaceRoot: "",
			FileAccess: FileAccessConfig{
				MaxFileSize: 1024,
			},
		}

		err := tools.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "workspace root is required")
	})
}

func TestLoggingConfigValidate(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		logging := LoggingConfig{
			Level:  "info",
			Format: "text",
		}

		err := logging.Validate()
		assert.NoError(t, err)
	})

	t.Run("case insensitive level", func(t *testing.T) {
		logging := LoggingConfig{
			Level:  "INFO",
			Format: "text",
		}

		err := logging.Validate()
		assert.NoError(t, err)
	})
}

func TestContains(t *testing.T) {
	slice := []string{"apple", "banana", "orange"}

	assert.True(t, contains(slice, "apple"))
	assert.True(t, contains(slice, "banana"))
	assert.True(t, contains(slice, "orange"))
	assert.False(t, contains(slice, "grape"))
	assert.False(t, contains(slice, ""))
}

func TestGetEnvOrDefault(t *testing.T) {
	// Save current env var
	oldValue := os.Getenv("TEST_ENV_VAR")
	defer os.Setenv("TEST_ENV_VAR", oldValue)

	t.Run("env var exists", func(t *testing.T) {
		os.Setenv("TEST_ENV_VAR", "test-value")
		result := getEnvOrDefault("TEST_ENV_VAR", "default")
		assert.Equal(t, "test-value", result)
	})

	t.Run("env var does not exist", func(t *testing.T) {
		os.Unsetenv("TEST_ENV_VAR")
		result := getEnvOrDefault("TEST_ENV_VAR", "default")
		assert.Equal(t, "default", result)
	})

	t.Run("env var is empty", func(t *testing.T) {
		os.Setenv("TEST_ENV_VAR", "")
		result := getEnvOrDefault("TEST_ENV_VAR", "default")
		assert.Equal(t, "default", result)
	})
}

func TestConfigPaths(t *testing.T) {
	homeDir, err := os.UserHomeDir()
	require.NoError(t, err)

	cfg := NewDefaultConfig()

	expectedHistoryPath := filepath.Join(homeDir, ".config", "coda", "history.json")
	assert.Equal(t, expectedHistoryPath, cfg.Session.HistoryFile)
}
