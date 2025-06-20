package errors

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"strings"
	"time"
)

// RecoveryStrategy defines how to recover from specific PDF parsing errors
type RecoveryStrategy interface {
	Recover(err *PDFError, context *ParseContext) (interface{}, error)
	CanRecover(err *PDFError) bool
	GetName() string
	GetDescription() string
}

// ParseContext provides context information for recovery operations
type ParseContext struct {
	Reader      io.ReadSeeker          `json:"-"`
	ObjectCache map[string]interface{} `json:"-"`
	Options     *ParseOptions          `json:"options"`
	FilePath    string                 `json:"file_path"`
	CurrentPage int                    `json:"current_page"`
	Errors      *ErrorCollection       `json:"-"`
	Logger      *log.Logger            `json:"-"`
	StartTime   time.Time              `json:"start_time"`
}

// ParseOptions configures parsing behavior and recovery strategies
type ParseOptions struct {
	StrictMode          bool               `json:"strict_mode"`
	MaxRecoveryAttempts int                `json:"max_recovery_attempts"`
	EnableFallbacks     bool               `json:"enable_fallbacks"`
	SkipCorruptedPages  bool               `json:"skip_corrupted_pages"`
	RepairXRef          bool               `json:"repair_xref"`
	IgnoreFilters       []string           `json:"ignore_filters"`
	FallbackFilters     []string           `json:"fallback_filters"`
	MaxMemoryUsage      int64              `json:"max_memory_usage"`
	TimeoutSeconds      int                `json:"timeout_seconds"`
	EnableDebugLogging  bool               `json:"enable_debug_logging"`
	RecoveryStrategies  map[ErrorType]bool `json:"recovery_strategies"`
}

// DefaultParseOptions returns sensible default parsing options
func DefaultParseOptions() *ParseOptions {
	return &ParseOptions{
		StrictMode:          false,
		MaxRecoveryAttempts: 3,
		EnableFallbacks:     true,
		SkipCorruptedPages:  true,
		RepairXRef:          true,
		IgnoreFilters:       []string{},
		FallbackFilters:     []string{"FlateDecode", "ASCIIHexDecode", "ASCII85Decode"},
		MaxMemoryUsage:      500 * 1024 * 1024, // 500MB
		TimeoutSeconds:      30,
		EnableDebugLogging:  false,
		RecoveryStrategies: map[ErrorType]bool{
			ErrorTypeMalformedObject:   true,
			ErrorTypeInvalidStream:     true,
			ErrorTypeMissingObject:     true,
			ErrorTypeInvalidFilter:     true,
			ErrorTypeResourceNotFound:  true,
			ErrorTypeInvalidFont:       true,
			ErrorTypeInvalidImage:      true,
			ErrorTypeMalformedPage:     true,
			ErrorTypeInvalidAnnotation: true,
		},
	}
}

// RecoveryManager coordinates recovery strategies for different error types
type RecoveryManager struct {
	strategies map[ErrorType][]RecoveryStrategy
	logger     *log.Logger
}

// NewRecoveryManager creates a new recovery manager with default strategies
func NewRecoveryManager(logger *log.Logger) *RecoveryManager {
	rm := &RecoveryManager{
		strategies: make(map[ErrorType][]RecoveryStrategy),
		logger:     logger,
	}

	// Register default recovery strategies
	rm.registerDefaultStrategies()
	return rm
}

// registerDefaultStrategies registers built-in recovery strategies
func (rm *RecoveryManager) registerDefaultStrategies() {
	// Stream recovery strategies
	rm.RegisterStrategy(ErrorTypeInvalidStream, &StreamRecoveryStrategy{})
	rm.RegisterStrategy(ErrorTypeInvalidFilter, &FilterRecoveryStrategy{})

	// Object recovery strategies
	rm.RegisterStrategy(ErrorTypeMalformedObject, &ObjectRecoveryStrategy{})
	rm.RegisterStrategy(ErrorTypeMissingObject, &MissingObjectStrategy{})

	// XRef recovery strategies
	rm.RegisterStrategy(ErrorTypeCorruptedXRef, &XRefRecoveryStrategy{})

	// Resource recovery strategies
	rm.RegisterStrategy(ErrorTypeResourceNotFound, &ResourceRecoveryStrategy{})
	rm.RegisterStrategy(ErrorTypeInvalidFont, &FontRecoveryStrategy{})
	rm.RegisterStrategy(ErrorTypeInvalidImage, &ImageRecoveryStrategy{})

	// Page recovery strategies
	rm.RegisterStrategy(ErrorTypeMalformedPage, &PageRecoveryStrategy{})
}

// RegisterStrategy registers a recovery strategy for a specific error type
func (rm *RecoveryManager) RegisterStrategy(errorType ErrorType, strategy RecoveryStrategy) {
	if rm.strategies[errorType] == nil {
		rm.strategies[errorType] = make([]RecoveryStrategy, 0)
	}
	rm.strategies[errorType] = append(rm.strategies[errorType], strategy)
}

// AttemptRecovery tries to recover from an error using available strategies
func (rm *RecoveryManager) AttemptRecovery(err *PDFError, context *ParseContext) (interface{}, error) {
	strategies, exists := rm.strategies[err.Type]
	if !exists || len(strategies) == 0 {
		return nil, fmt.Errorf("no recovery strategies available for error type: %s", err.Type.String())
	}

	for i, strategy := range strategies {
		if !strategy.CanRecover(err) {
			continue
		}

		if rm.logger != nil && context.Options.EnableDebugLogging {
			rm.logger.Printf("Attempting recovery with strategy %d/%d: %s", i+1, len(strategies), strategy.GetName())
		}

		result, recoveryErr := strategy.Recover(err, context)
		if recoveryErr == nil {
			if rm.logger != nil && context.Options.EnableDebugLogging {
				rm.logger.Printf("Recovery successful with strategy: %s", strategy.GetName())
			}
			return result, nil
		}

		if rm.logger != nil && context.Options.EnableDebugLogging {
			rm.logger.Printf("Recovery failed with strategy %s: %v", strategy.GetName(), recoveryErr)
		}
	}

	return nil, fmt.Errorf("all recovery strategies failed for error: %s", err.Message)
}

// StreamRecoveryStrategy handles corrupted or invalid streams
type StreamRecoveryStrategy struct{}

func (s *StreamRecoveryStrategy) GetName() string        { return "StreamRecovery" }
func (s *StreamRecoveryStrategy) GetDescription() string { return "Recovers corrupted PDF streams" }

func (s *StreamRecoveryStrategy) CanRecover(err *PDFError) bool {
	return err.Type == ErrorTypeInvalidStream || err.Type == ErrorTypeCorruptedData
}

func (s *StreamRecoveryStrategy) Recover(err *PDFError, context *ParseContext) (interface{}, error) {
	// Try to extract readable portions of the stream
	// This is a simplified implementation - in practice, you'd need more sophisticated logic
	return nil, fmt.Errorf("stream recovery not yet implemented")
}

// FilterRecoveryStrategy handles invalid or unsupported filters
type FilterRecoveryStrategy struct{}

func (f *FilterRecoveryStrategy) GetName() string        { return "FilterRecovery" }
func (f *FilterRecoveryStrategy) GetDescription() string { return "Handles unsupported filters" }

func (f *FilterRecoveryStrategy) CanRecover(err *PDFError) bool {
	return err.Type == ErrorTypeInvalidFilter
}

func (f *FilterRecoveryStrategy) Recover(err *PDFError, context *ParseContext) (interface{}, error) {
	// Try fallback filters or skip filter entirely
	return nil, fmt.Errorf("filter recovery not yet implemented")
}

// ObjectRecoveryStrategy handles malformed PDF objects
type ObjectRecoveryStrategy struct{}

func (o *ObjectRecoveryStrategy) GetName() string        { return "ObjectRecovery" }
func (o *ObjectRecoveryStrategy) GetDescription() string { return "Recovers malformed PDF objects" }

func (o *ObjectRecoveryStrategy) CanRecover(err *PDFError) bool {
	return err.Type == ErrorTypeMalformedObject
}

func (o *ObjectRecoveryStrategy) Recover(err *PDFError, context *ParseContext) (interface{}, error) {
	// Try to parse object with relaxed rules or use default values
	return nil, fmt.Errorf("object recovery not yet implemented")
}

// MissingObjectStrategy handles missing object references
type MissingObjectStrategy struct{}

func (m *MissingObjectStrategy) GetName() string        { return "MissingObjectRecovery" }
func (m *MissingObjectStrategy) GetDescription() string { return "Handles missing object references" }

func (m *MissingObjectStrategy) CanRecover(err *PDFError) bool {
	return err.Type == ErrorTypeMissingObject
}

func (m *MissingObjectStrategy) Recover(err *PDFError, context *ParseContext) (interface{}, error) {
	// Provide default object or skip reference
	return nil, fmt.Errorf("missing object recovery not yet implemented")
}

// XRefRecoveryStrategy rebuilds corrupted cross-reference tables
type XRefRecoveryStrategy struct{}

func (x *XRefRecoveryStrategy) GetName() string        { return "XRefRecovery" }
func (x *XRefRecoveryStrategy) GetDescription() string { return "Rebuilds corrupted xref tables" }

func (x *XRefRecoveryStrategy) CanRecover(err *PDFError) bool {
	return err.Type == ErrorTypeCorruptedXRef
}

func (x *XRefRecoveryStrategy) Recover(err *PDFError, context *ParseContext) (interface{}, error) {
	if !context.Options.RepairXRef {
		return nil, fmt.Errorf("xref repair disabled")
	}

	// Scan the entire file for obj/endobj pairs to rebuild xref
	return x.rebuildXRef(context.Reader)
}

func (x *XRefRecoveryStrategy) rebuildXRef(reader io.ReadSeeker) (interface{}, error) {
	// Seek to beginning
	_, err := reader.Seek(0, io.SeekStart)
	if err != nil {
		return nil, fmt.Errorf("failed to seek to beginning: %w", err)
	}

	scanner := bufio.NewScanner(reader)
	objects := make(map[int]int64) // object number -> offset
	var currentOffset int64

	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, " 0 obj") {
			// Parse object number
			parts := strings.Fields(line)
			if len(parts) >= 3 && parts[1] == "0" && parts[2] == "obj" {
				var objNum int
				if n, err := fmt.Sscanf(parts[0], "%d", &objNum); n == 1 && err == nil {
					objects[objNum] = currentOffset
				}
			}
		}
		currentOffset += int64(len(line)) + 1 // +1 for newline
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error scanning file: %w", err)
	}

	return objects, nil
}

// ResourceRecoveryStrategy handles missing resources
type ResourceRecoveryStrategy struct{}

func (r *ResourceRecoveryStrategy) GetName() string        { return "ResourceRecovery" }
func (r *ResourceRecoveryStrategy) GetDescription() string { return "Handles missing resources" }

func (r *ResourceRecoveryStrategy) CanRecover(err *PDFError) bool {
	return err.Type == ErrorTypeResourceNotFound
}

func (r *ResourceRecoveryStrategy) Recover(err *PDFError, context *ParseContext) (interface{}, error) {
	// Provide default resource or skip
	return nil, fmt.Errorf("resource recovery not yet implemented")
}

// FontRecoveryStrategy handles invalid or missing fonts
type FontRecoveryStrategy struct{}

func (f *FontRecoveryStrategy) GetName() string        { return "FontRecovery" }
func (f *FontRecoveryStrategy) GetDescription() string { return "Handles invalid fonts" }

func (f *FontRecoveryStrategy) CanRecover(err *PDFError) bool {
	return err.Type == ErrorTypeInvalidFont
}

func (f *FontRecoveryStrategy) Recover(err *PDFError, context *ParseContext) (interface{}, error) {
	// Use fallback font
	return map[string]interface{}{
		"Type":     "/Font",
		"Subtype":  "/Type1",
		"BaseFont": "/Helvetica",
	}, nil
}

// ImageRecoveryStrategy handles invalid or corrupted images
type ImageRecoveryStrategy struct{}

func (i *ImageRecoveryStrategy) GetName() string        { return "ImageRecovery" }
func (i *ImageRecoveryStrategy) GetDescription() string { return "Handles corrupted images" }

func (i *ImageRecoveryStrategy) CanRecover(err *PDFError) bool {
	return err.Type == ErrorTypeInvalidImage
}

func (i *ImageRecoveryStrategy) Recover(err *PDFError, context *ParseContext) (interface{}, error) {
	// Skip corrupted image or provide placeholder
	return nil, fmt.Errorf("image recovery not yet implemented")
}

// PageRecoveryStrategy handles malformed pages
type PageRecoveryStrategy struct{}

func (p *PageRecoveryStrategy) GetName() string        { return "PageRecovery" }
func (p *PageRecoveryStrategy) GetDescription() string { return "Handles malformed pages" }

func (p *PageRecoveryStrategy) CanRecover(err *PDFError) bool {
	return err.Type == ErrorTypeMalformedPage
}

func (p *PageRecoveryStrategy) Recover(err *PDFError, context *ParseContext) (interface{}, error) {
	if context.Options.SkipCorruptedPages {
		// Return minimal page structure
		return map[string]interface{}{
			"Type":     "/Page",
			"MediaBox": []float64{0, 0, 612, 792}, // Default US Letter
		}, nil
	}
	return nil, fmt.Errorf("page recovery disabled")
}
