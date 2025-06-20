package streaming

import (
	"strings"
	"sync"
)

// PDFObject represents a PDF object found in the stream
type PDFObject struct {
	Number     int    `json:"number"`
	Generation int    `json:"generation"`
	Offset     int64  `json:"offset"`
	Length     int64  `json:"length"`
	Content    string `json:"content"`
}

// XRefEntry represents an entry in the cross-reference table
type XRefEntry struct {
	Offset     int64 `json:"offset"`
	Generation int   `json:"generation"`
	InUse      bool  `json:"in_use"`
}

// ChunkResult contains the results of processing a single chunk
type ChunkResult struct {
	Offset      int64                  `json:"offset"`
	Size        int64                  `json:"size"`
	ObjectCount int                    `json:"object_count"`
	TextLength  int                    `json:"text_length"`
	ImageCount  int                    `json:"image_count"`
	FormCount   int                    `json:"form_count"`
	Processed   map[string]interface{} `json:"processed"`
}

// ProcessedContent contains all extracted content from the document
type ProcessedContent struct {
	Text   string      `json:"text"`
	Images []ImageInfo `json:"images"`
	Forms  []FormInfo  `json:"forms"`
	Pages  []PageInfo  `json:"pages"`
}

// ProcessingProgress provides information about processing progress
type ProcessingProgress struct {
	CurrentPage  int `json:"current_page"`
	TextSize     int `json:"text_size"`
	ImageCount   int `json:"image_count"`
	FormCount    int `json:"form_count"`
	ObjectsFound int `json:"objects_found"`
	XRefEntries  int `json:"xref_entries"`
}

// Rectangle represents a rectangular area
type Rectangle struct {
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

// PageInfo contains information about a single page
type PageInfo struct {
	Number   int       `json:"number"`
	Offset   int64     `json:"offset"`
	Length   int64     `json:"length"`
	MediaBox Rectangle `json:"media_box"`
}

// ImageInfo contains information about an image
type ImageInfo struct {
	ObjectNumber int    `json:"object_number"`
	Offset       int64  `json:"offset"`
	Length       int64  `json:"length"`
	Width        int    `json:"width"`
	Height       int    `json:"height"`
	Format       string `json:"format,omitempty"`
}

// FormInfo contains information about a form field
type FormInfo struct {
	ObjectNumber int    `json:"object_number"`
	Offset       int64  `json:"offset"`
	FieldType    string `json:"field_type"`
	FieldName    string `json:"field_name"`
	FieldValue   string `json:"field_value"`
}

// PageBuffer manages page information with memory limits
type PageBuffer struct {
	pages    []PageInfo
	maxPages int
	mutex    sync.RWMutex
}

// NewPageBuffer creates a new page buffer
func NewPageBuffer(maxSize int) *PageBuffer {
	maxPages := maxSize / 1024 // Rough estimate: 1KB per page info
	if maxPages < 10 {
		maxPages = 10 // Minimum of 10 pages
	}

	return &PageBuffer{
		pages:    make([]PageInfo, 0),
		maxPages: maxPages,
	}
}

// AddPage adds a page to the buffer
func (pb *PageBuffer) AddPage(page PageInfo) {
	pb.mutex.Lock()
	defer pb.mutex.Unlock()

	if len(pb.pages) >= pb.maxPages {
		// Remove oldest page to make room
		pb.pages = pb.pages[1:]
	}

	pb.pages = append(pb.pages, page)
}

// Flush returns all pages and clears the buffer
func (pb *PageBuffer) Flush() []PageInfo {
	pb.mutex.Lock()
	defer pb.mutex.Unlock()

	result := make([]PageInfo, len(pb.pages))
	copy(result, pb.pages)
	pb.pages = pb.pages[:0]

	return result
}

// Clear removes all pages from the buffer
func (pb *PageBuffer) Clear() {
	pb.mutex.Lock()
	defer pb.mutex.Unlock()

	pb.pages = pb.pages[:0]
}

// Len returns the number of pages in the buffer
func (pb *PageBuffer) Len() int {
	pb.mutex.RLock()
	defer pb.mutex.RUnlock()

	return len(pb.pages)
}

// TextBuffer manages text content with size limits
type TextBuffer struct {
	content strings.Builder
	maxSize int
	mutex   sync.RWMutex
}

// NewTextBuffer creates a new text buffer
func NewTextBuffer(maxSize int) *TextBuffer {
	return &TextBuffer{
		maxSize: maxSize,
	}
}

// Append adds text to the buffer
func (tb *TextBuffer) Append(text string) {
	tb.mutex.Lock()
	defer tb.mutex.Unlock()

	// Check if adding this text would exceed the limit
	if tb.content.Len()+len(text) > tb.maxSize {
		// Calculate how much we can add
		remaining := tb.maxSize - tb.content.Len()
		if remaining > 0 {
			tb.content.WriteString(text[:remaining])
		}
		return
	}

	tb.content.WriteString(text)
}

// Flush returns all text and clears the buffer
func (tb *TextBuffer) Flush() string {
	tb.mutex.Lock()
	defer tb.mutex.Unlock()

	result := tb.content.String()
	tb.content.Reset()

	return result
}

// Clear removes all text from the buffer
func (tb *TextBuffer) Clear() {
	tb.mutex.Lock()
	defer tb.mutex.Unlock()

	tb.content.Reset()
}

// Len returns the current text length
func (tb *TextBuffer) Len() int {
	tb.mutex.RLock()
	defer tb.mutex.RUnlock()

	return tb.content.Len()
}

// String returns the current text content without clearing
func (tb *TextBuffer) String() string {
	tb.mutex.RLock()
	defer tb.mutex.RUnlock()

	return tb.content.String()
}

// ImageBuffer manages image information with count limits
type ImageBuffer struct {
	images    []ImageInfo
	maxImages int
	mutex     sync.RWMutex
}

// NewImageBuffer creates a new image buffer
func NewImageBuffer(maxImages int) *ImageBuffer {
	return &ImageBuffer{
		images:    make([]ImageInfo, 0),
		maxImages: maxImages,
	}
}

// AddImage adds an image to the buffer
func (ib *ImageBuffer) AddImage(image ImageInfo) {
	ib.mutex.Lock()
	defer ib.mutex.Unlock()

	if len(ib.images) >= ib.maxImages {
		// Remove oldest image to make room
		ib.images = ib.images[1:]
	}

	ib.images = append(ib.images, image)
}

// Flush returns all images and clears the buffer
func (ib *ImageBuffer) Flush() []ImageInfo {
	ib.mutex.Lock()
	defer ib.mutex.Unlock()

	result := make([]ImageInfo, len(ib.images))
	copy(result, ib.images)
	ib.images = ib.images[:0]

	return result
}

// Clear removes all images from the buffer
func (ib *ImageBuffer) Clear() {
	ib.mutex.Lock()
	defer ib.mutex.Unlock()

	ib.images = ib.images[:0]
}

// Len returns the number of images in the buffer
func (ib *ImageBuffer) Len() int {
	ib.mutex.RLock()
	defer ib.mutex.RUnlock()

	return len(ib.images)
}

// FormBuffer manages form field information with count limits
type FormBuffer struct {
	forms    []FormInfo
	maxForms int
	mutex    sync.RWMutex
}

// NewFormBuffer creates a new form buffer
func NewFormBuffer(maxForms int) *FormBuffer {
	return &FormBuffer{
		forms:    make([]FormInfo, 0),
		maxForms: maxForms,
	}
}

// AddForm adds a form field to the buffer
func (fb *FormBuffer) AddForm(form FormInfo) {
	fb.mutex.Lock()
	defer fb.mutex.Unlock()

	if len(fb.forms) >= fb.maxForms {
		// Remove oldest form to make room
		fb.forms = fb.forms[1:]
	}

	fb.forms = append(fb.forms, form)
}

// Flush returns all forms and clears the buffer
func (fb *FormBuffer) Flush() []FormInfo {
	fb.mutex.Lock()
	defer fb.mutex.Unlock()

	result := make([]FormInfo, len(fb.forms))
	copy(result, fb.forms)
	fb.forms = fb.forms[:0]

	return result
}

// Clear removes all forms from the buffer
func (fb *FormBuffer) Clear() {
	fb.mutex.Lock()
	defer fb.mutex.Unlock()

	fb.forms = fb.forms[:0]
}

// Len returns the number of forms in the buffer
func (fb *FormBuffer) Len() int {
	fb.mutex.RLock()
	defer fb.mutex.RUnlock()

	return len(fb.forms)
}

// StreamingResult represents the final result of streaming processing
type StreamingResult struct {
	Content         *ProcessedContent  `json:"content"`
	Progress        ProcessingProgress `json:"progress"`
	MemoryStats     MemoryStats        `json:"memory_stats"`
	ProcessingStats ProcessingStats    `json:"processing_stats"`
}

// ProcessingStats provides statistics about the processing operation
type ProcessingStats struct {
	TotalChunks     int   `json:"total_chunks"`
	ProcessedChunks int   `json:"processed_chunks"`
	TotalObjects    int   `json:"total_objects"`
	ProcessingTime  int64 `json:"processing_time_ms"`
	BytesProcessed  int64 `json:"bytes_processed"`
}

// StreamingConfig configures the entire streaming operation
type StreamingConfig struct {
	ChunkSize      int64 `json:"chunk_size_bytes"`
	MaxMemory      int64 `json:"max_memory_bytes"`
	ExtractText    bool  `json:"extract_text"`
	ExtractImages  bool  `json:"extract_images"`
	ExtractForms   bool  `json:"extract_forms"`
	PreserveFormat bool  `json:"preserve_format"`
	EnableCaching  bool  `json:"enable_caching"`
	CacheSize      int   `json:"cache_size"`
	BufferPoolSize int   `json:"buffer_pool_size"`
}

// DefaultStreamingConfig returns sensible defaults for streaming operations
func DefaultStreamingConfig() StreamingConfig {
	return StreamingConfig{
		ChunkSize:      1024 * 1024,      // 1MB chunks
		MaxMemory:      64 * 1024 * 1024, // 64MB max memory
		ExtractText:    true,
		ExtractImages:  true,
		ExtractForms:   true,
		PreserveFormat: false,
		EnableCaching:  true,
		CacheSize:      1000,
		BufferPoolSize: 10,
	}
}
