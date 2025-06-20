package errors

import (
	"fmt"
	"runtime/debug"
	"time"
)

// PDFError represents a comprehensive PDF parsing error with context and recovery information
type PDFError struct {
	Type        ErrorType `json:"type"`
	Message     string    `json:"message"`
	Context     string    `json:"context,omitempty"`
	Offset      int64     `json:"offset,omitempty"`
	ObjectNum   int       `json:"object_num,omitempty"`
	GenNum      int       `json:"generation_num,omitempty"`
	Recoverable bool      `json:"recoverable"`
	StackTrace  string    `json:"stack_trace,omitempty"`
	Timestamp   time.Time `json:"timestamp"`
	FilePath    string    `json:"file_path,omitempty"`
	PageNumber  int       `json:"page_number,omitempty"`
}

// ErrorType represents different categories of PDF parsing errors
type ErrorType int

const (
	ErrorTypeUnknown ErrorType = iota
	ErrorTypeInvalidHeader
	ErrorTypeCorruptedXRef
	ErrorTypeMalformedObject
	ErrorTypeInvalidStream
	ErrorTypeMissingObject
	ErrorTypeCircularReference
	ErrorTypeInvalidEncoding
	ErrorTypeCorruptedData
	ErrorTypeUnsupportedFeature
	ErrorTypeSecurityRestriction
	ErrorTypeInvalidFilter
	ErrorTypeInvalidFont
	ErrorTypeInvalidImage
	ErrorTypeInvalidForm
	ErrorTypeMalformedPage
	ErrorTypeInvalidAnnotation
	ErrorTypeInvalidMetadata
	ErrorTypeInvalidStructure
	ErrorTypeResourceNotFound
	ErrorTypeMemoryExhausted
	ErrorTypeTimeout
)

// ErrorSeverity indicates how critical an error is
type ErrorSeverity int

const (
	SeverityInfo ErrorSeverity = iota
	SeverityWarning
	SeverityError
	SeverityCritical
	SeverityFatal
)

// Error implements the error interface
func (e *PDFError) Error() string {
	if e.Context != "" {
		return fmt.Sprintf("[%s] %s: %s", e.Type.String(), e.Message, e.Context)
	}
	return fmt.Sprintf("[%s] %s", e.Type.String(), e.Message)
}

// String returns a string representation of the ErrorType
func (et ErrorType) String() string {
	switch et {
	case ErrorTypeInvalidHeader:
		return "INVALID_HEADER"
	case ErrorTypeCorruptedXRef:
		return "CORRUPTED_XREF"
	case ErrorTypeMalformedObject:
		return "MALFORMED_OBJECT"
	case ErrorTypeInvalidStream:
		return "INVALID_STREAM"
	case ErrorTypeMissingObject:
		return "MISSING_OBJECT"
	case ErrorTypeCircularReference:
		return "CIRCULAR_REFERENCE"
	case ErrorTypeInvalidEncoding:
		return "INVALID_ENCODING"
	case ErrorTypeCorruptedData:
		return "CORRUPTED_DATA"
	case ErrorTypeUnsupportedFeature:
		return "UNSUPPORTED_FEATURE"
	case ErrorTypeSecurityRestriction:
		return "SECURITY_RESTRICTION"
	case ErrorTypeInvalidFilter:
		return "INVALID_FILTER"
	case ErrorTypeInvalidFont:
		return "INVALID_FONT"
	case ErrorTypeInvalidImage:
		return "INVALID_IMAGE"
	case ErrorTypeInvalidForm:
		return "INVALID_FORM"
	case ErrorTypeMalformedPage:
		return "MALFORMED_PAGE"
	case ErrorTypeInvalidAnnotation:
		return "INVALID_ANNOTATION"
	case ErrorTypeInvalidMetadata:
		return "INVALID_METADATA"
	case ErrorTypeInvalidStructure:
		return "INVALID_STRUCTURE"
	case ErrorTypeResourceNotFound:
		return "RESOURCE_NOT_FOUND"
	case ErrorTypeMemoryExhausted:
		return "MEMORY_EXHAUSTED"
	case ErrorTypeTimeout:
		return "TIMEOUT"
	default:
		return "UNKNOWN"
	}
}

// GetSeverity returns the severity level for a given error type
func (et ErrorType) GetSeverity() ErrorSeverity {
	switch et {
	case ErrorTypeInvalidHeader, ErrorTypeCorruptedXRef, ErrorTypeCorruptedData:
		return SeverityCritical
	case ErrorTypeMalformedObject, ErrorTypeInvalidStream, ErrorTypeMissingObject:
		return SeverityError
	case ErrorTypeCircularReference, ErrorTypeInvalidEncoding, ErrorTypeInvalidFilter:
		return SeverityError
	case ErrorTypeUnsupportedFeature, ErrorTypeSecurityRestriction:
		return SeverityWarning
	case ErrorTypeInvalidFont, ErrorTypeInvalidImage, ErrorTypeInvalidForm:
		return SeverityWarning
	case ErrorTypeMalformedPage, ErrorTypeInvalidAnnotation, ErrorTypeInvalidMetadata:
		return SeverityWarning
	case ErrorTypeResourceNotFound:
		return SeverityWarning
	case ErrorTypeMemoryExhausted, ErrorTypeTimeout:
		return SeverityFatal
	default:
		return SeverityError
	}
}

// IsRecoverable determines if an error type is generally recoverable
func (et ErrorType) IsRecoverable() bool {
	switch et {
	case ErrorTypeInvalidHeader, ErrorTypeCorruptedXRef:
		return false // Usually fatal
	case ErrorTypeCorruptedData, ErrorTypeMemoryExhausted, ErrorTypeTimeout:
		return false // Usually fatal
	case ErrorTypeMalformedObject, ErrorTypeInvalidStream, ErrorTypeMissingObject:
		return true // Often recoverable with fallbacks
	case ErrorTypeCircularReference, ErrorTypeInvalidEncoding, ErrorTypeInvalidFilter:
		return true // Can often be worked around
	case ErrorTypeUnsupportedFeature, ErrorTypeSecurityRestriction:
		return true // Can be skipped or handled gracefully
	case ErrorTypeInvalidFont, ErrorTypeInvalidImage, ErrorTypeInvalidForm:
		return true // Can use fallbacks or skip
	case ErrorTypeMalformedPage, ErrorTypeInvalidAnnotation, ErrorTypeInvalidMetadata:
		return true // Non-critical, can continue
	case ErrorTypeResourceNotFound:
		return true // Can use defaults or skip
	default:
		return false
	}
}

// NewPDFError creates a new PDFError with full context
func NewPDFError(errorType ErrorType, message string) *PDFError {
	return &PDFError{
		Type:        errorType,
		Message:     message,
		Recoverable: errorType.IsRecoverable(),
		Timestamp:   time.Now(),
		StackTrace:  string(debug.Stack()),
	}
}

// NewPDFErrorWithContext creates a new PDFError with additional context
func NewPDFErrorWithContext(errorType ErrorType, message, context string) *PDFError {
	return &PDFError{
		Type:        errorType,
		Message:     message,
		Context:     context,
		Recoverable: errorType.IsRecoverable(),
		Timestamp:   time.Now(),
		StackTrace:  string(debug.Stack()),
	}
}

// NewPDFErrorWithLocation creates a new PDFError with file location information
func NewPDFErrorWithLocation(errorType ErrorType, message string, offset int64, objNum, genNum int) *PDFError {
	return &PDFError{
		Type:        errorType,
		Message:     message,
		Offset:      offset,
		ObjectNum:   objNum,
		GenNum:      genNum,
		Recoverable: errorType.IsRecoverable(),
		Timestamp:   time.Now(),
		StackTrace:  string(debug.Stack()),
	}
}

// WrapError wraps a standard error as a PDFError
func WrapError(errorType ErrorType, err error) *PDFError {
	return &PDFError{
		Type:        errorType,
		Message:     err.Error(),
		Recoverable: errorType.IsRecoverable(),
		Timestamp:   time.Now(),
		StackTrace:  string(debug.Stack()),
	}
}

// WithContext adds context to an existing PDFError
func (e *PDFError) WithContext(context string) *PDFError {
	e.Context = context
	return e
}

// WithLocation adds location information to an existing PDFError
func (e *PDFError) WithLocation(offset int64, objNum, genNum int) *PDFError {
	e.Offset = offset
	e.ObjectNum = objNum
	e.GenNum = genNum
	return e
}

// WithFile adds file path information to an existing PDFError
func (e *PDFError) WithFile(filePath string) *PDFError {
	e.FilePath = filePath
	return e
}

// WithPage adds page number information to an existing PDFError
func (e *PDFError) WithPage(pageNumber int) *PDFError {
	e.PageNumber = pageNumber
	return e
}

// GetSeverity returns the severity of this specific error
func (e *PDFError) GetSeverity() ErrorSeverity {
	return e.Type.GetSeverity()
}

// IsCritical returns true if this error is critical or fatal
func (e *PDFError) IsCritical() bool {
	severity := e.GetSeverity()
	return severity == SeverityCritical || severity == SeverityFatal
}

// ErrorCollection manages multiple PDF errors
type ErrorCollection struct {
	Errors   []*PDFError `json:"errors"`
	Warnings []*PDFError `json:"warnings"`
	FilePath string      `json:"file_path,omitempty"`
}

// NewErrorCollection creates a new error collection
func NewErrorCollection(filePath string) *ErrorCollection {
	return &ErrorCollection{
		Errors:   make([]*PDFError, 0),
		Warnings: make([]*PDFError, 0),
		FilePath: filePath,
	}
}

// Add adds an error to the appropriate collection based on severity
func (ec *ErrorCollection) Add(err *PDFError) {
	if err.FilePath == "" && ec.FilePath != "" {
		err.FilePath = ec.FilePath
	}

	severity := err.GetSeverity()
	if severity == SeverityWarning || severity == SeverityInfo {
		ec.Warnings = append(ec.Warnings, err)
	} else {
		ec.Errors = append(ec.Errors, err)
	}
}

// HasCriticalErrors returns true if any critical errors exist
func (ec *ErrorCollection) HasCriticalErrors() bool {
	for _, err := range ec.Errors {
		if err.IsCritical() {
			return true
		}
	}
	return false
}

// GetRecoverableErrors returns all recoverable errors
func (ec *ErrorCollection) GetRecoverableErrors() []*PDFError {
	recoverable := make([]*PDFError, 0)
	for _, err := range ec.Errors {
		if err.Recoverable {
			recoverable = append(recoverable, err)
		}
	}
	return recoverable
}

// GetUnrecoverableErrors returns all unrecoverable errors
func (ec *ErrorCollection) GetUnrecoverableErrors() []*PDFError {
	unrecoverable := make([]*PDFError, 0)
	for _, err := range ec.Errors {
		if !err.Recoverable {
			unrecoverable = append(unrecoverable, err)
		}
	}
	return unrecoverable
}

// Count returns the total number of errors and warnings
func (ec *ErrorCollection) Count() (errors, warnings int) {
	return len(ec.Errors), len(ec.Warnings)
}

// Summary returns a text summary of all errors and warnings
func (ec *ErrorCollection) Summary() string {
	errorCount, warningCount := ec.Count()
	if errorCount == 0 && warningCount == 0 {
		return "No errors or warnings"
	}

	summary := fmt.Sprintf("Found %d error(s) and %d warning(s)", errorCount, warningCount)

	if ec.HasCriticalErrors() {
		summary += " (including critical errors)"
	}

	return summary
}
