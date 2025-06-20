package stability

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/a3tai/mcp-pdf-reader/internal/pdf"
)

// StableExtractionService wraps PDF extraction with comprehensive stability features
type StableExtractionService struct {
	extractionService *pdf.ExtractionService
	stabilityManager  *StabilityManager
	logger            *log.Logger
	config            ExtractionConfig
}

// ExtractionConfig configures stable extraction behavior
type ExtractionConfig struct {
	MaxFileSize       int64         `json:"max_file_size"`
	TimeoutSeconds    int           `json:"timeout_seconds"`
	MemoryLimitMB     int           `json:"memory_limit_mb"`
	EnableDebugLogs   bool          `json:"enable_debug_logs"`
	PreventCrashes    bool          `json:"prevent_crashes"`
	GCAfterExtraction bool          `json:"gc_after_extraction"`
	MaxRetries        int           `json:"max_retries"`
	RetryDelay        time.Duration `json:"retry_delay"`
}

// DefaultExtractionConfig returns sensible defaults
func DefaultExtractionConfig() ExtractionConfig {
	return ExtractionConfig{
		MaxFileSize:       100 * 1024 * 1024, // 100MB
		TimeoutSeconds:    120,               // 2 minutes
		MemoryLimitMB:     1024,              // 1GB
		EnableDebugLogs:   false,
		PreventCrashes:    true,
		GCAfterExtraction: true,
		MaxRetries:        2,
		RetryDelay:        time.Second * 5,
	}
}

// NewStableExtractionService creates a new stable extraction service
func NewStableExtractionService(maxFileSize int64, config ...ExtractionConfig) *StableExtractionService {
	var cfg ExtractionConfig
	if len(config) > 0 {
		cfg = config[0]
	} else {
		cfg = DefaultExtractionConfig()
	}

	// Override max file size if provided
	if maxFileSize > 0 {
		cfg.MaxFileSize = maxFileSize
	}

	logger := log.New(os.Stderr, "[StableExtraction] ", log.LstdFlags)

	// Create stability manager
	stabilityConfig := StabilityConfig{
		MemoryThresholdMB: cfg.MemoryLimitMB,
		CheckPeriod:       time.Second * 5,
		TimeoutSeconds:    cfg.TimeoutSeconds,
		MaxPanics:         10,
		EnableGCForcing:   cfg.GCAfterExtraction,
		EnableDebugLogs:   cfg.EnableDebugLogs,
	}

	stabilityManager := NewStabilityManager(stabilityConfig)

	// Set runtime memory limit
	if cfg.MemoryLimitMB > 0 {
		stabilityManager.SetMemoryLimit(cfg.MemoryLimitMB)
	}

	return &StableExtractionService{
		extractionService: pdf.NewExtractionService(cfg.MaxFileSize),
		stabilityManager:  stabilityManager,
		logger:            logger,
		config:            cfg,
	}
}

// ExtractStructured performs structured extraction with stability monitoring
func (ses *StableExtractionService) ExtractStructured(req pdf.PDFExtractRequest) (*pdf.PDFExtractResult, error) {
	return ses.executeWithStability("ExtractStructured", func() (interface{}, error) {
		return ses.extractionService.ExtractStructured(req)
	})
}

// ExtractComplete performs complete extraction with stability monitoring
func (ses *StableExtractionService) ExtractComplete(req pdf.PDFExtractRequest) (*pdf.PDFExtractResult, error) {
	return ses.executeWithStability("ExtractComplete", func() (interface{}, error) {
		return ses.extractionService.ExtractComplete(req)
	})
}

// ExtractSemantic performs semantic extraction with stability monitoring
func (ses *StableExtractionService) ExtractSemantic(req pdf.PDFExtractRequest) (*pdf.PDFExtractResult, error) {
	return ses.executeWithStability("ExtractSemantic", func() (interface{}, error) {
		return ses.extractionService.ExtractSemantic(req)
	})
}

// ExtractTables performs table extraction with stability monitoring
func (ses *StableExtractionService) ExtractTables(req pdf.PDFExtractRequest) (*pdf.PDFExtractResult, error) {
	return ses.executeWithStability("ExtractTables", func() (interface{}, error) {
		return ses.extractionService.ExtractTables(req)
	})
}

// ExtractForms performs form extraction with stability monitoring
func (ses *StableExtractionService) ExtractForms(req pdf.PDFExtractRequest) (*pdf.PDFExtractResult, error) {
	return ses.executeWithStability("ExtractForms", func() (interface{}, error) {
		return ses.extractionService.ExtractForms(req)
	})
}

// QueryContent performs content querying with stability monitoring
func (ses *StableExtractionService) QueryContent(req pdf.PDFQueryRequest) (*pdf.PDFQueryResult, error) {
	result, err := ses.stabilityManager.WithRecovery(context.Background(), "QueryContent", func() (interface{}, error) {
		return ses.extractionService.QueryContent(req)
	})
	if err != nil {
		return nil, err
	}

	if queryResult, ok := result.(*pdf.PDFQueryResult); ok {
		return queryResult, nil
	}

	return nil, fmt.Errorf("unexpected result type from QueryContent")
}

// GetPageInfo gets page information with stability monitoring
func (ses *StableExtractionService) GetPageInfo(path string) (*pdf.PDFPageInfoResult, error) {
	result, err := ses.stabilityManager.WithRecovery(context.Background(), "GetPageInfo", func() (interface{}, error) {
		return ses.extractionService.GetPageInfo(path)
	})
	if err != nil {
		return nil, err
	}

	if pageInfo, ok := result.(*pdf.PDFPageInfoResult); ok {
		return pageInfo, nil
	}

	return nil, fmt.Errorf("unexpected result type from GetPageInfo")
}

// GetMetadata gets document metadata with stability monitoring
func (ses *StableExtractionService) GetMetadata(path string) (*pdf.DocumentMetadata, error) {
	result, err := ses.stabilityManager.WithRecovery(context.Background(), "GetMetadata", func() (interface{}, error) {
		return ses.extractionService.GetMetadata(path)
	})
	if err != nil {
		return nil, err
	}

	if metadata, ok := result.(*pdf.DocumentMetadata); ok {
		return metadata, nil
	}

	return nil, fmt.Errorf("unexpected result type from GetMetadata")
}

// executeWithStability executes an operation with comprehensive stability monitoring
func (ses *StableExtractionService) executeWithStability(operation string, fn func() (interface{}, error)) (*pdf.PDFExtractResult, error) {
	var lastErr error

	for attempt := 0; attempt <= ses.config.MaxRetries; attempt++ {
		if attempt > 0 {
			ses.logger.Printf("Retrying %s (attempt %d/%d)", operation, attempt+1, ses.config.MaxRetries+1)
			time.Sleep(ses.config.RetryDelay)

			// Force GC between retries
			if ses.config.GCAfterExtraction {
				ses.stabilityManager.ForceGC()
			}
		}

		// Execute with stability monitoring
		result, err := ses.stabilityManager.WithRecovery(context.Background(), operation, fn)

		if err == nil {
			// Success - perform cleanup if configured
			if ses.config.GCAfterExtraction {
				ses.stabilityManager.ForceGC()
			}

			// Type assertion for PDFExtractResult
			if extractResult, ok := result.(*pdf.PDFExtractResult); ok {
				return extractResult, nil
			}

			// If not the expected type, continue to handle it
			return nil, fmt.Errorf("unexpected result type from %s", operation)
		}

		lastErr = err

		// Check if this is a recoverable error
		if !ses.isRecoverableError(err) {
			break
		}

		if ses.config.EnableDebugLogs {
			ses.logger.Printf("Attempt %d of %s failed: %v", attempt+1, operation, err)
		}
	}

	// All attempts failed
	return nil, ses.enhanceError(operation, lastErr)
}

// isRecoverableError determines if an error is worth retrying
func (ses *StableExtractionService) isRecoverableError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()

	// Recoverable errors
	recoverableErrors := []string{
		"timeout",
		"memory",
		"panic",
		"temporary",
		"connection",
		"resource temporarily unavailable",
	}

	for _, recoverable := range recoverableErrors {
		if containsString(errStr, recoverable) {
			return true
		}
	}

	// Non-recoverable errors
	nonRecoverableErrors := []string{
		"file not found",
		"permission denied",
		"invalid file",
		"not a PDF",
		"corrupted",
		"malformed",
	}

	for _, nonRecoverable := range nonRecoverableErrors {
		if containsString(errStr, nonRecoverable) {
			return false
		}
	}

	// Default to recoverable for unknown errors
	return true
}

// enhanceError adds context and troubleshooting information to errors
func (ses *StableExtractionService) enhanceError(operation string, err error) error {
	if err == nil {
		return nil
	}

	// Get stability status
	healthStatus := ses.stabilityManager.GetHealthStatus()
	memStats := ses.stabilityManager.memoryMonitor.GetStats()
	panicCount := ses.stabilityManager.panicHandler.GetPanicCount()

	enhanced := fmt.Sprintf("Operation %s failed: %v\n\n", operation, err)
	enhanced += "System Status:\n"
	enhanced += fmt.Sprintf("- Memory Usage: %d MB (max: %d MB)\n",
		memStats.CurrentAlloc/1024/1024, memStats.MaxAlloc/1024/1024)
	enhanced += fmt.Sprintf("- Memory Violations: %d\n", memStats.Violations)
	enhanced += fmt.Sprintf("- Recent Panics: %d\n", panicCount)
	enhanced += fmt.Sprintf("- System Healthy: %v\n", healthStatus["healthy"])

	enhanced += "\nTroubleshooting:\n"
	enhanced += "- Try with a smaller PDF file\n"
	enhanced += "- Increase memory limits if processing large files\n"
	enhanced += "- Check if the PDF file is corrupted or password-protected\n"
	enhanced += "- Use pdf_validate_file to check document integrity\n"

	if memStats.Violations > 0 {
		enhanced += "- Consider reducing concurrent operations due to memory pressure\n"
	}

	if panicCount > 0 {
		enhanced += "- Recent panics detected - the PDF may have malformed content\n"
	}

	return fmt.Errorf("%s", enhanced)
}

// GetHealthStatus returns the current health status of the extraction service
func (ses *StableExtractionService) GetHealthStatus() map[string]interface{} {
	return ses.stabilityManager.GetHealthStatus()
}

// Reset clears all stability statistics
func (ses *StableExtractionService) Reset() {
	ses.stabilityManager.Reset()
}

// containsString checks if a string contains a substring (case-insensitive)
func containsString(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
			len(s) > len(substr) &&
				(s[:len(substr)] == substr ||
					s[len(s)-len(substr):] == substr ||
					indexOf(s, substr) >= 0))
}

// indexOf returns the index of substr in s, or -1 if not found
func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// Close properly closes the stable extraction service and cleans up resources
func (ses *StableExtractionService) Close() error {
	// Force final garbage collection
	if ses.config.GCAfterExtraction {
		ses.stabilityManager.ForceGC()
	}

	return nil
}
