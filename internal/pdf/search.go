package pdf

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Search handles PDF search and discovery operations
type Search struct {
	maxFileSize int64
	validator   *Validator
}

// NewSearch creates a new PDF search handler with the specified constraints
func NewSearch(maxFileSize int64) *Search {
	return &Search{
		maxFileSize: maxFileSize,
		validator:   NewValidator(maxFileSize),
	}
}

// isPathWithinDirectory checks if a path is within the specified directory
func (s *Search) isPathWithinDirectory(path, directory string) (bool, error) {
	// Resolve both paths to absolute paths
	absPath, err := filepath.Abs(path)
	if err != nil {
		return false, fmt.Errorf("failed to resolve path: %w", err)
	}

	absDir, err := filepath.Abs(directory)
	if err != nil {
		return false, fmt.Errorf("failed to resolve directory: %w", err)
	}

	// Evaluate any symlinks to get the real path
	realPath, err := filepath.EvalSymlinks(absPath)
	if err != nil {
		// If the file doesn't exist yet, just use the absolute path
		if !os.IsNotExist(err) {
			return false, fmt.Errorf("failed to evaluate symlinks: %w", err)
		}
		realPath = absPath
	}

	realDir, err := filepath.EvalSymlinks(absDir)
	if err != nil {
		return false, fmt.Errorf("failed to evaluate directory symlinks: %w", err)
	}

	// Ensure both paths use consistent separators
	realPath = filepath.Clean(realPath)
	realDir = filepath.Clean(realDir)

	// Check if the path starts with the directory path
	// Add a separator to the directory to ensure exact match
	if !strings.HasSuffix(realDir, string(filepath.Separator)) {
		realDir += string(filepath.Separator)
	}

	return strings.HasPrefix(realPath, realDir) || realPath == strings.TrimSuffix(realDir, string(filepath.Separator)), nil
}

// SearchDirectory searches for PDF files in the specified directory
func (s *Search) SearchDirectory(req PDFSearchDirectoryRequest) (*PDFSearchDirectoryResult, error) {
	if req.Directory == "" {
		return nil, fmt.Errorf("directory cannot be empty")
	}

	// Check if directory exists
	if _, err := os.Stat(req.Directory); os.IsNotExist(err) {
		return nil, fmt.Errorf("directory does not exist: %s", req.Directory)
	}

	var pdfFiles []FileInfo
	query := strings.ToLower(strings.TrimSpace(req.Query))

	// Resolve the search directory to prevent traversal
	absDirectory, err := filepath.Abs(req.Directory)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve directory path: %w", err)
	}

	err = filepath.Walk(absDirectory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			// Continue walking even if we encounter an error with a specific file
			return nil //nolint:nilerr // Intentionally continue on file errors
		}

		// Security check: ensure path is within the configured directory
		withinDir, err := s.isPathWithinDirectory(path, absDirectory)
		if err != nil || !withinDir {
			// Skip files outside the directory or with path errors
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Check if it's a PDF file
		if !s.isPDFFile(info.Name()) {
			return nil
		}

		// Quick validation without opening the file
		if err := s.validator.ValidateFileInfo(path, info); err != nil {
			// Skip invalid files but continue processing
			return nil //nolint:nilerr // Intentionally continue on validation errors
		}

		// Apply query filter if provided
		if query != "" && !s.matchesQuery(info.Name(), query) {
			return nil
		}

		// Create file info
		fileInfo := FileInfo{
			Path:         path,
			Name:         info.Name(),
			Size:         info.Size(),
			ModifiedTime: info.ModTime().Format("2006-01-02 15:04:05"),
		}

		pdfFiles = append(pdfFiles, fileInfo)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("error walking directory: %w", err)
	}

	result := &PDFSearchDirectoryResult{
		Files:       pdfFiles,
		TotalCount:  len(pdfFiles),
		Directory:   absDirectory,
		SearchQuery: req.Query,
	}

	return result, nil
}

// FindPDFsInDirectory finds all PDF files in a directory without query filtering
func (s *Search) FindPDFsInDirectory(directory string) ([]FileInfo, error) {
	req := PDFSearchDirectoryRequest{
		Directory: directory,
		Query:     "", // No query filter
	}

	result, err := s.SearchDirectory(req)
	if err != nil {
		return nil, err
	}

	return result.Files, nil
}

// FindPDFsInDirectoryLimited finds PDF files in a directory with a limit on the number of results
func (s *Search) FindPDFsInDirectoryLimited(directory string, limit int) ([]FileInfo, error) {
	if directory == "" {
		return nil, fmt.Errorf("directory cannot be empty")
	}

	// Check if directory exists
	if _, err := os.Stat(directory); os.IsNotExist(err) {
		return nil, fmt.Errorf("directory does not exist: %s", directory)
	}

	var pdfFiles []FileInfo
	foundCount := 0

	// Resolve the search directory to prevent traversal
	absDirectory, err := filepath.Abs(directory)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve directory path: %w", err)
	}

	// Use filepath.WalkDir for better performance (Go 1.16+)
	err = filepath.WalkDir(absDirectory, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			// Continue walking even if we encounter an error with a specific file
			return nil
		}

		// Security check: ensure path is within the configured directory
		withinDir, err := s.isPathWithinDirectory(path, absDirectory)
		if err != nil || !withinDir {
			// Skip files/dirs outside the directory or with path errors
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip directories
		if d.IsDir() {
			// Skip hidden directories to improve performance
			if strings.HasPrefix(d.Name(), ".") && path != absDirectory {
				return filepath.SkipDir
			}
			return nil
		}

		// Stop if we've reached the limit
		if limit > 0 && foundCount >= limit {
			return filepath.SkipAll
		}

		// Check if it's a PDF file
		if !s.isPDFFile(d.Name()) {
			return nil
		}

		// Get file info
		info, err := d.Info()
		if err != nil {
			return nil
		}

		// Quick validation without opening the file
		if err := s.validator.ValidateFileInfo(path, info); err != nil {
			// Skip invalid files but continue processing
			return nil
		}

		// Create file info
		fileInfo := FileInfo{
			Path:         path,
			Name:         info.Name(),
			Size:         info.Size(),
			ModifiedTime: info.ModTime().Format("2006-01-02 15:04:05"),
		}

		pdfFiles = append(pdfFiles, fileInfo)
		foundCount++
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("error walking directory: %w", err)
	}

	return pdfFiles, nil
}

// CountPDFsInDirectory counts the number of valid PDF files in a directory
func (s *Search) CountPDFsInDirectory(directory string) (int, error) {
	files, err := s.FindPDFsInDirectory(directory)
	if err != nil {
		return 0, err
	}

	return len(files), nil
}

// isPDFFile checks if a file has a PDF extension
func (s *Search) isPDFFile(filename string) bool {
	return strings.HasSuffix(strings.ToLower(filename), ".pdf")
}

// matchesQuery performs fuzzy matching on the filename
func (s *Search) matchesQuery(filename, query string) bool {
	if query == "" {
		return true
	}

	fileName := strings.ToLower(filename)

	// Exact substring match
	if strings.Contains(fileName, query) {
		return true
	}

	// Remove extension for name-only matching
	nameWithoutExt := strings.TrimSuffix(fileName, ".pdf")
	if strings.Contains(nameWithoutExt, query) {
		return true
	}

	// Word-based matching (split by common separators)
	words := s.splitIntoWords(nameWithoutExt)
	queryWords := s.splitIntoWords(query)

	// Check if all query words are found in filename words
	for _, queryWord := range queryWords {
		found := false
		for _, word := range words {
			if strings.Contains(word, queryWord) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}

// splitIntoWords splits a string into words using common separators
func (s *Search) splitIntoWords(text string) []string {
	// Split by common separators
	separators := []string{" ", "_", "-", ".", "(", ")", "[", "]"}

	words := []string{text}
	for _, sep := range separators {
		var newWords []string
		for _, word := range words {
			parts := strings.Split(word, sep)
			for _, part := range parts {
				if part != "" {
					newWords = append(newWords, strings.ToLower(part))
				}
			}
		}
		words = newWords
	}

	return words
}

// SearchByPattern searches for PDF files matching a specific pattern
func (s *Search) SearchByPattern(directory, pattern string) (*PDFSearchDirectoryResult, error) {
	if pattern == "" {
		return s.SearchDirectory(PDFSearchDirectoryRequest{Directory: directory})
	}

	var pdfFiles []FileInfo

	// Resolve the search directory to prevent traversal
	absDirectory, err := filepath.Abs(directory)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve directory path: %w", err)
	}

	err = filepath.Walk(absDirectory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil //nolint:nilerr // Intentionally continue on file errors
		}

		// Security check: ensure path is within the configured directory
		withinDir, err := s.isPathWithinDirectory(path, absDirectory)
		if err != nil || !withinDir {
			// Skip files/dirs outside the directory or with path errors
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if info.IsDir() {
			return nil
		}

		if !s.isPDFFile(info.Name()) {
			return nil
		}

		// Validate file
		if err := s.validator.ValidateFileInfo(path, info); err != nil {
			return nil //nolint:nilerr // Intentionally continue on validation errors
		}

		// Check if filename matches pattern
		matched, err := filepath.Match(pattern, info.Name())
		if err != nil {
			return nil //nolint:nilerr // Continue on pattern error
		}

		if matched {
			fileInfo := FileInfo{
				Path:         path,
				Name:         info.Name(),
				Size:         info.Size(),
				ModifiedTime: info.ModTime().Format("2006-01-02 15:04:05"),
			}
			pdfFiles = append(pdfFiles, fileInfo)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("error walking directory: %w", err)
	}

	result := &PDFSearchDirectoryResult{
		Files:       pdfFiles,
		TotalCount:  len(pdfFiles),
		Directory:   absDirectory,
		SearchQuery: pattern,
	}

	return result, nil
}
