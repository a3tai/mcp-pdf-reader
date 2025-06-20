package stability

import (
	"context"
	"fmt"
	"log"
	"os"
	"runtime"
	"runtime/debug"
	"sync"
	"time"
)

// MemoryMonitor provides memory usage monitoring and enforcement
type MemoryMonitor struct {
	threshold   uint64        // Memory threshold in bytes
	checkPeriod time.Duration // How often to check memory
	logger      *log.Logger
	mu          sync.RWMutex
	isActive    bool
	stats       MemoryStats
}

// MemoryStats tracks memory usage statistics
type MemoryStats struct {
	MaxAlloc     uint64    `json:"max_alloc"`
	CurrentAlloc uint64    `json:"current_alloc"`
	CheckCount   int64     `json:"check_count"`
	GCCount      int64     `json:"gc_count"`
	LastCheck    time.Time `json:"last_check"`
	Violations   int64     `json:"violations"`
}

// PanicRecoveryHandler handles panics and provides detailed logging
type PanicRecoveryHandler struct {
	logger    *log.Logger
	maxPanics int
	panics    []PanicRecord
	mu        sync.RWMutex
}

// PanicRecord stores information about a panic
type PanicRecord struct {
	Timestamp  time.Time `json:"timestamp"`
	Message    string    `json:"message"`
	StackTrace string    `json:"stack_trace"`
	Context    string    `json:"context"`
	Recovered  bool      `json:"recovered"`
}

// StabilityManager coordinates all stability features
type StabilityManager struct {
	memoryMonitor *MemoryMonitor
	panicHandler  *PanicRecoveryHandler
	timeout       time.Duration
	logger        *log.Logger
	config        StabilityConfig
}

// StabilityConfig configures stability monitoring
type StabilityConfig struct {
	MemoryThresholdMB int           `json:"memory_threshold_mb"`
	CheckPeriod       time.Duration `json:"check_period"`
	TimeoutSeconds    int           `json:"timeout_seconds"`
	MaxPanics         int           `json:"max_panics"`
	EnableGCForcing   bool          `json:"enable_gc_forcing"`
	EnableDebugLogs   bool          `json:"enable_debug_logs"`
}

// DefaultStabilityConfig returns sensible default configuration
func DefaultStabilityConfig() StabilityConfig {
	return StabilityConfig{
		MemoryThresholdMB: 1024, // 1GB
		CheckPeriod:       time.Second * 5,
		TimeoutSeconds:    120, // 2 minutes
		MaxPanics:         10,
		EnableGCForcing:   true,
		EnableDebugLogs:   false,
	}
}

// NewStabilityManager creates a new stability manager
func NewStabilityManager(config StabilityConfig) *StabilityManager {
	logger := log.New(os.Stderr, "[Stability] ", log.LstdFlags)

	memMonitor := &MemoryMonitor{
		threshold:   uint64(config.MemoryThresholdMB) * 1024 * 1024,
		checkPeriod: config.CheckPeriod,
		logger:      logger,
		stats:       MemoryStats{},
	}

	panicHandler := &PanicRecoveryHandler{
		logger:    logger,
		maxPanics: config.MaxPanics,
		panics:    make([]PanicRecord, 0),
	}

	return &StabilityManager{
		memoryMonitor: memMonitor,
		panicHandler:  panicHandler,
		timeout:       time.Duration(config.TimeoutSeconds) * time.Second,
		logger:        logger,
		config:        config,
	}
}

// WithRecovery wraps a function with comprehensive error recovery
func (sm *StabilityManager) WithRecovery(ctx context.Context, operation string, fn func() (interface{}, error)) (result interface{}, err error) {
	// Set up panic recovery
	defer func() {
		if r := recover(); r != nil {
			panicMsg := fmt.Sprintf("Panic in %s: %v", operation, r)
			stackTrace := string(debug.Stack())

			// Record the panic
			sm.panicHandler.RecordPanic(panicMsg, stackTrace, operation, true)

			// Return as error
			err = fmt.Errorf("operation panicked: %v", r)

			if sm.config.EnableDebugLogs {
				sm.logger.Printf("PANIC RECOVERED in %s: %v", operation, r)
				sm.logger.Printf("Stack trace: %s", stackTrace)
			}
		}
	}()

	// Create timeout context
	timeoutCtx, cancel := context.WithTimeout(ctx, sm.timeout)
	defer cancel()

	// Start memory monitoring
	stopMonitor := sm.memoryMonitor.Start(timeoutCtx)
	defer stopMonitor()

	// Execute operation in goroutine
	type operationResult struct {
		data interface{}
		err  error
	}

	resultChan := make(chan operationResult, 1)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				resultChan <- operationResult{
					err: fmt.Errorf("operation goroutine panic: %v", r),
				}
			}
		}()

		data, err := fn()
		resultChan <- operationResult{data: data, err: err}
	}()

	// Wait for result or timeout
	select {
	case res := <-resultChan:
		return res.data, res.err
	case <-timeoutCtx.Done():
		return nil, fmt.Errorf("operation '%s' timed out after %v", operation, sm.timeout)
	}
}

// Start begins memory monitoring
func (mm *MemoryMonitor) Start(ctx context.Context) func() {
	mm.mu.Lock()
	if mm.isActive {
		mm.mu.Unlock()
		return func() {} // Already active
	}
	mm.isActive = true
	mm.mu.Unlock()

	done := make(chan struct{})

	go func() {
		defer func() {
			mm.mu.Lock()
			mm.isActive = false
			mm.mu.Unlock()
		}()

		ticker := time.NewTicker(mm.checkPeriod)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				mm.checkMemory()
			case <-ctx.Done():
				return
			case <-done:
				return
			}
		}
	}()

	return func() { close(done) }
}

// checkMemory performs a memory check and enforcement
func (mm *MemoryMonitor) checkMemory() {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	mm.mu.Lock()
	mm.stats.CurrentAlloc = memStats.Alloc
	mm.stats.CheckCount++
	mm.stats.LastCheck = time.Now()

	if memStats.Alloc > mm.stats.MaxAlloc {
		mm.stats.MaxAlloc = memStats.Alloc
	}
	mm.mu.Unlock()

	if memStats.Alloc > mm.threshold {
		mm.handleMemoryViolation(memStats)
	}
}

// handleMemoryViolation handles memory threshold violations
func (mm *MemoryMonitor) handleMemoryViolation(memStats runtime.MemStats) {
	mm.mu.Lock()
	mm.stats.Violations++
	mm.mu.Unlock()

	mm.logger.Printf("Memory threshold exceeded: %d MB > %d MB",
		memStats.Alloc/1024/1024, mm.threshold/1024/1024)

	// Force garbage collection
	runtime.GC()
	mm.mu.Lock()
	mm.stats.GCCount++
	mm.mu.Unlock()

	// Check again after GC
	runtime.ReadMemStats(&memStats)

	if memStats.Alloc > mm.threshold {
		mm.logger.Printf("Memory still high after GC: %d MB", memStats.Alloc/1024/1024)
		panic(fmt.Sprintf("memory threshold exceeded: %d > %d bytes", memStats.Alloc, mm.threshold))
	} else {
		mm.logger.Printf("Memory reduced after GC: %d MB", memStats.Alloc/1024/1024)
	}
}

// GetStats returns current memory statistics
func (mm *MemoryMonitor) GetStats() MemoryStats {
	mm.mu.RLock()
	defer mm.mu.RUnlock()
	return mm.stats
}

// RecordPanic records a panic occurrence
func (prh *PanicRecoveryHandler) RecordPanic(message, stackTrace, context string, recovered bool) {
	prh.mu.Lock()
	defer prh.mu.Unlock()

	record := PanicRecord{
		Timestamp:  time.Now(),
		Message:    message,
		StackTrace: stackTrace,
		Context:    context,
		Recovered:  recovered,
	}

	prh.panics = append(prh.panics, record)

	// Keep only the most recent panics
	if len(prh.panics) > prh.maxPanics {
		prh.panics = prh.panics[len(prh.panics)-prh.maxPanics:]
	}

	prh.logger.Printf("PANIC RECORDED: %s", message)
}

// GetPanics returns recent panic records
func (prh *PanicRecoveryHandler) GetPanics() []PanicRecord {
	prh.mu.RLock()
	defer prh.mu.RUnlock()

	// Return a copy to avoid race conditions
	result := make([]PanicRecord, len(prh.panics))
	copy(result, prh.panics)
	return result
}

// GetPanicCount returns the number of recorded panics
func (prh *PanicRecoveryHandler) GetPanicCount() int {
	prh.mu.RLock()
	defer prh.mu.RUnlock()
	return len(prh.panics)
}

// LogPanic logs panic details
func (prh *PanicRecoveryHandler) LogPanic(panicValue interface{}, stackTrace []byte) {
	prh.logger.Printf("PANIC: %v", panicValue)
	if len(stackTrace) > 0 {
		prh.logger.Printf("Stack trace: %s", string(stackTrace))
	}
}

// GetHealthStatus returns overall stability health status
func (sm *StabilityManager) GetHealthStatus() map[string]interface{} {
	memStats := sm.memoryMonitor.GetStats()
	panicCount := sm.panicHandler.GetPanicCount()

	return map[string]interface{}{
		"memory_stats": memStats,
		"panic_count":  panicCount,
		"timeout":      sm.timeout.String(),
		"healthy":      panicCount < sm.config.MaxPanics && memStats.Violations < 10,
		"uptime":       time.Since(memStats.LastCheck),
	}
}

// ForceGC forces garbage collection if enabled
func (sm *StabilityManager) ForceGC() {
	if sm.config.EnableGCForcing {
		sm.logger.Printf("Forcing garbage collection")
		runtime.GC()
		debug.FreeOSMemory()
	}
}

// SetMemoryLimit sets the Go runtime memory limit
func (sm *StabilityManager) SetMemoryLimit(limitMB int) {
	if limitMB > 0 {
		limit := int64(limitMB) * 1024 * 1024
		debug.SetMemoryLimit(limit)
		sm.logger.Printf("Set memory limit to %d MB", limitMB)
	}
}

// Reset clears all recorded statistics and panics
func (sm *StabilityManager) Reset() {
	sm.memoryMonitor.mu.Lock()
	sm.memoryMonitor.stats = MemoryStats{}
	sm.memoryMonitor.mu.Unlock()

	sm.panicHandler.mu.Lock()
	sm.panicHandler.panics = make([]PanicRecord, 0)
	sm.panicHandler.mu.Unlock()

	sm.logger.Printf("Stability manager reset")
}
