package chat

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// ContextManager manages chat context and related information
type ContextManager struct {
	currentPath    string
	openFiles      map[string]*FileContext
	recentTools    []ToolExecution
	projectInfo    *ProjectInfo
	maxContextSize int
	maxToolHistory int
	mu             sync.RWMutex
}

// FileContext represents context information about an open file
type FileContext struct {
	Path           string          `json:"path"`
	LastModified   time.Time       `json:"last_modified"`
	LastAccessed   time.Time       `json:"last_accessed"`
	Highlights     []CodeHighlight `json:"highlights"`
	References     []string        `json:"references"`
	AccessCount    int             `json:"access_count"`
	RelevanceScore float64         `json:"relevance_score"`
}

// CodeHighlight represents a highlighted section of code
type CodeHighlight struct {
	StartLine   int       `json:"start_line"`
	EndLine     int       `json:"end_line"`
	Content     string    `json:"content"`
	Type        string    `json:"type"` // error, warning, info, selection
	Description string    `json:"description"`
	Timestamp   time.Time `json:"timestamp"`
}

// ToolExecution represents a tool execution record
type ToolExecution struct {
	Tool       string                 `json:"tool"`
	Parameters map[string]interface{} `json:"parameters"`
	Result     interface{}            `json:"result"`
	Success    bool                   `json:"success"`
	Error      string                 `json:"error,omitempty"`
	Timestamp  time.Time              `json:"timestamp"`
	Duration   time.Duration          `json:"duration"`
}

// ProjectInfo contains information about the current project
type ProjectInfo struct {
	RootPath      string            `json:"root_path"`
	Language      string            `json:"language"`
	Framework     string            `json:"framework"`
	BuildSystem   string            `json:"build_system"`
	Dependencies  []Dependency      `json:"dependencies"`
	Configuration map[string]string `json:"configuration"`
	DetectedAt    time.Time         `json:"detected_at"`
}

// Dependency represents a project dependency
type Dependency struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Type    string `json:"type"` // runtime, dev, test
}

// NewContextManager creates a new context manager
func NewContextManager(maxContextSize, maxToolHistory int) *ContextManager {
	return &ContextManager{
		openFiles:      make(map[string]*FileContext),
		recentTools:    make([]ToolExecution, 0, maxToolHistory),
		maxContextSize: maxContextSize,
		maxToolHistory: maxToolHistory,
	}
}

// SetCurrentPath sets the current working directory
func (cm *ContextManager) SetCurrentPath(path string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.currentPath = path
}

// GetCurrentPath returns the current working directory
func (cm *ContextManager) GetCurrentPath() string {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.currentPath
}

// TrackFileAccess tracks access to a file
func (cm *ContextManager) TrackFileAccess(path string, highlight *CodeHighlight) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	absPath, err := filepath.Abs(path)
	if err != nil {
		absPath = path
	}

	fileCtx, exists := cm.openFiles[absPath]
	if !exists {
		fileCtx = &FileContext{
			Path:       absPath,
			Highlights: make([]CodeHighlight, 0),
			References: make([]string, 0),
		}
		cm.openFiles[absPath] = fileCtx
	}

	fileCtx.LastAccessed = time.Now()
	fileCtx.AccessCount++

	if highlight != nil {
		highlight.Timestamp = time.Now()
		fileCtx.Highlights = append(fileCtx.Highlights, *highlight)

		// Keep only recent highlights
		if len(fileCtx.Highlights) > 10 {
			fileCtx.Highlights = fileCtx.Highlights[len(fileCtx.Highlights)-10:]
		}
	}

	// Update relevance score
	cm.updateRelevanceScores()
}

// TrackToolExecution tracks a tool execution
func (cm *ContextManager) TrackToolExecution(execution ToolExecution) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	execution.Timestamp = time.Now()
	cm.recentTools = append(cm.recentTools, execution)

	// Keep only recent executions
	if len(cm.recentTools) > cm.maxToolHistory {
		cm.recentTools = cm.recentTools[len(cm.recentTools)-cm.maxToolHistory:]
	}

	// Extract file references from tool execution
	cm.extractFileReferences(execution)
}

// SetProjectInfo sets the project information
func (cm *ContextManager) SetProjectInfo(info *ProjectInfo) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	info.DetectedAt = time.Now()
	cm.projectInfo = info
}

// GetProjectInfo returns the project information
func (cm *ContextManager) GetProjectInfo() *ProjectInfo {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.projectInfo
}

// GetRelevantContext returns context relevant to the current conversation
func (cm *ContextManager) GetRelevantContext(maxTokens int) map[string]interface{} {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	context := make(map[string]interface{})

	// Add current directory
	context["current_directory"] = cm.currentPath

	// Add project info if available
	if cm.projectInfo != nil {
		context["project"] = cm.projectInfo
	}

	// Add recent files sorted by relevance
	relevantFiles := cm.getMostRelevantFiles(5)
	if len(relevantFiles) > 0 {
		context["recent_files"] = relevantFiles
	}

	// Add recent tool executions
	if len(cm.recentTools) > 0 {
		recentTools := cm.recentTools
		if len(recentTools) > 5 {
			recentTools = recentTools[len(recentTools)-5:]
		}
		context["recent_tools"] = recentTools
	}

	// Add error context if any
	errorContext := cm.getErrorContext()
	if len(errorContext) > 0 {
		context["errors"] = errorContext
	}

	return context
}

// ClearContext clears all tracked context
func (cm *ContextManager) ClearContext() {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.openFiles = make(map[string]*FileContext)
	cm.recentTools = make([]ToolExecution, 0, cm.maxToolHistory)
}

// ExportContext exports the context for persistence
func (cm *ContextManager) ExportContext() ([]byte, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	data := map[string]interface{}{
		"current_path": cm.currentPath,
		"open_files":   cm.openFiles,
		"recent_tools": cm.recentTools,
		"project_info": cm.projectInfo,
	}

	return json.Marshal(data)
}

// ImportContext imports previously exported context
func (cm *ContextManager) ImportContext(data []byte) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	var imported struct {
		CurrentPath string                  `json:"current_path"`
		OpenFiles   map[string]*FileContext `json:"open_files"`
		RecentTools []ToolExecution         `json:"recent_tools"`
		ProjectInfo *ProjectInfo            `json:"project_info"`
	}

	if err := json.Unmarshal(data, &imported); err != nil {
		return err
	}

	cm.currentPath = imported.CurrentPath
	cm.openFiles = imported.OpenFiles
	cm.recentTools = imported.RecentTools
	cm.projectInfo = imported.ProjectInfo

	return nil
}

// Private methods

func (cm *ContextManager) updateRelevanceScores() {
	now := time.Now()

	for _, fileCtx := range cm.openFiles {
		// Base score on access count
		score := float64(fileCtx.AccessCount)

		// Factor in recency (decay over time)
		hoursSinceAccess := now.Sub(fileCtx.LastAccessed).Hours()
		recencyFactor := 1.0 / (1.0 + hoursSinceAccess/24.0)
		score *= recencyFactor

		// Boost for files with highlights
		if len(fileCtx.Highlights) > 0 {
			score *= 1.5
		}

		// Boost for files with errors
		for _, highlight := range fileCtx.Highlights {
			if highlight.Type == "error" {
				score *= 2.0
				break
			}
		}

		fileCtx.RelevanceScore = score
	}
}

func (cm *ContextManager) getMostRelevantFiles(limit int) []*FileContext {
	files := make([]*FileContext, 0, len(cm.openFiles))
	for _, fileCtx := range cm.openFiles {
		files = append(files, fileCtx)
	}

	// Sort by relevance score
	sort.Slice(files, func(i, j int) bool {
		return files[i].RelevanceScore > files[j].RelevanceScore
	})

	if len(files) > limit {
		files = files[:limit]
	}

	return files
}

func (cm *ContextManager) getErrorContext() []map[string]interface{} {
	var errors []map[string]interface{}

	// Check file highlights for errors
	for _, fileCtx := range cm.openFiles {
		for _, highlight := range fileCtx.Highlights {
			if highlight.Type == "error" {
				errors = append(errors, map[string]interface{}{
					"file":        fileCtx.Path,
					"line":        highlight.StartLine,
					"description": highlight.Description,
					"timestamp":   highlight.Timestamp,
				})
			}
		}
	}

	// Check tool execution errors
	for _, exec := range cm.recentTools {
		if !exec.Success && exec.Error != "" {
			errors = append(errors, map[string]interface{}{
				"tool":      exec.Tool,
				"error":     exec.Error,
				"timestamp": exec.Timestamp,
			})
		}
	}

	// Sort by timestamp (most recent first)
	sort.Slice(errors, func(i, j int) bool {
		t1 := errors[i]["timestamp"].(time.Time)
		t2 := errors[j]["timestamp"].(time.Time)
		return t1.After(t2)
	})

	if len(errors) > 5 {
		errors = errors[:5]
	}

	return errors
}

func (cm *ContextManager) extractFileReferences(execution ToolExecution) {
	// Extract file paths from parameters
	for key, value := range execution.Parameters {
		if strings.Contains(strings.ToLower(key), "path") || strings.Contains(strings.ToLower(key), "file") {
			if pathStr, ok := value.(string); ok {
				if fileCtx, exists := cm.openFiles[pathStr]; exists {
					fileCtx.References = append(fileCtx.References, execution.Tool)
				}
			}
		}
	}
}

// DetectProjectInfo attempts to detect project information
func DetectProjectInfo(rootPath string) *ProjectInfo {
	info := &ProjectInfo{
		RootPath:      rootPath,
		Configuration: make(map[string]string),
		Dependencies:  make([]Dependency, 0),
	}

	// Check for Go project
	if exists(filepath.Join(rootPath, "go.mod")) {
		info.Language = "Go"
		info.BuildSystem = "go"
		// Parse go.mod for dependencies
		if deps := parseGoMod(filepath.Join(rootPath, "go.mod")); deps != nil {
			info.Dependencies = deps
		}
	}

	// Check for Node.js project
	if exists(filepath.Join(rootPath, "package.json")) {
		info.Language = "JavaScript/TypeScript"
		info.BuildSystem = "npm"
		// Parse package.json for dependencies
		if deps := parsePackageJSON(filepath.Join(rootPath, "package.json")); deps != nil {
			info.Dependencies = deps
		}
	}

	// Check for Python project
	if exists(filepath.Join(rootPath, "requirements.txt")) {
		info.Language = "Python"
		info.BuildSystem = "pip"
	} else if exists(filepath.Join(rootPath, "pyproject.toml")) {
		info.Language = "Python"
		info.BuildSystem = "poetry"
	}

	// Check for Rust project
	if exists(filepath.Join(rootPath, "Cargo.toml")) {
		info.Language = "Rust"
		info.BuildSystem = "cargo"
	}

	// Check for configuration files
	configFiles := []string{".env", "config.yaml", "config.json", "settings.py"}
	for _, cf := range configFiles {
		if exists(filepath.Join(rootPath, cf)) {
			info.Configuration[cf] = "present"
		}
	}

	return info
}

// Helper functions

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func parseGoMod(path string) []Dependency {
	// Simplified implementation - real implementation would parse go.mod
	return nil
}

func parsePackageJSON(path string) []Dependency {
	// Simplified implementation - real implementation would parse package.json
	return nil
}
