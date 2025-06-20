package pdf

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/a3tai/mcp-pdf-reader/internal/descriptions"
)

// DirectoryCache provides TTL-based caching for directory contents
type DirectoryCache struct {
	entries map[string]*CacheEntry
	ttl     time.Duration
	mu      sync.RWMutex
}

// CacheEntry represents a cached directory scan result
type CacheEntry struct {
	files      []FileInfo
	lastUpdate time.Time
	scanning   bool
	scanMu     sync.Mutex
}

// LazyDirectoryScanner performs efficient directory scanning with limits
type LazyDirectoryScanner struct {
	maxDepth    int
	fileLimit   int
	timeLimit   time.Duration
	skipHidden  bool
	skipSymlink bool
}

// PDFServerInfo provides optimized server info operations
type PDFServerInfo struct {
	cache       *DirectoryCache
	scanner     *LazyDirectoryScanner
	mu          sync.RWMutex
	service     *Service
	initialized bool
}

// ScanResult represents the result of a directory scan
type ScanResult struct {
	Files        []FileInfo
	FromCache    bool
	CacheAge     time.Duration
	ScanTime     time.Duration
	FilesScanned int
	Truncated    bool
}

// NewDirectoryCache creates a new directory cache with specified TTL
func NewDirectoryCache(ttl time.Duration) *DirectoryCache {
	return &DirectoryCache{
		entries: make(map[string]*CacheEntry),
		ttl:     ttl,
	}
}

// Get retrieves cached directory contents if valid
func (c *DirectoryCache) Get(path string) *CacheEntry {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, exists := c.entries[path]
	if !exists {
		return nil
	}

	// Check if cache entry is still valid
	if time.Since(entry.lastUpdate) > c.ttl {
		return nil
	}

	return entry
}

// Set stores directory contents in cache
func (c *DirectoryCache) Set(path string, files []FileInfo) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries[path] = &CacheEntry{
		files:      files,
		lastUpdate: time.Now(),
		scanning:   false,
	}
}

// SetScanning marks a directory as currently being scanned
func (c *DirectoryCache) SetScanning(path string, scanning bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry, exists := c.entries[path]
	if !exists {
		entry = &CacheEntry{
			files:      nil,
			lastUpdate: time.Time{},
			scanning:   scanning,
		}
		c.entries[path] = entry
	} else {
		entry.scanning = scanning
	}
}

// IsScanning checks if a directory is currently being scanned
func (c *DirectoryCache) IsScanning(path string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, exists := c.entries[path]
	return exists && entry.scanning
}

// Clear removes expired entries from cache
func (c *DirectoryCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for path, entry := range c.entries {
		if now.Sub(entry.lastUpdate) > c.ttl {
			delete(c.entries, path)
		}
	}
}

// NewLazyDirectoryScanner creates a new lazy directory scanner
func NewLazyDirectoryScanner(maxDepth, fileLimit int, timeLimit time.Duration) *LazyDirectoryScanner {
	return &LazyDirectoryScanner{
		maxDepth:    maxDepth,
		fileLimit:   fileLimit,
		timeLimit:   timeLimit,
		skipHidden:  true,
		skipSymlink: true,
	}
}

// ScanDirectory performs lazy directory scanning with context cancellation
func (s *LazyDirectoryScanner) ScanDirectory(ctx context.Context, root string) (*ScanResult, error) {
	startTime := time.Now()
	visited := make(map[string]bool)
	var files []FileInfo
	filesScanned := 0
	truncated := false

	err := s.scanRecursive(ctx, root, 0, visited, &files, &filesScanned, &truncated, startTime)

	result := &ScanResult{
		Files:        files,
		FromCache:    false,
		ScanTime:     time.Since(startTime),
		FilesScanned: filesScanned,
		Truncated:    truncated,
	}

	return result, err
}

// scanRecursive performs the actual recursive directory traversal
func (s *LazyDirectoryScanner) scanRecursive(ctx context.Context, path string, depth int,
	visited map[string]bool, files *[]FileInfo, filesScanned *int, truncated *bool, startTime time.Time,
) error {
	// Check context cancellation
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// Check limits
	if s.maxDepth > 0 && depth >= s.maxDepth {
		return nil
	}

	if s.fileLimit > 0 && len(*files) >= s.fileLimit {
		*truncated = true
		return nil
	}

	if s.timeLimit > 0 && time.Since(startTime) > s.timeLimit {
		*truncated = true
		return nil
	}

	// Resolve symlinks and check for cycles
	realPath, err := filepath.EvalSymlinks(path)
	if err != nil {
		// Skip if we can't resolve symlinks
		return nil
	}

	if visited[realPath] {
		return nil // Skip cycles
	}
	visited[realPath] = true

	// Read directory
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil // Skip directories we can't read
	}

	for _, entry := range entries {
		// Check context periodically
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		entryPath := filepath.Join(path, entry.Name())
		*filesScanned++

		// Skip hidden files if configured
		if s.skipHidden && strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		// Skip symlinks if configured
		if s.skipSymlink && entry.Type()&os.ModeSymlink != 0 {
			continue
		}

		if entry.IsDir() {
			// Recurse into subdirectory
			if err := s.scanRecursive(ctx, entryPath, depth+1, visited, files, filesScanned, truncated, startTime); err != nil {
				return err
			}
		} else {
			// Check if it's a PDF file
			if s.isPDFFile(entry.Name()) {
				info, err := entry.Info()
				if err != nil {
					continue
				}

				fileInfo := FileInfo{
					Name:         entry.Name(),
					Path:         entryPath,
					Size:         info.Size(),
					ModifiedTime: info.ModTime().Format("2006-01-02 15:04:05"),
				}

				*files = append(*files, fileInfo)

				// Check file limit after adding
				if s.fileLimit > 0 && len(*files) >= s.fileLimit {
					*truncated = true
					return nil
				}
			}
		}
	}

	return nil
}

// isPDFFile checks if a file is a PDF based on extension
func (s *LazyDirectoryScanner) isPDFFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	return ext == ".pdf"
}

// NewPDFServerInfo creates a new optimized server info handler
func NewPDFServerInfo(service *Service) *PDFServerInfo {
	return &PDFServerInfo{
		cache:   NewDirectoryCache(5 * time.Minute),             // 5-minute cache TTL
		scanner: NewLazyDirectoryScanner(5, 100, 3*time.Second), // max 5 levels, 100 files, 3 second limit
		service: service,
	}
}

// GetServerInfo performs optimized server info retrieval
func (p *PDFServerInfo) GetServerInfo(ctx context.Context, serverName, version, defaultDirectory string) (*PDFServerInfoResult, error) {
	// Validate directory
	validatedDir := defaultDirectory
	if err := p.service.pathValidator.ValidateDirectory(defaultDirectory); err != nil {
		validatedDir = p.service.pathValidator.GetConfiguredDirectory()
	}

	// Try to get from cache first
	var scanResult *ScanResult
	var err error

	if cached := p.cache.Get(validatedDir); cached != nil {
		scanResult = &ScanResult{
			Files:     cached.files,
			FromCache: true,
			CacheAge:  time.Since(cached.lastUpdate),
		}
	} else {
		// Check if already scanning
		if p.cache.IsScanning(validatedDir) {
			// Return empty results if scan is in progress to avoid blocking
			scanResult = &ScanResult{
				Files:     []FileInfo{},
				FromCache: false,
			}
		} else {
			// Mark as scanning and perform scan
			p.cache.SetScanning(validatedDir, true)
			defer p.cache.SetScanning(validatedDir, false)

			// Create context with timeout if none provided
			scanCtx := ctx
			if ctx == context.Background() {
				var cancel context.CancelFunc
				scanCtx, cancel = context.WithTimeout(ctx, 10*time.Second)
				defer cancel()
			}

			scanResult, err = p.scanner.ScanDirectory(scanCtx, validatedDir)
			if err != nil && ctx.Err() == nil {
				// Only return error if not due to context cancellation
				scanResult = &ScanResult{Files: []FileInfo{}}
			}

			// Cache the results
			if scanResult != nil {
				p.cache.Set(validatedDir, scanResult.Files)
			}
		}
	}

	// Build result
	result := &PDFServerInfoResult{
		ServerName:        serverName,
		Version:           version,
		DefaultDirectory:  validatedDir,
		MaxFileSize:       p.service.maxFileSize,
		AvailableTools:    p.getAvailableTools(),
		DirectoryContents: scanResult.Files,
		UsageGuidance:     p.getUsageGuidance(),
		SupportedFormats:  p.service.GetSupportedImageFormats(),
	}

	return result, nil
}

// getAvailableTools returns the list of available tools
func (p *PDFServerInfo) getAvailableTools() []ToolInfo {
	return []ToolInfo{
		{
			Name:        "pdf_read_file",
			Description: descriptions.GetToolDescription("pdf_read_file"),
			Usage:       "Use this tool to extract readable text from PDF files. Best for text-based PDFs.",
			Parameters:  "path (required): Full path to the PDF file (supports both absolute and relative paths)",
		},
		{
			Name:        "pdf_assets_file",
			Description: descriptions.GetToolDescription("pdf_assets_file"),
			Usage: "Use this tool when a PDF contains scanned images or when pdf_read_file indicates " +
				"'scanned_images' or 'mixed' content type. Extracts JPEG, PNG and other image formats.",
			Parameters: "path (required): Full path to the PDF file (supports both absolute and relative paths)",
		},
		{
			Name:        "pdf_validate_file",
			Description: descriptions.GetToolDescription("pdf_validate_file"),
			Usage:       "Use this tool to check if a file is a valid PDF before attempting to read it.",
			Parameters:  "path (required): Full path to the PDF file (supports both absolute and relative paths)",
		},
		{
			Name:        "pdf_stats_file",
			Description: descriptions.GetToolDescription("pdf_stats_file"),
			Usage:       "Use this tool to get metadata, page count, file size, and document properties of a PDF.",
			Parameters:  "path (required): Full path to the PDF file (supports both absolute and relative paths)",
		},
		{
			Name:        "pdf_search_directory",
			Description: descriptions.GetToolDescription("pdf_search_directory"),
			Usage: "Use this tool to find PDF files in the default directory or any specified " +
				"directory. Supports fuzzy search by filename.",
			Parameters: "directory (optional): Directory path to search (uses current directory if empty, supports relative paths), " +
				"query (optional): Search query for fuzzy matching",
		},
		{
			Name:        "pdf_stats_directory",
			Description: descriptions.GetToolDescription("pdf_stats_directory"),
			Usage: "Use this tool to get an overview of all PDF files in a directory including " +
				"total count, sizes, and file information.",
			Parameters: "directory (optional): Directory path to analyze (uses current directory if empty, supports relative paths)",
		},
		{
			Name:        "pdf_server_info",
			Description: descriptions.GetToolDescription("pdf_server_info"),
			Usage:       "Use this tool to get comprehensive server information and available capabilities.",
			Parameters:  "No parameters required",
		},
		{
			Name:        "pdf_get_page_info",
			Description: descriptions.GetToolDescription("pdf_get_page_info"),
			Usage:       "Use this tool to get page dimensions, rotation, media box, and other page properties.",
			Parameters:  "path (required): Full path to the PDF file (supports both absolute and relative paths)",
		},
		{
			Name:        "pdf_get_metadata",
			Description: descriptions.GetToolDescription("pdf_get_metadata"),
			Usage:       "Use this tool to get document metadata, creation dates, author info, and document properties.",
			Parameters:  "path (required): Full path to the PDF file (supports both absolute and relative paths)",
		},
		{
			Name:        "pdf_extract_structured",
			Description: descriptions.GetToolDescription("pdf_extract_structured"),
			Usage:       "Use this tool for advanced text extraction with layout and formatting details.",
			Parameters:  "path (required): Full path to the PDF file (supports both absolute and relative paths), mode (optional): extraction mode",
		},
		{
			Name:        "pdf_extract_complete",
			Description: descriptions.GetToolDescription("pdf_extract_complete"),
			Usage:       "Use this tool for complete document analysis and content extraction.",
			Parameters:  "path (required): Full path to the PDF file (supports both absolute and relative paths)",
		},
		{
			Name:        "pdf_extract_tables",
			Description: descriptions.GetToolDescription("pdf_extract_tables"),
			Usage:       "Use this tool to extract table data with proper row/column structure.",
			Parameters:  "path (required): Full path to the PDF file (supports both absolute and relative paths)",
		},
		{
			Name:        "pdf_extract_forms",
			Description: descriptions.GetToolDescription("pdf_extract_forms"),
			Usage:       "Use this tool to extract interactive form elements from PDF files.",
			Parameters:  "path (required): Full path to the PDF file (supports both absolute and relative paths)",
		},
		{
			Name:        "pdf_extract_semantic",
			Description: descriptions.GetToolDescription("pdf_extract_semantic"),
			Usage:       "Use this tool for intelligent content extraction with semantic understanding.",
			Parameters:  "path (required): Full path to the PDF file (supports both absolute and relative paths)",
		},
		{
			Name:        "pdf_query_content",
			Description: descriptions.GetToolDescription("pdf_query_content"),
			Usage:       "Use this tool to search within PDF content using structured queries.",
			Parameters:  "path (required): Full path to the PDF file (supports both absolute and relative paths), query (required): search criteria",
		},
		{
			Name:        "pdf_analyze_document",
			Description: descriptions.GetToolDescription("pdf_analyze_document"),
			Usage:       "Use this tool for deep document analysis and classification.",
			Parameters:  "path (required): Full path to the PDF file (supports both absolute and relative paths)",
		},
	}
}

// getUsageGuidance returns comprehensive usage guidance
func (p *PDFServerInfo) getUsageGuidance() string {
	maxFileSizeMB := p.service.maxFileSize / (1024 * 1024)

	return fmt.Sprintf(`PDF MCP Server Usage Guide:

1. START WITH DISCOVERY:
   - Use 'pdf_search_directory' to find available PDF files
   - Use 'pdf_stats_directory' to get an overview of the directory
   - Use 'pdf_server_info' to get server capabilities and current directory contents

2. VALIDATE FILES:
   - Use 'pdf_validate_file' to check if a file is readable before processing

3. READ CONTENT:
   - Use 'pdf_read_file' first to extract text content
   - Check the 'content_type' field in the response:
     * "text": PDF contains readable text
     * "scanned_images": PDF contains only scanned images (no extractable text)
     * "mixed": PDF contains both text and images
     * "no_content": PDF appears empty or unreadable

4. ADVANCED EXTRACTION:
   - Use 'pdf_extract_structured' for layout-aware text extraction
   - Use 'pdf_extract_complete' for comprehensive content analysis
   - Use 'pdf_extract_tables' for tabular data extraction
   - Use 'pdf_extract_forms' for interactive form elements
   - Use 'pdf_extract_semantic' for intelligent content grouping

5. EXTRACT IMAGES WHEN NEEDED:
   - Use 'pdf_assets_file' when:
     * content_type is "scanned_images" (document is likely scanned)
     * content_type is "mixed" and you need the images
     * has_images is true and you want to extract visual content

6. GET METADATA AND ANALYSIS:
   - Use 'pdf_stats_file' to get document properties, creation dates, author info
   - Use 'pdf_get_page_info' to get page dimensions and layout properties
   - Use 'pdf_get_metadata' for comprehensive document metadata
   - Use 'pdf_analyze_document' for deep document analysis and classification

7. SEARCH AND QUERY:
   - Use 'pdf_query_content' to search within document content
   - Use 'pdf_search_directory' with fuzzy search to find specific files

PERFORMANCE OPTIMIZATIONS:
- Server info results are cached for 5 minutes to improve response times
- Directory scanning is limited to 100 files and 3 seconds to prevent timeouts
- Lazy loading ensures fast initial responses
- Context-aware operations support cancellation

IMPORTANT NOTES:
- Always use absolute file paths
- The server can handle files up to %dMB
- For scanned documents, pdf_assets_file will extract images but cannot perform OCR
- Some PDFs may have images that cannot be extracted due to format limitations
- Use pdf_validate_file before processing to avoid errors
- Large directories may have truncated results - use pdf_search_directory for comprehensive searches`, maxFileSizeMB)
}

// ClearCache clears expired cache entries
func (p *PDFServerInfo) ClearCache() {
	p.cache.Clear()
}

// GetCacheStats returns cache statistics
func (p *PDFServerInfo) GetCacheStats() map[string]interface{} {
	p.cache.mu.RLock()
	defer p.cache.mu.RUnlock()

	stats := make(map[string]interface{})
	stats["total_entries"] = len(p.cache.entries)

	validEntries := 0
	for _, entry := range p.cache.entries {
		if time.Since(entry.lastUpdate) <= p.cache.ttl {
			validEntries++
		}
	}

	stats["valid_entries"] = validEntries
	stats["cache_ttl_minutes"] = p.cache.ttl.Minutes()

	return stats
}
