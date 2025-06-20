package streaming

import (
	"context"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

// PageStreamer handles page-by-page streaming processing of PDF documents
type PageStreamer struct {
	parser      *StreamParser
	processor   *ChunkProcessor
	pageSize    int // Max pages to keep in memory
	currentPage int
	totalPages  int
	pageQueue   []StreamPage
	pageMutex   sync.RWMutex

	// Page tracking
	pageOffsets map[int]int64       // Page number -> file offset
	pageObjects map[int][]PDFObject // Page number -> objects

	// Configuration
	config PageStreamerConfig
}

// PageStreamerConfig configures the page streamer
type PageStreamerConfig struct {
	MaxPagesInMemory int    // Maximum pages to keep in memory
	PageBufferSize   int    // Size of page processing buffer
	EnableProgress   bool   // Whether to track progress
	ExtractText      bool   // Extract text from pages
	ExtractImages    bool   // Extract images from pages
	ExtractForms     bool   // Extract forms from pages
	CallbackMode     string // "sequential" or "parallel"
}

// DefaultPageStreamerConfig returns sensible defaults
func DefaultPageStreamerConfig() PageStreamerConfig {
	return PageStreamerConfig{
		MaxPagesInMemory: 5,
		PageBufferSize:   1024 * 1024, // 1MB per page buffer
		EnableProgress:   true,
		ExtractText:      true,
		ExtractImages:    true,
		ExtractForms:     true,
		CallbackMode:     "sequential",
	}
}

// StreamPage represents a single page being processed
type StreamPage struct {
	Number      int          `json:"number"`
	Offset      int64        `json:"offset"`
	Length      int64        `json:"length"`
	Objects     []PDFObject  `json:"objects"`
	Content     PageContent  `json:"content"`
	Metadata    PageMetadata `json:"metadata"`
	ProcessedAt int64        `json:"processed_at"`
	Status      string       `json:"status"`
	Error       string       `json:"error,omitempty"`
}

// PageContent contains extracted content from a page
type PageContent struct {
	Text       string      `json:"text"`
	Images     []ImageInfo `json:"images"`
	Forms      []FormInfo  `json:"forms"`
	TextBlocks []TextBlock `json:"text_blocks,omitempty"`
}

// PageMetadata contains metadata about a page
type PageMetadata struct {
	MediaBox    Rectangle `json:"media_box"`
	CropBox     Rectangle `json:"crop_box,omitempty"`
	Rotation    int       `json:"rotation"`
	Resources   []string  `json:"resources,omitempty"`
	Annotations int       `json:"annotations"`
	HasImages   bool      `json:"has_images"`
	HasForms    bool      `json:"has_forms"`
	TextLength  int       `json:"text_length"`
}

// TextBlock represents a block of text with positioning
type TextBlock struct {
	Text     string  `json:"text"`
	X        float64 `json:"x"`
	Y        float64 `json:"y"`
	Width    float64 `json:"width"`
	Height   float64 `json:"height"`
	FontSize float64 `json:"font_size"`
	FontName string  `json:"font_name,omitempty"`
}

// PageCallback is called for each processed page
type PageCallback func(*StreamPage) error

// NewPageStreamer creates a new page streamer
func NewPageStreamer(parser *StreamParser, config ...PageStreamerConfig) *PageStreamer {
	cfg := DefaultPageStreamerConfig()
	if len(config) > 0 {
		cfg = config[0]
	}

	return &PageStreamer{
		parser:      parser,
		pageSize:    cfg.MaxPagesInMemory,
		pageOffsets: make(map[int]int64),
		pageObjects: make(map[int][]PDFObject),
		config:      cfg,
		currentPage: 1,
	}
}

// StreamPages processes all pages in the PDF using the provided callback
func (ps *PageStreamer) StreamPages(ctx context.Context, callback PageCallback) error {
	if callback == nil {
		return fmt.Errorf("callback cannot be nil")
	}

	// First pass: discover all pages and their locations
	if err := ps.discoverPages(ctx); err != nil {
		return fmt.Errorf("failed to discover pages: %w", err)
	}

	// Second pass: process each page
	for pageNum := 1; pageNum <= ps.totalPages; pageNum++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		page, err := ps.processPage(ctx, pageNum)
		if err != nil {
			// Log error but continue with other pages
			page = &StreamPage{
				Number: pageNum,
				Status: "error",
				Error:  err.Error(),
			}
		}

		// Call the callback for this page
		if err := callback(page); err != nil {
			return fmt.Errorf("callback failed for page %d: %w", pageNum, err)
		}

		// Release page memory if we're over the limit
		ps.maybeReleasePage(pageNum)
	}

	return nil
}

// StreamPagesWithProgress processes pages with progress reporting
func (ps *PageStreamer) StreamPagesWithProgress(ctx context.Context, callback PageCallback,
	progressCallback func(PageProgress),
) error {
	if callback == nil {
		return fmt.Errorf("callback cannot be nil")
	}

	// Discover pages first
	if err := ps.discoverPages(ctx); err != nil {
		return fmt.Errorf("failed to discover pages: %w", err)
	}

	// Process each page with progress
	for pageNum := 1; pageNum <= ps.totalPages; pageNum++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Report progress
		if progressCallback != nil {
			progress := PageProgress{
				CurrentPage:  pageNum,
				TotalPages:   ps.totalPages,
				PercentDone:  float64(pageNum-1) / float64(ps.totalPages) * 100,
				PagesInQueue: len(ps.pageQueue),
			}
			progressCallback(progress)
		}

		page, err := ps.processPage(ctx, pageNum)
		if err != nil {
			page = &StreamPage{
				Number: pageNum,
				Status: "error",
				Error:  err.Error(),
			}
		}

		if err := callback(page); err != nil {
			return fmt.Errorf("callback failed for page %d: %w", pageNum, err)
		}

		ps.maybeReleasePage(pageNum)
	}

	// Final progress report
	if progressCallback != nil {
		progress := PageProgress{
			CurrentPage:  ps.totalPages,
			TotalPages:   ps.totalPages,
			PercentDone:  100.0,
			PagesInQueue: 0,
		}
		progressCallback(progress)
	}

	return nil
}

// GetPageCount returns the total number of pages discovered
func (ps *PageStreamer) GetPageCount() int {
	return ps.totalPages
}

// GetCurrentPage returns the current page being processed
func (ps *PageStreamer) GetCurrentPage() int {
	return ps.currentPage
}

// GetPageInfo returns information about a specific page without processing it
func (ps *PageStreamer) GetPageInfo(pageNum int) (*PageInfo, error) {
	if pageNum < 1 || pageNum > ps.totalPages {
		return nil, fmt.Errorf("invalid page number: %d (total: %d)", pageNum, ps.totalPages)
	}

	offset, exists := ps.pageOffsets[pageNum]
	if !exists {
		return nil, fmt.Errorf("page %d not found in offset table", pageNum)
	}

	return &PageInfo{
		Number: pageNum,
		Offset: offset,
		Length: ps.estimatePageLength(pageNum),
	}, nil
}

// Reset resets the page streamer for a new document
func (ps *PageStreamer) Reset() {
	ps.pageMutex.Lock()
	defer ps.pageMutex.Unlock()

	ps.currentPage = 1
	ps.totalPages = 0
	ps.pageQueue = nil
	ps.pageOffsets = make(map[int]int64)
	ps.pageObjects = make(map[int][]PDFObject)
}

// Internal methods

// discoverPages scans the PDF to find all page locations
func (ps *PageStreamer) discoverPages(ctx context.Context) error {
	pageCount := 0

	// Look for page tree and individual page objects
	err := ps.parser.ProcessInChunks(func(chunk []byte, offset int64) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Find page objects in this chunk
		pages := ps.findPageObjects(chunk, offset)
		for _, page := range pages {
			pageCount++
			ps.pageOffsets[pageCount] = page.Offset
			ps.pageObjects[pageCount] = []PDFObject{page}
		}

		return nil
	})
	if err != nil {
		return err
	}

	ps.totalPages = pageCount
	return nil
}

// findPageObjects finds page objects in a chunk of data
func (ps *PageStreamer) findPageObjects(data []byte, offset int64) []PDFObject {
	var pages []PDFObject

	// Look for page object patterns
	pageRegex := regexp.MustCompile(`(\d+)\s+(\d+)\s+obj\s*<<[^>]*\/Type\s*\/Page[^>]*>>`)
	matches := pageRegex.FindAllSubmatch(data, -1)

	for _, match := range matches {
		if len(match) >= 3 {
			objNum, _ := strconv.Atoi(string(match[1]))
			objGen, _ := strconv.Atoi(string(match[2]))

			// Calculate approximate offset of this match
			matchOffset := offset + int64(strings.Index(string(data), string(match[0])))

			page := PDFObject{
				Number:     objNum,
				Generation: objGen,
				Offset:     matchOffset,
				Content:    string(match[0]),
			}

			pages = append(pages, page)
		}
	}

	return pages
}

// processPage processes a single page and extracts its content
func (ps *PageStreamer) processPage(ctx context.Context, pageNum int) (*StreamPage, error) {
	offset, exists := ps.pageOffsets[pageNum]
	if !exists {
		return nil, fmt.Errorf("page %d not found", pageNum)
	}

	// Seek to page location
	if _, err := ps.parser.Seek(offset, io.SeekStart); err != nil {
		return nil, fmt.Errorf("failed to seek to page %d: %w", pageNum, err)
	}

	// Read page content
	pageData, err := ps.parser.ReadChunk()
	if err != nil {
		return nil, fmt.Errorf("failed to read page %d: %w", pageNum, err)
	}

	// Process page content
	page := &StreamPage{
		Number:      pageNum,
		Offset:      offset,
		Length:      int64(len(pageData)),
		Status:      "processing",
		ProcessedAt: ps.getCurrentTime(),
	}

	// Extract content based on configuration
	if ps.config.ExtractText {
		page.Content.Text = ps.extractPageText(pageData)
		page.Content.TextBlocks = ps.extractTextBlocks(pageData)
	}

	if ps.config.ExtractImages {
		page.Content.Images = ps.extractPageImages(pageData, offset)
	}

	if ps.config.ExtractForms {
		page.Content.Forms = ps.extractPageForms(pageData, offset)
	}

	// Extract metadata
	page.Metadata = ps.extractPageMetadata(pageData)
	page.Metadata.TextLength = len(page.Content.Text)
	page.Metadata.HasImages = len(page.Content.Images) > 0
	page.Metadata.HasForms = len(page.Content.Forms) > 0

	page.Status = "completed"
	return page, nil
}

// extractPageText extracts text content from page data
func (ps *PageStreamer) extractPageText(pageData []byte) string {
	var textParts []string

	// Look for text showing operators
	textRegex := regexp.MustCompile(`\((.*?)\)\s*Tj`)
	matches := textRegex.FindAllSubmatch(pageData, -1)

	for _, match := range matches {
		if len(match) > 1 {
			textParts = append(textParts, string(match[1]))
		}
	}

	return strings.Join(textParts, " ")
}

// extractTextBlocks extracts positioned text blocks
func (ps *PageStreamer) extractTextBlocks(pageData []byte) []TextBlock {
	var blocks []TextBlock

	// Simple text block extraction - can be enhanced
	textRegex := regexp.MustCompile(`(\d+(?:\.\d+)?)\s+(\d+(?:\.\d+)?)\s+Td\s*\((.*?)\)\s*Tj`)
	matches := textRegex.FindAllSubmatch(pageData, -1)

	for _, match := range matches {
		if len(match) >= 4 {
			x, _ := strconv.ParseFloat(string(match[1]), 64)
			y, _ := strconv.ParseFloat(string(match[2]), 64)
			text := string(match[3])

			block := TextBlock{
				Text:   text,
				X:      x,
				Y:      y,
				Width:  float64(len(text)) * 6, // Rough estimate
				Height: 12,                     // Default height
			}
			blocks = append(blocks, block)
		}
	}

	return blocks
}

// extractPageImages extracts image references from page data
func (ps *PageStreamer) extractPageImages(pageData []byte, offset int64) []ImageInfo {
	var images []ImageInfo

	// Look for image references
	imageRegex := regexp.MustCompile(`/Im(\d+)\s+Do`)
	matches := imageRegex.FindAllSubmatch(pageData, -1)

	for _, match := range matches {
		if len(match) > 1 {
			imageNum, _ := strconv.Atoi(string(match[1]))
			image := ImageInfo{
				ObjectNumber: imageNum,
				Offset:       offset,
				Length:       0, // Will be determined later
			}
			images = append(images, image)
		}
	}

	return images
}

// extractPageForms extracts form field references from page data
func (ps *PageStreamer) extractPageForms(pageData []byte, offset int64) []FormInfo {
	var forms []FormInfo

	// Look for annotation/form references
	formRegex := regexp.MustCompile(`/Annot\d+\s+(\d+)\s+(\d+)\s+R`)
	matches := formRegex.FindAllSubmatch(pageData, -1)

	for _, match := range matches {
		if len(match) >= 3 {
			objNum, _ := strconv.Atoi(string(match[1]))
			form := FormInfo{
				ObjectNumber: objNum,
				Offset:       offset,
				FieldType:    "Unknown",
			}
			forms = append(forms, form)
		}
	}

	return forms
}

// extractPageMetadata extracts metadata from page data
func (ps *PageStreamer) extractPageMetadata(pageData []byte) PageMetadata {
	metadata := PageMetadata{}

	// Extract MediaBox
	mediaBoxRegex := regexp.MustCompile(`/MediaBox\s*\[\s*(\d+(?:\.\d+)?)\s+(\d+(?:\.\d+)?)\s+(\d+(?:\.\d+)?)\s+(\d+(?:\.\d+)?)\s*\]`)
	if matches := mediaBoxRegex.FindSubmatch(pageData); len(matches) >= 5 {
		x, _ := strconv.ParseFloat(string(matches[1]), 64)
		y, _ := strconv.ParseFloat(string(matches[2]), 64)
		width, _ := strconv.ParseFloat(string(matches[3]), 64)
		height, _ := strconv.ParseFloat(string(matches[4]), 64)

		metadata.MediaBox = Rectangle{
			X:      x,
			Y:      y,
			Width:  width,
			Height: height,
		}
	}

	// Extract rotation
	rotateRegex := regexp.MustCompile(`/Rotate\s+(\d+)`)
	if matches := rotateRegex.FindSubmatch(pageData); len(matches) >= 2 {
		rotation, _ := strconv.Atoi(string(matches[1]))
		metadata.Rotation = rotation
	}

	return metadata
}

// maybeReleasePage releases a page from memory if we're over the limit
func (ps *PageStreamer) maybeReleasePage(pageNum int) {
	ps.pageMutex.Lock()
	defer ps.pageMutex.Unlock()

	if len(ps.pageQueue) >= ps.pageSize {
		// Remove oldest page
		if len(ps.pageQueue) > 0 {
			ps.pageQueue = ps.pageQueue[1:]
		}
	}
}

// estimatePageLength estimates the length of a page
func (ps *PageStreamer) estimatePageLength(pageNum int) int64 {
	// This is a rough estimate - could be improved with actual parsing
	return 4096 // 4KB average page size
}

// getCurrentTime returns current timestamp
func (ps *PageStreamer) getCurrentTime() int64 {
	return int64(1000) // Simplified - would use time.Now() in real implementation
}

// PageProgress provides progress information for page processing
type PageProgress struct {
	CurrentPage  int     `json:"current_page"`
	TotalPages   int     `json:"total_pages"`
	PercentDone  float64 `json:"percent_done"`
	PagesInQueue int     `json:"pages_in_queue"`
}
