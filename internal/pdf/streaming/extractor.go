package streaming

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"
)

// StreamingExtractor provides a high-level API for streaming PDF processing
type StreamingExtractor struct {
	config    StreamingConfig
	parser    *StreamParser
	processor *ChunkProcessor
}

// NewStreamingExtractor creates a new streaming extractor with the given configuration
func NewStreamingExtractor(config ...StreamingConfig) *StreamingExtractor {
	cfg := DefaultStreamingConfig()
	if len(config) > 0 {
		cfg = config[0]
	}

	return &StreamingExtractor{
		config: cfg,
	}
}

// ExtractFromFile extracts content from a PDF file using streaming processing
func (se *StreamingExtractor) ExtractFromFile(ctx context.Context, filePath string) (*StreamingResult, error) {
	// Open file
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Get file info for size validation
	fileInfo, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	return se.ExtractFromReader(ctx, file, fileInfo.Size())
}

// ExtractFromReader extracts content from a PDF using an io.ReadSeeker
func (se *StreamingExtractor) ExtractFromReader(ctx context.Context, reader io.ReadSeeker, size int64) (*StreamingResult, error) {
	startTime := time.Now()

	// Initialize parser
	parserOpts := StreamOptions{
		ChunkSizeMB:     int(se.config.ChunkSize / (1024 * 1024)),
		MaxMemoryMB:     int(se.config.MaxMemory / (1024 * 1024)),
		XRefCacheSize:   se.config.CacheSize,
		ObjectCacheSize: se.config.CacheSize / 2,
		GCTrigger:       0.8,
		BufferPoolSize:  se.config.BufferPoolSize,
	}

	parser, err := NewStreamParser(reader, parserOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to create stream parser: %w", err)
	}
	defer parser.Close()

	// Initialize processor
	processorConfig := ProcessorConfig{
		MaxTextBufferSize:  int(se.config.MaxMemory / 4), // Use 1/4 of max memory for text
		MaxImageBuffer:     100,
		MaxFormFields:      500,
		ExtractImages:      se.config.ExtractImages,
		ExtractForms:       se.config.ExtractForms,
		ExtractText:        se.config.ExtractText,
		PreserveFormatting: se.config.PreserveFormat,
	}

	processor := NewChunkProcessor(parser, processorConfig)
	defer processor.Reset()

	// Track processing statistics
	stats := ProcessingStats{
		TotalChunks:     0,
		ProcessedChunks: 0,
		TotalObjects:    0,
		BytesProcessed:  0,
	}

	// Process file in chunks
	err = parser.ProcessInChunks(func(chunk []byte, offset int64) error {
		// Check for context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Process this chunk
		result, err := processor.ProcessChunk(chunk, offset)
		if err != nil {
			return fmt.Errorf("failed to process chunk at offset %d: %w", offset, err)
		}

		// Update statistics
		stats.TotalChunks++
		stats.ProcessedChunks++
		stats.TotalObjects += result.ObjectCount
		stats.BytesProcessed += int64(len(chunk))

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("streaming processing failed: %w", err)
	}

	// Flush all buffers to get final content
	content, err := processor.FlushBuffers()
	if err != nil {
		return nil, fmt.Errorf("failed to flush buffers: %w", err)
	}

	// Get final statistics
	stats.ProcessingTime = time.Since(startTime).Milliseconds()
	memStats := parser.GetMemoryUsage()
	progress := processor.GetProgress()

	return &StreamingResult{
		Content:         content,
		Progress:        progress,
		MemoryStats:     memStats,
		ProcessingStats: stats,
	}, nil
}

// ExtractTextStream extracts only text content in a streaming fashion
func (se *StreamingExtractor) ExtractTextStream(ctx context.Context, reader io.ReadSeeker, writer io.Writer) error {
	// Configure for text-only extraction
	config := se.config
	config.ExtractText = true
	config.ExtractImages = false
	config.ExtractForms = false

	tempExtractor := &StreamingExtractor{config: config}

	result, err := tempExtractor.ExtractFromReader(ctx, reader, 0)
	if err != nil {
		return err
	}

	// Write text to output writer
	_, err = writer.Write([]byte(result.Content.Text))
	return err
}

// ExtractWithProgress extracts content with progress reporting
func (se *StreamingExtractor) ExtractWithProgress(ctx context.Context, reader io.ReadSeeker, size int64,
	progressCallback func(ProcessingProgress),
) (*StreamingResult, error) {
	startTime := time.Now()

	// Initialize components
	parserOpts := StreamOptions{
		ChunkSizeMB:     int(se.config.ChunkSize / (1024 * 1024)),
		MaxMemoryMB:     int(se.config.MaxMemory / (1024 * 1024)),
		XRefCacheSize:   se.config.CacheSize,
		ObjectCacheSize: se.config.CacheSize / 2,
		GCTrigger:       0.8,
		BufferPoolSize:  se.config.BufferPoolSize,
	}

	parser, err := NewStreamParser(reader, parserOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to create stream parser: %w", err)
	}
	defer parser.Close()

	processorConfig := ProcessorConfig{
		MaxTextBufferSize:  int(se.config.MaxMemory / 4),
		MaxImageBuffer:     100,
		MaxFormFields:      500,
		ExtractImages:      se.config.ExtractImages,
		ExtractForms:       se.config.ExtractForms,
		ExtractText:        se.config.ExtractText,
		PreserveFormatting: se.config.PreserveFormat,
	}

	processor := NewChunkProcessor(parser, processorConfig)
	defer processor.Reset()

	stats := ProcessingStats{}

	// Process with progress reporting
	err = parser.ProcessInChunks(func(chunk []byte, offset int64) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		result, err := processor.ProcessChunk(chunk, offset)
		if err != nil {
			return fmt.Errorf("failed to process chunk at offset %d: %w", offset, err)
		}

		// Update statistics
		stats.TotalChunks++
		stats.ProcessedChunks++
		stats.TotalObjects += result.ObjectCount
		stats.BytesProcessed += int64(len(chunk))

		// Report progress
		if progressCallback != nil {
			progress := processor.GetProgress()
			progressCallback(progress)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("streaming processing failed: %w", err)
	}

	// Get final results
	content, err := processor.FlushBuffers()
	if err != nil {
		return nil, fmt.Errorf("failed to flush buffers: %w", err)
	}

	stats.ProcessingTime = time.Since(startTime).Milliseconds()
	memStats := parser.GetMemoryUsage()
	progress := processor.GetProgress()

	return &StreamingResult{
		Content:         content,
		Progress:        progress,
		MemoryStats:     memStats,
		ProcessingStats: stats,
	}, nil
}

// EstimateMemoryUsage estimates memory usage for processing a file of given size
func (se *StreamingExtractor) EstimateMemoryUsage(fileSize int64) MemoryEstimate {
	// Calculate number of chunks
	chunkCount := (fileSize + se.config.ChunkSize - 1) / se.config.ChunkSize

	// Estimate memory components
	parserMemory := se.config.MaxMemory / 2          // Parser uses up to half of max memory
	bufferMemory := se.config.MaxMemory / 4          // Buffers use up to quarter
	cacheMemory := int64(se.config.CacheSize * 1024) // Rough estimate for cache

	totalEstimate := parserMemory + bufferMemory + cacheMemory

	return MemoryEstimate{
		FileSize:        fileSize,
		EstimatedChunks: chunkCount,
		ParserMemory:    parserMemory,
		BufferMemory:    bufferMemory,
		CacheMemory:     cacheMemory,
		TotalEstimate:   totalEstimate,
		Recommendation:  se.getMemoryRecommendation(totalEstimate),
	}
}

// ValidateConfiguration validates the streaming configuration
func (se *StreamingExtractor) ValidateConfiguration() error {
	if se.config.ChunkSize <= 0 {
		return fmt.Errorf("chunk size must be greater than 0")
	}
	if se.config.MaxMemory <= 0 {
		return fmt.Errorf("max memory must be greater than 0")
	}
	if se.config.ChunkSize > se.config.MaxMemory {
		return fmt.Errorf("chunk size cannot be larger than max memory")
	}
	if se.config.CacheSize <= 0 {
		return fmt.Errorf("cache size must be greater than 0")
	}
	if se.config.BufferPoolSize <= 0 {
		return fmt.Errorf("buffer pool size must be greater than 0")
	}

	return nil
}

// GetConfiguration returns the current configuration
func (se *StreamingExtractor) GetConfiguration() StreamingConfig {
	return se.config
}

// UpdateConfiguration updates the configuration
func (se *StreamingExtractor) UpdateConfiguration(config StreamingConfig) error {
	tempExtractor := &StreamingExtractor{config: config}
	if err := tempExtractor.ValidateConfiguration(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	se.config = config
	return nil
}

// Internal helper methods

func (se *StreamingExtractor) getMemoryRecommendation(estimatedMemory int64) string {
	maxMem := se.config.MaxMemory

	if estimatedMemory <= maxMem {
		return "Configuration should work well for this file size"
	} else if estimatedMemory <= maxMem*2 {
		return "Consider increasing max memory or reducing chunk size"
	} else {
		return "File may be too large for current configuration, consider streaming-only operations"
	}
}

// MemoryEstimate provides memory usage estimates
type MemoryEstimate struct {
	FileSize        int64  `json:"file_size"`
	EstimatedChunks int64  `json:"estimated_chunks"`
	ParserMemory    int64  `json:"parser_memory"`
	BufferMemory    int64  `json:"buffer_memory"`
	CacheMemory     int64  `json:"cache_memory"`
	TotalEstimate   int64  `json:"total_estimate"`
	Recommendation  string `json:"recommendation"`
}

// StreamingOptions provides builder-pattern configuration
type StreamingOptions struct {
	extractor *StreamingExtractor
}

// NewStreamingOptions creates a new options builder
func NewStreamingOptions() *StreamingOptions {
	return &StreamingOptions{
		extractor: NewStreamingExtractor(),
	}
}

// WithChunkSize sets the chunk size
func (so *StreamingOptions) WithChunkSize(sizeBytes int64) *StreamingOptions {
	so.extractor.config.ChunkSize = sizeBytes
	return so
}

// WithMaxMemory sets the maximum memory usage
func (so *StreamingOptions) WithMaxMemory(memoryBytes int64) *StreamingOptions {
	so.extractor.config.MaxMemory = memoryBytes
	return so
}

// WithTextExtraction enables/disables text extraction
func (so *StreamingOptions) WithTextExtraction(enabled bool) *StreamingOptions {
	so.extractor.config.ExtractText = enabled
	return so
}

// WithImageExtraction enables/disables image extraction
func (so *StreamingOptions) WithImageExtraction(enabled bool) *StreamingOptions {
	so.extractor.config.ExtractImages = enabled
	return so
}

// WithFormExtraction enables/disables form extraction
func (so *StreamingOptions) WithFormExtraction(enabled bool) *StreamingOptions {
	so.extractor.config.ExtractForms = enabled
	return so
}

// WithCaching enables/disables caching with specified size
func (so *StreamingOptions) WithCaching(enabled bool, cacheSize int) *StreamingOptions {
	so.extractor.config.EnableCaching = enabled
	so.extractor.config.CacheSize = cacheSize
	return so
}

// Build creates the configured streaming extractor
func (so *StreamingOptions) Build() (*StreamingExtractor, error) {
	if err := so.extractor.ValidateConfiguration(); err != nil {
		return nil, err
	}
	return so.extractor, nil
}
