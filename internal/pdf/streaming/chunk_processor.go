package streaming

import (
	"bufio"
	"bytes"
	"fmt"
	"regexp"
	"strings"
	"sync"
)

// ChunkProcessor handles processing of PDF chunks for content extraction
type ChunkProcessor struct {
	parser      *StreamParser
	pageBuffer  *PageBuffer
	textBuffer  *TextBuffer
	imageBuffer *ImageBuffer
	formBuffer  *FormBuffer

	// Processing state
	currentPage int
	xrefTable   map[int64]XRefEntry
	objectCache map[string]PDFObject

	// Configuration
	config ProcessorConfig
	mutex  sync.RWMutex
}

// ProcessorConfig configures the chunk processor behavior
type ProcessorConfig struct {
	MaxTextBufferSize  int  // Maximum text buffer size in bytes
	MaxImageBuffer     int  // Maximum number of images to buffer
	MaxFormFields      int  // Maximum number of form fields to buffer
	ExtractImages      bool // Whether to extract images
	ExtractForms       bool // Whether to extract forms
	ExtractText        bool // Whether to extract text (default: true)
	PreserveFormatting bool // Whether to preserve text formatting
}

// DefaultProcessorConfig returns sensible defaults
func DefaultProcessorConfig() ProcessorConfig {
	return ProcessorConfig{
		MaxTextBufferSize:  5 * 1024 * 1024, // 5MB
		MaxImageBuffer:     100,             // 100 images
		MaxFormFields:      500,             // 500 form fields
		ExtractImages:      true,
		ExtractForms:       true,
		ExtractText:        true,
		PreserveFormatting: false,
	}
}

// NewChunkProcessor creates a new chunk processor
func NewChunkProcessor(parser *StreamParser, config ...ProcessorConfig) *ChunkProcessor {
	cfg := DefaultProcessorConfig()
	if len(config) > 0 {
		cfg = config[0]
	}

	return &ChunkProcessor{
		parser:      parser,
		pageBuffer:  NewPageBuffer(cfg.MaxTextBufferSize),
		textBuffer:  NewTextBuffer(cfg.MaxTextBufferSize),
		imageBuffer: NewImageBuffer(cfg.MaxImageBuffer),
		formBuffer:  NewFormBuffer(cfg.MaxFormFields),
		xrefTable:   make(map[int64]XRefEntry),
		objectCache: make(map[string]PDFObject),
		config:      cfg,
		currentPage: 1,
	}
}

// ProcessChunk processes a chunk of PDF data
func (cp *ChunkProcessor) ProcessChunk(data []byte, offset int64) (*ChunkResult, error) {
	cp.mutex.Lock()
	defer cp.mutex.Unlock()

	result := &ChunkResult{
		Offset:    offset,
		Size:      int64(len(data)),
		Processed: make(map[string]interface{}),
	}

	// Find PDF objects in this chunk
	objects, err := cp.findPDFObjects(data, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to find PDF objects: %w", err)
	}

	// Process each object found
	for _, obj := range objects {
		if err := cp.processObject(obj); err != nil {
			// Log error but continue processing
			continue
		}
	}

	// Extract content from processed objects
	if cp.config.ExtractText {
		textContent := cp.extractTextContent(objects)
		if len(textContent) > 0 {
			result.Processed["text"] = textContent
			// Also add to text buffer for accumulation across chunks
			cp.textBuffer.Append(textContent + " ")
		}
	}

	if cp.config.ExtractImages {
		images := cp.extractImageContent(objects)
		if len(images) > 0 {
			result.Processed["images"] = images
		}
	}

	if cp.config.ExtractForms {
		forms := cp.extractFormContent(objects)
		if len(forms) > 0 {
			result.Processed["forms"] = forms
		}
	}

	// Update statistics
	result.ObjectCount = len(objects)
	result.TextLength = cp.textBuffer.Len()
	result.ImageCount = cp.imageBuffer.Len()
	result.FormCount = cp.formBuffer.Len()

	return result, nil
}

// FlushBuffers flushes all content buffers and returns the accumulated content
func (cp *ChunkProcessor) FlushBuffers() (*ProcessedContent, error) {
	cp.mutex.Lock()
	defer cp.mutex.Unlock()

	content := &ProcessedContent{
		Text:   cp.textBuffer.Flush(),
		Images: cp.imageBuffer.Flush(),
		Forms:  cp.formBuffer.Flush(),
		Pages:  cp.pageBuffer.Flush(),
	}

	return content, nil
}

// GetProgress returns processing progress information
func (cp *ChunkProcessor) GetProgress() ProcessingProgress {
	cp.mutex.RLock()
	defer cp.mutex.RUnlock()

	return ProcessingProgress{
		CurrentPage:  cp.currentPage,
		TextSize:     cp.textBuffer.Len(),
		ImageCount:   cp.imageBuffer.Len(),
		FormCount:    cp.formBuffer.Len(),
		ObjectsFound: len(cp.objectCache),
		XRefEntries:  len(cp.xrefTable),
	}
}

// Reset resets the processor state for a new document
func (cp *ChunkProcessor) Reset() {
	cp.mutex.Lock()
	defer cp.mutex.Unlock()

	cp.currentPage = 1
	cp.xrefTable = make(map[int64]XRefEntry)
	cp.objectCache = make(map[string]PDFObject)
	cp.pageBuffer.Clear()
	cp.textBuffer.Clear()
	cp.imageBuffer.Clear()
	cp.formBuffer.Clear()
}

// Internal methods

// findPDFObjects scans chunk data for PDF objects
func (cp *ChunkProcessor) findPDFObjects(data []byte, offset int64) ([]PDFObject, error) {
	var objects []PDFObject

	// Regular expressions for PDF object patterns
	objStartRegex := regexp.MustCompile(`(\d+)\s+(\d+)\s+obj`)
	objEndRegex := regexp.MustCompile(`endobj`)

	scanner := bufio.NewScanner(bytes.NewReader(data))
	scanner.Split(bufio.ScanLines)

	var currentObj *PDFObject
	var objContent strings.Builder
	lineOffset := offset

	for scanner.Scan() {
		line := scanner.Text()

		// Check for object start
		if matches := objStartRegex.FindStringSubmatch(line); matches != nil {
			// Finish previous object if exists
			if currentObj != nil {
				currentObj.Content = objContent.String()
				objects = append(objects, *currentObj)
			}

			// Start new object
			currentObj = &PDFObject{
				Number:     parseInt(matches[1]),
				Generation: parseInt(matches[2]),
				Offset:     lineOffset,
			}
			objContent.Reset()
		}

		// Check for object end
		if objEndRegex.MatchString(line) && currentObj != nil {
			currentObj.Content = objContent.String()
			currentObj.Length = int64(len(objContent.String()))
			objects = append(objects, *currentObj)
			currentObj = nil
			objContent.Reset()
		} else if currentObj != nil {
			// Accumulate object content
			objContent.WriteString(line)
			objContent.WriteString("\n")
		}

		lineOffset += int64(len(line)) + 1 // +1 for newline
	}

	return objects, nil
}

// processObject processes a single PDF object
func (cp *ChunkProcessor) processObject(obj PDFObject) error {
	// Cache the object
	key := fmt.Sprintf("%d_%d", obj.Number, obj.Generation)
	cp.objectCache[key] = obj

	// Determine object type and process accordingly
	objType := cp.determineObjectType(obj.Content)

	switch objType {
	case "Page":
		return cp.processPageObject(obj)
	case "Text":
		return cp.processTextObject(obj)
	case "Image":
		return cp.processImageObject(obj)
	case "Form":
		return cp.processFormObject(obj)
	case "XRef":
		return cp.processXRefObject(obj)
	default:
		// Unknown object type, store for later analysis
		return nil
	}
}

// determineObjectType analyzes object content to determine its type
func (cp *ChunkProcessor) determineObjectType(content string) string {
	content = strings.ToLower(content)

	if strings.Contains(content, "/type /page") {
		return "Page"
	}
	if strings.Contains(content, "/subtype /image") {
		return "Image"
	}
	if strings.Contains(content, "/type /annot") || strings.Contains(content, "/ft /") {
		return "Form"
	}
	if strings.Contains(content, "bt ") || strings.Contains(content, "tj ") || strings.Contains(content, "td ") {
		return "Text"
	}
	if strings.Contains(content, "xref") {
		return "XRef"
	}

	return "Unknown"
}

// processPageObject processes a page object
func (cp *ChunkProcessor) processPageObject(obj PDFObject) error {
	pageInfo := PageInfo{
		Number: cp.currentPage,
		Offset: obj.Offset,
		Length: obj.Length,
	}

	// Extract page dimensions if available
	if mediaBox := cp.extractMediaBox(obj.Content); mediaBox != nil {
		pageInfo.MediaBox = *mediaBox
	}

	cp.pageBuffer.AddPage(pageInfo)
	cp.currentPage++

	return nil
}

// processTextObject processes text content objects
func (cp *ChunkProcessor) processTextObject(obj PDFObject) error {
	text := cp.extractTextFromObject(obj.Content)
	if text != "" {
		cp.textBuffer.Append(text)
	}
	return nil
}

// processImageObject processes image objects
func (cp *ChunkProcessor) processImageObject(obj PDFObject) error {
	if !cp.config.ExtractImages {
		return nil
	}

	imageInfo := ImageInfo{
		ObjectNumber: obj.Number,
		Offset:       obj.Offset,
		Length:       obj.Length,
	}

	// Extract image properties
	if width := cp.extractImageWidth(obj.Content); width > 0 {
		imageInfo.Width = width
	}
	if height := cp.extractImageHeight(obj.Content); height > 0 {
		imageInfo.Height = height
	}

	cp.imageBuffer.AddImage(imageInfo)
	return nil
}

// processFormObject processes form field objects
func (cp *ChunkProcessor) processFormObject(obj PDFObject) error {
	if !cp.config.ExtractForms {
		return nil
	}

	formInfo := FormInfo{
		ObjectNumber: obj.Number,
		Offset:       obj.Offset,
		FieldType:    cp.extractFormFieldType(obj.Content),
		FieldName:    cp.extractFormFieldName(obj.Content),
		FieldValue:   cp.extractFormFieldValue(obj.Content),
	}

	cp.formBuffer.AddForm(formInfo)
	return nil
}

// processXRefObject processes cross-reference table objects
func (cp *ChunkProcessor) processXRefObject(obj PDFObject) error {
	entries := cp.parseXRefEntries(obj.Content)
	for offset, entry := range entries {
		cp.xrefTable[offset] = entry
	}
	return nil
}

// extractTextContent extracts text from processed objects
// extractTextContent extracts text from processed objects
func (cp *ChunkProcessor) extractTextContent(objects []PDFObject) string {
	var textParts []string

	for _, obj := range objects {
		// Try multiple text extraction methods
		if text := cp.extractTextFromObject(obj.Content); text != "" {
			textParts = append(textParts, text)
		}

		// Also try to extract text from any object that might contain text
		if cp.containsTextContent(obj.Content) {
			if text := cp.extractTextFallback(obj.Content); text != "" {
				textParts = append(textParts, text)
			}
		}
	}

	return strings.Join(textParts, " ")
}

// extractImageContent extracts image information from objects
func (cp *ChunkProcessor) extractImageContent(objects []PDFObject) []ImageInfo {
	var images []ImageInfo

	for _, obj := range objects {
		if cp.determineObjectType(obj.Content) == "Image" {
			imageInfo := ImageInfo{
				ObjectNumber: obj.Number,
				Offset:       obj.Offset,
				Length:       obj.Length,
				Width:        cp.extractImageWidth(obj.Content),
				Height:       cp.extractImageHeight(obj.Content),
			}
			images = append(images, imageInfo)
		}
	}

	return images
}

// extractFormContent extracts form information from objects
func (cp *ChunkProcessor) extractFormContent(objects []PDFObject) []FormInfo {
	var forms []FormInfo

	for _, obj := range objects {
		if cp.determineObjectType(obj.Content) == "Form" {
			formInfo := FormInfo{
				ObjectNumber: obj.Number,
				Offset:       obj.Offset,
				FieldType:    cp.extractFormFieldType(obj.Content),
				FieldName:    cp.extractFormFieldName(obj.Content),
				FieldValue:   cp.extractFormFieldValue(obj.Content),
			}
			forms = append(forms, formInfo)
		}
	}

	return forms
}

// Helper methods for content extraction

func (cp *ChunkProcessor) extractTextFromObject(content string) string {
	var textParts []string

	// Multiple text extraction patterns for different PDF text operators
	patterns := []string{
		`\((.*?)\)\s*Tj`, // Standard text showing
		`\((.*?)\)\s*TJ`, // Text showing with adjustments
		`\[(.*?)\]\s*TJ`, // Array of text strings
		`\((.*?)\)\s*'`,  // Move to next line and show text
		`\((.*?)\)\s*"`,  // Set word and character spacing, move to next line and show text
	}

	for _, pattern := range patterns {
		textRegex := regexp.MustCompile(pattern)
		matches := textRegex.FindAllStringSubmatch(content, -1)

		for _, match := range matches {
			if len(match) > 1 && strings.TrimSpace(match[1]) != "" {
				textParts = append(textParts, strings.TrimSpace(match[1]))
			}
		}
	}

	return strings.Join(textParts, " ")
}

func (cp *ChunkProcessor) extractMediaBox(content string) *Rectangle {
	mediaBoxRegex := regexp.MustCompile(`/MediaBox\s*\[\s*(\d+(?:\.\d+)?)\s+(\d+(?:\.\d+)?)\s+(\d+(?:\.\d+)?)\s+(\d+(?:\.\d+)?)\s*\]`)
	matches := mediaBoxRegex.FindStringSubmatch(content)

	if len(matches) == 5 {
		return &Rectangle{
			X:      parseFloat(matches[1]),
			Y:      parseFloat(matches[2]),
			Width:  parseFloat(matches[3]),
			Height: parseFloat(matches[4]),
		}
	}

	return nil
}

func (cp *ChunkProcessor) extractImageWidth(content string) int {
	widthRegex := regexp.MustCompile(`/Width\s+(\d+)`)
	matches := widthRegex.FindStringSubmatch(content)
	if len(matches) > 1 {
		return parseInt(matches[1])
	}
	return 0
}

func (cp *ChunkProcessor) extractImageHeight(content string) int {
	heightRegex := regexp.MustCompile(`/Height\s+(\d+)`)
	matches := heightRegex.FindStringSubmatch(content)
	if len(matches) > 1 {
		return parseInt(matches[1])
	}
	return 0
}

// containsTextContent checks if content might contain text
func (cp *ChunkProcessor) containsTextContent(content string) bool {
	textIndicators := []string{"BT", "ET", "Tj", "TJ", "Td", "TD", "Tm", "T*"}
	for _, indicator := range textIndicators {
		if strings.Contains(content, indicator) {
			return true
		}
	}
	return false
}

// extractTextFallback provides fallback text extraction for edge cases
func (cp *ChunkProcessor) extractTextFallback(content string) string {
	var textParts []string

	// Look for text between BT/ET markers
	btEtRegex := regexp.MustCompile(`BT\s(.*?)\sET`)
	matches := btEtRegex.FindAllStringSubmatch(content, -1)

	for _, match := range matches {
		if len(match) > 1 {
			// Extract any parenthetical content within BT/ET blocks
			innerRegex := regexp.MustCompile(`\(([^)]+)\)`)
			innerMatches := innerRegex.FindAllStringSubmatch(match[1], -1)

			for _, innerMatch := range innerMatches {
				if len(innerMatch) > 1 && strings.TrimSpace(innerMatch[1]) != "" {
					textParts = append(textParts, strings.TrimSpace(innerMatch[1]))
				}
			}
		}
	}

	return strings.Join(textParts, " ")
}

func (cp *ChunkProcessor) extractFormFieldType(content string) string {
	ftRegex := regexp.MustCompile(`/FT\s*/(\w+)`)
	matches := ftRegex.FindStringSubmatch(content)
	if len(matches) > 1 {
		return matches[1]
	}
	return "Unknown"
}

func (cp *ChunkProcessor) extractFormFieldName(content string) string {
	nameRegex := regexp.MustCompile(`/T\s*\((.*?)\)`)
	matches := nameRegex.FindStringSubmatch(content)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

func (cp *ChunkProcessor) extractFormFieldValue(content string) string {
	valueRegex := regexp.MustCompile(`/V\s*\((.*?)\)`)
	matches := valueRegex.FindStringSubmatch(content)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

func (cp *ChunkProcessor) parseXRefEntries(content string) map[int64]XRefEntry {
	entries := make(map[int64]XRefEntry)

	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if len(line) == 18 { // XRef entries are typically 18 characters
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				offset := parseInt64(parts[0])
				generation := parseInt(parts[1])
				flag := parts[2]

				entries[offset] = XRefEntry{
					Offset:     offset,
					Generation: generation,
					InUse:      flag == "n",
				}
			}
		}
	}

	return entries
}

// Utility functions

func parseInt(s string) int {
	// Simple integer parsing - can be enhanced with error handling
	var result int
	fmt.Sscanf(s, "%d", &result)
	return result
}

func parseInt64(s string) int64 {
	var result int64
	fmt.Sscanf(s, "%d", &result)
	return result
}

func parseFloat(s string) float64 {
	var result float64
	fmt.Sscanf(s, "%f", &result)
	return result
}
