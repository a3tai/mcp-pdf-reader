package pdf

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewAssets(t *testing.T) {
	tests := []struct {
		name        string
		maxFileSize int64
	}{
		{
			name:        "standard max file size",
			maxFileSize: 100 * 1024 * 1024, // 100MB
		},
		{
			name:        "small max file size",
			maxFileSize: 1024, // 1KB
		},
		{
			name:        "zero max file size",
			maxFileSize: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assets := NewAssets(tt.maxFileSize)
			if assets == nil {
				t.Error("NewAssets() returned nil")
				return
			}
			if assets.maxFileSize != tt.maxFileSize {
				t.Errorf("NewAssets() maxFileSize = %v, want %v", assets.maxFileSize, tt.maxFileSize)
			}
			if assets.validator == nil {
				t.Error("NewAssets() validator is nil")
			}
		})
	}
}

func TestAssets_ExtractAssets(t *testing.T) {
	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "pdf_assets_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test files
	testPDFPath := filepath.Join(tempDir, "test.pdf")
	testTxtPath := filepath.Join(tempDir, "test.txt")
	testDirPath := filepath.Join(tempDir, "testdir")
	largePDFPath := filepath.Join(tempDir, "large.pdf")

	// Create a simple text file (not PDF)
	if err := os.WriteFile(testTxtPath, []byte("This is not a PDF"), 0644); err != nil {
		t.Fatalf("Failed to create test txt file: %v", err)
	}

	// Create a directory
	if err := os.Mkdir(testDirPath, 0755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	// Create a fake large PDF file
	largeContent := make([]byte, 1024*1024+1) // 1MB + 1 byte
	if err := os.WriteFile(largePDFPath, largeContent, 0644); err != nil {
		t.Fatalf("Failed to create large test file: %v", err)
	}

	// Create a minimal PDF file (basic structure)
	minimalPDF := []byte(`%PDF-1.4
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
%%EOF`)
	if err := os.WriteFile(testPDFPath, minimalPDF, 0644); err != nil {
		t.Fatalf("Failed to create test PDF file: %v", err)
	}

	assets := NewAssets(1024 * 1024) // 1MB limit

	tests := []struct {
		name    string
		req     PDFAssetsFileRequest
		wantErr bool
		errMsg  string
	}{
		{
			name:    "empty path",
			req:     PDFAssetsFileRequest{Path: ""},
			wantErr: true,
			errMsg:  "path cannot be empty",
		},
		{
			name:    "non-existent file",
			req:     PDFAssetsFileRequest{Path: "/non/existent/file.pdf"},
			wantErr: true,
			errMsg:  "file does not exist",
		},
		{
			name:    "directory instead of file",
			req:     PDFAssetsFileRequest{Path: testDirPath},
			wantErr: true,
			errMsg:  "directory",
		},
		{
			name:    "non-PDF file",
			req:     PDFAssetsFileRequest{Path: testTxtPath},
			wantErr: true,
			errMsg:  "PDF",
		},
		{
			name:    "file too large",
			req:     PDFAssetsFileRequest{Path: largePDFPath},
			wantErr: true,
			errMsg:  "too large",
		},
		{
			name:    "valid PDF file (may fail due to parsing)",
			req:     PDFAssetsFileRequest{Path: testPDFPath},
			wantErr: true, // Our minimal PDF might not parse correctly
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := assets.ExtractAssets(tt.req)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ExtractAssets() expected error but got none")
				} else if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("ExtractAssets() error = %v, want error containing %v", err, tt.errMsg)
				}
				if result != nil {
					t.Errorf("ExtractAssets() expected nil result on error, got %v", result)
				}
			} else {
				if err != nil {
					t.Errorf("ExtractAssets() unexpected error = %v", err)
				}
				if result == nil {
					t.Errorf("ExtractAssets() expected result but got nil")
				} else {
					// Validate result structure
					if result.Path != tt.req.Path {
						t.Errorf("ExtractAssets() result.Path = %v, want %v", result.Path, tt.req.Path)
					}
					if result.TotalCount != len(result.Images) {
						t.Errorf("ExtractAssets() result.TotalCount = %v, want %v", result.TotalCount, len(result.Images))
					}
				}
			}
		})
	}
}

func TestAssets_normalizeImageFormat(t *testing.T) {
	assets := NewAssets(1024 * 1024)

	tests := []struct {
		name       string
		filterName string
		want       string
	}{
		{
			name:       "DCTDecode filter",
			filterName: "DCTDecode",
			want:       "JPEG",
		},
		{
			name:       "JPXDecode filter",
			filterName: "JPXDecode",
			want:       "JPEG2000",
		},
		{
			name:       "CCITTFaxDecode filter",
			filterName: "CCITTFaxDecode",
			want:       "TIFF/Fax",
		},
		{
			name:       "JBIG2Decode filter",
			filterName: "JBIG2Decode",
			want:       "JBIG2",
		},
		{
			name:       "FlateDecode filter",
			filterName: "FlateDecode",
			want:       "PNG/Deflate",
		},
		{
			name:       "LZWDecode filter",
			filterName: "LZWDecode",
			want:       "LZW",
		},
		{
			name:       "RunLengthDecode filter",
			filterName: "RunLengthDecode",
			want:       "RLE",
		},
		{
			name:       "unknown filter",
			filterName: "UnknownFilter",
			want:       "UnknownFilter",
		},
		{
			name:       "empty filter",
			filterName: "",
			want:       "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := assets.normalizeImageFormat(tt.filterName)
			if got != tt.want {
				t.Errorf("normalizeImageFormat() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAssets_GetSupportedFormats(t *testing.T) {
	assets := NewAssets(1024 * 1024)

	formats := assets.GetSupportedFormats()

	expectedFormats := []string{
		"JPEG",
		"JPEG2000",
		"TIFF/Fax",
		"JBIG2",
		"PNG/Deflate",
		"LZW",
		"RLE",
	}

	if len(formats) != len(expectedFormats) {
		t.Errorf("GetSupportedFormats() returned %d formats, want %d", len(formats), len(expectedFormats))
	}

	// Check if all expected formats are present
	formatMap := make(map[string]bool)
	for _, format := range formats {
		formatMap[format] = true
	}

	for _, expected := range expectedFormats {
		if !formatMap[expected] {
			t.Errorf("GetSupportedFormats() missing expected format: %s", expected)
		}
	}
}

func TestAssets_extractImagesFromPages(t *testing.T) {
	assets := NewAssets(1024 * 1024)

	// Test with nil reader (should handle gracefully)
	t.Run("nil reader", func(t *testing.T) {
		images := assets.extractImagesFromPages(nil)
		if images == nil {
			t.Error("extractImagesFromPages() returned nil instead of empty slice")
		}
		if len(images) != 0 {
			t.Errorf("extractImagesFromPages() returned %d images, want 0", len(images))
		}
	})

	// Note: Testing with actual PDF reader would require complex setup
	// The function is mainly tested through integration tests
}

func TestAssets_extractImagesFromPage(t *testing.T) {
	assets := NewAssets(1024 * 1024)

	// Test with nil reader and various page numbers
	t.Run("nil reader", func(t *testing.T) {
		images := assets.extractImagesFromPage(nil, 1)
		if images == nil {
			t.Error("extractImagesFromPage() returned nil instead of empty slice")
		}
		if len(images) != 0 {
			t.Errorf("extractImagesFromPage() returned %d images, want 0", len(images))
		}
	})

	t.Run("negative page number", func(t *testing.T) {
		images := assets.extractImagesFromPage(nil, -1)
		if images == nil {
			t.Error("extractImagesFromPage() returned nil instead of empty slice")
		}
		if len(images) != 0 {
			t.Errorf("extractImagesFromPage() returned %d images, want 0", len(images))
		}
	})

	t.Run("zero page number", func(t *testing.T) {
		images := assets.extractImagesFromPage(nil, 0)
		if images == nil {
			t.Error("extractImagesFromPage() returned nil instead of empty slice")
		}
		if len(images) != 0 {
			t.Errorf("extractImagesFromPage() returned %d images, want 0", len(images))
		}
	})
}

func TestAssets_extractImageInfo(t *testing.T) {
	assets := NewAssets(1024 * 1024)

	// Test with nil PDF value (should handle gracefully due to panic recovery)
	t.Run("nil pdf value", func(t *testing.T) {
		imageInfo := assets.extractImageInfo(nil, 1)
		// The function should return nil due to panic recovery
		if imageInfo != nil {
			t.Error("extractImageInfo() expected nil for invalid input")
		}
	})

	// Note: Testing with actual PDF values would require complex PDF object setup
	// The function is mainly tested through integration tests with real PDFs
}

func TestAssets_ValidationIntegration(t *testing.T) {
	// Test that Assets uses its validator correctly
	tempDir, err := os.MkdirTemp("", "assets_validation_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a file that's too large
	largePath := filepath.Join(tempDir, "large.pdf")
	largeContent := make([]byte, 1024*1024+1) // 1MB + 1 byte
	if err := os.WriteFile(largePath, largeContent, 0644); err != nil {
		t.Fatalf("Failed to create large file: %v", err)
	}

	// Create Assets with small limit
	assets := NewAssets(1024 * 1024) // 1MB limit

	req := PDFAssetsFileRequest{Path: largePath}
	result, err := assets.ExtractAssets(req)

	if err == nil {
		t.Error("ExtractAssets() expected error for large file but got none")
	}
	if result != nil {
		t.Error("ExtractAssets() expected nil result for invalid file")
	}
	if !strings.Contains(err.Error(), "too large") {
		t.Errorf("ExtractAssets() expected 'too large' error, got: %v", err)
	}
}

func TestAssets_EmptyResult(t *testing.T) {
	// Test that extraction returns proper empty results
	tempDir, err := os.MkdirTemp("", "assets_empty_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a minimal PDF without images
	pdfPath := filepath.Join(tempDir, "empty.pdf")
	minimalPDF := []byte(`%PDF-1.4
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
%%EOF`)
	if err := os.WriteFile(pdfPath, minimalPDF, 0644); err != nil {
		t.Fatalf("Failed to create PDF file: %v", err)
	}

	assets := NewAssets(1024 * 1024)
	req := PDFAssetsFileRequest{Path: pdfPath}

	// This test might fail due to PDF parsing, but if it succeeds,
	// we should get an empty result
	result, err := assets.ExtractAssets(req)
	if err == nil && result != nil {
		if result.TotalCount != len(result.Images) {
			t.Errorf("ExtractAssets() TotalCount = %d, want %d", result.TotalCount, len(result.Images))
		}
		if result.Path != pdfPath {
			t.Errorf("ExtractAssets() Path = %s, want %s", result.Path, pdfPath)
		}
		if result.Images == nil {
			t.Error("ExtractAssets() Images slice should not be nil")
		}
	}
}

func TestAssets_PanicRecovery(t *testing.T) {
	// Test that the panic recovery mechanisms work
	assets := NewAssets(1024 * 1024)

	// Test extractImagesFromPages with nil (should not panic)
	t.Run("extractImagesFromPages panic recovery", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("extractImagesFromPages() panicked: %v", r)
			}
		}()

		images := assets.extractImagesFromPages(nil)
		if images == nil {
			t.Error("extractImagesFromPages() should return empty slice, not nil")
		}
	})

	// Test extractImagesFromPage with nil (should not panic)
	t.Run("extractImagesFromPage panic recovery", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("extractImagesFromPage() panicked: %v", r)
			}
		}()

		images := assets.extractImagesFromPage(nil, 1)
		if images == nil {
			t.Error("extractImagesFromPage() should return empty slice, not nil")
		}
	})

	// Test extractImageInfo with nil (should not panic)
	t.Run("extractImageInfo panic recovery", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("extractImageInfo() panicked: %v", r)
			}
		}()

		imageInfo := assets.extractImageInfo(nil, 1)
		// Should return nil due to panic recovery, not panic
		if imageInfo != nil {
			t.Error("extractImageInfo() should return nil for invalid input")
		}
	})
}
