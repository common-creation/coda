package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFileSecretsManager(t *testing.T) {
	tempDir := t.TempDir()
	secretsPath := filepath.Join(tempDir, ".secrets")

	manager, err := NewFileSecretsManager(secretsPath)
	if err != nil {
		t.Fatalf("Failed to create file secrets manager: %v", err)
	}

	t.Run("set and get API key", func(t *testing.T) {
		provider := "openai"
		testKey := "sk-test-key-123456789"

		// Set key
		err := manager.SetAPIKey(provider, testKey)
		if err != nil {
			t.Fatalf("Failed to set API key: %v", err)
		}

		// Get key
		retrievedKey, err := manager.GetAPIKey(provider)
		if err != nil {
			t.Fatalf("Failed to get API key: %v", err)
		}

		if retrievedKey != testKey {
			t.Errorf("Expected key %s, got %s", testKey, retrievedKey)
		}
	})

	t.Run("get non-existent key", func(t *testing.T) {
		_, err := manager.GetAPIKey("non-existent")
		if err == nil {
			t.Error("Expected error for non-existent key, got nil")
		}
	})

	t.Run("delete API key", func(t *testing.T) {
		provider := "azure"
		testKey := "azure-test-key"

		// Set key
		err := manager.SetAPIKey(provider, testKey)
		if err != nil {
			t.Fatalf("Failed to set API key: %v", err)
		}

		// Delete key
		err = manager.DeleteAPIKey(provider)
		if err != nil {
			t.Fatalf("Failed to delete API key: %v", err)
		}

		// Try to get deleted key
		_, err = manager.GetAPIKey(provider)
		if err == nil {
			t.Error("Expected error for deleted key, got nil")
		}
	})

	t.Run("list providers", func(t *testing.T) {
		// Set multiple keys
		providers := map[string]string{
			"openai": "key1",
			"azure":  "key2",
			"custom": "key3",
		}

		for provider, key := range providers {
			err := manager.SetAPIKey(provider, key)
			if err != nil {
				t.Fatalf("Failed to set API key for %s: %v", provider, err)
			}
		}

		// List providers
		list, err := manager.ListProviders()
		if err != nil {
			t.Fatalf("Failed to list providers: %v", err)
		}

		if len(list) != len(providers) {
			t.Errorf("Expected %d providers, got %d", len(providers), len(list))
		}

		// Check all providers are in the list
		for provider := range providers {
			found := false
			for _, p := range list {
				if p == provider {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Provider %s not found in list", provider)
			}
		}
	})

	t.Run("file permissions", func(t *testing.T) {
		// Set a key to ensure file exists
		err := manager.SetAPIKey("test", "test-key")
		if err != nil {
			t.Fatalf("Failed to set API key: %v", err)
		}

		// Check file permissions
		info, err := os.Stat(secretsPath)
		if err != nil {
			t.Fatalf("Failed to stat secrets file: %v", err)
		}

		mode := info.Mode()
		if mode.Perm() != 0600 {
			t.Errorf("Expected file permissions 0600, got %v", mode.Perm())
		}
	})
}

func TestGetAPIKeyFromEnv(t *testing.T) {
	// Save current env vars
	oldOpenAI := os.Getenv("OPENAI_API_KEY")
	oldAzure := os.Getenv("AZURE_OPENAI_API_KEY")
	oldCoda := os.Getenv("CODA_API_KEY")

	// Restore env vars after test
	defer func() {
		os.Setenv("OPENAI_API_KEY", oldOpenAI)
		os.Setenv("AZURE_OPENAI_API_KEY", oldAzure)
		os.Setenv("CODA_API_KEY", oldCoda)
	}()

	// Clear env vars
	os.Unsetenv("OPENAI_API_KEY")
	os.Unsetenv("AZURE_OPENAI_API_KEY")
	os.Unsetenv("CODA_API_KEY")

	t.Run("OpenAI env var", func(t *testing.T) {
		testKey := "sk-openai-test-key"
		os.Setenv("OPENAI_API_KEY", testKey)

		key := getAPIKeyFromEnv("openai")
		if key != testKey {
			t.Errorf("Expected %s, got %s", testKey, key)
		}
	})

	t.Run("Azure env var", func(t *testing.T) {
		testKey := "azure-test-key"
		os.Setenv("AZURE_OPENAI_API_KEY", testKey)

		key := getAPIKeyFromEnv("azure")
		if key != testKey {
			t.Errorf("Expected %s, got %s", testKey, key)
		}
	})

	t.Run("Generic CODA env var", func(t *testing.T) {
		testKey := "coda-generic-key"
		os.Setenv("CODA_API_KEY", testKey)

		key := getAPIKeyFromEnv("custom")
		if key != testKey {
			t.Errorf("Expected %s, got %s", testKey, key)
		}
	})

	t.Run("Provider-specific CODA env var", func(t *testing.T) {
		// Clear generic CODA_API_KEY to test provider-specific fallback
		os.Unsetenv("CODA_API_KEY")
		
		testKey := "coda-custom-key"
		os.Setenv("CODA_custom_API_KEY", testKey)

		key := getAPIKeyFromEnv("custom")
		if key != testKey {
			t.Errorf("Expected %s, got %s", testKey, key)
		}
	})
}

func TestValidateAPIKey(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		wantErr bool
	}{
		{
			name:    "valid key",
			key:     "sk-1234567890abcdefghijklmnop",
			wantErr: false,
		},
		{
			name:    "empty key",
			key:     "",
			wantErr: true,
		},
		{
			name:    "too short",
			key:     "short",
			wantErr: true,
		},
		{
			name:    "placeholder 1",
			key:     "your-api-key",
			wantErr: true,
		},
		{
			name:    "placeholder 2",
			key:     "YOUR_API_KEY",
			wantErr: true,
		},
		{
			name:    "placeholder 3",
			key:     "sk-...",
			wantErr: true,
		},
		{
			name:    "placeholder 4",
			key:     "xxx",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAPIKey(tt.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateAPIKey() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetProvidersFromEnv(t *testing.T) {
	// Save current env vars
	oldOpenAI := os.Getenv("OPENAI_API_KEY")
	oldAzure := os.Getenv("AZURE_OPENAI_API_KEY")

	// Restore env vars after test
	defer func() {
		os.Setenv("OPENAI_API_KEY", oldOpenAI)
		os.Setenv("AZURE_OPENAI_API_KEY", oldAzure)
	}()

	t.Run("no env vars", func(t *testing.T) {
		os.Unsetenv("OPENAI_API_KEY")
		os.Unsetenv("AZURE_OPENAI_API_KEY")

		providers := getProvidersFromEnv()
		if len(providers) != 0 {
			t.Errorf("Expected 0 providers, got %d", len(providers))
		}
	})

	t.Run("OpenAI only", func(t *testing.T) {
		os.Setenv("OPENAI_API_KEY", "test-key")
		os.Unsetenv("AZURE_OPENAI_API_KEY")

		providers := getProvidersFromEnv()
		if len(providers) != 1 || providers[0] != "openai" {
			t.Errorf("Expected [openai], got %v", providers)
		}
	})

	t.Run("both providers", func(t *testing.T) {
		os.Setenv("OPENAI_API_KEY", "test-key1")
		os.Setenv("AZURE_OPENAI_API_KEY", "test-key2")

		providers := getProvidersFromEnv()
		if len(providers) != 2 {
			t.Errorf("Expected 2 providers, got %d", len(providers))
		}

		hasOpenAI := false
		hasAzure := false
		for _, p := range providers {
			if p == "openai" {
				hasOpenAI = true
			}
			if p == "azure" {
				hasAzure = true
			}
		}

		if !hasOpenAI || !hasAzure {
			t.Errorf("Expected both openai and azure, got %v", providers)
		}
	})
}

func TestGetServiceName(t *testing.T) {
	service := GetServiceName()
	expected := "com.common-creation.coda"

	if service != expected {
		t.Errorf("Expected service name %s, got %s", expected, service)
	}
}
