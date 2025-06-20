package pdf

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/a3tai/mcp-pdf-reader/internal/pdf/extraction"
	"github.com/ledongthuc/pdf"
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

	// Open PDF file
	f, r, err := pdf.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open PDF: %w", err)
	}
	defer f.Close()

	// Get total page count
	pageCount := r.NumPage()
	if pageCount == 0 {
		return []PageInfo{}, nil
	}

	// Extract page information for each page
	pages := make([]PageInfo, 0, pageCount)

	for i := 1; i <= pageCount; i++ {
		page := r.Page(i)
		if page.V.IsNull() {
			// Skip invalid pages but log the issue
			continue
		}

		pageInfo := PageInfo{
			Number: i,
		}

		// Extract MediaBox (required) - try direct extraction first
		if mediaBox := page.V.Key("MediaBox"); !mediaBox.IsNull() {
			if mediaBoxRect, err := s.extractRectangle(mediaBox); err == nil {
				pageInfo.MediaBox = mediaBoxRect
				pageInfo.Width = mediaBoxRect.Width
				pageInfo.Height = mediaBoxRect.Height
			} else {
				// Log the specific error for debugging
				fmt.Fprintf(os.Stderr, "[MediaBox] Failed to extract MediaBox from page %d: %v\n", i+1, err)

				// Fallback to inherited MediaBox
				if inheritedMediaBox := s.getInheritedMediaBox(page); inheritedMediaBox != nil {
					pageInfo.MediaBox = *inheritedMediaBox
					pageInfo.Width = inheritedMediaBox.Width
					pageInfo.Height = inheritedMediaBox.Height
					fmt.Fprintf(os.Stderr, "[MediaBox] Using inherited MediaBox for page %d: %.2fx%.2f\n",
						i+1, inheritedMediaBox.Width, inheritedMediaBox.Height)
				} else {
					// Use default Letter size if all extraction fails
					pageInfo.Width = 612.0
					pageInfo.Height = 792.0
					pageInfo.MediaBox = Rectangle{X: 0, Y: 0, Width: 612.0, Height: 792.0}
					fmt.Fprintf(os.Stderr, "[MediaBox] Using default dimensions for page %d\n", i+1)
				}
			}
		} else {
			// Check inherited MediaBox from parent Pages node
			if inheritedMediaBox := s.getInheritedMediaBox(page); inheritedMediaBox != nil {
				pageInfo.MediaBox = *inheritedMediaBox
				pageInfo.Width = inheritedMediaBox.Width
				pageInfo.Height = inheritedMediaBox.Height
				fmt.Fprintf(os.Stderr, "[MediaBox] Using inherited MediaBox for page %d: %.2fx%.2f\n",
					i+1, inheritedMediaBox.Width, inheritedMediaBox.Height)
			} else {
				// Default to Letter size
				pageInfo.Width = 612.0
				pageInfo.Height = 792.0
				pageInfo.MediaBox = Rectangle{X: 0, Y: 0, Width: 612.0, Height: 792.0}
				fmt.Fprintf(os.Stderr, "[MediaBox] Using default dimensions for page %d\n", i+1)
			}
		}

		// Extract CropBox (optional)
		if cropBox := page.V.Key("CropBox"); !cropBox.IsNull() {
			if cropBoxRect, err := s.extractRectangle(cropBox); err == nil {
				pageInfo.CropBox = cropBoxRect
			}
		}

		// Extract Rotation (optional)
		if rotation := page.V.Key("Rotate"); !rotation.IsNull() {
			if rotVal := rotation.Float64(); rotVal != 0 {
				pageInfo.Rotation = int(rotVal)
			}
		}

		pages = append(pages, pageInfo)
	}

	return pages, nil
}

// extractRectangle extracts a rectangle from a PDF Value (MediaBox, CropBox, etc.)
func (s *ExtractionService) extractRectangle(rectValue pdf.Value) (Rectangle, error) {
	// Handle null values
	if rectValue.IsNull() {
		return Rectangle{}, fmt.Errorf("rectangle value is null")
	}

	// Handle array format: [x1 y1 x2 y2]
	if rectValue.Kind() == pdf.Array {
		arr := rectValue
		if arr.Len() != 4 {
			return Rectangle{}, fmt.Errorf("invalid rectangle array length: %d, expected 4", arr.Len())
		}

		coords := make([]float64, 4)
		for i := 0; i < 4; i++ {
			val := arr.Index(i)
			if val.IsNull() {
				return Rectangle{}, fmt.Errorf("coordinate at index %d is null", i)
			}

			// Handle different numeric types
			switch val.Kind() {
			case pdf.Integer:
				coords[i] = float64(val.Int64())
			case pdf.Real:
				coords[i] = val.Float64()
			default:
				return Rectangle{}, fmt.Errorf("invalid coordinate type at index %d: %v", i, val.Kind())
			}
		}

		x1, y1, x2, y2 := coords[0], coords[1], coords[2], coords[3]

		// Validate rectangle dimensions
		if x2 <= x1 || y2 <= y1 {
			return Rectangle{}, fmt.Errorf("invalid rectangle dimensions: [%.2f %.2f %.2f %.2f]", x1, y1, x2, y2)
		}

		// Calculate width and height from coordinates
		width := x2 - x1
		height := y2 - y1

		return Rectangle{
			X:      x1,
			Y:      y1,
			Width:  width,
			Height: height,
		}, nil
	}

	// Handle indirect object references
	if rectValue.Kind() == pdf.Dict {
		// This might be an indirect reference - try to resolve it
		return Rectangle{}, fmt.Errorf("indirect object references not yet supported for rectangles")
	}

	return Rectangle{}, fmt.Errorf("unsupported rectangle format: %v", rectValue.Kind())
}

// getInheritedMediaBox traverses up the page tree to find an inherited MediaBox
func (s *ExtractionService) getInheritedMediaBox(page pdf.Page) *Rectangle {
	// Try to find MediaBox in parent pages
	current := page.V

	// Look for Parent reference and traverse up the tree
	for i := 0; i < 10; i++ { // Limit iterations to prevent infinite loops
		parent := current.Key("Parent")
		if parent.IsNull() {
			break
		}

		// Check if parent has MediaBox
		if mediaBox := parent.Key("MediaBox"); !mediaBox.IsNull() {
			if rect, err := s.extractRectangle(mediaBox); err == nil {
				return &rect
			}
		}

		// Move up to the next parent
		current = parent
	}

	// If no inherited MediaBox found, return default US Letter size
	return &Rectangle{
		X:      0,
		Y:      0,
		Width:  612.0, // 8.5 inches * 72 DPI
		Height: 792.0, // 11 inches * 72 DPI
	}
}

// GetMetadata extracts comprehensive document metadata
func (s *ExtractionService) GetMetadata(path string) (*DocumentMetadata, error) {
	if err := s.validatePath(path); err != nil {
		return nil, err
	}

	// Open and parse PDF for metadata
	f, r, err := pdf.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open PDF: %w", err)
	}
	defer f.Close()

	metadata := &DocumentMetadata{}

	// Extract metadata safely with panic recovery
	s.extractMetadata(r, metadata)

	return metadata, nil
}

// extractMetadata safely extracts metadata from PDF reader
func (s *ExtractionService) extractMetadata(r *pdf.Reader, metadata *DocumentMetadata) {
	// Safely extract metadata using the PDF library's API
	// The ledongthuc/pdf library requires careful handling of Value types

	defer func() {
		// Recover from any panics during metadata extraction
		if recover() != nil {
			// Metadata extraction failed, but we can continue with basic metadata
		}
	}()

	// Try to get document info
	trailer := r.Trailer()
	if trailer.IsNull() {
		return
	}

	info := trailer.Key("Info")
	if info.IsNull() {
		return
	}

	// Extract title
	if title := info.Key("Title"); !title.IsNull() {
		if titleStr := title.String(); titleStr != "" {
			metadata.Title = strings.TrimSpace(titleStr)
		}
	}

	// Extract author
	if author := info.Key("Author"); !author.IsNull() {
		if authorStr := author.String(); authorStr != "" {
			metadata.Author = strings.TrimSpace(authorStr)
		}
	}

	// Extract subject
	if subject := info.Key("Subject"); !subject.IsNull() {
		if subjectStr := subject.String(); subjectStr != "" {
			metadata.Subject = strings.TrimSpace(subjectStr)
		}
	}

	// Extract creator
	if creator := info.Key("Creator"); !creator.IsNull() {
		if creatorStr := creator.String(); creatorStr != "" {
			metadata.Creator = strings.TrimSpace(creatorStr)
		}
	}

	// Extract producer
	if producer := info.Key("Producer"); !producer.IsNull() {
		if producerStr := producer.String(); producerStr != "" {
			metadata.Producer = strings.TrimSpace(producerStr)
		}
	}

	// Extract creation date
	if creationDate := info.Key("CreationDate"); !creationDate.IsNull() {
		if dateStr := creationDate.String(); dateStr != "" {
			metadata.CreationDate = strings.TrimSpace(dateStr)
		}
	}

	// Extract modification date
	if modDate := info.Key("ModDate"); !modDate.IsNull() {
		if dateStr := modDate.String(); dateStr != "" {
			metadata.ModificationDate = strings.TrimSpace(dateStr)
		}
	}

	// Extract keywords
	if keywords := info.Key("Keywords"); !keywords.IsNull() {
		if keywordsStr := keywords.String(); keywordsStr != "" {
			// Split keywords by common separators (comma, semicolon, space)
			keywordList := strings.FieldsFunc(strings.TrimSpace(keywordsStr), func(r rune) bool {
				return r == ',' || r == ';' || r == ' '
			})

			// Clean and filter empty keywords
			var cleanKeywords []string
			for _, kw := range keywordList {
				if trimmed := strings.TrimSpace(kw); trimmed != "" {
					cleanKeywords = append(cleanKeywords, trimmed)
				}
			}
			if len(cleanKeywords) > 0 {
				metadata.Keywords = cleanKeywords
			}
		}
	}

	// Try to extract additional metadata from catalog
	s.extractCatalogMetadata(r, metadata)

	// Check if document is encrypted (ledongthuc/pdf has limited encryption support)
	metadata.Encrypted = false
}

// extractCatalogMetadata extracts additional metadata from document catalog
func (s *ExtractionService) extractCatalogMetadata(r *pdf.Reader, metadata *DocumentMetadata) {
	defer func() {
		// Recover from any panics during catalog metadata extraction
		if recover() != nil {
			// Catalog metadata extraction failed, but we can continue
		}
	}()

	// Try to get catalog
	trailer := r.Trailer()
	if trailer.IsNull() {
		return
	}

	root := trailer.Key("Root")
	if root.IsNull() {
		return
	}

	// Extract page layout
	if pageLayout := root.Key("PageLayout"); !pageLayout.IsNull() {
		if layoutStr := pageLayout.String(); layoutStr != "" {
			metadata.PageLayout = strings.TrimSpace(layoutStr)
		}
	}

	// Extract page mode
	if pageMode := root.Key("PageMode"); !pageMode.IsNull() {
		if modeStr := pageMode.String(); modeStr != "" {
			metadata.PageMode = strings.TrimSpace(modeStr)
		}
	}

	// Extract version from header if available
	// ledongthuc/pdf doesn't expose version directly, so we'll use a default
	metadata.Version = "1.4"
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
