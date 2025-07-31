//go:build linux
// +build linux

package config

import (
	"errors"
)

// Linux-specific implementation
// TODO: Implement Secret Service API integration (libsecret)
// For now, we'll use file-based storage as fallback

// isPlatformStorageAvailable checks if Secret Service is available
func isPlatformStorageAvailable() bool {
	// TODO: Check for Secret Service availability
	// For now, always use file-based storage
	return false
}

// getPlatformAPIKey retrieves API key from Secret Service
func getPlatformAPIKey(provider string) (string, error) {
	// TODO: Implement Secret Service retrieval
	return "", errors.New("Secret Service not implemented yet")
}

// setPlatformAPIKey stores API key in Secret Service
func setPlatformAPIKey(provider string, key string) error {
	// TODO: Implement Secret Service storage
	return errors.New("Secret Service not implemented yet")
}

// deletePlatformAPIKey removes API key from Secret Service
func deletePlatformAPIKey(provider string) error {
	// TODO: Implement Secret Service deletion
	return errors.New("Secret Service not implemented yet")
}

// listPlatformProviders lists providers in Secret Service
func listPlatformProviders() ([]string, error) {
	// TODO: Implement Secret Service listing
	return []string{}, nil
}
