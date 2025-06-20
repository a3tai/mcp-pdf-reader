package pdf

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestNewExtractionService(t *testing.T) {
	tests := []struct {
		name        string
		maxFileSize int64
		want        int64
	}{
		{
			name:        "standard max file size",
			maxFileSize: 100 * 1024 * 1024,
			want:        100 * 1024 * 1024,
		},
		{
			name:        "small max file size",
			maxFileSize: 1024,
			want:        1024,
		},
		{
			name:        "zero max file size",
			maxFileSize: 0,
			want:        0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewExtractionService(tt.maxFileSize)
			if service == nil {
				t.Fatal("NewExtractionService returned nil")
			}
			if service.maxFileSize != tt.want {
				t.Errorf("NewExtractionService() maxFileSize = %v, want %v", service.maxFileSize, tt.want)
			}
		})
	}
}

func TestExtractionService_ExtractStructured(t *testing.T) {
	service := NewExtractionService(100 * 1024 * 1024)

	tests := []struct {
		name      string
		req       PDFExtractRequest
		wantError bool
		errorMsg  string
	}{
		{
			name: "empty path",
			req: PDFExtractRequest{
				Path: "",
				Mode: "structured",
			},
			wantError: true,
			errorMsg:  "path cannot be empty",
		},
		{
			name: "non-existent file",
			req: PDFExtractRequest{
				Path: "/non/existent/file.pdf",
				Mode: "structured",
			},
			wantError: true,
			errorMsg:  "file does not exist",
		},
		{
			name: "directory instead of file",
			req: PDFExtractRequest{
				Path: createTempDir(t),
				Mode: "structured",
			},
			wantError: true,
			errorMsg:  "path is a directory, not a file",
		},
		{
			name: "non-PDF file",
			req: PDFExtractRequest{
				Path: createTempFile(t, "test.txt", "not a pdf"),
				Mode: "structured",
			},
			wantError: true,
			errorMsg:  "not a PDF file",
		},
		{
			name: "file too large",
			req: PDFExtractRequest{
				Path: createLargeFile(t, service.maxFileSize+1),
				Mode: "structured",
			},
			wantError: true,
			errorMsg:  "file too large",
		},
		{
			name: "valid request with default mode",
			req: PDFExtractRequest{
				Path: createTempFile(t, "test.pdf", generateMinimalPDFContent()),
				Mode: "",
			},
			wantError: false, // Valid PDF should parse successfully
		},
		{
			name: "valid request with structured mode",
			req: PDFExtractRequest{
				Path: createTempFile(t, "test.pdf", generateMinimalPDFContent()),
				Mode: "structured",
				Config: ExtractConfig{
					ExtractText:        true,
					IncludeCoordinates: true,
				},
			},
			wantError: false, // Valid PDF should parse successfully
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := service.ExtractStructured(tt.req)

			if tt.wantError {
				if err == nil {
					t.Errorf("ExtractStructured() expected error but got none")
					return
				}
				if tt.errorMsg != "" && !containsString(err.Error(), tt.errorMsg) {
					t.Errorf("ExtractStructured() error = %v, want error containing %v", err, tt.errorMsg)
				}
				return
			}

			if err != nil {
				t.Errorf("ExtractStructured() unexpected error = %v", err)
				return
			}

			if result == nil {
				t.Error("ExtractStructured() returned nil result")
				return
			}

			// Validate result structure
			if result.FilePath != tt.req.Path {
				t.Errorf("ExtractStructured() FilePath = %v, want %v", result.FilePath, tt.req.Path)
			}

			expectedMode := tt.req.Mode
			if expectedMode == "" {
				expectedMode = "structured"
			}
			if result.Mode != expectedMode {
				t.Errorf("ExtractStructured() Mode = %v, want %v", result.Mode, expectedMode)
			}

			if result.Summary.TotalElements < 0 {
				t.Errorf("ExtractStructured() Summary.TotalElements = %v, want >= 0", result.Summary.TotalElements)
			}
		})
	}
}

func TestExtractionService_ExtractTables(t *testing.T) {
	service := NewExtractionService(100 * 1024 * 1024)

	tests := []struct {
		name      string
		req       PDFExtractRequest
		wantError bool
		errorMsg  string
	}{
		{
			name: "empty path",
			req: PDFExtractRequest{
				Path: "",
			},
			wantError: true,
		},
		{
			name: "valid request",
			req: PDFExtractRequest{
				Path: createTempFile(t, "test.pdf", generateMinimalPDFContent()),
				Config: ExtractConfig{
					ExtractTables:      true,
					IncludeCoordinates: true,
				},
			},
			wantError: false, // Valid PDF should parse successfully
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := service.ExtractTables(tt.req)

			if tt.wantError {
				if err == nil {
					t.Errorf("ExtractTables() expected error but got none")
					return
				}
				if tt.errorMsg != "" && !containsString(err.Error(), tt.errorMsg) {
					t.Errorf("ExtractTables() error = %v, want error containing %v", err, tt.errorMsg)
				}
				return
			}

			if err != nil {
				t.Errorf("ExtractTables() unexpected error = %v", err)
				return
			}

			if result == nil {
				t.Error("ExtractTables() returned nil result")
				return
			}

			if result.Mode != "table" {
				t.Errorf("ExtractTables() Mode = %v, want table", result.Mode)
			}
		})
	}
}

func TestExtractionService_ExtractSemantic(t *testing.T) {
	service := NewExtractionService(100 * 1024 * 1024)

	req := PDFExtractRequest{
		Path: createTempFile(t, "test.pdf", generateMinimalPDFContent()),
	}

	result, err := service.ExtractSemantic(req)
	if err != nil {
		t.Errorf("ExtractSemantic() unexpected error = %v", err)
		return
	}

	if result == nil {
		t.Error("ExtractSemantic() returned nil result")
	}
}

func TestExtractionService_ExtractComplete(t *testing.T) {
	service := NewExtractionService(100 * 1024 * 1024)

	// Test basic functionality
	req := PDFExtractRequest{
		Path: createTempFile(t, "test.pdf", generateMinimalPDFContent()),
		Mode: "complete",
	}

	result, err := service.ExtractComplete(req)
	if err != nil {
		t.Errorf("ExtractComplete() unexpected error = %v", err)
		return
	}

	if result == nil {
		t.Error("ExtractComplete() returned nil result")
	}
}

func TestExtractionService_QueryContent(t *testing.T) {
	service := NewExtractionService(100 * 1024 * 1024)

	tests := []struct {
		name      string
		req       PDFQueryRequest
		wantError bool
		errorMsg  string
	}{
		{
			name: "empty path",
			req: PDFQueryRequest{
				Path: "",
				Query: ContentQuery{
					TextQuery: "test",
				},
			},
			wantError: true,
		},
		{
			name: "valid query",
			req: PDFQueryRequest{
				Path: createTempFile(t, "test.pdf", generateMinimalPDFContent()),
				Query: ContentQuery{
					TextQuery:     "test",
					ContentTypes:  []string{"text"},
					MinConfidence: 0.5,
				},
			},
			wantError: false, // Valid PDF should parse successfully
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := service.QueryContent(tt.req)

			if tt.wantError {
				if err == nil {
					t.Errorf("QueryContent() expected error but got none")
					return
				}
				if tt.errorMsg != "" && !containsString(err.Error(), tt.errorMsg) {
					t.Errorf("QueryContent() error = %v, want error containing %v", err, tt.errorMsg)
				}
				return
			}

			if err != nil {
				t.Errorf("QueryContent() unexpected error = %v", err)
				return
			}

			if result == nil {
				t.Error("QueryContent() returned nil result")
				return
			}

			if result.FilePath != tt.req.Path {
				t.Errorf("QueryContent() FilePath = %v, want %v", result.FilePath, tt.req.Path)
			}

			if result.MatchCount < 0 {
				t.Errorf("QueryContent() MatchCount = %v, want >= 0", result.MatchCount)
			}
		})
	}
}

func TestExtractionService_GetPageInfo(t *testing.T) {
	service := NewExtractionService(100 * 1024 * 1024)

	tests := []struct {
		name      string
		path      string
		wantError bool
	}{
		{
			name:      "empty path",
			path:      "",
			wantError: true,
		},
		{
			name:      "non-existent file",
			path:      "/non/existent/file.pdf",
			wantError: true,
		},
		{
			name:      "valid path",
			path:      createTempFile(t, "test.pdf", generateMinimalPDFContent()),
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := service.GetPageInfo(tt.path)

			if tt.wantError {
				if err == nil {
					t.Errorf("GetPageInfo() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("GetPageInfo() unexpected error = %v", err)
				return
			}

			if result == nil {
				t.Error("GetPageInfo() returned nil result")
			}
		})
	}
}

func TestExtractionService_GetMetadata(t *testing.T) {
	service := NewExtractionService(100 * 1024 * 1024)

	tests := []struct {
		name      string
		path      string
		wantError bool
	}{
		{
			name:      "empty path",
			path:      "",
			wantError: true,
		},
		{
			name:      "non-existent file",
			path:      "/non/existent/file.pdf",
			wantError: true,
		},
		{
			name:      "valid path",
			path:      createTempFile(t, "test.pdf", generateMinimalPDFContent()),
			wantError: false,
		},
		{
			name:      "real PDF file",
			path:      "../../docs/examples/dev-example.pdf",
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Skip tests if the PDF file doesn't exist (e.g., in CI environments)
			if !tt.wantError && !fileExists(tt.path) {
				t.Skipf("Test PDF file not found: %s", tt.path)
				return
			}

			result, err := service.GetMetadata(tt.path)

			if tt.wantError {
				if err == nil {
					t.Errorf("GetMetadata() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("GetMetadata() unexpected error = %v", err)
				return
			}

			if result == nil {
				t.Error("GetMetadata() returned nil result")
				return
			}

			// For real PDF files, verify that we can extract some metadata
			if tt.name == "real PDF file" {
				// The metadata extraction should work and return a non-nil result
				// We don't expect specific values, but the extraction should succeed
				t.Logf("Extracted metadata: Title='%s', Author='%s', Subject='%s', Producer='%s', Version='%s', Encrypted=%v",
					result.Title, result.Author, result.Subject, result.Producer, result.Version, result.Encrypted)

				// Verify that our implementation returns a valid version
				if result.Version == "" {
					t.Error("GetMetadata() Version should not be empty")
				}
			}
		})
	}
}

func TestExtractionService_buildQuerySummary(t *testing.T) {
	service := NewExtractionService(100 * 1024 * 1024)

	elements := []ContentElement{
		{
			Type:       "text",
			PageNumber: 1,
			Confidence: 0.9,
		},
		{
			Type:       "image",
			PageNumber: 1,
			Confidence: 0.8,
		},
		{
			Type:       "text",
			PageNumber: 2,
			Confidence: 0.7,
		},
	}

	summary := service.buildQuerySummary(elements)

	expectedTypeBreakdown := map[string]int{
		"text":  2,
		"image": 1,
	}

	if len(summary.TypeBreakdown) != len(expectedTypeBreakdown) {
		t.Errorf("buildQuerySummary() TypeBreakdown length = %v, want %v", len(summary.TypeBreakdown), len(expectedTypeBreakdown))
	}

	for contentType, expectedCount := range expectedTypeBreakdown {
		if count, exists := summary.TypeBreakdown[contentType]; !exists || count != expectedCount {
			t.Errorf("buildQuerySummary() TypeBreakdown[%s] = %v, want %v", contentType, count, expectedCount)
		}
	}

	expectedPageBreakdown := map[int]int{
		1: 2,
		2: 1,
	}

	for page, expectedCount := range expectedPageBreakdown {
		if count, exists := summary.PageBreakdown[page]; !exists || count != expectedCount {
			t.Errorf("buildQuerySummary() PageBreakdown[%d] = %v, want %v", page, count, expectedCount)
		}
	}

	expectedConfidence := (0.9 + 0.8 + 0.7) / 3.0
	if abs(summary.Confidence-expectedConfidence) > 0.01 {
		t.Errorf("buildQuerySummary() Confidence = %v, want %v", summary.Confidence, expectedConfidence)
	}
}

// Helper functions

func createTempDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "pdf_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	t.Cleanup(func() {
		os.RemoveAll(dir)
	})
	return dir
}

func createTempFile(t *testing.T, name, content string) string {
	t.Helper()
	dir := createTempDir(t)
	filePath := filepath.Join(dir, name)

	err := os.WriteFile(filePath, []byte(content), 0o644)
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	return filePath
}

func createLargeFile(t *testing.T, size int64) string {
	t.Helper()
	dir := createTempDir(t)
	filePath := filepath.Join(dir, "large.pdf")

	file, err := os.Create(filePath)
	if err != nil {
		t.Fatalf("Failed to create large file: %v", err)
	}
	defer file.Close()

	if err := file.Truncate(size); err != nil {
		t.Fatalf("Failed to truncate file: %v", err)
	}

	return filePath
}

func generateMinimalPDFContent() string {
	// Create a properly formatted PDF with accurate byte offsets
	pdf := "%PDF-1.4\n"

	// Object 1 - Catalog
	obj1Start := len(pdf)
	pdf += "1 0 obj\n<<\n/Type /Catalog\n/Pages 2 0 R\n>>\nendobj\n"

	// Object 2 - Pages
	obj2Start := len(pdf)
	pdf += "2 0 obj\n<<\n/Type /Pages\n/Kids [3 0 R]\n/Count 1\n>>\nendobj\n"

	// Object 3 - Page
	obj3Start := len(pdf)
	pdf += "3 0 obj\n<<\n/Type /Page\n/Parent 2 0 R\n/MediaBox [0 0 612 792]\n/Resources <<>>\n>>\nendobj\n"

	// Cross-reference table
	xrefStart := len(pdf)
	pdf += "xref\n0 4\n0000000000 65535 f \n"
	pdf += fmt.Sprintf("%010d 00000 n \n", obj1Start)
	pdf += fmt.Sprintf("%010d 00000 n \n", obj2Start)
	pdf += fmt.Sprintf("%010d 00000 n \n", obj3Start)

	// Trailer
	pdf += "trailer\n<<\n/Size 4\n/Root 1 0 R\n>>\nstartxref\n"
	pdf += fmt.Sprintf("%d\n", xrefStart)
	pdf += "%%EOF"

	return pdf
}

// Helper function to check if a file exists
func fileExists(path string) bool {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return false
	}
	return true
}

func containsString(s, substr string) bool {
	return len(substr) == 0 || len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsString(s[1:], substr)))
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
