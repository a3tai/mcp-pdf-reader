package errors

import (
	"context"
	"fmt"
	"log"
	"os"
	"runtime/debug"
	"time"

	"github.com/ledongthuc/pdf"
)

// RobustParser wraps PDF parsing with comprehensive error handling and recovery
type RobustParser struct {
	recoveryManager *RecoveryManager
	errorCollection *ErrorCollection
	options         *ParseOptions
	logger          *log.Logger
	context         *ParseContext
}

// ParseResult contains the results of robust parsing
type ParseResult struct {
	Reader           *pdf.Reader      `json:"-"`
	File             *os.File         `json:"-"`
	Success          bool             `json:"success"`
	Errors           *ErrorCollection `json:"errors"`
	RecoveryAttempts int              `json:"recovery_attempts"`
	ProcessingTime   time.Duration    `json:"processing_time"`
	PagesProcessed   int              `json:"pages_processed"`
	TotalPages       int              `json:"total_pages"`
	Warnings         []string         `json:"warnings"`
}

// NewRobustParser creates a new robust parser with default configuration
func NewRobustParser() *RobustParser {
	logger := log.New(os.Stderr, "[RobustParser] ", log.LstdFlags)

	return &RobustParser{
		recoveryManager: NewRecoveryManager(logger),
		options:         DefaultParseOptions(),
		logger:          logger,
	}
}

// NewRobustParserWithOptions creates a new robust parser with custom options
func NewRobustParserWithOptions(options *ParseOptions) *RobustParser {
	logger := log.New(os.Stderr, "[RobustParser] ", log.LstdFlags)

	return &RobustParser{
		recoveryManager: NewRecoveryManager(logger),
		options:         options,
		logger:          logger,
	}
}

// ParseFile parses a PDF file with robust error handling and recovery
func (rp *RobustParser) ParseFile(filePath string) (*ParseResult, error) {
	startTime := time.Now()

	// Initialize error collection
	rp.errorCollection = NewErrorCollection(filePath)

	// Initialize parse context
	rp.context = &ParseContext{
		Options:     rp.options,
		FilePath:    filePath,
		CurrentPage: 0,
		Errors:      rp.errorCollection,
		Logger:      rp.logger,
		StartTime:   startTime,
		ObjectCache: make(map[string]interface{}),
	}

	result := &ParseResult{
		Errors:         rp.errorCollection,
		ProcessingTime: 0,
		Warnings:       make([]string, 0),
	}

	// Set up panic recovery
	defer func() {
		if r := recover(); r != nil {
			panicErr := NewPDFError(ErrorTypeCorruptedData, fmt.Sprintf("Parser panic: %v", r))
			panicErr.StackTrace = string(debug.Stack())
			panicErr.FilePath = filePath
			rp.errorCollection.Add(panicErr)

			if rp.logger != nil {
				rp.logger.Printf("PANIC during PDF parsing: %v", r)
				if rp.options.EnableDebugLogging {
					rp.logger.Printf("Stack trace: %s", panicErr.StackTrace)
				}
			}

			result.Success = false
			result.ProcessingTime = time.Since(startTime)
		}
	}()

	// Set up timeout if configured
	var ctx context.Context
	var cancel context.CancelFunc

	if rp.options.TimeoutSeconds > 0 {
		ctx, cancel = context.WithTimeout(context.Background(), time.Duration(rp.options.TimeoutSeconds)*time.Second)
		defer cancel()
	} else {
		ctx = context.Background()
	}

	// Channel to handle timeout
	resultChan := make(chan *ParseResult, 1)
	errChan := make(chan error, 1)

	// Perform parsing in goroutine to handle timeout
	go func() {
		parseResult, err := rp.parseWithRecovery(filePath)
		if err != nil {
			errChan <- err
		} else {
			resultChan <- parseResult
		}
	}()

	// Wait for result or timeout
	select {
	case <-ctx.Done():
		timeoutErr := NewPDFError(ErrorTypeTimeout, "PDF parsing timed out")
		timeoutErr.FilePath = filePath
		rp.errorCollection.Add(timeoutErr)

		result.Success = false
		result.ProcessingTime = time.Since(startTime)
		return result, fmt.Errorf("parsing timed out after %d seconds", rp.options.TimeoutSeconds)

	case err := <-errChan:
		result.Success = false
		result.ProcessingTime = time.Since(startTime)
		return result, err

	case finalResult := <-resultChan:
		finalResult.ProcessingTime = time.Since(startTime)
		return finalResult, nil
	}
}

// parseWithRecovery performs the actual parsing with recovery mechanisms
func (rp *RobustParser) parseWithRecovery(filePath string) (*ParseResult, error) {
	result := &ParseResult{
		Errors:   rp.errorCollection,
		Warnings: make([]string, 0),
	}

	// Attempt to open and parse the PDF
	file, pdfReader, err := rp.attemptOpen(filePath)
	if err != nil {
		// Try recovery if initial open fails
		if recoveredReader, recoveryErr := rp.attemptFileRecovery(filePath, err); recoveryErr == nil {
			result.Reader = recoveredReader
			result.File = file
			result.Success = true
		} else {
			return result, fmt.Errorf("failed to open PDF and recovery failed: %w", err)
		}
	} else {
		result.Reader = pdfReader
		result.File = file
		result.Success = true
	}

	// Update context with reader
	if file != nil {
		rp.context.Reader = file
	}

	// Get basic document info
	if result.Reader != nil {
		result.TotalPages = result.Reader.NumPage()

		if rp.logger != nil && rp.options.EnableDebugLogging {
			rp.logger.Printf("Successfully opened PDF with %d pages", result.TotalPages)
		}

		// Validate pages with recovery
		result.PagesProcessed = rp.validatePagesWithRecovery(result.Reader, result.TotalPages)
	}

	// Generate warnings for recoverable errors
	for _, err := range rp.errorCollection.GetRecoverableErrors() {
		result.Warnings = append(result.Warnings, fmt.Sprintf("Recovered from: %s", err.Message))
	}

	return result, nil
}

// attemptOpen tries to open the PDF file with the standard library
func (rp *RobustParser) attemptOpen(filePath string) (*os.File, *pdf.Reader, error) {
	if rp.logger != nil && rp.options.EnableDebugLogging {
		rp.logger.Printf("Attempting to open PDF: %s", filePath)
	}

	file, pdfReader, err := pdf.Open(filePath)
	if err != nil {
		pdfErr := WrapError(ErrorTypeInvalidHeader, err)
		pdfErr.FilePath = filePath
		rp.errorCollection.Add(pdfErr)

		if rp.logger != nil {
			rp.logger.Printf("Initial PDF open failed: %v", err)
		}

		return nil, nil, err
	}

	return file, pdfReader, nil
}

// attemptFileRecovery tries to recover from file-level errors
func (rp *RobustParser) attemptFileRecovery(filePath string, originalErr error) (*pdf.Reader, error) {
	if !rp.options.EnableFallbacks {
		return nil, fmt.Errorf("fallbacks disabled")
	}

	if rp.logger != nil {
		rp.logger.Printf("Attempting file recovery for: %s", filePath)
	}

	// Try different recovery strategies based on the error
	pdfErr := WrapError(ErrorTypeCorruptedData, originalErr)
	pdfErr.FilePath = filePath

	// Attempt recovery through the recovery manager
	recovered, err := rp.recoveryManager.AttemptRecovery(pdfErr, rp.context)
	if err != nil {
		return nil, fmt.Errorf("file recovery failed: %w", err)
	}

	// Try to cast recovered result to pdf.Reader
	if reader, ok := recovered.(*pdf.Reader); ok {
		return reader, nil
	}

	return nil, fmt.Errorf("recovery did not produce a valid PDF reader")
}

// validatePagesWithRecovery validates all pages and attempts recovery for corrupted pages
func (rp *RobustParser) validatePagesWithRecovery(reader *pdf.Reader, totalPages int) int {
	validPages := 0

	for pageNum := 1; pageNum <= totalPages; pageNum++ {
		rp.context.CurrentPage = pageNum

		if rp.validatePageWithRecovery(reader, pageNum) {
			validPages++
		}
	}

	if rp.logger != nil && rp.options.EnableDebugLogging {
		rp.logger.Printf("Validated %d/%d pages successfully", validPages, totalPages)
	}

	return validPages
}

// validatePageWithRecovery validates a single page and attempts recovery if needed
func (rp *RobustParser) validatePageWithRecovery(reader *pdf.Reader, pageNum int) bool {
	defer func() {
		if r := recover(); r != nil {
			pageErr := NewPDFError(ErrorTypeMalformedPage, fmt.Sprintf("Page %d validation panic: %v", pageNum, r))
			pageErr.PageNumber = pageNum
			pageErr.FilePath = rp.context.FilePath
			rp.errorCollection.Add(pageErr)

			if rp.logger != nil {
				rp.logger.Printf("Page %d validation panic: %v", pageNum, r)
			}

			// Attempt page recovery
			if rp.options.SkipCorruptedPages {
				if rp.logger != nil {
					rp.logger.Printf("Skipping corrupted page %d", pageNum)
				}
			}
		}
	}()

	// Attempt to access the page
	page := reader.Page(pageNum)
	if page.V.IsNull() {
		pageErr := NewPDFError(ErrorTypeMalformedPage, fmt.Sprintf("Page %d is null or invalid", pageNum))
		pageErr.PageNumber = pageNum
		pageErr.FilePath = rp.context.FilePath
		rp.errorCollection.Add(pageErr)

		// Attempt recovery
		if rp.options.SkipCorruptedPages {
			if rp.logger != nil {
				rp.logger.Printf("Skipping invalid page %d", pageNum)
			}
			return false
		}
	}

	// Try to extract basic information from the page
	_, err := rp.validatePageContent(page, pageNum)
	if err != nil {
		pageErr := WrapError(ErrorTypeMalformedPage, err)
		pageErr.PageNumber = pageNum
		pageErr.FilePath = rp.context.FilePath
		rp.errorCollection.Add(pageErr)

		// Attempt recovery through recovery manager
		if rp.options.RecoveryStrategies[ErrorTypeMalformedPage] {
			_, recoveryErr := rp.recoveryManager.AttemptRecovery(pageErr, rp.context)
			if recoveryErr == nil {
				rp.context.Errors.Add(NewPDFErrorWithContext(ErrorTypeMalformedPage,
					fmt.Sprintf("Recovered page %d", pageNum), "page_recovery_successful"))
				return true
			}
		}

		if rp.options.SkipCorruptedPages {
			return false
		}
	}

	return true
}

// validatePageContent validates the content of a page
func (rp *RobustParser) validatePageContent(page pdf.Page, pageNum int) (bool, error) {
	// Check MediaBox
	mediaBox := page.V.Key("MediaBox")
	if mediaBox.IsNull() {
		return false, fmt.Errorf("page %d missing MediaBox", pageNum)
	}

	// Try to extract text (this will reveal many content issues)
	_, err := page.GetPlainText(nil)
	if err != nil {
		return false, fmt.Errorf("page %d text extraction failed: %w", pageNum, err)
	}

	return true, nil
}

// Close properly closes resources used by the robust parser
func (rp *RobustParser) Close() error {
	// Clean up any resources
	if rp.context != nil && rp.context.ObjectCache != nil {
		rp.context.ObjectCache = nil
	}
	return nil
}

// GetErrorSummary returns a summary of all errors encountered during parsing
func (rp *RobustParser) GetErrorSummary() string {
	if rp.errorCollection == nil {
		return "No errors recorded"
	}
	return rp.errorCollection.Summary()
}

// SetLogger sets a custom logger for the robust parser
func (rp *RobustParser) SetLogger(logger *log.Logger) {
	rp.logger = logger
	if rp.recoveryManager != nil {
		rp.recoveryManager.logger = logger
	}
}

// EnableDebugLogging enables or disables debug logging
func (rp *RobustParser) EnableDebugLogging(enabled bool) {
	if rp.options == nil {
		rp.options = DefaultParseOptions()
	}
	rp.options.EnableDebugLogging = enabled
}
