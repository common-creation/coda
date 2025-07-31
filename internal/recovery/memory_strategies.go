package recovery

import (
	"context"
	"fmt"
	"runtime"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/log"
)

// SessionCleanupStrategy cleans up old sessions to free memory.
type SessionCleanupStrategy struct {
	logger          *log.Logger
	state           *ApplicationState
	maxSessions     int
	sessionLifetime time.Duration
	mu              sync.Mutex
}

// NewSessionCleanupStrategy creates a new session cleanup strategy.
func NewSessionCleanupStrategy(state *ApplicationState, logger *log.Logger) *SessionCleanupStrategy {
	return &SessionCleanupStrategy{
		logger:          logger,
		state:           state,
		maxSessions:     10,             // Keep maximum 10 sessions
		sessionLifetime: 24 * time.Hour, // Sessions expire after 24 hours
	}
}

// CanRecover determines if this strategy can handle the given error.
func (s *SessionCleanupStrategy) CanRecover(err error) bool {
	if err == nil {
		return false
	}

	errMsg := strings.ToLower(err.Error())

	// Check for memory-related errors
	return strings.Contains(errMsg, "out of memory") ||
		strings.Contains(errMsg, "memory") ||
		strings.Contains(errMsg, "cannot allocate") ||
		strings.Contains(errMsg, "no space left") ||
		s.isHighMemoryUsage()
}

// Recover attempts to recover by cleaning up old sessions.
func (s *SessionCleanupStrategy) Recover(ctx context.Context, err error) error {
	s.logger.Debug("Attempting session cleanup recovery", "error", err.Error())

	if !s.CanRecover(err) {
		return fmt.Errorf("error is not recoverable by session cleanup: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.state.mu.Lock()
	defer s.state.mu.Unlock()

	initialCount := len(s.state.Sessions)
	cleanedCount := 0

	// Remove expired sessions
	now := time.Now()
	for sessionID, session := range s.state.Sessions {
		if now.Sub(session.LastUsed) > s.sessionLifetime {
			delete(s.state.Sessions, sessionID)
			cleanedCount++
			s.logger.Debug("Removed expired session", "session_id", sessionID, "last_used", session.LastUsed)
		}
	}

	// If we still have too many sessions, remove the oldest ones
	if len(s.state.Sessions) > s.maxSessions {
		// Convert to slice for sorting by last used time
		type sessionWithID struct {
			ID      string
			Session SessionData
		}

		sessions := make([]sessionWithID, 0, len(s.state.Sessions))
		for id, session := range s.state.Sessions {
			sessions = append(sessions, sessionWithID{ID: id, Session: session})
		}

		// Sort by last used time (oldest first)
		for i := 0; i < len(sessions)-1; i++ {
			for j := i + 1; j < len(sessions); j++ {
				if sessions[i].Session.LastUsed.After(sessions[j].Session.LastUsed) {
					sessions[i], sessions[j] = sessions[j], sessions[i]
				}
			}
		}

		// Remove oldest sessions
		sessionsToRemove := len(sessions) - s.maxSessions
		for i := 0; i < sessionsToRemove; i++ {
			sessionID := sessions[i].ID
			delete(s.state.Sessions, sessionID)
			cleanedCount++
			s.logger.Debug("Removed old session to reduce memory usage", "session_id", sessionID)
		}
	}

	s.logger.Info("Session cleanup completed",
		"initial_sessions", initialCount,
		"cleaned_sessions", cleanedCount,
		"remaining_sessions", len(s.state.Sessions))

	if cleanedCount == 0 {
		return fmt.Errorf("no sessions were cleaned up")
	}

	return nil
}

// Priority returns the priority of this strategy.
func (s *SessionCleanupStrategy) Priority() int {
	return 1 // High priority for memory cleanup
}

// Name returns the name of this recovery strategy.
func (s *SessionCleanupStrategy) Name() string {
	return "SessionCleanupStrategy"
}

// isHighMemoryUsage checks if the application is using too much memory.
func (s *SessionCleanupStrategy) isHighMemoryUsage() bool {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	// Check if we're using more than 512MB
	return memStats.Alloc > 512*1024*1024
}

// CacheClearStrategy clears various caches to free memory.
type CacheClearStrategy struct {
	logger *log.Logger
	caches map[string]Cache
	mu     sync.RWMutex
}

// Cache interface for different types of caches.
type Cache interface {
	Clear() error
	Size() int
	Name() string
}

// NewCacheClearStrategy creates a new cache clear strategy.
func NewCacheClearStrategy(logger *log.Logger) *CacheClearStrategy {
	return &CacheClearStrategy{
		logger: logger,
		caches: make(map[string]Cache),
	}
}

// RegisterCache registers a cache for cleanup.
func (c *CacheClearStrategy) RegisterCache(name string, cache Cache) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.caches[name] = cache
}

// CanRecover determines if this strategy can handle the given error.
func (c *CacheClearStrategy) CanRecover(err error) bool {
	if err == nil {
		return false
	}

	errMsg := strings.ToLower(err.Error())

	// Check for memory-related errors
	return strings.Contains(errMsg, "out of memory") ||
		strings.Contains(errMsg, "memory") ||
		strings.Contains(errMsg, "cache") ||
		strings.Contains(errMsg, "buffer") ||
		c.shouldClearCaches()
}

// Recover attempts to recover by clearing caches.
func (c *CacheClearStrategy) Recover(ctx context.Context, err error) error {
	c.logger.Debug("Attempting cache clear recovery", "error", err.Error())

	if !c.CanRecover(err) {
		return fmt.Errorf("error is not recoverable by cache clearing: %w", err)
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	clearedCaches := 0
	totalFreed := 0

	for name, cache := range c.caches {
		initialSize := cache.Size()

		if clearErr := cache.Clear(); clearErr != nil {
			c.logger.Warn("Failed to clear cache", "cache", name, "error", clearErr.Error())
			continue
		}

		finalSize := cache.Size()
		freed := initialSize - finalSize
		totalFreed += freed
		clearedCaches++

		c.logger.Debug("Cleared cache",
			"cache", name,
			"initial_size", initialSize,
			"freed", freed)
	}

	// Also clear Go's internal caches
	debug.FreeOSMemory()

	c.logger.Info("Cache clear recovery completed",
		"cleared_caches", clearedCaches,
		"total_freed", totalFreed)

	if clearedCaches == 0 {
		return fmt.Errorf("no caches were cleared")
	}

	return nil
}

// Priority returns the priority of this strategy.
func (c *CacheClearStrategy) Priority() int {
	return 2 // Medium priority
}

// Name returns the name of this recovery strategy.
func (c *CacheClearStrategy) Name() string {
	return "CacheClearStrategy"
}

// shouldClearCaches determines if caches should be cleared based on memory usage.
func (c *CacheClearStrategy) shouldClearCaches() bool {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	// Clear caches if heap size is over 256MB
	return memStats.HeapAlloc > 256*1024*1024
}

// GCForceStrategy forces garbage collection to free memory.
type GCForceStrategy struct {
	logger      *log.Logger
	lastGC      time.Time
	minInterval time.Duration
	mu          sync.Mutex
}

// NewGCForceStrategy creates a new GC force strategy.
func NewGCForceStrategy(logger *log.Logger) *GCForceStrategy {
	return &GCForceStrategy{
		logger:      logger,
		minInterval: 30 * time.Second, // Don't force GC more than once every 30 seconds
	}
}

// CanRecover determines if this strategy can handle the given error.
func (g *GCForceStrategy) CanRecover(err error) bool {
	if err == nil {
		return false
	}

	errMsg := strings.ToLower(err.Error())

	// Check for memory-related errors
	return strings.Contains(errMsg, "out of memory") ||
		strings.Contains(errMsg, "memory") ||
		strings.Contains(errMsg, "allocation") ||
		strings.Contains(errMsg, "heap") ||
		g.shouldForceGC()
}

// Recover attempts to recover by forcing garbage collection.
func (g *GCForceStrategy) Recover(ctx context.Context, err error) error {
	g.logger.Debug("Attempting GC force recovery", "error", err.Error())

	if !g.CanRecover(err) {
		return fmt.Errorf("error is not recoverable by forced GC: %w", err)
	}

	g.mu.Lock()
	defer g.mu.Unlock()

	// Check if enough time has passed since last forced GC
	if time.Since(g.lastGC) < g.minInterval {
		return fmt.Errorf("forced GC attempted too recently, waiting %v", g.minInterval-time.Since(g.lastGC))
	}

	// Capture memory stats before GC
	var beforeStats runtime.MemStats
	runtime.ReadMemStats(&beforeStats)

	// Force garbage collection
	g.logger.Debug("Forcing garbage collection")
	runtime.GC()
	runtime.GC() // Call twice to ensure finalizers are run

	// Force return of memory to OS
	debug.FreeOSMemory()

	// Capture memory stats after GC
	var afterStats runtime.MemStats
	runtime.ReadMemStats(&afterStats)

	g.lastGC = time.Now()

	freedMemory := int64(beforeStats.HeapAlloc) - int64(afterStats.HeapAlloc)

	g.logger.Info("Forced GC completed",
		"heap_before", beforeStats.HeapAlloc,
		"heap_after", afterStats.HeapAlloc,
		"freed_memory", freedMemory,
		"goroutines", runtime.NumGoroutine())

	// Consider recovery successful if we freed some memory
	if freedMemory > 0 {
		return nil
	}

	return fmt.Errorf("forced GC did not free significant memory")
}

// Priority returns the priority of this strategy.
func (g *GCForceStrategy) Priority() int {
	return 3 // Low priority, last resort
}

// Name returns the name of this recovery strategy.
func (g *GCForceStrategy) Name() string {
	return "GCForceStrategy"
}

// shouldForceGC determines if GC should be forced based on memory usage.
func (g *GCForceStrategy) shouldForceGC() bool {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	// Force GC if:
	// 1. Heap is large (>128MB)
	// 2. High number of goroutines (might indicate leaks)
	// 3. GC hasn't run in a while
	return memStats.HeapAlloc > 128*1024*1024 ||
		runtime.NumGoroutine() > 1000 ||
		time.Since(time.Unix(int64(memStats.LastGC/1e9), 0)) > 5*time.Minute
}

// MemoryDiagnostics provides memory usage diagnostics.
func (g *GCForceStrategy) MemoryDiagnostics() map[string]interface{} {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	return map[string]interface{}{
		"heap_alloc":      memStats.HeapAlloc,
		"heap_sys":        memStats.HeapSys,
		"heap_idle":       memStats.HeapIdle,
		"heap_inuse":      memStats.HeapInuse,
		"heap_released":   memStats.HeapReleased,
		"heap_objects":    memStats.HeapObjects,
		"stack_inuse":     memStats.StackInuse,
		"stack_sys":       memStats.StackSys,
		"num_gc":          memStats.NumGC,
		"last_gc":         time.Unix(int64(memStats.LastGC/1e9), 0),
		"gc_cpu_fraction": memStats.GCCPUFraction,
		"goroutines":      runtime.NumGoroutine(),
		"num_cgocall":     runtime.NumCgoCall(),
	}
}
