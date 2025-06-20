package pdf

import (
	"fmt"
	"time"

	"github.com/a3tai/mcp-pdf-reader/internal/pdf/security"
)

// Service handles PDF file operations by orchestrating various PDF components
type Service struct {
	maxFileSize       int64
	reader            *Reader
	validator         *Validator
	stats             *Stats
	assets            *Assets
	search            *Search
	extractionService *ExtractionService
	pathValidator     *security.PathValidator
}

// NewService creates a new PDF service with all components
func NewService(maxFileSize int64, configuredDirectory string) (*Service, error) {
	pathValidator, err := security.NewPathValidator(configuredDirectory)
	if err != nil {
		return nil, fmt.Errorf("failed to create path validator: %w", err)
	}

	return &Service{
		maxFileSize:       maxFileSize,
		reader:            NewReader(maxFileSize),
		validator:         NewValidator(maxFileSize),
		stats:             NewStats(maxFileSize),
		assets:            NewAssets(maxFileSize),
		search:            NewSearch(maxFileSize),
		extractionService: NewExtractionService(maxFileSize),
		pathValidator:     pathValidator,
	}, nil
}

// PDFReadFile reads the content of a PDF file
func (s *Service) PDFReadFile(req PDFReadFileRequest) (*PDFReadFileResult, error) {
	if err := s.pathValidator.ValidatePath(req.Path); err != nil {
		return nil, fmt.Errorf("security validation failed: %w", err)
	}
	return s.reader.ReadFile(req)
}

// PDFAssetsFile extracts visual assets like images from a PDF file
func (s *Service) PDFAssetsFile(req PDFAssetsFileRequest) (*PDFAssetsFileResult, error) {
	if err := s.pathValidator.ValidatePath(req.Path); err != nil {
		return nil, fmt.Errorf("security validation failed: %w", err)
	}
	return s.assets.ExtractAssets(req)
}

// PDFValidateFile performs validation on a PDF file
func (s *Service) PDFValidateFile(req PDFValidateFileRequest) (*PDFValidateFileResult, error) {
	if err := s.pathValidator.ValidatePath(req.Path); err != nil {
		return nil, fmt.Errorf("security validation failed: %w", err)
	}
	return s.validator.ValidateFile(req)
}

// PDFStatsFile returns detailed statistics about a single PDF file
func (s *Service) PDFStatsFile(req PDFStatsFileRequest) (*PDFStatsFileResult, error) {
	if err := s.pathValidator.ValidatePath(req.Path); err != nil {
		return nil, fmt.Errorf("security validation failed: %w", err)
	}
	return s.stats.GetFileStats(req)
}

// PDFSearchDirectory searches for PDF files in a directory
func (s *Service) PDFSearchDirectory(req PDFSearchDirectoryRequest) (*PDFSearchDirectoryResult, error) {
	// If no directory specified, use configured directory
	if req.Directory == "" {
		req.Directory = s.pathValidator.GetConfiguredDirectory()
	}

	// Validate directory is within configured bounds
	if err := s.pathValidator.ValidateDirectory(req.Directory); err != nil {
		return nil, fmt.Errorf("security validation failed: %w", err)
	}

	return s.search.SearchDirectory(req)
}

// PDFStatsDirectory returns statistics about PDF files in a directory
func (s *Service) PDFStatsDirectory(req PDFStatsDirectoryRequest) (*PDFStatsDirectoryResult, error) {
	return s.stats.GetDirectoryStats(req)
}

// GetMaxFileSize returns the maximum file size limit
func (s *Service) GetMaxFileSize() int64 {
	return s.maxFileSize
}

// IsValidPDF performs a quick validation check on a file
func (s *Service) IsValidPDF(filePath string) bool {
	return s.validator.IsValidPDF(filePath)
}

// CountPDFsInDirectory counts the number of valid PDF files in a directory
func (s *Service) CountPDFsInDirectory(directory string) (int, error) {
	return s.search.CountPDFsInDirectory(directory)
}

// FindPDFsInDirectory finds all PDF files in a directory without filtering
func (s *Service) FindPDFsInDirectory(directory string) ([]FileInfo, error) {
	return s.search.FindPDFsInDirectory(directory)
}

// SearchByPattern searches for PDF files matching a specific pattern
func (s *Service) SearchByPattern(directory, pattern string) (*PDFSearchDirectoryResult, error) {
	return s.search.SearchByPattern(directory, pattern)
}

// GetSupportedImageFormats returns a list of supported image formats for asset extraction
func (s *Service) GetSupportedImageFormats() []string {
	return s.assets.GetSupportedFormats()
}

// PDFServerInfo returns comprehensive server information and usage guidance
func (s *Service) PDFServerInfo(req PDFServerInfoRequest, serverName, version,
	defaultDirectory string,
) (*PDFServerInfoResult, error) {
	// Validate the default directory is within bounds
	validatedDir := defaultDirectory
	if err := s.pathValidator.ValidateDirectory(defaultDirectory); err != nil {
		// Use the configured directory if validation fails
		validatedDir = s.pathValidator.GetConfiguredDirectory()
	}

	// Get directory contents with a timeout to prevent hanging
	// Limit to first 100 files for performance
	directoryContents := []FileInfo{}

	// Create a channel to receive results
	resultChan := make(chan []FileInfo, 1)
	errorChan := make(chan error, 1)

	// Run directory search in a goroutine with timeout
	go func() {
		files, err := s.search.FindPDFsInDirectoryLimited(validatedDir, 100)
		if err != nil {
			errorChan <- err
			return
		}
		resultChan <- files
	}()

	// Wait for result with timeout
	select {
	case files := <-resultChan:
		directoryContents = files
	case <-errorChan:
		// Don't fail completely if directory scan fails, just return empty contents
		directoryContents = []FileInfo{}
	case <-time.After(5 * time.Second):
		// Timeout after 5 seconds
		directoryContents = []FileInfo{}
	}

	// Define available tools with detailed information
	availableTools := []ToolInfo{
		{
			Name:        "pdf_read_file",
			Description: "Read and extract text content from a PDF file",
			Usage:       "Use this tool to extract readable text from PDF files. Best for text-based PDFs.",
			Parameters:  "path (required): Full absolute path to the PDF file",
		},
		{
			Name:        "pdf_assets_file",
			Description: "Extract visual assets like images from a PDF file",
			Usage: "Use this tool when a PDF contains scanned images or when pdf_read_file indicates " +
				"'scanned_images' or 'mixed' content type. Extracts JPEG, PNG and other image formats.",
			Parameters: "path (required): Full absolute path to the PDF file",
		},
		{
			Name:        "pdf_validate_file",
			Description: "Validate if a file is a readable PDF",
			Usage:       "Use this tool to check if a file is a valid PDF before attempting to read it.",
			Parameters:  "path (required): Full absolute path to the PDF file",
		},
		{
			Name:        "pdf_stats_file",
			Description: "Get detailed statistics about a PDF file",
			Usage:       "Use this tool to get metadata, page count, file size, and document properties of a PDF.",
			Parameters:  "path (required): Full absolute path to the PDF file",
		},
		{
			Name:        "pdf_search_directory",
			Description: "Search for PDF files in a directory with optional fuzzy search",
			Usage: "Use this tool to find PDF files in the default directory or any specified " +
				"directory. Supports fuzzy search by filename.",
			Parameters: "directory (optional): Directory path to search (uses default if empty), " +
				"query (optional): Search query for fuzzy matching",
		},
		{
			Name:        "pdf_stats_directory",
			Description: "Get statistics about PDF files in a directory",
			Usage: "Use this tool to get an overview of all PDF files in a directory including " +
				"total count, sizes, and file information.",
			Parameters: "directory (optional): Directory path to analyze (uses default if empty)",
		},
	}

	usageGuidance := `PDF MCP Server Usage Guide:

1. START WITH DISCOVERY:
   - Use 'pdf_search_directory' to find available PDF files
   - Use 'pdf_stats_directory' to get an overview of the directory

2. VALIDATE FILES:
   - Use 'pdf_validate_file' to check if a file is readable before processing

3. READ CONTENT:
   - Use 'pdf_read_file' first to extract text content
   - Check the 'content_type' field in the response:
     * "text": PDF contains readable text
     * "scanned_images": PDF contains only scanned images (no extractable text)
     * "mixed": PDF contains both text and images
     * "no_content": PDF appears empty or unreadable

4. EXTRACT IMAGES WHEN NEEDED:
   - Use 'pdf_assets_file' when:
     * content_type is "scanned_images" (document is likely scanned)
     * content_type is "mixed" and you need the images
     * has_images is true and you want to extract visual content

5. GET METADATA:
   - Use 'pdf_stats_file' to get document properties, creation dates, author info, etc.

IMPORTANT NOTES:
- Always use absolute file paths
- The server can handle files up to ` + fmt.Sprintf("%d", s.maxFileSize/(1024*1024)) + `MB
- For scanned documents, pdf_assets_file will extract images but cannot perform OCR
- Some PDFs may have images that cannot be extracted due to format limitations`

	result := &PDFServerInfoResult{
		ServerName:        serverName,
		Version:           version,
		DefaultDirectory:  validatedDir,
		MaxFileSize:       s.maxFileSize,
		AvailableTools:    availableTools,
		DirectoryContents: directoryContents,
		UsageGuidance:     usageGuidance,
		SupportedFormats:  s.GetSupportedImageFormats(),
	}

	return result, nil
}

// ExtractStructured performs structured content extraction with positioning and formatting
func (s *Service) ExtractStructured(req PDFExtractStructuredRequest) (*PDFExtractResult, error) {
	// Validate path is within configured directory
	if err := s.pathValidator.ValidatePath(req.Path); err != nil {
		return nil, fmt.Errorf("security validation failed: %w", err)
	}

	// Convert to internal request format
	extractReq := PDFExtractRequest{
		Path:   req.Path,
		Mode:   req.Mode,
		Config: ExtractConfig(req.Config),
		Query:  s.convertQuery(req.Query),
	}

	if extractReq.Mode == "" {
		extractReq.Mode = "structured"
	}

	return s.extractionService.ExtractStructured(extractReq)
}

// ExtractTables performs table detection and extraction
func (s *Service) ExtractTables(req PDFExtractTablesRequest) (*PDFExtractResult, error) {
	extractReq := PDFExtractRequest{
		Path:   req.Path,
		Mode:   "table",
		Config: ExtractConfig(req.Config),
	}

	return s.extractionService.ExtractTables(extractReq)
}

// ExtractForms performs form field extraction
func (s *Service) ExtractForms(req PDFExtractRequest) (*PDFExtractResult, error) {
	extractReq := PDFExtractRequest{
		Path:   req.Path,
		Mode:   "form",
		Config: req.Config,
	}

	return s.extractionService.ExtractForms(extractReq)
}

// ExtractSemantic performs semantic content grouping
func (s *Service) ExtractSemantic(req PDFExtractSemanticRequest) (*PDFExtractResult, error) {
	extractReq := PDFExtractRequest{
		Path:   req.Path,
		Mode:   "semantic",
		Config: ExtractConfig(req.Config),
	}

	return s.extractionService.ExtractSemantic(extractReq)
}

// ExtractComplete performs comprehensive extraction of all content types
func (s *Service) ExtractComplete(req PDFExtractCompleteRequest) (*PDFExtractResult, error) {
	// Validate path is within configured directory
	if err := s.pathValidator.ValidatePath(req.Path); err != nil {
		return nil, fmt.Errorf("security validation failed: %w", err)
	}

	// Convert to internal request format
	extractReq := PDFExtractRequest{
		Path:   req.Path,
		Mode:   "complete",
		Config: ExtractConfig(req.Config),
	}

	return s.extractionService.ExtractComplete(extractReq)
}

// QueryContent searches extracted content using the provided query
func (s *Service) QueryContent(req PDFQueryContentRequest) (*PDFQueryResult, error) {
	queryReq := PDFQueryRequest(req)

	result, err := s.extractionService.QueryContent(queryReq)
	if err != nil {
		return nil, err
	}

	// Convert back to MCP format
	return &PDFQueryResult{
		FilePath:   result.FilePath,
		Query:      req.Query,
		MatchCount: result.MatchCount,
		Elements:   s.convertElements(result.Elements),
		Summary:    result.Summary,
	}, nil
}

// GetPageInfo returns detailed page information
func (s *Service) GetPageInfo(req PDFGetPageInfoRequest) (*PDFPageInfoResult, error) {
	path := req.Path
	pages, err := s.extractionService.GetPageInfo(path)
	if err != nil {
		return nil, err
	}

	// Convert to MCP format
	mcpPages := make([]PageInfo, len(pages))
	for i, page := range pages {
		mcpPages[i] = PageInfo{
			Number:   page.Number,
			Width:    page.Width,
			Height:   page.Height,
			Rotation: page.Rotation,
			MediaBox: Rectangle{
				X:      page.MediaBox.X,
				Y:      page.MediaBox.Y,
				Width:  page.MediaBox.Width,
				Height: page.MediaBox.Height,
			},
		}
	}

	return &PDFPageInfoResult{
		FilePath: path,
		Pages:    mcpPages,
	}, nil
}

// GetMetadata extracts comprehensive document metadata
func (s *Service) GetMetadata(req PDFGetMetadataRequest) (*PDFMetadataResult, error) {
	path := req.Path
	metadata, err := s.extractionService.GetMetadata(path)
	if err != nil {
		return nil, err
	}

	// Convert to MCP format
	mcpMetadata := DocumentMetadata{
		Title:            metadata.Title,
		Author:           metadata.Author,
		Subject:          metadata.Subject,
		Creator:          metadata.Creator,
		Producer:         metadata.Producer,
		Keywords:         metadata.Keywords,
		PageLayout:       metadata.PageLayout,
		PageMode:         metadata.PageMode,
		Version:          metadata.Version,
		Encrypted:        metadata.Encrypted,
		CustomProperties: metadata.CustomProperties,
	}

	if metadata.CreationDate != "" {
		mcpMetadata.CreationDate = metadata.CreationDate
	}
	if metadata.ModificationDate != "" {
		mcpMetadata.ModificationDate = metadata.ModificationDate
	}

	return &PDFMetadataResult{
		FilePath: path,
		Metadata: mcpMetadata,
	}, nil
}

// Helper methods for type conversion

func (s *Service) convertQuery(q *ContentQuery) *ContentQuery {
	if q == nil {
		return nil
	}

	return q
}

func (s *Service) convertElements(elements []ContentElement) []ContentElement {
	return elements
}

// ValidateConfiguration validates the service configuration
func (s *Service) ValidateConfiguration() error {
	if s.maxFileSize <= 0 {
		return fmt.Errorf("maxFileSize must be greater than 0")
	}

	if s.maxFileSize > 1024*1024*1024 { // 1GB limit
		return fmt.Errorf("maxFileSize cannot exceed 1GB")
	}

	return nil
}
