package config

import (
	"fmt"

	"github.com/common-creation/coda/internal/logging"
)

// SetupLogging initializes the logging system from the configuration
func (c *Config) SetupLogging() error {
	logger, err := logging.ConfigureLogger(c.Logging)
	if err != nil {
		return fmt.Errorf("failed to configure logging: %w", err)
	}

	// Set as the default logger
	logging.SetDefault(logger)

	// Log initialization
	logging.InfoWith("Logging system initialized", logging.Fields{
		"level":     c.Logging.Level,
		"outputs":   len(c.Logging.Outputs),
		"privacy":   c.Logging.Privacy.Enabled,
		"sampling":  c.Logging.Sampling.Enabled,
		"buffering": c.Logging.Buffering.Enabled,
	})

	return nil
}

// GetLoggerWithContext creates a logger with configuration context
func (c *Config) GetLoggerWithContext() *logging.Logger {
	return logging.GetDefault().With(logging.Fields{
		"component": "config",
		"workspace": c.Tools.WorkspaceRoot,
		"provider":  c.AI.Provider,
		"model":     c.AI.Model,
	})
}
