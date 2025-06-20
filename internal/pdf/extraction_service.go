package pdf

import (
	"fmt"
	"os"
)

// ExtractionService provides enhanced PDF content extraction capabilities
type ExtractionService struct {
	maxFileSize int64
}

// NewExtractionService creates a new extraction service
func NewExtractionService(maxFileSize int64) *ExtractionService {
	return &ExtractionService{
		maxFileSize: maxFileSize,
	}
}

// Tool request/response types for MCP protocol

// PDFExtractRequest represents a request for structured content extraction
type PDFExtractRequest struct {
	Path   string        `json:"path"`
	Mode   string        `json:"mode,omitempty"`
	Config ExtractConfig `json:"config,omitempty"`
	Query  *ContentQuery `json:"query,omitempty"`
}

// ExtractConfig provides simplified configuration for MCP tools
type ExtractConfig struct {
	ExtractText        bool    `json:"extract_text,omitempty"`
	ExtractImages      bool    `json:"extract_images,omitempty"`
	ExtractTables      bool    `json:"extract_tables,omitempty"`
	ExtractForms       bool    `json:"extract_forms,omitempty"`
	ExtractAnnotations bool    `json:"extract_annotations,omitempty"`
	IncludeCoordinates bool    `json:"include_coordinates,omitempty"`
	IncludeFormatting  bool    `json:"include_formatting,omitempty"`
	Pages              []int   `json:"pages,omitempty"`
	MinConfidence      float64 `json:"min_confidence,omitempty"`
}

// PDFQueryRequest represents a request to query extracted content
type PDFQueryRequest struct {
	Path  string       `json:"path"`
	Query ContentQuery `json:"query"`
}

// ExtractStructured performs structured content extraction with positioning and formatting
func (s *ExtractionService) ExtractStructured(req PDFExtractRequest) (*PDFExtractResult, error) {
	if err := s.validatePath(req.Path); err != nil {
		return nil, err
	}

	// Set default mode if not specified
	mode := req.Mode
	if mode == "" {
		mode = "structured"
	}

	// For now, return a placeholder result
	// TODO: Implement actual structured extraction
	return &PDFExtractResult{
		FilePath:       req.Path,
		Mode:           mode,
		TotalPages:     1,
		ProcessedPages: []int{1},
		Elements:       []ContentElement{},
		Tables:         []TableElement{},
		Summary: ExtractionSummary{
			ContentTypes:  make(map[string]int),
			TotalElements: 0,
			Quality:       "medium",
		},
		Metadata: DocumentMetadata{},
		Warnings: []string{"Structured extraction not yet fully implemented"},
	}, nil
}

// ExtractTables performs table detection and extraction
func (s *ExtractionService) ExtractTables(req PDFExtractRequest) (*PDFExtractResult, error) {
	if err := s.validatePath(req.Path); err != nil {
		return nil, err
	}

	// Force table mode
	req.Mode = "table"
	req.Config.ExtractTables = true
	req.Config.ExtractText = true // Need text for table detection

	return s.ExtractStructured(req)
}

// ExtractSemantic performs semantic content grouping
func (s *ExtractionService) ExtractSemantic(req PDFExtractRequest) (*PDFExtractResult, error) {
	if err := s.validatePath(req.Path); err != nil {
		return nil, err
	}

	// Force semantic mode
	req.Mode = "semantic"
	req.Config.ExtractText = true
	req.Config.IncludeCoordinates = true
	req.Config.IncludeFormatting = true

	return s.ExtractStructured(req)
}

// ExtractComplete performs comprehensive extraction of all content types
func (s *ExtractionService) ExtractComplete(req PDFExtractRequest) (*PDFExtractResult, error) {
	if err := s.validatePath(req.Path); err != nil {
		return nil, err
	}

	// Force complete mode with all extraction types enabled
	req.Mode = "complete"
	req.Config.ExtractText = true
	req.Config.ExtractImages = true
	req.Config.ExtractTables = true
	req.Config.ExtractForms = true
	req.Config.ExtractAnnotations = true
	req.Config.IncludeCoordinates = true
	req.Config.IncludeFormatting = true

	return s.ExtractStructured(req)
}

// QueryContent searches extracted content using the provided query
func (s *ExtractionService) QueryContent(req PDFQueryRequest) (*PDFQueryResult, error) {
	if err := s.validatePath(req.Path); err != nil {
		return nil, err
	}

	// First extract content in structured mode
	extractReq := PDFExtractRequest{
		Path: req.Path,
		Mode: "structured",
		Config: ExtractConfig{
			ExtractText:        true,
			ExtractImages:      true,
			ExtractTables:      true,
			ExtractForms:       true,
			ExtractAnnotations: true,
			IncludeCoordinates: true,
			IncludeFormatting:  true,
		},
	}

	extractResult, err := s.ExtractStructured(extractReq)
	if err != nil {
		return nil, fmt.Errorf("failed to extract content for querying: %w", err)
	}

	// For now, return all elements (query filtering not yet implemented)
	// TODO: Implement actual query filtering
	result := &PDFQueryResult{
		FilePath:   req.Path,
		Query:      req.Query,
		MatchCount: len(extractResult.Elements),
		Elements:   extractResult.Elements,
		Summary:    s.buildQuerySummary(extractResult.Elements),
	}

	return result, nil
}

// GetPageInfo returns detailed page information
func (s *ExtractionService) GetPageInfo(path string) ([]PageInfo, error) {
	if err := s.validatePath(path); err != nil {
		return nil, err
	}

	// TODO: Implement actual page info extraction
	return []PageInfo{}, nil
}

// GetMetadata extracts comprehensive document metadata
func (s *ExtractionService) GetMetadata(path string) (*DocumentMetadata, error) {
	if err := s.validatePath(path); err != nil {
		return nil, err
	}

	// TODO: Implement actual metadata extraction
	return &DocumentMetadata{}, nil
}

// Helper methods

func (s *ExtractionService) validatePath(path string) error {
	if path == "" {
		return fmt.Errorf("path cannot be empty")
	}

	fileInfo, err := os.Stat(path)
	if os.IsNotExist(err) {
		return fmt.Errorf("file does not exist: %s", path)
	}
	if err != nil {
		return fmt.Errorf("cannot access file: %w", err)
	}

	if fileInfo.IsDir() {
		return fmt.Errorf("path is a directory, not a file: %s", path)
	}

	if fileInfo.Size() > s.maxFileSize {
		return fmt.Errorf("file too large: %d bytes (max: %d bytes)", fileInfo.Size(), s.maxFileSize)
	}

	return nil
}

func (s *ExtractionService) buildQuerySummary(elements []ContentElement) QuerySummary {
	typeBreakdown := make(map[string]int)
	pageBreakdown := make(map[int]int)
	totalConfidence := 0.0

	for _, element := range elements {
		typeBreakdown[string(element.Type)]++
		pageBreakdown[element.PageNumber]++
		totalConfidence += element.Confidence
	}

	avgConfidence := 0.0
	if len(elements) > 0 {
		avgConfidence = totalConfidence / float64(len(elements))
	}

	return QuerySummary{
		TypeBreakdown: typeBreakdown,
		PageBreakdown: pageBreakdown,
		Confidence:    avgConfidence,
	}
}
