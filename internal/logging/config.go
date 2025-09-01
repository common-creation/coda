package logging

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	Level     string          `yaml:"level" json:"level"`
	Format    string          `yaml:"format" json:"format"`       // text or json
	Timestamp bool            `yaml:"timestamp" json:"timestamp"` // whether to include timestamps
	Outputs   []OutputConfig  `yaml:"outputs" json:"outputs"`
	Sampling  SamplingConfig  `yaml:"sampling" json:"sampling"`
	Privacy   PrivacyConfig   `yaml:"privacy" json:"privacy"`
	Buffering BufferingConfig `yaml:"buffering" json:"buffering"`
	Rotation  RotationConfig  `yaml:"rotation" json:"rotation"`
}

// OutputConfig configures a log output
type OutputConfig struct {
	Type    string                 `yaml:"type" json:"type"`                         // console, file, json
	Target  string                 `yaml:"target,omitempty" json:"target,omitempty"` // file path for file outputs
	Format  string                 `yaml:"format,omitempty" json:"format,omitempty"` // json, text
	Options map[string]interface{} `yaml:"options,omitempty" json:"options,omitempty"`
}

// PrivacyConfig controls privacy protection in logs
type PrivacyConfig struct {
	Enabled        bool     `yaml:"enabled" json:"enabled"`
	SensitiveKeys  []string `yaml:"sensitive_keys" json:"sensitive_keys"`
	MaskChar       string   `yaml:"mask_char" json:"mask_char"`
	PreserveLength int      `yaml:"preserve_length" json:"preserve_length"`
}

// BufferingConfig controls log buffering behavior
type BufferingConfig struct {
	Enabled       bool          `yaml:"enabled" json:"enabled"`
	Size          int           `yaml:"size" json:"size"`
	FlushLevel    LogLevel      `yaml:"flush_level" json:"flush_level"`
	FlushInterval time.Duration `yaml:"flush_interval" json:"flush_interval"`
}

// RotationConfig controls log file rotation
type RotationConfig struct {
	MaxSize     int64         `yaml:"max_size" json:"max_size"` // bytes
	MaxAge      time.Duration `yaml:"max_age" json:"max_age"`   // duration
	MaxBackups  int           `yaml:"max_backups" json:"max_backups"`
	Compression bool          `yaml:"compression" json:"compression"`
}

// DefaultConfig returns a default logging configuration
func DefaultConfig() LoggingConfig {
	return LoggingConfig{
		Level:     "info",
		Format:    "text",
		Timestamp: true,
		Outputs: []OutputConfig{
			{
				Type:   "console",
				Format: "text",
				Options: map[string]interface{}{
					"colorize": true,
				},
			},
		},
		Sampling: SamplingConfig{
			Enabled: false,
		},
		Privacy: PrivacyConfig{
			Enabled:        true,
			SensitiveKeys:  []string{"api_key", "apikey", "token", "password", "secret", "authorization"},
			MaskChar:       "*",
			PreserveLength: 4,
		},
		Buffering: BufferingConfig{
			Enabled:       true,
			Size:          4096,
			FlushLevel:    LevelError,
			FlushInterval: 5 * time.Second,
		},
		Rotation: RotationConfig{
			MaxSize:     100 * 1024 * 1024,  // 100MB
			MaxAge:      7 * 24 * time.Hour, // 7 days
			MaxBackups:  10,
			Compression: true,
		},
	}
}

// DevelopmentConfig returns a configuration suitable for development
func DevelopmentConfig() LoggingConfig {
	config := DefaultConfig()
	config.Level = "debug"
	config.Outputs = []OutputConfig{
		{
			Type:   "console",
			Format: "text",
			Options: map[string]interface{}{
				"colorize": true,
			},
		},
		{
			Type:   "file",
			Target: "logs/debug.log",
			Format: "json",
		},
	}
	return config
}

// ProductionConfig returns a configuration suitable for production
func ProductionConfig() LoggingConfig {
	config := DefaultConfig()
	config.Level = "info"
	config.Outputs = []OutputConfig{
		{
			Type:   "file",
			Target: "logs/app.log",
			Format: "json",
		},
		{
			Type:   "file",
			Target: "logs/error.log",
			Format: "json",
			Options: map[string]interface{}{
				"min_level": "error",
			},
		},
	}
	config.Sampling = SamplingConfig{
		Enabled:     true,
		Rate:        0.1, // Sample 10% of debug logs
		BurstLimit:  100,
		BurstWindow: time.Minute,
	}
	return config
}

// ConfigureLogger creates and configures a logger from config
func ConfigureLogger(config LoggingConfig) (*Logger, error) {
	// Parse log level
	level, err := parseLogLevel(config.Level)
	if err != nil {
		return nil, fmt.Errorf("invalid log level %q: %w", config.Level, err)
	}

	// Create logger
	logger := NewLogger(level)

	// Configure sampling
	logger.sampling = config.Sampling

	// Configure sanitizer if privacy is enabled
	if config.Privacy.Enabled {
		sanitizer := &DefaultSanitizer{
			sensitiveKeys: config.Privacy.SensitiveKeys,
		}
		logger.SetSanitizer(sanitizer)
	}

	// Clear default outputs and add configured ones
	logger.outputs = []LogOutput{}

	// Configure outputs
	for _, outputConfig := range config.Outputs {
		output, err := createOutput(outputConfig, config)
		if err != nil {
			return nil, fmt.Errorf("failed to create output %q: %w", outputConfig.Type, err)
		}
		logger.AddOutput(output)
	}

	return logger, nil
}

// parseLogLevel parses a string log level
func parseLogLevel(level string) (LogLevel, error) {
	switch strings.ToLower(level) {
	case "debug":
		return LevelDebug, nil
	case "info":
		return LevelInfo, nil
	case "warn", "warning":
		return LevelWarn, nil
	case "error":
		return LevelError, nil
	case "fatal":
		return LevelFatal, nil
	default:
		return LevelInfo, fmt.Errorf("unknown log level: %s", level)
	}
}

// createOutput creates a log output from configuration
func createOutput(config OutputConfig, globalConfig LoggingConfig) (LogOutput, error) {
	switch strings.ToLower(config.Type) {
	case "console":
		colorize := true
		if val, ok := config.Options["colorize"].(bool); ok {
			colorize = val
		}
		return NewConsoleOutput(colorize), nil

	case "file":
		if config.Target == "" {
			return nil, fmt.Errorf("file output requires target path")
		}

		// Ensure directory exists
		dir := filepath.Dir(config.Target)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create log directory %q: %w", dir, err)
		}

		fileConfig := FileOutputConfig{
			Filename:    config.Target,
			MaxSize:     globalConfig.Rotation.MaxSize,
			MaxAge:      globalConfig.Rotation.MaxAge,
			Compression: globalConfig.Rotation.Compression,
			BufferSize:  globalConfig.Buffering.Size,
		}

		return NewFileOutput(fileConfig)

	case "json":
		target := os.Stdout
		if config.Target != "" {
			file, err := os.OpenFile(config.Target, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
			if err != nil {
				return nil, fmt.Errorf("failed to open JSON output file: %w", err)
			}
			target = file
		}
		return NewJSONOutput(target), nil

	default:
		return nil, fmt.Errorf("unknown output type: %s", config.Type)
	}
}

// SetupLogging configures the global logger based on environment
func SetupLogging(environment string, customConfig *LoggingConfig) error {
	var config LoggingConfig

	if customConfig != nil {
		config = *customConfig
	} else {
		switch strings.ToLower(environment) {
		case "development", "dev", "debug":
			config = DevelopmentConfig()
		case "production", "prod", "release":
			config = ProductionConfig()
		default:
			config = DefaultConfig()
		}
	}

	logger, err := ConfigureLogger(config)
	if err != nil {
		return fmt.Errorf("failed to configure logger: %w", err)
	}

	SetDefault(logger)
	return nil
}

// GetEnvironmentFromEnvVar determines logging environment from environment variable
func GetEnvironmentFromEnvVar() string {
	env := os.Getenv("CODA_ENV")
	if env == "" {
		env = os.Getenv("ENV")
	}
	if env == "" {
		env = os.Getenv("ENVIRONMENT")
	}
	if env == "" {
		env = "development" // Default
	}
	return env
}
