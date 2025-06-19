package pdf

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ledongthuc/pdf"
)

// Stats handles PDF statistics operations
type Stats struct {
	maxFileSize int64
	validator   *Validator
}

// NewStats creates a new PDF stats analyzer with the specified constraints
func NewStats(maxFileSize int64) *Stats {
	return &Stats{
		maxFileSize: maxFileSize,
		validator:   NewValidator(maxFileSize),
	}
}

// GetFileStats returns detailed statistics about a single PDF file
func (s *Stats) GetFileStats(req PDFStatsFileRequest) (*PDFStatsFileResult, error) {
	if req.Path == "" {
		return nil, fmt.Errorf("path cannot be empty")
	}

	// Check if file exists and get basic info
	fileInfo, err := os.Stat(req.Path)
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("file does not exist: %s", req.Path)
	}
	if err != nil {
		return nil, fmt.Errorf("cannot access file: %w", err)
	}

	// Validate file
	if err := s.validator.ValidateFileInfo(req.Path, fileInfo); err != nil {
		return nil, err
	}

	// Open and parse PDF for metadata
	f, r, err := pdf.Open(req.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to open PDF: %w", err)
	}
	defer f.Close()

	result := &PDFStatsFileResult{
		Path:         req.Path,
		Size:         fileInfo.Size(),
		Pages:        r.NumPage(),
		ModifiedDate: fileInfo.ModTime().Format("2006-01-02 15:04:05"),
	}

	// Extract metadata if available
	s.extractMetadata(r, result)

	return result, nil
}

// GetDirectoryStats returns statistics about PDF files in a directory
//
//nolint:gocognit // Function complexity is necessary for comprehensive directory analysis
func (s *Stats) GetDirectoryStats(req PDFStatsDirectoryRequest) (*PDFStatsDirectoryResult, error) {
	directory := req.Directory
	if directory == "" {
		return nil, fmt.Errorf("directory cannot be empty")
	}

	// Check if directory exists
	if _, err := os.Stat(directory); os.IsNotExist(err) {
		return nil, fmt.Errorf("directory does not exist: %s", directory)
	}

	var totalFiles int
	var totalSize int64
	var largestFile int64
	var largestFileName string
	var smallestFile int64 = int64(^uint64(0) >> 1) // Max int64
	var smallestFileName string

	err := filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil //nolint:nilerr // Continue despite errors
		}

		if info.IsDir() {
			return nil
		}

		if strings.HasSuffix(strings.ToLower(info.Name()), ".pdf") {
			// Quick validation without opening the file
			if s.validator.ValidateFileInfo(path, info) == nil {
				totalFiles++
				totalSize += info.Size()

				if info.Size() > largestFile {
					largestFile = info.Size()
					largestFileName = info.Name()
				}

				if info.Size() < smallestFile {
					smallestFile = info.Size()
					smallestFileName = info.Name()
				}
			}
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("error walking directory: %w", err)
	}

	var averageSize int64
	if totalFiles > 0 {
		averageSize = totalSize / int64(totalFiles)
	}

	// If no files found, reset smallest file size
	if totalFiles == 0 {
		smallestFile = 0
	}

	result := &PDFStatsDirectoryResult{
		Directory:        directory,
		TotalFiles:       totalFiles,
		TotalSize:        totalSize,
		LargestFileSize:  largestFile,
		LargestFileName:  largestFileName,
		SmallestFileSize: smallestFile,
		SmallestFileName: smallestFileName,
		AverageFileSize:  averageSize,
	}

	return result, nil
}

// extractMetadata safely extracts metadata from PDF reader
func (s *Stats) extractMetadata(r *pdf.Reader, result *PDFStatsFileResult) {
	// Safely extract metadata using the PDF library's API
	// The ledongthuc/pdf library requires careful handling of Value types

	defer func() {
		// Recover from any panics during metadata extraction
		if recover() != nil {
			// Metadata extraction failed, but we can continue with basic stats
		}
	}()

	// Try to get document info
	trailer := r.Trailer()
	if trailer.IsNull() {
		return
	}

	info := trailer.Key("Info")
	if info.IsNull() {
		return
	}

	// Extract title
	if title := info.Key("Title"); !title.IsNull() {
		if titleStr := title.String(); titleStr != "" {
			result.Title = strings.TrimSpace(titleStr)
		}
	}

	// Extract author
	if author := info.Key("Author"); !author.IsNull() {
		if authorStr := author.String(); authorStr != "" {
			result.Author = strings.TrimSpace(authorStr)
		}
	}

	// Extract subject
	if subject := info.Key("Subject"); !subject.IsNull() {
		if subjectStr := subject.String(); subjectStr != "" {
			result.Subject = strings.TrimSpace(subjectStr)
		}
	}

	// Extract producer
	if producer := info.Key("Producer"); !producer.IsNull() {
		if producerStr := producer.String(); producerStr != "" {
			result.Producer = strings.TrimSpace(producerStr)
		}
	}

	// Extract creation date
	if creationDate := info.Key("CreationDate"); !creationDate.IsNull() {
		if dateStr := creationDate.String(); dateStr != "" {
			result.CreatedDate = strings.TrimSpace(dateStr)
		}
	}
}
