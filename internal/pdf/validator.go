package pdf

import (
	"fmt"
	"os"
	"strings"

	"github.com/ledongthuc/pdf"
)

// Validator handles PDF file validation operations
type Validator struct {
	maxFileSize int64
}

// NewValidator creates a new PDF validator with the specified constraints
func NewValidator(maxFileSize int64) *Validator {
	return &Validator{
		maxFileSize: maxFileSize,
	}
}

// ValidateFile performs comprehensive validation on a PDF file
func (v *Validator) ValidateFile(req PDFValidateFileRequest) (*PDFValidateFileResult, error) {
	result := &PDFValidateFileResult{
		Path:  req.Path,
		Valid: false,
	}

	err := v.validatePDFFile(req.Path)
	if err != nil {
		result.Message = err.Error()
		return result, nil //nolint:nilerr // Return result with validation error, not a processing error
	}

	result.Valid = true
	return result, nil
}

// validatePDFFile performs detailed validation on a PDF file
func (v *Validator) validatePDFFile(filePath string) error {
	if filePath == "" {
		return fmt.Errorf("path cannot be empty")
	}

	// Check if file exists and get basic info
	fileInfo, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		return fmt.Errorf("file does not exist: %s", filePath)
	}
	if err != nil {
		return fmt.Errorf("cannot access file: %w", err)
	}

	// Check if it's a regular file (not a directory)
	if fileInfo.IsDir() {
		return fmt.Errorf("path is a directory, not a file: %s", filePath)
	}

	// Check file extension
	if !strings.HasSuffix(strings.ToLower(filePath), ".pdf") {
		return fmt.Errorf("file is not a PDF: %s", filePath)
	}

	// Check file size
	if fileInfo.Size() == 0 {
		return fmt.Errorf("file is empty: %s", filePath)
	}

	if fileInfo.Size() > v.maxFileSize {
		return fmt.Errorf("file too large: %d bytes (max: %d bytes)",
			fileInfo.Size(), v.maxFileSize)
	}

	// Try to open the PDF to validate it's a valid PDF file
	f, _, err := pdf.Open(filePath)
	if err != nil {
		return fmt.Errorf("invalid PDF file: %w", err)
	}
	defer f.Close()

	return nil
}

// IsValidPDF performs a quick check to see if a file is a valid PDF
func (v *Validator) IsValidPDF(filePath string) bool {
	return v.validatePDFFile(filePath) == nil
}

// ValidateFileInfo performs basic validation on file info without opening the PDF
func (v *Validator) ValidateFileInfo(filePath string, fileInfo os.FileInfo) error {
	if fileInfo.IsDir() {
		return fmt.Errorf("path is a directory, not a file: %s", filePath)
	}

	if !strings.HasSuffix(strings.ToLower(filePath), ".pdf") {
		return fmt.Errorf("file is not a PDF: %s", filePath)
	}

	if fileInfo.Size() == 0 {
		return fmt.Errorf("file is empty: %s", filePath)
	}

	if fileInfo.Size() > v.maxFileSize {
		return fmt.Errorf("file too large: %d bytes (max: %d bytes)",
			fileInfo.Size(), v.maxFileSize)
	}

	return nil
}
