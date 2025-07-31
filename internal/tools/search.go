package tools

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sync"
	"unicode/utf8"
)

// SearchFilesTool implements file content searching functionality
type SearchFilesTool struct {
	security SecurityValidator
}

// NewSearchFilesTool creates a new SearchFilesTool instance
func NewSearchFilesTool(security SecurityValidator) *SearchFilesTool {
	return &SearchFilesTool{security: security}
}

func (s *SearchFilesTool) Name() string {
	return "search_files"
}

func (s *SearchFilesTool) Description() string {
	return "Search for content in files"
}

func (s *SearchFilesTool) Schema() ToolSchema {
	return ToolSchema{
		Type: "object",
		Properties: map[string]Property{
			"path": {
				Type:        "string",
				Description: "Directory path to search in",
				Default:     ".",
			},
			"query": {
				Type:        "string",
				Description: "Search query (text or regex)",
			},
			"file_pattern": {
				Type:        "string",
				Description: "File name pattern to match (glob)",
			},
			"case_sensitive": {
				Type:        "boolean",
				Description: "Case sensitive search",
				Default:     true,
			},
			"use_regex": {
				Type:        "boolean",
				Description: "Use regular expression for query",
				Default:     false,
			},
			"max_results": {
				Type:        "integer",
				Description: "Maximum number of results to return",
				Default:     100,
			},
			"context": {
				Type:        "integer",
				Description: "Number of context lines before and after match",
				Default:     0,
			},
			"exclude_binary": {
				Type:        "boolean",
				Description: "Exclude binary files from search",
				Default:     true,
			},
		},
		Required: []string{"query"},
	}
}

func (s *SearchFilesTool) Validate(params map[string]interface{}) error {
	query, ok := params["query"].(string)
	if !ok || query == "" {
		return fmt.Errorf("query is required and must be a non-empty string")
	}

	// Validate regex if use_regex is true
	if useRegex, ok := params["use_regex"].(bool); ok && useRegex {
		if _, err := regexp.Compile(query); err != nil {
			return fmt.Errorf("invalid regex pattern: %w", err)
		}
	}

	// Validate max_results
	if maxResults, exists := params["max_results"]; exists {
		switch v := maxResults.(type) {
		case int:
			if v < 1 {
				return fmt.Errorf("max_results must be at least 1")
			}
		case float64:
			if v < 1 {
				return fmt.Errorf("max_results must be at least 1")
			}
		default:
			return fmt.Errorf("max_results must be a number")
		}
	}

	// Validate context
	if context, exists := params["context"]; exists {
		switch v := context.(type) {
		case int:
			if v < 0 {
				return fmt.Errorf("context must be non-negative")
			}
		case float64:
			if v < 0 {
				return fmt.Errorf("context must be non-negative")
			}
		default:
			return fmt.Errorf("context must be a number")
		}
	}

	return nil
}

func (s *SearchFilesTool) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	// Extract parameters
	path := "."
	if p, exists := params["path"]; exists {
		path = p.(string)
	}

	query := params["query"].(string)

	filePattern := "*"
	if p, exists := params["file_pattern"]; exists && p.(string) != "" {
		filePattern = p.(string)
	}

	caseSensitive := true
	if c, exists := params["case_sensitive"]; exists {
		caseSensitive = c.(bool)
	}

	useRegex := false
	if r, exists := params["use_regex"]; exists {
		useRegex = r.(bool)
	}

	maxResults := 100
	if m, exists := params["max_results"]; exists {
		switch v := m.(type) {
		case int:
			maxResults = v
		case float64:
			maxResults = int(v)
		}
	}

	contextLines := 0
	if c, exists := params["context"]; exists {
		switch v := c.(type) {
		case int:
			contextLines = v
		case float64:
			contextLines = int(v)
		}
	}

	excludeBinary := true
	if e, exists := params["exclude_binary"]; exists {
		excludeBinary = e.(bool)
	}

	// Normalize path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve path: %w", err)
	}

	// Security check
	if s.security != nil {
		if err := s.security.ValidatePath(absPath); err != nil {
			return nil, fmt.Errorf("security validation failed: %w", err)
		}
		if err := s.security.ValidateOperation(OpRead, absPath); err != nil {
			return nil, fmt.Errorf("operation not allowed: %w", err)
		}
	}

	// Prepare search pattern
	var searchPattern *regexp.Regexp
	if useRegex {
		searchPattern, err = regexp.Compile(query)
		if err != nil {
			return nil, fmt.Errorf("invalid regex pattern: %w", err)
		}
	} else {
		// Escape special regex characters for literal search
		escapedQuery := regexp.QuoteMeta(query)
		if !caseSensitive {
			escapedQuery = "(?i)" + escapedQuery
		}
		searchPattern = regexp.MustCompile(escapedQuery)
	}

	// Collect files to search
	files, err := s.collectSearchFiles(absPath, filePattern)
	if err != nil {
		return nil, fmt.Errorf("failed to collect files: %w", err)
	}

	// Search files concurrently
	results := make([]SearchResult, 0)
	resultsChan := make(chan []SearchResult, len(files))
	var wg sync.WaitGroup

	// Use a semaphore to limit concurrent file operations
	sem := make(chan struct{}, 10) // Max 10 concurrent file reads

	for _, file := range files {
		if len(results) >= maxResults {
			break
		}

		wg.Add(1)
		go func(filePath string) {
			defer wg.Done()

			// Acquire semaphore
			sem <- struct{}{}
			defer func() { <-sem }()

			// Security check for individual file
			if s.security != nil {
				if err := s.security.ValidatePath(filePath); err != nil {
					return
				}
			}

			fileResults, err := s.searchFile(filePath, searchPattern, contextLines, maxResults-len(results), excludeBinary)
			if err == nil && len(fileResults) > 0 {
				resultsChan <- fileResults
			}
		}(file)
	}

	// Close results channel when all goroutines are done
	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	// Collect results
	for fileResults := range resultsChan {
		results = append(results, fileResults...)
		if len(results) >= maxResults {
			results = results[:maxResults]
			break
		}
	}

	return map[string]interface{}{
		"results": results,
		"count":   len(results),
		"query":   query,
		"path":    absPath,
	}, nil
}

// SearchResult represents a single search match
type SearchResult struct {
	File    string   `json:"file"`
	Line    int      `json:"line"`
	Column  int      `json:"column"`
	Match   string   `json:"match"`
	Context []string `json:"context,omitempty"`
}

// collectSearchFiles collects all files matching the pattern
func (s *SearchFilesTool) collectSearchFiles(basePath string, pattern string) ([]string, error) {
	var files []string

	// Convert glob pattern to regex
	globRegex := globToRegex(pattern)
	patternRegex := regexp.MustCompile(globRegex)

	err := filepath.Walk(basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip inaccessible paths
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Check if filename matches pattern
		filename := filepath.Base(path)
		if patternRegex.MatchString(filename) {
			files = append(files, path)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return files, nil
}

// searchFile searches for matches in a single file
func (s *SearchFilesTool) searchFile(filePath string, pattern *regexp.Regexp, contextLines int, maxResults int, excludeBinary bool) ([]SearchResult, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// Check if file is binary
	if excludeBinary {
		// Read first 512 bytes to check if binary
		header := make([]byte, 512)
		n, _ := file.Read(header)
		if n > 0 && isBinary(header[:n]) {
			return nil, nil // Skip binary files
		}
		// Reset file position
		file.Seek(0, 0)
	}

	var results []SearchResult
	scanner := bufio.NewScanner(file)
	lineNum := 0
	var contextBuffer []string

	// If context is requested, maintain a buffer
	if contextLines > 0 {
		contextBuffer = make([]string, 0, contextLines*2+1)
	}

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		// Find all matches in the line
		matches := pattern.FindAllStringIndex(line, -1)

		for _, match := range matches {
			if len(results) >= maxResults {
				return results, nil
			}

			result := SearchResult{
				File:   filePath,
				Line:   lineNum,
				Column: match[0] + 1, // 1-based column
				Match:  line,
			}

			// Add context if requested
			if contextLines > 0 {
				result.Context = s.getContext(file, lineNum, contextLines)
			}

			results = append(results, result)
		}

		// Maintain context buffer
		if contextLines > 0 {
			contextBuffer = append(contextBuffer, line)
			if len(contextBuffer) > contextLines*2+1 {
				contextBuffer = contextBuffer[1:]
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return results, nil
}

// getContext retrieves context lines around a match
func (s *SearchFilesTool) getContext(file *os.File, targetLine int, contextLines int) []string {
	// Save current position
	currentPos, _ := file.Seek(0, io.SeekCurrent)
	defer file.Seek(currentPos, io.SeekStart)

	// Reset to beginning
	file.Seek(0, io.SeekStart)

	scanner := bufio.NewScanner(file)
	lineNum := 0
	var lines []string
	startLine := targetLine - contextLines
	endLine := targetLine + contextLines

	for scanner.Scan() {
		lineNum++
		if lineNum >= startLine && lineNum <= endLine {
			lines = append(lines, scanner.Text())
		}
		if lineNum > endLine {
			break
		}
	}

	return lines
}

// isBinary checks if the content appears to be binary
func isBinary(data []byte) bool {
	if len(data) == 0 {
		return false
	}

	// Check for null bytes
	if bytes.Contains(data, []byte{0}) {
		return true
	}

	// Check if valid UTF-8
	if !utf8.Valid(data) {
		return true
	}

	// Count non-printable characters
	nonPrintable := 0
	for _, b := range data {
		if b < 32 && b != '\t' && b != '\n' && b != '\r' {
			nonPrintable++
		}
	}

	// If more than 30% non-printable, consider binary
	if float64(nonPrintable)/float64(len(data)) > 0.3 {
		return true
	}

	return false
}

// Register tool in the default registry
func init() {
	RegisterFactoryGlobal("search_files", func() Tool {
		return NewSearchFilesTool(nil)
	})
}
