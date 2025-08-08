package chat

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// ResultProcessor processes tool execution results
type ResultProcessor struct {
	formatter    OutputFormatter
	summarizer   ContentSummarizer
	cache        *ResultCache
	maxTokens    int
	tokenCounter TokenCounter
}

// OutputFormatter formats output for display
type OutputFormatter interface {
	Format(result ToolResult) string
	FormatError(err error) string
}

// ContentSummarizer summarizes large content
type ContentSummarizer interface {
	Summarize(content string, maxLength int) string
	ShouldSummarize(content string) bool
}

// ProcessedOutput represents processed tool results
type ProcessedOutput struct {
	DisplayOutput  string                 `json:"display_output"`
	AIFeedback     string                 `json:"ai_feedback"`
	Metadata       map[string]interface{} `json:"metadata"`
	TokenCount     int                    `json:"token_count"`
	Summarized     bool                   `json:"summarized"`
	CacheHit       bool                   `json:"cache_hit"`
	ProcessingTime time.Duration          `json:"processing_time"`
}

// ResultCache caches processed results
type ResultCache struct {
	cache     map[string]*CacheEntry
	maxSize   int
	maxAge    time.Duration
	mu        sync.RWMutex
	hitCount  int64
	missCount int64
}

// CacheEntry represents a cached result
type CacheEntry struct {
	Result    ProcessedOutput
	Timestamp time.Time
	HitCount  int
}

// NewResultProcessor creates a new result processor
func NewResultProcessor(maxTokens int, tokenCounter TokenCounter) *ResultProcessor {
	if tokenCounter == nil {
		tokenCounter = &SimpleTokenCounter{}
	}

	return &ResultProcessor{
		formatter:    NewDefaultFormatter(),
		summarizer:   NewDefaultSummarizer(),
		cache:        NewResultCache(100, 30*time.Minute),
		maxTokens:    maxTokens,
		tokenCounter: tokenCounter,
	}
}

// ProcessResults processes multiple tool results
func (p *ResultProcessor) ProcessResults(results []ToolResult) []ProcessedOutput {
	outputs := make([]ProcessedOutput, 0, len(results))

	for _, result := range results {
		output := p.ProcessSingleResult(result)
		outputs = append(outputs, output)
	}

	return outputs
}

// ProcessSingleResult processes a single tool result
func (p *ResultProcessor) ProcessSingleResult(result ToolResult) ProcessedOutput {
	startTime := time.Now()

	// Check cache first
	cacheKey := p.generateCacheKey(result)
	if cached, hit := p.cache.Get(cacheKey); hit {
		cached.CacheHit = true
		cached.ProcessingTime = time.Since(startTime)
		return cached
	}

	output := ProcessedOutput{
		Metadata: make(map[string]interface{}),
	}

	// Process based on result type
	if result.Error != nil {
		output.DisplayOutput = p.formatter.FormatError(result.Error)
		output.AIFeedback = p.generateErrorFeedback(result)
	} else {
		// Format display output
		output.DisplayOutput = p.formatter.Format(result)

		// Generate AI feedback
		output.AIFeedback = p.generateSuccessFeedback(result)

		// Check if summarization is needed
		if p.shouldSummarize(output.AIFeedback) {
			output.AIFeedback = p.summarizer.Summarize(output.AIFeedback, p.maxTokens/2)
			output.Summarized = true
		}
	}

	// Add metadata
	output.Metadata["tool"] = result.ToolName
	output.Metadata["duration"] = result.Duration.String()
	output.Metadata["timestamp"] = result.ExecutedAt.Format(time.RFC3339)

	// Count tokens
	output.TokenCount = p.tokenCounter.CountTokens(output.AIFeedback)

	// Cache the result
	p.cache.Set(cacheKey, output)

	output.ProcessingTime = time.Since(startTime)
	return output
}

// SetFormatter sets a custom output formatter
func (p *ResultProcessor) SetFormatter(formatter OutputFormatter) {
	p.formatter = formatter
}

// SetSummarizer sets a custom content summarizer
func (p *ResultProcessor) SetSummarizer(summarizer ContentSummarizer) {
	p.summarizer = summarizer
}

// GetCacheStats returns cache statistics
func (p *ResultProcessor) GetCacheStats() map[string]interface{} {
	return p.cache.GetStats()
}

// ClearCache clears the result cache
func (p *ResultProcessor) ClearCache() {
	p.cache.Clear()
}

// Private methods

func (p *ResultProcessor) generateCacheKey(result ToolResult) string {
	// Generate a unique key based on tool name and parameters
	key := fmt.Sprintf("%s:%s", result.ToolName, result.ToolCallID)
	return key
}

func (p *ResultProcessor) shouldSummarize(content string) bool {
	tokens := p.tokenCounter.CountTokens(content)
	return tokens > p.maxTokens/3 || p.summarizer.ShouldSummarize(content)
}

func (p *ResultProcessor) generateErrorFeedback(result ToolResult) string {
	return fmt.Sprintf("Tool '%s' execution failed: %v", result.ToolName, result.Error)
}

func (p *ResultProcessor) generateSuccessFeedback(result ToolResult) string {
	switch v := result.Result.(type) {
	case string:
		return v
	case []byte:
		return string(v)
	default:
		if data, err := json.Marshal(v); err == nil {
			return string(data)
		}
		return fmt.Sprintf("%v", v)
	}
}

// DefaultFormatter implements OutputFormatter
type DefaultFormatter struct {
	syntaxHighlighter SyntaxHighlighter
}

// SyntaxHighlighter interface for syntax highlighting
type SyntaxHighlighter interface {
	Highlight(content, language string) string
	DetectLanguage(filename string) string
}

// NewDefaultFormatter creates a new default formatter
func NewDefaultFormatter() *DefaultFormatter {
	return &DefaultFormatter{
		syntaxHighlighter: &PlainHighlighter{},
	}
}

// Format implements OutputFormatter
func (f *DefaultFormatter) Format(result ToolResult) string {
	var output strings.Builder

	// Add header
	output.WriteString(fmt.Sprintf("=== %s ===\n", result.ToolName))

	// Format based on tool type
	switch result.ToolName {
	case "read_file":
		return f.formatFileContent(result)
	case "list_files":
		return f.formatFileList(result)
	case "search_files":
		return f.formatSearchResults(result)
	case "write_file", "edit_file":
		return f.formatWriteResult(result)
	default:
		return f.formatGeneric(result)
	}
}

// FormatError implements OutputFormatter
func (f *DefaultFormatter) FormatError(err error) string {
	return fmt.Sprintf("❌ Error: %v", err)
}

func (f *DefaultFormatter) formatFileContent(result ToolResult) string {
	content, ok := result.Result.(string)
	if !ok {
		return f.formatGeneric(result)
	}

	// Extract filename from metadata if available
	filename := ""
	if result.Metadata != nil {
		if fn, ok := result.Metadata["filename"].(string); ok {
			filename = fn
		}
	}

	// Detect language and apply syntax highlighting
	if filename != "" && f.syntaxHighlighter != nil {
		lang := f.syntaxHighlighter.DetectLanguage(filename)
		if lang != "" {
			content = f.syntaxHighlighter.Highlight(content, lang)
		}
	}

	var output strings.Builder
	output.WriteString(fmt.Sprintf("📄 File: %s\n", filename))
	output.WriteString(strings.Repeat("-", 60) + "\n")
	output.WriteString(content)
	if !strings.HasSuffix(content, "\n") {
		output.WriteString("\n")
	}
	output.WriteString(strings.Repeat("-", 60) + "\n")

	return output.String()
}

func (f *DefaultFormatter) formatFileList(result ToolResult) string {
	files, ok := result.Result.([]string)
	if !ok {
		return f.formatGeneric(result)
	}

	var output strings.Builder
	output.WriteString("📁 Files and Directories:\n")
	output.WriteString(strings.Repeat("-", 40) + "\n")

	for _, file := range files {
		output.WriteString(fmt.Sprintf("  %s\n", file))
	}

	output.WriteString(strings.Repeat("-", 40) + "\n")
	output.WriteString(fmt.Sprintf("Total: %d items\n", len(files)))

	return output.String()
}

func (f *DefaultFormatter) formatSearchResults(result ToolResult) string {
	results, ok := result.Result.(map[string][]string)
	if !ok {
		return f.formatGeneric(result)
	}

	var output strings.Builder
	output.WriteString("🔍 Search Results:\n")
	output.WriteString(strings.Repeat("=", 60) + "\n")

	totalMatches := 0
	for file, matches := range results {
		output.WriteString(fmt.Sprintf("\n📄 %s (%d matches):\n", file, len(matches)))
		for i, match := range matches {
			output.WriteString(fmt.Sprintf("  %d: %s\n", i+1, match))
		}
		totalMatches += len(matches)
	}

	output.WriteString(strings.Repeat("=", 60) + "\n")
	output.WriteString(fmt.Sprintf("Total: %d matches in %d files\n", totalMatches, len(results)))

	return output.String()
}

func (f *DefaultFormatter) formatWriteResult(result ToolResult) string {
	var output strings.Builder
	output.WriteString("✅ Write operation completed\n")

	if result.Metadata != nil {
		if path, ok := result.Metadata["path"].(string); ok {
			output.WriteString(fmt.Sprintf("Path: %s\n", path))
		}
		if size, ok := result.Metadata["size"].(int); ok {
			output.WriteString(fmt.Sprintf("Size: %d bytes\n", size))
		}
	}

	return output.String()
}

func (f *DefaultFormatter) formatGeneric(result ToolResult) string {
	switch v := result.Result.(type) {
	case string:
		return v
	case []byte:
		return string(v)
	default:
		if data, err := json.MarshalIndent(v, "", "  "); err == nil {
			return string(data)
		}
		return fmt.Sprintf("%v", v)
	}
}

// PlainHighlighter provides no syntax highlighting
type PlainHighlighter struct{}

func (h *PlainHighlighter) Highlight(content, language string) string {
	return content
}

func (h *PlainHighlighter) DetectLanguage(filename string) string {
	// Simple extension-based detection
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".go":
		return "go"
	case ".js", ".jsx":
		return "javascript"
	case ".ts", ".tsx":
		return "typescript"
	case ".py":
		return "python"
	case ".java":
		return "java"
	case ".c", ".h":
		return "c"
	case ".cpp", ".cc", ".cxx", ".hpp":
		return "cpp"
	case ".rs":
		return "rust"
	case ".rb":
		return "ruby"
	case ".php":
		return "php"
	case ".sh", ".bash":
		return "bash"
	case ".yaml", ".yml":
		return "yaml"
	case ".json":
		return "json"
	case ".xml":
		return "xml"
	case ".md", ".markdown":
		return "markdown"
	default:
		return ""
	}
}

// DefaultSummarizer implements ContentSummarizer
type DefaultSummarizer struct {
	lineThreshold int
	charThreshold int
}

// NewDefaultSummarizer creates a new default summarizer
func NewDefaultSummarizer() *DefaultSummarizer {
	return &DefaultSummarizer{
		lineThreshold: 100,
		charThreshold: 5000,
	}
}

// Summarize implements ContentSummarizer
func (s *DefaultSummarizer) Summarize(content string, maxLength int) string {
	if len(content) <= maxLength {
		return content
	}

	lines := strings.Split(content, "\n")

	// Strategy 1: Take first and last portions
	if len(lines) > 20 {
		headLines := lines[:10]
		tailLines := lines[len(lines)-10:]

		summary := strings.Join(headLines, "\n")
		summary += fmt.Sprintf("\n\n... (%d lines omitted) ...\n\n", len(lines)-20)
		summary += strings.Join(tailLines, "\n")

		if len(summary) <= maxLength {
			return summary
		}
	}

	// Strategy 2: Simple truncation with ellipsis
	return content[:maxLength-3] + "..."
}

// ShouldSummarize implements ContentSummarizer
func (s *DefaultSummarizer) ShouldSummarize(content string) bool {
	lines := strings.Count(content, "\n")
	return lines > s.lineThreshold || len(content) > s.charThreshold
}

// ResultCache implementation

// NewResultCache creates a new result cache
func NewResultCache(maxSize int, maxAge time.Duration) *ResultCache {
	return &ResultCache{
		cache:   make(map[string]*CacheEntry),
		maxSize: maxSize,
		maxAge:  maxAge,
	}
}

// Get retrieves a cached result
func (c *ResultCache) Get(key string) (ProcessedOutput, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, exists := c.cache[key]
	if !exists {
		c.missCount++
		return ProcessedOutput{}, false
	}

	// Check if entry is expired
	if time.Since(entry.Timestamp) > c.maxAge {
		c.missCount++
		return ProcessedOutput{}, false
	}

	entry.HitCount++
	c.hitCount++
	return entry.Result, true
}

// Set stores a result in cache
func (c *ResultCache) Set(key string, result ProcessedOutput) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Evict oldest entry if cache is full
	if len(c.cache) >= c.maxSize {
		c.evictOldest()
	}

	c.cache[key] = &CacheEntry{
		Result:    result,
		Timestamp: time.Now(),
		HitCount:  0,
	}
}

// Clear clears the cache
func (c *ResultCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cache = make(map[string]*CacheEntry)
	c.hitCount = 0
	c.missCount = 0
}

// GetStats returns cache statistics
func (c *ResultCache) GetStats() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	total := c.hitCount + c.missCount
	hitRate := float64(0)
	if total > 0 {
		hitRate = float64(c.hitCount) / float64(total)
	}

	return map[string]interface{}{
		"size":       len(c.cache),
		"max_size":   c.maxSize,
		"hit_count":  c.hitCount,
		"miss_count": c.missCount,
		"hit_rate":   hitRate,
		"max_age":    c.maxAge.String(),
	}
}

func (c *ResultCache) evictOldest() {
	var oldestKey string
	var oldestTime time.Time

	for key, entry := range c.cache {
		if oldestKey == "" || entry.Timestamp.Before(oldestTime) {
			oldestKey = key
			oldestTime = entry.Timestamp
		}
	}

	if oldestKey != "" {
		delete(c.cache, oldestKey)
	}
}
