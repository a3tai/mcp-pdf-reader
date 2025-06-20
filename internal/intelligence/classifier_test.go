package intelligence

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestNewDocumentClassifier(t *testing.T) {
	classifier := NewDocumentClassifier()

	if classifier == nil {
		t.Fatal("Expected classifier to be created, got nil")
	}

	if !classifier.initialized {
		t.Error("Expected classifier to be initialized")
	}

	if len(classifier.rules) == 0 {
		t.Error("Expected classifier to have default rules loaded")
	}

	if classifier.version == "" {
		t.Error("Expected classifier to have a version")
	}
}

func TestNewDocumentClassifierWithConfig(t *testing.T) {
	config := ClassificationConfig{
		MinConfidenceThreshold: 0.5,
		MaxAlternatives:        2,
		EnableCustomRules:      false,
	}

	classifier := NewDocumentClassifierWithConfig(config)

	if classifier == nil {
		t.Fatal("Expected classifier to be created, got nil")
	}

	if classifier.config.MinConfidenceThreshold != 0.5 {
		t.Error("Expected custom config to be applied")
	}
}

func TestClassifyInvoice(t *testing.T) {
	classifier := NewDocumentClassifier()
	structure := createMockDocumentStructure()

	invoiceContent := `
		INVOICE #12345

		Bill To:
		John Doe
		123 Main St

		Invoice Date: 01/15/2024
		Due Date: 02/15/2024

		Item                Quantity    Price       Total
		Widget A           5           $10.00      $50.00
		Widget B           2           $25.00      $50.00

		Subtotal:                                  $100.00
		Tax (8%):                                  $8.00
		Total Amount Due:                          $108.00

		Payment Terms: Net 30
		Please remit payment by due date.
	`

	ctx := context.Background()
	result, err := classifier.Classify(ctx, structure, invoiceContent)
	if err != nil {
		t.Fatalf("Classification failed: %v", err)
	}

	if result.Classification.Type != DocumentTypeInvoice {
		t.Errorf("Expected DocumentTypeInvoice, got %v", result.Classification.Type)
	}

	if result.Classification.Confidence < 0.3 {
		t.Errorf("Expected confidence >= 0.3, got %f", result.Classification.Confidence)
	}

	if len(result.Classification.Reasons) == 0 {
		t.Error("Expected classification reasons to be provided")
	}
}

func TestClassifyReport(t *testing.T) {
	classifier := NewDocumentClassifier()
	structure := createMockDocumentStructure()

	reportContent := `
		QUARTERLY REPORT
		Q4 2024 ANALYSIS

		EXECUTIVE SUMMARY
		This report presents the findings of our quarterly analysis.

		METHODOLOGY
		We conducted a comprehensive review of all data sources.

		RESULTS
		The analysis revealed several key insights:
		1. Sales increased by 15%
		2. Customer satisfaction improved
		3. Market share grew by 3%

		RECOMMENDATIONS
		Based on our findings, we recommend:
		- Increase marketing spend
		- Expand product line
		- Improve customer service

		CONCLUSION
		The quarter showed positive results across all metrics.

		APPENDIX
		Detailed tables and charts are included below.
	`

	ctx := context.Background()
	result, err := classifier.Classify(ctx, structure, reportContent)
	if err != nil {
		t.Fatalf("Classification failed: %v", err)
	}

	if result.Classification.Type != DocumentTypeReport {
		t.Errorf("Expected DocumentTypeReport, got %v", result.Classification.Type)
	}

	if result.Classification.Confidence < 0.3 {
		t.Errorf("Expected confidence >= 0.3, got %f", result.Classification.Confidence)
	}
}

func TestClassifyForm(t *testing.T) {
	classifier := NewDocumentClassifier()
	structure := createMockDocumentStructure()

	formContent := `
		APPLICATION FORM

		Please complete all required fields below:

		Name: ________________________________

		Date of Birth: _______________________

		Address: _____________________________
		         _____________________________

		Phone Number: _______________________

		Email: ______________________________

		Please check all that apply:
		[ ] Option A
		[ ] Option B
		[ ] Option C

		Have you completed this before?
		( ) Yes  ( ) No

		Signature: __________________________

		Date: _______________________________

		Please submit this form by the deadline.
	`

	ctx := context.Background()
	result, err := classifier.Classify(ctx, structure, formContent)
	if err != nil {
		t.Fatalf("Classification failed: %v", err)
	}

	if result.Classification.Type != DocumentTypeForm {
		t.Errorf("Expected DocumentTypeForm, got %v", result.Classification.Type)
	}

	if result.Classification.Confidence < 0.3 {
		t.Errorf("Expected confidence >= 0.3, got %f", result.Classification.Confidence)
	}
}

func TestClassifyContract(t *testing.T) {
	classifier := NewDocumentClassifier()
	structure := createMockDocumentStructure()

	contractContent := `
		SERVICE AGREEMENT

		This Agreement is entered into between the parties as follows:

		WHEREAS, Party A desires to engage Party B for services;
		WHEREAS, Party B agrees to provide such services;

		NOW THEREFORE, the parties agree to the following terms and conditions:

		1. SCOPE OF SERVICES
		Party B shall provide consulting services as described herein.

		2. TERM
		This Agreement shall commence on the effective date and continue for one year.

		3. COMPENSATION
		Party A shall pay Party B the agreed-upon fees.

		4. TERMINATION
		Either party may terminate this Agreement with 30 days notice.

		5. GOVERNING LAW
		This Agreement shall be governed by the laws of the state.

		6. DISPUTE RESOLUTION
		Any disputes shall be resolved through binding arbitration.

		IN WITNESS WHEREOF, the parties have executed this Agreement.

		Party A: _________________________    Date: __________

		Party B: _________________________    Date: __________
	`

	ctx := context.Background()
	result, err := classifier.Classify(ctx, structure, contractContent)
	if err != nil {
		t.Fatalf("Classification failed: %v", err)
	}

	if result.Classification.Type != DocumentTypeContract {
		t.Errorf("Expected DocumentTypeContract, got %v", result.Classification.Type)
	}

	if result.Classification.Confidence < 0.3 {
		t.Errorf("Expected confidence >= 0.3, got %f", result.Classification.Confidence)
	}
}

func TestClassifyAcademicPaper(t *testing.T) {
	classifier := NewDocumentClassifier()
	structure := createMockDocumentStructure()

	academicContent := `
		A Study of Machine Learning Applications in Document Classification

		ABSTRACT
		This paper presents a comprehensive analysis of machine learning techniques
		for document classification. We propose a novel approach that combines
		multiple algorithms to achieve improved accuracy.

		INTRODUCTION
		Document classification has been an active area of research for decades.
		Previous studies have shown that traditional methods have limitations.

		LITERATURE REVIEW
		Smith et al. (2020) demonstrated that neural networks can improve classification.
		Johnson and Brown (2021) proposed a hybrid approach combining SVM and neural networks.

		METHODOLOGY
		We conducted experiments using a dataset of 10,000 documents.
		The methodology involved feature extraction, model training, and evaluation.

		RESULTS
		Our experiments showed a 15% improvement in accuracy compared to baseline methods.
		Figure 1 shows the performance comparison across different algorithms.

		DISCUSSION
		The results indicate that our proposed method is effective for document classification.
		However, there are limitations that need to be addressed in future work.

		CONCLUSION
		This study contributes to the field by demonstrating improved classification accuracy.
		Future research should explore additional feature engineering techniques.

		REFERENCES
		[1] Smith, A., et al. (2020). Neural Networks for Text Classification. Journal of AI, 15(3), 45-62.
		[2] Johnson, B., & Brown, C. (2021). Hybrid Approaches to Document Analysis. Proceedings of ICML.
	`

	ctx := context.Background()
	result, err := classifier.Classify(ctx, structure, academicContent)
	if err != nil {
		t.Fatalf("Classification failed: %v", err)
	}

	if result.Classification.Type != DocumentTypeAcademic {
		t.Errorf("Expected DocumentTypeAcademic, got %v", result.Classification.Type)
	}

	if result.Classification.Confidence < 0.3 {
		t.Errorf("Expected confidence >= 0.3, got %f", result.Classification.Confidence)
	}
}

func TestClassifyUnknownDocument(t *testing.T) {
	classifier := NewDocumentClassifier()
	structure := createMockDocumentStructure()

	unknownContent := `
		This is just some random text that doesn't match any particular document type.
		It has no specific formatting or keywords that would indicate what type of document it is.
		Just generic content without any clear patterns.
	`

	ctx := context.Background()
	result, err := classifier.Classify(ctx, structure, unknownContent)
	if err != nil {
		t.Fatalf("Classification failed: %v", err)
	}

	// Should either be Unknown or have very low confidence
	if result.Classification.Type != DocumentTypeUnknown && result.Classification.Confidence > 0.3 {
		t.Errorf("Expected DocumentTypeUnknown or low confidence for generic content, got %v with confidence %f",
			result.Classification.Type, result.Classification.Confidence)
	}
}

func TestExtractFeatures(t *testing.T) {
	classifier := NewDocumentClassifier()
	structure := createMockDocumentStructure()

	content := `
		This is a test document with some content.
		It has multiple sentences and paragraphs.

		Here's another paragraph with some numbers: 123, 456.
		And some currency: $100.00, $250.50.
		Plus an email: test@example.com
		And a URL: https://example.com
	`

	features := classifier.extractFeatures(structure, content)

	if features.WordCount == 0 {
		t.Error("Expected word count > 0")
	}

	if features.PageCount != 1 {
		t.Errorf("Expected page count = 1, got %d", features.PageCount)
	}

	if features.EmailCount == 0 {
		t.Error("Expected to detect email address")
	}

	if features.URLCount == 0 {
		t.Error("Expected to detect URL")
	}

	if features.NumericPatterns["currency"] == 0 {
		t.Error("Expected to detect currency patterns")
	}
}

func TestEvaluateKeywordRules(t *testing.T) {
	classifier := NewDocumentClassifier()

	rule := ClassificationRule{
		Name:            "test_rule",
		DocumentType:    DocumentTypeInvoice,
		Keywords:        []string{"invoice", "bill", "payment"},
		KeywordPatterns: []string{`(?i)invoice\s*#?\s*\d+`},
		Weight:          0.8,
		MinConfidence:   0.1,
	}

	content := "This is an INVOICE #12345 for payment due. Please remit bill amount."

	confidence, reasons := classifier.evaluateKeywordRules(rule, content)

	if confidence == 0 {
		t.Error("Expected confidence > 0 for matching content")
	}

	if len(reasons) == 0 {
		t.Error("Expected reasons to be provided")
	}

	// Should find keywords: invoice, payment, bill
	// Plus pattern match for "INVOICE #12345"
	expectedMatches := 4 // 3 keywords + 1 pattern
	if len(reasons) < expectedMatches {
		t.Errorf("Expected at least %d reasons, got %d", expectedMatches, len(reasons))
	}
}

func TestGetAllDefaultRules(t *testing.T) {
	rules := GetAllDefaultRules()

	if len(rules) == 0 {
		t.Error("Expected default rules to be returned")
	}

	// Check that we have rules for major document types
	documentTypes := make(map[DocumentType]bool)
	for _, rule := range rules {
		documentTypes[rule.DocumentType] = true
	}

	expectedTypes := []DocumentType{
		DocumentTypeInvoice,
		DocumentTypeReport,
		DocumentTypeForm,
		DocumentTypeContract,
		DocumentTypeAcademic,
	}

	for _, expectedType := range expectedTypes {
		if !documentTypes[expectedType] {
			t.Errorf("Expected rules for document type %v", expectedType)
		}
	}
}

func TestClassifierWithTimeout(t *testing.T) {
	classifier := NewDocumentClassifier()
	structure := createMockDocumentStructure()

	content := "Simple test content"

	// Create context with very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	// Wait a bit to ensure timeout
	time.Sleep(1 * time.Millisecond)

	_, err := classifier.Classify(ctx, structure, content)

	if err == nil {
		t.Error("Expected timeout error")
	}

	if err != context.DeadlineExceeded {
		t.Errorf("Expected context.DeadlineExceeded, got %v", err)
	}
}

func TestClassifierVersionAndConfig(t *testing.T) {
	classifier := NewDocumentClassifier()

	version := classifier.GetVersion()
	if version == "" {
		t.Error("Expected non-empty version")
	}

	config := classifier.GetConfig()
	if config.MinConfidenceThreshold == 0 {
		t.Error("Expected non-zero confidence threshold in config")
	}

	// Test setting new config
	newConfig := ClassificationConfig{
		MinConfidenceThreshold: 0.9,
		MaxAlternatives:        5,
	}
	classifier.SetConfig(newConfig)

	updatedConfig := classifier.GetConfig()
	if updatedConfig.MinConfidenceThreshold != 0.9 {
		t.Error("Expected config to be updated")
	}
}

func TestClassificationCaching(t *testing.T) {
	config := ClassificationConfig{
		CacheClassifications:   true,
		MinConfidenceThreshold: 0.1,
	}
	classifier := NewDocumentClassifierWithConfig(config)
	structure := createMockDocumentStructure()

	content := "Test content for caching"
	ctx := context.Background()

	// First classification
	start1 := time.Now()
	result1, err1 := classifier.Classify(ctx, structure, content)
	duration1 := time.Since(start1)

	if err1 != nil {
		t.Fatalf("First classification failed: %v", err1)
	}

	// Second classification (should be faster due to caching)
	start2 := time.Now()
	result2, err2 := classifier.Classify(ctx, structure, content)
	duration2 := time.Since(start2)

	if err2 != nil {
		t.Fatalf("Second classification failed: %v", err2)
	}

	// Results should be the same
	if result1.Classification.Type != result2.Classification.Type {
		t.Error("Expected cached result to match original")
	}

	// Second call should be faster (though this might be flaky in some environments)
	if duration2 > duration1 {
		t.Logf("Warning: Cached call was not faster (might be expected in test environment)")
	}
}

// Helper function to create a mock document structure for testing
func createMockDocumentStructure() *DocumentStructure {
	headerNode := &StructureNode{
		ID:    "header1",
		Type:  StructureTypeHeader,
		Level: 1,
	}

	return &DocumentStructure{
		Root: &StructureNode{
			ID:   "root",
			Type: StructureTypeDocument,
			Children: []*StructureNode{
				headerNode,
				{
					ID:   "para1",
					Type: StructureTypeParagraph,
				},
				{
					ID:   "table1",
					Type: StructureTypeTable,
				},
			},
		},
		Headers: map[int][]*StructureNode{
			1: {headerNode},
		},
		PageStructure: map[int][]*StructureNode{
			1: {
				{ID: "page1_content", Type: StructureTypeParagraph},
			},
		},
		Statistics: &StructureStatistics{
			TotalNodes:     4,
			HeaderCount:    1,
			ParagraphCount: 1,
			TableCount:     1,
			ListCount:      0,
			ImageCount:     0,
		},
	}
}

// Benchmark tests
func BenchmarkClassifyInvoice(b *testing.B) {
	classifier := NewDocumentClassifier()
	structure := createMockDocumentStructure()

	content := `
		INVOICE #12345
		Total Amount Due: $108.00
		Payment Terms: Net 30
	`

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := classifier.Classify(ctx, structure, content)
		if err != nil {
			b.Fatalf("Classification failed: %v", err)
		}
	}
}

func BenchmarkExtractFeatures(b *testing.B) {
	classifier := NewDocumentClassifier()
	structure := createMockDocumentStructure()

	content := strings.Repeat("This is test content with various words and sentences. ", 100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		classifier.extractFeatures(structure, content)
	}
}
