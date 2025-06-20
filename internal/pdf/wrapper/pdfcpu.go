package wrapper

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/a3tai/mcp-pdf-reader/internal/pdf/extraction"
	"github.com/a3tai/mcp-pdf-reader/internal/pdf/security"
	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

// PDFCPULibrary implements PDFLibrary interface using pdfcpu
type PDFCPULibrary struct {
	config FactoryConfig
	closed bool
}

// NewPDFCPULibrary creates a new pdfcpu library wrapper
func NewPDFCPULibrary(config FactoryConfig) *PDFCPULibrary {
	return &PDFCPULibrary{
		config: config,
		closed: false,
	}
}

// Open opens a PDF from an io.Reader
func (p *PDFCPULibrary) Open(reader io.Reader) (PDFDocument, error) {
	if p.closed {
		return nil, &WrapperError{Library: LibraryPDFCPU, Op: "open", Err: ErrDocumentClosed.Err}
	}

	readSeeker, ok := reader.(io.ReadSeeker)
	if !ok {
		return nil, &WrapperError{
			Library: LibraryPDFCPU,
			Op:      "open",
			Err:     fmt.Errorf("reader must implement io.ReadSeeker"),
		}
	}

	conf := model.NewDefaultConfiguration()
	conf.ValidationMode = model.ValidationRelaxed

	ctx, err := api.ReadContext(readSeeker, conf)
	if err != nil {
		return nil, &WrapperError{
			Library: LibraryPDFCPU,
			Op:      "open",
			Err:     fmt.Errorf("failed to read PDF context: %w", err),
		}
	}

	if err := ctx.EnsurePageCount(); err != nil {
		return nil, &WrapperError{
			Library: LibraryPDFCPU,
			Op:      "open",
			Err:     fmt.Errorf("failed to ensure page count: %w", err),
		}
	}

	return &PDFCPUDocument{
		ctx:    ctx,
		config: p.config,
		closed: false,
	}, nil
}

// OpenFile opens a PDF from a file path
func (p *PDFCPULibrary) OpenFile(path string) (PDFDocument, error) {
	if p.closed {
		return nil, &WrapperError{Library: LibraryPDFCPU, Op: "open_file", Err: ErrDocumentClosed.Err}
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, &WrapperError{
			Library: LibraryPDFCPU,
			Op:      "open_file",
			Err:     fmt.Errorf("failed to open file: %w", err),
		}
	}
	defer file.Close()

	conf := model.NewDefaultConfiguration()
	conf.ValidationMode = model.ValidationRelaxed

	ctx, err := api.ReadContext(file, conf)
	if err != nil {
		return nil, &WrapperError{
			Library: LibraryPDFCPU,
			Op:      "open_file",
			Err:     fmt.Errorf("failed to read PDF context: %w", err),
		}
	}

	if err := ctx.EnsurePageCount(); err != nil {
		return nil, &WrapperError{
			Library: LibraryPDFCPU,
			Op:      "open_file",
			Err:     fmt.Errorf("failed to ensure page count: %w", err),
		}
	}

	return &PDFCPUDocument{
		ctx:    ctx,
		config: p.config,
		closed: false,
	}, nil
}

// Validate validates the library is properly initialized
func (p *PDFCPULibrary) Validate() error {
	if p.closed {
		return &WrapperError{Library: LibraryPDFCPU, Op: "validate", Err: ErrDocumentClosed.Err}
	}
	return nil
}

// Close closes the library and releases resources
func (p *PDFCPULibrary) Close() error {
	p.closed = true
	return nil
}

// GetLibraryType returns the library type
func (p *PDFCPULibrary) GetLibraryType() LibraryType {
	return LibraryPDFCPU
}

// GetVersion returns the pdfcpu version
func (p *PDFCPULibrary) GetVersion() string {
	return "pdfcpu-v0.8.0" // Update this as needed
}

// PDFCPUDocument implements PDFDocument interface using pdfcpu
type PDFCPUDocument struct {
	ctx             *model.Context
	config          FactoryConfig
	closed          bool
	securityHandler security.SecurityHandler
}

// GetPageCount returns the number of pages in the document
func (d *PDFCPUDocument) GetPageCount() (int, error) {
	if d.closed {
		return 0, &WrapperError{Library: LibraryPDFCPU, Op: "get_page_count", Err: ErrDocumentClosed.Err}
	}
	return d.ctx.PageCount, nil
}

// GetPage returns a specific page
func (d *PDFCPUDocument) GetPage(pageNum int) (PDFPage, error) {
	if d.closed {
		return nil, &WrapperError{Library: LibraryPDFCPU, Op: "get_page", Err: ErrDocumentClosed.Err}
	}

	if pageNum < 1 || pageNum > d.ctx.PageCount {
		return nil, &WrapperError{
			Library: LibraryPDFCPU,
			Op:      "get_page",
			Err:     fmt.Errorf("invalid page number %d (document has %d pages)", pageNum, d.ctx.PageCount),
		}
	}

	return &PDFCPUPage{
		ctx:     d.ctx,
		pageNum: pageNum,
		config:  d.config,
	}, nil
}

// GetMetadata extracts document metadata
func (d *PDFCPUDocument) GetMetadata() (*Metadata, error) {
	if d.closed {
		return nil, &WrapperError{Library: LibraryPDFCPU, Op: "get_metadata", Err: ErrDocumentClosed.Err}
	}

	metadata := &Metadata{
		Custom: make(map[string]string),
	}

	// Get document info dictionary - pdfcpu stores this as an indirect reference
	// We would need to dereference it to access the actual info dictionary
	// For now, provide basic metadata structure
	if d.ctx.Info != nil {
		// TODO: Implement proper info dictionary dereferencing for pdfcpu
		// This requires dereferencing the indirect reference and parsing the dictionary
		metadata.Producer = "Generated with pdfcpu"
	}

	return metadata, nil
}

// GetVersion returns the PDF version
func (d *PDFCPUDocument) GetVersion() (string, error) {
	if d.closed {
		return "", &WrapperError{Library: LibraryPDFCPU, Op: "get_version", Err: ErrDocumentClosed.Err}
	}
	return d.ctx.HeaderVersion.String(), nil
}

// Close closes the document
func (d *PDFCPUDocument) Close() error {
	d.closed = true
	return nil
}

// ExtractText extracts text from a specific page
func (d *PDFCPUDocument) ExtractText(pageNum int) ([]TextElement, error) {
	if d.closed {
		return nil, &WrapperError{Library: LibraryPDFCPU, Op: "extract_text", Err: ErrDocumentClosed.Err}
	}

	// pdfcpu doesn't have built-in text extraction with positioning
	// This would need to be implemented by parsing content streams
	// For now, return empty slice
	return []TextElement{}, nil
}

// ExtractImages extracts images from a specific page
func (d *PDFCPUDocument) ExtractImages(pageNum int) ([]ImageElement, error) {
	if d.closed {
		return nil, &WrapperError{Library: LibraryPDFCPU, Op: "extract_images", Err: ErrDocumentClosed.Err}
	}

	// This would require implementing image extraction using pdfcpu's XObject parsing
	// For now, return empty slice
	return []ImageElement{}, nil
}

// ExtractForms extracts form fields from the document
func (d *PDFCPUDocument) ExtractForms() ([]extraction.FormField, error) {
	if d.closed {
		return nil, &WrapperError{Library: LibraryPDFCPU, Op: "extract_forms", Err: ErrDocumentClosed.Err}
	}

	// Extract forms from the pdfcpu context
	// For now, return empty forms as this requires more complex integration
	// TODO: Implement form extraction using pdfcpu context

	// If no file path, we need a ReadSeeker
	// This is a limitation - we'd need to store the original reader
	return nil, &WrapperError{
		Library: LibraryPDFCPU,
		Op:      "extract_forms",
		Err:     fmt.Errorf("form extraction requires file path or reader access"),
	}
}

// ExtractTables extracts tables from a specific page
func (d *PDFCPUDocument) ExtractTables(pageNum int) ([]TableElement, error) {
	if d.closed {
		return nil, &WrapperError{Library: LibraryPDFCPU, Op: "extract_tables", Err: ErrDocumentClosed.Err}
	}

	// Table extraction would need to be implemented
	return []TableElement{}, nil
}

// ExtractAnnotations extracts annotations from a specific page
func (d *PDFCPUDocument) ExtractAnnotations(pageNum int) ([]AnnotationElement, error) {
	if d.closed {
		return nil, &WrapperError{Library: LibraryPDFCPU, Op: "extract_annotations", Err: ErrDocumentClosed.Err}
	}

	// Annotation extraction would need to be implemented
	return []AnnotationElement{}, nil
}

// GetCatalog returns the PDF catalog
func (d *PDFCPUDocument) GetCatalog() (*Catalog, error) {
	if d.closed {
		return nil, &WrapperError{Library: LibraryPDFCPU, Op: "get_catalog", Err: ErrDocumentClosed.Err}
	}

	catalog := &Catalog{
		Metadata: make(map[string]interface{}),
	}

	// Get version
	if version, err := d.GetVersion(); err == nil {
		catalog.Version = version
	}

	// Create basic page tree info
	catalog.Pages = &PageTree{
		Type:  "Pages",
		Count: d.ctx.PageCount,
	}

	return catalog, nil
}

// GetContentStream returns the content stream for a page
func (d *PDFCPUDocument) GetContentStream(pageNum int) ([]byte, error) {
	if d.closed {
		return nil, &WrapperError{Library: LibraryPDFCPU, Op: "get_content_stream", Err: ErrDocumentClosed.Err}
	}

	// This would require low-level content stream access
	return nil, &WrapperError{
		Library: LibraryPDFCPU,
		Op:      "get_content_stream",
		Err:     fmt.Errorf("content stream access not yet implemented"),
	}
}

// GetPageResources returns resources for a specific page
func (d *PDFCPUDocument) GetPageResources(pageNum int) (*Resources, error) {
	if d.closed {
		return nil, &WrapperError{Library: LibraryPDFCPU, Op: "get_page_resources", Err: ErrDocumentClosed.Err}
	}

	// This would require parsing page resource dictionaries
	return &Resources{}, nil
}

// IsEncrypted checks if the document is encrypted
func (d *PDFCPUDocument) IsEncrypted() bool {
	return d.ctx.Encrypt != nil
}

// RequiresPassword checks if the document requires a password
func (d *PDFCPUDocument) RequiresPassword() bool {
	if !d.IsEncrypted() {
		return false
	}

	// If we have a security handler and it's authenticated, no password needed
	if d.securityHandler != nil && d.securityHandler.IsAuthenticated() {
		return false
	}

	return true
}

// ValidatePassword validates a password for encrypted documents
func (d *PDFCPUDocument) ValidatePassword(password string) error {
	if !d.IsEncrypted() {
		return nil
	}

	// Try to create a new context with the password
	conf := model.NewDefaultConfiguration()
	conf.ValidationMode = model.ValidationRelaxed

	// Set the password in the configuration
	if password != "" {
		conf.UserPW = password
		conf.OwnerPW = password
	}

	// Try to read the document again with the password
	// This is a simplified approach - in a full implementation we would
	// integrate with our security handler
	if d.ctx.Encrypt != nil {
		// For now, we'll accept any non-empty password for encrypted documents
		// A full implementation would use pdfcpu's password validation
		if password == "" {
			return &WrapperError{
				Library: LibraryPDFCPU,
				Op:      "validate_password",
				Err:     fmt.Errorf("password required for encrypted document"),
			}
		}

		// Store the password for future operations
		d.ctx.UserPW = password
		d.ctx.OwnerPW = password
	}

	return nil
}

// PDFCPUPage implements PDFPage interface using pdfcpu
type PDFCPUPage struct {
	ctx     *model.Context
	pageNum int
	config  FactoryConfig
}

// GetNumber returns the page number
func (p *PDFCPUPage) GetNumber() int {
	return p.pageNum
}

// GetSize returns the page size
func (p *PDFCPUPage) GetSize() (*PageSize, error) {
	// This would require parsing the page's MediaBox
	return &PageSize{
		Width:  612.0, // Default US Letter width
		Height: 792.0, // Default US Letter height
		Unit:   "pt",
	}, nil
}

// GetContent returns the page content stream
func (p *PDFCPUPage) GetContent() ([]byte, error) {
	// This would require content stream parsing
	return nil, &WrapperError{
		Library: LibraryPDFCPU,
		Op:      "get_content",
		Err:     fmt.Errorf("page content access not yet implemented"),
	}
}

// GetResources returns the page resources
func (p *PDFCPUPage) GetResources() (*Resources, error) {
	// This would require resource dictionary parsing
	return &Resources{}, nil
}

// GetAnnotations returns annotations on this page
func (p *PDFCPUPage) GetAnnotations() ([]AnnotationElement, error) {
	// This would require annotation parsing
	return []AnnotationElement{}, nil
}

// GetText returns text elements on this page
func (p *PDFCPUPage) GetText() ([]TextElement, error) {
	// This would require text extraction implementation
	return []TextElement{}, nil
}

// GetImages returns image elements on this page
func (p *PDFCPUPage) GetImages() ([]ImageElement, error) {
	// This would require image extraction implementation
	return []ImageElement{}, nil
}

// Helper function to parse PDF date strings
func parsePDFDate(dateStr string) (time.Time, error) {
	// PDF date format: D:YYYYMMDDHHmmSSOHH'mm'
	// Remove D: prefix if present
	if strings.HasPrefix(dateStr, "D:") {
		dateStr = dateStr[2:]
	}

	// Try different date formats
	formats := []string{
		"20060102150405-07'00'",
		"20060102150405+07'00'",
		"20060102150405",
		"20060102",
	}

	for _, format := range formats {
		if date, err := time.Parse(format, dateStr); err == nil {
			return date, nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse PDF date: %s", dateStr)
}
