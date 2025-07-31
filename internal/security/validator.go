package security

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// Operation represents a file operation type
type Operation string

const (
	OpRead    Operation = "read"
	OpWrite   Operation = "write"
	OpDelete  Operation = "delete"
	OpExecute Operation = "execute"
	OpList    Operation = "list"
)

// SecurityValidator defines the interface for security validation
type SecurityValidator interface {
	ValidatePath(path string) error
	ValidateOperation(op Operation, path string) error
	IsAllowedExtension(path string) bool
	CheckContent(content []byte) error
}

// DefaultValidator implements the SecurityValidator interface
type DefaultValidator struct {
	workingDir   string
	allowedPaths []string
	deniedPaths  []string
	maxFileSize  int64
	allowedExts  map[string]bool
	deniedExts   map[string]bool
}

// NewDefaultValidator creates a new DefaultValidator instance
func NewDefaultValidator(workingDir string) *DefaultValidator {
	if workingDir == "" {
		workingDir, _ = os.Getwd()
	}

	return &DefaultValidator{
		workingDir:   workingDir,
		maxFileSize:  100 * 1024 * 1024, // 100MB default
		allowedPaths: []string{},
		deniedPaths: []string{
			"/etc",
			"/sys",
			"/proc",
			"/dev",
			"/root",
			"/boot",
			"~/.ssh",
			"~/.gnupg",
		},
		allowedExts: map[string]bool{
			".txt":  true,
			".md":   true,
			".json": true,
			".yaml": true,
			".yml":  true,
			".toml": true,
			".ini":  true,
			".conf": true,
			".cfg":  true,
			".log":  true,
			".go":   true,
			".py":   true,
			".js":   true,
			".ts":   true,
			".jsx":  true,
			".tsx":  true,
			".java": true,
			".c":    true,
			".cpp":  true,
			".h":    true,
			".hpp":  true,
			".rs":   true,
			".rb":   true,
			".php":  true,
			".html": true,
			".css":  true,
			".scss": true,
			".sass": true,
			".xml":  true,
			".csv":  true,
			".sql":  true,
			".sh":   true,
			".bash": true,
			".zsh":  true,
			".fish": true,
			".ps1":  true,
			".bat":  true,
			".cmd":  true,
		},
		deniedExts: map[string]bool{
			".exe":   true,
			".dll":   true,
			".so":    true,
			".dylib": true,
			".app":   true,
			".deb":   true,
			".rpm":   true,
			".dmg":   true,
			".pkg":   true,
			".msi":   true,
		},
	}
}

// ValidatePath checks if the path is safe to access
func (v *DefaultValidator) ValidatePath(path string) error {
	// Normalize the path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to resolve path: %w", err)
	}

	// Resolve symlinks
	realPath, err := filepath.EvalSymlinks(absPath)
	if err != nil {
		// If the file doesn't exist yet, use the absolute path
		if os.IsNotExist(err) {
			realPath = absPath
		} else {
			return fmt.Errorf("failed to resolve symlinks: %w", err)
		}
	}

	// Check for path traversal attempts
	if strings.Contains(path, "..") {
		// More detailed check
		cleanPath := filepath.Clean(path)
		if strings.Contains(cleanPath, "..") {
			return fmt.Errorf("path traversal detected")
		}
	}

	// Check if path is within working directory
	workingAbs, err := filepath.Abs(v.workingDir)
	if err != nil {
		return fmt.Errorf("failed to resolve working directory: %w", err)
	}

	// Ensure the path is within the working directory
	relPath, err := filepath.Rel(workingAbs, realPath)
	if err != nil || strings.HasPrefix(relPath, "..") {
		return fmt.Errorf("path is outside working directory")
	}

	// Check against denied paths
	for _, denied := range v.deniedPaths {
		// Expand home directory
		if strings.HasPrefix(denied, "~") {
			homeDir, err := os.UserHomeDir()
			if err == nil {
				denied = strings.Replace(denied, "~", homeDir, 1)
			}
		}

		deniedAbs, err := filepath.Abs(denied)
		if err != nil {
			continue
		}

		// Check if the path is within a denied directory
		if strings.HasPrefix(realPath, deniedAbs) {
			return fmt.Errorf("access to %s is denied", denied)
		}
	}

	// Platform-specific checks
	if runtime.GOOS == "windows" {
		// Check for Windows system paths
		lowerPath := strings.ToLower(realPath)
		if strings.HasPrefix(lowerPath, "c:\\windows") ||
			strings.HasPrefix(lowerPath, "c:\\program files") ||
			strings.HasPrefix(lowerPath, "c:\\programdata") {
			return fmt.Errorf("access to system directory is denied")
		}
	}

	return nil
}

// ValidateOperation checks if the operation is allowed on the path
func (v *DefaultValidator) ValidateOperation(op Operation, path string) error {
	// First validate the path itself
	if err := v.ValidatePath(path); err != nil {
		return err
	}

	// Get file info
	info, err := os.Stat(path)
	if err != nil {
		// For write operations, the file might not exist yet
		if os.IsNotExist(err) && (op == OpWrite || op == OpDelete) {
			return nil
		}
		return fmt.Errorf("failed to stat file: %w", err)
	}

	// Check file size for read operations
	if op == OpRead && info.Size() > v.maxFileSize {
		return fmt.Errorf("file too large: %d bytes (max: %d bytes)", info.Size(), v.maxFileSize)
	}

	// Check permissions based on operation
	switch op {
	case OpExecute:
		// Never allow execution
		return fmt.Errorf("execution is not allowed")
	case OpWrite, OpDelete:
		// Check if file is writable
		if info.Mode()&0200 == 0 {
			return fmt.Errorf("file is not writable")
		}
	}

	// Check extension for write operations
	if op == OpWrite && !v.IsAllowedExtension(path) {
		return fmt.Errorf("file extension is not allowed for writing")
	}

	return nil
}

// IsAllowedExtension checks if the file extension is allowed
func (v *DefaultValidator) IsAllowedExtension(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))

	// Check denied extensions first
	if v.deniedExts[ext] {
		return false
	}

	// If no extension, check if it's a known text file
	if ext == "" {
		base := filepath.Base(path)
		// Common text files without extensions
		textFiles := []string{
			"Makefile", "Dockerfile", "Jenkinsfile", "Vagrantfile",
			"README", "LICENSE", "CHANGELOG", "AUTHORS", "CONTRIBUTORS",
			".gitignore", ".dockerignore", ".env", ".editorconfig",
		}
		for _, tf := range textFiles {
			if base == tf {
				return true
			}
		}
		return false
	}

	// Check allowed extensions
	return v.allowedExts[ext]
}

// CheckContent validates the content of a file
func (v *DefaultValidator) CheckContent(content []byte) error {
	// Check size
	if int64(len(content)) > v.maxFileSize {
		return fmt.Errorf("content too large: %d bytes (max: %d bytes)", len(content), v.maxFileSize)
	}

	// Check for potentially dangerous content patterns
	dangerousPatterns := []string{
		"\x00",           // Null bytes
		"<script",        // Script tags
		"javascript:",    // JavaScript protocol
		"data:text/html", // Data URLs with HTML
	}

	contentStr := string(content)
	for _, pattern := range dangerousPatterns {
		if strings.Contains(contentStr, pattern) {
			return fmt.Errorf("potentially dangerous content detected: %s", pattern)
		}
	}

	return nil
}

// SetMaxFileSize sets the maximum allowed file size
func (v *DefaultValidator) SetMaxFileSize(size int64) {
	v.maxFileSize = size
}

// AddAllowedPath adds a path to the allowed paths list
func (v *DefaultValidator) AddAllowedPath(path string) {
	v.allowedPaths = append(v.allowedPaths, path)
}

// AddDeniedPath adds a path to the denied paths list
func (v *DefaultValidator) AddDeniedPath(path string) {
	v.deniedPaths = append(v.deniedPaths, path)
}

// AddAllowedExtension adds an extension to the allowed list
func (v *DefaultValidator) AddAllowedExtension(ext string) {
	if !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}
	v.allowedExts[strings.ToLower(ext)] = true
}

// AddDeniedExtension adds an extension to the denied list
func (v *DefaultValidator) AddDeniedExtension(ext string) {
	if !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}
	v.deniedExts[strings.ToLower(ext)] = true
}
