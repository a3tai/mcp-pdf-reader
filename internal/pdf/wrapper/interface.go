package wrapper

import (
	"fmt"
	"io"
	"time"

	"github.com/a3tai/mcp-pdf-reader/internal/pdf/extraction"
)

// PDFLibrary defines the unified interface for PDF operations across different libraries
type PDFLibrary interface {
	// Core operations
	Open(reader io.Reader) (PDFDocument, error)
	OpenFile(path string) (PDFDocument, error)
	Validate() error
	Close() error

	// Library identification
	GetLibraryType() LibraryType
	GetVersion() string
}

// PDFDocument represents a PDF document with unified operations
type PDFDocument interface {
	// Basic document operations
	GetPageCount() (int, error)
	GetPage(pageNum int) (PDFPage, error)
	GetMetadata() (*Metadata, error)
	GetVersion() (string, error)
	Close() error

	// Content extraction operations
	ExtractText(pageNum int) ([]TextElement, error)
	ExtractImages(pageNum int) ([]ImageElement, error)
	ExtractForms() ([]extraction.FormField, error)
	ExtractTables(pageNum int) ([]TableElement, error)
	ExtractAnnotations(pageNum int) ([]AnnotationElement, error)

	// Advanced document operations
	GetCatalog() (*Catalog, error)
	GetContentStream(pageNum int) ([]byte, error)
	GetPageResources(pageNum int) (*Resources, error)

	// Security and encryption
	IsEncrypted() bool
	RequiresPassword() bool
	ValidatePassword(password string) error
}

// PDFPage represents a single page in a PDF document
type PDFPage interface {
	GetNumber() int
	GetSize() (*PageSize, error)
	GetContent() ([]byte, error)
	GetResources() (*Resources, error)
	GetAnnotations() ([]AnnotationElement, error)
	GetText() ([]TextElement, error)
	GetImages() ([]ImageElement, error)
}

// LibraryType represents the underlying PDF library being used
type LibraryType string

const (
	LibraryCustom     LibraryType = "custom"
	LibraryPDFCPU     LibraryType = "pdfcpu"
	LibraryLedongthuc LibraryType = "ledongthuc"
	LibraryAuto       LibraryType = "auto" // Automatically select best library
)

// Common data structures used across all implementations

// Metadata contains PDF document metadata
type Metadata struct {
	Title        string            `json:"title,omitempty"`
	Author       string            `json:"author,omitempty"`
	Subject      string            `json:"subject,omitempty"`
	Keywords     string            `json:"keywords,omitempty"`
	Creator      string            `json:"creator,omitempty"`
	Producer     string            `json:"producer,omitempty"`
	CreationDate *time.Time        `json:"creation_date,omitempty"`
	ModDate      *time.Time        `json:"modification_date,omitempty"`
	Trapped      string            `json:"trapped,omitempty"`
	Custom       map[string]string `json:"custom,omitempty"`
}

// PageSize represents the dimensions of a PDF page
type PageSize struct {
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
	Unit   string  `json:"unit"` // "pt", "in", "mm", "cm"
}

// Point represents a coordinate point
type Point struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// Rectangle represents a rectangular area
type Rectangle struct {
	LowerLeft  Point   `json:"lower_left"`
	UpperRight Point   `json:"upper_right"`
	Width      float64 `json:"width"`
	Height     float64 `json:"height"`
}

// FontInfo represents font information
type FontInfo struct {
	Name     string  `json:"name"`
	Size     float64 `json:"size"`
	Bold     bool    `json:"bold"`
	Italic   bool    `json:"italic"`
	Encoding string  `json:"encoding,omitempty"`
}

// Color represents a color value
type Color struct {
	R float64 `json:"r"` // Red component (0-1)
	G float64 `json:"g"` // Green component (0-1)
	B float64 `json:"b"` // Blue component (0-1)
	A float64 `json:"a"` // Alpha component (0-1)
}

// TextElement represents extracted text with positioning and formatting
type TextElement struct {
	Text     string    `json:"text"`
	Position Rectangle `json:"position"`
	Font     FontInfo  `json:"font"`
	Color    Color     `json:"color,omitempty"`
	Style    TextStyle `json:"style,omitempty"`
}

// TextStyle represents text styling information
type TextStyle struct {
	Bold        bool    `json:"bold"`
	Italic      bool    `json:"italic"`
	Underline   bool    `json:"underline"`
	Strikeout   bool    `json:"strikeout"`
	Superscript bool    `json:"superscript"`
	Subscript   bool    `json:"subscript"`
	Rotation    float64 `json:"rotation,omitempty"`
}

// ImageElement represents extracted image content
type ImageElement struct {
	ID       string    `json:"id"`
	Position Rectangle `json:"position"`
	Width    int       `json:"width"`
	Height   int       `json:"height"`
	Format   string    `json:"format"` // "jpeg", "png", "tiff", etc.
	Data     []byte    `json:"data,omitempty"`
	Checksum string    `json:"checksum,omitempty"`
}

// TableElement represents table structure
type TableElement struct {
	Position Rectangle  `json:"position"`
	Rows     []TableRow `json:"rows"`
	Headers  []string   `json:"headers,omitempty"`
	Caption  string     `json:"caption,omitempty"`
}

// TableRow represents a row in a table
type TableRow struct {
	Cells []TableCell `json:"cells"`
}

// TableCell represents a cell in a table
type TableCell struct {
	Text     string    `json:"text"`
	Position Rectangle `json:"position"`
	ColSpan  int       `json:"col_span,omitempty"`
	RowSpan  int       `json:"row_span,omitempty"`
	IsHeader bool      `json:"is_header,omitempty"`
}

// AnnotationElement represents PDF annotations
type AnnotationElement struct {
	Type       string                 `json:"type"`
	Subtype    string                 `json:"subtype"`
	Position   Rectangle              `json:"position"`
	Content    string                 `json:"content,omitempty"`
	Author     string                 `json:"author,omitempty"`
	ModDate    *time.Time             `json:"modification_date,omitempty"`
	Properties map[string]interface{} `json:"properties,omitempty"`
}

// Catalog represents the PDF catalog dictionary
type Catalog struct {
	Version    string                 `json:"version,omitempty"`
	Pages      *PageTree              `json:"pages,omitempty"`
	AcroForm   *AcroForm              `json:"acro_form,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
	StructTree *StructTree            `json:"struct_tree,omitempty"`
}

// PageTree represents the page tree structure
type PageTree struct {
	Type  string `json:"type"`
	Count int    `json:"count"`
	Kids  []Page `json:"kids,omitempty"`
}

// Page represents a page reference in the page tree
type Page struct {
	Type      string                 `json:"type"`
	Parent    *PageTree              `json:"parent,omitempty"`
	MediaBox  Rectangle              `json:"media_box"`
	CropBox   *Rectangle             `json:"crop_box,omitempty"`
	Resources *Resources             `json:"resources,omitempty"`
	Contents  []byte                 `json:"contents,omitempty"`
	Annots    []AnnotationElement    `json:"annotations,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// AcroForm represents the AcroForm dictionary for PDF forms
type AcroForm struct {
	Fields          []extraction.FormField `json:"fields"`
	NeedAppearances bool                   `json:"need_appearances"`
	SigFlags        int                    `json:"sig_flags,omitempty"`
	CO              []string               `json:"co,omitempty"`
	DR              *Resources             `json:"dr,omitempty"`
	DA              string                 `json:"da,omitempty"`
	Q               int                    `json:"q,omitempty"`
}

// StructTree represents the structure tree for accessibility
type StructTree struct {
	Type     string                 `json:"type"`
	Children []StructElement        `json:"children,omitempty"`
	IDTree   map[string]interface{} `json:"id_tree,omitempty"`
}

// StructElement represents a structure element
type StructElement struct {
	Type       string                 `json:"type"`
	Tag        string                 `json:"tag"`
	Children   []StructElement        `json:"children,omitempty"`
	Content    string                 `json:"content,omitempty"`
	Attributes map[string]interface{} `json:"attributes,omitempty"`
}

// Resources represents PDF page resources
type Resources struct {
	Font       map[string]FontResource    `json:"font,omitempty"`
	XObject    map[string]XObjectResource `json:"x_object,omitempty"`
	ColorSpace map[string]interface{}     `json:"color_space,omitempty"`
	Pattern    map[string]interface{}     `json:"pattern,omitempty"`
	Shading    map[string]interface{}     `json:"shading,omitempty"`
	ExtGState  map[string]interface{}     `json:"ext_g_state,omitempty"`
	Properties map[string]interface{}     `json:"properties,omitempty"`
}

// FontResource represents a font resource
type FontResource struct {
	Type     string `json:"type"`
	Subtype  string `json:"subtype"`
	BaseFont string `json:"base_font"`
	Encoding string `json:"encoding,omitempty"`
}

// XObjectResource represents an XObject resource (images, forms, etc.)
type XObjectResource struct {
	Type    string `json:"type"`
	Subtype string `json:"subtype"`
	Width   int    `json:"width,omitempty"`
	Height  int    `json:"height,omitempty"`
	Filter  string `json:"filter,omitempty"`
}

// Error types for wrapper operations
type WrapperError struct {
	Library LibraryType `json:"library"`
	Op      string      `json:"operation"`
	Err     error       `json:"error"`
}

func (e *WrapperError) Error() string {
	return fmt.Sprintf("PDF %s library error in %s: %v", e.Library, e.Op, e.Err)
}

func (e *WrapperError) Unwrap() error {
	return e.Err
}

// Common error variables
var (
	ErrUnsupportedLibrary = &WrapperError{Op: "factory", Err: fmt.Errorf("unsupported library type")}
	ErrDocumentClosed     = &WrapperError{Op: "document", Err: fmt.Errorf("document is closed")}
	ErrInvalidPage        = &WrapperError{Op: "page", Err: fmt.Errorf("invalid page number")}
	ErrPasswordRequired   = &WrapperError{Op: "security", Err: fmt.Errorf("password required")}
	ErrInvalidPassword    = &WrapperError{Op: "security", Err: fmt.Errorf("invalid password")}
)
