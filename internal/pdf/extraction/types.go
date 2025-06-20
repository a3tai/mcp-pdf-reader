package extraction

import (
	"time"
)

// ContentType represents the type of content extracted from PDF
type ContentType string

const (
	ContentTypeText       ContentType = "text"
	ContentTypeImage      ContentType = "image"
	ContentTypeVector     ContentType = "vector"
	ContentTypeForm       ContentType = "form"
	ContentTypeAnnotation ContentType = "annotation"
	ContentTypeMetadata   ContentType = "metadata"
	ContentTypeStructural ContentType = "structural"
)

// ExtractionMode defines how content should be extracted
type ExtractionMode string

const (
	ModeRaw        ExtractionMode = "raw"        // Basic text extraction
	ModeStructured ExtractionMode = "structured" // Preserve structure and positioning
	ModeSemantic   ExtractionMode = "semantic"   // Group related content logically
	ModeForm       ExtractionMode = "form"       // Focus on form fields and data
	ModeTable      ExtractionMode = "table"      // Detect and extract tabular data
	ModeComplete   ExtractionMode = "complete"   // Extract all available content types
)

// Coordinate represents a point in PDF coordinate space
type Coordinate struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// BoundingBox represents a rectangular area in PDF coordinate space
type BoundingBox struct {
	LowerLeft  Coordinate `json:"lower_left"`
	UpperRight Coordinate `json:"upper_right"`
	Width      float64    `json:"width"`
	Height     float64    `json:"height"`
}

// TextProperties represents text formatting and style information
type TextProperties struct {
	FontName    string  `json:"font_name,omitempty"`
	FontSize    float64 `json:"font_size,omitempty"`
	Bold        bool    `json:"bold,omitempty"`
	Italic      bool    `json:"italic,omitempty"`
	Color       string  `json:"color,omitempty"`
	Rotation    float64 `json:"rotation,omitempty"`
	CharSpacing float64 `json:"char_spacing,omitempty"`
	WordSpacing float64 `json:"word_spacing,omitempty"`
	ScaleH      float64 `json:"scale_h,omitempty"`
	ScaleV      float64 `json:"scale_v,omitempty"`
}

// ContentElement represents a single piece of content from a PDF
type ContentElement struct {
	ID          string           `json:"id"`
	Type        ContentType      `json:"type"`
	PageNumber  int              `json:"page_number"`
	BoundingBox BoundingBox      `json:"bounding_box"`
	Content     interface{}      `json:"content"`
	Properties  interface{}      `json:"properties,omitempty"`
	Children    []ContentElement `json:"children,omitempty"`
	Parent      *string          `json:"parent,omitempty"`
	ZOrder      int              `json:"z_order,omitempty"`
	Confidence  float64          `json:"confidence,omitempty"`
}

// TextElement represents extracted text content
type TextElement struct {
	Text       string         `json:"text"`
	Properties TextProperties `json:"properties"`
	Words      []WordElement  `json:"words,omitempty"`
	Lines      []LineElement  `json:"lines,omitempty"`
}

// WordElement represents a single word with positioning
type WordElement struct {
	Text        string         `json:"text"`
	BoundingBox BoundingBox    `json:"bounding_box"`
	Properties  TextProperties `json:"properties"`
	Confidence  float64        `json:"confidence,omitempty"`
}

// LineElement represents a line of text
type LineElement struct {
	Text        string         `json:"text"`
	BoundingBox BoundingBox    `json:"bounding_box"`
	Words       []WordElement  `json:"words"`
	Properties  TextProperties `json:"properties"`
	Baseline    float64        `json:"baseline,omitempty"`
}

// ImageElement represents extracted image content
type ImageElement struct {
	Format           string `json:"format"` // PNG, JPEG, etc.
	Width            int    `json:"width"`
	Height           int    `json:"height"`
	ColorSpace       string `json:"color_space,omitempty"`
	BitsPerComponent int    `json:"bits_per_component,omitempty"`
	Data             []byte `json:"data,omitempty"`
	Hash             string `json:"hash,omitempty"` // For deduplication
	Size             int64  `json:"size"`
}

// VectorElement represents vector graphics content
type VectorElement struct {
	Type        string      `json:"type"` // path, line, curve, etc.
	Commands    []VectorCmd `json:"commands"`
	StrokeColor string      `json:"stroke_color,omitempty"`
	FillColor   string      `json:"fill_color,omitempty"`
	StrokeWidth float64     `json:"stroke_width,omitempty"`
}

// VectorCmd represents a vector drawing command
type VectorCmd struct {
	Command string       `json:"command"` // moveto, lineto, curveto, etc.
	Points  []Coordinate `json:"points"`
}

// FormElement represents form fields and interactive elements
type FormElement struct {
	Field FormField `json:"field"`
}

// AnnotationElement represents PDF annotations
type AnnotationElement struct {
	AnnotationType string    `json:"annotation_type"` // highlight, note, link, etc.
	Content        string    `json:"content,omitempty"`
	Author         string    `json:"author,omitempty"`
	CreationDate   time.Time `json:"creation_date,omitempty"`
	ModifiedDate   time.Time `json:"modified_date,omitempty"`
	URI            string    `json:"uri,omitempty"` // For link annotations
	Destination    string    `json:"destination,omitempty"`
	Color          string    `json:"color,omitempty"`
}

// TableElement represents detected tabular data
type TableElement struct {
	Rows       []TableRow `json:"rows"`
	Columns    []TableCol `json:"columns"`
	CellCount  int        `json:"cell_count"`
	HasHeaders bool       `json:"has_headers,omitempty"`
	Confidence float64    `json:"confidence,omitempty"`
}

// TableRow represents a table row
type TableRow struct {
	Index       int         `json:"index"`
	Cells       []TableCell `json:"cells"`
	BoundingBox BoundingBox `json:"bounding_box"`
	IsHeader    bool        `json:"is_header,omitempty"`
}

// TableCol represents a table column
type TableCol struct {
	Index       int         `json:"index"`
	Header      string      `json:"header,omitempty"`
	BoundingBox BoundingBox `json:"bounding_box"`
	DataType    string      `json:"data_type,omitempty"` // text, number, date, etc.
}

// TableCell represents a single table cell
type TableCell struct {
	RowIndex    int         `json:"row_index"`
	ColIndex    int         `json:"col_index"`
	Content     string      `json:"content"`
	BoundingBox BoundingBox `json:"bounding_box"`
	Spans       CellSpan    `json:"spans,omitempty"`
	DataType    string      `json:"data_type,omitempty"`
	Confidence  float64     `json:"confidence,omitempty"`
}

// CellSpan represents cell spanning information
type CellSpan struct {
	RowSpan int `json:"row_span,omitempty"`
	ColSpan int `json:"col_span,omitempty"`
}

// StructuralElement represents structural information
type StructuralElement struct {
	StructType string `json:"struct_type"` // paragraph, heading, list, etc.
	Level      int    `json:"level,omitempty"`
	Role       string `json:"role,omitempty"`
	Title      string `json:"title,omitempty"`
	Language   string `json:"language,omitempty"`
}

// ExtractionConfig defines extraction parameters
type ExtractionConfig struct {
	Mode               ExtractionMode `json:"mode"`
	ExtractText        bool           `json:"extract_text"`
	ExtractImages      bool           `json:"extract_images"`
	ExtractVectors     bool           `json:"extract_vectors"`
	ExtractForms       bool           `json:"extract_forms"`
	ExtractAnnotations bool           `json:"extract_annotations"`
	ExtractTables      bool           `json:"extract_tables"`
	PreserveFormatting bool           `json:"preserve_formatting"`
	DetectStructure    bool           `json:"detect_structure"`
	IncludeCoordinates bool           `json:"include_coordinates"`
	IncludeProperties  bool           `json:"include_properties"`
	MinTextSize        float64        `json:"min_text_size,omitempty"`
	MaxTextSize        float64        `json:"max_text_size,omitempty"`
	MinImageSize       int            `json:"min_image_size,omitempty"`
	TableDetectionTh   float64        `json:"table_detection_threshold,omitempty"`
	OCREnabled         bool           `json:"ocr_enabled,omitempty"`
	OCRLanguages       []string       `json:"ocr_languages,omitempty"`
	Pages              []int          `json:"pages,omitempty"` // Specific pages to extract
}

// ExtractionResult represents the complete extraction result
type ExtractionResult struct {
	FilePath       string           `json:"file_path"`
	TotalPages     int              `json:"total_pages"`
	ProcessedPages []int            `json:"processed_pages"`
	Elements       []ContentElement `json:"elements"`
	Tables         []TableElement   `json:"tables,omitempty"`
	Metadata       PDFMetadata      `json:"metadata"`
	ExtractionInfo ExtractionInfo   `json:"extraction_info"`
	Warnings       []string         `json:"warnings,omitempty"`
	Errors         []string         `json:"errors,omitempty"`
}

// PDFMetadata represents document metadata
type PDFMetadata struct {
	Title            string            `json:"title,omitempty"`
	Author           string            `json:"author,omitempty"`
	Subject          string            `json:"subject,omitempty"`
	Creator          string            `json:"creator,omitempty"`
	Producer         string            `json:"producer,omitempty"`
	CreationDate     time.Time         `json:"creation_date,omitempty"`
	ModificationDate time.Time         `json:"modification_date,omitempty"`
	Keywords         []string          `json:"keywords,omitempty"`
	PageLayout       string            `json:"page_layout,omitempty"`
	PageMode         string            `json:"page_mode,omitempty"`
	Version          string            `json:"version,omitempty"`
	Encrypted        bool              `json:"encrypted"`
	CustomProperties map[string]string `json:"custom_properties,omitempty"`
}

// ExtractionInfo provides information about the extraction process
type ExtractionInfo struct {
	Mode            ExtractionMode  `json:"mode"`
	StartTime       time.Time       `json:"start_time"`
	EndTime         time.Time       `json:"end_time"`
	Duration        time.Duration   `json:"duration"`
	ElementCounts   ElementCounts   `json:"element_counts"`
	ProcessingStats ProcessingStats `json:"processing_stats"`
}

// ElementCounts tracks the number of each content type extracted
type ElementCounts struct {
	Text        int `json:"text"`
	Images      int `json:"images"`
	Vectors     int `json:"vectors"`
	Forms       int `json:"forms"`
	Annotations int `json:"annotations"`
	Tables      int `json:"tables"`
	Total       int `json:"total"`
}

// ProcessingStats provides statistics about the extraction process
type ProcessingStats struct {
	TextExtractionTime     time.Duration `json:"text_extraction_time"`
	ImageExtractionTime    time.Duration `json:"image_extraction_time"`
	VectorExtractionTime   time.Duration `json:"vector_extraction_time"`
	StructureDetectionTime time.Duration `json:"structure_detection_time"`
	OCRTime                time.Duration `json:"ocr_time,omitempty"`
	BytesProcessed         int64         `json:"bytes_processed"`
	MemoryUsed             int64         `json:"memory_used,omitempty"`
}

// Query represents a content query for filtering results
type Query struct {
	ContentTypes  []ContentType          `json:"content_types,omitempty"`
	Pages         []int                  `json:"pages,omitempty"`
	BoundingBox   *BoundingBox           `json:"bounding_box,omitempty"`
	TextQuery     string                 `json:"text_query,omitempty"`
	Properties    map[string]interface{} `json:"properties,omitempty"`
	MinConfidence float64                `json:"min_confidence,omitempty"`
}

// ExtractionRequest represents a request for content extraction
type ExtractionRequest struct {
	FilePath string           `json:"file_path"`
	Config   ExtractionConfig `json:"config"`
	Query    *Query           `json:"query,omitempty"`
}
