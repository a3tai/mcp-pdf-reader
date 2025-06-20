package pagerange

import (
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/a3tai/mcp-pdf-reader/internal/pdf/streaming"
)

// PageRangeExtractor handles efficient extraction from specific page ranges
type PageRangeExtractor struct {
	streamParser *streaming.StreamParser
	pageIndex    *PageIndex
	cache        *PageObjectCache
	config       ExtractorConfig
}

// ExtractorConfig configures the page range extractor
type ExtractorConfig struct {
	MaxCacheSize    int64 // Maximum cache size in bytes
	EnableCaching   bool  // Whether to enable object caching
	PreloadObjects  bool  // Whether to preload required objects
	ParallelEnabled bool  // Whether to enable parallel processing
}

// PageRange represents a range of pages to extract
type PageRange struct {
	Start int `json:"start"`
	End   int `json:"end"`
}

// ExtractOptions configures what content to extract
type ExtractOptions struct {
	ContentTypes       []string `json:"content_types"`       // text, images, forms, metadata
	PreserveFormatting bool     `json:"preserve_formatting"` // Whether to preserve text formatting
	IncludeMetadata    bool     `json:"include_metadata"`    // Whether to include page metadata
	ExtractImages      bool     `json:"extract_images"`      // Whether to extract images
	ExtractForms       bool     `json:"extract_forms"`       // Whether to extract forms
	OutputFormat       string   `json:"output_format"`       // json, xml, plain
}

// ExtractedContent represents the result of page range extraction
type ExtractedContent struct {
	Pages      map[int]*PageContent `json:"pages"`
	TotalPages int                  `json:"total_pages"`
	Ranges     []PageRange          `json:"ranges"`
	Metadata   ExtractionMetadata   `json:"metadata"`
}

// PageContent represents content extracted from a single page
type PageContent struct {
	PageNumber int                    `json:"page_number"`
	Text       string                 `json:"text"`
	Images     []ImageReference       `json:"images"`
	Forms      []FormField            `json:"forms"`
	Metadata   PageMetadata           `json:"metadata"`
	TextBlocks []FormattedTextBlock   `json:"text_blocks,omitempty"`
	Objects    map[string]interface{} `json:"objects,omitempty"`
}

// ImageReference represents an image in a page
type ImageReference struct {
	ObjectID   int     `json:"object_id"`
	X          float64 `json:"x"`
	Y          float64 `json:"y"`
	Width      float64 `json:"width"`
	Height     float64 `json:"height"`
	Format     string  `json:"format"`
	ColorSpace string  `json:"color_space,omitempty"`
}

// FormField represents a form field in a page
type FormField struct {
	FieldType  string  `json:"field_type"`
	FieldName  string  `json:"field_name"`
	FieldValue string  `json:"field_value"`
	X          float64 `json:"x"`
	Y          float64 `json:"y"`
	Width      float64 `json:"width"`
	Height     float64 `json:"height"`
}

// PageMetadata represents metadata about a page
type PageMetadata struct {
	MediaBox      Rectangle `json:"media_box"`
	CropBox       Rectangle `json:"crop_box,omitempty"`
	Rotation      int       `json:"rotation"`
	UserUnit      float64   `json:"user_unit,omitempty"`
	ResourceCount int       `json:"resource_count"`
	ObjectCount   int       `json:"object_count"`
}

// Rectangle represents a rectangular area
type Rectangle struct {
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

// FormattedTextBlock represents a text block with formatting information
type FormattedTextBlock struct {
	Text     string  `json:"text"`
	X        float64 `json:"x"`
	Y        float64 `json:"y"`
	Width    float64 `json:"width"`
	Height   float64 `json:"height"`
	FontName string  `json:"font_name"`
	FontSize float64 `json:"font_size"`
	Color    string  `json:"color,omitempty"`
}

// ExtractionMetadata provides information about the extraction process
type ExtractionMetadata struct {
	ProcessingTime int64  `json:"processing_time_ms"`
	CacheHits      int    `json:"cache_hits"`
	CacheMisses    int    `json:"cache_misses"`
	ObjectsParsed  int    `json:"objects_parsed"`
	BytesRead      int64  `json:"bytes_read"`
	MemoryUsage    int64  `json:"memory_usage"`
	Status         string `json:"status"`
}

// ObjectRef represents a reference to a PDF object
type ObjectRef struct {
	ObjectID   int   `json:"object_id"`
	Generation int   `json:"generation"`
	Offset     int64 `json:"offset"`
}

// NewPageRangeExtractor creates a new page range extractor
func NewPageRangeExtractor(config ...ExtractorConfig) *PageRangeExtractor {
	cfg := ExtractorConfig{
		MaxCacheSize:    50 * 1024 * 1024, // 50MB default cache
		EnableCaching:   true,
		PreloadObjects:  true,
		ParallelEnabled: false, // Disabled for now
	}

	if len(config) > 0 {
		cfg = config[0]
	}

	return &PageRangeExtractor{
		cache:  NewPageObjectCache(cfg.MaxCacheSize),
		config: cfg,
	}
}

// ExtractFromFile extracts content from specific page ranges in a file
func (pre *PageRangeExtractor) ExtractFromFile(filePath string, ranges []PageRange, options ExtractOptions) (*ExtractedContent, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	return pre.ExtractRange(file, ranges, options)
}

// ExtractRange extracts content from specific page ranges using a reader
func (pre *PageRangeExtractor) ExtractRange(reader io.ReadSeeker, ranges []PageRange, options ExtractOptions) (*ExtractedContent, error) {
	startTime := getCurrentTimeMillis()

	// Initialize streaming parser
	parserOpts := streaming.StreamOptions{
		ChunkSizeMB:     2,    // 2MB chunks for page range extraction
		MaxMemoryMB:     64,   // 64MB max memory
		XRefCacheSize:   2000, // Larger cache for object lookups
		ObjectCacheSize: 1000,
		GCTrigger:       0.8,
		BufferPoolSize:  15,
	}

	parser, err := streaming.NewStreamParser(reader, parserOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to create stream parser: %w", err)
	}
	defer parser.Close()

	pre.streamParser = parser

	// Build page index to locate pages without parsing all content
	if err := pre.buildPageIndex(reader); err != nil {
		return nil, fmt.Errorf("failed to build page index: %w", err)
	}

	// Check if we have any pages
	if pre.pageIndex.TotalPages == 0 {
		return nil, fmt.Errorf("no pages found in PDF")
	}

	// Validate ranges
	validRanges := pre.validateRanges(ranges)

	// Calculate required objects for requested pages
	requiredObjects, err := pre.calculateRequiredObjects(validRanges)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate required objects: %w", err)
	}

	// Preload objects if enabled
	if pre.config.PreloadObjects {
		if err := pre.preloadObjects(reader, requiredObjects); err != nil {
			return nil, fmt.Errorf("failed to preload objects: %w", err)
		}
	}

	// Extract content from specific pages
	content := &ExtractedContent{
		Pages:      make(map[int]*PageContent),
		TotalPages: pre.pageIndex.TotalPages,
		Ranges:     validRanges,
	}

	for _, r := range validRanges {
		for pageNum := r.Start; pageNum <= r.End && pageNum <= pre.pageIndex.TotalPages; pageNum++ {
			pageContent, err := pre.extractPageContent(reader, pageNum, options)
			if err != nil {
				return nil, fmt.Errorf("failed to extract page %d: %w", pageNum, err)
			}
			content.Pages[pageNum] = pageContent
		}
	}

	// Fill in extraction metadata
	endTime := getCurrentTimeMillis()
	content.Metadata = ExtractionMetadata{
		ProcessingTime: endTime - startTime,
		CacheHits:      int(pre.cache.GetStats().Hits),
		CacheMisses:    int(pre.cache.GetStats().Misses),
		ObjectsParsed:  len(requiredObjects),
		MemoryUsage:    parser.GetMemoryUsage().CurrentBytes,
		Status:         "completed",
	}

	return content, nil
}

// buildPageIndex builds an index of page locations without parsing all content
func (pre *PageRangeExtractor) buildPageIndex(reader io.ReadSeeker) error {
	pre.pageIndex = &PageIndex{
		PageOffsets: make(map[int]int64),
		PageObjects: make(map[int]ObjectRef),
		Resources:   make(map[int][]ObjectRef),
	}

	// Build page index using proper PDF structure
	err := pre.buildPageIndexFromCatalog()
	if err != nil {
		// Fallback to pattern matching if catalog parsing fails
		return pre.buildPageIndexFromPatterns(reader)
	}

	// Extract resource references for each page
	for pageNum := 1; pageNum <= pre.pageIndex.TotalPages; pageNum++ {
		resources, err := pre.extractPageResources(reader, pageNum)
		if err != nil {
			// Continue with other pages if one fails
			continue
		}
		pre.pageIndex.Resources[pageNum] = resources
	}

	return nil
}

// buildPageIndexFromCatalog builds the page index using the PDF catalog structure
func (pre *PageRangeExtractor) buildPageIndexFromCatalog() error {
	// Find the catalog object (object 1 in most PDFs)
	catalogObj, err := pre.streamParser.GetObject(1, 0)
	if err != nil {
		return fmt.Errorf("failed to get catalog object: %w", err)
	}

	// Extract Pages reference from catalog
	pagesRegex := regexp.MustCompile(`/Pages\s+(\d+)\s+(\d+)\s+R`)
	matches := pagesRegex.FindStringSubmatch(catalogObj.Content)
	if len(matches) < 3 {
		return fmt.Errorf("Pages reference not found in catalog")
	}

	pagesObjID, _ := strconv.Atoi(matches[1])
	pagesGeneration, _ := strconv.Atoi(matches[2])

	// Get the Pages object
	pagesObj, err := pre.streamParser.GetObject(pagesObjID, pagesGeneration)
	if err != nil {
		return fmt.Errorf("failed to get Pages object %d %d: %w", pagesObjID, pagesGeneration, err)
	}

	// Parse the page tree
	pageNum := 1
	err = pre.parsePageTree(pagesObj.Content, &pageNum, pagesObjID)
	if err != nil {
		return fmt.Errorf("failed to parse page tree: %w", err)
	}

	pre.pageIndex.TotalPages = pageNum - 1
	return nil
}

// parsePageTree recursively parses the page tree structure
func (pre *PageRangeExtractor) parsePageTree(content string, pageNum *int, objID int) error {
	// Check if this is a Pages node or a Page node
	// Be more specific: check for "/Type /Page" but not "/Type /Pages"
	if strings.Contains(content, "/Type /Page") && !strings.Contains(content, "/Type /Pages") {
		// This is a Page object - use the actual object ID
		objRef := ObjectRef{
			ObjectID:   objID,
			Generation: 0,
			Offset:     0, // Will be filled by XRef table
		}

		pre.pageIndex.PageObjects[*pageNum] = objRef
		pre.pageIndex.PageOffsets[*pageNum] = 0 // Will be filled by XRef table
		*pageNum++
		return nil
	}

	// This is a Pages node - find Kids array
	kidsRegex := regexp.MustCompile(`/Kids\s*\[\s*((?:\d+\s+\d+\s+R\s*)+)\]`)
	matches := kidsRegex.FindStringSubmatch(content)
	if len(matches) < 2 {
		return fmt.Errorf("Kids array not found in Pages object")
	}

	// Parse kid references
	kidRefs := parseObjectReferences(matches[1])
	for _, kidRef := range kidRefs {
		// Get the kid object
		kidObj, err := pre.streamParser.GetObject(kidRef.ObjectID, kidRef.Generation)
		if err != nil {
			continue // Skip problematic kids
		}

		// Recursively parse the kid
		err = pre.parsePageTree(kidObj.Content, pageNum, kidRef.ObjectID)
		if err != nil {
			continue // Skip problematic kids
		}
	}

	return nil
}

// buildPageIndexFromPatterns builds the page index using pattern matching (fallback)
func (pre *PageRangeExtractor) buildPageIndexFromPatterns(reader io.ReadSeeker) error {
	var pageObjects []ObjectRef

	err := pre.streamParser.ProcessInChunks(func(chunk []byte, offset int64) error {
		// Look for page object patterns
		pageRegex := regexp.MustCompile(`(\d+)\s+(\d+)\s+obj\s*<<[^>]*?/Type\s*/Page[^>]*?>>`)
		matches := pageRegex.FindAllSubmatch(chunk, -1)

		for _, match := range matches {
			if len(match) >= 3 {
				objID, _ := strconv.Atoi(string(match[1]))
				generation, _ := strconv.Atoi(string(match[2]))

				// Calculate object offset
				matchPos := strings.Index(string(chunk), string(match[0]))
				objOffset := offset + int64(matchPos)

				pageObjects = append(pageObjects, ObjectRef{
					ObjectID:   objID,
					Generation: generation,
					Offset:     objOffset,
				})
			}
		}

		return nil
	})

	if err != nil {
		return err
	}

	// Build index from found page objects
	pageNum := 1
	for _, obj := range pageObjects {
		pre.pageIndex.PageOffsets[pageNum] = obj.Offset
		pre.pageIndex.PageObjects[pageNum] = obj
		pageNum++
	}

	pre.pageIndex.TotalPages = pageNum - 1
	return nil
}

// parseObjectReferences parses object references from a string
func parseObjectReferences(content string) []ObjectRef {
	var refs []ObjectRef

	refRegex := regexp.MustCompile(`(\d+)\s+(\d+)\s+R`)
	matches := refRegex.FindAllStringSubmatch(content, -1)

	for _, match := range matches {
		if len(match) >= 3 {
			objID, _ := strconv.Atoi(match[1])
			generation, _ := strconv.Atoi(match[2])

			refs = append(refs, ObjectRef{
				ObjectID:   objID,
				Generation: generation,
			})
		}
	}

	return refs
}

// validateRanges validates and normalizes page ranges
func (pre *PageRangeExtractor) validateRanges(ranges []PageRange) []PageRange {
	var validRanges []PageRange

	for _, r := range ranges {
		// Normalize range bounds
		start := r.Start
		end := r.End

		if start < 1 {
			start = 1
		}
		if end > pre.pageIndex.TotalPages {
			end = pre.pageIndex.TotalPages
		}
		if start > end {
			continue // Skip invalid range
		}

		validRanges = append(validRanges, PageRange{
			Start: start,
			End:   end,
		})
	}

	return validRanges
}

// calculateRequiredObjects determines which objects are needed for the requested pages
func (pre *PageRangeExtractor) calculateRequiredObjects(ranges []PageRange) ([]ObjectRef, error) {
	requiredObjects := make(map[string]ObjectRef)

	for _, r := range ranges {
		for pageNum := r.Start; pageNum <= r.End; pageNum++ {
			// Add page object
			if pageObj, exists := pre.pageIndex.PageObjects[pageNum]; exists {
				key := fmt.Sprintf("%d_%d", pageObj.ObjectID, pageObj.Generation)
				requiredObjects[key] = pageObj
			}

			// Add resource objects for this page
			if resources, exists := pre.pageIndex.Resources[pageNum]; exists {
				for _, resource := range resources {
					key := fmt.Sprintf("%d_%d", resource.ObjectID, resource.Generation)
					requiredObjects[key] = resource
				}
			}
		}
	}

	// Convert map to slice
	var objects []ObjectRef
	for _, obj := range requiredObjects {
		objects = append(objects, obj)
	}

	return objects, nil
}

// preloadObjects loads required objects into cache
func (pre *PageRangeExtractor) preloadObjects(reader io.ReadSeeker, objects []ObjectRef) error {
	for _, obj := range objects {
		// Check if already cached
		if pre.cache.Contains(obj) {
			continue
		}

		// Use StreamParser's GetObject method
		pdfObj, err := pre.streamParser.GetObject(obj.ObjectID, obj.Generation)
		if err != nil {
			// Continue with other objects if one fails
			continue
		}

		pre.cache.Put(obj, pdfObj.Content)
	}

	return nil
}

// extractPageContent extracts content from a specific page
func (pre *PageRangeExtractor) extractPageContent(reader io.ReadSeeker, pageNum int, options ExtractOptions) (*PageContent, error) {
	pageContent := &PageContent{
		PageNumber: pageNum,
		Images:     []ImageReference{},
		Forms:      []FormField{},
		TextBlocks: []FormattedTextBlock{},
		Objects:    make(map[string]interface{}),
	}

	// Get page object
	pageObj, exists := pre.pageIndex.PageObjects[pageNum]
	if !exists {
		return nil, fmt.Errorf("page %d not found in index", pageNum)
	}

	// Parse page object
	pageObjContent, err := pre.getOrParseObject(reader, pageObj)
	if err != nil {
		return nil, fmt.Errorf("failed to parse page object: %w", err)
	}

	// Extract page metadata
	pageContent.Metadata = pre.extractPageMetadata(pageObjContent)

	// Extract different types of content based on options
	if containsString(options.ContentTypes, "text") {
		text, textBlocks := pre.extractTextContent(reader, pageNum, options.PreserveFormatting)
		pageContent.Text = text
		if options.PreserveFormatting {
			pageContent.TextBlocks = textBlocks
		}
	}

	if containsString(options.ContentTypes, "images") && options.ExtractImages {
		images := pre.extractImageReferences(reader, pageNum)
		pageContent.Images = images
	}

	if containsString(options.ContentTypes, "forms") && options.ExtractForms {
		forms := pre.extractFormFields(reader, pageNum)
		pageContent.Forms = forms
	}

	return pageContent, nil
}

// extractPageResources extracts resource references for a page
func (pre *PageRangeExtractor) extractPageResources(reader io.ReadSeeker, pageNum int) ([]ObjectRef, error) {
	pageObj, exists := pre.pageIndex.PageObjects[pageNum]
	if !exists {
		return nil, fmt.Errorf("page %d not found", pageNum)
	}

	pdfObj, err := pre.streamParser.GetObject(pageObj.ObjectID, pageObj.Generation)
	if err != nil {
		return nil, fmt.Errorf("failed to get page object %d %d: %w", pageObj.ObjectID, pageObj.Generation, err)
	}

	objContent := pdfObj.Content

	var resources []ObjectRef

	// Extract content stream references
	contentRegex := regexp.MustCompile(`/Contents\s+(\d+)\s+(\d+)\s+R`)
	matches := contentRegex.FindAllStringSubmatch(objContent, -1)
	for _, match := range matches {
		if len(match) >= 3 {
			objID, _ := strconv.Atoi(match[1])
			generation, _ := strconv.Atoi(match[2])
			resources = append(resources, ObjectRef{
				ObjectID:   objID,
				Generation: generation,
			})
		}
	}

	// Extract resource dictionary references
	resourceRegex := regexp.MustCompile(`/Resources\s+(\d+)\s+(\d+)\s+R`)
	matches = resourceRegex.FindAllStringSubmatch(objContent, -1)
	for _, match := range matches {
		if len(match) >= 3 {
			objID, _ := strconv.Atoi(match[1])
			generation, _ := strconv.Atoi(match[2])
			resources = append(resources, ObjectRef{
				ObjectID:   objID,
				Generation: generation,
			})
		}
	}

	return resources, nil
}

// Helper methods

func (pre *PageRangeExtractor) getOrParseObject(reader io.ReadSeeker, obj ObjectRef) (string, error) {
	// Check cache first
	if cached := pre.cache.Get(obj); cached != nil {
		if content, ok := cached.(string); ok {
			return content, nil
		}
	}

	// Use StreamParser's GetObject method
	pdfObj, err := pre.streamParser.GetObject(obj.ObjectID, obj.Generation)
	if err != nil {
		return "", fmt.Errorf("failed to get object %d %d: %w", obj.ObjectID, obj.Generation, err)
	}

	content := pdfObj.Content

	// Cache for future use
	pre.cache.Put(obj, content)
	return content, nil
}

func (pre *PageRangeExtractor) parseObjectAt(reader io.ReadSeeker, obj ObjectRef) (string, error) {
	// Seek to object position
	if obj.Offset > 0 {
		_, err := reader.Seek(obj.Offset, io.SeekStart)
		if err != nil {
			return "", fmt.Errorf("failed to seek to object: %w", err)
		}
	}

	// Read object content
	chunk, err := pre.streamParser.ReadChunk()
	if err != nil {
		return "", fmt.Errorf("failed to read object chunk: %w", err)
	}

	// Extract object content between obj and endobj
	objRegex := regexp.MustCompile(fmt.Sprintf(`%d\s+%d\s+obj\s*(.*?)\s*endobj`, obj.ObjectID, obj.Generation))
	matches := objRegex.FindSubmatch(chunk)
	if len(matches) > 1 {
		return string(matches[1]), nil
	}

	return "", fmt.Errorf("object %d %d not found", obj.ObjectID, obj.Generation)
}

func (pre *PageRangeExtractor) extractPageMetadata(pageContent string) PageMetadata {
	metadata := PageMetadata{}

	// Extract MediaBox
	if mediaBox := extractRectangle(pageContent, "MediaBox"); mediaBox != nil {
		metadata.MediaBox = *mediaBox
	}

	// Extract CropBox
	if cropBox := extractRectangle(pageContent, "CropBox"); cropBox != nil {
		metadata.CropBox = *cropBox
	}

	// Extract rotation
	rotateRegex := regexp.MustCompile(`/Rotate\s+(\d+)`)
	if matches := rotateRegex.FindStringSubmatch(pageContent); len(matches) > 1 {
		rotation, _ := strconv.Atoi(matches[1])
		metadata.Rotation = rotation
	}

	return metadata
}

func (pre *PageRangeExtractor) extractTextContent(reader io.ReadSeeker, pageNum int, preserveFormatting bool) (string, []FormattedTextBlock) {
	// This is a simplified implementation - would need more sophisticated text extraction
	var text strings.Builder
	var blocks []FormattedTextBlock

	// Get content streams for this page
	resources, exists := pre.pageIndex.Resources[pageNum]
	if !exists {
		return "", blocks
	}

	for _, resource := range resources {
		content, err := pre.getOrParseObject(reader, resource)
		if err != nil {
			continue
		}

		// Extract text using simple regex
		textRegex := regexp.MustCompile(`\((.*?)\)\s*Tj`)
		matches := textRegex.FindAllStringSubmatch(content, -1)

		for _, match := range matches {
			if len(match) > 1 {
				textContent := match[1]
				text.WriteString(textContent)
				text.WriteString(" ")

				if preserveFormatting {
					blocks = append(blocks, FormattedTextBlock{
						Text:     textContent,
						X:        0, // Would need position parsing
						Y:        0,
						FontName: "Unknown",
						FontSize: 12,
					})
				}
			}
		}
	}

	return strings.TrimSpace(text.String()), blocks
}

func (pre *PageRangeExtractor) extractImageReferences(reader io.ReadSeeker, pageNum int) []ImageReference {
	var images []ImageReference

	// Get content streams for this page
	resources, exists := pre.pageIndex.Resources[pageNum]
	if !exists {
		return images
	}

	for _, resource := range resources {
		content, err := pre.getOrParseObject(reader, resource)
		if err != nil {
			continue
		}

		// Look for image references
		imageRegex := regexp.MustCompile(`/Im(\d+)\s+Do`)
		matches := imageRegex.FindAllStringSubmatch(content, -1)

		for _, match := range matches {
			if len(match) > 1 {
				objID, _ := strconv.Atoi(match[1])
				images = append(images, ImageReference{
					ObjectID: objID,
					Format:   "Unknown",
				})
			}
		}
	}

	return images
}

func (pre *PageRangeExtractor) extractFormFields(reader io.ReadSeeker, pageNum int) []FormField {
	var forms []FormField

	// Get content streams for this page
	resources, exists := pre.pageIndex.Resources[pageNum]
	if !exists {
		return forms
	}

	for _, resource := range resources {
		content, err := pre.getOrParseObject(reader, resource)
		if err != nil {
			continue
		}

		// Look for form field annotations
		formRegex := regexp.MustCompile(`/FT\s*/(\w+)`)
		matches := formRegex.FindAllStringSubmatch(content, -1)

		for _, match := range matches {
			if len(match) > 1 {
				forms = append(forms, FormField{
					FieldType: match[1],
					FieldName: "Unknown",
				})
			}
		}
	}

	return forms
}

// Utility functions

func extractRectangle(content, boxType string) *Rectangle {
	regex := regexp.MustCompile(fmt.Sprintf(`/%s\s*\[\s*(\d+(?:\.\d+)?)\s+(\d+(?:\.\d+)?)\s+(\d+(?:\.\d+)?)\s+(\d+(?:\.\d+)?)\s*\]`, boxType))
	matches := regex.FindStringSubmatch(content)

	if len(matches) >= 5 {
		x, _ := strconv.ParseFloat(matches[1], 64)
		y, _ := strconv.ParseFloat(matches[2], 64)
		width, _ := strconv.ParseFloat(matches[3], 64)
		height, _ := strconv.ParseFloat(matches[4], 64)

		return &Rectangle{
			X:      x,
			Y:      y,
			Width:  width,
			Height: height,
		}
	}

	return nil
}

func containsString(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func getCurrentTimeMillis() int64 {
	// Simplified implementation - would use time.Now().UnixMilli() in real code
	return 1000
}
