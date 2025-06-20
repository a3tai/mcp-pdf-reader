package wrapper

import (
	"bytes"
	"fmt"
	"io"
	"os"

	"github.com/a3tai/mcp-pdf-reader/internal/pdf/custom"
	"github.com/a3tai/mcp-pdf-reader/internal/pdf/extraction"
	"github.com/a3tai/mcp-pdf-reader/internal/pdf/security"
)

// CustomPDFLibrary implements PDFLibrary interface using custom PDF parsing
type CustomPDFLibrary struct {
	config FactoryConfig
	closed bool
}

// NewCustomPDFLibrary creates a new custom library wrapper
func NewCustomPDFLibrary(config FactoryConfig) *CustomPDFLibrary {
	return &CustomPDFLibrary{
		config: config,
		closed: false,
	}
}

// Open opens a PDF from an io.Reader
func (c *CustomPDFLibrary) Open(reader io.Reader) (PDFDocument, error) {
	if c.closed {
		return nil, &WrapperError{Library: LibraryCustom, Op: "open", Err: ErrDocumentClosed.Err}
	}

	// Convert reader to ReadSeeker if needed
	var seeker io.ReadSeeker
	if rs, ok := reader.(io.ReadSeeker); ok {
		seeker = rs
	} else {
		// Buffer the entire content
		data, err := io.ReadAll(reader)
		if err != nil {
			return nil, &WrapperError{
				Library: LibraryCustom,
				Op:      "open",
				Err:     fmt.Errorf("failed to read data: %w", err),
			}
		}
		seeker = bytes.NewReader(data)
	}

	// Create and parse the PDF
	parser := custom.NewCustomPDFParser(seeker)
	if err := parser.Parse(); err != nil {
		return nil, &WrapperError{
			Library: LibraryCustom,
			Op:      "open",
			Err:     fmt.Errorf("failed to parse PDF: %w", err),
		}
	}

	doc := &CustomPDFDocument{
		parser: parser,
		config: c.config,
		closed: false,
	}

	// Initialize security handler if document is encrypted
	if doc.IsEncrypted() {
		if err := doc.initializeSecurityHandler(); err != nil {
			// Log the error but don't fail document opening
			// Security operations will fail later if needed
		}
	}

	return doc, nil
}

// OpenFile opens a PDF from a file path
func (c *CustomPDFLibrary) OpenFile(path string) (PDFDocument, error) {
	if c.closed {
		return nil, &WrapperError{Library: LibraryCustom, Op: "open_file", Err: ErrDocumentClosed.Err}
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, &WrapperError{
			Library: LibraryCustom,
			Op:      "open_file",
			Err:     fmt.Errorf("failed to open file: %w", err),
		}
	}

	// Don't close the file yet, we need it for the parser
	parser := custom.NewCustomPDFParser(file)
	if err := parser.Parse(); err != nil {
		file.Close()
		return nil, &WrapperError{
			Library: LibraryCustom,
			Op:      "open_file",
			Err:     fmt.Errorf("failed to parse PDF: %w", err),
		}
	}

	return &CustomPDFDocument{
		config:   c.config,
		closed:   false,
		filePath: path,
		parser:   parser,
		file:     file,
	}, nil
}

// Validate validates the library is properly initialized
func (c *CustomPDFLibrary) Validate() error {
	if c.closed {
		return &WrapperError{Library: LibraryCustom, Op: "validate", Err: ErrDocumentClosed.Err}
	}
	return nil
}

// Close closes the library and releases resources
func (c *CustomPDFLibrary) Close() error {
	c.closed = true
	return nil
}

// GetLibraryType returns the library type
func (c *CustomPDFLibrary) GetLibraryType() LibraryType {
	return LibraryCustom
}

// GetVersion returns the custom library version
func (c *CustomPDFLibrary) GetVersion() string {
	return "custom-v1.0.0"
}

// CustomPDFDocument implements PDFDocument interface using custom PDF parsing
type CustomPDFDocument struct {
	config          FactoryConfig
	closed          bool
	filePath        string
	parser          *custom.CustomPDFParser
	file            *os.File // For file-based documents
	securityHandler security.SecurityHandler
}

// GetPageCount returns the number of pages in the document
func (d *CustomPDFDocument) GetPageCount() (int, error) {
	if d.closed {
		return 0, &WrapperError{Library: LibraryCustom, Op: "get_page_count", Err: ErrDocumentClosed.Err}
	}

	catalog := d.parser.GetCatalog()
	if catalog == nil {
		return 0, &WrapperError{
			Library: LibraryCustom,
			Op:      "get_page_count",
			Err:     fmt.Errorf("no catalog found"),
		}
	}

	pagesObj := catalog.Get("Pages")
	if pagesObj.Type() != custom.TypeIndirectRef {
		return 0, &WrapperError{
			Library: LibraryCustom,
			Op:      "get_page_count",
			Err:     fmt.Errorf("invalid pages reference"),
		}
	}

	pagesDict, err := d.parser.ResolveIndirectObject(pagesObj)
	if err != nil {
		return 0, &WrapperError{
			Library: LibraryCustom,
			Op:      "get_page_count",
			Err:     fmt.Errorf("failed to resolve pages: %w", err),
		}
	}

	if pagesDict.Type() != custom.TypeDictionary {
		return 0, &WrapperError{
			Library: LibraryCustom,
			Op:      "get_page_count",
			Err:     fmt.Errorf("pages is not a dictionary"),
		}
	}

	count := pagesDict.(*custom.Dictionary).GetInt("Count")
	return int(count), nil
}

// GetPage returns a specific page
func (d *CustomPDFDocument) GetPage(pageNum int) (PDFPage, error) {
	if d.closed {
		return nil, &WrapperError{Library: LibraryCustom, Op: "get_page", Err: ErrDocumentClosed.Err}
	}

	// TODO: Implement page validation and creation
	return &CustomPDFPage{
		pageNum: pageNum,
		config:  d.config,
	}, nil
}

// GetMetadata extracts document metadata
func (d *CustomPDFDocument) GetMetadata() (*Metadata, error) {
	if d.closed {
		return nil, &WrapperError{Library: LibraryCustom, Op: "get_metadata", Err: ErrDocumentClosed.Err}
	}

	metadata := &Metadata{
		Custom: make(map[string]string),
	}

	// Get Info dictionary from trailer
	trailer := d.parser.GetTrailer()
	if trailer != nil {
		infoObj := trailer.Get("Info")
		if infoObj.Type() == custom.TypeIndirectRef {
			if infoDict, err := d.parser.ResolveIndirectObject(infoObj); err == nil {
				if infoDict.Type() == custom.TypeDictionary {
					info := infoDict.(*custom.Dictionary)
					metadata.Title = info.GetString("Title")
					metadata.Author = info.GetString("Author")
					metadata.Subject = info.GetString("Subject")
					metadata.Keywords = info.GetString("Keywords")
					metadata.Creator = info.GetString("Creator")
					metadata.Producer = info.GetString("Producer")
					// TODO: Parse dates
				}
			}
		}
	}

	return metadata, nil
}

// GetVersion returns the PDF version
func (d *CustomPDFDocument) GetVersion() (string, error) {
	if d.closed {
		return "", &WrapperError{Library: LibraryCustom, Op: "get_version", Err: ErrDocumentClosed.Err}
	}

	return d.parser.GetVersion(), nil
}

// Close closes the document
func (d *CustomPDFDocument) Close() error {
	d.closed = true
	if d.file != nil {
		return d.file.Close()
	}
	return nil
}

// ExtractText extracts text from a specific page
func (d *CustomPDFDocument) ExtractText(pageNum int) ([]TextElement, error) {
	if d.closed {
		return nil, &WrapperError{Library: LibraryCustom, Op: "extract_text", Err: ErrDocumentClosed.Err}
	}

	// TODO: Implement text extraction using custom parser
	return []TextElement{}, nil
}

// ExtractImages extracts images from a specific page
func (d *CustomPDFDocument) ExtractImages(pageNum int) ([]ImageElement, error) {
	if d.closed {
		return nil, &WrapperError{Library: LibraryCustom, Op: "extract_images", Err: ErrDocumentClosed.Err}
	}

	// TODO: Implement image extraction using custom parser
	return []ImageElement{}, nil
}

// ExtractForms extracts form fields from the document
func (d *CustomPDFDocument) ExtractForms() ([]extraction.FormField, error) {
	if d.closed {
		return nil, &WrapperError{Library: LibraryCustom, Op: "extract_forms", Err: ErrDocumentClosed.Err}
	}

	catalog := d.parser.GetCatalog()
	if catalog == nil {
		return []extraction.FormField{}, nil
	}

	acroFormParser := custom.NewAcroFormParser(d.parser)
	acroForm, err := acroFormParser.ParseAcroForm(catalog)
	if err != nil {
		return nil, &WrapperError{
			Library: LibraryCustom,
			Op:      "extract_forms",
			Err:     fmt.Errorf("failed to parse AcroForm: %w", err),
		}
	}

	if acroForm == nil {
		return []extraction.FormField{}, nil
	}

	return acroFormParser.ConvertToExtractionFormFields(acroForm.Fields)
}

// ExtractTables extracts tables from a specific page
func (d *CustomPDFDocument) ExtractTables(pageNum int) ([]TableElement, error) {
	if d.closed {
		return nil, &WrapperError{Library: LibraryCustom, Op: "extract_tables", Err: ErrDocumentClosed.Err}
	}

	// TODO: Implement table extraction using custom parser
	return []TableElement{}, nil
}

// ExtractAnnotations extracts annotations from a specific page
func (d *CustomPDFDocument) ExtractAnnotations(pageNum int) ([]AnnotationElement, error) {
	if d.closed {
		return nil, &WrapperError{Library: LibraryCustom, Op: "extract_annotations", Err: ErrDocumentClosed.Err}
	}

	// TODO: Implement annotation extraction using custom parser
	return []AnnotationElement{}, nil
}

// GetCatalog returns the PDF catalog
func (d *CustomPDFDocument) GetCatalog() (*Catalog, error) {
	if d.closed {
		return nil, &WrapperError{Library: LibraryCustom, Op: "get_catalog", Err: ErrDocumentClosed.Err}
	}

	catalogDict := d.parser.GetCatalog()
	if catalogDict == nil {
		return nil, &WrapperError{
			Library: LibraryCustom,
			Op:      "get_catalog",
			Err:     fmt.Errorf("no catalog found"),
		}
	}

	catalog := &Catalog{
		Version:  catalogDict.GetString("Version"),
		Metadata: make(map[string]interface{}),
	}

	// Convert dictionary to metadata map
	for _, key := range catalogDict.Keys {
		keyName := key.Value
		obj := catalogDict.Get(keyName)
		catalog.Metadata[keyName] = obj.String()
	}

	return catalog, nil
}

// GetContentStream returns the content stream for a page
func (d *CustomPDFDocument) GetContentStream(pageNum int) ([]byte, error) {
	if d.closed {
		return nil, &WrapperError{Library: LibraryCustom, Op: "get_content_stream", Err: ErrDocumentClosed.Err}
	}

	// TODO: Implement content stream access using custom parser
	return nil, &WrapperError{
		Library: LibraryCustom,
		Op:      "get_content_stream",
		Err:     fmt.Errorf("not yet implemented"),
	}
}

// GetPageResources returns resources for a specific page
func (d *CustomPDFDocument) GetPageResources(pageNum int) (*Resources, error) {
	if d.closed {
		return nil, &WrapperError{Library: LibraryCustom, Op: "get_page_resources", Err: ErrDocumentClosed.Err}
	}

	// TODO: Implement page resource extraction using custom parser
	return &Resources{}, nil
}

// IsEncrypted checks if the document is encrypted
func (d *CustomPDFDocument) IsEncrypted() bool {
	if d.closed || d.parser == nil {
		return false
	}

	if d.securityHandler != nil {
		return d.securityHandler.IsEncrypted()
	}

	// Fallback check
	trailer := d.parser.GetTrailer()
	if trailer == nil {
		return false
	}

	encryptObj := trailer.Get("Encrypt")
	return encryptObj.Type() != custom.TypeNull
}

// RequiresPassword checks if the document requires a password
func (d *CustomPDFDocument) RequiresPassword() bool {
	if d.closed || d.parser == nil {
		return false
	}

	// If we have a security handler, use it
	if d.securityHandler != nil {
		return d.securityHandler.IsEncrypted() && !d.securityHandler.IsAuthenticated()
	}

	// Otherwise, check if document is encrypted
	return d.IsEncrypted()
}

// ValidatePassword validates a password for encrypted documents
func (d *CustomPDFDocument) ValidatePassword(password string) error {
	if d.closed || d.parser == nil {
		return &WrapperError{
			Library: LibraryCustom,
			Op:      "validate_password",
			Err:     fmt.Errorf("document is closed"),
		}
	}

	if !d.IsEncrypted() {
		return nil // No password needed for unencrypted documents
	}

	// Initialize security handler if not already done
	if d.securityHandler == nil {
		if err := d.initializeSecurityHandler(); err != nil {
			return &WrapperError{
				Library: LibraryCustom,
				Op:      "validate_password",
				Err:     fmt.Errorf("failed to initialize security handler: %w", err),
			}
		}
	}

	if d.securityHandler == nil {
		return &WrapperError{
			Library: LibraryCustom,
			Op:      "validate_password",
			Err:     fmt.Errorf("no security handler available"),
		}
	}

	// Attempt authentication
	if err := d.securityHandler.Authenticate([]byte(password)); err != nil {
		return &WrapperError{
			Library: LibraryCustom,
			Op:      "validate_password",
			Err:     fmt.Errorf("password validation failed: %w", err),
		}
	}

	return nil
}

// initializeSecurityHandler creates and initializes the security handler for encrypted documents
func (d *CustomPDFDocument) initializeSecurityHandler() error {
	if d.parser == nil {
		return fmt.Errorf("parser not available")
	}

	trailer := d.parser.GetTrailer()
	if trailer == nil {
		return fmt.Errorf("trailer not available")
	}

	encryptObj := trailer.Get("Encrypt")
	if encryptObj.Type() == custom.TypeNull {
		return nil // Document is not encrypted
	}

	// Parse encryption dictionary
	secParser := security.NewSecurityParser(&customObjectResolver{parser: d.parser})

	// Convert custom object to map[string]interface{} for parsing
	encryptDict, err := d.convertToMap(encryptObj)
	if err != nil {
		return fmt.Errorf("failed to convert encryption object: %w", err)
	}

	encDict, err := secParser.ParseEncryptDict(encryptDict)
	if err != nil {
		return fmt.Errorf("failed to parse encryption dictionary: %w", err)
	}

	// Get file ID from trailer
	fileID := d.getFileID()

	// Create security handler
	d.securityHandler = security.NewStandardSecurityHandler(encDict, fileID)

	return nil
}

// convertToMap converts a custom PDF object to map[string]interface{} for security parsing
func (d *CustomPDFDocument) convertToMap(obj custom.PDFObject) (map[string]interface{}, error) {
	if obj.Type() != custom.TypeDictionary {
		return nil, fmt.Errorf("object is not a dictionary")
	}

	result := make(map[string]interface{})
	// This would need to be implemented based on the custom parser's API
	// For now, return an error indicating this needs implementation
	return result, fmt.Errorf("convertToMap not yet implemented - needs custom parser API details")
}

// getFileID extracts the file identifier from the PDF trailer
func (d *CustomPDFDocument) getFileID() []byte {
	if d.parser == nil {
		return nil
	}

	trailer := d.parser.GetTrailer()
	if trailer == nil {
		return nil
	}

	idObj := trailer.Get("ID")
	if idObj.Type() == custom.TypeNull {
		return nil
	}

	// This would need to be implemented based on the custom parser's API
	// For now, return a default file ID
	return []byte{0x01, 0x23, 0x45, 0x67, 0x89, 0xAB, 0xCD, 0xEF, 0xFE, 0xDC, 0xBA, 0x98, 0x76, 0x54, 0x32, 0x10}
}

// customObjectResolver implements security.ObjectResolver for the custom parser
type customObjectResolver struct {
	parser *custom.CustomPDFParser
}

func (r *customObjectResolver) ResolveObject(ref interface{}) (interface{}, error) {
	// This would need to be implemented based on the custom parser's API
	return nil, fmt.Errorf("ResolveObject not yet implemented - needs custom parser API details")
}

func (r *customObjectResolver) GetObject(objNum, genNum int) (interface{}, error) {
	// This would need to be implemented based on the custom parser's API
	return nil, fmt.Errorf("GetObject not yet implemented - needs custom parser API details")
}

// CustomPDFPage implements PDFPage interface using custom PDF parsing
type CustomPDFPage struct {
	pageNum int
	config  FactoryConfig
	// TODO: Add custom page fields here
	// page *pdf.Page
}

// GetNumber returns the page number
func (p *CustomPDFPage) GetNumber() int {
	return p.pageNum
}

// GetSize returns the page size
func (p *CustomPDFPage) GetSize() (*PageSize, error) {
	// TODO: Implement page size extraction using custom parser
	return &PageSize{
		Width:  612.0, // Default US Letter width
		Height: 792.0, // Default US Letter height
		Unit:   "pt",
	}, nil
}

// GetContent returns the page content stream
func (p *CustomPDFPage) GetContent() ([]byte, error) {
	// TODO: Implement content stream access using custom parser
	return nil, &WrapperError{
		Library: LibraryCustom,
		Op:      "get_content",
		Err:     fmt.Errorf("not yet implemented"),
	}
}

// GetResources returns the page resources
func (p *CustomPDFPage) GetResources() (*Resources, error) {
	// TODO: Implement resource extraction using custom parser
	return &Resources{}, nil
}

// GetAnnotations returns annotations on this page
func (p *CustomPDFPage) GetAnnotations() ([]AnnotationElement, error) {
	// TODO: Implement annotation extraction using custom parser
	return []AnnotationElement{}, nil
}

// GetText returns text elements on this page
func (p *CustomPDFPage) GetText() ([]TextElement, error) {
	// TODO: Implement text extraction using custom parser
	return []TextElement{}, nil
}

// GetImages returns image elements on this page
func (p *CustomPDFPage) GetImages() ([]ImageElement, error) {
	// TODO: Implement image extraction using custom parser
	return []ImageElement{}, nil
}
