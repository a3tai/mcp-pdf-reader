package pdf

import (
	"testing"
)

func TestMetadataExtraction(t *testing.T) {
	service := NewExtractionService(100 * 1024 * 1024)

	tests := []struct {
		name          string
		path          string
		wantError     bool
		errorContains string
		validateFunc  func(*testing.T, *DocumentMetadata)
	}{
		{
			name:          "empty path",
			path:          "",
			wantError:     true,
			errorContains: "path cannot be empty",
		},
		{
			name:          "non-existent file",
			path:          "/non/existent/file.pdf",
			wantError:     true,
			errorContains: "file does not exist",
		},
		{
			name:          "directory instead of file",
			path:          "../../docs",
			wantError:     true,
			errorContains: "path is a directory",
		},
		{
			name:      "real PDF file - PDF14.pdf",
			path:      "../../docs/PDF14.pdf",
			wantError: false,
			validateFunc: func(t *testing.T, metadata *DocumentMetadata) {
				// Verify that our implementation returns basic metadata
				if metadata == nil {
					t.Fatal("GetMetadata() returned nil metadata")
				}

				// Our implementation should at minimum provide version and encryption status
				if metadata.Version == "" {
					t.Error("GetMetadata() Version should not be empty")
				}

				// Log the extracted metadata for inspection
				t.Logf("Extracted metadata from PDF14.pdf:")
				t.Logf("  Title: '%s'", metadata.Title)
				t.Logf("  Author: '%s'", metadata.Author)
				t.Logf("  Subject: '%s'", metadata.Subject)
				t.Logf("  Creator: '%s'", metadata.Creator)
				t.Logf("  Producer: '%s'", metadata.Producer)
				t.Logf("  CreationDate: '%s'", metadata.CreationDate)
				t.Logf("  ModificationDate: '%s'", metadata.ModificationDate)
				t.Logf("  Keywords: %v", metadata.Keywords)
				t.Logf("  PageLayout: '%s'", metadata.PageLayout)
				t.Logf("  PageMode: '%s'", metadata.PageMode)
				t.Logf("  Version: '%s'", metadata.Version)
				t.Logf("  Encrypted: %v", metadata.Encrypted)

				// Verify the basic structure is correct
				if metadata.Version != "1.4" {
					t.Errorf("Expected version '1.4', got '%s'", metadata.Version)
				}

				// Our implementation should set encrypted to false for this test file
				if metadata.Encrypted {
					t.Error("Expected Encrypted to be false for test PDF")
				}
			},
		},
		{
			name:      "real PDF file - basic form PDF",
			path:      "../../docs/test-forms/basic-form.pdf",
			wantError: false,
			validateFunc: func(t *testing.T, metadata *DocumentMetadata) {
				if metadata == nil {
					t.Fatal("GetMetadata() returned nil metadata")
				}

				t.Logf("Extracted metadata from basic-form.pdf:")
				t.Logf("  Title: '%s'", metadata.Title)
				t.Logf("  Author: '%s'", metadata.Author)
				t.Logf("  Producer: '%s'", metadata.Producer)
				t.Logf("  Version: '%s'", metadata.Version)

				// Basic validation
				if metadata.Version == "" {
					t.Error("Version should not be empty")
				}
			},
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
					return
				}
				if tt.errorContains != "" && !containsString(err.Error(), tt.errorContains) {
					t.Errorf("GetMetadata() error = %v, want error containing %v", err, tt.errorContains)
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

			// Run custom validation if provided
			if tt.validateFunc != nil {
				tt.validateFunc(t, result)
			}
		})
	}
}

func TestMetadataExtractionComparison(t *testing.T) {
	// This test compares the results from our GetMetadata implementation
	// with the results from the working Stats.GetFileStats implementation
	// to ensure they extract similar information

	service := NewExtractionService(100 * 1024 * 1024)
	stats := NewStats(100 * 1024 * 1024)

	testFile := "../../docs/PDF14.pdf"
	if !fileExists(testFile) {
		t.Skipf("Test PDF file not found: %s", testFile)
		return
	}

	// Get metadata using our new implementation
	metadata, err := service.GetMetadata(testFile)
	if err != nil {
		t.Fatalf("GetMetadata() failed: %v", err)
	}

	// Get metadata using the working stats implementation
	statsReq := PDFStatsFileRequest{Path: testFile}
	statsResult, err := stats.GetFileStats(statsReq)
	if err != nil {
		t.Fatalf("GetFileStats() failed: %v", err)
	}

	// Compare the results - they should extract similar metadata
	t.Logf("Comparison of metadata extraction:")
	t.Logf("GetMetadata() Title: '%s'", metadata.Title)
	t.Logf("GetFileStats() Title: '%s'", statsResult.Title)

	t.Logf("GetMetadata() Author: '%s'", metadata.Author)
	t.Logf("GetFileStats() Author: '%s'", statsResult.Author)

	t.Logf("GetMetadata() Subject: '%s'", metadata.Subject)
	t.Logf("GetFileStats() Subject: '%s'", statsResult.Subject)

	t.Logf("GetMetadata() Producer: '%s'", metadata.Producer)
	t.Logf("GetFileStats() Producer: '%s'", statsResult.Producer)

	// If the stats implementation extracted metadata successfully,
	// our implementation should extract the same basic fields
	if statsResult.Title != "" && metadata.Title != statsResult.Title {
		t.Errorf("Title mismatch: GetMetadata()='%s', GetFileStats()='%s'", metadata.Title, statsResult.Title)
	}

	if statsResult.Author != "" && metadata.Author != statsResult.Author {
		t.Errorf("Author mismatch: GetMetadata()='%s', GetFileStats()='%s'", metadata.Author, statsResult.Author)
	}

	if statsResult.Subject != "" && metadata.Subject != statsResult.Subject {
		t.Errorf("Subject mismatch: GetMetadata()='%s', GetFileStats()='%s'", metadata.Subject, statsResult.Subject)
	}

	if statsResult.Producer != "" && metadata.Producer != statsResult.Producer {
		t.Errorf("Producer mismatch: GetMetadata()='%s', GetFileStats()='%s'", metadata.Producer, statsResult.Producer)
	}

	// Our GetMetadata should provide additional fields that stats doesn't
	if metadata.Version == "" {
		t.Error("GetMetadata() should provide Version information")
	}

	// Creator field should be available in GetMetadata but not in stats
	t.Logf("GetMetadata() Creator: '%s' (not available in stats)", metadata.Creator)
	t.Logf("GetMetadata() Version: '%s' (not available in stats)", metadata.Version)
	t.Logf("GetMetadata() Encrypted: %v (not available in stats)", metadata.Encrypted)
}
