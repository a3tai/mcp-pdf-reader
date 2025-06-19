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
