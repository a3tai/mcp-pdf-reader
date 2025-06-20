package intelligence

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/a3tai/mcp-pdf-reader/internal/pdf/extraction"
)

// DocumentAnalyzer orchestrates all document analysis components
type DocumentAnalyzer struct {
	structureDetector *StructureDetector
	classifier        *DocumentClassifier
	config            AnalysisConfig
}

// AnalysisConfig configures document analysis behavior
type AnalysisConfig struct {
	EnableStructureDetection bool    `json:"enable_structure_detection"`
	EnableClassification     bool    `json:"enable_classification"`
	EnableQualityMetrics     bool    `json:"enable_quality_metrics"`
	EnableSuggestions        bool    `json:"enable_suggestions"`
	MinConfidence            float64 `json:"min_confidence"`
	MaxProcessingTime        int     `json:"max_processing_time_ms"`
	DetailedAnalysis         bool    `json:"detailed_analysis"`
}

// DocumentAnalysis represents comprehensive analysis results
type DocumentAnalysis struct {
	Type        string           `json:"type"`
	Sections    []Section        `json:"sections"`
	Statistics  ContentStats     `json:"statistics"`
	Quality     QualityMetrics   `json:"quality"`
	Suggestions []string         `json:"suggestions"`
	Metadata    AnalysisMetadata `json:"metadata"`
}

// Section represents a document section
type Section struct {
	ID          string                      `json:"id"`
	Title       string                      `json:"title"`
	Level       int                         `json:"level"`
	Content     string                      `json:"content"`
	Type        string                      `json:"type"`
	PageStart   int                         `json:"page_start"`
	PageEnd     int                         `json:"page_end"`
	Elements    []extraction.ContentElement `json:"elements"`
	Subsections []Section                   `json:"subsections,omitempty"`
}

// ContentStats provides statistical information about document content
type ContentStats struct {
	WordCount      int                `json:"word_count"`
	CharacterCount int                `json:"character_count"`
	ParagraphCount int                `json:"paragraph_count"`
	SentenceCount  int                `json:"sentence_count"`
	PageCount      int                `json:"page_count"`
	TableCount     int                `json:"table_count"`
	ImageCount     int                `json:"image_count"`
	FormFieldCount int                `json:"form_field_count"`
	HeaderCount    int                `json:"header_count"`
	ListCount      int                `json:"list_count"`
	FootnoteCount  int                `json:"footnote_count"`
	LinkCount      int                `json:"link_count"`
	LanguageStats  map[string]int     `json:"language_stats,omitempty"`
	ReadingTime    float64            `json:"reading_time_minutes"`
	ContentDensity map[string]float64 `json:"content_density"`
}

// QualityMetrics provides quality assessment metrics
type QualityMetrics struct {
	OverallScore       float64            `json:"overall_score"`
	ReadabilityScore   float64            `json:"readability_score"`
	CompletenessScore  float64            `json:"completeness_score"`
	ConsistencyScore   float64            `json:"consistency_score"`
	AccessibilityScore float64            `json:"accessibility_score"`
	StructureScore     float64            `json:"structure_score"`
	DetailedMetrics    map[string]float64 `json:"detailed_metrics"`
	IssuesFound        []QualityIssue     `json:"issues_found"`
	PositiveIndicators []string           `json:"positive_indicators"`
}

// QualityIssue represents a quality issue found in the document
type QualityIssue struct {
	Type        string  `json:"type"`
	Severity    string  `json:"severity"`
	Description string  `json:"description"`
	Location    string  `json:"location"`
	Suggestion  string  `json:"suggestion"`
	Confidence  float64 `json:"confidence"`
}

// AnalysisMetadata provides metadata about the analysis process
type AnalysisMetadata struct {
	AnalysisVersion   string            `json:"analysis_version"`
	ProcessingTime    time.Duration     `json:"processing_time"`
	AnalyzedAt        time.Time         `json:"analyzed_at"`
	ComponentsUsed    []string          `json:"components_used"`
	ConfigurationUsed AnalysisConfig    `json:"configuration_used"`
	DocumentHash      string            `json:"document_hash,omitempty"`
	Warnings          []string          `json:"warnings,omitempty"`
	Errors            []string          `json:"errors,omitempty"`
	DebugInfo         map[string]string `json:"debug_info,omitempty"`
}

// DefaultAnalysisConfig returns default analysis configuration
func DefaultAnalysisConfig() AnalysisConfig {
	return AnalysisConfig{
		EnableStructureDetection: true,
		EnableClassification:     true,
		EnableQualityMetrics:     true,
		EnableSuggestions:        true,
		MinConfidence:            0.7,
		MaxProcessingTime:        10000, // 10 seconds
		DetailedAnalysis:         true,
	}
}

// NewDocumentAnalyzer creates a new document analyzer
func NewDocumentAnalyzer() *DocumentAnalyzer {
	return &DocumentAnalyzer{
		structureDetector: NewStructureDetector(false),
		classifier:        NewDocumentClassifier(),
		config:            DefaultAnalysisConfig(),
	}
}

// NewDocumentAnalyzerWithConfig creates a new document analyzer with custom configuration
func NewDocumentAnalyzerWithConfig(config AnalysisConfig) *DocumentAnalyzer {
	return &DocumentAnalyzer{
		structureDetector: NewStructureDetector(false),
		classifier:        NewDocumentClassifier(),
		config:            config,
	}
}

// Analyze performs comprehensive document analysis
func (da *DocumentAnalyzer) Analyze(elements []extraction.ContentElement) (*DocumentAnalysis, error) {
	startTime := time.Now()

	// Handle empty content gracefully with better debugging
	if len(elements) == 0 {
		// Create a basic analysis for empty content with debugging info
		analysis := &DocumentAnalysis{
			Type:     "unknown",
			Sections: []Section{},
			Statistics: ContentStats{
				WordCount:      0,
				CharacterCount: 0,
				ParagraphCount: 0,
				SentenceCount:  0,
				PageCount:      0,
				TableCount:     0,
				ImageCount:     0,
				FormFieldCount: 0,
				HeaderCount:    0,
				ListCount:      0,
				FootnoteCount:  0,
				LinkCount:      0,
				ReadingTime:    0.0,
				ContentDensity: make(map[string]float64),
			},
			Quality: QualityMetrics{
				OverallScore:       0.0,
				ReadabilityScore:   0.0,
				CompletenessScore:  0.0,
				ConsistencyScore:   0.0,
				AccessibilityScore: 0.0,
				StructureScore:     0.0,
				DetailedMetrics:    make(map[string]float64),
				IssuesFound: []QualityIssue{
					{
						Type:        "extraction",
						Severity:    "high",
						Description: "No content elements were extracted from the document",
					},
				},
				PositiveIndicators: []string{},
			},
			Suggestions: []string{
				"This PDF may be image-based or scanned - try using OCR tools",
				"The PDF may be corrupted or use unsupported features",
				"Check if the PDF requires a password or has security restrictions",
				"Verify the PDF file is valid and not corrupted",
				"Try using pdf_validate_file to check document integrity",
				"Use pdf_get_metadata to understand document properties",
				"Consider using pdf_assets_file if the document contains images",
			},
			Metadata: AnalysisMetadata{
				AnalysisVersion:   "1.0.0",
				AnalyzedAt:        startTime,
				ComponentsUsed:    []string{"empty-content-handler"},
				ConfigurationUsed: da.config,
				Warnings: []string{
					"No content elements provided for analysis",
					"Document may be image-based, corrupted, or use unsupported features",
				},
				Errors: []string{},
			},
		}

		return analysis, nil
	}

	analysis := &DocumentAnalysis{
		Metadata: AnalysisMetadata{
			AnalysisVersion:   "1.0.0",
			AnalyzedAt:        startTime,
			ComponentsUsed:    []string{},
			ConfigurationUsed: da.config,
			Warnings:          []string{},
			Errors:            []string{},
		},
	}

	// Track processing timeout
	timeout := time.Duration(da.config.MaxProcessingTime) * time.Millisecond
	done := make(chan bool, 1)
	var analysisErr error

	go func() {
		defer func() {
			if r := recover(); r != nil {
				analysisErr = fmt.Errorf("analysis panicked: %v", r)
			}
			done <- true
		}()

		// Perform analysis steps
		analysisErr = da.performAnalysis(elements, analysis)
	}()

	// Wait for completion or timeout
	select {
	case <-done:
		if analysisErr != nil {
			return nil, analysisErr
		}
	case <-time.After(timeout):
		return nil, fmt.Errorf("analysis timed out after %v", timeout)
	}

	// Set final metadata
	analysis.Metadata.ProcessingTime = time.Since(startTime)

	return analysis, nil
}

// performAnalysis executes the actual analysis steps
func (da *DocumentAnalyzer) performAnalysis(elements []extraction.ContentElement, analysis *DocumentAnalysis) error {
	var structure *DocumentStructure

	// Step 1: Detect document structure
	if da.config.EnableStructureDetection {
		var err error
		structure, err = da.structureDetector.DetectStructure(elements)
		if err != nil {
			analysis.Metadata.Warnings = append(analysis.Metadata.Warnings,
				fmt.Sprintf("Structure detection failed: %v", err))
		} else {
			analysis.Sections = da.convertToSections(structure)
			analysis.Metadata.ComponentsUsed = append(analysis.Metadata.ComponentsUsed, "structure_detection")
		}
	}

	// Step 2: Calculate content statistics
	analysis.Statistics = da.calculateContentStats(elements)
	analysis.Metadata.ComponentsUsed = append(analysis.Metadata.ComponentsUsed, "content_statistics")

	// Step 3: Classify document type
	if da.config.EnableClassification {
		docType, err := da.classifyDocument(elements, structure)
		if err != nil {
			analysis.Metadata.Warnings = append(analysis.Metadata.Warnings,
				fmt.Sprintf("Classification failed: %v", err))
			analysis.Type = "unknown"
		} else {
			analysis.Type = docType
			analysis.Metadata.ComponentsUsed = append(analysis.Metadata.ComponentsUsed, "classification")
		}
	}

	// Step 4: Assess document quality
	if da.config.EnableQualityMetrics {
		analysis.Quality = da.assessQuality(elements, analysis.Statistics, analysis.Sections)
		analysis.Metadata.ComponentsUsed = append(analysis.Metadata.ComponentsUsed, "quality_metrics")
	}

	// Step 5: Generate suggestions
	if da.config.EnableSuggestions {
		analysis.Suggestions = da.generateSuggestions(analysis.Quality, analysis.Statistics, analysis.Type)
		analysis.Metadata.ComponentsUsed = append(analysis.Metadata.ComponentsUsed, "suggestions")
	}

	return nil
}

// convertToSections converts document structure to sections
func (da *DocumentAnalyzer) convertToSections(structure *DocumentStructure) []Section {
	var sections []Section

	if structure.Root == nil {
		return sections
	}

	// Convert structure nodes to sections
	for _, node := range structure.Sections {
		section := Section{
			ID:        node.ID,
			Title:     node.Content,
			Level:     node.Level,
			Type:      string(node.Type),
			PageStart: node.PageNumber,
			PageEnd:   node.PageNumber,
			Elements:  node.Elements,
		}

		// Add content from child nodes
		section.Content = da.extractSectionContent(node)

		// Add subsections
		section.Subsections = da.convertChildrenToSections(node.Children)

		sections = append(sections, section)
	}

	return sections
}

// convertChildrenToSections recursively converts child nodes to subsections
func (da *DocumentAnalyzer) convertChildrenToSections(children []*StructureNode) []Section {
	var subsections []Section

	for _, child := range children {
		if child.Type == StructureTypeSection || child.Type == StructureTypeHeader {
			subsection := Section{
				ID:          child.ID,
				Title:       child.Content,
				Level:       child.Level,
				Type:        string(child.Type),
				PageStart:   child.PageNumber,
				PageEnd:     child.PageNumber,
				Elements:    child.Elements,
				Content:     da.extractSectionContent(child),
				Subsections: da.convertChildrenToSections(child.Children),
			}
			subsections = append(subsections, subsection)
		}
	}

	return subsections
}

// extractSectionContent extracts text content from a structure node
func (da *DocumentAnalyzer) extractSectionContent(node *StructureNode) string {
	var content strings.Builder

	// Add content from current node
	if node.Content != "" {
		content.WriteString(node.Content)
		content.WriteString("\n\n")
	}

	// Add content from child nodes
	for _, child := range node.Children {
		if child.Type == StructureTypeParagraph {
			content.WriteString(child.Content)
			content.WriteString("\n\n")
		}
	}

	return strings.TrimSpace(content.String())
}

// calculateContentStats calculates comprehensive content statistics
func (da *DocumentAnalyzer) calculateContentStats(elements []extraction.ContentElement) ContentStats {
	stats := ContentStats{
		ContentDensity: make(map[string]float64),
	}

	wordCount := 0
	charCount := 0
	sentenceCount := 0
	paragraphCount := 0
	pages := make(map[int]bool)

	for _, element := range elements {
		pages[element.PageNumber] = true

		switch element.Type {
		case extraction.ContentTypeText:
			if textElem, ok := element.Content.(extraction.TextElement); ok {
				text := textElem.Text
				words := strings.Fields(text)
				wordCount += len(words)
				charCount += len(text)

				// Count sentences (simple approach)
				sentences := strings.Split(text, ".")
				sentenceCount += len(sentences) - 1 // Subtract 1 for the last empty split

				// Count paragraphs (simple approach)
				paragraphs := strings.Split(text, "\n\n")
				paragraphCount += len(paragraphs)
			}
		case extraction.ContentTypeImage:
			stats.ImageCount++
		case extraction.ContentTypeForm:
			stats.FormFieldCount++
		}
	}

	stats.WordCount = wordCount
	stats.CharacterCount = charCount
	stats.SentenceCount = sentenceCount
	stats.ParagraphCount = paragraphCount
	stats.PageCount = len(pages)

	// Calculate reading time (average 200 words per minute)
	stats.ReadingTime = float64(wordCount) / 200.0

	// Calculate content density
	if stats.PageCount > 0 {
		stats.ContentDensity["words_per_page"] = float64(wordCount) / float64(stats.PageCount)
		stats.ContentDensity["characters_per_page"] = float64(charCount) / float64(stats.PageCount)
		stats.ContentDensity["images_per_page"] = float64(stats.ImageCount) / float64(stats.PageCount)
	}

	return stats
}

// classifyDocument classifies the document type
func (da *DocumentAnalyzer) classifyDocument(elements []extraction.ContentElement, structure *DocumentStructure) (string, error) {
	// Extract content text from elements
	content := da.extractContentText(elements)

	// If no structure available, create a minimal one
	if structure == nil {
		structure = &DocumentStructure{
			Root: &StructureNode{
				ID:       "root",
				Type:     StructureTypeDocument,
				Content:  "Document",
				Children: []*StructureNode{},
			},
			Sections:      []*StructureNode{},
			Headers:       make(map[int][]*StructureNode),
			ReadingOrder:  []*StructureNode{},
			PageStructure: make(map[int][]*StructureNode),
			Statistics:    &StructureStatistics{},
		}
	}

	// Use the classifier to determine document type
	ctx := context.Background()
	classificationResult, err := da.classifier.Classify(ctx, structure, content)
	if err != nil {
		return "unknown", err
	}

	return string(classificationResult.Classification.Type), nil
}

// extractContentText extracts all text content from elements as a single string
func (da *DocumentAnalyzer) extractContentText(elements []extraction.ContentElement) string {
	var content strings.Builder

	for _, element := range elements {
		if element.Type == extraction.ContentTypeText {
			if textElem, ok := element.Content.(extraction.TextElement); ok {
				content.WriteString(textElem.Text)
				content.WriteString(" ")
			}
		}
	}

	return strings.TrimSpace(content.String())
}

// assessQuality assesses document quality across multiple dimensions
func (da *DocumentAnalyzer) assessQuality(elements []extraction.ContentElement, stats ContentStats, sections []Section) QualityMetrics {
	quality := QualityMetrics{
		DetailedMetrics:    make(map[string]float64),
		IssuesFound:        []QualityIssue{},
		PositiveIndicators: []string{},
	}

	// Assess readability
	quality.ReadabilityScore = da.assessReadability(stats)
	quality.DetailedMetrics["readability"] = quality.ReadabilityScore

	// Assess completeness
	quality.CompletenessScore = da.assessCompleteness(stats, sections)
	quality.DetailedMetrics["completeness"] = quality.CompletenessScore

	// Assess consistency
	quality.ConsistencyScore = da.assessConsistency(elements)
	quality.DetailedMetrics["consistency"] = quality.ConsistencyScore

	// Assess structure
	quality.StructureScore = da.assessStructure(sections)
	quality.DetailedMetrics["structure"] = quality.StructureScore

	// Assess accessibility
	quality.AccessibilityScore = da.assessAccessibility(elements, stats)
	quality.DetailedMetrics["accessibility"] = quality.AccessibilityScore

	// Calculate overall score
	scores := []float64{
		quality.ReadabilityScore,
		quality.CompletenessScore,
		quality.ConsistencyScore,
		quality.StructureScore,
		quality.AccessibilityScore,
	}

	var sum float64
	for _, score := range scores {
		sum += score
	}
	quality.OverallScore = sum / float64(len(scores))

	// Generate quality issues and positive indicators
	da.generateQualityIssues(&quality, stats, sections)
	da.generatePositiveIndicators(&quality, stats, sections)

	return quality
}

// assessReadability calculates readability score
func (da *DocumentAnalyzer) assessReadability(stats ContentStats) float64 {
	if stats.WordCount == 0 || stats.SentenceCount == 0 {
		return 0.0
	}

	// Simple readability metric based on average sentence length
	avgSentenceLength := float64(stats.WordCount) / float64(stats.SentenceCount)

	// Ideal sentence length is around 15-20 words
	idealLength := 17.5
	deviation := math.Abs(avgSentenceLength - idealLength)

	// Score inversely proportional to deviation
	score := math.Max(0, 1.0-(deviation/idealLength))

	return score
}

// assessCompleteness calculates completeness score
func (da *DocumentAnalyzer) assessCompleteness(stats ContentStats, sections []Section) float64 {
	score := 0.0

	// Check for basic content
	if stats.WordCount > 100 {
		score += 0.3
	}

	// Check for structure
	if len(sections) > 0 {
		score += 0.3
	}

	// Check for multiple pages
	if stats.PageCount > 1 {
		score += 0.2
	}

	// Check for mixed content
	if stats.ImageCount > 0 || stats.FormFieldCount > 0 {
		score += 0.2
	}

	return math.Min(score, 1.0)
}

// assessConsistency calculates consistency score
func (da *DocumentAnalyzer) assessConsistency(elements []extraction.ContentElement) float64 {
	// Simple consistency check based on font usage
	fontCounts := make(map[string]int)
	totalTextElements := 0

	for _, element := range elements {
		if element.Type == extraction.ContentTypeText {
			if textElem, ok := element.Content.(extraction.TextElement); ok {
				fontCounts[textElem.Properties.FontName]++
				totalTextElements++
			}
		}
	}

	if totalTextElements == 0 {
		return 1.0
	}

	// Calculate entropy of font usage (lower entropy = more consistent)
	entropy := 0.0
	for _, count := range fontCounts {
		p := float64(count) / float64(totalTextElements)
		if p > 0 {
			entropy -= p * math.Log2(p)
		}
	}

	// Normalize entropy to 0-1 score (lower entropy = higher score)
	maxEntropy := math.Log2(float64(len(fontCounts)))
	if maxEntropy == 0 {
		return 1.0
	}

	return 1.0 - (entropy / maxEntropy)
}

// assessStructure calculates structure score
func (da *DocumentAnalyzer) assessStructure(sections []Section) float64 {
	if len(sections) == 0 {
		return 0.0
	}

	score := 0.0

	// Check for hierarchical structure
	hasHierarchy := false
	for _, section := range sections {
		if len(section.Subsections) > 0 {
			hasHierarchy = true
			break
		}
	}

	if hasHierarchy {
		score += 0.5
	}

	// Check for reasonable section count
	sectionCount := len(sections)
	if sectionCount >= 3 && sectionCount <= 10 {
		score += 0.3
	} else if sectionCount > 0 {
		score += 0.1
	}

	// Check for titled sections
	titledSections := 0
	for _, section := range sections {
		if section.Title != "" {
			titledSections++
		}
	}

	if titledSections == len(sections) {
		score += 0.2
	}

	return math.Min(score, 1.0)
}

// assessAccessibility calculates accessibility score
func (da *DocumentAnalyzer) assessAccessibility(elements []extraction.ContentElement, stats ContentStats) float64 {
	score := 0.0

	// Check for reasonable text size
	largeTextCount := 0
	totalTextElements := 0

	for _, element := range elements {
		if element.Type == extraction.ContentTypeText {
			if textElem, ok := element.Content.(extraction.TextElement); ok {
				totalTextElements++
				if textElem.Properties.FontSize >= 12.0 {
					largeTextCount++
				}
			}
		}
	}

	if totalTextElements > 0 {
		largeTextRatio := float64(largeTextCount) / float64(totalTextElements)
		score += 0.4 * largeTextRatio
	}

	// Check for structured content
	if stats.PageCount > 0 {
		score += 0.3
	}

	// Check for reasonable content density
	if stats.ContentDensity["words_per_page"] > 50 && stats.ContentDensity["words_per_page"] < 800 {
		score += 0.3
	}

	return math.Min(score, 1.0)
}

// generateQualityIssues identifies quality issues in the document
func (da *DocumentAnalyzer) generateQualityIssues(quality *QualityMetrics, stats ContentStats, sections []Section) {
	// Check for low readability
	if quality.ReadabilityScore < 0.5 {
		quality.IssuesFound = append(quality.IssuesFound, QualityIssue{
			Type:        "readability",
			Severity:    "medium",
			Description: "Document may be difficult to read due to sentence structure",
			Suggestion:  "Consider breaking long sentences into shorter ones",
			Confidence:  0.7,
		})
	}

	// Check for poor structure
	if quality.StructureScore < 0.3 {
		quality.IssuesFound = append(quality.IssuesFound, QualityIssue{
			Type:        "structure",
			Severity:    "high",
			Description: "Document lacks clear structure and organization",
			Suggestion:  "Add headers and organize content into logical sections",
			Confidence:  0.8,
		})
	}

	// Check for low content density
	if stats.ContentDensity["words_per_page"] < 50 {
		quality.IssuesFound = append(quality.IssuesFound, QualityIssue{
			Type:        "content_density",
			Severity:    "low",
			Description: "Document appears to have low content density",
			Suggestion:  "Consider consolidating content or adding more detailed information",
			Confidence:  0.6,
		})
	}
}

// generatePositiveIndicators identifies positive aspects of the document
func (da *DocumentAnalyzer) generatePositiveIndicators(quality *QualityMetrics, stats ContentStats, sections []Section) {
	if quality.ReadabilityScore > 0.8 {
		quality.PositiveIndicators = append(quality.PositiveIndicators, "Excellent readability with well-structured sentences")
	}

	if quality.StructureScore > 0.8 {
		quality.PositiveIndicators = append(quality.PositiveIndicators, "Well-organized document structure with clear sections")
	}

	if stats.ContentDensity["words_per_page"] > 200 && stats.ContentDensity["words_per_page"] < 600 {
		quality.PositiveIndicators = append(quality.PositiveIndicators, "Good content density with appropriate amount of information per page")
	}

	if len(sections) > 0 {
		quality.PositiveIndicators = append(quality.PositiveIndicators, "Document has clear sectional organization")
	}
}

// generateSuggestions generates actionable suggestions for document improvement
func (da *DocumentAnalyzer) generateSuggestions(quality QualityMetrics, stats ContentStats, docType string) []string {
	var suggestions []string

	// Generic suggestions based on quality scores
	if quality.OverallScore < 0.6 {
		suggestions = append(suggestions, "Consider reviewing and improving the overall document quality")
	}

	if quality.ReadabilityScore < 0.7 {
		suggestions = append(suggestions, "Improve readability by using shorter sentences and simpler language")
	}

	if quality.StructureScore < 0.7 {
		suggestions = append(suggestions, "Add clear headings and organize content into logical sections")
	}

	if quality.ConsistencyScore < 0.7 {
		suggestions = append(suggestions, "Ensure consistent formatting and font usage throughout the document")
	}

	// Document type specific suggestions
	switch docType {
	case "invoice":
		if stats.FormFieldCount == 0 {
			suggestions = append(suggestions, "Consider adding form fields for better data extraction")
		}
	case "report":
		if stats.ImageCount == 0 {
			suggestions = append(suggestions, "Consider adding charts or graphs to support your data")
		}
		if len(quality.IssuesFound) > 0 {
			suggestions = append(suggestions, "Ensure all sections have clear titles and consistent formatting")
		}
	case "form":
		if stats.FormFieldCount < 5 {
			suggestions = append(suggestions, "Ensure all form fields are properly defined and accessible")
		}
	}

	// Content-based suggestions
	if stats.WordCount < 100 {
		suggestions = append(suggestions, "Document appears to be very short - consider adding more detailed content")
	}

	if stats.PageCount == 1 && stats.WordCount > 1000 {
		suggestions = append(suggestions, "Consider breaking this content across multiple pages for better readability")
	}

	// Remove duplicates
	suggestionSet := make(map[string]bool)
	uniqueSuggestions := []string{}
	for _, suggestion := range suggestions {
		if !suggestionSet[suggestion] {
			suggestionSet[suggestion] = true
			uniqueSuggestions = append(uniqueSuggestions, suggestion)
		}
	}

	return uniqueSuggestions
}
