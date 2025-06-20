package pdf

// FileInfo represents information about a PDF file
type FileInfo struct {
	Path         string `json:"path"`
	Name         string `json:"name"`
	Size         int64  `json:"size"`
	ModifiedTime string `json:"modified_time"`
}

// ImageInfo represents information about an image in a PDF
type ImageInfo struct {
	PageNumber int    `json:"page_number"`
	Width      int    `json:"width"`
	Height     int    `json:"height"`
	Format     string `json:"format"`
	Size       int64  `json:"size"`
}

// Request Types

// PDFReadFileRequest represents a request to read a PDF file
type PDFReadFileRequest struct {
	Path string `json:"path"`
}

// PDFAssetsFileRequest represents a request to get visual assets from a PDF file
type PDFAssetsFileRequest struct {
	Path string `json:"path"`
}

// PDFValidateFileRequest represents a request to validate a PDF file
type PDFValidateFileRequest struct {
	Path string `json:"path"`
}

// PDFStatsFileRequest represents a request to get stats about a PDF file
type PDFStatsFileRequest struct {
	Path string `json:"path"`
}

// PDFSearchDirectoryRequest represents a request to search for PDF files in a directory
type PDFSearchDirectoryRequest struct {
	Directory string `json:"directory"`
	Query     string `json:"query"`
}

// PDFStatsDirectoryRequest represents a request to get directory statistics
type PDFStatsDirectoryRequest struct {
	Directory string `json:"directory"`
}

// Response Types

// PDFReadFileResult represents the result of a PDF read operation
type PDFReadFileResult struct {
	Content     string `json:"content"`
	Path        string `json:"path"`
	Pages       int    `json:"pages"`
	Size        int64  `json:"size"`
	ContentType string `json:"content_type"` // "text", "scanned_images", "mixed", "no_content"
	HasImages   bool   `json:"has_images"`   // Whether the PDF contains extractable images
	ImageCount  int    `json:"image_count"`  // Number of images detected
}

// PDFAssetsFileResult represents the result of a PDF assets extraction operation
type PDFAssetsFileResult struct {
	Path       string      `json:"path"`
	Images     []ImageInfo `json:"images"`
	TotalCount int         `json:"total_count"`
}

// PDFValidateFileResult represents the result of a PDF validation operation
type PDFValidateFileResult struct {
	Valid   bool   `json:"valid"`
	Path    string `json:"path"`
	Message string `json:"message,omitempty"`
}

// PDFStatsFileResult represents the result of a PDF file stats operation
type PDFStatsFileResult struct {
	Path         string `json:"path"`
	Size         int64  `json:"size"`
	Pages        int    `json:"pages"`
	CreatedDate  string `json:"created_date,omitempty"`
	ModifiedDate string `json:"modified_date"`
	Title        string `json:"title,omitempty"`
	Author       string `json:"author,omitempty"`
	Subject      string `json:"subject,omitempty"`
	Producer     string `json:"producer,omitempty"`
}

// PDFSearchDirectoryResult represents the result of a PDF search operation
type PDFSearchDirectoryResult struct {
	Files       []FileInfo `json:"files"`
	TotalCount  int        `json:"total_count"`
	Directory   string     `json:"directory"`
	SearchQuery string     `json:"search_query,omitempty"`
}

// PDFStatsDirectoryResult represents the result of directory statistics
type PDFStatsDirectoryResult struct {
	Directory        string `json:"directory"`
	TotalFiles       int    `json:"total_files"`
	TotalSize        int64  `json:"total_size"`
	LargestFileSize  int64  `json:"largest_file_size"`
	LargestFileName  string `json:"largest_file_name"`
	SmallestFileSize int64  `json:"smallest_file_size"`
	SmallestFileName string `json:"smallest_file_name"`
	AverageFileSize  int64  `json:"average_file_size"`
}

// PDFServerInfoRequest represents a request to get server information and capabilities
type PDFServerInfoRequest struct {
	// No parameters needed for server info
}

// PDFServerInfoResult represents server information and usage guidance
type PDFServerInfoResult struct {
	ServerName        string     `json:"server_name"`
	Version           string     `json:"version"`
	DefaultDirectory  string     `json:"default_directory"`
	MaxFileSize       int64      `json:"max_file_size"`
	AvailableTools    []ToolInfo `json:"available_tools"`
	DirectoryContents []FileInfo `json:"directory_contents"`
	UsageGuidance     string     `json:"usage_guidance"`
	SupportedFormats  []string   `json:"supported_formats"`
}

// ToolInfo represents information about an available tool
type ToolInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Usage       string `json:"usage"`
	Parameters  string `json:"parameters"`
}

// New Extraction Tool Request Types

// PDFExtractStructuredRequest represents a request for structured content extraction
type PDFExtractStructuredRequest struct {
	Path   string           `json:"path"`
	Mode   string           `json:"mode,omitempty"`
	Config ExtractionConfig `json:"config,omitempty"`
	Query  *ContentQuery    `json:"query,omitempty"`
}

// PDFExtractTablesRequest represents a request for table extraction
type PDFExtractTablesRequest struct {
	Path   string           `json:"path"`
	Config ExtractionConfig `json:"config,omitempty"`
}

// PDFExtractSemanticRequest represents a request for semantic content extraction
type PDFExtractSemanticRequest struct {
	Path   string           `json:"path"`
	Config ExtractionConfig `json:"config,omitempty"`
}

// PDFExtractCompleteRequest represents a request for complete content extraction
type PDFExtractCompleteRequest struct {
	Path   string           `json:"path"`
	Config ExtractionConfig `json:"config,omitempty"`
}

// PDFQueryContentRequest represents a request to query extracted content
type PDFQueryContentRequest struct {
	Path  string       `json:"path"`
	Query ContentQuery `json:"query"`
}

// PDFGetPageInfoRequest represents a request for page information
type PDFGetPageInfoRequest struct {
	Path string `json:"path"`
}

// PDFGetMetadataRequest represents a request for document metadata
type PDFGetMetadataRequest struct {
	Path string `json:"path"`
}

// Configuration Types

// ExtractionConfig provides configuration for extraction operations
type ExtractionConfig struct {
	ExtractText        bool    `json:"extract_text,omitempty"`
	ExtractImages      bool    `json:"extract_images,omitempty"`
	ExtractTables      bool    `json:"extract_tables,omitempty"`
	ExtractForms       bool    `json:"extract_forms,omitempty"`
	ExtractAnnotations bool    `json:"extract_annotations,omitempty"`
	IncludeCoordinates bool    `json:"include_coordinates,omitempty"`
	IncludeFormatting  bool    `json:"include_formatting,omitempty"`
	Pages              []int   `json:"pages,omitempty"`
	MinConfidence      float64 `json:"min_confidence,omitempty"`
}

// ContentQuery represents a query for filtering content
type ContentQuery struct {
	ContentTypes  []string   `json:"content_types,omitempty"`
	Pages         []int      `json:"pages,omitempty"`
	BoundingBox   *Rectangle `json:"bounding_box,omitempty"`
	TextQuery     string     `json:"text_query,omitempty"`
	MinConfidence float64    `json:"min_confidence,omitempty"`
}

// Rectangle represents a rectangular area
type Rectangle struct {
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

// Response Types

// PDFExtractResult represents the result of content extraction
type PDFExtractResult struct {
	FilePath       string            `json:"file_path"`
	Mode           string            `json:"mode"`
	TotalPages     int               `json:"total_pages"`
	ProcessedPages []int             `json:"processed_pages"`
	Elements       []ContentElement  `json:"elements"`
	Tables         []TableElement    `json:"tables,omitempty"`
	Summary        ExtractionSummary `json:"summary"`
	Metadata       DocumentMetadata  `json:"metadata"`
	Warnings       []string          `json:"warnings,omitempty"`
	Errors         []string          `json:"errors,omitempty"`
}

// ContentElement represents a piece of extracted content
type ContentElement struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	PageNumber  int                    `json:"page_number"`
	BoundingBox Rectangle              `json:"bounding_box"`
	Content     interface{}            `json:"content"`
	Properties  map[string]interface{} `json:"properties,omitempty"`
	Children    []ContentElement       `json:"children,omitempty"`
	Parent      *string                `json:"parent,omitempty"`
	ZOrder      int                    `json:"z_order,omitempty"`
	Confidence  float64                `json:"confidence,omitempty"`
}

// TableElement represents extracted table data
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
	BoundingBox Rectangle   `json:"bounding_box"`
	IsHeader    bool        `json:"is_header,omitempty"`
}

// TableCol represents a table column
type TableCol struct {
	Index       int       `json:"index"`
	Header      string    `json:"header,omitempty"`
	BoundingBox Rectangle `json:"bounding_box"`
	DataType    string    `json:"data_type,omitempty"`
}

// TableCell represents a table cell
type TableCell struct {
	RowIndex    int       `json:"row_index"`
	ColIndex    int       `json:"col_index"`
	Content     string    `json:"content"`
	BoundingBox Rectangle `json:"bounding_box"`
	DataType    string    `json:"data_type,omitempty"`
	Confidence  float64   `json:"confidence,omitempty"`
}

// ExtractionSummary provides a summary of extraction results
type ExtractionSummary struct {
	ContentTypes  map[string]int `json:"content_types"`
	TotalElements int            `json:"total_elements"`
	PageBreakdown []PageSummary  `json:"page_breakdown,omitempty"`
	HasStructure  bool           `json:"has_structure"`
	Quality       string         `json:"quality"`
	Suggestions   []string       `json:"suggestions,omitempty"`
}

// PageSummary provides summary for a single page
type PageSummary struct {
	Page     int            `json:"page"`
	Elements int            `json:"elements"`
	Types    map[string]int `json:"types"`
}

// DocumentMetadata represents document metadata
type DocumentMetadata struct {
	Title            string            `json:"title,omitempty"`
	Author           string            `json:"author,omitempty"`
	Subject          string            `json:"subject,omitempty"`
	Creator          string            `json:"creator,omitempty"`
	Producer         string            `json:"producer,omitempty"`
	CreationDate     string            `json:"creation_date,omitempty"`
	ModificationDate string            `json:"modification_date,omitempty"`
	Keywords         []string          `json:"keywords,omitempty"`
	PageLayout       string            `json:"page_layout,omitempty"`
	PageMode         string            `json:"page_mode,omitempty"`
	Version          string            `json:"version,omitempty"`
	Encrypted        bool              `json:"encrypted"`
	CustomProperties map[string]string `json:"custom_properties,omitempty"`
}

// PDFQueryResult represents query results
type PDFQueryResult struct {
	FilePath   string           `json:"file_path"`
	Query      ContentQuery     `json:"query"`
	MatchCount int              `json:"match_count"`
	Elements   []ContentElement `json:"elements"`
	Summary    QuerySummary     `json:"summary"`
}

// QuerySummary provides query result summary
type QuerySummary struct {
	TypeBreakdown map[string]int `json:"type_breakdown"`
	PageBreakdown map[int]int    `json:"page_breakdown"`
	Confidence    float64        `json:"avg_confidence"`
}

// PageInfo represents information about a PDF page
type PageInfo struct {
	Number   int       `json:"number"`
	Width    float64   `json:"width"`
	Height   float64   `json:"height"`
	Rotation int       `json:"rotation"`
	MediaBox Rectangle `json:"media_box"`
	CropBox  Rectangle `json:"crop_box,omitempty"`
}

// PDFPageInfoResult represents page information results
type PDFPageInfoResult struct {
	FilePath string     `json:"file_path"`
	Pages    []PageInfo `json:"pages"`
}

// PDFMetadataResult represents metadata extraction results
type PDFMetadataResult struct {
	FilePath string           `json:"file_path"`
	Metadata DocumentMetadata `json:"metadata"`
}
