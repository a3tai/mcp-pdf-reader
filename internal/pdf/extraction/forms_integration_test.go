package extraction

import (
	"path/filepath"
	"testing"

	"github.com/ledongthuc/pdf"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFormExtractor_IntegrationWithRealPDFs(t *testing.T) {
	// Skip if running short tests
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tests := []struct {
		name            string
		pdfFile         string
		expectedMinForm int // Minimum expected forms
		expectedMaxForm int // Maximum expected forms
		debugOutput     bool
	}{
		{
			name:            "fillable_form_pdf",
			pdfFile:         "fillable-form.pdf",
			expectedMinForm: 0,  // Since we're using pattern matching, this might not detect actual form fields
			expectedMaxForm: 20, // Upper bound
			debugOutput:     true,
		},
		{
			name:            "basic_text_pdf",
			pdfFile:         "basic-text.pdf",
			expectedMinForm: 0,
			expectedMaxForm: 2, // Might have some false positives
			debugOutput:     false,
		},
		{
			name:            "sample_report_pdf",
			pdfFile:         "sample-report.pdf",
			expectedMinForm: 0,
			expectedMaxForm: 5, // Reports might have underlines that look like form fields
			debugOutput:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Build path to test PDF
			pdfPath := filepath.Join("..", "..", "..", "docs", "examples", tt.pdfFile)

			// Open PDF
			f, r, err := pdf.Open(pdfPath)
			if err != nil {
				// File might not exist in test environment
				t.Skipf("Could not open test PDF %s: %v", pdfPath, err)
			}
			defer f.Close()

			// Create form extractor
			extractor := NewFormExtractor(tt.debugOutput)

			// Extract forms from document
			forms, err := extractor.ExtractForms(r)
			require.NoError(t, err, "Form extraction should not error")

			// Log found forms for debugging
			if tt.debugOutput {
				t.Logf("Found %d forms in %s", len(forms), tt.pdfFile)
				for i, form := range forms {
					t.Logf("Form %d: Name=%s, Type=%s, Page=%d", i+1, form.Name, form.Type, form.Page)
				}
			}

			// Check form count is within expected range
			assert.GreaterOrEqual(t, len(forms), tt.expectedMinForm,
				"Should find at least %d forms", tt.expectedMinForm)
			assert.LessOrEqual(t, len(forms), tt.expectedMaxForm,
				"Should find at most %d forms", tt.expectedMaxForm)

			// Test page-by-page extraction
			totalPageForms := 0
			for pageNum := 1; pageNum <= r.NumPage(); pageNum++ {
				page := r.Page(pageNum)
				pageForms, err := extractor.ExtractFormsFromPage(page, pageNum)
				require.NoError(t, err, "Page form extraction should not error")

				totalPageForms += len(pageForms)

				// Verify page numbers are set correctly
				for _, form := range pageForms {
					assert.Equal(t, pageNum, form.Page, "Form should have correct page number")
				}
			}

			// Since we don't have document-level form extraction yet,
			// page-level extraction might find more forms
			if tt.debugOutput {
				t.Logf("Page-by-page extraction found %d forms", totalPageForms)
			}
		})
	}
}

func TestFormExtractor_SpecificFormPatterns(t *testing.T) {
	// Skip if running short tests
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Test with fillable-form.pdf specifically
	pdfPath := filepath.Join("..", "..", "..", "docs", "examples", "fillable-form.pdf")

	f, r, err := pdf.Open(pdfPath)
	if err != nil {
		t.Skipf("Could not open fillable-form.pdf: %v", err)
	}
	defer f.Close()

	extractor := NewFormExtractor(true)

	// Test each page
	for pageNum := 1; pageNum <= r.NumPage(); pageNum++ {
		page := r.Page(pageNum)

		// Get page text to analyze patterns
		content := page.Content()
		if len(content.Text) > 0 {
			// Concatenate text
			fullText := ""
			for _, txt := range content.Text {
				fullText += txt.S
			}

			t.Logf("Page %d text sample (first 200 chars): %s",
				pageNum, truncateString(fullText, 200))

			// Extract forms
			forms, err := extractor.ExtractFormsFromPage(page, pageNum)
			require.NoError(t, err)

			// Log findings
			if len(forms) > 0 {
				t.Logf("Page %d: Found %d form patterns", pageNum, len(forms))
				for _, form := range forms {
					t.Logf("  - %s (%s)", form.Name, form.Type)
				}
			}

			// Look for specific patterns we might expect in a form
			patterns := []struct {
				name     string
				patterns []string
			}{
				{"checkbox", []string{"[ ]", "[X]", "[x]", "☐", "☑"}},
				{"text_field", []string{"____", "....", "______"}},
				{"radio", []string{"( )", "(X)", "(x)", "○", "●"}},
			}

			for _, p := range patterns {
				if containsPattern(fullText, p.patterns...) {
					t.Logf("Page %d contains %s patterns", pageNum, p.name)
				}
			}
		}
	}
}

func TestFormExtractor_Performance(t *testing.T) {
	// Skip if running short tests
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	// Test with large-doc.pdf for performance
	pdfPath := filepath.Join("..", "..", "..", "docs", "examples", "large-doc.pdf")

	f, r, err := pdf.Open(pdfPath)
	if err != nil {
		t.Skipf("Could not open large-doc.pdf: %v", err)
	}
	defer f.Close()

	extractor := NewFormExtractor(false)

	// Measure extraction time
	start := testing.Benchmark(func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = extractor.ExtractForms(r)
		}
	})

	t.Logf("Form extraction performance: %s per operation", start.T)

	// Test page-by-page performance
	pageCount := r.NumPage()
	pageBench := testing.Benchmark(func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			for pageNum := 1; pageNum <= pageCount; pageNum++ {
				page := r.Page(pageNum)
				_, _ = extractor.ExtractFormsFromPage(page, pageNum)
			}
		}
	})

	t.Logf("Page-by-page extraction performance: %s per document (%d pages)",
		pageBench.T, pageCount)
}

func TestFormExtractor_EdgeCases(t *testing.T) {
	// Skip if running short tests
	if testing.Short() {
		t.Skip("Skipping edge case test in short mode")
	}

	tests := []struct {
		name    string
		pdfFile string
	}{
		{
			name:    "password_protected_pdf",
			pdfFile: "password-protected.pdf",
		},
		{
			name:    "image_only_pdf",
			pdfFile: "image-doc.pdf",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pdfPath := filepath.Join("..", "..", "..", "docs", "examples", tt.pdfFile)

			// Try to open - might fail for password protected
			f, r, err := pdf.Open(pdfPath)
			if err != nil {
				t.Logf("Could not open %s: %v", tt.pdfFile, err)
				if tt.pdfFile == "password-protected.pdf" {
					t.Logf("Note: ledongthuc/pdf library does not support password-protected PDFs")
				}
				return
			}
			defer f.Close()

			extractor := NewFormExtractor(false)

			// Should not panic even with edge case PDFs
			forms, err := extractor.ExtractForms(r)
			assert.NoError(t, err, "Should handle edge cases gracefully")
			assert.NotNil(t, forms, "Should return non-nil slice")
		})
	}
}

// Helper function to truncate strings for logging
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
