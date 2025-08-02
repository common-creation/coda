/*
Copyright © 2025 CODA Project

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/common-creation/coda/internal/config"
)

var (
	cfgFile    string
	debugMode  bool
	quiet      bool
	noColor    bool
	workingDir string
	cfg        *config.Config
	
	// Version information
	appVersion string
	appCommit  string
	appDate    string
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "coda",
	Short: "CODA - AI-powered coding assistant",
	Long: `CODA is an intelligent coding assistant that helps you write, 
understand, and manage code through natural language interaction.

It provides:
- Interactive chat with AI models
- File operations and code analysis
- Project context awareness
- Tool integration for enhanced productivity`,
	RunE: runRoot,
}

// SetVersion sets the version information for the application
func SetVersion(version, commit, date string) {
	appVersion = version
	appCommit = commit
	appDate = date
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.coda/config.yaml)")
	rootCmd.PersistentFlags().BoolVar(&debugMode, "debug", false, "enable debug mode")
	rootCmd.PersistentFlags().BoolVar(&quiet, "quiet", false, "suppress non-essential output")
	rootCmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "disable colored output")
	rootCmd.PersistentFlags().StringVar(&workingDir, "working-dir", "", "set working directory")

	// Add chat-related flags to root command for direct chat invocation
	rootCmd.Flags().StringVar(&model, "model", "", "AI model to use (overrides config)")
	rootCmd.Flags().BoolVar(&noStream, "no-stream", false, "disable streaming responses")
	rootCmd.Flags().StringVar(&sessionID, "session", "", "specify session ID to load")
	rootCmd.Flags().BoolVar(&continueSession, "continue", false, "continue last session")
	rootCmd.Flags().BoolVar(&noTools, "no-tools", false, "disable tool execution")
	rootCmd.Flags().BoolVar(&autoApprove, "auto-approve", false, "auto-approve all tool executions (use with caution)")

	// Bind flags to viper
	viper.BindPFlag("debug", rootCmd.PersistentFlags().Lookup("debug"))
	viper.BindPFlag("quiet", rootCmd.PersistentFlags().Lookup("quiet"))
	viper.BindPFlag("no_color", rootCmd.PersistentFlags().Lookup("no-color"))
	viper.BindPFlag("working_dir", rootCmd.PersistentFlags().Lookup("working-dir"))

	// Environment variable support
	viper.SetEnvPrefix("CODA")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	// Load configuration
	var err error
	cfg, err = loadConfiguration()
	if err != nil {
		if !quiet {
			fmt.Fprintf(os.Stderr, "Warning: Failed to load configuration: %v\n", err)
		}
		// Use default configuration
		cfg = config.NewDefaultConfig()
	}

	// Apply command line overrides
	if debugMode {
		cfg.Logging.Level = "debug"
		// Note: Verbose field doesn't exist in LoggingConfig
	}

	// Set working directory
	if workingDir != "" {
		if err := os.Chdir(workingDir); err != nil {
			fmt.Fprintf(os.Stderr, "Error: Failed to change directory to %s: %v\n", workingDir, err)
			os.Exit(1)
		}
	}

	// Initialize logging
	if err := initializeLogging(cfg); err != nil {
		if !quiet {
			fmt.Fprintf(os.Stderr, "Warning: Failed to initialize logging: %v\n", err)
		}
	}

	// Disable color if requested
	if noColor || os.Getenv("NO_COLOR") != "" {
		disableColors()
	}
}

func loadConfiguration() (*config.Config, error) {
	if cfgFile != "" {
		// Use config file from the flag
		viper.SetConfigFile(cfgFile)
	} else {
		// Search for config in standard locations
		home, err := os.UserHomeDir()
		if err == nil {
			viper.AddConfigPath(filepath.Join(home, ".coda"))
			viper.AddConfigPath(home)
		}

		// Current directory
		viper.AddConfigPath(".")

		// Config file name
		viper.SetConfigType("yaml")
		viper.SetConfigName("config")
		viper.SetConfigName(".coda")
	}

	// Read config file
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, err
		}
		// Config file not found is not an error, we'll use defaults
	}

	// Load configuration
	loader := config.NewLoader()

	// Load config (handles both file and environment variables)
	if viper.ConfigFileUsed() != "" {
		return loader.Load(viper.ConfigFileUsed())
	}

	// Load with empty path (will search default paths and use env vars)
	return loader.Load("")
}

func initializeLogging(cfg *config.Config) error {
	// This would initialize the logging system based on configuration
	// For now, it's a placeholder
	return nil
}

func disableColors() {
	// Disable colored output
	os.Setenv("NO_COLOR", "1")
}

// runRoot is executed when no subcommands are provided
func runRoot(cmd *cobra.Command, args []string) error {
	// Check if help flag is set
	helpFlag, _ := cmd.Flags().GetBool("help")
	if helpFlag {
		return cmd.Help()
	}
	
	// If any arguments are provided, or if we should start chat by default,
	// run the chat command directly
	if len(args) > 0 || shouldStartChatByDefault() {
		// Execute chat command with the provided arguments
		return runChat(cmd, args)
	}
	
	// Default behavior: show help
	return cmd.Help()
}

// shouldStartChatByDefault determines if chat should start when no subcommands are provided
func shouldStartChatByDefault() bool {
	// For now, always start chat when no subcommand is provided
	// This could be configurable in the future
	return true
}

// GetConfig returns the loaded configuration
func GetConfig() *config.Config {
	if cfg == nil {
		cfg = config.NewDefaultConfig()
	}
	return cfg
}

// IsDebug returns whether debug mode is enabled
func IsDebug() bool {
	return debugMode || viper.GetBool("debug")
}

// IsQuiet returns whether quiet mode is enabled
func IsQuiet() bool {
	return quiet || viper.GetBool("quiet")
}

// ShowError displays an error message to the user
func ShowError(format string, args ...interface{}) {
	if !IsQuiet() {
		msg := fmt.Sprintf(format, args...)
		if !noColor && os.Getenv("NO_COLOR") == "" {
			fmt.Fprintf(os.Stderr, "\033[31mError: %s\033[0m\n", msg)
		} else {
			fmt.Fprintf(os.Stderr, "Error: %s\n", msg)
		}
	}
}

// ShowWarning displays a warning message to the user
func ShowWarning(format string, args ...interface{}) {
	if !IsQuiet() {
		msg := fmt.Sprintf(format, args...)
		if !noColor && os.Getenv("NO_COLOR") == "" {
			fmt.Fprintf(os.Stderr, "\033[33mWarning: %s\033[0m\n", msg)
		} else {
			fmt.Fprintf(os.Stderr, "Warning: %s\n", msg)
		}
	}
}

// ShowInfo displays an informational message to the user
func ShowInfo(format string, args ...interface{}) {
	if !IsQuiet() {
		msg := fmt.Sprintf(format, args...)
		fmt.Println(msg)
	}
}

// ShowSuccess displays a success message to the user
func ShowSuccess(format string, args ...interface{}) {
	if !IsQuiet() {
		msg := fmt.Sprintf(format, args...)
		if !noColor && os.Getenv("NO_COLOR") == "" {
			fmt.Printf("\033[32m✓ %s\033[0m\n", msg)
		} else {
			fmt.Printf("✓ %s\n", msg)
		}
	}
}

// ExitWithError prints an error and exits with status 1
func ExitWithError(format string, args ...interface{}) {
	ShowError(format, args...)
	os.Exit(1)
}

// CheckError checks if an error occurred and exits if so
func CheckError(err error, message string) {
	if err != nil {
		if message != "" {
			ExitWithError("%s: %v", message, err)
		} else {
			ExitWithError("%v", err)
		}
	}
}
