package pdf

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidator_ValidateFile(t *testing.T) {
	validator := NewValidator(1024 * 1024) // 1MB limit

	tests := []struct {
		name        string
		req         PDFValidateFileRequest
		expectValid bool
		expectError bool
	}{
		{
			name: "empty path",
			req: PDFValidateFileRequest{
				Path: "",
			},
			expectValid: false,
			expectError: false, // ValidateFile doesn't return processing errors
		},
		{
			name: "non-existent file",
			req: PDFValidateFileRequest{
				Path: "/non/existent/file.pdf",
			},
			expectValid: false,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := validator.ValidateFile(tt.req)

			if tt.expectError && err == nil {
				t.Errorf("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if result == nil {
				t.Fatalf("result should not be nil")
			}

			if result.Valid != tt.expectValid {
				t.Errorf("expected Valid=%v but got %v", tt.expectValid, result.Valid)
			}

			if result.Path != tt.req.Path {
				t.Errorf("expected Path=%s but got %s", tt.req.Path, result.Path)
			}

			if !tt.expectValid && result.Message == "" {
				t.Errorf("expected validation message for invalid file")
			}
		})
	}
}

func TestValidator_ValidateFileInfo(t *testing.T) {
	validator := NewValidator(1024 * 1024) // 1MB limit

	// Create a temporary directory and files for testing
	tempDir, err := os.MkdirTemp("", "pdf_validator_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test files
	validPDFPath := filepath.Join(tempDir, "valid.pdf")
	largePDFPath := filepath.Join(tempDir, "large.pdf")
	emptyPDFPath := filepath.Join(tempDir, "empty.pdf")
	nonPDFPath := filepath.Join(tempDir, "document.txt")

	// Create files with different sizes
	if err := os.WriteFile(validPDFPath, make([]byte, 1024), 0o644); err != nil {
		t.Fatalf("failed to create valid PDF: %v", err)
	}
	if err := os.WriteFile(largePDFPath, make([]byte, 2*1024*1024), 0o644); err != nil {
		t.Fatalf("failed to create large PDF: %v", err)
	}
	if err := os.WriteFile(emptyPDFPath, []byte{}, 0o644); err != nil {
		t.Fatalf("failed to create empty PDF: %v", err)
	}
	if err := os.WriteFile(nonPDFPath, []byte("not a pdf"), 0o644); err != nil {
		t.Fatalf("failed to create non-PDF: %v", err)
	}

	tests := []struct {
		name        string
		filePath    string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid PDF file",
			filePath:    validPDFPath,
			expectError: false,
		},
		{
			name:        "large PDF file",
			filePath:    largePDFPath,
			expectError: true,
			errorMsg:    "file too large",
		},
		{
			name:        "empty PDF file",
			filePath:    emptyPDFPath,
			expectError: true,
			errorMsg:    "file is empty",
		},
		{
			name:        "non-PDF file",
			filePath:    nonPDFPath,
			expectError: true,
			errorMsg:    "file is not a PDF",
		},
		{
			name:        "directory instead of file",
			filePath:    tempDir,
			expectError: true,
			errorMsg:    "path is a directory",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fileInfo, err := os.Stat(tt.filePath)
			if err != nil {
				t.Fatalf("failed to stat file: %v", err)
			}

			err = validator.ValidateFileInfo(tt.filePath, fileInfo)

			if tt.expectError && err == nil {
				t.Errorf("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if tt.expectError && tt.errorMsg != "" {
				if err == nil || err.Error() == "" {
					t.Errorf("expected error message containing '%s'", tt.errorMsg)
				}
			}
		})
	}
}

func TestValidator_IsValidPDF(t *testing.T) {
	validator := NewValidator(1024 * 1024) // 1MB limit

	tests := []struct {
		name     string
		filePath string
		expected bool
	}{
		{
			name:     "empty path",
			filePath: "",
			expected: false,
		},
		{
			name:     "non-existent file",
			filePath: "/non/existent/file.pdf",
			expected: false,
		},
		{
			name:     "non-PDF extension",
			filePath: "/path/to/document.txt",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.IsValidPDF(tt.filePath)
			if result != tt.expected {
				t.Errorf("expected %v but got %v", tt.expected, result)
			}
		})
	}
}

func TestValidator_validatePDFFile_EdgeCases(t *testing.T) {
	validator := NewValidator(1024) // Small limit for testing

	tests := []struct {
		name        string
		filePath    string
		expectError bool
		setupFunc   func(string) error
	}{
		{
			name:        "file with PDF extension but not PDF content",
			filePath:    "fake.pdf",
			expectError: true,
			setupFunc: func(path string) error {
				return os.WriteFile(path, []byte("This is not a PDF file"), 0o644)
			},
		},
	}

	tempDir, err := os.MkdirTemp("", "pdf_validator_edge_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fullPath := filepath.Join(tempDir, tt.filePath)

			if tt.setupFunc != nil {
				if err := tt.setupFunc(fullPath); err != nil {
					t.Fatalf("failed to setup test file: %v", err)
				}
			}

			err := validator.validatePDFFile(fullPath)

			if tt.expectError && err == nil {
				t.Errorf("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestNewValidator(t *testing.T) {
	maxFileSize := int64(2 * 1024 * 1024) // 2MB
	validator := NewValidator(maxFileSize)

	if validator == nil {
		t.Fatal("NewValidator returned nil")
	}

	if validator.maxFileSize != maxFileSize {
		t.Errorf("expected maxFileSize=%d but got %d", maxFileSize, validator.maxFileSize)
	}
}

func BenchmarkValidator_ValidateFileInfo(b *testing.B) {
	validator := NewValidator(1024 * 1024)

	// Create a temporary file for benchmarking
	tempDir, err := os.MkdirTemp("", "pdf_validator_bench")
	if err != nil {
		b.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	testFile := filepath.Join(tempDir, "test.pdf")
	if err := os.WriteFile(testFile, make([]byte, 1024), 0o644); err != nil {
		b.Fatalf("failed to create test file: %v", err)
	}

	fileInfo, err := os.Stat(testFile)
	if err != nil {
		b.Fatalf("failed to stat file: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = validator.ValidateFileInfo(testFile, fileInfo)
	}
}
