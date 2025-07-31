package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
)

// Config represents the application configuration
type Config struct {
	AppName     string `json:"app_name"`
	Version     string `json:"version"`
	Environment string `json:"environment"`
	Settings    struct {
		MaxRetries int  `json:"max_retries"`
		Debug      bool `json:"debug"`
	} `json:"settings"`
}

// LoadConfig loads configuration from a JSON file
func LoadConfig(filename string) (*Config, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return &config, nil
}

// WriteFile writes content to a file
func WriteFile(filename, content string) error {
	return ioutil.WriteFile(filename, []byte(content), 0644)
}

// ReadFile reads content from a file
func ReadFile(filename string) (string, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// ProcessData processes data from the data file
func ProcessData(filename string) ([]string, error) {
	content, err := ReadFile(filename)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(content, "\n")
	var processed []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			processed = append(processed, strings.ToUpper(line))
		}
	}

	return processed, nil
}

// FileExists checks if a file exists
func FileExists(filename string) bool {
	_, err := os.Stat(filename)
	return !os.IsNotExist(err)
}

// CreateBackup creates a backup of a file
func CreateBackup(filename string) error {
	if !FileExists(filename) {
		return fmt.Errorf("file does not exist: %s", filename)
	}

	content, err := ReadFile(filename)
	if err != nil {
		return err
	}

	backupName := filename + ".backup"
	return WriteFile(backupName, content)
}

// ValidateName validates a name input
func ValidateName(name string) error {
	if name == "" {
		return fmt.Errorf("name cannot be empty")
	}
	if len(name) > 100 {
		return fmt.Errorf("name too long (max 100 characters)")
	}
	if strings.Contains(name, "\n") || strings.Contains(name, "\t") {
		return fmt.Errorf("name cannot contain newlines or tabs")
	}
	return nil
}
