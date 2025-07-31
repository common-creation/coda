package recovery

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/log"
)

// PanicInfo holds information about a panic.
type PanicInfo struct {
	Value      interface{} `json:"value"`
	StackTrace string      `json:"stack_trace"`
	Timestamp  time.Time   `json:"timestamp"`
	Goroutine  int         `json:"goroutine"`
	SessionID  string      `json:"session_id"`
}

// PanicRecoveryStrategy handles panic recovery with stack trace saving and state restoration.
type PanicRecoveryStrategy struct {
	logger       *log.Logger
	state        *ApplicationState
	crashLogDir  string
	maxCrashLogs int
	mu           sync.RWMutex
	lastPanic    time.Time
	panicCount   int
}

// NewPanicRecoveryStrategy creates a new panic recovery strategy.
func NewPanicRecoveryStrategy(state *ApplicationState, logger *log.Logger) *PanicRecoveryStrategy {
	crashLogDir := filepath.Join(os.TempDir(), "coda", "crash-logs")
	if err := os.MkdirAll(crashLogDir, 0755); err != nil {
		logger.Warn("Failed to create crash log directory", "error", err.Error())
	}

	return &PanicRecoveryStrategy{
		logger:       logger,
		state:        state,
		crashLogDir:  crashLogDir,
		maxCrashLogs: 10, // Keep maximum 10 crash logs
	}
}

// CanRecover determines if this strategy can handle the given error.
func (p *PanicRecoveryStrategy) CanRecover(err error) bool {
	if err == nil {
		return false
	}

	errMsg := strings.ToLower(err.Error())

	// Check for panic-related errors
	return strings.Contains(errMsg, "panic") ||
		strings.Contains(errMsg, "runtime error") ||
		strings.Contains(errMsg, "nil pointer") ||
		strings.Contains(errMsg, "index out of range") ||
		strings.Contains(errMsg, "slice bounds") ||
		strings.Contains(errMsg, "division by zero") ||
		p.detectPanicCondition()
}

// Recover attempts to recover from panic conditions.
func (p *PanicRecoveryStrategy) Recover(ctx context.Context, err error) error {
	p.logger.Debug("Attempting panic recovery", "error", err.Error())

	if !p.CanRecover(err) {
		return fmt.Errorf("error is not recoverable by panic recovery: %w", err)
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	now := time.Now()
	p.panicCount++
	p.lastPanic = now

	// Create panic info
	panicInfo := &PanicInfo{
		Value:      err.Error(),
		StackTrace: string(debug.Stack()),
		Timestamp:  now,
		Goroutine:  runtime.NumGoroutine(),
		SessionID:  p.getCurrentSessionID(),
	}

	// Save crash log
	if crashLogErr := p.saveCrashLog(panicInfo); crashLogErr != nil {
		p.logger.Warn("Failed to save crash log", "error", crashLogErr.Error())
	}

	// Attempt partial state restoration
	if restoreErr := p.restorePartialState(); restoreErr != nil {
		p.logger.Warn("Failed to restore partial state", "error", restoreErr.Error())
	}

	// Clean up resources
	p.performEmergencyCleanup()

	p.logger.Info("Panic recovery completed",
		"panic_count", p.panicCount,
		"goroutines", runtime.NumGoroutine(),
		"last_panic", p.lastPanic)

	return nil
}

// Priority returns the priority of this strategy.
func (p *PanicRecoveryStrategy) Priority() int {
	return 1 // High priority for panic recovery
}

// Name returns the name of this recovery strategy.
func (p *PanicRecoveryStrategy) Name() string {
	return "PanicRecoveryStrategy"
}

// saveCrashLog saves panic information to a crash log file.
func (p *PanicRecoveryStrategy) saveCrashLog(panicInfo *PanicInfo) error {
	// Clean up old crash logs first
	p.cleanupOldCrashLogs()

	filename := fmt.Sprintf("crash_%s.log", panicInfo.Timestamp.Format("20060102_150405"))
	filepath := filepath.Join(p.crashLogDir, filename)

	file, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("failed to create crash log file: %w", err)
	}
	defer file.Close()

	// Write crash information
	content := fmt.Sprintf(`PANIC REPORT
================================================================================
Timestamp: %s
Session ID: %s
Goroutines: %d
Panic Count: %d

PANIC VALUE:
%v

STACK TRACE:
%s

SYSTEM INFO:
Go Version: %s
OS: %s
Arch: %s
CPUs: %d

MEMORY STATS:
%s
================================================================================
`,
		panicInfo.Timestamp.Format(time.RFC3339),
		panicInfo.SessionID,
		panicInfo.Goroutine,
		p.panicCount,
		panicInfo.Value,
		panicInfo.StackTrace,
		runtime.Version(),
		runtime.GOOS,
		runtime.GOARCH,
		runtime.NumCPU(),
		p.getMemoryStatsString())

	if _, err := file.WriteString(content); err != nil {
		return fmt.Errorf("failed to write crash log: %w", err)
	}

	p.logger.Info("Crash log saved", "file", filepath)
	return nil
}

// restorePartialState attempts to restore the application to a consistent state.
func (p *PanicRecoveryStrategy) restorePartialState() error {
	if p.state == nil {
		return fmt.Errorf("no state available for restoration")
	}

	p.state.mu.Lock()
	defer p.state.mu.Unlock()

	// Clear potentially corrupted data
	for sessionID, session := range p.state.Sessions {
		// Keep only recent sessions
		if time.Since(session.LastUsed) > 1*time.Hour {
			delete(p.state.Sessions, sessionID)
			p.logger.Debug("Removed old session during panic recovery", "session_id", sessionID)
		}
	}

	// Reset UI state to safe defaults
	p.state.UIState = UIState{
		ActiveView:      "chat", // Safe default view
		WindowSize:      WindowSize{Width: 800, Height: 600},
		Settings:        make(map[string]interface{}),
		LastInteraction: time.Now(),
	}

	p.state.LastUpdate = time.Now()

	p.logger.Debug("Partial state restoration completed")
	return nil
}

// performEmergencyCleanup performs emergency cleanup of resources.
func (p *PanicRecoveryStrategy) performEmergencyCleanup() {
	// Force garbage collection
	runtime.GC()
	runtime.GC() // Call twice to ensure finalizers run

	// Free OS memory
	debug.FreeOSMemory()

	// Set GC target percentage to be more aggressive
	debug.SetGCPercent(50)

	p.logger.Debug("Emergency cleanup completed")
}

// getCurrentSessionID gets the current session ID (placeholder implementation).
func (p *PanicRecoveryStrategy) getCurrentSessionID() string {
	if p.state != nil {
		p.state.mu.RLock()
		defer p.state.mu.RUnlock()

		// Return the most recently used session ID
		var mostRecent string
		var mostRecentTime time.Time

		for id, session := range p.state.Sessions {
			if session.LastUsed.After(mostRecentTime) {
				mostRecent = id
				mostRecentTime = session.LastUsed
			}
		}

		if mostRecent != "" {
			return mostRecent
		}
	}

	return fmt.Sprintf("unknown_%d", time.Now().UnixNano())
}

// detectPanicCondition detects conditions that might lead to panics.
func (p *PanicRecoveryStrategy) detectPanicCondition() bool {
	// Check for high goroutine count (potential goroutine leak)
	if runtime.NumGoroutine() > 10000 {
		return true
	}

	// Check for recent panics
	if time.Since(p.lastPanic) < 5*time.Minute && p.panicCount > 0 {
		return true
	}

	return false
}

// cleanupOldCrashLogs removes old crash log files.
func (p *PanicRecoveryStrategy) cleanupOldCrashLogs() {
	entries, err := os.ReadDir(p.crashLogDir)
	if err != nil {
		return
	}

	if len(entries) <= p.maxCrashLogs {
		return
	}

	// Sort by modification time and remove oldest files
	// This is a simplified implementation
	for i, entry := range entries {
		if i < len(entries)-p.maxCrashLogs {
			filepath := filepath.Join(p.crashLogDir, entry.Name())
			if err := os.Remove(filepath); err != nil {
				p.logger.Warn("Failed to remove old crash log", "file", filepath, "error", err.Error())
			}
		}
	}
}

// getMemoryStatsString returns formatted memory statistics.
func (p *PanicRecoveryStrategy) getMemoryStatsString() string {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	return fmt.Sprintf(`Heap Alloc: %d bytes
Heap Sys: %d bytes
Heap Idle: %d bytes
Heap Inuse: %d bytes
Total Alloc: %d bytes
Sys: %d bytes
Mallocs: %d
Frees: %d
Num GC: %d
GC CPU Fraction: %.4f`,
		memStats.HeapAlloc,
		memStats.HeapSys,
		memStats.HeapIdle,
		memStats.HeapInuse,
		memStats.TotalAlloc,
		memStats.Sys,
		memStats.Mallocs,
		memStats.Frees,
		memStats.NumGC,
		memStats.GCCPUFraction)
}

// SafeModeStrategy switches the application to safe mode when critical errors occur.
type SafeModeStrategy struct {
	logger         *log.Logger
	state          *ApplicationState
	safeModeActive bool
	mu             sync.RWMutex
}

// NewSafeModeStrategy creates a new safe mode strategy.
func NewSafeModeStrategy(state *ApplicationState, logger *log.Logger) *SafeModeStrategy {
	return &SafeModeStrategy{
		logger: logger,
		state:  state,
	}
}

// CanRecover determines if this strategy can handle the given error.
func (s *SafeModeStrategy) CanRecover(err error) bool {
	if err == nil {
		return false
	}

	errMsg := strings.ToLower(err.Error())

	// Safe mode is a last resort for critical errors
	return strings.Contains(errMsg, "panic") ||
		strings.Contains(errMsg, "fatal") ||
		strings.Contains(errMsg, "critical") ||
		strings.Contains(errMsg, "corrupted") ||
		strings.Contains(errMsg, "system failure") ||
		s.shouldActivateSafeMode()
}

// Recover attempts to recover by activating safe mode.
func (s *SafeModeStrategy) Recover(ctx context.Context, err error) error {
	s.logger.Debug("Attempting safe mode recovery", "error", err.Error())

	if !s.CanRecover(err) {
		return fmt.Errorf("error is not recoverable by safe mode: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.safeModeActive {
		return fmt.Errorf("safe mode is already active")
	}

	s.safeModeActive = true

	// Configure safe mode settings
	s.configureSafeMode()

	s.logger.Warn("Safe mode activated", "reason", err.Error())

	return nil
}

// Priority returns the priority of this strategy.
func (s *SafeModeStrategy) Priority() int {
	return 10 // Lowest priority, last resort
}

// Name returns the name of this recovery strategy.
func (s *SafeModeStrategy) Name() string {
	return "SafeModeStrategy"
}

// configureSafeMode configures the application for safe mode operation.
func (s *SafeModeStrategy) configureSafeMode() {
	if s.state == nil {
		return
	}

	s.state.mu.Lock()
	defer s.state.mu.Unlock()

	// Clear all sessions except the most recent one
	var mostRecent string
	var mostRecentTime time.Time

	for id, session := range s.state.Sessions {
		if session.LastUsed.After(mostRecentTime) {
			mostRecent = id
			mostRecentTime = session.LastUsed
		}
	}

	// Keep only the most recent session
	newSessions := make(map[string]SessionData)
	if mostRecent != "" {
		newSessions[mostRecent] = s.state.Sessions[mostRecent]
	}
	s.state.Sessions = newSessions

	// Reset UI to minimal state
	s.state.UIState = UIState{
		ActiveView: "safe_mode",
		WindowSize: WindowSize{Width: 600, Height: 400},
		Settings: map[string]interface{}{
			"safe_mode": true,
			"features_disabled": []string{
				"file_operations",
				"network_requests",
				"plugin_execution",
			},
		},
		LastInteraction: time.Now(),
	}

	s.state.LastUpdate = time.Now()

	s.logger.Info("Safe mode configuration completed",
		"sessions_kept", len(s.state.Sessions),
		"active_view", s.state.UIState.ActiveView)
}

// IsActive returns whether safe mode is currently active.
func (s *SafeModeStrategy) IsActive() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.safeModeActive
}

// Deactivate attempts to deactivate safe mode.
func (s *SafeModeStrategy) Deactivate() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.safeModeActive {
		return fmt.Errorf("safe mode is not active")
	}

	// Perform safety checks before deactivating
	if !s.systemHealthCheck() {
		return fmt.Errorf("system health check failed, cannot deactivate safe mode")
	}

	s.safeModeActive = false

	// Restore normal operation
	if s.state != nil {
		s.state.mu.Lock()
		s.state.UIState.Settings["safe_mode"] = false
		delete(s.state.UIState.Settings, "features_disabled")
		s.state.UIState.ActiveView = "chat" // Return to normal view
		s.state.mu.Unlock()
	}

	s.logger.Info("Safe mode deactivated")
	return nil
}

// shouldActivateSafeMode determines if safe mode should be activated.
func (s *SafeModeStrategy) shouldActivateSafeMode() bool {
	// Check for critical system conditions
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	// Activate safe mode if:
	// 1. Very high memory usage (>1GB)
	// 2. Excessive goroutines (>20,000)
	// 3. High GC pressure
	return memStats.HeapAlloc > 1024*1024*1024 ||
		runtime.NumGoroutine() > 20000 ||
		memStats.GCCPUFraction > 0.5
}

// systemHealthCheck performs a basic system health check.
func (s *SafeModeStrategy) systemHealthCheck() bool {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	// Check if system conditions have improved
	return memStats.HeapAlloc < 512*1024*1024 && // Less than 512MB
		runtime.NumGoroutine() < 1000 && // Reasonable goroutine count
		memStats.GCCPUFraction < 0.1 // Low GC pressure
}
