package pdf

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/a3tai/mcp-pdf-reader/internal/pdf/pagerange"
	"github.com/a3tai/mcp-pdf-reader/internal/pdf/security"
	"github.com/a3tai/mcp-pdf-reader/internal/pdf/streaming"
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
	serverInfo        *PDFServerInfo
	streamingEnabled  bool
}

// NewService creates a new PDF service with all components
func NewService(maxFileSize int64, configuredDirectory string) (*Service, error) {
	pathValidator, err := security.NewPathValidator(configuredDirectory)
	if err != nil {
		return nil, fmt.Errorf("failed to create path validator: %w", err)
	}

	service := &Service{
		maxFileSize:       maxFileSize,
		reader:            NewReader(maxFileSize),
		validator:         NewValidator(maxFileSize),
		stats:             NewStats(maxFileSize),
		assets:            NewAssets(maxFileSize),
		search:            NewSearch(maxFileSize),
		extractionService: NewExtractionService(maxFileSize),
		pathValidator:     pathValidator,
	}

	// Initialize server info with self-reference
	service.serverInfo = NewPDFServerInfo(service)
	service.streamingEnabled = true

	return service, nil
}

// PDFReadFile reads the content of a PDF file
func (s *Service) PDFReadFile(req PDFReadFileRequest) (*PDFReadFileResult, error) {
	// Normalize relative paths
	normalizedPath, err := s.pathValidator.NormalizePath(req.Path)
	if err != nil {
		return nil, fmt.Errorf("security validation failed: %w", err)
	}

	// Update request with normalized path
	req.Path = normalizedPath
	return s.reader.ReadFile(req)
}

// PDFAssetsFile extracts visual assets like images from a PDF file
func (s *Service) PDFAssetsFile(req PDFAssetsFileRequest) (*PDFAssetsFileResult, error) {
	// Normalize relative paths
	normalizedPath, err := s.pathValidator.NormalizePath(req.Path)
	if err != nil {
		return nil, fmt.Errorf("security validation failed: %w", err)
	}

	// Update request with normalized path
	req.Path = normalizedPath
	return s.assets.ExtractAssets(req)
}

// PDFValidateFile performs validation on a PDF file
func (s *Service) PDFValidateFile(req PDFValidateFileRequest) (*PDFValidateFileResult, error) {
	// Normalize relative paths
	normalizedPath, err := s.pathValidator.NormalizePath(req.Path)
	if err != nil {
		return nil, fmt.Errorf("security validation failed: %w", err)
	}

	// Update request with normalized path
	req.Path = normalizedPath
	return s.validator.ValidateFile(req)
}

// PDFStatsFile returns detailed statistics about a single PDF file
func (s *Service) PDFStatsFile(req PDFStatsFileRequest) (*PDFStatsFileResult, error) {
	// Normalize relative paths
	normalizedPath, err := s.pathValidator.NormalizePath(req.Path)
	if err != nil {
		return nil, fmt.Errorf("security validation failed: %w", err)
	}

	// Update request with normalized path
	req.Path = normalizedPath
	return s.stats.GetFileStats(req)
}

// PDFSearchDirectory searches for PDF files in a directory
func (s *Service) PDFSearchDirectory(req PDFSearchDirectoryRequest) (*PDFSearchDirectoryResult, error) {
	// If no directory specified, use configured directory
	if req.Directory == "" {
		req.Directory = s.pathValidator.GetConfiguredDirectory()
	} else {
		// Normalize relative directory paths
		normalizedDir, err := s.pathValidator.NormalizePath(req.Directory)
		if err != nil {
			return nil, fmt.Errorf("security validation failed: %w", err)
		}
		req.Directory = normalizedDir
	}

	// Validate directory is within configured bounds
	if err := s.pathValidator.ValidateDirectory(req.Directory); err != nil {
		return nil, fmt.Errorf("security validation failed: %w", err)
	}

	return s.search.SearchDirectory(req)
}

// PDFStatsDirectory returns statistics about PDF files in a directory
func (s *Service) PDFStatsDirectory(req PDFStatsDirectoryRequest) (*PDFStatsDirectoryResult, error) {
	// If no directory specified, use configured directory
	if req.Directory == "" {
		req.Directory = s.pathValidator.GetConfiguredDirectory()
	} else {
		// Normalize relative directory paths
		normalizedDir, err := s.pathValidator.NormalizePath(req.Directory)
		if err != nil {
			return nil, fmt.Errorf("security validation failed: %w", err)
		}
		req.Directory = normalizedDir
	}

	// Validate directory is within configured bounds
	if err := s.pathValidator.ValidateDirectory(req.Directory); err != nil {
		return nil, fmt.Errorf("security validation failed: %w", err)
	}

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
func (s *Service) PDFServerInfo(ctx context.Context, req PDFServerInfoRequest, serverName, version,
	defaultDirectory string,
) (*PDFServerInfoResult, error) {
	// Use optimized server info with context
	return s.serverInfo.GetServerInfo(ctx, serverName, version, defaultDirectory)
}

// ExtractStructured performs structured content extraction with positioning and formatting
func (s *Service) ExtractStructured(req PDFExtractStructuredRequest) (*PDFExtractResult, error) {
	// Normalize relative paths
	normalizedPath, err := s.pathValidator.NormalizePath(req.Path)
	if err != nil {
		return nil, fmt.Errorf("security validation failed: %w", err)
	}

	// Convert to internal request format
	extractReq := PDFExtractRequest{
		Path:   normalizedPath,
		Mode:   req.Mode,
		Config: ExtractConfig(req.Config),
	}

	if extractReq.Mode == "" {
		extractReq.Mode = "structured"
	}

	return s.extractionService.ExtractStructured(extractReq)
}

// ExtractTables performs table detection and extraction
// ExtractTables performs table extraction
func (s *Service) ExtractTables(req PDFExtractTablesRequest) (*PDFExtractResult, error) {
	// Normalize relative paths
	normalizedPath, err := s.pathValidator.NormalizePath(req.Path)
	if err != nil {
		return nil, fmt.Errorf("security validation failed: %w", err)
	}

	extractReq := PDFExtractRequest{
		Path:   normalizedPath,
		Mode:   "table",
		Config: ExtractConfig(req.Config),
	}

	return s.extractionService.ExtractTables(extractReq)
}

// ExtractForms performs form extraction
func (s *Service) ExtractForms(req PDFExtractRequest) (*PDFExtractResult, error) {
	// Normalize relative paths
	normalizedPath, err := s.pathValidator.NormalizePath(req.Path)
	if err != nil {
		return nil, fmt.Errorf("security validation failed: %w", err)
	}

	extractReq := PDFExtractRequest{
		Path:   normalizedPath,
		Mode:   "form",
		Config: req.Config,
	}

	return s.extractionService.ExtractForms(extractReq)
}

// ExtractSemantic performs semantic content grouping
// ExtractSemantic performs semantic extraction
func (s *Service) ExtractSemantic(req PDFExtractSemanticRequest) (*PDFExtractResult, error) {
	// Normalize relative paths
	normalizedPath, err := s.pathValidator.NormalizePath(req.Path)
	if err != nil {
		return nil, fmt.Errorf("security validation failed: %w", err)
	}

	extractReq := PDFExtractRequest{
		Path:   normalizedPath,
		Mode:   "semantic",
		Config: ExtractConfig(req.Config),
	}

	return s.extractionService.ExtractSemantic(extractReq)
}

// ExtractComplete performs comprehensive extraction of all content types
func (s *Service) ExtractComplete(req PDFExtractCompleteRequest) (*PDFExtractResult, error) {
	// Normalize relative paths
	normalizedPath, err := s.pathValidator.NormalizePath(req.Path)
	if err != nil {
		return nil, fmt.Errorf("security validation failed: %w", err)
	}

	// Convert to internal request format
	extractReq := PDFExtractRequest{
		Path:   normalizedPath,
		Mode:   "complete",
		Config: ExtractConfig(req.Config),
	}

	return s.extractionService.ExtractComplete(extractReq)
}

// QueryContent searches extracted content using the provided query
func (s *Service) QueryContent(req PDFQueryContentRequest) (*PDFQueryResult, error) {
	// Normalize relative paths
	normalizedPath, err := s.pathValidator.NormalizePath(req.Path)
	if err != nil {
		return nil, fmt.Errorf("security validation failed: %w", err)
	}

	// Update request with normalized path
	req.Path = normalizedPath
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
	// Normalize relative paths
	normalizedPath, err := s.pathValidator.NormalizePath(req.Path)
	if err != nil {
		return nil, fmt.Errorf("security validation failed: %w", err)
	}

	pages, err := s.extractionService.GetPageInfo(normalizedPath)
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
		FilePath: normalizedPath,
		Pages:    mcpPages,
	}, nil
}

// GetMetadata extracts comprehensive document metadata
func (s *Service) GetMetadata(req PDFGetMetadataRequest) (*PDFMetadataResult, error) {
	// Normalize relative paths
	normalizedPath, err := s.pathValidator.NormalizePath(req.Path)
	if err != nil {
		return nil, fmt.Errorf("security validation failed: %w", err)
	}

	metadata, err := s.extractionService.GetMetadata(normalizedPath)
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
		FilePath: normalizedPath,
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

// Streaming Processing Methods

// StreamProcessFile processes a PDF file using streaming for large file support
func (s *Service) StreamProcessFile(req PDFStreamProcessRequest) (*PDFStreamProcessResult, error) {
	if !s.streamingEnabled {
		return nil, fmt.Errorf("streaming processing is not enabled")
	}

	// Normalize relative paths
	normalizedPath, err := s.pathValidator.NormalizePath(req.Path)
	if err != nil {
		return nil, fmt.Errorf("security validation failed: %w", err)
	}

	// Check if file exists and get info
	fileInfo, err := os.Stat(normalizedPath)
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("file does not exist: %s", normalizedPath)
	}
	if err != nil {
		return nil, fmt.Errorf("cannot access file: %w", err)
	}

	// Open file for streaming
	file, err := os.Open(normalizedPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Configure streaming extractor
	config := s.getStreamingConfig(req.Config)
	config.ExtractText = req.ExtractText
	config.ExtractImages = req.ExtractImages
	config.ExtractForms = req.ExtractForms
	config.PreserveFormat = req.PreserveFormat

	extractor := streaming.NewStreamingExtractor(config)

	// Process with or without progress reporting
	ctx := context.Background()
	var result *streaming.StreamingResult

	if req.ProgressReport {
		result, err = extractor.ExtractWithProgress(ctx, file, fileInfo.Size(),
			func(progress streaming.ProcessingProgress) {
				// Progress callback - could be extended to support callbacks
			})
	} else {
		result, err = extractor.ExtractFromReader(ctx, file, fileInfo.Size())
	}

	if err != nil {
		return &PDFStreamProcessResult{
			FilePath: normalizedPath,
			Status:   "error",
			Error:    err.Error(),
		}, nil
	}

	// Convert result to service format
	return s.convertStreamingResult(normalizedPath, result), nil
}

// StreamProcessPages processes specific pages using streaming
func (s *Service) StreamProcessPages(req PDFStreamPageRequest) (*PDFStreamPageResult, error) {
	if !s.streamingEnabled {
		return nil, fmt.Errorf("streaming processing is not enabled")
	}

	// Normalize relative paths
	normalizedPath, err := s.pathValidator.NormalizePath(req.Path)
	if err != nil {
		return nil, fmt.Errorf("security validation failed: %w", err)
	}

	// Open file
	file, err := os.Open(normalizedPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Create streaming parser
	config := s.getStreamingConfig(req.Config)
	parserOpts := streaming.StreamOptions{
		ChunkSizeMB:     int(config.ChunkSize / (1024 * 1024)),
		MaxMemoryMB:     int(config.MaxMemory / (1024 * 1024)),
		XRefCacheSize:   config.CacheSize,
		ObjectCacheSize: config.CacheSize / 2,
		GCTrigger:       0.8,
		BufferPoolSize:  config.BufferPoolSize,
	}

	parser, err := streaming.NewStreamParser(file, parserOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to create stream parser: %w", err)
	}
	defer parser.Close()

	// Create page streamer
	pageConfig := streaming.PageStreamerConfig{
		ExtractText:   req.ExtractText,
		ExtractImages: req.ExtractImages,
		ExtractForms:  req.ExtractForms,
	}
	streamer := streaming.NewPageStreamer(parser, pageConfig)

	// Process pages
	var processedPages []StreamPage
	ctx := context.Background()

	err = streamer.StreamPages(ctx, func(page *streaming.StreamPage) error {
		streamPage := s.convertStreamPage(page)

		// Filter by page range if specified
		if req.StartPage > 0 && page.Number < req.StartPage {
			return nil
		}
		if req.EndPage > 0 && page.Number > req.EndPage {
			return nil
		}

		processedPages = append(processedPages, streamPage)
		return nil
	})
	if err != nil {
		return &PDFStreamPageResult{
			FilePath: normalizedPath,
			Status:   "error",
			Error:    err.Error(),
		}, nil
	}

	return &PDFStreamPageResult{
		FilePath:    normalizedPath,
		Pages:       processedPages,
		TotalPages:  streamer.GetPageCount(),
		ProcessedAt: time.Now().UnixMilli(),
		Status:      "completed",
	}, nil
}

// StreamExtractText extracts only text using streaming for memory efficiency
func (s *Service) StreamExtractText(req PDFStreamTextRequest) (*PDFStreamTextResult, error) {
	if !s.streamingEnabled {
		return nil, fmt.Errorf("streaming processing is not enabled")
	}

	// Normalize relative paths
	normalizedPath, err := s.pathValidator.NormalizePath(req.Path)
	if err != nil {
		return nil, fmt.Errorf("security validation failed: %w", err)
	}

	// Open file
	file, err := os.Open(normalizedPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Configure for text-only extraction
	config := s.getStreamingConfig(req.Config)
	config.ExtractText = true
	config.ExtractImages = false
	config.ExtractForms = false

	extractor := streaming.NewStreamingExtractor(config)

	var textLength int
	var outputPath string

	if req.OutputPath != "" {
		// Stream to file
		outputFile, err := os.Create(req.OutputPath)
		if err != nil {
			return nil, fmt.Errorf("failed to create output file: %w", err)
		}
		defer outputFile.Close()

		ctx := context.Background()
		err = extractor.ExtractTextStream(ctx, file, outputFile)
		if err != nil {
			return &PDFStreamTextResult{
				FilePath: normalizedPath,
				Status:   "error",
				Error:    err.Error(),
			}, nil
		}

		// Get file size for text length
		info, _ := outputFile.Stat()
		textLength = int(info.Size())
		outputPath = req.OutputPath
	} else {
		// Extract to memory
		ctx := context.Background()
		result, err := extractor.ExtractFromReader(ctx, file, 0)
		if err != nil {
			return &PDFStreamTextResult{
				FilePath: normalizedPath,
				Status:   "error",
				Error:    err.Error(),
			}, nil
		}

		textLength = len(result.Content.Text)
	}

	return &PDFStreamTextResult{
		FilePath:    normalizedPath,
		OutputPath:  outputPath,
		TextLength:  textLength,
		ProcessedAt: time.Now().UnixMilli(),
		Status:      "completed",
	}, nil
}

// Helper methods for streaming

// getStreamingConfig converts service config to streaming config
func (s *Service) getStreamingConfig(config *StreamingConfig) streaming.StreamingConfig {
	if config == nil {
		// Use defaults based on service limits
		return streaming.StreamingConfig{
			ChunkSize:      1024 * 1024,       // 1MB chunks
			MaxMemory:      s.maxFileSize / 4, // Use 1/4 of max file size as memory limit
			ExtractText:    true,
			ExtractImages:  true,
			ExtractForms:   true,
			PreserveFormat: false,
			EnableCaching:  true,
			CacheSize:      1000,
			BufferPoolSize: 10,
		}
	}

	return streaming.StreamingConfig{
		ChunkSize:      int64(config.ChunkSizeMB) * 1024 * 1024,
		MaxMemory:      int64(config.MaxMemoryMB) * 1024 * 1024,
		ExtractText:    true,
		ExtractImages:  true,
		ExtractForms:   true,
		PreserveFormat: false,
		EnableCaching:  config.EnableCaching,
		CacheSize:      config.CacheSize,
		BufferPoolSize: config.BufferPoolSize,
	}
}

// convertStreamingResult converts streaming result to service format
func (s *Service) convertStreamingResult(filePath string, result *streaming.StreamingResult) *PDFStreamProcessResult {
	return &PDFStreamProcessResult{
		FilePath: filePath,
		Content: &StreamProcessedContent{
			Text:   result.Content.Text,
			Images: s.convertStreamImages(result.Content.Images),
			Forms:  s.convertStreamForms(result.Content.Forms),
			Pages:  s.convertStreamPages(result.Content.Pages),
		},
		Progress: &StreamProcessingProgress{
			CurrentPage:  result.Progress.CurrentPage,
			TextSize:     result.Progress.TextSize,
			ImageCount:   result.Progress.ImageCount,
			FormCount:    result.Progress.FormCount,
			ObjectsFound: result.Progress.ObjectsFound,
		},
		MemoryStats: &StreamMemoryStats{
			CurrentBytes:    result.MemoryStats.CurrentBytes,
			MaxBytes:        result.MemoryStats.MaxBytes,
			UsagePercent:    result.MemoryStats.UsagePercent,
			XRefCacheSize:   result.MemoryStats.XRefCacheSize,
			ObjectCacheSize: result.MemoryStats.ObjectCacheSize,
		},
		ProcessingStats: &StreamProcessingStats{
			TotalChunks:     result.ProcessingStats.TotalChunks,
			ProcessedChunks: result.ProcessingStats.ProcessedChunks,
			TotalObjects:    result.ProcessingStats.TotalObjects,
			ProcessingTime:  result.ProcessingStats.ProcessingTime,
			BytesProcessed:  result.ProcessingStats.BytesProcessed,
		},
		Status: "completed",
	}
}

// convertStreamPage converts streaming page to service format
func (s *Service) convertStreamPage(page *streaming.StreamPage) StreamPage {
	return StreamPage{
		Number: page.Number,
		Offset: page.Offset,
		Length: page.Length,
		Content: StreamPageContent{
			Text:       page.Content.Text,
			Images:     s.convertStreamPageImages(page.Content.Images),
			Forms:      s.convertStreamPageForms(page.Content.Forms),
			TextBlocks: s.convertStreamTextBlocks(page.Content.TextBlocks),
		},
		Metadata: StreamPageMetadata{
			MediaBox:    Rectangle(page.Metadata.MediaBox),
			Rotation:    page.Metadata.Rotation,
			HasImages:   page.Metadata.HasImages,
			HasForms:    page.Metadata.HasForms,
			TextLength:  page.Metadata.TextLength,
			ObjectCount: page.Metadata.Annotations,
		},
		ProcessedAt: page.ProcessedAt,
		Status:      page.Status,
		Error:       page.Error,
	}
}

// Helper conversion methods
func (s *Service) convertStreamImages(images []streaming.ImageInfo) []StreamImageInfo {
	result := make([]StreamImageInfo, len(images))
	for i, img := range images {
		result[i] = StreamImageInfo{
			ObjectNumber: img.ObjectNumber,
			Offset:       img.Offset,
			Length:       img.Length,
			Width:        img.Width,
			Height:       img.Height,
			Format:       img.Format,
		}
	}
	return result
}

func (s *Service) convertStreamForms(forms []streaming.FormInfo) []StreamFormInfo {
	result := make([]StreamFormInfo, len(forms))
	for i, form := range forms {
		result[i] = StreamFormInfo{
			ObjectNumber: form.ObjectNumber,
			Offset:       form.Offset,
			FieldType:    form.FieldType,
			FieldName:    form.FieldName,
			FieldValue:   form.FieldValue,
		}
	}
	return result
}

func (s *Service) convertStreamPages(pages []streaming.PageInfo) []StreamPageInfo {
	result := make([]StreamPageInfo, len(pages))
	for i, page := range pages {
		result[i] = StreamPageInfo{
			Number:   page.Number,
			Offset:   page.Offset,
			Length:   page.Length,
			MediaBox: Rectangle(page.MediaBox),
		}
	}
	return result
}

func (s *Service) convertStreamPageImages(images []streaming.ImageInfo) []StreamImageInfo {
	result := make([]StreamImageInfo, len(images))
	for i, img := range images {
		result[i] = StreamImageInfo{
			ObjectNumber: img.ObjectNumber,
			Offset:       img.Offset,
			Length:       img.Length,
			Width:        img.Width,
			Height:       img.Height,
			Format:       img.Format,
		}
	}
	return result
}

func (s *Service) convertStreamPageForms(forms []streaming.FormInfo) []StreamFormInfo {
	result := make([]StreamFormInfo, len(forms))
	for i, form := range forms {
		result[i] = StreamFormInfo{
			ObjectNumber: form.ObjectNumber,
			Offset:       form.Offset,
			FieldType:    form.FieldType,
			FieldName:    form.FieldName,
			FieldValue:   form.FieldValue,
		}
	}
	return result
}

func (s *Service) convertStreamTextBlocks(blocks []streaming.TextBlock) []StreamTextBlock {
	result := make([]StreamTextBlock, len(blocks))
	for i, block := range blocks {
		result[i] = StreamTextBlock{
			Text:     block.Text,
			X:        block.X,
			Y:        block.Y,
			Width:    block.Width,
			Height:   block.Height,
			FontSize: block.FontSize,
			FontName: block.FontName,
		}
	}
	return result
}

// EnableStreaming enables or disables streaming functionality
func (s *Service) EnableStreaming(enabled bool) {
	s.streamingEnabled = enabled
}

// IsStreamingEnabled returns whether streaming is enabled
func (s *Service) IsStreamingEnabled() bool {
	return s.streamingEnabled
}

// Page Range Extraction Methods

// ExtractPageRange extracts content from specific page ranges efficiently
func (s *Service) ExtractPageRange(req PDFExtractPageRangeRequest) (*PDFExtractPageRangeResult, error) {
	// Normalize relative paths
	normalizedPath, err := s.pathValidator.NormalizePath(req.Path)
	if err != nil {
		return nil, fmt.Errorf("security validation failed: %w", err)
	}

	// Create page range extractor
	config := pagerange.ExtractorConfig{
		MaxCacheSize:    50 * 1024 * 1024, // 50MB cache
		EnableCaching:   true,
		PreloadObjects:  true,
		ParallelEnabled: false,
	}
	extractor := pagerange.NewPageRangeExtractor(config)

	// Convert service ranges to pagerange ranges
	ranges := make([]pagerange.PageRange, len(req.Ranges))
	for i, r := range req.Ranges {
		ranges[i] = pagerange.PageRange{
			Start: r.Start,
			End:   r.End,
		}
	}

	// Configure extraction options
	options := pagerange.ExtractOptions{
		ContentTypes:       req.ContentTypes,
		PreserveFormatting: req.PreserveFormatting,
		IncludeMetadata:    req.IncludeMetadata,
		ExtractImages:      req.ExtractImages,
		ExtractForms:       req.ExtractForms,
		OutputFormat:       req.OutputFormat,
	}

	// Extract content
	result, err := extractor.ExtractFromFile(normalizedPath, ranges, options)
	if err != nil {
		return &PDFExtractPageRangeResult{
			FilePath: normalizedPath,
			Status:   "error",
			Error:    err.Error(),
		}, nil
	}

	// Convert result to service format
	return s.convertPageRangeResult(normalizedPath, result), nil
}

// Helper methods for page range extraction

func (s *Service) convertPageRangeResult(filePath string, result *pagerange.ExtractedContent) *PDFExtractPageRangeResult {
	// Convert pages
	pages := make(map[int]ExtractedPageContent)
	for pageNum, pageContent := range result.Pages {
		pages[pageNum] = ExtractedPageContent{
			PageNumber: pageContent.PageNumber,
			Text:       pageContent.Text,
			Images:     s.convertImageReferences(pageContent.Images),
			Forms:      s.convertFormFields(pageContent.Forms),
			Metadata: ExtractedPageMetadata{
				MediaBox:      Rectangle(pageContent.Metadata.MediaBox),
				CropBox:       Rectangle(pageContent.Metadata.CropBox),
				Rotation:      pageContent.Metadata.Rotation,
				UserUnit:      pageContent.Metadata.UserUnit,
				ResourceCount: pageContent.Metadata.ResourceCount,
				ObjectCount:   pageContent.Metadata.ObjectCount,
			},
			TextBlocks: s.convertTextBlocks(pageContent.TextBlocks),
		}
	}

	// Convert ranges
	ranges := make([]PageRangeSpec, len(result.Ranges))
	for i, r := range result.Ranges {
		ranges[i] = PageRangeSpec{
			Start: r.Start,
			End:   r.End,
		}
	}

	return &PDFExtractPageRangeResult{
		FilePath:    filePath,
		Pages:       pages,
		TotalPages:  result.TotalPages,
		Ranges:      ranges,
		ProcessedAt: time.Now().UnixMilli(),
		Metadata: ExtractionResultMetadata{
			ProcessingTime: result.Metadata.ProcessingTime,
			CacheHits:      result.Metadata.CacheHits,
			CacheMisses:    result.Metadata.CacheMisses,
			ObjectsParsed:  result.Metadata.ObjectsParsed,
			BytesRead:      result.Metadata.BytesRead,
			MemoryUsage:    result.Metadata.MemoryUsage,
		},
		Status: "completed",
	}
}

func (s *Service) convertImageReferences(images []pagerange.ImageReference) []ExtractedImageReference {
	result := make([]ExtractedImageReference, len(images))
	for i, img := range images {
		result[i] = ExtractedImageReference{
			ObjectID:   img.ObjectID,
			X:          img.X,
			Y:          img.Y,
			Width:      img.Width,
			Height:     img.Height,
			Format:     img.Format,
			ColorSpace: img.ColorSpace,
		}
	}
	return result
}

func (s *Service) convertFormFields(forms []pagerange.FormField) []ExtractedFormField {
	result := make([]ExtractedFormField, len(forms))
	for i, form := range forms {
		result[i] = ExtractedFormField{
			FieldType:  form.FieldType,
			FieldName:  form.FieldName,
			FieldValue: form.FieldValue,
			X:          form.X,
			Y:          form.Y,
			Width:      form.Width,
			Height:     form.Height,
		}
	}
	return result
}

func (s *Service) convertTextBlocks(blocks []pagerange.FormattedTextBlock) []ExtractedTextBlock {
	result := make([]ExtractedTextBlock, len(blocks))
	for i, block := range blocks {
		result[i] = ExtractedTextBlock{
			Text:     block.Text,
			X:        block.X,
			Y:        block.Y,
			Width:    block.Width,
			Height:   block.Height,
			FontName: block.FontName,
			FontSize: block.FontSize,
			Color:    block.Color,
		}
	}
	return result
}
