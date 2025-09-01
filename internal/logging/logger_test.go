package logging

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestLogger_BasicLogging(t *testing.T) {
	var buf bytes.Buffer
	output := NewConsoleOutputWithWriter(&buf, false)

	logger := NewLogger(LevelDebug)
	logger.outputs = []LogOutput{output}

	logger.Info("test message")

	logOutput := buf.String()
	if !strings.Contains(logOutput, "test message") {
		t.Errorf("Expected log to contain 'test message', got: %s", logOutput)
	}
	if !strings.Contains(logOutput, "[INFO]") {
		t.Errorf("Expected log to contain '[INFO]', got: %s", logOutput)
	}
}

func TestLogger_WithFields(t *testing.T) {
	var buf bytes.Buffer
	output := NewConsoleOutputWithWriter(&buf, false)

	logger := NewLogger(LevelDebug)
	logger.outputs = []LogOutput{output}

	logger.InfoWith("test message", Fields{
		"user_id": 123,
		"action":  "test",
	})

	logOutput := buf.String()
	if !strings.Contains(logOutput, "user_id=123") {
		t.Errorf("Expected log to contain field 'user_id=123', got: %s", logOutput)
	}
	if !strings.Contains(logOutput, "action=test") {
		t.Errorf("Expected log to contain field 'action=test', got: %s", logOutput)
	}
}

func TestLogger_LevelFiltering(t *testing.T) {
	var buf bytes.Buffer
	output := NewConsoleOutputWithWriter(&buf, false)

	logger := NewLogger(LevelWarn)
	logger.outputs = []LogOutput{output}

	logger.Debug("debug message")
	logger.Info("info message")
	logger.Warn("warn message")

	logOutput := buf.String()
	if strings.Contains(logOutput, "debug message") {
		t.Errorf("Debug message should not appear with WARN level, got: %s", logOutput)
	}
	if strings.Contains(logOutput, "info message") {
		t.Errorf("Info message should not appear with WARN level, got: %s", logOutput)
	}
	if !strings.Contains(logOutput, "warn message") {
		t.Errorf("Warn message should appear with WARN level, got: %s", logOutput)
	}
}

func TestLogger_WithMethod(t *testing.T) {
	var buf bytes.Buffer
	output := NewConsoleOutputWithWriter(&buf, false)

	logger := NewLogger(LevelDebug)
	logger.outputs = []LogOutput{output}

	contextLogger := logger.With(Fields{"component": "test"})
	contextLogger.Info("test message")

	logOutput := buf.String()
	if !strings.Contains(logOutput, "component=test") {
		t.Errorf("Expected log to contain field 'component=test', got: %s", logOutput)
	}
}

func TestJSONOutput(t *testing.T) {
	var buf bytes.Buffer
	output := NewJSONOutput(&buf)

	logger := NewLogger(LevelDebug)
	logger.outputs = []LogOutput{output}

	logger.InfoWith("test message", Fields{
		"user_id": 123,
		"action":  "test",
	})

	var logEntry LogEntry
	if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
		t.Fatalf("Failed to parse JSON log output: %v", err)
	}

	if logEntry.Message != "test message" {
		t.Errorf("Expected message 'test message', got: %s", logEntry.Message)
	}
	if logEntry.Level != LevelInfo {
		t.Errorf("Expected level INFO, got: %s", logEntry.Level)
	}
	if logEntry.Fields["user_id"] != float64(123) {
		t.Errorf("Expected user_id field to be 123, got: %v", logEntry.Fields["user_id"])
	}
}

func TestSanitizer(t *testing.T) {
	sanitizer := NewDefaultSanitizer()

	fields := Fields{
		"api_key": "secret123456",
		"user":    "john",
		"token":   "abc123",
		"normal":  "value",
	}

	sanitized := sanitizer.Sanitize(fields)

	if !strings.Contains(sanitized["api_key"].(string), "***") {
		t.Errorf("Expected api_key to be sanitized, got: %s", sanitized["api_key"])
	}
	if !strings.Contains(sanitized["token"].(string), "***") {
		t.Errorf("Expected token to be sanitized, got: %s", sanitized["token"])
	}
	if sanitized["user"] != "john" {
		t.Errorf("Expected user field to remain unchanged, got: %s", sanitized["user"])
	}
	if sanitized["normal"] != "value" {
		t.Errorf("Expected normal field to remain unchanged, got: %s", sanitized["normal"])
	}
}

func TestContextIntegration(t *testing.T) {
	logger := NewLogger(LevelDebug)

	ctx := WithLogger(context.Background(), logger)
	retrievedLogger := FromContext(ctx)

	if retrievedLogger != logger {
		t.Errorf("Expected to retrieve the same logger from context")
	}

	// Test with context without logger
	emptyCtx := context.Background()
	defaultLogger := FromContext(emptyCtx)

	if defaultLogger == nil {
		t.Errorf("Expected to get default logger when context has no logger")
	}
}

func TestConfigureLogger(t *testing.T) {
	config := LoggingConfig{
		Level: "debug",
		Outputs: []OutputConfig{
			{
				Type:   "console",
				Format: "text",
				Options: map[string]interface{}{
					"colorize": false,
				},
			},
		},
		Privacy: PrivacyConfig{
			Enabled:       true,
			SensitiveKeys: []string{"secret"},
		},
	}

	logger, err := ConfigureLogger(config)
	if err != nil {
		t.Fatalf("Failed to configure logger: %v", err)
	}

	if logger.level != LevelDebug {
		t.Errorf("Expected debug level, got: %v", logger.level)
	}

	if len(logger.outputs) != 1 {
		t.Errorf("Expected 1 output, got: %d", len(logger.outputs))
	}

	if logger.sanitizer == nil {
		t.Errorf("Expected sanitizer to be configured")
	}
}

func TestMultiOutput(t *testing.T) {
	var buf1, buf2 bytes.Buffer

	output1 := NewConsoleOutputWithWriter(&buf1, false)
	output2 := NewJSONOutput(&buf2)
	multiOutput := NewMultiOutput(output1, output2)

	logger := NewLogger(LevelDebug)
	logger.outputs = []LogOutput{multiOutput}

	logger.Info("test message")

	// Check console output
	consoleOutput := buf1.String()
	if !strings.Contains(consoleOutput, "test message") {
		t.Errorf("Console output should contain message, got: %s", consoleOutput)
	}

	// Check JSON output
	var logEntry LogEntry
	if err := json.Unmarshal(buf2.Bytes(), &logEntry); err != nil {
		t.Fatalf("Failed to parse JSON output: %v", err)
	}
	if logEntry.Message != "test message" {
		t.Errorf("JSON output should contain message, got: %s", logEntry.Message)
	}
}

func TestParseLogLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected LogLevel
		hasError bool
	}{
		{"debug", LevelDebug, false},
		{"DEBUG", LevelDebug, false},
		{"info", LevelInfo, false},
		{"INFO", LevelInfo, false},
		{"warn", LevelWarn, false},
		{"warning", LevelWarn, false},
		{"error", LevelError, false},
		{"fatal", LevelFatal, false},
		{"invalid", LevelInfo, true},
	}

	for _, test := range tests {
		level, err := parseLogLevel(test.input)

		if test.hasError {
			if err == nil {
				t.Errorf("Expected error for input %q, but got none", test.input)
			}
		} else {
			if err != nil {
				t.Errorf("Expected no error for input %q, but got: %v", test.input, err)
			}
			if level != test.expected {
				t.Errorf("Expected level %v for input %q, got: %v", test.expected, test.input, level)
			}
		}
	}
}

func TestLoggerConcurrency(t *testing.T) {
	var buf bytes.Buffer
	output := NewConsoleOutputWithWriter(&buf, false)

	logger := NewLogger(LevelDebug)
	logger.outputs = []LogOutput{output}

	// Test concurrent logging
	done := make(chan bool)

	for i := 0; i < 10; i++ {
		go func(id int) {
			logger.InfoWith("concurrent message", Fields{"goroutine": id})
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	logOutput := buf.String()
	lines := strings.Split(strings.TrimSpace(logOutput), "\n")

	if len(lines) != 10 {
		t.Errorf("Expected 10 log lines, got: %d", len(lines))
	}

	// Check that all messages contain the expected text
	for _, line := range lines {
		if !strings.Contains(line, "concurrent message") {
			t.Errorf("Expected all lines to contain 'concurrent message', line: %s", line)
		}
	}
}
