package ui

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/bubbles/key"
	"gopkg.in/yaml.v3"
)

// KeyBindingManager manages keybinding configuration and customization
type KeyBindingManager struct {
	keymap     KeyMap
	configPath string
	conflicts  []string
}

// NewKeyBindingManager creates a new keybinding manager
func NewKeyBindingManager(configPath string) *KeyBindingManager {
	return &KeyBindingManager{
		keymap:     DefaultKeyMap(),
		configPath: configPath,
		conflicts:  make([]string, 0),
	}
}

// LoadConfig loads keybinding configuration from file
func (kbm *KeyBindingManager) LoadConfig() error {
	if kbm.configPath == "" {
		return fmt.Errorf("no config path specified")
	}

	// Check if config file exists
	if _, err := os.Stat(kbm.configPath); os.IsNotExist(err) {
		// Create default config if it doesn't exist
		return kbm.CreateDefaultConfig()
	}

	// Read config file
	data, err := os.ReadFile(kbm.configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse YAML
	var config KeyBindingConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}

	// Load key bindings from config
	if err := kbm.keymap.LoadFromConfig(config); err != nil {
		return fmt.Errorf("failed to load key bindings: %w", err)
	}

	// Validate for conflicts
	kbm.conflicts = kbm.keymap.Validate()
	if len(kbm.conflicts) > 0 {
		fmt.Fprintf(os.Stderr, "Warning: Key binding conflicts detected:\n")
		for _, conflict := range kbm.conflicts {
			fmt.Fprintf(os.Stderr, "  - %s\n", conflict)
		}
	}

	return nil
}

// SaveConfig saves current keybinding configuration to file
func (kbm *KeyBindingManager) SaveConfig() error {
	if kbm.configPath == "" {
		return fmt.Errorf("no config path specified")
	}

	// Create config directory if it doesn't exist
	dir := filepath.Dir(kbm.configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Convert current keymap to config format
	config := kbm.exportToConfig()

	// Marshal to YAML
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write to file
	if err := os.WriteFile(kbm.configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// CreateDefaultConfig creates a default configuration file
func (kbm *KeyBindingManager) CreateDefaultConfig() error {
	config := KeyBindingConfig{
		Style: "default",
		Bindings: map[string]KeyBinding{
			// Example custom bindings
			"quick_save": {
				Keys:        []string{"ctrl+s"},
				Description: "Quick save current input",
				Context:     "insert",
				Mode:        "insert",
			},
			"toggle_theme": {
				Keys:        []string{"F2"},
				Description: "Toggle UI theme",
				Context:     "global",
				Mode:        "all",
			},
			"focus_chat": {
				Keys:        []string{"ctrl+1"},
				Description: "Focus chat view",
				Context:     "global",
				Mode:        "normal",
			},
		},
	}

	// Marshal to YAML
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal default config: %w", err)
	}

	// Create config directory if it doesn't exist
	dir := filepath.Dir(kbm.configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Write to file
	if err := os.WriteFile(kbm.configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write default config file: %w", err)
	}

	return nil
}

// exportToConfig converts current keymap to config format
func (kbm *KeyBindingManager) exportToConfig() KeyBindingConfig {
	config := KeyBindingConfig{
		Style:    "default", // This could be detected from current keymap
		Bindings: make(map[string]KeyBinding),
	}

	// Export custom bindings
	for name, binding := range kbm.keymap.Custom {
		if binding.Keys() != nil {
			config.Bindings[name] = KeyBinding{
				Keys:        binding.Keys(),
				Description: binding.Help().Key, // This might need adjustment based on actual binding structure
				Context:     "custom",
				Mode:        "all",
			}
		}
	}

	return config
}

// GetKeyMap returns the current keymap
func (kbm *KeyBindingManager) GetKeyMap() KeyMap {
	return kbm.keymap
}

// SetKeyMap sets the keymap
func (kbm *KeyBindingManager) SetKeyMap(keymap KeyMap) {
	kbm.keymap = keymap
	kbm.conflicts = kbm.keymap.Validate()
}

// GetConflicts returns any key binding conflicts
func (kbm *KeyBindingManager) GetConflicts() []string {
	return kbm.conflicts
}

// HasConflicts returns true if there are key binding conflicts
func (kbm *KeyBindingManager) HasConflicts() bool {
	return len(kbm.conflicts) > 0
}

// ResetToDefaults resets keybindings to default values
func (kbm *KeyBindingManager) ResetToDefaults() {
	kbm.keymap.Reset()
	kbm.conflicts = make([]string, 0)
}

// SetStyle changes the keybinding style (vim, emacs, default)
func (kbm *KeyBindingManager) SetStyle(style string) error {
	switch style {
	case "vim":
		kbm.keymap = VimKeyMap()
	case "emacs":
		kbm.keymap = EmacsKeyMap()
	case "default":
		kbm.keymap = DefaultKeyMap()
	default:
		return fmt.Errorf("unknown keybinding style: %s", style)
	}

	kbm.conflicts = kbm.keymap.Validate()
	return nil
}

// AddCustomBinding adds a custom key binding
func (kbm *KeyBindingManager) AddCustomBinding(name string, keys []string, description string) error {
	if kbm.keymap.Custom == nil {
		kbm.keymap.Custom = make(map[string]key.Binding)
	}

	// Create new binding
	binding := key.NewBinding(
		key.WithKeys(keys...),
		key.WithHelp(keys[0], description),
	)

	kbm.keymap.Custom[name] = binding

	// Re-validate for conflicts
	kbm.conflicts = kbm.keymap.Validate()

	return nil
}

// RemoveCustomBinding removes a custom key binding
func (kbm *KeyBindingManager) RemoveCustomBinding(name string) {
	if kbm.keymap.Custom != nil {
		delete(kbm.keymap.Custom, name)
		kbm.conflicts = kbm.keymap.Validate()
	}
}

// ListCustomBindings returns all custom bindings
func (kbm *KeyBindingManager) ListCustomBindings() map[string][]string {
	result := make(map[string][]string)

	for name, binding := range kbm.keymap.Custom {
		if binding.Keys() != nil {
			result[name] = binding.Keys()
		}
	}

	return result
}

// GetBindingHelp returns help text for a specific binding
func (kbm *KeyBindingManager) GetBindingHelp(mode Mode) []string {
	return kbm.keymap.GetHelpText(mode)
}
