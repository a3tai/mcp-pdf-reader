package extraction

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/a3tai/mcp-pdf-reader/internal/pdf/errors"
	"github.com/ledongthuc/pdf"
)

// Constants for PDF processing
const (
	defaultTableDetectionThreshold = 0.7
	defaultConfidenceThreshold     = 0.8
	estimatedConfidenceThreshold   = 0.7
	minimumConfidenceThreshold     = 0.5

	// Default page dimensions and spacing
	defaultLineHeight   = 12.0
	defaultFontSize     = 12.0
	defaultLeftMargin   = 72.0
	defaultRightMargin  = 540.0
	defaultPageWidth    = 468.0
	defaultTopMargin    = 720.0
	defaultBottomMargin = 732.0

	// Table detection constants
	minTableElements   = 4
	rowTolerance       = 5.0
	proximityThreshold = 20.0

	// Limits
	minRowsForTable = 2
)

// Engine defines the interface for PDF content extraction
type Engine interface {
	// Extract performs content extraction based on the provided request
	Extract(req ExtractionRequest) (*ExtractionResult, error)

	// Query searches extracted content using the provided query
	Query(elements []ContentElement, query Query) ([]ContentElement, error)

	// GetMetadata extracts document metadata
	GetMetadata(filePath string) (*PDFMetadata, error)

	// GetPageInfo returns information about PDF pages
	GetPageInfo(filePath string) ([]PageInfo, error)
}

// PageInfo represents information about a single PDF page
type PageInfo struct {
	Number   int         `json:"number"`
	Width    float64     `json:"width"`
	Height   float64     `json:"height"`
	Rotation int         `json:"rotation"`
	MediaBox BoundingBox `json:"media_box"`
	CropBox  BoundingBox `json:"crop_box,omitempty"`
	BleedBox BoundingBox `json:"bleed_box,omitempty"`
	TrimBox  BoundingBox `json:"trim_box,omitempty"`
	ArtBox   BoundingBox `json:"art_box,omitempty"`
}

// DefaultEngine implements the Engine interface
type DefaultEngine struct {
	maxFileSize      int64
	maxTextSize      int
	ocrEnabled       bool
	tableDetectionTh float64
	debugMode        bool
	pdfReader        *pdf.Reader
	filePath         string
}

// NewEngine creates a new extraction engine with default settings
func NewEngine() *DefaultEngine {
	return &DefaultEngine{
		maxFileSize:      100 * 1024 * 1024, // 100MB
		maxTextSize:      50 * 1024 * 1024,  // 50MB
		ocrEnabled:       false,
		tableDetectionTh: defaultTableDetectionThreshold,
		debugMode:        false,
	}
}

// NewEngineWithConfig creates a new extraction engine with custom configuration
func NewEngineWithConfig(maxFileSize, maxTextSize int64, ocrEnabled bool) *DefaultEngine {
	return &DefaultEngine{
		maxFileSize:      maxFileSize,
		maxTextSize:      int(maxTextSize),
		ocrEnabled:       ocrEnabled,
		tableDetectionTh: defaultTableDetectionThreshold,
		debugMode:        false,
	}
}

// Extract performs comprehensive content extraction from a PDF with robust error handling
func (e *DefaultEngine) Extract(req ExtractionRequest) (*ExtractionResult, error) {
	startTime := time.Now()

	// Validate request
	if err := e.validateRequest(req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// Use robust parser for better error handling
	robustParser := errors.NewRobustParser()
	robustParser.EnableDebugLogging(true)

	// Attempt robust parsing
	parseResult, err := robustParser.ParseFile(req.FilePath)
	if err != nil && !parseResult.Success {
		return nil, fmt.Errorf("failed to open PDF with robust parser: %w", err)
	}

	f := parseResult.File
	pdfReader := parseResult.Reader
	defer func() {
		if f != nil {
			f.Close()
		}
		robustParser.Close()
	}()

	// Store pdfReader for form extraction
	e.pdfReader = pdfReader
	e.filePath = req.FilePath

	// Initialize result
	result := &ExtractionResult{
		FilePath:       req.FilePath,
		TotalPages:     pdfReader.NumPage(),
		ProcessedPages: []int{},
		Elements:       []ContentElement{},
		Tables:         []TableElement{},
		Warnings:       []string{},
		Errors:         []string{},
		ExtractionInfo: ExtractionInfo{
			Mode:            req.Config.Mode,
			StartTime:       startTime,
			ElementCounts:   ElementCounts{},
			ProcessingStats: ProcessingStats{},
		},
	}

	// Extract metadata
	metadata, err := e.extractMetadata(pdfReader)
	if err != nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("metadata extraction failed: %v", err))
	} else {
		result.Metadata = *metadata
	}

	// Determine pages to process
	pagesToProcess := e.determinePagesToProcess(req.Config.Pages, pdfReader.NumPage())
	result.ProcessedPages = pagesToProcess

	// Extract content from each page with enhanced error handling
	for _, pageNum := range pagesToProcess {
		pageElements, pageErrors := e.extractPageContentWithRecovery(pdfReader, pageNum, req.Config, req.FilePath)
		result.Elements = append(result.Elements, pageElements...)

		if len(pageErrors) > 0 {
			for _, err := range pageErrors {
				result.Errors = append(result.Errors, fmt.Sprintf("page %d: %v", pageNum, err))
			}
		}
	}

	// Add warnings from robust parser
	if parseResult.Errors != nil {
		errorCount, warningCount := parseResult.Errors.Count()
		if errorCount > 0 || warningCount > 0 {
			result.Warnings = append(result.Warnings, fmt.Sprintf("PDF parsing encountered %d errors and %d warnings", errorCount, warningCount))
		}
	}

	// Post-process content based on mode
	if err := e.postProcessContent(result, req.Config); err != nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("post-processing failed: %v", err))
	}

	// Apply query filter if provided
	if req.Query != nil {
		filteredElements, err := e.Query(result.Elements, *req.Query)
		if err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("query filter failed: %v", err))
		} else {
			result.Elements = filteredElements
		}
	}

	// Finalize extraction info
	endTime := time.Now()
	result.ExtractionInfo.EndTime = endTime
	result.ExtractionInfo.Duration = endTime.Sub(startTime)
	result.ExtractionInfo.ElementCounts = e.countElements(result.Elements)

	return result, nil
}

// extractPageContentWithRecovery extracts all content from a single page with enhanced error handling
func (e *DefaultEngine) extractPageContentWithRecovery(
	pdfReader *pdf.Reader, pageNum int, config ExtractionConfig, filePath string,
) ([]ContentElement, []error) {
	// Enhanced panic recovery with detailed logging
	defer func() {
		if r := recover(); r != nil {
			// Log the panic for debugging
			fmt.Fprintf(os.Stderr, "[ExtractEngine] PANIC on page %d of %s: %v\n", pageNum, filePath, r)
		}
	}()

	// Delegate to original method for backward compatibility
	return e.extractPageContent(pdfReader, pageNum, config)
}

// extractPageContent extracts all content from a single page (original method)
func (e *DefaultEngine) extractPageContent(
	pdfReader *pdf.Reader, pageNum int, config ExtractionConfig,
) ([]ContentElement, []error) {
	var elements []ContentElement
	var errors []error

	// Add panic recovery for malformed PDF streams
	defer func() {
		if r := recover(); r != nil {
			errors = append(errors, fmt.Errorf("panic during page content extraction on page %d: %v", pageNum, r))
		}
	}()

	page := pdfReader.Page(pageNum)
	if page.V.IsNull() {
		return elements, []error{fmt.Errorf("invalid page %d", pageNum)}
	}

	// Get page dimensions (for future use in coordinate calculations)
	_, err := e.getPageInfo(page, pageNum)
	if err != nil {
		errors = append(errors, fmt.Errorf("failed to get page info: %w", err))
		// Continue with default dimensions
	}

	// Extract text content
	if config.ExtractText {
		textElements, textErrors := e.extractTextFromPage(page, pageNum, config)
		elements = append(elements, textElements...)
		errors = append(errors, textErrors...)
	}

	// Extract images
	if config.ExtractImages {
		imageElements, imageErrors := e.extractImagesFromPage(page, pageNum, config)
		elements = append(elements, imageElements...)
		errors = append(errors, imageErrors...)
	}

	// Extract vector graphics
	if config.ExtractVectors {
		vectorElements, vectorErrors := e.extractVectorsFromPage(page, pageNum, config)
		elements = append(elements, vectorElements...)
		errors = append(errors, vectorErrors...)
	}

	// Extract form fields
	if config.ExtractForms {
		formElements, formErrors := e.extractFormsFromPage(page, pageNum, config)
		elements = append(elements, formElements...)
		errors = append(errors, formErrors...)
	}

	// Extract annotations
	if config.ExtractAnnotations {
		annotationElements, annotErrors := e.extractAnnotationsFromPage(page, pageNum, config)
		elements = append(elements, annotationElements...)
		errors = append(errors, annotErrors...)
	}

	return elements, errors
}

// extractTextFromPage extracts text content with positioning and formatting
func (e *DefaultEngine) extractTextFromPage(
	page pdf.Page, pageNum int, config ExtractionConfig,
) ([]ContentElement, []error) {
	var elements []ContentElement
	var errors []error

	// Get basic text content
	textContent, err := page.GetPlainText(nil)
	if err != nil {
		errors = append(errors, fmt.Errorf("failed to extract text: %w", err))
		return elements, errors
	}

	if strings.TrimSpace(textContent) == "" {
		return elements, errors
	}

	// Create basic text element
	textElement := ContentElement{
		ID:         e.generateID("text", pageNum, 0),
		Type:       ContentTypeText,
		PageNumber: pageNum,
		Content: TextElement{
			Text:       textContent,
			Properties: TextProperties{},
		},
		Confidence: 1.0,
	}

	// If structured mode, try to extract positioning and formatting
	if config.Mode == ModeStructured || config.Mode == ModeComplete {
		if structuredElements, err := e.extractStructuredText(page, pageNum, config); err != nil {
			errors = append(errors, fmt.Errorf("structured text extraction failed: %w", err))
			elements = append(elements, textElement) // Fallback to basic text
		} else {
			elements = append(elements, structuredElements...)
		}
	} else {
		elements = append(elements, textElement)
	}

	return elements, errors
}

// extractStructuredText attempts to extract text with positioning and formatting
func (e *DefaultEngine) extractStructuredText(
	page pdf.Page, pageNum int, config ExtractionConfig,
) ([]ContentElement, error) {
	var elements []ContentElement

	// This is a simplified implementation - in practice, you would parse
	// the page's content stream to get detailed positioning and formatting

	// Get text content and create word-level elements if possible
	textContent, err := page.GetPlainText(nil)
	if err != nil {
		return nil, err
	}

	// Split into lines and words for basic structure
	lines := strings.Split(textContent, "\n")

	for lineIdx, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}

		// Create line element
		lineElement := ContentElement{
			ID:         e.generateID("line", pageNum, lineIdx),
			Type:       ContentTypeText,
			PageNumber: pageNum,
			BoundingBox: BoundingBox{
				LowerLeft:  Coordinate{X: defaultLeftMargin, Y: defaultTopMargin - float64(lineIdx)*defaultLineHeight},
				UpperRight: Coordinate{X: defaultRightMargin, Y: defaultBottomMargin - float64(lineIdx)*defaultLineHeight},
				Width:      defaultPageWidth,
				Height:     defaultLineHeight,
			},
			Content: TextElement{
				Text: line,
				Properties: TextProperties{
					FontSize: defaultFontSize,
				},
			},
			Confidence: defaultConfidenceThreshold,
		}

		// Add word-level elements if requested
		if config.IncludeCoordinates {
			words := strings.Fields(line)
			wordWidth := defaultPageWidth / float64(len(words)) // Estimated word width

			for wordIdx, word := range words {
				wordElement := ContentElement{
					ID:         e.generateID("word", pageNum, lineIdx*1000+wordIdx),
					Type:       ContentTypeText,
					PageNumber: pageNum,
					BoundingBox: BoundingBox{
						LowerLeft: Coordinate{
							X: defaultLeftMargin + float64(wordIdx)*wordWidth,
							Y: defaultTopMargin - float64(lineIdx)*defaultLineHeight,
						},
						UpperRight: Coordinate{
							X: defaultLeftMargin + float64(wordIdx+1)*wordWidth,
							Y: defaultBottomMargin - float64(lineIdx)*defaultLineHeight,
						},
						Width:  wordWidth,
						Height: defaultLineHeight,
					},
					Content: TextElement{
						Text: word,
						Properties: TextProperties{
							FontSize: defaultFontSize,
						},
					},
					Parent:     &lineElement.ID,
					Confidence: estimatedConfidenceThreshold,
				}
				lineElement.Children = append(lineElement.Children, wordElement)
			}
		}

		elements = append(elements, lineElement)
	}

	return elements, nil
}

// extractImagesFromPage extracts image content from a page
func (e *DefaultEngine) extractImagesFromPage(
	page pdf.Page, pageNum int, config ExtractionConfig,
) ([]ContentElement, []error) {
	var elements []ContentElement
	var errors []error

	// Add panic recovery for malformed PDF streams
	defer func() {
		if r := recover(); r != nil {
			errors = append(errors, fmt.Errorf("panic during image extraction on page %d: %v", pageNum, r))
		}
	}()

	// Get page resources
	resources := page.V.Key("Resources")
	if resources.IsNull() {
		return elements, errors
	}

	// Get XObject dictionary
	xObjects := resources.Key("XObject")
	if xObjects.IsNull() || xObjects.Kind() != pdf.Dict {
		return elements, errors
	}

	imageIndex := 0
	for _, key := range xObjects.Keys() {
		// Process each XObject with individual error recovery
		func() {
			defer func() {
				if r := recover(); r != nil {
					errors = append(errors, fmt.Errorf("error processing XObject '%s' on page %d: %v", key, pageNum, r))
				}
			}()

			obj := xObjects.Key(key)
			if obj.IsNull() {
				return
			}

			// Check if this XObject is an image
			subtype := obj.Key("Subtype")
			if subtype.IsNull() || subtype.Name() != "Image" {
				return
			}

			// Extract image information with safe access
			width := int(obj.Key("Width").Int64())
			height := int(obj.Key("Height").Int64())

			// Skip invalid dimensions
			if width <= 0 || height <= 0 {
				errors = append(errors, fmt.Errorf("invalid image dimensions for XObject '%s': %dx%d", key, width, height))
				return
			}

			// Get color space
			colorSpace := "Unknown"
			if cs := obj.Key("ColorSpace"); !cs.IsNull() {
				if cs.Kind() == pdf.Name {
					colorSpace = cs.Name()
				}
			}

			// Get bits per component
			bitsPerComponent := int(obj.Key("BitsPerComponent").Int64())
			if bitsPerComponent == 0 {
				bitsPerComponent = 8 // Default
			}

			// Create image element
			// Note: Stream data extraction would require more complex PDF parsing
			var imageData []byte
			imageHash := e.generateHashFromData(imageData)

			imageElement := ContentElement{
				ID:         e.generateID("image", pageNum, imageIndex),
				Type:       ContentTypeImage,
				PageNumber: pageNum,
				BoundingBox: BoundingBox{
					// Position would need to be calculated from the transformation matrix
					// This is a simplified implementation
					LowerLeft:  Coordinate{X: 0, Y: 0},
					UpperRight: Coordinate{X: float64(width), Y: float64(height)},
					Width:      float64(width),
					Height:     float64(height),
				},
				Content: ImageElement{
					Format:           "Unknown", // Would need to be determined from the stream
					Width:            width,
					Height:           height,
					ColorSpace:       colorSpace,
					BitsPerComponent: bitsPerComponent,
					Data:             imageData,
					Hash:             imageHash,
					Size:             int64(len(imageData)),
				},
				Confidence: 1.0,
			}

			elements = append(elements, imageElement)
			imageIndex++
		}()
	}

	return elements, errors
}

// extractVectorsFromPage extracts vector graphics from a page
func (e *DefaultEngine) extractVectorsFromPage(
	page pdf.Page, pageNum int, config ExtractionConfig,
) ([]ContentElement, []error) {
	var elements []ContentElement
	var errors []error

	// Vector extraction would require parsing the page's content stream
	// This is a complex task that involves interpreting PDF graphics operators
	// For now, we'll return an empty result with a note

	if e.debugMode {
		errors = append(errors, fmt.Errorf("vector extraction not yet implemented - requires content stream parsing"))
	}

	return elements, errors
}

// extractFormsFromPage extracts form fields from a page
func (e *DefaultEngine) extractFormsFromPage(
	page pdf.Page, pageNum int, config ExtractionConfig,
) ([]ContentElement, []error) {
	var elements []ContentElement
	var errors []error

	// Form extraction requires document-level access
	if e.pdfReader == nil {
		errors = append(errors, fmt.Errorf("PDF reader not available for form extraction"))
		return elements, errors
	}

	// Extract all forms from the document (done once)
	// Use file-based extraction if file path is available
	formExtractor := NewFormExtractor(e.debugMode)
	var forms []FormField
	var err error

	if e.filePath != "" {
		// Preferred method: extract forms using pdfcpu with full access to PDF structure
		forms, err = formExtractor.ExtractFormsFromFile(e.filePath)
	} else {
		// Fallback: use heuristic extraction from pdf.Reader
		forms, err = formExtractor.ExtractForms(e.pdfReader)
	}

	if err != nil {
		errors = append(errors, fmt.Errorf("failed to extract forms: %w", err))
		return elements, errors
	}

	// Filter forms for this specific page
	for _, form := range forms {
		if form.Page == pageNum {
			element := ContentElement{
				Type:       ContentTypeForm,
				PageNumber: pageNum,
				Confidence: estimatedConfidenceThreshold,
				Content: FormElement{
					Field: form,
				},
			}
			if form.Bounds != nil {
				element.BoundingBox = *form.Bounds
			}
			elements = append(elements, element)
		}
	}

	return elements, errors
}

// extractAnnotationsFromPage extracts annotations from a page
func (e *DefaultEngine) extractAnnotationsFromPage(
	page pdf.Page, pageNum int, config ExtractionConfig,
) ([]ContentElement, []error) {
	var elements []ContentElement
	var errors []error

	// Get annotations array
	annotations := page.V.Key("Annots")
	if annotations.IsNull() {
		return elements, errors
	}

	// Process each annotation
	annotIndex := 0
	if annotations.Kind() == pdf.Array {
		for i := 0; i < annotations.Len(); i++ {
			annot := annotations.Index(i)
			if annot.IsNull() {
				continue
			}

			// Get annotation type
			annotType := annot.Key("Subtype")
			if annotType.IsNull() {
				continue
			}

			// Get annotation content
			content := ""
			if contents := annot.Key("Contents"); !contents.IsNull() {
				content = contents.Text()
			}

			// Get annotation rectangle
			rect := annot.Key("Rect")
			var bbox BoundingBox
			if !rect.IsNull() && rect.Kind() == pdf.Array && rect.Len() >= 4 {
				bbox = BoundingBox{
					LowerLeft: Coordinate{
						X: rect.Index(0).Float64(),
						Y: rect.Index(1).Float64(),
					},
					UpperRight: Coordinate{
						X: rect.Index(2).Float64(),
						Y: rect.Index(3).Float64(),
					},
				}
				bbox.Width = bbox.UpperRight.X - bbox.LowerLeft.X
				bbox.Height = bbox.UpperRight.Y - bbox.LowerLeft.Y
			}

			annotElement := ContentElement{
				ID:          e.generateID("annotation", pageNum, annotIndex),
				Type:        ContentTypeAnnotation,
				PageNumber:  pageNum,
				BoundingBox: bbox,
				Content: AnnotationElement{
					AnnotationType: annotType.Name(),
					Content:        content,
				},
				Confidence: 1.0,
			}

			elements = append(elements, annotElement)
			annotIndex++
		}
	}

	return elements, errors
}

// postProcessContent performs post-processing based on extraction mode
func (e *DefaultEngine) postProcessContent(result *ExtractionResult, config ExtractionConfig) error {
	switch config.Mode {
	case ModeTable:
		return e.detectTables(result, config)
	case ModeSemantic:
		return e.groupSemanticContent(result, config)
	case ModeComplete:
		// Perform all post-processing
		if err := e.detectTables(result, config); err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("table detection failed: %v", err))
		}
		if err := e.groupSemanticContent(result, config); err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("semantic grouping failed: %v", err))
		}
	case ModeRaw, ModeStructured, ModeForm:
		// No additional post-processing needed for these modes
	}

	return nil
}

// detectTables attempts to detect tabular structures in the content
func (e *DefaultEngine) detectTables(result *ExtractionResult, config ExtractionConfig) error {
	// Table detection algorithm would analyze text positioning and alignment
	// This is a simplified implementation

	textElements := e.filterElementsByType(result.Elements, ContentTypeText)
	if len(textElements) < minTableElements {
		return nil
	}

	// Group elements by approximate Y coordinates (rows)
	rows := e.groupElementsByRow(textElements, rowTolerance)

	if len(rows) < minRowsForTable {
		return nil
	}

	// Check if rows have similar column structure
	if table, confidence := e.analyzeTableStructure(rows); confidence > config.TableDetectionTh {
		result.Tables = append(result.Tables, *table)
	}

	return nil
}

// groupSemanticContent groups related content elements
func (e *DefaultEngine) groupSemanticContent(result *ExtractionResult, _ ExtractionConfig) error {
	// Semantic grouping would analyze content relationships
	// This could include grouping nearby text, associating labels with values, etc.

	// For now, just group elements by proximity
	return e.groupElementsByProximity(result.Elements, proximityThreshold)
}

// Query filters content elements based on the provided query
func (e *DefaultEngine) Query(elements []ContentElement, query Query) ([]ContentElement, error) {
	var filtered []ContentElement

	for _, element := range elements {
		if e.matchesQuery(element, query) {
			filtered = append(filtered, element)
		}
	}

	return filtered, nil
}

// matchesQuery checks if an element matches the query criteria
func (e *DefaultEngine) matchesQuery(element ContentElement, query Query) bool {
	// Check content type filter
	if len(query.ContentTypes) > 0 {
		found := false
		for _, ct := range query.ContentTypes {
			if element.Type == ct {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check page filter
	if len(query.Pages) > 0 {
		found := false
		for _, page := range query.Pages {
			if element.PageNumber == page {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check confidence threshold
	if query.MinConfidence > 0 && element.Confidence < query.MinConfidence {
		return false
	}

	// Check bounding box intersection
	if query.BoundingBox != nil {
		if !e.boundingBoxesIntersect(element.BoundingBox, *query.BoundingBox) {
			return false
		}
	}

	// Check text query
	if query.TextQuery != "" {
		if !e.elementContainsText(element, query.TextQuery) {
			return false
		}
	}

	return true
}

// Helper methods

func (e *DefaultEngine) validateRequest(req ExtractionRequest) error {
	if req.FilePath == "" {
		return fmt.Errorf("file path cannot be empty")
	}

	if req.Config.Mode == "" {
		req.Config.Mode = ModeRaw // Default mode
	}

	return nil
}

func (e *DefaultEngine) extractMetadata(pdfReader *pdf.Reader) (*PDFMetadata, error) {
	metadata := &PDFMetadata{}

	// Extract basic metadata if available
	// This would require accessing the document's Info dictionary
	// For now, return empty metadata

	return metadata, nil
}

func (e *DefaultEngine) determinePagesToProcess(requestedPages []int, totalPages int) []int {
	if len(requestedPages) == 0 {
		// Process all pages
		pages := make([]int, totalPages)
		for i := 0; i < totalPages; i++ {
			pages[i] = i + 1
		}
		return pages
	}

	// Filter valid page numbers
	var validPages []int
	for _, page := range requestedPages {
		if page >= 1 && page <= totalPages {
			validPages = append(validPages, page)
		}
	}

	return validPages
}

func (e *DefaultEngine) getPageInfo(page pdf.Page, pageNum int) (*PageInfo, error) {
	// Try to extract MediaBox with robust error handling
	mediaBox, err := e.extractMediaBox(page)
	if err != nil {
		// Log the error but continue with default dimensions
		fmt.Fprintf(os.Stderr, "[MediaBox] Failed to extract MediaBox for page %d: %v\n", pageNum, err)

		// Use default US Letter size
		mediaBox = &BoundingBox{
			LowerLeft:  Coordinate{X: 0, Y: 0},
			UpperRight: Coordinate{X: 612, Y: 792},
			Width:      612.0,
			Height:     792.0,
		}
		fmt.Fprintf(os.Stderr, "[MediaBox] Using default dimensions for page %d\n", pageNum)
	}

	return &PageInfo{
		Number:   pageNum,
		Width:    mediaBox.Width,
		Height:   mediaBox.Height,
		MediaBox: *mediaBox,
	}, nil
}

// extractMediaBox extracts MediaBox with robust error handling and inheritance support
func (e *DefaultEngine) extractMediaBox(page pdf.Page) (*BoundingBox, error) {
	// Enhanced MediaBox extraction with better error handling
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintf(os.Stderr, "[MediaBox] PANIC during MediaBox extraction: %v\n", r)
		}
	}()

	// Try direct MediaBox extraction first
	mediaBox := page.V.Key("MediaBox")
	if !mediaBox.IsNull() {
		if bbox, err := e.parseMediaBoxValue(mediaBox); err == nil {
			return bbox, nil
		} else {
			fmt.Fprintf(os.Stderr, "[MediaBox] Direct extraction failed: %v\n", err)
		}
	}

	// Try inheritance from parent pages
	if inheritedBox := e.getInheritedMediaBox(page); inheritedBox != nil {
		return inheritedBox, nil
	}

	return nil, fmt.Errorf("no valid MediaBox found")
}

// parseMediaBoxValue parses a MediaBox PDF value into a BoundingBox with enhanced error handling
func (e *DefaultEngine) parseMediaBoxValue(mediaBox pdf.Value) (*BoundingBox, error) {
	// Add panic recovery for corrupted MediaBox data
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintf(os.Stderr, "[MediaBox] PANIC during MediaBox parsing: %v\n", r)
		}
	}()

	if mediaBox.IsNull() {
		return nil, fmt.Errorf("MediaBox value is null")
	}

	if mediaBox.Kind() != pdf.Array {
		return nil, fmt.Errorf("MediaBox is not an array: %v", mediaBox.Kind())
	}

	if mediaBox.Len() != 4 {
		return nil, fmt.Errorf("invalid MediaBox array length: %d, expected 4", mediaBox.Len())
	}

	coords := make([]float64, 4)
	for i := 0; i < 4; i++ {
		val := mediaBox.Index(i)
		if val.IsNull() {
			return nil, fmt.Errorf("coordinate at index %d is null", i)
		}

		// Enhanced numeric type handling with better error recovery
		switch val.Kind() {
		case pdf.Integer:
			coords[i] = float64(val.Int64())
		case pdf.Real:
			coords[i] = val.Float64()
		default:
			// Try to extract numeric value as string fallback
			if str := val.Text(); str != "" {
				if f, err := parseFloatValue(str); err == nil {
					coords[i] = f
					fmt.Fprintf(os.Stderr, "[MediaBox] WARNING: Recovered coordinate %d from string: %s\n", i, str)
				} else {
					return nil, fmt.Errorf("invalid coordinate type at index %d: %v (failed string conversion)", i, val.Kind())
				}
			} else {
				return nil, fmt.Errorf("invalid coordinate type at index %d: %v", i, val.Kind())
			}
		}
	}

	llx, lly, urx, ury := coords[0], coords[1], coords[2], coords[3]

	// Validate rectangle dimensions with more tolerance
	if urx <= llx || ury <= lly {
		// Try to fix inverted coordinates
		if llx > urx {
			llx, urx = urx, llx
			fmt.Fprintf(os.Stderr, "[MediaBox] WARNING: Fixed inverted X coordinates\n")
		}
		if lly > ury {
			lly, ury = ury, lly
			fmt.Fprintf(os.Stderr, "[MediaBox] WARNING: Fixed inverted Y coordinates\n")
		}

		// If still invalid, return error
		if urx <= llx || ury <= lly {
			return nil, fmt.Errorf("invalid MediaBox dimensions: [%.2f %.2f %.2f %.2f]", llx, lly, urx, ury)
		}
	}

	return &BoundingBox{
		LowerLeft:  Coordinate{X: llx, Y: lly},
		UpperRight: Coordinate{X: urx, Y: ury},
		Width:      urx - llx,
		Height:     ury - lly,
	}, nil
}

// parseFloatValue attempts to parse a string as a float64 with fallback handling
func parseFloatValue(s string) (float64, error) {
	// Try standard parsing first
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return f, nil
	}

	// Try parsing with common PDF number formats
	s = strings.TrimSpace(s)
	if strings.HasSuffix(s, "f") || strings.HasSuffix(s, "F") {
		s = s[:len(s)-1]
		if f, err := strconv.ParseFloat(s, 64); err == nil {
			return f, nil
		}
	}

	return 0, fmt.Errorf("unable to parse '%s' as float", s)
}

// getInheritedMediaBox traverses up the page tree to find an inherited MediaBox
func (e *DefaultEngine) getInheritedMediaBox(page pdf.Page) *BoundingBox {
	current := page.V

	// Look for Parent reference and traverse up the tree
	for i := 0; i < 10; i++ { // Limit iterations to prevent infinite loops
		parent := current.Key("Parent")
		if parent.IsNull() {
			break
		}

		// Check if parent has MediaBox
		if mediaBox := parent.Key("MediaBox"); !mediaBox.IsNull() {
			if bbox, err := e.parseMediaBoxValue(mediaBox); err == nil {
				fmt.Fprintf(os.Stderr, "[MediaBox] Using inherited MediaBox: %.2fx%.2f\n", bbox.Width, bbox.Height)
				return bbox
			}
		}

		// Move up to the next parent
		current = parent
	}

	// Return default US Letter size if no inheritance found
	return &BoundingBox{
		LowerLeft:  Coordinate{X: 0, Y: 0},
		UpperRight: Coordinate{X: 612, Y: 792},
		Width:      612.0,
		Height:     792.0,
	}
}

func (e *DefaultEngine) generateID(prefix string, pageNum, index int) string {
	return fmt.Sprintf("%s_%d_%d", prefix, pageNum, index)
}

func (e *DefaultEngine) generateHashFromData(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

func (e *DefaultEngine) countElements(elements []ContentElement) ElementCounts {
	counts := ElementCounts{}

	for i := range elements {
		switch elements[i].Type {
		case ContentTypeText:
			counts.Text++
		case ContentTypeImage:
			counts.Images++
		case ContentTypeVector:
			counts.Vectors++
		case ContentTypeForm:
			counts.Forms++
		case ContentTypeAnnotation:
			counts.Annotations++
		case ContentTypeMetadata, ContentTypeStructural:
			// These types don't have specific counters yet
		}
		counts.Total++
	}

	return counts
}

func (e *DefaultEngine) filterElementsByType(elements []ContentElement, contentType ContentType) []ContentElement {
	var filtered []ContentElement
	for i := range elements {
		if elements[i].Type == contentType {
			filtered = append(filtered, elements[i])
		}
	}
	return filtered
}

func (e *DefaultEngine) groupElementsByRow(elements []ContentElement, tolerance float64) [][]ContentElement {
	if len(elements) == 0 {
		return nil
	}

	// Sort elements by Y coordinate
	sort.Slice(elements, func(i, j int) bool {
		return elements[i].BoundingBox.LowerLeft.Y > elements[j].BoundingBox.LowerLeft.Y
	})

	var rows [][]ContentElement
	currentRow := []ContentElement{elements[0]}
	currentY := elements[0].BoundingBox.LowerLeft.Y

	for i := 1; i < len(elements); i++ {
		elementY := elements[i].BoundingBox.LowerLeft.Y
		if abs(elementY-currentY) <= tolerance {
			// Same row
			currentRow = append(currentRow, elements[i])
		} else {
			// New row
			rows = append(rows, currentRow)
			currentRow = []ContentElement{elements[i]}
			currentY = elementY
		}
	}

	if len(currentRow) > 0 {
		rows = append(rows, currentRow)
	}

	return rows
}

func (e *DefaultEngine) analyzeTableStructure(rows [][]ContentElement) (*TableElement, float64) {
	// Simple table structure analysis
	if len(rows) < 2 {
		return nil, 0.0
	}

	// Check if rows have consistent column counts
	colCounts := make(map[int]int)
	for _, row := range rows {
		colCounts[len(row)]++
	}

	// Find most common column count
	maxCount := 0
	commonColCount := 0
	for count, frequency := range colCounts {
		if frequency > maxCount {
			maxCount = frequency
			commonColCount = count
		}
	}

	// Calculate confidence based on consistency
	confidence := float64(maxCount) / float64(len(rows))

	if confidence < minimumConfidenceThreshold {
		return nil, confidence
	}

	// Build table structure
	table := &TableElement{
		Rows:       make([]TableRow, 0, len(rows)),
		Columns:    make([]TableCol, commonColCount),
		CellCount:  0,
		HasHeaders: len(rows) > 0,
		Confidence: confidence,
	}

	// Initialize columns
	for i := 0; i < commonColCount; i++ {
		table.Columns[i] = TableCol{
			Index: i,
		}
	}

	// Process rows
	for rowIdx, row := range rows {
		if len(row) != commonColCount {
			continue // Skip inconsistent rows
		}

		tableRow := TableRow{
			Index:    rowIdx,
			Cells:    make([]TableCell, len(row)),
			IsHeader: rowIdx == 0,
		}

		for colIdx := range row {
			element := row[colIdx]
			cell := TableCell{
				RowIndex:    rowIdx,
				ColIndex:    colIdx,
				BoundingBox: element.BoundingBox,
				Confidence:  element.Confidence,
			}

			// Extract text content
			if textElement, ok := element.Content.(TextElement); ok {
				cell.Content = textElement.Text
			}

			tableRow.Cells[colIdx] = cell
			table.CellCount++
		}

		table.Rows = append(table.Rows, tableRow)
	}

	return table, confidence
}

func (e *DefaultEngine) groupElementsByProximity(elements []ContentElement, threshold float64) error {
	// Proximity grouping implementation would analyze spatial relationships
	// and create parent-child relationships between nearby elements
	return nil
}

func (e *DefaultEngine) boundingBoxesIntersect(box1, box2 BoundingBox) bool {
	return !(box1.UpperRight.X < box2.LowerLeft.X ||
		box2.UpperRight.X < box1.LowerLeft.X ||
		box1.UpperRight.Y < box2.LowerLeft.Y ||
		box2.UpperRight.Y < box1.LowerLeft.Y)
}

func (e *DefaultEngine) elementContainsText(element ContentElement, query string) bool {
	switch content := element.Content.(type) {
	case TextElement:
		return strings.Contains(strings.ToLower(content.Text), strings.ToLower(query))
	case AnnotationElement:
		return strings.Contains(strings.ToLower(content.Content), strings.ToLower(query))
	}
	return false
}

func (e *DefaultEngine) GetMetadata(filePath string) (*PDFMetadata, error) {
	f, pdfReader, err := pdf.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open PDF: %w", err)
	}
	defer f.Close()

	return e.extractMetadata(pdfReader)
}

// GetPageInfo returns information about all pages in the PDF
func (e *DefaultEngine) GetPageInfo(filePath string) ([]PageInfo, error) {
	f, pdfReader, err := pdf.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open PDF: %w", err)
	}
	defer f.Close()

	var pages []PageInfo
	for pageNum := 1; pageNum <= pdfReader.NumPage(); pageNum++ {
		page := pdfReader.Page(pageNum)
		if page.V.IsNull() {
			continue
		}

		pageInfo, err := e.getPageInfo(page, pageNum)
		if err != nil {
			return nil, fmt.Errorf("failed to get info for page %d: %w", pageNum, err)
		}

		pages = append(pages, *pageInfo)
	}

	return pages, nil
}

// abs returns the absolute value of a float64
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
