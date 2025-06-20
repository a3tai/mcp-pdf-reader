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

// Streaming Request Types

// PDFStreamProcessRequest represents a request to process a PDF using streaming
type PDFStreamProcessRequest struct {
	Path           string                 `json:"path"`
	Config         *StreamingConfig       `json:"config,omitempty"`
	ExtractText    bool                   `json:"extract_text"`
	ExtractImages  bool                   `json:"extract_images"`
	ExtractForms   bool                   `json:"extract_forms"`
	PreserveFormat bool                   `json:"preserve_format"`
	ProgressReport bool                   `json:"progress_report"`
	Options        map[string]interface{} `json:"options,omitempty"`
}

// PDFStreamPageRequest represents a request to process pages using streaming
type PDFStreamPageRequest struct {
	Path          string           `json:"path"`
	Config        *StreamingConfig `json:"config,omitempty"`
	StartPage     int              `json:"start_page,omitempty"`
	EndPage       int              `json:"end_page,omitempty"`
	ExtractText   bool             `json:"extract_text"`
	ExtractImages bool             `json:"extract_images"`
	ExtractForms  bool             `json:"extract_forms"`
}

// PDFStreamTextRequest represents a request for text-only streaming extraction
type PDFStreamTextRequest struct {
	Path       string           `json:"path"`
	Config     *StreamingConfig `json:"config,omitempty"`
	OutputPath string           `json:"output_path,omitempty"`
}

// Streaming Response Types

// PDFStreamProcessResult represents the result of streaming PDF processing
type PDFStreamProcessResult struct {
	FilePath        string                    `json:"file_path"`
	Content         *StreamProcessedContent   `json:"content"`
	Progress        *StreamProcessingProgress `json:"progress"`
	MemoryStats     *StreamMemoryStats        `json:"memory_stats"`
	ProcessingStats *StreamProcessingStats    `json:"processing_stats"`
	Status          string                    `json:"status"`
	Error           string                    `json:"error,omitempty"`
}

// PDFStreamPageResult represents the result of page streaming
type PDFStreamPageResult struct {
	FilePath    string       `json:"file_path"`
	Pages       []StreamPage `json:"pages"`
	TotalPages  int          `json:"total_pages"`
	ProcessedAt int64        `json:"processed_at"`
	Status      string       `json:"status"`
	Error       string       `json:"error,omitempty"`
}

// PDFStreamTextResult represents the result of text streaming
type PDFStreamTextResult struct {
	FilePath    string `json:"file_path"`
	OutputPath  string `json:"output_path,omitempty"`
	TextLength  int    `json:"text_length"`
	ProcessedAt int64  `json:"processed_at"`
	Status      string `json:"status"`
	Error       string `json:"error,omitempty"`
}

// Streaming Configuration Types

// StreamingConfig configures streaming processing parameters
type StreamingConfig struct {
	ChunkSizeMB    int     `json:"chunk_size_mb"`
	MaxMemoryMB    int     `json:"max_memory_mb"`
	CacheSize      int     `json:"cache_size"`
	BufferPoolSize int     `json:"buffer_pool_size"`
	GCTrigger      float64 `json:"gc_trigger"`
	EnableCaching  bool    `json:"enable_caching"`
}

// StreamProcessedContent contains all extracted content from streaming
type StreamProcessedContent struct {
	Text       string            `json:"text"`
	Images     []StreamImageInfo `json:"images"`
	Forms      []StreamFormInfo  `json:"forms"`
	Pages      []StreamPageInfo  `json:"pages"`
	TextBlocks []StreamTextBlock `json:"text_blocks,omitempty"`
}

// StreamProcessingProgress provides progress information
type StreamProcessingProgress struct {
	CurrentPage  int `json:"current_page"`
	TotalPages   int `json:"total_pages"`
	TextSize     int `json:"text_size"`
	ImageCount   int `json:"image_count"`
	FormCount    int `json:"form_count"`
	ObjectsFound int `json:"objects_found"`
}

// StreamMemoryStats provides memory usage statistics
type StreamMemoryStats struct {
	CurrentBytes    int64   `json:"current_bytes"`
	MaxBytes        int64   `json:"max_bytes"`
	UsagePercent    float64 `json:"usage_percent"`
	CacheHitRate    float64 `json:"cache_hit_rate"`
	XRefCacheSize   int     `json:"xref_cache_size"`
	ObjectCacheSize int     `json:"object_cache_size"`
}

// StreamProcessingStats provides processing statistics
type StreamProcessingStats struct {
	TotalChunks     int   `json:"total_chunks"`
	ProcessedChunks int   `json:"processed_chunks"`
	TotalObjects    int   `json:"total_objects"`
	ProcessingTime  int64 `json:"processing_time_ms"`
	BytesProcessed  int64 `json:"bytes_processed"`
	StartTime       int64 `json:"start_time"`
	EndTime         int64 `json:"end_time"`
}

// StreamPage represents a processed page in streaming mode
type StreamPage struct {
	Number      int                `json:"number"`
	Offset      int64              `json:"offset"`
	Length      int64              `json:"length"`
	Content     StreamPageContent  `json:"content"`
	Metadata    StreamPageMetadata `json:"metadata"`
	ProcessedAt int64              `json:"processed_at"`
	Status      string             `json:"status"`
	Error       string             `json:"error,omitempty"`
}

// StreamPageContent contains content extracted from a page
type StreamPageContent struct {
	Text       string            `json:"text"`
	Images     []StreamImageInfo `json:"images"`
	Forms      []StreamFormInfo  `json:"forms"`
	TextBlocks []StreamTextBlock `json:"text_blocks,omitempty"`
}

// StreamPageMetadata contains metadata about a page
type StreamPageMetadata struct {
	MediaBox    Rectangle `json:"media_box"`
	CropBox     Rectangle `json:"crop_box,omitempty"`
	Rotation    int       `json:"rotation"`
	HasImages   bool      `json:"has_images"`
	HasForms    bool      `json:"has_forms"`
	TextLength  int       `json:"text_length"`
	ObjectCount int       `json:"object_count"`
}

// StreamImageInfo contains information about images in streaming mode
type StreamImageInfo struct {
	ObjectNumber int    `json:"object_number"`
	PageNumber   int    `json:"page_number"`
	Offset       int64  `json:"offset"`
	Length       int64  `json:"length"`
	Width        int    `json:"width"`
	Height       int    `json:"height"`
	Format       string `json:"format,omitempty"`
}

// StreamFormInfo contains information about forms in streaming mode
type StreamFormInfo struct {
	ObjectNumber int    `json:"object_number"`
	PageNumber   int    `json:"page_number"`
	Offset       int64  `json:"offset"`
	FieldType    string `json:"field_type"`
	FieldName    string `json:"field_name"`
	FieldValue   string `json:"field_value"`
}

// StreamPageInfo contains information about pages in streaming mode
type StreamPageInfo struct {
	Number   int       `json:"number"`
	Offset   int64     `json:"offset"`
	Length   int64     `json:"length"`
	MediaBox Rectangle `json:"media_box"`
}

// StreamTextBlock represents a positioned text block in streaming mode
type StreamTextBlock struct {
	Text     string  `json:"text"`
	X        float64 `json:"x"`
	Y        float64 `json:"y"`
	Width    float64 `json:"width"`
	Height   float64 `json:"height"`
	FontSize float64 `json:"font_size"`
	FontName string  `json:"font_name,omitempty"`
}

// Page Range Extraction Types

// PDFExtractPageRangeRequest represents a request to extract specific page ranges
type PDFExtractPageRangeRequest struct {
	Path               string          `json:"path"`
	Ranges             []PageRangeSpec `json:"ranges"`
	ContentTypes       []string        `json:"content_types"`       // text, images, forms, metadata
	PreserveFormatting bool            `json:"preserve_formatting"` // Whether to preserve text formatting
	IncludeMetadata    bool            `json:"include_metadata"`    // Whether to include page metadata
	ExtractImages      bool            `json:"extract_images"`      // Whether to extract images
	ExtractForms       bool            `json:"extract_forms"`       // Whether to extract forms
	OutputFormat       string          `json:"output_format"`       // json, xml, plain
}

// PDFExtractPageRangeResult represents the result of page range extraction
type PDFExtractPageRangeResult struct {
	FilePath    string                       `json:"file_path"`
	Pages       map[int]ExtractedPageContent `json:"pages"`
	TotalPages  int                          `json:"total_pages"`
	Ranges      []PageRangeSpec              `json:"ranges"`
	ProcessedAt int64                        `json:"processed_at"`
	Metadata    ExtractionResultMetadata     `json:"metadata"`
	Status      string                       `json:"status"`
	Error       string                       `json:"error,omitempty"`
}

// PageRangeSpec represents a range of pages to extract
type PageRangeSpec struct {
	Start int `json:"start"`
	End   int `json:"end"`
}

// ExtractedPageContent represents content extracted from a single page
type ExtractedPageContent struct {
	PageNumber int                       `json:"page_number"`
	Text       string                    `json:"text"`
	Images     []ExtractedImageReference `json:"images"`
	Forms      []ExtractedFormField      `json:"forms"`
	Metadata   ExtractedPageMetadata     `json:"metadata"`
	TextBlocks []ExtractedTextBlock      `json:"text_blocks,omitempty"`
}

// ExtractedPageMetadata represents metadata about an extracted page
type ExtractedPageMetadata struct {
	MediaBox      Rectangle `json:"media_box"`
	CropBox       Rectangle `json:"crop_box,omitempty"`
	Rotation      int       `json:"rotation"`
	UserUnit      float64   `json:"user_unit,omitempty"`
	ResourceCount int       `json:"resource_count"`
	ObjectCount   int       `json:"object_count"`
}

// ExtractedImageReference represents an image reference in an extracted page
type ExtractedImageReference struct {
	ObjectID   int     `json:"object_id"`
	X          float64 `json:"x"`
	Y          float64 `json:"y"`
	Width      float64 `json:"width"`
	Height     float64 `json:"height"`
	Format     string  `json:"format"`
	ColorSpace string  `json:"color_space,omitempty"`
}

// ExtractedFormField represents a form field in an extracted page
type ExtractedFormField struct {
	FieldType  string  `json:"field_type"`
	FieldName  string  `json:"field_name"`
	FieldValue string  `json:"field_value"`
	X          float64 `json:"x"`
	Y          float64 `json:"y"`
	Width      float64 `json:"width"`
	Height     float64 `json:"height"`
}

// ExtractedTextBlock represents a positioned text block in an extracted page
type ExtractedTextBlock struct {
	Text     string  `json:"text"`
	X        float64 `json:"x"`
	Y        float64 `json:"y"`
	Width    float64 `json:"width"`
	Height   float64 `json:"height"`
	FontName string  `json:"font_name"`
	FontSize float64 `json:"font_size"`
	Color    string  `json:"color,omitempty"`
}

// ExtractionResultMetadata provides information about the extraction process
type ExtractionResultMetadata struct {
	ProcessingTime int64 `json:"processing_time_ms"`
	CacheHits      int   `json:"cache_hits"`
	CacheMisses    int   `json:"cache_misses"`
	ObjectsParsed  int   `json:"objects_parsed"`
	BytesRead      int64 `json:"bytes_read"`
	MemoryUsage    int64 `json:"memory_usage"`
}
