package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// SecretsManager provides secure storage for API keys and other secrets
type SecretsManager interface {
	GetAPIKey(provider string) (string, error)
	SetAPIKey(provider string, key string) error
	DeleteAPIKey(provider string) error
	ListProviders() ([]string, error)
}

// secretsStore represents the internal storage structure
type secretsStore struct {
	Keys map[string]string `json:"keys"`
}

// FileSecretsManager implements file-based secrets storage
type FileSecretsManager struct {
	filePath string
	mu       sync.RWMutex
}

// PlatformSecretsManager provides platform-specific secure storage
type PlatformSecretsManager struct {
	fallback SecretsManager
}

// NewSecretsManager creates a new secrets manager with platform-specific storage
func NewSecretsManager() (SecretsManager, error) {
	// Try to use platform-specific secure storage
	platformManager := &PlatformSecretsManager{}

	// Create file-based fallback
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	secretsPath := filepath.Join(homeDir, ".config", "coda", ".secrets")
	fallback, err := NewFileSecretsManager(secretsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create fallback secrets manager: %w", err)
	}

	platformManager.fallback = fallback

	// Check if platform-specific storage is available
	if !isPlatformStorageAvailable() {
		// Log warning about using file-based storage
		fmt.Fprintf(os.Stderr, "Warning: Using file-based secrets storage. Consider using your platform's secure credential storage.\n")
		return fallback, nil
	}

	return platformManager, nil
}

// NewFileSecretsManager creates a new file-based secrets manager
func NewFileSecretsManager(filePath string) (*FileSecretsManager, error) {
	// Ensure directory exists
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create secrets directory: %w", err)
	}

	return &FileSecretsManager{
		filePath: filePath,
	}, nil
}

// GetAPIKey retrieves an API key for the specified provider
func (f *FileSecretsManager) GetAPIKey(provider string) (string, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	// First check environment variables
	if key := getAPIKeyFromEnv(provider); key != "" {
		return key, nil
	}

	// Load from file
	store, err := f.load()
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("no API key found for provider: %s", provider)
		}
		return "", err
	}

	key, exists := store.Keys[provider]
	if !exists {
		return "", fmt.Errorf("no API key found for provider: %s", provider)
	}

	return key, nil
}

// SetAPIKey stores an API key for the specified provider
func (f *FileSecretsManager) SetAPIKey(provider string, key string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	// Load existing store
	store, err := f.load()
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	if store == nil {
		store = &secretsStore{
			Keys: make(map[string]string),
		}
	}

	// Update key
	store.Keys[provider] = key

	// Save to file
	return f.save(store)
}

// DeleteAPIKey removes an API key for the specified provider
func (f *FileSecretsManager) DeleteAPIKey(provider string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	// Load existing store
	store, err := f.load()
	if err != nil {
		if os.IsNotExist(err) {
			return nil // Nothing to delete
		}
		return err
	}

	// Delete key
	delete(store.Keys, provider)

	// Save to file
	return f.save(store)
}

// ListProviders returns a list of providers with stored API keys
func (f *FileSecretsManager) ListProviders() ([]string, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	// Load from file
	store, err := f.load()
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}

	providers := make([]string, 0, len(store.Keys))
	for provider := range store.Keys {
		providers = append(providers, provider)
	}

	// Also check environment variables
	envProviders := getProvidersFromEnv()
	for _, provider := range envProviders {
		found := false
		for _, p := range providers {
			if p == provider {
				found = true
				break
			}
		}
		if !found {
			providers = append(providers, provider)
		}
	}

	return providers, nil
}

// load reads the secrets file
func (f *FileSecretsManager) load() (*secretsStore, error) {
	data, err := os.ReadFile(f.filePath)
	if err != nil {
		return nil, err
	}

	var store secretsStore
	if err := json.Unmarshal(data, &store); err != nil {
		return nil, fmt.Errorf("failed to parse secrets file: %w", err)
	}

	if store.Keys == nil {
		store.Keys = make(map[string]string)
	}

	return &store, nil
}

// save writes the secrets file with restricted permissions
func (f *FileSecretsManager) save(store *secretsStore) error {
	data, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal secrets: %w", err)
	}

	// Write with restricted permissions
	if err := os.WriteFile(f.filePath, data, 0600); err != nil {
		return fmt.Errorf("failed to write secrets file: %w", err)
	}

	return nil
}

// Platform-specific implementations

// GetAPIKey for platform-specific manager
func (p *PlatformSecretsManager) GetAPIKey(provider string) (string, error) {
	// First check environment variables
	if key := getAPIKeyFromEnv(provider); key != "" {
		return key, nil
	}

	// Try platform-specific storage
	key, err := getPlatformAPIKey(provider)
	if err == nil && key != "" {
		return key, nil
	}

	// Fall back to file-based storage
	return p.fallback.GetAPIKey(provider)
}

// SetAPIKey for platform-specific manager
func (p *PlatformSecretsManager) SetAPIKey(provider string, key string) error {
	// Try platform-specific storage
	if err := setPlatformAPIKey(provider, key); err == nil {
		return nil
	}

	// Fall back to file-based storage
	return p.fallback.SetAPIKey(provider, key)
}

// DeleteAPIKey for platform-specific manager
func (p *PlatformSecretsManager) DeleteAPIKey(provider string) error {
	// Try platform-specific storage
	if err := deletePlatformAPIKey(provider); err == nil {
		return nil
	}

	// Fall back to file-based storage
	return p.fallback.DeleteAPIKey(provider)
}

// ListProviders for platform-specific manager
func (p *PlatformSecretsManager) ListProviders() ([]string, error) {
	providers := []string{}

	// Get from platform storage
	platformProviders, _ := listPlatformProviders()
	providers = append(providers, platformProviders...)

	// Get from fallback
	fallbackProviders, err := p.fallback.ListProviders()
	if err != nil {
		return nil, err
	}

	// Merge and deduplicate
	providerMap := make(map[string]bool)
	for _, p := range providers {
		providerMap[p] = true
	}
	for _, p := range fallbackProviders {
		providerMap[p] = true
	}

	result := make([]string, 0, len(providerMap))
	for p := range providerMap {
		result = append(result, p)
	}

	return result, nil
}

// Helper functions

// getAPIKeyFromEnv checks environment variables for API keys
func getAPIKeyFromEnv(provider string) string {
	// Check provider-specific env var
	switch provider {
	case "openai":
		if key := os.Getenv("OPENAI_API_KEY"); key != "" {
			return key
		}
	case "azure":
		if key := os.Getenv("AZURE_OPENAI_API_KEY"); key != "" {
			return key
		}
		if key := os.Getenv("AZURE_OPENAI_KEY"); key != "" {
			return key
		}
	}

	// Check generic CODA env var
	if key := os.Getenv("CODA_API_KEY"); key != "" {
		return key
	}

	// Check CODA provider-specific env var
	envKey := fmt.Sprintf("CODA_%s_API_KEY", provider)
	return os.Getenv(envKey)
}

// getProvidersFromEnv returns providers that have API keys in environment variables
func getProvidersFromEnv() []string {
	providers := []string{}

	if os.Getenv("OPENAI_API_KEY") != "" {
		providers = append(providers, "openai")
	}
	if os.Getenv("AZURE_OPENAI_API_KEY") != "" || os.Getenv("AZURE_OPENAI_KEY") != "" {
		providers = append(providers, "azure")
	}

	return providers
}

// Platform-specific functions are implemented in platform-specific files

// GetServiceName returns the service name for keychain/credential storage
func GetServiceName() string {
	return "com.common-creation.coda"
}

// ValidateAPIKey performs basic validation on an API key
func ValidateAPIKey(key string) error {
	if key == "" {
		return errors.New("API key cannot be empty")
	}

	if len(key) < 10 {
		return errors.New("API key is too short")
	}

	// Check for common placeholder values
	placeholders := []string{
		"your-api-key",
		"your_api_key",
		"YOUR_API_KEY",
		"sk-...",
		"xxx",
		"XXX",
		"placeholder",
		"PLACEHOLDER",
	}

	for _, placeholder := range placeholders {
		if key == placeholder {
			return fmt.Errorf("API key appears to be a placeholder: %s", key)
		}
	}

	return nil
}
