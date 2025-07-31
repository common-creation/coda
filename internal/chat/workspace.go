package chat

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

// WorkspaceConfig represents workspace-specific configuration
type WorkspaceConfig struct {
	Instructions string                 `json:"instructions"`
	Context      map[string]string      `json:"context"`
	Rules        []string               `json:"rules"`
	Preferences  map[string]interface{} `json:"preferences"`
	Metadata     WorkspaceMetadata      `json:"metadata"`
}

// WorkspaceMetadata contains metadata about the workspace config
type WorkspaceMetadata struct {
	Source       string    `json:"source"`
	LoadedAt     time.Time `json:"loaded_at"`
	LastModified time.Time `json:"last_modified"`
	Format       string    `json:"format"`
}

// WorkspaceLoader loads and manages workspace configurations
type WorkspaceLoader struct {
	searchPaths []string
	cache       map[string]*WorkspaceConfig
	fileWatcher *FileWatcher
	mu          sync.RWMutex
	onChange    func(*WorkspaceConfig)
	configFiles []string
}

// FileWatcher watches for file changes
type FileWatcher struct {
	files    map[string]time.Time
	interval time.Duration
	stopChan chan struct{}
	mu       sync.RWMutex
}

// ConfigFilePattern defines patterns for config files
var ConfigFilePatterns = []string{
	".coda/CODA.md",
	".claude/CLAUDE.md",
	"CODA.md",
	"CLAUDE.md",
	".coda/config.yaml",
	".coda/config.json",
}

// NewWorkspaceLoader creates a new workspace loader
func NewWorkspaceLoader() *WorkspaceLoader {
	return &WorkspaceLoader{
		searchPaths: []string{},
		cache:       make(map[string]*WorkspaceConfig),
		configFiles: ConfigFilePatterns,
		fileWatcher: NewFileWatcher(30 * time.Second),
	}
}

// LoadWorkspaceConfig loads workspace configuration from the current directory
func (l *WorkspaceLoader) LoadWorkspaceConfig(startDir string) (*WorkspaceConfig, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Check cache first
	if config, exists := l.cache[startDir]; exists {
		return config, nil
	}

	// Search for config files
	configPath, err := l.findConfigFile(startDir)
	if err != nil {
		return nil, err
	}

	if configPath == "" {
		return nil, nil // No config found
	}

	// Load the config
	config, err := l.loadConfigFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load config from %s: %w", configPath, err)
	}

	// Cache the config
	l.cache[startDir] = config

	// Start watching the file
	if l.fileWatcher != nil {
		l.fileWatcher.Watch(configPath, func() {
			l.reloadConfig(startDir, configPath)
		})
	}

	return config, nil
}

// SetOnChangeHandler sets a handler for config changes
func (l *WorkspaceLoader) SetOnChangeHandler(handler func(*WorkspaceConfig)) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.onChange = handler
}

// AddSearchPath adds a custom search path
func (l *WorkspaceLoader) AddSearchPath(path string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.searchPaths = append(l.searchPaths, path)
}

// AddConfigFile adds a custom config file pattern
func (l *WorkspaceLoader) AddConfigFile(pattern string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.configFiles = append(l.configFiles, pattern)
}

// ClearCache clears the config cache
func (l *WorkspaceLoader) ClearCache() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.cache = make(map[string]*WorkspaceConfig)
}

// Stop stops the file watcher
func (l *WorkspaceLoader) Stop() {
	if l.fileWatcher != nil {
		l.fileWatcher.Stop()
	}
}

// Private methods

func (l *WorkspaceLoader) findConfigFile(startDir string) (string, error) {
	// Search from startDir up to root
	currentDir := startDir
	for {
		// Check each config file pattern
		for _, pattern := range l.configFiles {
			configPath := filepath.Join(currentDir, pattern)
			if _, err := os.Stat(configPath); err == nil {
				return configPath, nil
			}
		}

		// Move to parent directory
		parentDir := filepath.Dir(currentDir)
		if parentDir == currentDir {
			break // Reached root
		}
		currentDir = parentDir
	}

	// Check additional search paths
	for _, searchPath := range l.searchPaths {
		for _, pattern := range l.configFiles {
			configPath := filepath.Join(searchPath, pattern)
			if _, err := os.Stat(configPath); err == nil {
				return configPath, nil
			}
		}
	}

	return "", nil
}

func (l *WorkspaceLoader) loadConfigFile(path string) (*WorkspaceConfig, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return nil, err
	}

	config := &WorkspaceConfig{
		Context:     make(map[string]string),
		Rules:       make([]string, 0),
		Preferences: make(map[string]interface{}),
		Metadata: WorkspaceMetadata{
			Source:       path,
			LoadedAt:     time.Now(),
			LastModified: stat.ModTime(),
		},
	}

	// Detect format based on extension
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".md", ".markdown":
		return l.parseMarkdownConfig(file, config)
	case ".yaml", ".yml":
		return l.parseYAMLConfig(file, config)
	case ".json":
		return l.parseJSONConfig(file, config)
	default:
		// Try to detect format from content
		return l.parseAutoDetect(file, config)
	}
}

func (l *WorkspaceLoader) parseMarkdownConfig(reader io.Reader, config *WorkspaceConfig) (*WorkspaceConfig, error) {
	config.Metadata.Format = "markdown"

	scanner := bufio.NewScanner(reader)
	var currentSection string
	var sectionContent strings.Builder
	var inCodeBlock bool
	var codeBlockLang string
	var codeBlockContent strings.Builder

	processSection := func() {
		if currentSection != "" && sectionContent.Len() > 0 {
			content := strings.TrimSpace(sectionContent.String())
			switch strings.ToLower(currentSection) {
			case "instructions", "instruction", "prompt":
				config.Instructions = content
			case "context":
				// Parse context as key-value pairs
				lines := strings.Split(content, "\n")
				for _, line := range lines {
					if parts := strings.SplitN(line, ":", 2); len(parts) == 2 {
						key := strings.TrimSpace(parts[0])
						value := strings.TrimSpace(parts[1])
						config.Context[key] = value
					}
				}
			case "rules", "rule":
				// Parse rules as list items
				lines := strings.Split(content, "\n")
				for _, line := range lines {
					line = strings.TrimSpace(line)
					line = strings.TrimPrefix(line, "- ")
					line = strings.TrimPrefix(line, "* ")
					line = strings.TrimPrefix(line, "+ ")
					if line != "" {
						config.Rules = append(config.Rules, line)
					}
				}
			case "preferences", "preference", "settings", "setting":
				// Try to parse as YAML or JSON
				var prefs map[string]interface{}
				if err := yaml.Unmarshal([]byte(content), &prefs); err == nil {
					config.Preferences = prefs
				} else if err := json.Unmarshal([]byte(content), &prefs); err == nil {
					config.Preferences = prefs
				}
			}
			sectionContent.Reset()
		}
	}

	for scanner.Scan() {
		line := scanner.Text()

		// Handle code blocks
		if strings.HasPrefix(line, "```") {
			if inCodeBlock {
				// End of code block
				inCodeBlock = false
				if codeBlockLang == "yaml" || codeBlockLang == "yml" {
					// Parse YAML block
					var data map[string]interface{}
					if err := yaml.Unmarshal([]byte(codeBlockContent.String()), &data); err == nil {
						// Merge into config
						if instructions, ok := data["instructions"].(string); ok {
							config.Instructions = instructions
						}
						if context, ok := data["context"].(map[string]interface{}); ok {
							for k, v := range context {
								config.Context[k] = fmt.Sprintf("%v", v)
							}
						}
						if rules, ok := data["rules"].([]interface{}); ok {
							for _, rule := range rules {
								config.Rules = append(config.Rules, fmt.Sprintf("%v", rule))
							}
						}
						if prefs, ok := data["preferences"].(map[string]interface{}); ok {
							config.Preferences = prefs
						}
					}
				} else if codeBlockLang == "json" {
					// Parse JSON block
					var data map[string]interface{}
					if err := json.Unmarshal([]byte(codeBlockContent.String()), &data); err == nil {
						// Similar merge logic as YAML
						if instructions, ok := data["instructions"].(string); ok {
							config.Instructions = instructions
						}
						// ... (similar to YAML parsing)
					}
				}
				codeBlockContent.Reset()
			} else {
				// Start of code block
				inCodeBlock = true
				codeBlockLang = strings.TrimPrefix(line, "```")
			}
			continue
		}

		if inCodeBlock {
			codeBlockContent.WriteString(line + "\n")
			continue
		}

		// Check for section headers
		if strings.HasPrefix(line, "#") {
			processSection()

			// Extract section name
			headerMatch := regexp.MustCompile(`^#+\s+(.+)$`).FindStringSubmatch(line)
			if len(headerMatch) > 1 {
				currentSection = headerMatch[1]
			}
		} else {
			sectionContent.WriteString(line + "\n")
		}
	}

	// Process final section
	processSection()

	// If no sections were found, treat entire content as instructions
	if config.Instructions == "" && len(config.Context) == 0 && len(config.Rules) == 0 {
		scanner = bufio.NewScanner(reader)
		var allContent strings.Builder
		for scanner.Scan() {
			allContent.WriteString(scanner.Text() + "\n")
		}
		config.Instructions = strings.TrimSpace(allContent.String())
	}

	return config, nil
}

func (l *WorkspaceLoader) parseYAMLConfig(reader io.Reader, config *WorkspaceConfig) (*WorkspaceConfig, error) {
	config.Metadata.Format = "yaml"

	decoder := yaml.NewDecoder(reader)
	var data map[string]interface{}
	if err := decoder.Decode(&data); err != nil {
		return nil, err
	}

	// Map YAML data to config
	if instructions, ok := data["instructions"].(string); ok {
		config.Instructions = instructions
	}

	if context, ok := data["context"].(map[string]interface{}); ok {
		for k, v := range context {
			config.Context[k] = fmt.Sprintf("%v", v)
		}
	}

	if rules, ok := data["rules"].([]interface{}); ok {
		for _, rule := range rules {
			config.Rules = append(config.Rules, fmt.Sprintf("%v", rule))
		}
	}

	if prefs, ok := data["preferences"].(map[string]interface{}); ok {
		config.Preferences = prefs
	}

	return config, nil
}

func (l *WorkspaceLoader) parseJSONConfig(reader io.Reader, config *WorkspaceConfig) (*WorkspaceConfig, error) {
	config.Metadata.Format = "json"

	decoder := json.NewDecoder(reader)
	var data map[string]interface{}
	if err := decoder.Decode(&data); err != nil {
		return nil, err
	}

	// Map JSON data to config (similar to YAML)
	if instructions, ok := data["instructions"].(string); ok {
		config.Instructions = instructions
	}

	if context, ok := data["context"].(map[string]interface{}); ok {
		for k, v := range context {
			config.Context[k] = fmt.Sprintf("%v", v)
		}
	}

	if rules, ok := data["rules"].([]interface{}); ok {
		for _, rule := range rules {
			config.Rules = append(config.Rules, fmt.Sprintf("%v", rule))
		}
	}

	if prefs, ok := data["preferences"].(map[string]interface{}); ok {
		config.Preferences = prefs
	}

	return config, nil
}

func (l *WorkspaceLoader) parseAutoDetect(reader io.Reader, config *WorkspaceConfig) (*WorkspaceConfig, error) {
	// Read content
	content, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	// Try JSON first
	var jsonData map[string]interface{}
	if err := json.Unmarshal(content, &jsonData); err == nil {
		return l.parseJSONConfig(bytes.NewReader(content), config)
	}

	// Try YAML
	var yamlData map[string]interface{}
	if err := yaml.Unmarshal(content, &yamlData); err == nil {
		return l.parseYAMLConfig(bytes.NewReader(content), config)
	}

	// Default to Markdown
	return l.parseMarkdownConfig(bytes.NewReader(content), config)
}

func (l *WorkspaceLoader) reloadConfig(dir, path string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	config, err := l.loadConfigFile(path)
	if err != nil {
		// Log error but don't crash
		fmt.Printf("Error reloading config: %v\n", err)
		return
	}

	// Update cache
	l.cache[dir] = config

	// Notify handler
	if l.onChange != nil {
		l.onChange(config)
	}
}

// FileWatcher implementation

// NewFileWatcher creates a new file watcher
func NewFileWatcher(interval time.Duration) *FileWatcher {
	return &FileWatcher{
		files:    make(map[string]time.Time),
		interval: interval,
		stopChan: make(chan struct{}),
	}
}

// Watch starts watching a file
func (w *FileWatcher) Watch(path string, onChange func()) {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Get initial mod time
	if stat, err := os.Stat(path); err == nil {
		w.files[path] = stat.ModTime()
	}

	// Start watcher goroutine
	go func() {
		ticker := time.NewTicker(w.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if w.checkFile(path) && onChange != nil {
					onChange()
				}
			case <-w.stopChan:
				return
			}
		}
	}()
}

// Stop stops the file watcher
func (w *FileWatcher) Stop() {
	close(w.stopChan)
}

func (w *FileWatcher) checkFile(path string) bool {
	w.mu.Lock()
	defer w.mu.Unlock()

	stat, err := os.Stat(path)
	if err != nil {
		return false
	}

	lastMod, exists := w.files[path]
	if !exists {
		w.files[path] = stat.ModTime()
		return false
	}

	if stat.ModTime().After(lastMod) {
		w.files[path] = stat.ModTime()
		return true
	}

	return false
}

// ApplyWorkspaceConfig applies workspace config to prompt builder
func ApplyWorkspaceConfig(config *WorkspaceConfig, promptBuilder *PromptBuilder) {
	if config == nil {
		return
	}

	// Add instructions as custom prompt
	if config.Instructions != "" {
		promptBuilder.AddCustomPrompt("workspace_instructions", config.Instructions)
	}

	// Add context fields
	for key, value := range config.Context {
		promptBuilder.SetContextField(key, value)
	}

	// Add rules as custom prompt
	if len(config.Rules) > 0 {
		rulesPrompt := "Workspace Rules:\n"
		for _, rule := range config.Rules {
			rulesPrompt += fmt.Sprintf("- %s\n", rule)
		}
		promptBuilder.AddCustomPrompt("workspace_rules", rulesPrompt)
	}

	// Apply preferences
	// This would depend on specific preference types
	if lang, ok := config.Preferences["language"].(string); ok {
		promptBuilder.SetContextField("preferred_language", lang)
	}
}
