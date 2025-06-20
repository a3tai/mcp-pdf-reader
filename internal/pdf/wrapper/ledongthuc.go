package wrapper

import (
	"fmt"
	"io"
	"os"

	"github.com/a3tai/mcp-pdf-reader/internal/pdf/extraction"
	"github.com/ledongthuc/pdf"
)

// LedongthucLibrary implements PDFLibrary interface using ledongthuc/pdf
type LedongthucLibrary struct {
	config FactoryConfig
	closed bool
}

// NewLedongthucLibrary creates a new ledongthuc library wrapper
func NewLedongthucLibrary(config FactoryConfig) *LedongthucLibrary {
	return &LedongthucLibrary{
		config: config,
		closed: false,
	}
}

// Open opens a PDF from an io.Reader
func (l *LedongthucLibrary) Open(reader io.Reader) (PDFDocument, error) {
	if l.closed {
		return nil, &WrapperError{Library: LibraryLedongthuc, Op: "open", Err: ErrDocumentClosed.Err}
	}

	// ledongthuc/pdf doesn't support reading from io.Reader directly
	// This is a limitation of the library
	return nil, &WrapperError{
		Library: LibraryLedongthuc,
		Op:      "open",
		Err:     fmt.Errorf("ledongthuc/pdf does not support io.Reader, use OpenFile instead"),
	}
}

// OpenFile opens a PDF from a file path
func (l *LedongthucLibrary) OpenFile(path string) (PDFDocument, error) {
	if l.closed {
		return nil, &WrapperError{Library: LibraryLedongthuc, Op: "open_file", Err: ErrDocumentClosed.Err}
	}

	f, pdfReader, err := pdf.Open(path)
	if err != nil {
		return nil, &WrapperError{
			Library: LibraryLedongthuc,
			Op:      "open_file",
			Err:     fmt.Errorf("failed to open PDF: %w", err),
		}
	}

	return &LedongthucDocument{
		reader:   pdfReader,
		config:   l.config,
		closed:   false,
		filePath: path,
		file:     f,
	}, nil
}

// Validate validates the library is properly initialized
func (l *LedongthucLibrary) Validate() error {
	if l.closed {
		return &WrapperError{Library: LibraryLedongthuc, Op: "validate", Err: ErrDocumentClosed.Err}
	}
	return nil
}

// Close closes the library and releases resources
func (l *LedongthucLibrary) Close() error {
	l.closed = true
	return nil
}

// GetLibraryType returns the library type
func (l *LedongthucLibrary) GetLibraryType() LibraryType {
	return LibraryLedongthuc
}

// GetVersion returns the ledongthuc/pdf version
func (l *LedongthucLibrary) GetVersion() string {
	return "ledongthuc/pdf-v1.0" // Update this as needed
}

// LedongthucDocument implements PDFDocument interface using ledongthuc/pdf
type LedongthucDocument struct {
	reader   *pdf.Reader
	config   FactoryConfig
	closed   bool
	filePath string
	file     *os.File
}

// GetPageCount returns the number of pages in the document
func (d *LedongthucDocument) GetPageCount() (int, error) {
	if d.closed {
		return 0, &WrapperError{Library: LibraryLedongthuc, Op: "get_page_count", Err: ErrDocumentClosed.Err}
	}
	return d.reader.NumPage(), nil
}

// GetPage returns a specific page
func (d *LedongthucDocument) GetPage(pageNum int) (PDFPage, error) {
	if d.closed {
		return nil, &WrapperError{Library: LibraryLedongthuc, Op: "get_page", Err: ErrDocumentClosed.Err}
	}

	if pageNum < 1 || pageNum > d.reader.NumPage() {
		return nil, &WrapperError{
			Library: LibraryLedongthuc,
			Op:      "get_page",
			Err:     fmt.Errorf("invalid page number %d (document has %d pages)", pageNum, d.reader.NumPage()),
		}
	}

	page := d.reader.Page(pageNum)
	return &LedongthucPage{
		page:    page,
		pageNum: pageNum,
		config:  d.config,
	}, nil
}

// GetMetadata extracts document metadata
func (d *LedongthucDocument) GetMetadata() (*Metadata, error) {
	if d.closed {
		return nil, &WrapperError{Library: LibraryLedongthuc, Op: "get_metadata", Err: ErrDocumentClosed.Err}
	}

	metadata := &Metadata{
		Custom: make(map[string]string),
	}

	// ledongthuc/pdf has limited metadata support
	// We can try to extract basic information if available

	// Note: ledongthuc/pdf doesn't expose metadata directly
	// This would need to be implemented by accessing the underlying PDF structure
	// For now, return basic metadata with what we can determine

	return metadata, nil
}

// GetVersion returns the PDF version
func (d *LedongthucDocument) GetVersion() (string, error) {
	if d.closed {
		return "", &WrapperError{Library: LibraryLedongthuc, Op: "get_version", Err: ErrDocumentClosed.Err}
	}

	// ledongthuc/pdf doesn't expose version directly
	// Return a default version
	return "1.4", nil
}

// Close closes the document
func (d *LedongthucDocument) Close() error {
	d.closed = true
	if d.file != nil {
		return d.file.Close()
	}
	return nil
}

// ExtractText extracts text from a specific page
func (d *LedongthucDocument) ExtractText(pageNum int) ([]TextElement, error) {
	if d.closed {
		return nil, &WrapperError{Library: LibraryLedongthuc, Op: "extract_text", Err: ErrDocumentClosed.Err}
	}

	if pageNum < 1 || pageNum > d.reader.NumPage() {
		return nil, &WrapperError{
			Library: LibraryLedongthuc,
			Op:      "extract_text",
			Err:     fmt.Errorf("invalid page number %d", pageNum),
		}
	}

	page := d.reader.Page(pageNum)
	content := page.Content()

	var textElements []TextElement

	// Convert ledongthuc text content to our TextElement format
	for _, text := range content.Text {
		// Use FontSize as height approximation since ledongthuc doesn't provide text height
		height := text.FontSize
		if height == 0 {
			height = 12.0 // Default height
		}

		element := TextElement{
			Text: text.S,
			Position: Rectangle{
				LowerLeft:  Point{X: text.X, Y: text.Y},
				UpperRight: Point{X: text.X + text.W, Y: text.Y + height},
				Width:      text.W,
				Height:     height,
			},
			Font: FontInfo{
				Name: text.Font,
				Size: text.FontSize,
			},
		}
		textElements = append(textElements, element)
	}

	return textElements, nil
}

// ExtractImages extracts images from a specific page
func (d *LedongthucDocument) ExtractImages(pageNum int) ([]ImageElement, error) {
	if d.closed {
		return nil, &WrapperError{Library: LibraryLedongthuc, Op: "extract_images", Err: ErrDocumentClosed.Err}
	}

	// ledongthuc/pdf has very limited image extraction capabilities
	// Return empty slice for now
	return []ImageElement{}, nil
}

// ExtractForms extracts form fields from the document
func (d *LedongthucDocument) ExtractForms() ([]extraction.FormField, error) {
	if d.closed {
		return nil, &WrapperError{Library: LibraryLedongthuc, Op: "extract_forms", Err: ErrDocumentClosed.Err}
	}

	// ledongthuc/pdf doesn't have form extraction capabilities
	// Use the basic form extractor that looks for patterns
	formExtractor := extraction.NewFormExtractor(d.config.DebugMode)
	return formExtractor.ExtractForms(d.reader)
}

// ExtractTables extracts tables from a specific page
func (d *LedongthucDocument) ExtractTables(pageNum int) ([]TableElement, error) {
	if d.closed {
		return nil, &WrapperError{Library: LibraryLedongthuc, Op: "extract_tables", Err: ErrDocumentClosed.Err}
	}

	// ledongthuc/pdf doesn't have table extraction
	return []TableElement{}, nil
}

// ExtractAnnotations extracts annotations from a specific page
func (d *LedongthucDocument) ExtractAnnotations(pageNum int) ([]AnnotationElement, error) {
	if d.closed {
		return nil, &WrapperError{Library: LibraryLedongthuc, Op: "extract_annotations", Err: ErrDocumentClosed.Err}
	}

	// ledongthuc/pdf doesn't have annotation extraction
	return []AnnotationElement{}, nil
}

// GetCatalog returns the PDF catalog
func (d *LedongthucDocument) GetCatalog() (*Catalog, error) {
	if d.closed {
		return nil, &WrapperError{Library: LibraryLedongthuc, Op: "get_catalog", Err: ErrDocumentClosed.Err}
	}

	catalog := &Catalog{
		Metadata: make(map[string]interface{}),
	}

	// Create basic page tree info
	catalog.Pages = &PageTree{
		Type:  "Pages",
		Count: d.reader.NumPage(),
	}

	return catalog, nil
}

// GetContentStream returns the content stream for a page
func (d *LedongthucDocument) GetContentStream(pageNum int) ([]byte, error) {
	if d.closed {
		return nil, &WrapperError{Library: LibraryLedongthuc, Op: "get_content_stream", Err: ErrDocumentClosed.Err}
	}

	// ledongthuc/pdf doesn't expose raw content streams
	return nil, &WrapperError{
		Library: LibraryLedongthuc,
		Op:      "get_content_stream",
		Err:     fmt.Errorf("content stream access not supported by ledongthuc/pdf"),
	}
}

// GetPageResources returns resources for a specific page
func (d *LedongthucDocument) GetPageResources(pageNum int) (*Resources, error) {
	if d.closed {
		return nil, &WrapperError{Library: LibraryLedongthuc, Op: "get_page_resources", Err: ErrDocumentClosed.Err}
	}

	// ledongthuc/pdf doesn't expose page resources directly
	return &Resources{}, nil
}

// IsEncrypted checks if the document is encrypted
func (d *LedongthucDocument) IsEncrypted() bool {
	// ledongthuc/pdf doesn't support encrypted documents well
	return false
}

// RequiresPassword checks if the document requires a password
func (d *LedongthucDocument) RequiresPassword() bool {
	return false
}

// ValidatePassword validates a password for encrypted documents
func (d *LedongthucDocument) ValidatePassword(password string) error {
	// ledongthuc/pdf doesn't support password-protected documents
	return &WrapperError{
		Library: LibraryLedongthuc,
		Op:      "validate_password",
		Err:     fmt.Errorf("password-protected PDFs not supported by ledongthuc/pdf"),
	}
}

// LedongthucPage implements PDFPage interface using ledongthuc/pdf
type LedongthucPage struct {
	page    pdf.Page
	pageNum int
	config  FactoryConfig
}

// GetNumber returns the page number
func (p *LedongthucPage) GetNumber() int {
	return p.pageNum
}

// GetSize returns the page size
func (p *LedongthucPage) GetSize() (*PageSize, error) {
	// Get page size from the page object
	// ledongthuc/pdf doesn't directly expose page dimensions
	// We'll need to calculate from content or use defaults
	return &PageSize{
		Width:  612.0, // Default US Letter width
		Height: 792.0, // Default US Letter height
		Unit:   "pt",
	}, nil
}

// GetContent returns the page content stream
func (p *LedongthucPage) GetContent() ([]byte, error) {
	// ledongthuc/pdf doesn't expose raw content streams
	return nil, &WrapperError{
		Library: LibraryLedongthuc,
		Op:      "get_content",
		Err:     fmt.Errorf("raw content access not supported by ledongthuc/pdf"),
	}
}

// GetResources returns the page resources
func (p *LedongthucPage) GetResources() (*Resources, error) {
	// ledongthuc/pdf doesn't expose resource dictionaries
	return &Resources{}, nil
}

// GetAnnotations returns annotations on this page
func (p *LedongthucPage) GetAnnotations() ([]AnnotationElement, error) {
	// ledongthuc/pdf doesn't support annotation extraction
	return []AnnotationElement{}, nil
}

// GetText returns text elements on this page
func (p *LedongthucPage) GetText() ([]TextElement, error) {
	content := p.page.Content()
	var textElements []TextElement

	// Convert ledongthuc text content to our TextElement format
	for _, text := range content.Text {
		// Use FontSize as height approximation since ledongthuc doesn't provide text height
		height := text.FontSize
		if height == 0 {
			height = 12.0 // Default height
		}

		element := TextElement{
			Text: text.S,
			Position: Rectangle{
				LowerLeft:  Point{X: text.X, Y: text.Y},
				UpperRight: Point{X: text.X + text.W, Y: text.Y + height},
				Width:      text.W,
				Height:     height,
			},
			Font: FontInfo{
				Name: text.Font,
				Size: text.FontSize,
			},
		}
		textElements = append(textElements, element)
	}

	return textElements, nil
}

// GetImages returns image elements on this page
func (p *LedongthucPage) GetImages() ([]ImageElement, error) {
	// ledongthuc/pdf has very limited image support
	return []ImageElement{}, nil
}
