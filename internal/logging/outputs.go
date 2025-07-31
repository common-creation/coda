package logging

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// ConsoleOutput writes formatted logs to console
type ConsoleOutput struct {
	writer   io.Writer
	colorize bool
	mu       sync.Mutex
}

// NewConsoleOutput creates a new console output
func NewConsoleOutput(colorize bool) *ConsoleOutput {
	return &ConsoleOutput{
		writer:   os.Stdout,
		colorize: colorize,
	}
}

// NewConsoleOutputWithWriter creates a console output with custom writer
func NewConsoleOutputWithWriter(writer io.Writer, colorize bool) *ConsoleOutput {
	return &ConsoleOutput{
		writer:   writer,
		colorize: colorize,
	}
}

// Write outputs a log entry to console
func (c *ConsoleOutput) Write(entry *LogEntry) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	timestamp := entry.Timestamp.Format("2006-01-02 15:04:05")
	level := entry.Level.String()

	if c.colorize {
		level = c.colorizeLevel(entry.Level)
	}

	// Basic format: timestamp [LEVEL] message
	output := fmt.Sprintf("%s [%s] %s", timestamp, level, entry.Message)

	// Add fields if present
	if len(entry.Fields) > 0 {
		fieldsStr := c.formatFields(entry.Fields)
		output += fmt.Sprintf(" %s", fieldsStr)
	}

	// Add caller info for warnings and errors
	if entry.Caller != "" {
		output += fmt.Sprintf(" (%s)", entry.Caller)
	}

	output += "\n"

	// Write stack trace on separate lines for errors
	if entry.StackTrace != "" && entry.Level >= LevelError {
		output += fmt.Sprintf("Stack trace:\n%s", entry.StackTrace)
	}

	_, err := c.writer.Write([]byte(output))
	return err
}

// colorizeLevel adds ANSI color codes to log levels
func (c *ConsoleOutput) colorizeLevel(level LogLevel) string {
	const (
		colorReset  = "\033[0m"
		colorRed    = "\033[31m"
		colorYellow = "\033[33m"
		colorBlue   = "\033[34m"
		colorGray   = "\033[37m"
		colorPurple = "\033[35m"
	)

	switch level {
	case LevelDebug:
		return colorGray + "DEBUG" + colorReset
	case LevelInfo:
		return colorBlue + "INFO" + colorReset
	case LevelWarn:
		return colorYellow + "WARN" + colorReset
	case LevelError:
		return colorRed + "ERROR" + colorReset
	case LevelFatal:
		return colorPurple + "FATAL" + colorReset
	default:
		return level.String()
	}
}

// formatFields formats structured fields for console output
func (c *ConsoleOutput) formatFields(fields Fields) string {
	if len(fields) == 0 {
		return ""
	}

	result := "{"
	first := true
	for k, v := range fields {
		if !first {
			result += ", "
		}
		result += fmt.Sprintf("%s=%v", k, v)
		first = false
	}
	result += "}"
	return result
}

// Close implements the LogOutput interface
func (c *ConsoleOutput) Close() error {
	// Console output doesn't need explicit closing
	return nil
}

// FileOutput writes logs to a file with rotation support
type FileOutput struct {
	filename    string
	file        *os.File
	maxSize     int64 // bytes
	maxAge      time.Duration
	compression bool
	mu          sync.Mutex
	buffered    *bufio.Writer
}

// FileOutputConfig configures file output options
type FileOutputConfig struct {
	Filename    string
	MaxSize     int64         // Max file size in bytes
	MaxAge      time.Duration // Max age of log files
	Compression bool          // Compress rotated files
	BufferSize  int           // Buffer size for writes
}

// NewFileOutput creates a new file output
func NewFileOutput(config FileOutputConfig) (*FileOutput, error) {
	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(config.Filename), 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	file, err := os.OpenFile(config.Filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	bufferSize := config.BufferSize
	if bufferSize <= 0 {
		bufferSize = 4096 // Default buffer size
	}

	return &FileOutput{
		filename:    config.Filename,
		file:        file,
		maxSize:     config.MaxSize,
		maxAge:      config.MaxAge,
		compression: config.Compression,
		buffered:    bufio.NewWriterSize(file, bufferSize),
	}, nil
}

// Write outputs a log entry to file
func (f *FileOutput) Write(entry *LogEntry) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	// Check if rotation is needed
	if err := f.checkRotation(); err != nil {
		return fmt.Errorf("rotation check failed: %w", err)
	}

	// Format as JSON for file output
	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal log entry: %w", err)
	}

	// Write to buffered writer
	if _, err := f.buffered.Write(data); err != nil {
		return fmt.Errorf("failed to write to buffer: %w", err)
	}

	if _, err := f.buffered.WriteString("\n"); err != nil {
		return fmt.Errorf("failed to write newline: %w", err)
	}

	// Flush for error and fatal levels
	if entry.Level >= LevelError {
		if err := f.buffered.Flush(); err != nil {
			return fmt.Errorf("failed to flush buffer: %w", err)
		}
	}

	return nil
}

// checkRotation checks if log rotation is needed
func (f *FileOutput) checkRotation() error {
	if f.maxSize <= 0 {
		return nil // No size limit
	}

	stat, err := f.file.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat log file: %w", err)
	}

	if stat.Size() < f.maxSize {
		return nil // No rotation needed
	}

	return f.rotate()
}

// rotate performs log file rotation
func (f *FileOutput) rotate() error {
	// Flush and close current file
	if err := f.buffered.Flush(); err != nil {
		return fmt.Errorf("failed to flush before rotation: %w", err)
	}

	if err := f.file.Close(); err != nil {
		return fmt.Errorf("failed to close file for rotation: %w", err)
	}

	// Generate rotated filename
	timestamp := time.Now().Format("2006-01-02-15-04-05")
	ext := filepath.Ext(f.filename)
	base := f.filename[:len(f.filename)-len(ext)]
	rotatedName := fmt.Sprintf("%s.%s%s", base, timestamp, ext)

	// Move current file to rotated name
	if err := os.Rename(f.filename, rotatedName); err != nil {
		return fmt.Errorf("failed to rotate log file: %w", err)
	}

	// Create new file
	file, err := os.OpenFile(f.filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to create new log file: %w", err)
	}

	f.file = file
	f.buffered.Reset(file)

	// Compress rotated file if enabled
	if f.compression {
		go f.compressFile(rotatedName)
	}

	return nil
}

// compressFile compresses a rotated log file (placeholder implementation)
func (f *FileOutput) compressFile(filename string) {
	// TODO: Implement compression (gzip)
	// This is a placeholder for the compression functionality
}

// Close closes the file output
func (f *FileOutput) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.buffered != nil {
		if err := f.buffered.Flush(); err != nil {
			return fmt.Errorf("failed to flush buffer: %w", err)
		}
	}

	if f.file != nil {
		if err := f.file.Close(); err != nil {
			return fmt.Errorf("failed to close file: %w", err)
		}
	}

	return nil
}

// JSONOutput writes structured logs as JSON
type JSONOutput struct {
	writer io.Writer
	mu     sync.Mutex
}

// NewJSONOutput creates a new JSON output
func NewJSONOutput(writer io.Writer) *JSONOutput {
	return &JSONOutput{
		writer: writer,
	}
}

// Write outputs a log entry as JSON
func (j *JSONOutput) Write(entry *LogEntry) error {
	j.mu.Lock()
	defer j.mu.Unlock()

	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal log entry: %w", err)
	}

	data = append(data, '\n')
	_, err = j.writer.Write(data)
	return err
}

// Close implements the LogOutput interface
func (j *JSONOutput) Close() error {
	// JSON output doesn't need explicit closing
	return nil
}

// MultiOutput writes to multiple outputs simultaneously
type MultiOutput struct {
	outputs []LogOutput
}

// NewMultiOutput creates a new multi-output logger
func NewMultiOutput(outputs ...LogOutput) *MultiOutput {
	return &MultiOutput{
		outputs: outputs,
	}
}

// Write outputs to all configured outputs
func (m *MultiOutput) Write(entry *LogEntry) error {
	var errs []string

	for _, output := range m.outputs {
		if err := output.Write(entry); err != nil {
			errs = append(errs, err.Error())
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("multi-output errors: %v", errs)
	}
	return nil
}

// Close closes all outputs
func (m *MultiOutput) Close() error {
	var errs []string

	for _, output := range m.outputs {
		if err := output.Close(); err != nil {
			errs = append(errs, err.Error())
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("multi-output close errors: %v", errs)
	}
	return nil
}
