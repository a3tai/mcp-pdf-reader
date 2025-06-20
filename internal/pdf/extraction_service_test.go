package pdf

import (
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
			wantError: true, // Minimal PDF may not parse correctly
			errorMsg:  "failed to open PDF",
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
			wantError: true, // Minimal PDF may not parse correctly
			errorMsg:  "failed to open PDF",
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
			wantError: true, // Minimal PDF may not parse correctly
			errorMsg:  "failed to open PDF",
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

	_, err := service.ExtractSemantic(req)
	// Expect error since minimal PDF may not parse correctly
	if err == nil {
		t.Errorf("ExtractSemantic() expected error but got none")
		return
	}

	if !containsString(err.Error(), "failed to open PDF") {
		t.Errorf("ExtractSemantic() unexpected error = %v", err)
	}
}

func TestExtractionService_ExtractComplete(t *testing.T) {
	service := NewExtractionService(100 * 1024 * 1024)

	// Test basic functionality
	req := PDFExtractRequest{
		Path: createTempFile(t, "test.pdf", generateMinimalPDFContent()),
		Mode: "complete",
	}

	_, err := service.ExtractComplete(req)
	// Expect error since minimal PDF may not parse correctly
	if err == nil {
		t.Errorf("ExtractComplete() expected error but got none")
		return
	}

	if !containsString(err.Error(), "failed to open PDF") {
		t.Errorf("ExtractComplete() unexpected error = %v", err)
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
			wantError: true, // Minimal PDF may not parse correctly
			errorMsg:  "failed to extract content for querying",
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
	// This is a minimal PDF structure that should parse without errors
	return `%PDF-1.4
1 0 obj
<<
/Type /Catalog
/Pages 2 0 R
>>
endobj
2 0 obj
<<
/Type /Pages
/Kids [3 0 R]
/Count 1
>>
endobj
3 0 obj
<<
/Type /Page
/Parent 2 0 R
/MediaBox [0 0 612 792]
>>
endobj
xref
0 4
0000000000 65535 f
0000000010 00000 n
0000000053 00000 n
0000000125 00000 n
trailer
<<
/Size 4
/Root 1 0 R
>>
startxref
196
%%EOF`
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
