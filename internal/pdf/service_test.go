package pdf

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewService(t *testing.T) {
	maxFileSize := int64(1024 * 1024) // 1MB
	service := NewService(maxFileSize)

	if service == nil {
		t.Fatal("NewService returned nil")
	}

	if service.maxFileSize != maxFileSize {
		t.Errorf("Expected maxFileSize to be %d, got %d", maxFileSize, service.maxFileSize)
	}

	// Verify all components are initialized
	if service.reader == nil {
		t.Error("reader component should not be nil")
	}
	if service.validator == nil {
		t.Error("validator component should not be nil")
	}
	if service.stats == nil {
		t.Error("stats component should not be nil")
	}
	if service.assets == nil {
		t.Error("assets component should not be nil")
	}
	if service.search == nil {
		t.Error("search component should not be nil")
	}
}

func TestService_GetMaxFileSize(t *testing.T) {
	maxFileSize := int64(2 * 1024 * 1024) // 2MB
	service := NewService(maxFileSize)

	result := service.GetMaxFileSize()
	if result != maxFileSize {
		t.Errorf("Expected GetMaxFileSize to return %d, got %d", maxFileSize, result)
	}
}

func TestService_ValidateConfiguration(t *testing.T) {
	tests := []struct {
		name          string
		maxFileSize   int64
		expectedError bool
		errorMsg      string
	}{
		{
			name:          "valid configuration",
			maxFileSize:   1024 * 1024, // 1MB
			expectedError: false,
		},
		{
			name:          "zero max file size",
			maxFileSize:   0,
			expectedError: true,
			errorMsg:      "maxFileSize must be greater than 0",
		},
		{
			name:          "negative max file size",
			maxFileSize:   -1,
			expectedError: true,
			errorMsg:      "maxFileSize must be greater than 0",
		},
		{
			name:          "max file size too large",
			maxFileSize:   2 * 1024 * 1024 * 1024, // 2GB
			expectedError: true,
			errorMsg:      "maxFileSize cannot exceed 1GB",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewService(tt.maxFileSize)
			err := service.ValidateConfiguration()

			if tt.expectedError && err == nil {
				t.Errorf("expected error but got none")
			}
			if !tt.expectedError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if tt.expectedError && tt.errorMsg != "" && err.Error() != tt.errorMsg {
				t.Errorf("expected error message '%s' but got '%s'", tt.errorMsg, err.Error())
			}
		})
	}
}

func TestService_PDFValidateFile(t *testing.T) {
	service := NewService(1024 * 1024)

	// Create temp directory for test files
	tempDir, err := os.MkdirTemp("", "service_validate_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test file
	testFile := filepath.Join(tempDir, "test.pdf")
	if err := os.WriteFile(testFile, make([]byte, 1024), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	req := PDFValidateFileRequest{
		Path: testFile,
	}

	result, err := service.PDFValidateFile(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("result should not be nil")
	}

	if result.Path != testFile {
		t.Errorf("expected path %s but got %s", testFile, result.Path)
	}

	// The file should be invalid since it's not a real PDF
	if result.Valid {
		t.Errorf("expected file to be invalid")
	}
}

func TestService_PDFSearchDirectory(t *testing.T) {
	service := NewService(1024 * 1024)

	// Create temp directory for test files
	tempDir, err := os.MkdirTemp("", "service_search_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test PDF files
	testFiles := []string{"doc1.pdf", "doc2.pdf"}
	for _, filename := range testFiles {
		filePath := filepath.Join(tempDir, filename)
		if err := os.WriteFile(filePath, make([]byte, 1024), 0644); err != nil {
			t.Fatalf("failed to create test file %s: %v", filename, err)
		}
	}

	req := PDFSearchDirectoryRequest{
		Directory: tempDir,
		Query:     "",
	}

	result, err := service.PDFSearchDirectory(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("result should not be nil")
	}

	if result.Directory != tempDir {
		t.Errorf("expected directory %s but got %s", tempDir, result.Directory)
	}

	if result.TotalCount != 2 {
		t.Errorf("expected 2 files but got %d", result.TotalCount)
	}
}

func TestService_PDFStatsDirectory(t *testing.T) {
	service := NewService(1024 * 1024)

	// Create temp directory for test files
	tempDir, err := os.MkdirTemp("", "service_stats_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test PDF files
	testFiles := map[string]int{
		"small.pdf":  512,
		"medium.pdf": 1024,
		"large.pdf":  2048,
	}

	for filename, size := range testFiles {
		filePath := filepath.Join(tempDir, filename)
		if err := os.WriteFile(filePath, make([]byte, size), 0644); err != nil {
			t.Fatalf("failed to create test file %s: %v", filename, err)
		}
	}

	req := PDFStatsDirectoryRequest{
		Directory: tempDir,
	}

	result, err := service.PDFStatsDirectory(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("result should not be nil")
	}

	if result.Directory != tempDir {
		t.Errorf("expected directory %s but got %s", tempDir, result.Directory)
	}

	if result.TotalFiles != 3 {
		t.Errorf("expected 3 files but got %d", result.TotalFiles)
	}

	expectedTotalSize := int64(512 + 1024 + 2048)
	if result.TotalSize != expectedTotalSize {
		t.Errorf("expected total size %d but got %d", expectedTotalSize, result.TotalSize)
	}
}

func TestService_IsValidPDF(t *testing.T) {
	service := NewService(1024 * 1024)

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
			result := service.IsValidPDF(tt.filePath)
			if result != tt.expected {
				t.Errorf("expected %v but got %v", tt.expected, result)
			}
		})
	}
}

func TestService_CountPDFsInDirectory(t *testing.T) {
	service := NewService(1024 * 1024)

	// Create temp directory for test files
	tempDir, err := os.MkdirTemp("", "service_count_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test files
	pdfFiles := []string{"doc1.pdf", "doc2.pdf", "doc3.pdf"}
	nonPdfFiles := []string{"doc.txt", "image.jpg"}

	for _, filename := range append(pdfFiles, nonPdfFiles...) {
		filePath := filepath.Join(tempDir, filename)
		if err := os.WriteFile(filePath, make([]byte, 1024), 0644); err != nil {
			t.Fatalf("failed to create test file %s: %v", filename, err)
		}
	}

	count, err := service.CountPDFsInDirectory(tempDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedCount := len(pdfFiles)
	if count != expectedCount {
		t.Errorf("expected count %d but got %d", expectedCount, count)
	}
}

func TestService_FindPDFsInDirectory(t *testing.T) {
	service := NewService(1024 * 1024)

	// Create temp directory for test files
	tempDir, err := os.MkdirTemp("", "service_find_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test files
	pdfFiles := []string{"doc1.pdf", "doc2.pdf"}
	nonPdfFiles := []string{"doc.txt", "image.jpg"}

	for _, filename := range append(pdfFiles, nonPdfFiles...) {
		filePath := filepath.Join(tempDir, filename)
		if err := os.WriteFile(filePath, make([]byte, 1024), 0644); err != nil {
			t.Fatalf("failed to create test file %s: %v", filename, err)
		}
	}

	files, err := service.FindPDFsInDirectory(tempDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedCount := len(pdfFiles)
	if len(files) != expectedCount {
		t.Errorf("expected %d files but got %d", expectedCount, len(files))
	}

	// Verify all returned files are PDFs
	for _, file := range files {
		if !service.search.isPDFFile(file.Name) {
			t.Errorf("non-PDF file returned: %s", file.Name)
		}
	}
}

func TestService_SearchByPattern(t *testing.T) {
	service := NewService(1024 * 1024)

	// Create temp directory for test files
	tempDir, err := os.MkdirTemp("", "service_pattern_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test files with different patterns
	testFiles := []string{
		"report_2023.pdf",
		"report_2024.pdf",
		"summary_2023.pdf",
		"document.pdf",
	}

	for _, filename := range testFiles {
		filePath := filepath.Join(tempDir, filename)
		if err := os.WriteFile(filePath, make([]byte, 1024), 0644); err != nil {
			t.Fatalf("failed to create test file %s: %v", filename, err)
		}
	}

	result, err := service.SearchByPattern(tempDir, "report_*.pdf")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedCount := 2 // report_2023.pdf and report_2024.pdf
	if result.TotalCount != expectedCount {
		t.Errorf("expected %d files but got %d", expectedCount, result.TotalCount)
	}
}

func TestService_GetSupportedImageFormats(t *testing.T) {
	service := NewService(1024 * 1024)

	formats := service.GetSupportedImageFormats()
	if len(formats) == 0 {
		t.Error("expected at least one supported image format")
	}

	// Verify some expected formats are present
	expectedFormats := []string{"JPEG", "PNG/Deflate"}
	formatMap := make(map[string]bool)
	for _, format := range formats {
		formatMap[format] = true
	}

	for _, expected := range expectedFormats {
		if !formatMap[expected] {
			t.Errorf("expected format %s not found in supported formats", expected)
		}
	}
}

func TestService_PDFReadFile_ErrorHandling(t *testing.T) {
	service := NewService(1024 * 1024)

	// Test with empty path
	req := PDFReadFileRequest{
		Path: "",
	}

	result, err := service.PDFReadFile(req)
	if err == nil {
		t.Error("expected error for empty path")
	}
	if result != nil {
		t.Error("result should be nil on error")
	}
}

func TestService_PDFAssetsFile_ErrorHandling(t *testing.T) {
	service := NewService(1024 * 1024)

	// Test with empty path
	req := PDFAssetsFileRequest{
		Path: "",
	}

	result, err := service.PDFAssetsFile(req)
	if err == nil {
		t.Error("expected error for empty path")
	}
	if result != nil {
		t.Error("result should be nil on error")
	}
}

func TestService_PDFStatsFile_ErrorHandling(t *testing.T) {
	service := NewService(1024 * 1024)

	// Test with empty path
	req := PDFStatsFileRequest{
		Path: "",
	}

	result, err := service.PDFStatsFile(req)
	if err == nil {
		t.Error("expected error for empty path")
	}
	if result != nil {
		t.Error("result should be nil on error")
	}
}
