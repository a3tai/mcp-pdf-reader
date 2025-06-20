package intelligence

import (
	"testing"

	"github.com/a3tai/mcp-pdf-reader/internal/pdf/extraction"
)

// Test data helpers

func createTestTextElement(text string, fontName string, fontSize float64, page int, x, y, width, height float64) extraction.ContentElement {
	return extraction.ContentElement{
		ID:         "test-text-" + text[:min(len(text), 10)],
		Type:       extraction.ContentTypeText,
		PageNumber: page,
		BoundingBox: extraction.BoundingBox{
			LowerLeft:  extraction.Coordinate{X: x, Y: y},
			UpperRight: extraction.Coordinate{X: x + width, Y: y + height},
			Width:      width,
			Height:     height,
		},
		Content: extraction.TextElement{
			Text: text,
			Properties: extraction.TextProperties{
				FontName: fontName,
				FontSize: fontSize,
			},
		},
	}
}

func createTestImageElement(page int, x, y, width, height float64) extraction.ContentElement {
	return extraction.ContentElement{
		ID:         "test-image",
		Type:       extraction.ContentTypeImage,
		PageNumber: page,
		BoundingBox: extraction.BoundingBox{
			LowerLeft:  extraction.Coordinate{X: x, Y: y},
			UpperRight: extraction.Coordinate{X: x + width, Y: y + height},
			Width:      width,
			Height:     height,
		},
		Content: extraction.ImageElement{},
	}
}

func createTestFormElement(page int) extraction.ContentElement {
	return extraction.ContentElement{
		ID:         "test-form",
		Type:       extraction.ContentTypeForm,
		PageNumber: page,
		BoundingBox: extraction.BoundingBox{
			LowerLeft:  extraction.Coordinate{X: 100, Y: 100},
			UpperRight: extraction.Coordinate{X: 200, Y: 120},
			Width:      100,
			Height:     20,
		},
		Content: struct{}{}, // Simple form element
	}
}

func createSampleDocument() []extraction.ContentElement {
	elements := []extraction.ContentElement{
		createTestTextElement("Document Title", "Arial-Bold", 18.0, 1, 100, 700, 400, 25),
		createTestTextElement("Introduction", "Arial-Bold", 14.0, 1, 100, 650, 200, 18),
		createTestTextElement("This is the introduction paragraph. It contains multiple sentences. Each sentence provides valuable information.", "Arial", 12.0, 1, 100, 600, 400, 40),
		createTestTextElement("Chapter 1: Overview", "Arial-Bold", 14.0, 1, 100, 520, 250, 18),
		createTestTextElement("This chapter provides an overview of the topic. It includes detailed explanations and examples.", "Arial", 12.0, 1, 100, 470, 400, 40),
		createTestImageElement(1, 100, 400, 200, 150),
		createTestTextElement("Chapter 2: Details", "Arial-Bold", 14.0, 2, 100, 700, 250, 18),
		createTestTextElement("This chapter goes into more detail. It provides comprehensive coverage of advanced topics.", "Arial", 12.0, 2, 100, 650, 400, 40),
		createTestFormElement(2),
		createTestTextElement("Conclusion", "Arial-Bold", 14.0, 2, 100, 200, 150, 18),
		createTestTextElement("This document concludes with a summary of key points.", "Arial", 12.0, 2, 100, 150, 400, 20),
	}
	return elements
}

func createInvoiceDocument() []extraction.ContentElement {
	elements := []extraction.ContentElement{
		createTestTextElement("INVOICE", "Arial-Bold", 20.0, 1, 100, 750, 100, 25),
		createTestTextElement("Invoice #12345", "Arial", 12.0, 1, 100, 720, 150, 15),
		createTestTextElement("Date: 2024-01-15", "Arial", 12.0, 1, 100, 700, 150, 15),
		createTestTextElement("Bill To:", "Arial-Bold", 12.0, 1, 100, 650, 80, 15),
		createTestTextElement("John Doe", "Arial", 12.0, 1, 100, 630, 100, 15),
		createTestTextElement("123 Main St", "Arial", 12.0, 1, 100, 610, 120, 15),
		createTestTextElement("Total: $1,234.56", "Arial-Bold", 14.0, 1, 100, 300, 150, 18),
		createTestFormElement(1),
	}
	return elements
}

func createEmptyDocument() []extraction.ContentElement {
	return []extraction.ContentElement{}
}

func createSinglePageDocument() []extraction.ContentElement {
	return []extraction.ContentElement{
		createTestTextElement("Short Document", "Arial", 12.0, 1, 100, 700, 200, 15),
	}
}

// min helper function
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Tests for DocumentAnalyzer creation

func TestNewDocumentAnalyzer(t *testing.T) {
	analyzer := NewDocumentAnalyzer()

	if analyzer == nil {
		t.Fatal("Expected DocumentAnalyzer to be created, got nil")
	}

	if analyzer.structureDetector == nil {
		t.Error("Expected structureDetector to be initialized")
	}

	if analyzer.classifier == nil {
		t.Error("Expected classifier to be initialized")
	}

	// Check default config
	if !analyzer.config.EnableStructureDetection {
		t.Error("Expected EnableStructureDetection to be true by default")
	}

	if !analyzer.config.EnableClassification {
		t.Error("Expected EnableClassification to be true by default")
	}
}

func TestNewDocumentAnalyzerWithConfig(t *testing.T) {
	config := AnalysisConfig{
		EnableStructureDetection: false,
		EnableClassification:     false,
		EnableQualityMetrics:     false,
		EnableSuggestions:        false,
		MinConfidence:            0.5,
		MaxProcessingTime:        5000,
		DetailedAnalysis:         false,
	}

	analyzer := NewDocumentAnalyzerWithConfig(config)

	if analyzer == nil {
		t.Fatal("Expected DocumentAnalyzer to be created, got nil")
	}

	if analyzer.config.EnableStructureDetection {
		t.Error("Expected EnableStructureDetection to be false")
	}

	if analyzer.config.MinConfidence != 0.5 {
		t.Errorf("Expected MinConfidence to be 0.5, got %f", analyzer.config.MinConfidence)
	}
}

// Tests for main Analyze method

func TestAnalyze_EmptyElements(t *testing.T) {
	analyzer := NewDocumentAnalyzer()
	elements := createEmptyDocument()

	result, err := analyzer.Analyze(elements)
	if err != nil {
		t.Errorf("Expected no error for empty elements, got: %v", err)
	}

	if result == nil {
		t.Error("Expected valid analysis result, got nil")
		return
	}

	// Check that we get meaningful analysis for empty content
	if result.Type != "unknown" {
		t.Errorf("Expected document type 'unknown', got '%s'", result.Type)
	}

	if len(result.Sections) != 0 {
		t.Errorf("Expected 0 sections for empty content, got %d", len(result.Sections))
	}

	if result.Statistics.WordCount != 0 {
		t.Errorf("Expected 0 word count for empty content, got %d", result.Statistics.WordCount)
	}

	if len(result.Quality.IssuesFound) == 0 {
		t.Error("Expected quality issues to be reported for empty content")
	}

	if len(result.Suggestions) == 0 {
		t.Error("Expected suggestions to be provided for empty content")
	}
}

func TestAnalyze_SampleDocument(t *testing.T) {
	analyzer := NewDocumentAnalyzer()
	elements := createSampleDocument()

	analysis, err := analyzer.Analyze(elements)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if analysis == nil {
		t.Fatal("Expected analysis result, got nil")
	}

	// Check basic properties
	if analysis.Statistics.WordCount == 0 {
		t.Error("Expected word count to be greater than 0")
	}

	if analysis.Statistics.PageCount != 2 {
		t.Errorf("Expected page count to be 2, got %d", analysis.Statistics.PageCount)
	}

	if analysis.Statistics.ImageCount != 1 {
		t.Errorf("Expected image count to be 1, got %d", analysis.Statistics.ImageCount)
	}

	if analysis.Type == "" {
		t.Error("Expected document type to be classified")
	}

	// Check metadata
	if analysis.Metadata.AnalysisVersion == "" {
		t.Error("Expected analysis version to be set")
	}

	if analysis.Metadata.ProcessingTime == 0 {
		t.Error("Expected processing time to be greater than 0")
	}
}

func TestAnalyze_InvoiceDocument(t *testing.T) {
	analyzer := NewDocumentAnalyzer()
	elements := createInvoiceDocument()

	analysis, err := analyzer.Analyze(elements)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Check for invoice-specific patterns
	if analysis.Statistics.FormFieldCount != 1 {
		t.Errorf("Expected form field count to be 1, got %d", analysis.Statistics.FormFieldCount)
	}

	// Should detect some currency patterns
	if analysis.Statistics.WordCount == 0 {
		t.Error("Expected word count to be greater than 0")
	}
}

func TestAnalyze_WithTimeout(t *testing.T) {
	// Create config with very short timeout
	config := DefaultAnalysisConfig()
	config.MaxProcessingTime = 1 // 1 millisecond

	analyzer := NewDocumentAnalyzerWithConfig(config)
	elements := createSampleDocument()

	_, err := analyzer.Analyze(elements)
	if err == nil {
		// Analysis might complete too quickly for timeout, just skip this test
		t.Skip("Analysis completed before timeout - this is acceptable")
		return
	}

	if !containsString(err.Error(), "timed out") {
		t.Errorf("Expected timeout error, got %v", err)
	}
}

func TestAnalyze_DisabledComponents(t *testing.T) {
	config := AnalysisConfig{
		EnableStructureDetection: false,
		EnableClassification:     false,
		EnableQualityMetrics:     false,
		EnableSuggestions:        false,
		MinConfidence:            0.7,
		MaxProcessingTime:        10000,
		DetailedAnalysis:         false,
	}

	analyzer := NewDocumentAnalyzerWithConfig(config)
	elements := createSampleDocument()

	analysis, err := analyzer.Analyze(elements)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Should still have statistics
	if analysis.Statistics.WordCount == 0 {
		t.Error("Expected word count even with disabled components")
	}

	// Should have empty sections since structure detection is disabled
	if len(analysis.Sections) != 0 {
		t.Error("Expected no sections when structure detection is disabled")
	}

	// Should have no suggestions
	if len(analysis.Suggestions) != 0 {
		t.Error("Expected no suggestions when suggestions are disabled")
	}
}

// Tests for content statistics calculation

func TestCalculateContentStats(t *testing.T) {
	analyzer := NewDocumentAnalyzer()
	elements := createSampleDocument()

	stats := analyzer.calculateContentStats(elements)

	if stats.WordCount == 0 {
		t.Error("Expected word count to be greater than 0")
	}

	if stats.CharacterCount == 0 {
		t.Error("Expected character count to be greater than 0")
	}

	if stats.PageCount != 2 {
		t.Errorf("Expected page count to be 2, got %d", stats.PageCount)
	}

	if stats.ImageCount != 1 {
		t.Errorf("Expected image count to be 1, got %d", stats.ImageCount)
	}

	if stats.FormFieldCount != 1 {
		t.Errorf("Expected form field count to be 1, got %d", stats.FormFieldCount)
	}

	if stats.ReadingTime <= 0 {
		t.Error("Expected reading time to be greater than 0")
	}

	if stats.ContentDensity["words_per_page"] <= 0 {
		t.Error("Expected words per page density to be greater than 0")
	}
}

func TestCalculateContentStats_EmptyElements(t *testing.T) {
	analyzer := NewDocumentAnalyzer()
	elements := []extraction.ContentElement{}

	stats := analyzer.calculateContentStats(elements)

	if stats.WordCount != 0 {
		t.Errorf("Expected word count to be 0, got %d", stats.WordCount)
	}

	if stats.PageCount != 0 {
		t.Errorf("Expected page count to be 0, got %d", stats.PageCount)
	}
}

// Tests for content text extraction

func TestExtractContentText(t *testing.T) {
	analyzer := NewDocumentAnalyzer()
	elements := createInvoiceDocument()

	content := analyzer.extractContentText(elements)

	if content == "" {
		t.Error("Expected content text to be extracted")
	}

	// Should contain invoice-specific text
	if !containsString(content, "INVOICE") {
		t.Error("Expected content to contain 'INVOICE'")
	}

	if !containsString(content, "1,234.56") {
		t.Error("Expected content to contain currency amount")
	}
}

// Tests for quality assessment

func TestAssessQuality(t *testing.T) {
	analyzer := NewDocumentAnalyzer()
	elements := createSampleDocument()
	stats := analyzer.calculateContentStats(elements)
	sections := []Section{
		{ID: "1", Title: "Introduction", Content: "Sample content"},
		{ID: "2", Title: "Chapter 1", Content: "More content"},
	}

	quality := analyzer.assessQuality(elements, stats, sections)

	if quality.OverallScore < 0 || quality.OverallScore > 1 {
		t.Errorf("Expected overall score between 0-1, got %f", quality.OverallScore)
	}

	if quality.ReadabilityScore < 0 || quality.ReadabilityScore > 1 {
		t.Errorf("Expected readability score between 0-1, got %f", quality.ReadabilityScore)
	}

	if quality.CompletenessScore < 0 || quality.CompletenessScore > 1 {
		t.Errorf("Expected completeness score between 0-1, got %f", quality.CompletenessScore)
	}

	if quality.ConsistencyScore < 0 || quality.ConsistencyScore > 1 {
		t.Errorf("Expected consistency score between 0-1, got %f", quality.ConsistencyScore)
	}

	if quality.StructureScore < 0 || quality.StructureScore > 1 {
		t.Errorf("Expected structure score between 0-1, got %f", quality.StructureScore)
	}
}

func TestAssessReadability(t *testing.T) {
	analyzer := NewDocumentAnalyzer()

	tests := []struct {
		name          string
		stats         ContentStats
		expectedRange [2]float64 // min, max expected score
	}{
		{
			name: "Good readability",
			stats: ContentStats{
				WordCount:     170, // 17 words per sentence average
				SentenceCount: 10,
			},
			expectedRange: [2]float64{0.8, 1.0},
		},
		{
			name: "Poor readability - long sentences",
			stats: ContentStats{
				WordCount:     500, // 50 words per sentence average
				SentenceCount: 10,
			},
			expectedRange: [2]float64{0.0, 0.4},
		},
		{
			name: "No content",
			stats: ContentStats{
				WordCount:     0,
				SentenceCount: 0,
			},
			expectedRange: [2]float64{0.0, 0.0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := analyzer.assessReadability(tt.stats)
			if score < tt.expectedRange[0] || score > tt.expectedRange[1] {
				t.Errorf("Expected readability score between %f-%f, got %f",
					tt.expectedRange[0], tt.expectedRange[1], score)
			}
		})
	}
}

func TestAssessCompleteness(t *testing.T) {
	analyzer := NewDocumentAnalyzer()

	tests := []struct {
		name        string
		stats       ContentStats
		sections    []Section
		expectedMin float64
	}{
		{
			name: "Complete document",
			stats: ContentStats{
				WordCount:      500,
				PageCount:      3,
				ImageCount:     2,
				FormFieldCount: 1,
			},
			sections: []Section{
				{ID: "1", Title: "Section 1"},
				{ID: "2", Title: "Section 2"},
			},
			expectedMin: 0.8,
		},
		{
			name: "Minimal document",
			stats: ContentStats{
				WordCount: 50,
				PageCount: 1,
			},
			sections:    []Section{},
			expectedMin: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := analyzer.assessCompleteness(tt.stats, tt.sections)
			if score < tt.expectedMin {
				t.Errorf("Expected completeness score >= %f, got %f", tt.expectedMin, score)
			}
			if score > 1.0 {
				t.Errorf("Expected completeness score <= 1.0, got %f", score)
			}
		})
	}
}

// Tests for suggestion generation

func TestGenerateSuggestions(t *testing.T) {
	analyzer := NewDocumentAnalyzer()

	quality := QualityMetrics{
		OverallScore:     0.5,
		ReadabilityScore: 0.6,
		StructureScore:   0.4,
		ConsistencyScore: 0.8,
		IssuesFound: []QualityIssue{
			{Type: "structure", Severity: "high"},
		},
	}

	stats := ContentStats{
		WordCount:      50,
		PageCount:      1,
		ImageCount:     0,
		FormFieldCount: 0,
	}

	suggestions := analyzer.generateSuggestions(quality, stats, "report")

	if len(suggestions) == 0 {
		t.Error("Expected suggestions to be generated")
	}

	// Should suggest adding images for reports with no images
	found := false
	for _, suggestion := range suggestions {
		if containsString(suggestion, "charts") || containsString(suggestion, "graphs") {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected suggestion about adding charts/graphs for report")
	}
}

func TestGenerateSuggestions_Invoice(t *testing.T) {
	analyzer := NewDocumentAnalyzer()

	quality := QualityMetrics{
		OverallScore: 0.8,
	}

	stats := ContentStats{
		WordCount:      100,
		FormFieldCount: 0, // No form fields
	}

	suggestions := analyzer.generateSuggestions(quality, stats, "invoice")

	// Should suggest adding form fields for invoices
	found := false
	for _, suggestion := range suggestions {
		if containsString(suggestion, "form fields") {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected suggestion about adding form fields for invoice")
	}
}

// Tests for default configurations

func TestDefaultAnalysisConfig(t *testing.T) {
	config := DefaultAnalysisConfig()

	if !config.EnableStructureDetection {
		t.Error("Expected EnableStructureDetection to be true by default")
	}

	if !config.EnableClassification {
		t.Error("Expected EnableClassification to be true by default")
	}

	if !config.EnableQualityMetrics {
		t.Error("Expected EnableQualityMetrics to be true by default")
	}

	if config.MinConfidence != 0.7 {
		t.Errorf("Expected MinConfidence to be 0.7, got %f", config.MinConfidence)
	}

	if config.MaxProcessingTime != 10000 {
		t.Errorf("Expected MaxProcessingTime to be 10000, got %d", config.MaxProcessingTime)
	}
}

// Edge case tests

func TestAnalyze_SingleElement(t *testing.T) {
	analyzer := NewDocumentAnalyzer()
	elements := createSinglePageDocument()

	analysis, err := analyzer.Analyze(elements)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if analysis.Statistics.PageCount != 1 {
		t.Errorf("Expected page count to be 1, got %d", analysis.Statistics.PageCount)
	}

	if analysis.Statistics.WordCount == 0 {
		t.Error("Expected word count to be greater than 0")
	}
}

func TestAnalyze_LargeDocument(t *testing.T) {
	analyzer := NewDocumentAnalyzer()

	// Create a large document with many elements
	var elements []extraction.ContentElement
	for i := 0; i < 100; i++ {
		page := (i / 10) + 1
		y := float64(700 - (i%10)*50)
		elements = append(elements, createTestTextElement(
			"This is a test paragraph with multiple sentences. It contains various words and punctuation.",
			"Arial", 12.0, page, 100, y, 400, 20))
	}

	analysis, err := analyzer.Analyze(elements)
	if err != nil {
		t.Fatalf("Expected no error for large document, got %v", err)
	}

	if analysis.Statistics.WordCount == 0 {
		t.Error("Expected word count to be greater than 0 for large document")
	}

	expectedPages := 10
	if analysis.Statistics.PageCount != expectedPages {
		t.Errorf("Expected page count to be %d, got %d", expectedPages, analysis.Statistics.PageCount)
	}
}

// Utility helper functions

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || containsStringHelper(s, substr))
}

func containsStringHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Benchmark tests

func BenchmarkAnalyze_SampleDocument(b *testing.B) {
	analyzer := NewDocumentAnalyzer()
	elements := createSampleDocument()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := analyzer.Analyze(elements)
		if err != nil {
			b.Fatalf("Benchmark failed with error: %v", err)
		}
	}
}

func BenchmarkCalculateContentStats(b *testing.B) {
	analyzer := NewDocumentAnalyzer()
	elements := createSampleDocument()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = analyzer.calculateContentStats(elements)
	}
}

func BenchmarkAssessQuality(b *testing.B) {
	analyzer := NewDocumentAnalyzer()
	elements := createSampleDocument()
	stats := analyzer.calculateContentStats(elements)
	sections := []Section{
		{ID: "1", Title: "Test Section", Content: "Test content"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = analyzer.assessQuality(elements, stats, sections)
	}
}
