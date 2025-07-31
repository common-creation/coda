//go:build darwin
// +build darwin

package config

import (
	"fmt"
	"os/exec"
	"strings"
)

// macOS-specific implementation using the Keychain

// isPlatformStorageAvailable checks if macOS Keychain is available
func isPlatformStorageAvailable() bool {
	// Check if security command is available
	_, err := exec.LookPath("security")
	return err == nil
}

// getPlatformAPIKey retrieves API key from macOS Keychain
func getPlatformAPIKey(provider string) (string, error) {
	service := GetServiceName()
	account := fmt.Sprintf("%s-%s", service, provider)

	cmd := exec.Command("security", "find-generic-password",
		"-s", service,
		"-a", account,
		"-w") // -w returns only the password

	output, err := cmd.Output()
	if err != nil {
		// Check if it's just not found
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.ExitCode() == 44 { // Item not found
				return "", fmt.Errorf("API key not found for provider: %s", provider)
			}
		}
		return "", fmt.Errorf("failed to retrieve API key from Keychain: %w", err)
	}

	// Trim newline from output
	key := strings.TrimSpace(string(output))
	return key, nil
}

// setPlatformAPIKey stores API key in macOS Keychain
func setPlatformAPIKey(provider string, key string) error {
	service := GetServiceName()
	account := fmt.Sprintf("%s-%s", service, provider)

	// First, try to delete existing entry (ignore errors)
	_ = deletePlatformAPIKey(provider)

	// Add new entry
	cmd := exec.Command("security", "add-generic-password",
		"-s", service,
		"-a", account,
		"-w", key,
		"-U") // Update if exists

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to store API key in Keychain: %w", err)
	}

	return nil
}

// deletePlatformAPIKey removes API key from macOS Keychain
func deletePlatformAPIKey(provider string) error {
	service := GetServiceName()
	account := fmt.Sprintf("%s-%s", service, provider)

	cmd := exec.Command("security", "delete-generic-password",
		"-s", service,
		"-a", account)

	if err := cmd.Run(); err != nil {
		// Check if it's just not found
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.ExitCode() == 44 { // Item not found
				return nil // Nothing to delete
			}
		}
		return fmt.Errorf("failed to delete API key from Keychain: %w", err)
	}

	return nil
}

// listPlatformProviders lists providers stored in macOS Keychain
func listPlatformProviders() ([]string, error) {
	service := GetServiceName()

	// Use security dump-keychain to list items
	cmd := exec.Command("security", "dump-keychain")
	output, err := cmd.Output()
	if err != nil {
		// If we can't list, just return empty
		return []string{}, nil
	}

	// Parse output to find matching service entries
	providers := []string{}
	lines := strings.Split(string(output), "\n")

	for i, line := range lines {
		if strings.Contains(line, fmt.Sprintf("\"svce\"<blob>=\"%s\"", service)) {
			// Look for account line in nearby lines
			for j := i - 5; j <= i+5 && j >= 0 && j < len(lines); j++ {
				if strings.Contains(lines[j], "\"acct\"<blob>=") {
					// Extract account name
					parts := strings.Split(lines[j], "\"")
					if len(parts) >= 4 {
						account := parts[3]
						// Remove service prefix
						prefix := fmt.Sprintf("%s-", service)
						if strings.HasPrefix(account, prefix) {
							provider := strings.TrimPrefix(account, prefix)
							providers = append(providers, provider)
						}
					}
				}
			}
		}
	}

	// Deduplicate
	seen := make(map[string]bool)
	unique := []string{}
	for _, p := range providers {
		if !seen[p] {
			seen[p] = true
			unique = append(unique, p)
		}
	}

	return unique, nil
}
