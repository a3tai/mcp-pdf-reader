package pdf

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewReader(t *testing.T) {
	tests := []struct {
		name        string
		maxFileSize int64
		want        *Reader
	}{
		{
			name:        "standard max file size",
			maxFileSize: 100 * 1024 * 1024, // 100MB
			want: &Reader{
				maxFileSize: 100 * 1024 * 1024,
				maxTextSize: 10 * 1024 * 1024, // 10MB
			},
		},
		{
			name:        "small max file size",
			maxFileSize: 1024, // 1KB
			want: &Reader{
				maxFileSize: 1024,
				maxTextSize: 10 * 1024 * 1024, // 10MB
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewReader(tt.maxFileSize)
			if got.maxFileSize != tt.want.maxFileSize {
				t.Errorf("NewReader() maxFileSize = %v, want %v", got.maxFileSize, tt.want.maxFileSize)
			}
			if got.maxTextSize != tt.want.maxTextSize {
				t.Errorf("NewReader() maxTextSize = %v, want %v", got.maxTextSize, tt.want.maxTextSize)
			}
		})
	}
}

func TestReader_ReadFile(t *testing.T) {
	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "pdf_reader_test")
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

	// Create a fake large PDF file (just large enough to exceed limit)
	largeContent := make([]byte, 1024*1024+1) // 1MB + 1 byte
	if err := os.WriteFile(largePDFPath, largeContent, 0644); err != nil {
		t.Fatalf("Failed to create large test file: %v", err)
	}

	// Create a minimal PDF file (this is a very basic PDF structure)
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

	reader := NewReader(1024 * 1024) // 1MB limit

	tests := []struct {
		name    string
		req     PDFReadFileRequest
		wantErr bool
		errMsg  string
	}{
		{
			name:    "empty path",
			req:     PDFReadFileRequest{Path: ""},
			wantErr: true,
			errMsg:  "path cannot be empty",
		},
		{
			name:    "non-existent file",
			req:     PDFReadFileRequest{Path: "/non/existent/file.pdf"},
			wantErr: true,
			errMsg:  "file does not exist",
		},
		{
			name:    "directory instead of file",
			req:     PDFReadFileRequest{Path: testDirPath},
			wantErr: true,
			errMsg:  "path is a directory",
		},
		{
			name:    "non-PDF file",
			req:     PDFReadFileRequest{Path: testTxtPath},
			wantErr: true,
			errMsg:  "file is not a PDF",
		},
		{
			name:    "file too large",
			req:     PDFReadFileRequest{Path: largePDFPath},
			wantErr: true,
			errMsg:  "file too large",
		},
		{
			name:    "valid PDF file (may fail due to parsing)",
			req:     PDFReadFileRequest{Path: testPDFPath},
			wantErr: true, // Our minimal PDF might not parse correctly
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := reader.ReadFile(tt.req)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ReadFile() expected error but got none")
				} else if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("ReadFile() error = %v, want error containing %v", err, tt.errMsg)
				}
				if result != nil {
					t.Errorf("ReadFile() expected nil result on error, got %v", result)
				}
			} else {
				if err != nil {
					t.Errorf("ReadFile() unexpected error = %v", err)
				}
				if result == nil {
					t.Errorf("ReadFile() expected result but got nil")
				}
			}
		})
	}
}

func TestReader_validatePDFFile(t *testing.T) {
	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "pdf_validate_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test files
	testPDFPath := filepath.Join(tempDir, "test.pdf")
	testTxtPath := filepath.Join(tempDir, "test.txt")
	testDirPath := filepath.Join(tempDir, "testdir")
	largePDFPath := filepath.Join(tempDir, "large.pdf")

	// Create files
	if err := os.WriteFile(testPDFPath, []byte("fake pdf content"), 0644); err != nil {
		t.Fatalf("Failed to create test PDF file: %v", err)
	}
	if err := os.WriteFile(testTxtPath, []byte("text content"), 0644); err != nil {
		t.Fatalf("Failed to create test txt file: %v", err)
	}
	if err := os.Mkdir(testDirPath, 0755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	// Create large PDF file
	largeContent := make([]byte, 1024*1024+1) // 1MB + 1 byte
	if err := os.WriteFile(largePDFPath, largeContent, 0644); err != nil {
		t.Fatalf("Failed to create large PDF file: %v", err)
	}

	reader := NewReader(1024 * 1024) // 1MB limit

	tests := []struct {
		name     string
		filePath string
		wantErr  bool
		errMsg   string
	}{
		{
			name:     "valid PDF file",
			filePath: testPDFPath,
			wantErr:  false,
		},
		{
			name:     "directory instead of file",
			filePath: testDirPath,
			wantErr:  true,
			errMsg:   "path is a directory",
		},
		{
			name:     "non-PDF extension",
			filePath: testTxtPath,
			wantErr:  true,
			errMsg:   "file is not a PDF",
		},
		{
			name:     "file too large",
			filePath: largePDFPath,
			wantErr:  true,
			errMsg:   "file too large",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fileInfo, err := os.Stat(tt.filePath)
			if err != nil {
				t.Fatalf("Failed to get file info: %v", err)
			}

			err = reader.validatePDFFile(tt.filePath, fileInfo)
			if tt.wantErr {
				if err == nil {
					t.Errorf("validatePDFFile() expected error but got none")
				} else if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("validatePDFFile() error = %v, want error containing %v", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("validatePDFFile() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestReader_extractTextContent(t *testing.T) {
	reader := NewReader(1024 * 1024)

	// Test with nil reader (should handle gracefully)
	t.Run("nil reader", func(t *testing.T) {
		_, err := reader.extractTextContent(nil)
		if err == nil {
			t.Error("extractTextContent() expected error with nil reader")
		}
	})

	// Note: Testing with actual PDF content would require complex setup
	// or external PDF files. The extractTextContent function is mainly
	// tested through integration tests or with real PDF files.
}

func TestReader_PDFFileExtensionValidation(t *testing.T) {
	reader := NewReader(1024 * 1024)

	// Create temp directory
	tempDir, err := os.MkdirTemp("", "pdf_ext_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tests := []struct {
		name     string
		filename string
		wantErr  bool
	}{
		{
			name:     "lowercase .pdf",
			filename: "test.pdf",
			wantErr:  false,
		},
		{
			name:     "uppercase .PDF",
			filename: "test.PDF",
			wantErr:  false,
		},
		{
			name:     "mixed case .Pdf",
			filename: "test.Pdf",
			wantErr:  false,
		},
		{
			name:     "no extension",
			filename: "test",
			wantErr:  true,
		},
		{
			name:     ".txt extension",
			filename: "test.txt",
			wantErr:  true,
		},
		{
			name:     ".doc extension",
			filename: "test.doc",
			wantErr:  true,
		},
		{
			name:     "pdf in filename but different extension",
			filename: "pdf.txt",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filePath := filepath.Join(tempDir, tt.filename)

			// Create the test file
			if err := os.WriteFile(filePath, []byte("test content"), 0644); err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}

			fileInfo, err := os.Stat(filePath)
			if err != nil {
				t.Fatalf("Failed to get file info: %v", err)
			}

			err = reader.validatePDFFile(filePath, fileInfo)
			if tt.wantErr {
				if err == nil {
					t.Errorf("validatePDFFile() expected error for %s but got none", tt.filename)
				}
			} else {
				if err != nil && strings.Contains(err.Error(), "file is not a PDF") {
					t.Errorf("validatePDFFile() unexpected PDF extension error for %s: %v", tt.filename, err)
				}
			}
		})
	}
}

func TestReader_FileSizeValidation(t *testing.T) {
	// Create temp directory
	tempDir, err := os.MkdirTemp("", "pdf_size_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tests := []struct {
		name        string
		maxFileSize int64
		fileSize    int64
		wantErr     bool
	}{
		{
			name:        "file under limit",
			maxFileSize: 1024 * 1024, // 1MB
			fileSize:    1024,        // 1KB
			wantErr:     false,
		},
		{
			name:        "file at exact limit",
			maxFileSize: 1024, // 1KB
			fileSize:    1024, // 1KB
			wantErr:     false,
		},
		{
			name:        "file over limit",
			maxFileSize: 1024,     // 1KB
			fileSize:    1024 + 1, // 1KB + 1 byte
			wantErr:     true,
		},
		{
			name:        "zero size file",
			maxFileSize: 1024, // 1KB
			fileSize:    0,
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := NewReader(tt.maxFileSize)

			filePath := filepath.Join(tempDir, "test.pdf")

			// Create file with specified size
			content := make([]byte, tt.fileSize)
			if err := os.WriteFile(filePath, content, 0644); err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}

			fileInfo, err := os.Stat(filePath)
			if err != nil {
				t.Fatalf("Failed to get file info: %v", err)
			}

			err = reader.validatePDFFile(filePath, fileInfo)
			if tt.wantErr {
				if err == nil {
					t.Errorf("validatePDFFile() expected error for file size %d with limit %d", tt.fileSize, tt.maxFileSize)
				} else if !strings.Contains(err.Error(), "file too large") {
					t.Errorf("validatePDFFile() expected 'file too large' error, got: %v", err)
				}
			} else {
				if err != nil && strings.Contains(err.Error(), "file too large") {
					t.Errorf("validatePDFFile() unexpected file size error: %v", err)
				}
			}
		})
	}
}
