package pdf

import (
	"fmt"
)

// Service handles PDF file operations by orchestrating various PDF components
type Service struct {
	maxFileSize int64
	reader      *Reader
	validator   *Validator
	stats       *Stats
	assets      *Assets
	search      *Search
}

// NewService creates a new PDF service with all components
func NewService(maxFileSize int64) *Service {
	return &Service{
		maxFileSize: maxFileSize,
		reader:      NewReader(maxFileSize),
		validator:   NewValidator(maxFileSize),
		stats:       NewStats(maxFileSize),
		assets:      NewAssets(maxFileSize),
		search:      NewSearch(maxFileSize),
	}
}

// PDFReadFile reads the content of a PDF file
func (s *Service) PDFReadFile(req PDFReadFileRequest) (*PDFReadFileResult, error) {
	return s.reader.ReadFile(req)
}

// PDFAssetsFile extracts visual assets like images from a PDF file
func (s *Service) PDFAssetsFile(req PDFAssetsFileRequest) (*PDFAssetsFileResult, error) {
	return s.assets.ExtractAssets(req)
}

// PDFValidateFile performs validation on a PDF file
func (s *Service) PDFValidateFile(req PDFValidateFileRequest) (*PDFValidateFileResult, error) {
	return s.validator.ValidateFile(req)
}

// PDFStatsFile returns detailed statistics about a single PDF file
func (s *Service) PDFStatsFile(req PDFStatsFileRequest) (*PDFStatsFileResult, error) {
	return s.stats.GetFileStats(req)
}

// PDFSearchDirectory searches for PDF files in the specified directory
func (s *Service) PDFSearchDirectory(req PDFSearchDirectoryRequest) (*PDFSearchDirectoryResult, error) {
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
func (s *Service) PDFServerInfo(req PDFServerInfoRequest, serverName, version, defaultDirectory string) (*PDFServerInfoResult, error) {
	// Get directory contents
	directoryContents, err := s.search.FindPDFsInDirectory(defaultDirectory)
	if err != nil {
		// Don't fail completely if directory scan fails, just return empty contents
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
			Usage:       "Use this tool when a PDF contains scanned images or when pdf_read_file indicates 'scanned_images' or 'mixed' content type. Extracts JPEG, PNG and other image formats.",
			Parameters:  "path (required): Full absolute path to the PDF file",
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
			Usage:       "Use this tool to find PDF files in the default directory or any specified directory. Supports fuzzy search by filename.",
			Parameters:  "directory (optional): Directory path to search (uses default if empty), query (optional): Search query for fuzzy matching",
		},
		{
			Name:        "pdf_stats_directory",
			Description: "Get statistics about PDF files in a directory",
			Usage:       "Use this tool to get an overview of all PDF files in a directory including total count, sizes, and file information.",
			Parameters:  "directory (optional): Directory path to analyze (uses default if empty)",
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
		DefaultDirectory:  defaultDirectory,
		MaxFileSize:       s.maxFileSize,
		AvailableTools:    availableTools,
		DirectoryContents: directoryContents,
		UsageGuidance:     usageGuidance,
		SupportedFormats:  s.GetSupportedImageFormats(),
	}

	return result, nil
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
