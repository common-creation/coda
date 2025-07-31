package tools

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode/utf8"
)

// ReadFileTool implements file reading functionality
type ReadFileTool struct {
	security SecurityValidator
}

// NewReadFileTool creates a new ReadFileTool instance
func NewReadFileTool(security SecurityValidator) *ReadFileTool {
	return &ReadFileTool{security: security}
}

func (r *ReadFileTool) Name() string {
	return "read_file"
}

func (r *ReadFileTool) Description() string {
	return "Read the contents of a file"
}

func (r *ReadFileTool) Schema() ToolSchema {
	return ToolSchema{
		Type: "object",
		Properties: map[string]Property{
			"path": {
				Type:        "string",
				Description: "Path to the file to read",
			},
			"offset": {
				Type:        "integer",
				Description: "Start reading from this byte offset (optional)",
				Default:     0,
			},
			"limit": {
				Type:        "integer",
				Description: "Maximum number of bytes to read (optional)",
				Default:     -1,
			},
		},
		Required: []string{"path"},
	}
}

func (r *ReadFileTool) Validate(params map[string]interface{}) error {
	path, ok := params["path"].(string)
	if !ok || path == "" {
		return fmt.Errorf("path is required and must be a string")
	}

	// Validate offset if provided
	if offset, exists := params["offset"]; exists {
		if _, ok := offset.(int); !ok {
			if _, ok := offset.(float64); !ok {
				return fmt.Errorf("offset must be a number")
			}
		}
	}

	// Validate limit if provided
	if limit, exists := params["limit"]; exists {
		if _, ok := limit.(int); !ok {
			if _, ok := limit.(float64); !ok {
				return fmt.Errorf("limit must be a number")
			}
		}
	}

	return nil
}

func (r *ReadFileTool) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	path := params["path"].(string)
	offset := int64(0)
	limit := int64(-1)

	// Convert float64 to int64 for numeric parameters
	if val, exists := params["offset"]; exists {
		switch v := val.(type) {
		case int:
			offset = int64(v)
		case float64:
			offset = int64(v)
		}
	}

	if val, exists := params["limit"]; exists {
		switch v := val.(type) {
		case int:
			limit = int64(v)
		case float64:
			limit = int64(v)
		}
	}

	// Normalize path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve path: %w", err)
	}

	// Security check
	if r.security != nil {
		if err := r.security.ValidatePath(absPath); err != nil {
			return nil, fmt.Errorf("security validation failed: %w", err)
		}
		if err := r.security.ValidateOperation(OpRead, absPath); err != nil {
			return nil, fmt.Errorf("operation not allowed: %w", err)
		}
	}

	// Open file
	file, err := os.Open(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Seek to offset if specified
	if offset > 0 {
		_, err = file.Seek(offset, io.SeekStart)
		if err != nil {
			return nil, fmt.Errorf("failed to seek to offset: %w", err)
		}
	}

	// Read file content
	var reader io.Reader = file
	if limit > 0 {
		reader = io.LimitReader(file, limit)
	}

	content, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Check if content is valid UTF-8
	if !utf8.Valid(content) {
		return nil, fmt.Errorf("file contains invalid UTF-8 content")
	}

	// Check content security
	if r.security != nil {
		if err := r.security.CheckContent(content); err != nil {
			return nil, fmt.Errorf("content validation failed: %w", err)
		}
	}

	return string(content), nil
}

// WriteFileTool implements file writing functionality
type WriteFileTool struct {
	security SecurityValidator
}

// NewWriteFileTool creates a new WriteFileTool instance
func NewWriteFileTool(security SecurityValidator) *WriteFileTool {
	return &WriteFileTool{security: security}
}

func (w *WriteFileTool) Name() string {
	return "write_file"
}

func (w *WriteFileTool) Description() string {
	return "Write content to a file"
}

func (w *WriteFileTool) Schema() ToolSchema {
	return ToolSchema{
		Type: "object",
		Properties: map[string]Property{
			"path": {
				Type:        "string",
				Description: "Path to the file to write",
			},
			"content": {
				Type:        "string",
				Description: "Content to write to the file",
			},
			"create_dirs": {
				Type:        "boolean",
				Description: "Create parent directories if they don't exist",
				Default:     true,
			},
			"backup": {
				Type:        "boolean",
				Description: "Create a backup of existing file",
				Default:     false,
			},
		},
		Required: []string{"path", "content"},
	}
}

func (w *WriteFileTool) Validate(params map[string]interface{}) error {
	path, ok := params["path"].(string)
	if !ok || path == "" {
		return fmt.Errorf("path is required and must be a string")
	}

	if _, ok := params["content"].(string); !ok {
		return fmt.Errorf("content is required and must be a string")
	}

	return nil
}

func (w *WriteFileTool) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	path := params["path"].(string)
	content := params["content"].(string)
	createDirs := true
	backup := false

	if val, exists := params["create_dirs"]; exists {
		createDirs, _ = val.(bool)
	}

	if val, exists := params["backup"]; exists {
		backup, _ = val.(bool)
	}

	// Normalize path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve path: %w", err)
	}

	// Security check
	if w.security != nil {
		if err := w.security.ValidatePath(absPath); err != nil {
			return nil, fmt.Errorf("security validation failed: %w", err)
		}
		if err := w.security.ValidateOperation(OpWrite, absPath); err != nil {
			return nil, fmt.Errorf("operation not allowed: %w", err)
		}
		if err := w.security.CheckContent([]byte(content)); err != nil {
			return nil, fmt.Errorf("content validation failed: %w", err)
		}
	}

	// Create directories if needed
	if createDirs {
		dir := filepath.Dir(absPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create directories: %w", err)
		}
	}

	// Create backup if requested and file exists
	if backup {
		if _, err := os.Stat(absPath); err == nil {
			backupPath := absPath + ".bak"
			if err := copyFile(absPath, backupPath); err != nil {
				return nil, fmt.Errorf("failed to create backup: %w", err)
			}
		}
	}

	// Write file
	if err := os.WriteFile(absPath, []byte(content), 0644); err != nil {
		return nil, fmt.Errorf("failed to write file: %w", err)
	}

	return map[string]interface{}{
		"path":    absPath,
		"size":    len(content),
		"success": true,
	}, nil
}

// EditFileTool implements file editing functionality
type EditFileTool struct {
	security SecurityValidator
}

// NewEditFileTool creates a new EditFileTool instance
func NewEditFileTool(security SecurityValidator) *EditFileTool {
	return &EditFileTool{security: security}
}

func (e *EditFileTool) Name() string {
	return "edit_file"
}

func (e *EditFileTool) Description() string {
	return "Edit a file by replacing content"
}

func (e *EditFileTool) Schema() ToolSchema {
	return ToolSchema{
		Type: "object",
		Properties: map[string]Property{
			"path": {
				Type:        "string",
				Description: "Path to the file to edit",
			},
			"old_text": {
				Type:        "string",
				Description: "Text to search for and replace",
			},
			"new_text": {
				Type:        "string",
				Description: "Text to replace with",
			},
			"regex": {
				Type:        "boolean",
				Description: "Use regular expression for matching",
				Default:     false,
			},
			"all": {
				Type:        "boolean",
				Description: "Replace all occurrences",
				Default:     true,
			},
		},
		Required: []string{"path", "old_text", "new_text"},
	}
}

func (e *EditFileTool) Validate(params map[string]interface{}) error {
	path, ok := params["path"].(string)
	if !ok || path == "" {
		return fmt.Errorf("path is required and must be a string")
	}

	if _, ok := params["old_text"].(string); !ok {
		return fmt.Errorf("old_text is required and must be a string")
	}

	if _, ok := params["new_text"].(string); !ok {
		return fmt.Errorf("new_text is required and must be a string")
	}

	// Validate regex pattern if regex mode is enabled
	if useRegex, ok := params["regex"].(bool); ok && useRegex {
		pattern := params["old_text"].(string)
		if _, err := regexp.Compile(pattern); err != nil {
			return fmt.Errorf("invalid regex pattern: %w", err)
		}
	}

	return nil
}

func (e *EditFileTool) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	path := params["path"].(string)
	oldText := params["old_text"].(string)
	newText := params["new_text"].(string)
	useRegex := false
	replaceAll := true

	if val, exists := params["regex"]; exists {
		useRegex, _ = val.(bool)
	}

	if val, exists := params["all"]; exists {
		replaceAll, _ = val.(bool)
	}

	// Normalize path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve path: %w", err)
	}

	// Security check
	if e.security != nil {
		if err := e.security.ValidatePath(absPath); err != nil {
			return nil, fmt.Errorf("security validation failed: %w", err)
		}
		if err := e.security.ValidateOperation(OpRead, absPath); err != nil {
			return nil, fmt.Errorf("read operation not allowed: %w", err)
		}
		if err := e.security.ValidateOperation(OpWrite, absPath); err != nil {
			return nil, fmt.Errorf("write operation not allowed: %w", err)
		}
	}

	// Read current content
	content, err := os.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Check if content is valid UTF-8
	if !utf8.Valid(content) {
		return nil, fmt.Errorf("file contains invalid UTF-8 content")
	}

	originalContent := string(content)
	newContent := originalContent
	replacements := 0

	// Perform replacement
	if useRegex {
		re, err := regexp.Compile(oldText)
		if err != nil {
			return nil, fmt.Errorf("invalid regex pattern: %w", err)
		}

		if replaceAll {
			newContent = re.ReplaceAllString(originalContent, newText)
			replacements = strings.Count(originalContent, oldText) - strings.Count(newContent, oldText)
		} else {
			loc := re.FindStringIndex(originalContent)
			if loc != nil {
				newContent = originalContent[:loc[0]] + newText + originalContent[loc[1]:]
				replacements = 1
			}
		}
	} else {
		if replaceAll {
			newContent = strings.ReplaceAll(originalContent, oldText, newText)
			replacements = strings.Count(originalContent, oldText)
		} else {
			index := strings.Index(originalContent, oldText)
			if index >= 0 {
				newContent = originalContent[:index] + newText + originalContent[index+len(oldText):]
				replacements = 1
			}
		}
	}

	// Check if any changes were made
	if replacements == 0 {
		return map[string]interface{}{
			"path":         absPath,
			"replacements": 0,
			"success":      true,
			"message":      "No matches found",
		}, nil
	}

	// Security check new content
	if e.security != nil {
		if err := e.security.CheckContent([]byte(newContent)); err != nil {
			return nil, fmt.Errorf("new content validation failed: %w", err)
		}
	}

	// Create temp file for atomic write
	tmpFile, err := os.CreateTemp(filepath.Dir(absPath), ".tmp-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	// Write new content to temp file
	if _, err := tmpFile.WriteString(newContent); err != nil {
		tmpFile.Close()
		return nil, fmt.Errorf("failed to write temp file: %w", err)
	}
	tmpFile.Close()

	// Get original file permissions
	info, err := os.Stat(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat original file: %w", err)
	}

	// Set permissions on temp file
	if err := os.Chmod(tmpPath, info.Mode()); err != nil {
		return nil, fmt.Errorf("failed to set permissions: %w", err)
	}

	// Atomically replace original file
	if err := os.Rename(tmpPath, absPath); err != nil {
		return nil, fmt.Errorf("failed to replace file: %w", err)
	}

	return map[string]interface{}{
		"path":         absPath,
		"replacements": replacements,
		"success":      true,
	}, nil
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destination.Close()

	_, err = io.Copy(destination, source)
	return err
}

// Register tools in the default registry
func init() {
	// Note: SecurityValidator should be injected when creating the tools
	// This is just for registration purposes
	RegisterFactoryGlobal("read_file", func() Tool {
		return NewReadFileTool(nil)
	})

	RegisterFactoryGlobal("write_file", func() Tool {
		return NewWriteFileTool(nil)
	})

	RegisterFactoryGlobal("edit_file", func() Tool {
		return NewEditFileTool(nil)
	})
}
