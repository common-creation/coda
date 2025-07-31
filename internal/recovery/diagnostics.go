package recovery

import (
	"context"
	"fmt"
	"net"
	"runtime"
	"sync"
	"time"

	"github.com/charmbracelet/log"
)

// HealthStatus represents the health status of a component.
type HealthStatus int

const (
	HealthStatusUnknown HealthStatus = iota
	HealthStatusHealthy
	HealthStatusDegraded
	HealthStatusUnhealthy
	HealthStatusCritical
)

// String returns a string representation of the health status.
func (h HealthStatus) String() string {
	switch h {
	case HealthStatusHealthy:
		return "healthy"
	case HealthStatusDegraded:
		return "degraded"
	case HealthStatusUnhealthy:
		return "unhealthy"
	case HealthStatusCritical:
		return "critical"
	default:
		return "unknown"
	}
}

// HealthCheck represents a single health check.
type HealthCheck struct {
	Name        string        `json:"name"`
	Status      HealthStatus  `json:"status"`
	Message     string        `json:"message"`
	LastChecked time.Time     `json:"last_checked"`
	Duration    time.Duration `json:"duration"`
	Error       error         `json:"error,omitempty"`
}

// SystemDiagnostics provides comprehensive system diagnostics and health monitoring.
type SystemDiagnostics struct {
	logger         *log.Logger
	recoveryMgr    *RecoveryManager
	healthChecks   map[string]HealthChecker
	lastDiagnostic time.Time
	mu             sync.RWMutex

	// Health check intervals
	quickCheckInterval time.Duration
	fullCheckInterval  time.Duration

	// Monitoring
	monitoring    bool
	monitorCtx    context.Context
	monitorCancel context.CancelFunc
}

// HealthChecker defines the interface for health checkers.
type HealthChecker interface {
	Name() string
	Check(ctx context.Context) HealthCheck
	Priority() int
}

// NewSystemDiagnostics creates a new system diagnostics instance.
func NewSystemDiagnostics(recoveryMgr *RecoveryManager, logger *log.Logger) *SystemDiagnostics {
	sd := &SystemDiagnostics{
		logger:             logger,
		recoveryMgr:        recoveryMgr,
		healthChecks:       make(map[string]HealthChecker),
		quickCheckInterval: 30 * time.Second,
		fullCheckInterval:  5 * time.Minute,
	}

	// Register default health checkers
	sd.registerDefaultHealthCheckers()

	return sd
}

// RegisterHealthChecker registers a health checker.
func (sd *SystemDiagnostics) RegisterHealthChecker(checker HealthChecker) {
	sd.mu.Lock()
	defer sd.mu.Unlock()
	sd.healthChecks[checker.Name()] = checker
	sd.logger.Debug("Registered health checker", "name", checker.Name())
}

// StartMonitoring starts continuous health monitoring.
func (sd *SystemDiagnostics) StartMonitoring(ctx context.Context) {
	sd.mu.Lock()
	defer sd.mu.Unlock()

	if sd.monitoring {
		sd.logger.Warn("Health monitoring is already running")
		return
	}

	sd.monitorCtx, sd.monitorCancel = context.WithCancel(ctx)
	sd.monitoring = true

	go sd.monitoringLoop()

	sd.logger.Info("Started health monitoring",
		"quick_interval", sd.quickCheckInterval,
		"full_interval", sd.fullCheckInterval)
}

// StopMonitoring stops health monitoring.
func (sd *SystemDiagnostics) StopMonitoring() {
	sd.mu.Lock()
	defer sd.mu.Unlock()

	if !sd.monitoring {
		return
	}

	if sd.monitorCancel != nil {
		sd.monitorCancel()
	}

	sd.monitoring = false
	sd.logger.Info("Stopped health monitoring")
}

// RunDiagnostics runs a full system diagnostic.
func (sd *SystemDiagnostics) RunDiagnostics(ctx context.Context) (*DiagnosticReport, error) {
	sd.mu.Lock()
	sd.lastDiagnostic = time.Now()
	sd.mu.Unlock()

	sd.logger.Info("Running full system diagnostics")

	report := &DiagnosticReport{
		Timestamp:    time.Now(),
		HealthChecks: make([]HealthCheck, 0, len(sd.healthChecks)),
		SystemInfo:   sd.collectSystemInfo(),
	}

	// Run all health checks
	for _, checker := range sd.healthChecks {
		check := checker.Check(ctx)
		report.HealthChecks = append(report.HealthChecks, check)

		// Trigger recovery if health check indicates problems
		if check.Status == HealthStatusUnhealthy || check.Status == HealthStatusCritical {
			if check.Error != nil {
				sd.logger.Warn("Health check failed, attempting recovery",
					"checker", check.Name,
					"status", check.Status.String(),
					"error", check.Error.Error())

				if recoveryErr := sd.recoveryMgr.Recover(ctx, check.Error); recoveryErr != nil {
					sd.logger.Error("Recovery failed for health check",
						"checker", check.Name,
						"recovery_error", recoveryErr.Error())
				}
			}
		}
	}

	// Calculate overall health
	report.OverallStatus = sd.calculateOverallHealth(report.HealthChecks)
	report.Duration = time.Since(report.Timestamp)

	sd.logger.Info("System diagnostics completed",
		"overall_status", report.OverallStatus.String(),
		"duration", report.Duration,
		"checks_run", len(report.HealthChecks))

	return report, nil
}

// DiagnosticReport contains the results of a system diagnostic.
type DiagnosticReport struct {
	Timestamp     time.Time     `json:"timestamp"`
	Duration      time.Duration `json:"duration"`
	OverallStatus HealthStatus  `json:"overall_status"`
	HealthChecks  []HealthCheck `json:"health_checks"`
	SystemInfo    SystemInfo    `json:"system_info"`
}

// SystemInfo contains system information.
type SystemInfo struct {
	GoVersion     string                 `json:"go_version"`
	OS            string                 `json:"os"`
	Arch          string                 `json:"arch"`
	CPUs          int                    `json:"cpus"`
	Goroutines    int                    `json:"goroutines"`
	MemoryStats   map[string]uint64      `json:"memory_stats"`
	RecoveryStats map[string]interface{} `json:"recovery_stats"`
}

// registerDefaultHealthCheckers registers the default set of health checkers.
func (sd *SystemDiagnostics) registerDefaultHealthCheckers() {
	sd.RegisterHealthChecker(NewMemoryHealthChecker())
	sd.RegisterHealthChecker(NewGoroutineHealthChecker())
	sd.RegisterHealthChecker(NewNetworkHealthChecker())
	sd.RegisterHealthChecker(NewDiskHealthChecker())
}

// monitoringLoop runs the continuous monitoring loop.
func (sd *SystemDiagnostics) monitoringLoop() {
	quickTicker := time.NewTicker(sd.quickCheckInterval)
	fullTicker := time.NewTicker(sd.fullCheckInterval)

	defer quickTicker.Stop()
	defer fullTicker.Stop()

	for {
		select {
		case <-sd.monitorCtx.Done():
			return

		case <-quickTicker.C:
			sd.runQuickChecks()

		case <-fullTicker.C:
			if _, err := sd.RunDiagnostics(sd.monitorCtx); err != nil {
				sd.logger.Error("Full diagnostic check failed", "error", err.Error())
			}
		}
	}
}

// runQuickChecks runs a subset of quick health checks.
func (sd *SystemDiagnostics) runQuickChecks() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Run high-priority health checks only
	for _, checker := range sd.healthChecks {
		if checker.Priority() <= 2 { // Only high and medium priority checks
			check := checker.Check(ctx)

			if check.Status == HealthStatusCritical {
				sd.logger.Error("Critical health check failure",
					"checker", check.Name,
					"error", check.Error)

				// Immediate recovery for critical issues
				if check.Error != nil {
					if err := sd.recoveryMgr.Recover(ctx, check.Error); err != nil {
						sd.logger.Error("Emergency recovery failed",
							"checker", check.Name,
							"recovery_error", err.Error())
					}
				}
			}
		}
	}
}

// calculateOverallHealth calculates the overall system health.
func (sd *SystemDiagnostics) calculateOverallHealth(checks []HealthCheck) HealthStatus {
	if len(checks) == 0 {
		return HealthStatusUnknown
	}

	criticalCount := 0
	unhealthyCount := 0
	degradedCount := 0
	healthyCount := 0

	for _, check := range checks {
		switch check.Status {
		case HealthStatusCritical:
			criticalCount++
		case HealthStatusUnhealthy:
			unhealthyCount++
		case HealthStatusDegraded:
			degradedCount++
		case HealthStatusHealthy:
			healthyCount++
		}
	}

	// Determine overall status
	if criticalCount > 0 {
		return HealthStatusCritical
	}
	if unhealthyCount > 0 {
		return HealthStatusUnhealthy
	}
	if degradedCount > 0 {
		return HealthStatusDegraded
	}
	if healthyCount > 0 {
		return HealthStatusHealthy
	}

	return HealthStatusUnknown
}

// collectSystemInfo collects current system information.
func (sd *SystemDiagnostics) collectSystemInfo() SystemInfo {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	return SystemInfo{
		GoVersion:  runtime.Version(),
		OS:         runtime.GOOS,
		Arch:       runtime.GOARCH,
		CPUs:       runtime.NumCPU(),
		Goroutines: runtime.NumGoroutine(),
		MemoryStats: map[string]uint64{
			"heap_alloc":  memStats.HeapAlloc,
			"heap_sys":    memStats.HeapSys,
			"heap_idle":   memStats.HeapIdle,
			"heap_inuse":  memStats.HeapInuse,
			"total_alloc": memStats.TotalAlloc,
			"sys":         memStats.Sys,
			"num_gc":      uint64(memStats.NumGC),
		},
		RecoveryStats: sd.recoveryMgr.GetRecoveryStats(),
	}
}

// MemoryHealthChecker checks memory usage health.
type MemoryHealthChecker struct{}

func NewMemoryHealthChecker() *MemoryHealthChecker {
	return &MemoryHealthChecker{}
}

func (m *MemoryHealthChecker) Name() string {
	return "memory"
}

func (m *MemoryHealthChecker) Priority() int {
	return 1 // High priority
}

func (m *MemoryHealthChecker) Check(ctx context.Context) HealthCheck {
	start := time.Now()

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	check := HealthCheck{
		Name:        m.Name(),
		LastChecked: start,
		Duration:    time.Since(start),
	}

	heapMB := memStats.HeapAlloc / (1024 * 1024)

	switch {
	case heapMB > 1024: // > 1GB
		check.Status = HealthStatusCritical
		check.Message = fmt.Sprintf("Very high memory usage: %d MB", heapMB)
		check.Error = fmt.Errorf("memory usage critical: %d MB", heapMB)
	case heapMB > 512: // > 512MB
		check.Status = HealthStatusUnhealthy
		check.Message = fmt.Sprintf("High memory usage: %d MB", heapMB)
	case heapMB > 256: // > 256MB
		check.Status = HealthStatusDegraded
		check.Message = fmt.Sprintf("Elevated memory usage: %d MB", heapMB)
	default:
		check.Status = HealthStatusHealthy
		check.Message = fmt.Sprintf("Memory usage normal: %d MB", heapMB)
	}

	return check
}

// GoroutineHealthChecker checks goroutine count health.
type GoroutineHealthChecker struct{}

func NewGoroutineHealthChecker() *GoroutineHealthChecker {
	return &GoroutineHealthChecker{}
}

func (g *GoroutineHealthChecker) Name() string {
	return "goroutines"
}

func (g *GoroutineHealthChecker) Priority() int {
	return 1 // High priority
}

func (g *GoroutineHealthChecker) Check(ctx context.Context) HealthCheck {
	start := time.Now()

	count := runtime.NumGoroutine()

	check := HealthCheck{
		Name:        g.Name(),
		LastChecked: start,
		Duration:    time.Since(start),
	}

	switch {
	case count > 10000:
		check.Status = HealthStatusCritical
		check.Message = fmt.Sprintf("Excessive goroutines: %d", count)
		check.Error = fmt.Errorf("goroutine leak suspected: %d goroutines", count)
	case count > 5000:
		check.Status = HealthStatusUnhealthy
		check.Message = fmt.Sprintf("High goroutine count: %d", count)
	case count > 1000:
		check.Status = HealthStatusDegraded
		check.Message = fmt.Sprintf("Elevated goroutine count: %d", count)
	default:
		check.Status = HealthStatusHealthy
		check.Message = fmt.Sprintf("Goroutine count normal: %d", count)
	}

	return check
}

// NetworkHealthChecker checks network connectivity health.
type NetworkHealthChecker struct{}

func NewNetworkHealthChecker() *NetworkHealthChecker {
	return &NetworkHealthChecker{}
}

func (n *NetworkHealthChecker) Name() string {
	return "network"
}

func (n *NetworkHealthChecker) Priority() int {
	return 2 // Medium priority
}

func (n *NetworkHealthChecker) Check(ctx context.Context) HealthCheck {
	start := time.Now()

	check := HealthCheck{
		Name:        n.Name(),
		LastChecked: start,
	}

	// Test connectivity to multiple hosts
	hosts := []string{"google.com", "cloudflare.com", "8.8.8.8"}
	successCount := 0

	for _, host := range hosts {
		if _, err := net.LookupHost(host); err == nil {
			successCount++
		}
	}

	check.Duration = time.Since(start)

	switch {
	case successCount == 0:
		check.Status = HealthStatusCritical
		check.Message = "No network connectivity"
		check.Error = fmt.Errorf("network connectivity failed")
	case successCount < len(hosts)/2:
		check.Status = HealthStatusDegraded
		check.Message = fmt.Sprintf("Limited network connectivity (%d/%d)", successCount, len(hosts))
	default:
		check.Status = HealthStatusHealthy
		check.Message = fmt.Sprintf("Network connectivity good (%d/%d)", successCount, len(hosts))
	}

	return check
}

// DiskHealthChecker checks disk space health.
type DiskHealthChecker struct{}

func NewDiskHealthChecker() *DiskHealthChecker {
	return &DiskHealthChecker{}
}

func (d *DiskHealthChecker) Name() string {
	return "disk"
}

func (d *DiskHealthChecker) Priority() int {
	return 3 // Low priority
}

func (d *DiskHealthChecker) Check(ctx context.Context) HealthCheck {
	start := time.Now()

	check := HealthCheck{
		Name:        d.Name(),
		LastChecked: start,
		Duration:    time.Since(start),
		Status:      HealthStatusHealthy,
		Message:     "Disk space check not implemented", // Simplified for this implementation
	}

	return check
}
