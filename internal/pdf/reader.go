package pdf

import (
	"fmt"
	"os"
	"strings"

	"github.com/ledongthuc/pdf"
)

// Reader handles PDF file reading operations
type Reader struct {
	maxFileSize int64
	maxTextSize int
}

// NewReader creates a new PDF reader with the specified constraints
func NewReader(maxFileSize int64) *Reader {
	return &Reader{
		maxFileSize: maxFileSize,
		maxTextSize: 10 * 1024 * 1024, // 10MB text limit
	}
}

// ReadFile extracts text content from a PDF file
func (r *Reader) ReadFile(req PDFReadFileRequest) (*PDFReadFileResult, error) {
	if req.Path == "" {
		return nil, fmt.Errorf("path cannot be empty")
	}

	// Check if file exists and get basic info
	fileInfo, err := os.Stat(req.Path)
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("file does not exist: %s", req.Path)
	}
	if err != nil {
		return nil, fmt.Errorf("cannot access file: %w", err)
	}

	// Validate file type
	if err := r.validatePDFFile(req.Path, fileInfo); err != nil {
		return nil, err
	}

	// Open and parse PDF
	f, pdfReader, err := pdf.Open(req.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to open PDF: %w", err)
	}
	defer f.Close()

	// Extract text content
	content, err := r.extractTextContent(pdfReader)
	if err != nil {
		return nil, fmt.Errorf("failed to extract text content: %w", err)
	}

	// Analyze content type and detect images
	contentType := r.analyzeContentType(content, pdfReader)
	hasImages, imageCount := r.detectImages(pdfReader)

	result := &PDFReadFileResult{
		Content:     content,
		Path:        req.Path,
		Pages:       pdfReader.NumPage(),
		Size:        fileInfo.Size(),
		ContentType: contentType,
		HasImages:   hasImages,
		ImageCount:  imageCount,
	}

	return result, nil
}

// validatePDFFile performs basic validation on a PDF file
func (r *Reader) validatePDFFile(filePath string, fileInfo os.FileInfo) error {
	// Check if it's a regular file (not a directory)
	if fileInfo.IsDir() {
		return fmt.Errorf("path is a directory, not a file: %s", filePath)
	}

	// Check file extension
	if !strings.HasSuffix(strings.ToLower(filePath), ".pdf") {
		return fmt.Errorf("file is not a PDF: %s", filePath)
	}

	// Check file size
	if fileInfo.Size() > r.maxFileSize {
		return fmt.Errorf("file too large: %d bytes (max: %d bytes)",
			fileInfo.Size(), r.maxFileSize)
	}

	return nil
}

// extractTextContent extracts text content from a PDF reader
func (r *Reader) extractTextContent(pdfReader *pdf.Reader) (string, error) {
	var builder strings.Builder
	totalLength := 0

	for pageNum := 1; pageNum <= pdfReader.NumPage(); pageNum++ {
		page := pdfReader.Page(pageNum)
		if page.V.IsNull() {
			continue
		}

		content, err := page.GetPlainText(nil)
		if err != nil {
			// Continue with other pages even if one fails
			continue
		}

		// Check if adding this content would exceed the limit
		if totalLength+len(content) > r.maxTextSize {
			remaining := r.maxTextSize - totalLength
			if remaining > 0 {
				builder.WriteString(content[:remaining])
			}
			break
		}

		builder.WriteString(content)
		totalLength += len(content)

		// Add page separator for readability
		if pageNum < pdfReader.NumPage() {
			builder.WriteString("\n\n--- Page Break ---\n\n")
		}
	}

	text := builder.String()
	if text == "" {
		return "", fmt.Errorf("no text content could be extracted from PDF")
	}

	return text, nil
}

// analyzeContentType determines the type of content in the PDF
func (r *Reader) analyzeContentType(textContent string, pdfReader *pdf.Reader) string {
	// Minimum text length to consider content meaningful
	const minMeaningfulTextLength = 50

	// Check if we extracted meaningful text
	cleanText := strings.TrimSpace(textContent)

	// Remove page breaks to get actual content
	textWithoutBreaks := strings.ReplaceAll(cleanText, "--- Page Break ---", "")
	textWithoutBreaks = strings.TrimSpace(textWithoutBreaks)

	// Check for images
	hasImages, _ := r.detectImages(pdfReader)

	// Determine content type based on text and images
	if textWithoutBreaks == "" {
		if hasImages {
			return "scanned_images"
		}
		return "no_content"
	}

	// Consider it mostly text if we have substantial text content
	// Rough heuristic: if text is less than minimum threshold, it might be mostly images
	if len(textWithoutBreaks) < minMeaningfulTextLength {
		if hasImages {
			return "scanned_images"
		}
		return "no_content"
	}

	// If we have both meaningful text and images, it's mixed
	if hasImages {
		return "mixed"
	}

	return "text"
}

// detectImages scans the PDF for image objects
func (r *Reader) detectImages(pdfReader *pdf.Reader) (bool, int) {
	imageCount := 0

	for pageNum := 1; pageNum <= pdfReader.NumPage(); pageNum++ {
		pageImages := r.countImagesOnPage(pdfReader, pageNum)
		imageCount += pageImages
	}

	return imageCount > 0, imageCount
}

// countImagesOnPage counts images on a specific page
func (r *Reader) countImagesOnPage(pdfReader *pdf.Reader, pageNum int) int {
	defer func() {
		// Recover from any panics during image detection
		if recover() != nil {
			// Image detection failed for this page
		}
	}()

	page := pdfReader.Page(pageNum)
	if page.V.IsNull() {
		return 0
	}

	// Get page resources
	resources := page.V.Key("Resources")
	if resources.IsNull() {
		return 0
	}

	// Get XObject dictionary (where images are typically stored)
	xObjects := resources.Key("XObject")
	if xObjects.IsNull() || xObjects.Kind() != pdf.Dict {
		return 0
	}

	imageCount := 0
	// Iterate through XObjects looking for images
	for _, key := range xObjects.Keys() {
		obj := xObjects.Key(key)
		if obj.IsNull() {
			continue
		}

		// Check if this XObject is an image
		subtype := obj.Key("Subtype")
		if subtype.IsNull() || subtype.Name() != "Image" {
			continue
		}

		imageCount++
	}

	return imageCount
}
