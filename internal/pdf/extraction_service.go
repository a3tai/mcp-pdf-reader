package pdf

import (
	"fmt"
	"os"
	"time"

	"github.com/a3tai/mcp-pdf-reader/internal/pdf/extraction"
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

	// Create extraction engine
	engine := extraction.NewEngine()

	// Convert config
	extractConfig := extraction.ExtractionConfig{
		Mode:               extraction.ExtractionMode(mode),
		ExtractText:        req.Config.ExtractText,
		ExtractImages:      req.Config.ExtractImages,
		ExtractTables:      req.Config.ExtractTables,
		ExtractForms:       req.Config.ExtractForms,
		ExtractAnnotations: req.Config.ExtractAnnotations,
		IncludeCoordinates: req.Config.IncludeCoordinates,
		PreserveFormatting: req.Config.IncludeFormatting,
		Pages:              req.Config.Pages,
	}

	// Create extraction request
	extractReq := extraction.ExtractionRequest{
		FilePath: req.Path,
		Config:   extractConfig,
	}

	// Add query if provided
	if req.Query != nil {
		extractReq.Query = &extraction.Query{
			ContentTypes:  convertContentTypes(req.Query.ContentTypes),
			Pages:         req.Query.Pages,
			TextQuery:     req.Query.TextQuery,
			MinConfidence: req.Query.MinConfidence,
		}

		// Convert bounding box if provided
		if req.Query.BoundingBox != nil {
			extractReq.Query.BoundingBox = &extraction.BoundingBox{
				LowerLeft: extraction.Coordinate{
					X: req.Query.BoundingBox.X,
					Y: req.Query.BoundingBox.Y,
				},
				UpperRight: extraction.Coordinate{
					X: req.Query.BoundingBox.X + req.Query.BoundingBox.Width,
					Y: req.Query.BoundingBox.Y + req.Query.BoundingBox.Height,
				},
				Width:  req.Query.BoundingBox.Width,
				Height: req.Query.BoundingBox.Height,
			}
		}
	}

	// Perform extraction
	result, err := engine.Extract(extractReq)
	if err != nil {
		return nil, fmt.Errorf("extraction failed: %w", err)
	}

	// Convert extraction result to PDFExtractResult
	return s.convertExtractionResult(result, req.Path, mode), nil
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

// ExtractForms performs form field extraction
func (s *ExtractionService) ExtractForms(req PDFExtractRequest) (*PDFExtractResult, error) {
	if err := s.validatePath(req.Path); err != nil {
		return nil, err
	}

	// Force form mode
	req.Mode = "form"
	req.Config.ExtractForms = true
	req.Config.IncludeCoordinates = true

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

// convertExtractionResult converts internal extraction result to API format
func (s *ExtractionService) convertExtractionResult(result *extraction.ExtractionResult, filePath, mode string) *PDFExtractResult {
	// Convert elements
	elements := make([]ContentElement, len(result.Elements))
	for i, elem := range result.Elements {
		elements[i] = ContentElement{
			Type:       string(elem.Type),
			Content:    elem.Content,
			PageNumber: elem.PageNumber,
			BoundingBox: Rectangle{
				X:      elem.BoundingBox.LowerLeft.X,
				Y:      elem.BoundingBox.LowerLeft.Y,
				Width:  elem.BoundingBox.Width,
				Height: elem.BoundingBox.Height,
			},
			Confidence: elem.Confidence,
		}
	}

	// Convert tables - for now, tables are extracted as structured text elements
	tables := make([]TableElement, 0)

	// Build summary
	contentTypes := make(map[string]int)
	if result.ExtractionInfo.ElementCounts.Text > 0 {
		contentTypes["text"] = result.ExtractionInfo.ElementCounts.Text
	}
	if result.ExtractionInfo.ElementCounts.Images > 0 {
		contentTypes["image"] = result.ExtractionInfo.ElementCounts.Images
	}
	if result.ExtractionInfo.ElementCounts.Forms > 0 {
		contentTypes["form"] = result.ExtractionInfo.ElementCounts.Forms
	}
	if result.ExtractionInfo.ElementCounts.Tables > 0 {
		contentTypes["table"] = result.ExtractionInfo.ElementCounts.Tables
	}
	if result.ExtractionInfo.ElementCounts.Annotations > 0 {
		contentTypes["annotation"] = result.ExtractionInfo.ElementCounts.Annotations
	}

	// Determine quality based on errors/warnings
	quality := "high"
	if len(result.Errors) > 0 {
		quality = "low"
	} else if len(result.Warnings) > 0 {
		quality = "medium"
	}

	return &PDFExtractResult{
		FilePath:       filePath,
		Mode:           mode,
		TotalPages:     result.TotalPages,
		ProcessedPages: result.ProcessedPages,
		Elements:       elements,
		Tables:         tables,
		Summary: ExtractionSummary{
			ContentTypes:  contentTypes,
			TotalElements: len(elements),
			Quality:       quality,
		},
		Metadata: DocumentMetadata{
			Title:            result.Metadata.Title,
			Author:           result.Metadata.Author,
			Subject:          result.Metadata.Subject,
			Creator:          result.Metadata.Creator,
			Producer:         result.Metadata.Producer,
			Keywords:         result.Metadata.Keywords,
			CreationDate:     result.Metadata.CreationDate.Format(time.RFC3339),
			ModificationDate: result.Metadata.ModificationDate.Format(time.RFC3339),
			PageLayout:       result.Metadata.PageLayout,
			PageMode:         result.Metadata.PageMode,
			Version:          result.Metadata.Version,
			Encrypted:        result.Metadata.Encrypted,
			CustomProperties: result.Metadata.CustomProperties,
		},
		Warnings: result.Warnings,
		Errors:   result.Errors,
	}
}

// convertContentTypes converts string content types to extraction.ContentType
func convertContentTypes(types []string) []extraction.ContentType {
	if len(types) == 0 {
		return nil
	}

	result := make([]extraction.ContentType, len(types))
	for i, t := range types {
		result[i] = extraction.ContentType(t)
	}
	return result
}
