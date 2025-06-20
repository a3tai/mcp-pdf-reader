package intelligence

import (
	"time"
)

// DocumentType represents the type/category of a document
type DocumentType string

const (
	DocumentTypeUnknown      DocumentType = "unknown"
	DocumentTypeInvoice      DocumentType = "invoice"
	DocumentTypeReport       DocumentType = "report"
	DocumentTypeForm         DocumentType = "form"
	DocumentTypeContract     DocumentType = "contract"
	DocumentTypeAcademic     DocumentType = "academic_paper"
	DocumentTypeManual       DocumentType = "manual"
	DocumentTypeLetter       DocumentType = "letter"
	DocumentTypeBrochure     DocumentType = "brochure"
	DocumentTypeTechnical    DocumentType = "technical_document"
	DocumentTypeFinancial    DocumentType = "financial_statement"
	DocumentTypeLegal        DocumentType = "legal_document"
	DocumentTypeResume       DocumentType = "resume"
	DocumentTypePresentation DocumentType = "presentation"
	DocumentTypeNewsletter   DocumentType = "newsletter"
	DocumentTypeCatalog      DocumentType = "catalog"
)

// DocumentClassification represents the result of document classification
type DocumentClassification struct {
	// Primary classification
	Type       DocumentType `json:"type"`
	Confidence float64      `json:"confidence"` // 0.0 to 1.0

	// Alternative classifications
	Alternatives []ClassificationAlternative `json:"alternatives,omitempty"`

	// Classification reasoning
	Reasons []ClassificationReason `json:"reasons"`

	// Metrics and analysis
	Metrics ClassificationMetrics `json:"metrics"`

	// Processing metadata
	ProcessedAt time.Time `json:"processed_at"`
	Version     string    `json:"version"`
	ModelUsed   string    `json:"model_used"`
}

// ClassificationAlternative represents an alternative classification possibility
type ClassificationAlternative struct {
	Type       DocumentType `json:"type"`
	Confidence float64      `json:"confidence"`
	Reasons    []string     `json:"reasons,omitempty"`
}

// ClassificationReason explains why a particular classification was made
type ClassificationReason struct {
	Rule       string  `json:"rule"`       // Name of the rule that triggered
	Category   string  `json:"category"`   // Type of evidence (keyword, structure, pattern)
	Evidence   string  `json:"evidence"`   // What evidence was found
	Confidence float64 `json:"confidence"` // Confidence contribution of this reason
	Location   string  `json:"location"`   // Where in document the evidence was found
	Weight     float64 `json:"weight"`     // Weight/importance of this rule
}

// ClassificationMetrics provides detailed metrics about the classification
type ClassificationMetrics struct {
	// Rule-based metrics
	TotalRulesEvaluated int                `json:"total_rules_evaluated"`
	RulesMatched        int                `json:"rules_matched"`
	RuleScores          map[string]float64 `json:"rule_scores"`

	// Content analysis metrics
	KeywordMatches      map[string]int     `json:"keyword_matches"`
	StructureSignatures map[string]bool    `json:"structure_signatures"`
	ContentHeuristics   map[string]float64 `json:"content_heuristics"`

	// Document characteristics
	DocumentLength     int            `json:"document_length"`
	PageCount          int            `json:"page_count"`
	StructuralElements map[string]int `json:"structural_elements"`

	// Quality indicators
	TextQuality      float64 `json:"text_quality"`
	StructureQuality float64 `json:"structure_quality"`
	OverallQuality   float64 `json:"overall_quality"`
}

// ClassificationRule defines a rule for document classification
type ClassificationRule struct {
	// Rule identification
	Name         string       `json:"name"`
	DocumentType DocumentType `json:"document_type"`
	Category     string       `json:"category"` // keyword, structure, pattern, heuristic

	// Rule definition
	Keywords        []string        `json:"keywords,omitempty"`
	KeywordPatterns []string        `json:"keyword_patterns,omitempty"` // regex patterns
	StructureRules  []StructureRule `json:"structure_rules,omitempty"`
	ContentRules    []ContentRule   `json:"content_rules,omitempty"`

	// Rule parameters
	Weight          float64 `json:"weight"`           // Importance weight (0.0 to 1.0)
	MinConfidence   float64 `json:"min_confidence"`   // Minimum confidence to trigger
	RequiredMatches int     `json:"required_matches"` // Minimum matches needed

	// Rule metadata
	Description string `json:"description"`
	Priority    int    `json:"priority"`
	Enabled     bool   `json:"enabled"`
	Version     string `json:"version"`
}

// StructureRule defines structural requirements for classification
type StructureRule struct {
	ElementType     string  `json:"element_type"` // header, table, list, etc.
	MinCount        int     `json:"min_count"`
	MaxCount        int     `json:"max_count"`
	RequiredContent string  `json:"required_content,omitempty"`
	Position        string  `json:"position,omitempty"` // top, bottom, middle
	Confidence      float64 `json:"confidence"`
}

// ContentRule defines content-based requirements for classification
type ContentRule struct {
	RuleType      string  `json:"rule_type"` // regex, contains, starts_with, ends_with
	Pattern       string  `json:"pattern"`
	CaseSensitive bool    `json:"case_sensitive"`
	MinMatches    int     `json:"min_matches"`
	MaxMatches    int     `json:"max_matches"`
	Confidence    float64 `json:"confidence"`
	Location      string  `json:"location,omitempty"` // first_page, last_page, header, footer
}

// ClassificationConfig provides configuration for the document classifier
type ClassificationConfig struct {
	// Classification behavior
	MinConfidenceThreshold float64 `json:"min_confidence_threshold"` // Minimum confidence to accept classification
	MaxAlternatives        int     `json:"max_alternatives"`         // Maximum alternative classifications to return
	EnableMLModel          bool    `json:"enable_ml_model"`          // Whether to use ML model (future)

	// Rule processing
	EnableCustomRules     bool   `json:"enable_custom_rules"`
	CustomRulesPath       string `json:"custom_rules_path,omitempty"`
	RuleWeightingStrategy string `json:"rule_weighting_strategy"` // uniform, priority, confidence_based

	// Content analysis
	EnableDeepAnalysis   bool `json:"enable_deep_analysis"` // Perform detailed content analysis
	MaxContentLength     int  `json:"max_content_length"`   // Maximum content length to analyze
	KeywordCaseSensitive bool `json:"keyword_case_sensitive"`

	// Performance tuning
	MaxProcessingTime    int  `json:"max_processing_time_ms"` // Maximum processing time in milliseconds
	CacheClassifications bool `json:"cache_classifications"`
	CacheExpiryMinutes   int  `json:"cache_expiry_minutes"`

	// Debug and logging
	EnableDebugMode        bool `json:"enable_debug_mode"`
	LogClassificationSteps bool `json:"log_classification_steps"`
	ReturnDetailedMetrics  bool `json:"return_detailed_metrics"`
}

// ClassificationRuleSet represents a collection of classification rules
type ClassificationRuleSet struct {
	Version     string               `json:"version"`
	CreatedAt   time.Time            `json:"created_at"`
	Rules       []ClassificationRule `json:"rules"`
	Metadata    map[string]string    `json:"metadata,omitempty"`
	Description string               `json:"description,omitempty"`
}

// ClassificationResult represents the overall result of document analysis
type ClassificationResult struct {
	// Primary results
	Classification DocumentClassification `json:"classification"`

	// Processing information
	ProcessingTime time.Duration `json:"processing_time"`
	RulesApplied   []string      `json:"rules_applied"`
	Warnings       []string      `json:"warnings,omitempty"`
	Errors         []string      `json:"errors,omitempty"`

	// Input information
	SourceFile   string `json:"source_file,omitempty"`
	DocumentHash string `json:"document_hash,omitempty"`
	AnalysisID   string `json:"analysis_id"`
}

// DocumentFeatures represents extracted features used for classification
type DocumentFeatures struct {
	// Text-based features
	WordCount         int     `json:"word_count"`
	UniqueWords       int     `json:"unique_words"`
	AverageWordLength float64 `json:"average_word_length"`
	SentenceCount     int     `json:"sentence_count"`
	ParagraphCount    int     `json:"paragraph_count"`

	// Structure-based features
	HeaderCount    int   `json:"header_count"`
	HeaderLevels   []int `json:"header_levels"`
	ListCount      int   `json:"list_count"`
	TableCount     int   `json:"table_count"`
	ImageCount     int   `json:"image_count"`
	FormFieldCount int   `json:"form_field_count"`

	// Content patterns
	NumericPatterns map[string]int `json:"numeric_patterns"` // phone, date, currency, etc.
	EmailCount      int            `json:"email_count"`
	URLCount        int            `json:"url_count"`

	// Document layout
	PageCount    int  `json:"page_count"`
	ColumnLayout bool `json:"column_layout"`
	HasWatermark bool `json:"has_watermark"`
	HasSignature bool `json:"has_signature"`

	// Quality indicators
	TextDensity     float64 `json:"text_density"`
	ImageDensity    float64 `json:"image_density"`
	WhitespaceRatio float64 `json:"whitespace_ratio"`
}

// Default configurations

// DefaultClassificationConfig returns a default classification configuration
func DefaultClassificationConfig() ClassificationConfig {
	return ClassificationConfig{
		MinConfidenceThreshold: 0.7,
		MaxAlternatives:        3,
		EnableMLModel:          false,
		EnableCustomRules:      true,
		RuleWeightingStrategy:  "confidence_based",
		EnableDeepAnalysis:     true,
		MaxContentLength:       50000,
		KeywordCaseSensitive:   false,
		MaxProcessingTime:      5000, // 5 seconds
		CacheClassifications:   true,
		CacheExpiryMinutes:     60,
		EnableDebugMode:        false,
		LogClassificationSteps: false,
		ReturnDetailedMetrics:  true,
	}
}

// DocumentTypeDisplayName returns a human-readable name for a document type
func (dt DocumentType) DisplayName() string {
	switch dt {
	case DocumentTypeInvoice:
		return "Invoice"
	case DocumentTypeReport:
		return "Report"
	case DocumentTypeForm:
		return "Form"
	case DocumentTypeContract:
		return "Contract"
	case DocumentTypeAcademic:
		return "Academic Paper"
	case DocumentTypeManual:
		return "Manual/Guide"
	case DocumentTypeLetter:
		return "Letter"
	case DocumentTypeBrochure:
		return "Brochure"
	case DocumentTypeTechnical:
		return "Technical Document"
	case DocumentTypeFinancial:
		return "Financial Statement"
	case DocumentTypeLegal:
		return "Legal Document"
	case DocumentTypeResume:
		return "Resume/CV"
	case DocumentTypePresentation:
		return "Presentation"
	case DocumentTypeNewsletter:
		return "Newsletter"
	case DocumentTypeCatalog:
		return "Catalog"
	default:
		return "Unknown"
	}
}

// IsValid checks if the document type is valid
func (dt DocumentType) IsValid() bool {
	switch dt {
	case DocumentTypeUnknown, DocumentTypeInvoice, DocumentTypeReport, DocumentTypeForm,
		DocumentTypeContract, DocumentTypeAcademic, DocumentTypeManual, DocumentTypeLetter,
		DocumentTypeBrochure, DocumentTypeTechnical, DocumentTypeFinancial, DocumentTypeLegal,
		DocumentTypeResume, DocumentTypePresentation, DocumentTypeNewsletter, DocumentTypeCatalog:
		return true
	default:
		return false
	}
}

// AllDocumentTypes returns all valid document types
func AllDocumentTypes() []DocumentType {
	return []DocumentType{
		DocumentTypeInvoice,
		DocumentTypeReport,
		DocumentTypeForm,
		DocumentTypeContract,
		DocumentTypeAcademic,
		DocumentTypeManual,
		DocumentTypeLetter,
		DocumentTypeBrochure,
		DocumentTypeTechnical,
		DocumentTypeFinancial,
		DocumentTypeLegal,
		DocumentTypeResume,
		DocumentTypePresentation,
		DocumentTypeNewsletter,
		DocumentTypeCatalog,
	}
}
