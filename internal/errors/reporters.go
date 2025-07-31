// Package errors provides error reporting implementations.
package errors

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

// FileReporter writes error reports to local files.
type FileReporter struct {
	logDir      string
	maxFileSize int64
	maxFiles    int
}

// NewFileReporter creates a new file-based error reporter.
func NewFileReporter(logDir string, maxFileSize int64, maxFiles int) *FileReporter {
	// Ensure log directory exists
	os.MkdirAll(logDir, 0755)

	return &FileReporter{
		logDir:      logDir,
		maxFileSize: maxFileSize,
		maxFiles:    maxFiles,
	}
}

// Report writes an error report to a file.
func (r *FileReporter) Report(ctx context.Context, category ErrorCategory, err error, errCtx *ErrorContext) error {
	report := ErrorReport{
		ID:        generateReportID(),
		Timestamp: time.Now(),
		Category:  category,
		Error:     err.Error(),
		Context:   errCtx,
		SystemInfo: SystemInfo{
			OS:           runtime.GOOS,
			Architecture: runtime.GOARCH,
			GoVersion:    runtime.Version(),
			NumCPU:       runtime.NumCPU(),
			NumGoroutine: runtime.NumGoroutine(),
		},
	}

	// Add memory stats for system errors
	if category == SystemError {
		var memStats runtime.MemStats
		runtime.ReadMemStats(&memStats)
		report.MemoryStats = &memStats
	}

	filename := fmt.Sprintf("error_%s_%s.json",
		category.String(),
		report.Timestamp.Format("20060102_150405"))

	filepath := filepath.Join(r.logDir, filename)

	file, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("failed to create error report file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(report); err != nil {
		return fmt.Errorf("failed to write error report: %w", err)
	}

	// Clean up old files if necessary
	r.cleanupOldFiles()

	return nil
}

// DebugReporter collects detailed debug information for development.
type DebugReporter struct {
	collectGoroutines bool
	collectMemStats   bool
	debugDir          string
}

// NewDebugReporter creates a new debug information reporter.
func NewDebugReporter(debugDir string, collectGoroutines, collectMemStats bool) *DebugReporter {
	os.MkdirAll(debugDir, 0755)

	return &DebugReporter{
		collectGoroutines: collectGoroutines,
		collectMemStats:   collectMemStats,
		debugDir:          debugDir,
	}
}

// Report collects and saves debug information.
func (r *DebugReporter) Report(ctx context.Context, category ErrorCategory, err error, errCtx *ErrorContext) error {
	debugInfo := DebugInfo{
		Timestamp:   time.Now(),
		Category:    category,
		Error:       err.Error(),
		Context:     errCtx,
		Environment: r.collectEnvironmentInfo(),
	}

	if r.collectGoroutines {
		debugInfo.Goroutines = r.collectGoroutineInfo()
	}

	if r.collectMemStats {
		debugInfo.MemoryStats = r.collectMemoryStats()
	}

	filename := fmt.Sprintf("debug_%s_%s.json",
		category.String(),
		debugInfo.Timestamp.Format("20060102_150405"))

	filepath := filepath.Join(r.debugDir, filename)

	file, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("failed to create debug report: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")

	return encoder.Encode(debugInfo)
}

// CrashDumpReporter creates crash dumps for critical errors.
type CrashDumpReporter struct {
	dumpDir   string
	threshold ErrorCategory
}

// NewCrashDumpReporter creates a new crash dump reporter.
func NewCrashDumpReporter(dumpDir string, threshold ErrorCategory) *CrashDumpReporter {
	os.MkdirAll(dumpDir, 0755)

	return &CrashDumpReporter{
		dumpDir:   dumpDir,
		threshold: threshold,
	}
}

// Report creates a crash dump for severe errors.
func (r *CrashDumpReporter) Report(ctx context.Context, category ErrorCategory, err error, errCtx *ErrorContext) error {
	// Only create crash dumps for severe errors
	if category == UserError {
		return nil
	}

	crashDump := CrashDump{
		Timestamp:   time.Now(),
		Category:    category,
		Error:       err.Error(),
		Context:     errCtx,
		SystemInfo:  r.collectSystemInfo(),
		ProcessInfo: r.collectProcessInfo(),
		StackTrace:  errCtx.StackTrace,
		Goroutines:  r.collectAllGoroutines(),
		MemoryDump:  r.collectMemoryDump(),
	}

	filename := fmt.Sprintf("crash_%s_%s.json",
		category.String(),
		crashDump.Timestamp.Format("20060102_150405"))

	filepath := filepath.Join(r.dumpDir, filename)

	file, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("failed to create crash dump: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")

	return encoder.Encode(crashDump)
}

// StatisticsReporter collects anonymized error statistics.
type StatisticsReporter struct {
	statsFile string
	stats     map[string]*ErrorStats
}

// NewStatisticsReporter creates a new statistics reporter.
func NewStatisticsReporter(statsFile string) *StatisticsReporter {
	return &StatisticsReporter{
		statsFile: statsFile,
		stats:     make(map[string]*ErrorStats),
	}
}

// Report updates error statistics.
func (r *StatisticsReporter) Report(ctx context.Context, category ErrorCategory, err error, errCtx *ErrorContext) error {
	categoryKey := category.String()

	if r.stats[categoryKey] == nil {
		r.stats[categoryKey] = &ErrorStats{
			Category:   category,
			Count:      0,
			FirstSeen:  time.Now(),
			LastSeen:   time.Now(),
			ErrorTypes: make(map[string]int),
		}
	}

	stats := r.stats[categoryKey]
	stats.Count++
	stats.LastSeen = time.Now()

	// Anonymize error message - only keep error type
	errorType := fmt.Sprintf("%T", err)
	stats.ErrorTypes[errorType]++

	// Periodically save statistics
	return r.saveStats()
}

// Data structures for error reporting

// ErrorReport represents a complete error report.
type ErrorReport struct {
	ID          string            `json:"id"`
	Timestamp   time.Time         `json:"timestamp"`
	Category    ErrorCategory     `json:"category"`
	Error       string            `json:"error"`
	Context     *ErrorContext     `json:"context"`
	SystemInfo  SystemInfo        `json:"system_info"`
	MemoryStats *runtime.MemStats `json:"memory_stats,omitempty"`
}

// DebugInfo contains detailed debug information.
type DebugInfo struct {
	Timestamp   time.Time         `json:"timestamp"`
	Category    ErrorCategory     `json:"category"`
	Error       string            `json:"error"`
	Context     *ErrorContext     `json:"context"`
	Environment map[string]string `json:"environment"`
	Goroutines  []GoroutineInfo   `json:"goroutines,omitempty"`
	MemoryStats *runtime.MemStats `json:"memory_stats,omitempty"`
}

// CrashDump contains comprehensive crash information.
type CrashDump struct {
	Timestamp   time.Time       `json:"timestamp"`
	Category    ErrorCategory   `json:"category"`
	Error       string          `json:"error"`
	Context     *ErrorContext   `json:"context"`
	SystemInfo  SystemInfo      `json:"system_info"`
	ProcessInfo ProcessInfo     `json:"process_info"`
	StackTrace  []StackFrame    `json:"stack_trace"`
	Goroutines  []GoroutineInfo `json:"goroutines"`
	MemoryDump  MemoryDump      `json:"memory_dump"`
}

// SystemInfo contains system-level information.
type SystemInfo struct {
	OS           string `json:"os"`
	Architecture string `json:"architecture"`
	GoVersion    string `json:"go_version"`
	NumCPU       int    `json:"num_cpu"`
	NumGoroutine int    `json:"num_goroutine"`
}

// ProcessInfo contains process-level information.
type ProcessInfo struct {
	PID         int      `json:"pid"`
	CommandLine []string `json:"command_line"`
	WorkingDir  string   `json:"working_dir"`
	Environment []string `json:"environment,omitempty"`
}

// GoroutineInfo contains information about a goroutine.
type GoroutineInfo struct {
	ID    int    `json:"id"`
	State string `json:"state"`
	Stack string `json:"stack"`
}

// MemoryDump contains memory usage information.
type MemoryDump struct {
	Alloc      uint64 `json:"alloc"`
	TotalAlloc uint64 `json:"total_alloc"`
	Sys        uint64 `json:"sys"`
	NumGC      uint32 `json:"num_gc"`
	HeapAlloc  uint64 `json:"heap_alloc"`
	HeapSys    uint64 `json:"heap_sys"`
	HeapInuse  uint64 `json:"heap_inuse"`
	StackInuse uint64 `json:"stack_inuse"`
	StackSys   uint64 `json:"stack_sys"`
}

// ErrorStats contains anonymized error statistics.
type ErrorStats struct {
	Category   ErrorCategory  `json:"category"`
	Count      int64          `json:"count"`
	FirstSeen  time.Time      `json:"first_seen"`
	LastSeen   time.Time      `json:"last_seen"`
	ErrorTypes map[string]int `json:"error_types"`
}

// Helper methods for reporters

func (r *FileReporter) cleanupOldFiles() {
	// Implementation for cleaning up old log files
	// This is a simplified version - in production, you'd want more sophisticated cleanup
}

func (r *DebugReporter) collectEnvironmentInfo() map[string]string {
	env := make(map[string]string)

	// Collect relevant environment variables (be careful about secrets)
	relevantVars := []string{"GO_ENV", "DEBUG", "LOG_LEVEL", "HOME", "PWD"}

	for _, varName := range relevantVars {
		if value := os.Getenv(varName); value != "" {
			env[varName] = value
		}
	}

	return env
}

func (r *DebugReporter) collectGoroutineInfo() []GoroutineInfo {
	// This is a simplified implementation
	// In practice, you'd parse runtime.Stack() output
	return []GoroutineInfo{
		{
			ID:    1,
			State: "running",
			Stack: "simplified stack trace",
		},
	}
}

func (r *DebugReporter) collectMemoryStats() *runtime.MemStats {
	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)
	return &stats
}

func (r *CrashDumpReporter) collectSystemInfo() SystemInfo {
	return SystemInfo{
		OS:           runtime.GOOS,
		Architecture: runtime.GOARCH,
		GoVersion:    runtime.Version(),
		NumCPU:       runtime.NumCPU(),
		NumGoroutine: runtime.NumGoroutine(),
	}
}

func (r *CrashDumpReporter) collectProcessInfo() ProcessInfo {
	pid := os.Getpid()
	wd, _ := os.Getwd()

	return ProcessInfo{
		PID:         pid,
		CommandLine: os.Args,
		WorkingDir:  wd,
		// Environment is omitted to avoid leaking secrets
	}
}

func (r *CrashDumpReporter) collectAllGoroutines() []GoroutineInfo {
	// Simplified implementation
	return []GoroutineInfo{
		{
			ID:    1,
			State: "running",
			Stack: "main goroutine stack",
		},
	}
}

func (r *CrashDumpReporter) collectMemoryDump() MemoryDump {
	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)

	return MemoryDump{
		Alloc:      stats.Alloc,
		TotalAlloc: stats.TotalAlloc,
		Sys:        stats.Sys,
		NumGC:      stats.NumGC,
		HeapAlloc:  stats.HeapAlloc,
		HeapSys:    stats.HeapSys,
		HeapInuse:  stats.HeapInuse,
		StackInuse: stats.StackInuse,
		StackSys:   stats.StackSys,
	}
}

func (r *StatisticsReporter) saveStats() error {
	file, err := os.Create(r.statsFile)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(r.stats)
}

func generateReportID() string {
	return fmt.Sprintf("report_%d", time.Now().UnixNano())
}
