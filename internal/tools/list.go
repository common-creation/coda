package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// ListFilesTool implements directory listing functionality
type ListFilesTool struct {
	security SecurityValidator
}

// NewListFilesTool creates a new ListFilesTool instance
func NewListFilesTool(security SecurityValidator) *ListFilesTool {
	return &ListFilesTool{security: security}
}

func (l *ListFilesTool) Name() string {
	return "list_files"
}

func (l *ListFilesTool) Description() string {
	return "List files and directories"
}

func (l *ListFilesTool) Schema() ToolSchema {
	return ToolSchema{
		Type: "object",
		Properties: map[string]Property{
			"path": {
				Type:        "string",
				Description: "Directory path to list",
				Default:     ".",
			},
			"recursive": {
				Type:        "boolean",
				Description: "List files recursively",
				Default:     false,
			},
			"pattern": {
				Type:        "string",
				Description: "File pattern to match (glob or regex)",
			},
			"max_depth": {
				Type:        "integer",
				Description: "Maximum depth for recursive listing",
				Default:     -1,
			},
			"show_hidden": {
				Type:        "boolean",
				Description: "Show hidden files (starting with .)",
				Default:     false,
			},
			"sort": {
				Type:        "string",
				Description: "Sort order: name, size, time",
				Default:     "name",
				Enum:        []string{"name", "size", "time"},
			},
			"format": {
				Type:        "string",
				Description: "Output format: json, tree, list",
				Default:     "json",
				Enum:        []string{"json", "tree", "list"},
			},
		},
		Required: []string{},
	}
}

func (l *ListFilesTool) Validate(params map[string]interface{}) error {
	// Validate sort parameter
	if sort, exists := params["sort"]; exists {
		sortStr, ok := sort.(string)
		if !ok {
			return fmt.Errorf("sort must be a string")
		}
		if sortStr != "name" && sortStr != "size" && sortStr != "time" {
			return fmt.Errorf("sort must be one of: name, size, time")
		}
	}

	// Validate format parameter
	if format, exists := params["format"]; exists {
		formatStr, ok := format.(string)
		if !ok {
			return fmt.Errorf("format must be a string")
		}
		if formatStr != "json" && formatStr != "tree" && formatStr != "list" {
			return fmt.Errorf("format must be one of: json, tree, list")
		}
	}

	// Validate max_depth
	if depth, exists := params["max_depth"]; exists {
		switch v := depth.(type) {
		case int:
			if v < -1 {
				return fmt.Errorf("max_depth must be -1 or greater")
			}
		case float64:
			if v < -1 {
				return fmt.Errorf("max_depth must be -1 or greater")
			}
		default:
			return fmt.Errorf("max_depth must be a number")
		}
	}

	return nil
}

func (l *ListFilesTool) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	// Extract parameters
	path := "."
	if p, exists := params["path"]; exists {
		path = p.(string)
	}

	recursive := false
	if r, exists := params["recursive"]; exists {
		recursive = r.(bool)
	}

	pattern := ""
	if p, exists := params["pattern"]; exists {
		pattern = p.(string)
	}

	maxDepth := -1
	if d, exists := params["max_depth"]; exists {
		switch v := d.(type) {
		case int:
			maxDepth = v
		case float64:
			maxDepth = int(v)
		}
	}

	showHidden := false
	if s, exists := params["show_hidden"]; exists {
		showHidden = s.(bool)
	}

	sortBy := "name"
	if s, exists := params["sort"]; exists {
		sortBy = s.(string)
	}

	format := "json"
	if f, exists := params["format"]; exists {
		format = f.(string)
	}

	// Normalize path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve path: %w", err)
	}

	// Security check
	if l.security != nil {
		if err := l.security.ValidatePath(absPath); err != nil {
			return nil, fmt.Errorf("security validation failed: %w", err)
		}
		if err := l.security.ValidateOperation(OpList, absPath); err != nil {
			return nil, fmt.Errorf("operation not allowed: %w", err)
		}
	}

	// Check if path exists and is a directory
	info, err := os.Stat(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat path: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("path is not a directory")
	}

	// Collect files
	var files []FileInfo
	err = l.collectFiles(absPath, absPath, recursive, pattern, maxDepth, showHidden, 0, &files)
	if err != nil {
		return nil, fmt.Errorf("failed to collect files: %w", err)
	}

	// Sort files
	l.sortFiles(files, sortBy)

	// Format output
	switch format {
	case "tree":
		return l.formatTree(files, absPath), nil
	case "list":
		return l.formatList(files), nil
	default:
		return files, nil
	}
}

// FileInfo represents information about a file
type FileInfo struct {
	Path     string    `json:"path"`
	Name     string    `json:"name"`
	Size     int64     `json:"size"`
	Mode     string    `json:"mode"`
	ModTime  time.Time `json:"mod_time"`
	IsDir    bool      `json:"is_dir"`
	Relative string    `json:"relative"`
}

// collectFiles recursively collects file information
func (l *ListFilesTool) collectFiles(basePath, currentPath string, recursive bool, pattern string, maxDepth int, showHidden bool, currentDepth int, files *[]FileInfo) error {
	// Check depth limit
	if maxDepth != -1 && currentDepth > maxDepth {
		return nil
	}

	// Read directory
	entries, err := os.ReadDir(currentPath)
	if err != nil {
		// Don't fail completely if we can't read a subdirectory
		if currentDepth > 0 {
			return nil
		}
		return err
	}

	// Compile pattern if provided
	var patternRegex *regexp.Regexp
	if pattern != "" {
		// Try to compile as regex first
		patternRegex, err = regexp.Compile(pattern)
		if err != nil {
			// If not valid regex, convert glob to regex
			pattern = globToRegex(pattern)
			patternRegex = regexp.MustCompile(pattern)
		}
	}

	for _, entry := range entries {
		name := entry.Name()

		// Skip hidden files if not requested
		if !showHidden && strings.HasPrefix(name, ".") {
			continue
		}

		fullPath := filepath.Join(currentPath, name)

		// Get file info
		info, err := entry.Info()
		if err != nil {
			continue // Skip files we can't stat
		}

		// Calculate relative path
		relPath, err := filepath.Rel(basePath, fullPath)
		if err != nil {
			relPath = fullPath
		}

		// Check pattern match
		if patternRegex != nil && !patternRegex.MatchString(name) {
			// For directories, still recurse if recursive is enabled
			if recursive && info.IsDir() {
				err = l.collectFiles(basePath, fullPath, recursive, pattern, maxDepth, showHidden, currentDepth+1, files)
				if err != nil {
					return err
				}
			}
			continue
		}

		// Add file info
		fileInfo := FileInfo{
			Path:     fullPath,
			Name:     name,
			Size:     info.Size(),
			Mode:     info.Mode().String(),
			ModTime:  info.ModTime(),
			IsDir:    info.IsDir(),
			Relative: relPath,
		}

		*files = append(*files, fileInfo)

		// Recurse into directories if requested
		if recursive && info.IsDir() {
			err = l.collectFiles(basePath, fullPath, recursive, pattern, maxDepth, showHidden, currentDepth+1, files)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// sortFiles sorts the file list according to the specified order
func (l *ListFilesTool) sortFiles(files []FileInfo, sortBy string) {
	switch sortBy {
	case "size":
		sort.Slice(files, func(i, j int) bool {
			return files[i].Size < files[j].Size
		})
	case "time":
		sort.Slice(files, func(i, j int) bool {
			return files[i].ModTime.Before(files[j].ModTime)
		})
	default: // name
		sort.Slice(files, func(i, j int) bool {
			return files[i].Relative < files[j].Relative
		})
	}
}

// formatTree formats the file list as a tree structure
func (l *ListFilesTool) formatTree(files []FileInfo, basePath string) string {
	var result strings.Builder
	result.WriteString(filepath.Base(basePath) + "/\n")

	// Build tree structure
	tree := make(map[string][]FileInfo)
	for _, file := range files {
		dir := filepath.Dir(file.Relative)
		if dir == "." {
			dir = ""
		}
		tree[dir] = append(tree[dir], file)
	}

	// Print tree recursively
	l.printTree(&result, tree, "", "", true)

	return result.String()
}

// printTree recursively prints the tree structure
func (l *ListFilesTool) printTree(result *strings.Builder, tree map[string][]FileInfo, dir string, prefix string, isLast bool) {
	files := tree[dir]

	for i, file := range files {
		isLastFile := i == len(files)-1

		// Print current file
		if dir == "" {
			result.WriteString(prefix)
		}

		if isLastFile {
			result.WriteString("└── ")
		} else {
			result.WriteString("├── ")
		}

		result.WriteString(file.Name)
		if file.IsDir {
			result.WriteString("/")
		}
		result.WriteString("\n")

		// Recurse for directories
		if file.IsDir {
			newPrefix := prefix
			if isLastFile {
				newPrefix += "    "
			} else {
				newPrefix += "│   "
			}
			l.printTree(result, tree, file.Relative, newPrefix, isLastFile)
		}
	}
}

// formatList formats the file list as a detailed list
func (l *ListFilesTool) formatList(files []FileInfo) string {
	var result strings.Builder

	for _, file := range files {
		fileType := "file"
		if file.IsDir {
			fileType = "dir "
		}

		result.WriteString(fmt.Sprintf("%s %10d %s %s\n",
			fileType,
			file.Size,
			file.ModTime.Format("2006-01-02 15:04"),
			file.Relative,
		))
	}

	return result.String()
}

// globToRegex converts a glob pattern to a regular expression
func globToRegex(pattern string) string {
	// Escape special regex characters except * and ?
	pattern = regexp.QuoteMeta(pattern)
	// Convert glob wildcards to regex
	pattern = strings.ReplaceAll(pattern, `\*`, `.*`)
	pattern = strings.ReplaceAll(pattern, `\?`, `.`)
	return "^" + pattern + "$"
}

// Register tool in the default registry
func init() {
	RegisterFactoryGlobal("list_files", func() Tool {
		return NewListFilesTool(nil)
	})
}
