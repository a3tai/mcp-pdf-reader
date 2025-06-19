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
	Content string `json:"content"`
	Path    string `json:"path"`
	Pages   int    `json:"pages"`
	Size    int64  `json:"size"`
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
