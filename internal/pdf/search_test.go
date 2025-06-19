package pdf

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestSearch_SearchDirectory(t *testing.T) {
	search := NewSearch(1024 * 1024) // 1MB limit

	// Create a temporary directory with test files
	tempDir, err := os.MkdirTemp("", "pdf_search_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test files
	testFiles := map[string][]byte{
		"document1.pdf":        make([]byte, 1024),
		"research_paper.pdf":   make([]byte, 2048),
		"machine_learning.pdf": make([]byte, 512),
		"report.txt":           []byte("not a pdf"),
		"empty.pdf":            {},                        // Empty file
		"large.pdf":            make([]byte, 2*1024*1024), // Too large
	}

	for filename, content := range testFiles {
		filePath := filepath.Join(tempDir, filename)
		if err := os.WriteFile(filePath, content, 0o644); err != nil {
			t.Fatalf("failed to create test file %s: %v", filename, err)
		}
	}

	tests := []struct {
		name           string
		req            PDFSearchDirectoryRequest
		expectedCount  int
		expectedError  bool
		validateResult func(*testing.T, *PDFSearchDirectoryResult)
	}{
		{
			name: "search all PDFs",
			req: PDFSearchDirectoryRequest{
				Directory: tempDir,
				Query:     "",
			},
			expectedCount: 3, // Only valid PDFs: document1, research_paper, machine_learning
			expectedError: false,
			validateResult: func(t *testing.T, result *PDFSearchDirectoryResult) {
				if result.Directory != tempDir {
					t.Errorf("expected directory %s but got %s", tempDir, result.Directory)
				}
				if result.SearchQuery != "" {
					t.Errorf("expected empty search query but got %s", result.SearchQuery)
				}
			},
		},
		{
			name: "search with query 'machine'",
			req: PDFSearchDirectoryRequest{
				Directory: tempDir,
				Query:     "machine",
			},
			expectedCount: 1, // Only machine_learning.pdf
			expectedError: false,
			validateResult: func(t *testing.T, result *PDFSearchDirectoryResult) {
				if result.SearchQuery != "machine" {
					t.Errorf("expected search query 'machine' but got %s", result.SearchQuery)
				}
				if len(result.Files) > 0 && result.Files[0].Name != "machine_learning.pdf" {
					t.Errorf("expected machine_learning.pdf but got %s", result.Files[0].Name)
				}
			},
		},
		{
			name: "search with query 'research'",
			req: PDFSearchDirectoryRequest{
				Directory: tempDir,
				Query:     "research",
			},
			expectedCount: 1, // Only research_paper.pdf
			expectedError: false,
		},
		{
			name: "search with non-matching query",
			req: PDFSearchDirectoryRequest{
				Directory: tempDir,
				Query:     "nonexistent",
			},
			expectedCount: 0,
			expectedError: false,
		},
		{
			name: "empty directory path",
			req: PDFSearchDirectoryRequest{
				Directory: "",
				Query:     "",
			},
			expectedError: true,
		},
		{
			name: "non-existent directory",
			req: PDFSearchDirectoryRequest{
				Directory: "/non/existent/path",
				Query:     "",
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := search.SearchDirectory(tt.req)

			if tt.expectedError && err == nil {
				t.Errorf("expected error but got none")
			}
			if !tt.expectedError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if !tt.expectedError {
				if result == nil {
					t.Fatalf("result should not be nil")
				}

				if result.TotalCount != tt.expectedCount {
					t.Errorf("expected %d files but got %d", tt.expectedCount, result.TotalCount)
				}

				if len(result.Files) != tt.expectedCount {
					t.Errorf("expected %d files in slice but got %d", tt.expectedCount, len(result.Files))
				}

				// Validate all returned files are PDFs
				for _, file := range result.Files {
					if !search.isPDFFile(file.Name) {
						t.Errorf("non-PDF file returned: %s", file.Name)
					}
					if file.Path == "" {
						t.Errorf("file path is empty for %s", file.Name)
					}
					if file.Size <= 0 {
						t.Errorf("invalid file size for %s: %d", file.Name, file.Size)
					}
					if file.ModifiedTime == "" {
						t.Errorf("modified time is empty for %s", file.Name)
					}
				}

				if tt.validateResult != nil {
					tt.validateResult(t, result)
				}
			}
		})
	}
}

func TestSearch_FindPDFsInDirectory(t *testing.T) {
	search := NewSearch(1024 * 1024)

	// Create temp directory with test files
	tempDir, err := os.MkdirTemp("", "pdf_find_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test files
	testFiles := []string{"doc1.pdf", "doc2.pdf", "notpdf.txt"}
	for _, filename := range testFiles {
		filePath := filepath.Join(tempDir, filename)
		content := make([]byte, 1024)
		if err := os.WriteFile(filePath, content, 0o644); err != nil {
			t.Fatalf("failed to create test file %s: %v", filename, err)
		}
	}

	files, err := search.FindPDFsInDirectory(tempDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedCount := 2 // Only PDF files
	if len(files) != expectedCount {
		t.Errorf("expected %d files but got %d", expectedCount, len(files))
	}

	// Verify all returned files are PDFs
	for _, file := range files {
		if !search.isPDFFile(file.Name) {
			t.Errorf("non-PDF file returned: %s", file.Name)
		}
	}
}

func TestSearch_CountPDFsInDirectory(t *testing.T) {
	search := NewSearch(1024 * 1024)

	// Create temp directory with test files
	tempDir, err := os.MkdirTemp("", "pdf_count_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test files
	pdfFiles := []string{"doc1.pdf", "doc2.pdf", "doc3.pdf"}
	nonPdfFiles := []string{"doc.txt", "image.jpg", "data.csv"}

	for _, filename := range append(pdfFiles, nonPdfFiles...) {
		filePath := filepath.Join(tempDir, filename)
		content := make([]byte, 1024)
		if err := os.WriteFile(filePath, content, 0o644); err != nil {
			t.Fatalf("failed to create test file %s: %v", filename, err)
		}
	}

	count, err := search.CountPDFsInDirectory(tempDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedCount := len(pdfFiles)
	if count != expectedCount {
		t.Errorf("expected count %d but got %d", expectedCount, count)
	}
}

func TestSearch_SearchByPattern(t *testing.T) {
	search := NewSearch(1024 * 1024)

	// Create temp directory with test files
	tempDir, err := os.MkdirTemp("", "pdf_pattern_test")
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
		content := make([]byte, 1024)
		if err := os.WriteFile(filePath, content, 0o644); err != nil {
			t.Fatalf("failed to create test file %s: %v", filename, err)
		}
	}

	tests := []struct {
		name          string
		pattern       string
		expectedCount int
		expectedFiles []string
	}{
		{
			name:          "match all report files",
			pattern:       "report_*.pdf",
			expectedCount: 2,
			expectedFiles: []string{"report_2023.pdf", "report_2024.pdf"},
		},
		{
			name:          "match 2023 files",
			pattern:       "*_2023.pdf",
			expectedCount: 2,
			expectedFiles: []string{"report_2023.pdf", "summary_2023.pdf"},
		},
		{
			name:          "match all PDFs",
			pattern:       "*.pdf",
			expectedCount: 4,
		},
		{
			name:          "no matches",
			pattern:       "nonexistent_*.pdf",
			expectedCount: 0,
		},
		{
			name:          "empty pattern matches all",
			pattern:       "",
			expectedCount: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := search.SearchByPattern(tempDir, tt.pattern)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result.TotalCount != tt.expectedCount {
				t.Errorf("expected %d files but got %d", tt.expectedCount, result.TotalCount)
			}

			if len(result.Files) != tt.expectedCount {
				t.Errorf("expected %d files in slice but got %d", tt.expectedCount, len(result.Files))
			}

			// Verify expected files are present
			if tt.expectedFiles != nil {
				fileNames := make(map[string]bool)
				for _, file := range result.Files {
					fileNames[file.Name] = true
				}

				for _, expectedFile := range tt.expectedFiles {
					if !fileNames[expectedFile] {
						t.Errorf("expected file %s not found in results", expectedFile)
					}
				}
			}
		})
	}
}

func TestSearch_isPDFFile(t *testing.T) {
	search := NewSearch(1024 * 1024)

	tests := []struct {
		filename string
		expected bool
	}{
		{"document.pdf", true},
		{"DOCUMENT.PDF", true},
		{"Document.Pdf", true},
		{"file.PDF", true},
		{"document.txt", false},
		{"document.doc", false},
		{"document", false},
		{"pdf", false},
		{"", false},
		{"document.pdf.txt", false},
		{".pdf", true},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			result := search.isPDFFile(tt.filename)
			if result != tt.expected {
				t.Errorf("isPDFFile(%s) = %v, expected %v", tt.filename, result, tt.expected)
			}
		})
	}
}

func TestSearch_matchesQuery(t *testing.T) {
	search := NewSearch(1024 * 1024)

	tests := []struct {
		filename string
		query    string
		expected bool
	}{
		// Exact matches
		{"machine_learning.pdf", "machine", true},
		{"research_paper.pdf", "research", true},
		{"document.pdf", "document", true},

		// Case insensitive
		{"Machine_Learning.pdf", "machine", true},
		{"RESEARCH_PAPER.pdf", "research", true},

		// Partial matches
		{"machine_learning_guide.pdf", "learning", true},
		{"artificial_intelligence.pdf", "intel", true},

		// Word-based matching
		{"deep_learning_tutorial.pdf", "deep tutorial", true},
		{"machine_learning_basics.pdf", "machine basics", true},

		// No matches
		{"document.pdf", "nonexistent", false},
		{"research.pdf", "machine", false},

		// Empty query matches everything
		{"anything.pdf", "", true},

		// Special characters
		{"report-2023.pdf", "2023", true},
		{"summary (final).pdf", "final", true},
		{"data[backup].pdf", "backup", true},
	}

	for _, tt := range tests {
		t.Run(tt.filename+"_"+tt.query, func(t *testing.T) {
			result := search.matchesQuery(tt.filename, tt.query)
			if result != tt.expected {
				t.Errorf("matchesQuery(%s, %s) = %v, expected %v",
					tt.filename, tt.query, result, tt.expected)
			}
		})
	}
}

func TestSearch_splitIntoWords(t *testing.T) {
	search := NewSearch(1024 * 1024)

	tests := []struct {
		input    string
		expected []string
	}{
		{
			"machine_learning",
			[]string{"machine", "learning"},
		},
		{
			"deep-learning-guide",
			[]string{"deep", "learning", "guide"},
		},
		{
			"AI.research.paper",
			[]string{"ai", "research", "paper"},
		},
		{
			"document (final)",
			[]string{"document", "final"},
		},
		{
			"data[backup]",
			[]string{"data", "backup"},
		},
		{
			"simple",
			[]string{"simple"},
		},
		{
			"",
			[]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := search.splitIntoWords(tt.input)

			if len(result) != len(tt.expected) {
				t.Errorf("expected %d words but got %d", len(tt.expected), len(result))
				return
			}

			for i, word := range result {
				if word != tt.expected[i] {
					t.Errorf("word %d: expected %s but got %s", i, tt.expected[i], word)
				}
			}
		})
	}
}

func TestNewSearch(t *testing.T) {
	maxFileSize := int64(2 * 1024 * 1024) // 2MB
	search := NewSearch(maxFileSize)

	if search == nil {
		t.Fatal("NewSearch returned nil")
	}

	if search.maxFileSize != maxFileSize {
		t.Errorf("expected maxFileSize=%d but got %d", maxFileSize, search.maxFileSize)
	}

	if search.validator == nil {
		t.Error("validator should not be nil")
	}
}

func BenchmarkSearch_SearchDirectory(b *testing.B) {
	search := NewSearch(1024 * 1024)

	// Create temp directory with many files
	tempDir, err := os.MkdirTemp("", "pdf_search_bench")
	if err != nil {
		b.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create 100 test PDF files
	for i := 0; i < 100; i++ {
		filename := filepath.Join(tempDir, fmt.Sprintf("document_%03d.pdf", i))
		content := make([]byte, 1024)
		if err := os.WriteFile(filename, content, 0o644); err != nil {
			b.Fatalf("failed to create test file: %v", err)
		}
	}

	req := PDFSearchDirectoryRequest{
		Directory: tempDir,
		Query:     "",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := search.SearchDirectory(req)
		if err != nil {
			b.Fatalf("search failed: %v", err)
		}
	}
}

func BenchmarkSearch_matchesQuery(b *testing.B) {
	search := NewSearch(1024 * 1024)
	filename := "machine_learning_deep_neural_networks_tutorial.pdf"
	query := "machine learning"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		search.matchesQuery(filename, query)
	}
}
