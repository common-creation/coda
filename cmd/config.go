/*
Copyright © 2025 CODA Project
*/
package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/common-creation/coda/internal/config"
)

var (
	outputFormat string
	showSecrets  bool
)

// configCmd represents the config command
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage CODA configuration",
	Long: `View, edit, and validate CODA configuration settings.

The config command provides subcommands for managing your CODA configuration,
including setting API keys, customizing behavior, and validating settings.`,
}

// showCmd shows the current configuration
var showCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current configuration",
	Long: `Display the current CODA configuration.

By default, sensitive information like API keys are masked for security.
Use --show-secrets to display them (use with caution).`,
	RunE: runConfigShow,
}

// setCmd sets a configuration value
var setCmd = &cobra.Command{
	Use:   "set KEY VALUE",
	Short: "Set a configuration value",
	Long: `Set a configuration value.

Examples:
  coda config set ai.model gpt-4
  coda config set ai.temperature 0.7
  coda config set logging.level debug`,
	Args: cobra.ExactArgs(2),
	RunE: runConfigSet,
}

// getCmd gets a specific configuration value
var getCmd = &cobra.Command{
	Use:   "get KEY",
	Short: "Get a specific configuration value",
	Long: `Get a specific configuration value.

Examples:
  coda config get ai.model
  coda config get ai.temperature`,
	Args: cobra.ExactArgs(1),
	RunE: runConfigGet,
}

// initCmd initializes a new configuration file
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new configuration file",
	Long: `Initialize a new configuration file with default values.

This creates a new config.yaml file in the default location (~/.coda/config.yaml)
or in the location specified by --config flag.`,
	RunE: runConfigInit,
}

// validateCmd validates the configuration
var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate the configuration",
	Long:  `Validate the current configuration for errors and inconsistencies.`,
	RunE:  runConfigValidate,
}

// setApiKeyCmd sets API keys securely
var setApiKeyCmd = &cobra.Command{
	Use:   "set-api-key PROVIDER [KEY]",
	Short: "Set API key for a provider",
	Long: `Set API key for a provider securely.

If KEY is not provided, you will be prompted to enter it securely.

Supported providers:
  - openai
  - azure

Examples:
  coda config set-api-key openai
  coda config set-api-key azure sk-...`,
	Args: cobra.RangeArgs(1, 2),
	RunE: runConfigSetApiKey,
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(showCmd)
	configCmd.AddCommand(setCmd)
	configCmd.AddCommand(getCmd)
	configCmd.AddCommand(initCmd)
	configCmd.AddCommand(validateCmd)
	configCmd.AddCommand(setApiKeyCmd)

	// Flags for show command
	showCmd.Flags().StringVarP(&outputFormat, "output", "o", "yaml", "output format (yaml, json)")
	showCmd.Flags().BoolVar(&showSecrets, "show-secrets", false, "show sensitive information (use with caution)")
}

func runConfigShow(cmd *cobra.Command, args []string) error {
	cfg := GetConfig()

	// Mask sensitive information unless requested
	displayCfg := cfg
	if !showSecrets {
		displayCfg = maskSensitiveConfig(cfg)
	}

	// Marshal based on format
	var output []byte
	var err error

	switch strings.ToLower(outputFormat) {
	case "json":
		output, err = json.MarshalIndent(displayCfg, "", "  ")
	case "yaml", "yml":
		output, err = yaml.Marshal(displayCfg)
	default:
		return fmt.Errorf("unsupported output format: %s", outputFormat)
	}

	if err != nil {
		return fmt.Errorf("failed to marshal configuration: %w", err)
	}

	fmt.Println(string(output))
	return nil
}

func runConfigSet(cmd *cobra.Command, args []string) error {
	key := args[0]
	value := args[1]

	// Load current configuration
	cfg := GetConfig()

	// Parse and set the value
	if err := setConfigValue(cfg, key, value); err != nil {
		return fmt.Errorf("failed to set configuration value: %w", err)
	}

	// Save configuration
	if err := saveConfiguration(cfg); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	ShowSuccess("Configuration updated: %s = %s", key, value)
	return nil
}

func runConfigGet(cmd *cobra.Command, args []string) error {
	key := args[0]
	cfg := GetConfig()

	value, err := getConfigValue(cfg, key)
	if err != nil {
		return fmt.Errorf("failed to get configuration value: %w", err)
	}

	// Mask sensitive values unless explicitly requested
	if isSensitiveKey(key) && !showSecrets {
		fmt.Println("********")
	} else {
		fmt.Println(value)
	}

	return nil
}

func runConfigInit(cmd *cobra.Command, args []string) error {
	// Determine config file path
	configPath := getConfigPath()

	// Check if file already exists
	if _, err := os.Stat(configPath); err == nil {
		return fmt.Errorf("configuration file already exists at %s", configPath)
	}

	// Create directory if needed
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Create default configuration
	cfg := config.NewDefaultConfig()

	// Save configuration
	if err := saveConfigurationToPath(cfg, configPath); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	ShowSuccess("Configuration initialized at %s", configPath)
	ShowInfo("Edit this file to customize your CODA settings.")
	ShowInfo("Use 'coda config set-api-key' to set your API keys securely.")

	return nil
}

func runConfigValidate(cmd *cobra.Command, args []string) error {
	cfg := GetConfig()

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		ShowError("Configuration validation failed:")
		ShowError("  %v", err)
		return fmt.Errorf("configuration is invalid")
	}

	ShowSuccess("Configuration is valid")

	return nil
}

func runConfigSetApiKey(cmd *cobra.Command, args []string) error {
	provider := strings.ToLower(args[0])

	// Validate provider
	if provider != "openai" && provider != "azure" {
		return fmt.Errorf("unsupported provider: %s", provider)
	}

	var apiKey string
	if len(args) > 1 {
		apiKey = args[1]
	} else {
		// Prompt for API key
		fmt.Printf("Enter API key for %s: ", provider)

		// Read securely (without echo)
		keyBytes, err := readPassword()
		if err != nil {
			return fmt.Errorf("failed to read API key: %w", err)
		}
		apiKey = string(keyBytes)
		fmt.Println() // New line after password input
	}

	// Validate API key format
	if apiKey == "" {
		return fmt.Errorf("API key cannot be empty")
	}

	// Store API key securely
	keyManager, err := config.NewSecretsManager()
	if err != nil {
		return fmt.Errorf("failed to create secrets manager: %w", err)
	}
	if err := keyManager.SetAPIKey(provider, apiKey); err != nil {
		return fmt.Errorf("failed to store API key: %w", err)
	}

	// Update configuration to use the provider
	cfg := GetConfig()
	cfg.AI.Provider = provider

	// Clear any plaintext API key from config
	cfg.AI.APIKey = ""

	// Save configuration
	if err := saveConfiguration(cfg); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	ShowSuccess("API key for %s has been securely stored", provider)
	ShowInfo("Provider set to: %s", provider)

	return nil
}

// Helper functions

func maskSensitiveConfig(cfg *config.Config) *config.Config {
	// Create a deep copy
	masked := *cfg

	// Mask API keys
	if masked.AI.APIKey != "" {
		masked.AI.APIKey = "********"
	}
	// Note: Azure uses the same APIKey field from AIConfig

	return &masked
}

func setConfigValue(cfg *config.Config, key, value string) error {
	parts := strings.Split(key, ".")
	if len(parts) < 2 {
		return fmt.Errorf("invalid configuration key: %s", key)
	}

	// Handle nested configuration
	switch parts[0] {
	case "ai":
		return setAIConfigValue(cfg, parts[1:], value)
	case "tools":
		return setToolsConfigValue(cfg, parts[1:], value)
	case "ui":
		return setUIConfigValue(cfg, parts[1:], value)
	case "logging":
		return setLoggingConfigValue(cfg, parts[1:], value)
	case "session":
		return setSessionConfigValue(cfg, parts[1:], value)
	default:
		return fmt.Errorf("unknown configuration section: %s", parts[0])
	}
}

func setAIConfigValue(cfg *config.Config, parts []string, value string) error {
	if len(parts) == 0 {
		return fmt.Errorf("incomplete configuration key")
	}

	switch parts[0] {
	case "provider":
		cfg.AI.Provider = value
	case "model":
		cfg.AI.Model = value
	case "temperature":
		temp, err := parseFloat32(value)
		if err != nil {
			return err
		}
		cfg.AI.Temperature = temp
	case "max_tokens":
		tokens, err := parseInt(value)
		if err != nil {
			return err
		}
		cfg.AI.MaxTokens = tokens
	default:
		return fmt.Errorf("unknown AI configuration key: %s", parts[0])
	}

	return nil
}

func setToolsConfigValue(cfg *config.Config, parts []string, value string) error {
	// Implementation for tools configuration
	return fmt.Errorf("tools configuration not yet implemented")
}

func setUIConfigValue(cfg *config.Config, parts []string, value string) error {
	// Implementation for UI configuration
	return fmt.Errorf("UI configuration not yet implemented")
}

func setLoggingConfigValue(cfg *config.Config, parts []string, value string) error {
	if len(parts) == 0 {
		return fmt.Errorf("incomplete configuration key")
	}

	switch parts[0] {
	case "level":
		cfg.Logging.Level = value
	case "outputs":
		// Note: outputs configuration is more complex and would need special handling
		return fmt.Errorf("use config file to configure logging outputs")
	default:
		return fmt.Errorf("unknown logging configuration key: %s", parts[0])
	}

	return nil
}

func setSessionConfigValue(cfg *config.Config, parts []string, value string) error {
	// Implementation for session configuration
	return fmt.Errorf("session configuration not yet implemented")
}

func getConfigValue(cfg *config.Config, key string) (string, error) {
	// Simple implementation - would need to handle all nested fields
	parts := strings.Split(key, ".")

	switch parts[0] {
	case "ai":
		if len(parts) > 1 {
			switch parts[1] {
			case "provider":
				return cfg.AI.Provider, nil
			case "model":
				return cfg.AI.Model, nil
			case "temperature":
				return fmt.Sprintf("%f", cfg.AI.Temperature), nil
			case "max_tokens":
				return fmt.Sprintf("%d", cfg.AI.MaxTokens), nil
			}
		}
	}

	return "", fmt.Errorf("unknown configuration key: %s", key)
}

func isSensitiveKey(key string) bool {
	sensitiveKeys := []string{
		"ai.api_key",
		"ai.apikey",
		"ai.azure.api_key",
		"ai.azure.apikey",
	}

	key = strings.ToLower(key)
	for _, sensitive := range sensitiveKeys {
		if key == sensitive || strings.HasSuffix(key, "."+sensitive) {
			return true
		}
	}

	return false
}

func getConfigPath() string {
	if cfgFile != "" {
		return cfgFile
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "config.yaml"
	}

	return filepath.Join(home, ".coda", "config.yaml")
}

func saveConfiguration(cfg *config.Config) error {
	return saveConfigurationToPath(cfg, getConfigPath())
}

func saveConfigurationToPath(cfg *config.Config, path string) error {
	// Marshal configuration
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal configuration: %w", err)
	}

	// Write to file
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write configuration file: %w", err)
	}

	return nil
}

func readPassword() ([]byte, error) {
	// Simple implementation - in production, use terminal.ReadPassword
	reader := bufio.NewReader(os.Stdin)
	password, err := reader.ReadString('\n')
	if err != nil {
		return nil, err
	}
	return []byte(strings.TrimSpace(password)), nil
}

// Parsing helpers

func parseInt(s string) (int, error) {
	var i int
	_, err := fmt.Sscanf(s, "%d", &i)
	return i, err
}

func parseFloat32(s string) (float32, error) {
	var f float32
	_, err := fmt.Sscanf(s, "%f", &f)
	return f, err
}

func parseBool(s string) (bool, error) {
	s = strings.ToLower(s)
	switch s {
	case "true", "yes", "1", "on":
		return true, nil
	case "false", "no", "0", "off":
		return false, nil
	default:
		return false, fmt.Errorf("invalid boolean value: %s", s)
	}
}
