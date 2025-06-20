package streaming

import (
	"bufio"
	"fmt"
	"io"
	"runtime"
	"sync"
)

// StreamParser handles streaming PDF processing for large files
type StreamParser struct {
	reader     io.ReadSeeker
	buffReader *bufio.Reader
	chunkSize  int64
	offset     int64
	maxMemory  int64
	currentMem int64
	gcTrigger  float64

	// Caches for performance
	xrefCache   *LRUCache
	objectCache *LRUCache

	// Memory management
	memMutex   sync.RWMutex
	bufferPool sync.Pool

	// Configuration
	options StreamOptions
}

// StreamOptions configures the streaming parser
type StreamOptions struct {
	ChunkSizeMB     int     // Size of processing chunks in MB
	MaxMemoryMB     int     // Maximum memory usage in MB
	XRefCacheSize   int     // Number of xref entries to cache
	ObjectCacheSize int     // Number of objects to cache
	GCTrigger       float64 // Trigger GC when memory usage exceeds this percentage
	BufferPoolSize  int     // Size of buffer pool
}

// DefaultStreamOptions returns sensible defaults for streaming
func DefaultStreamOptions() StreamOptions {
	return StreamOptions{
		ChunkSizeMB:     1,    // 1MB chunks
		MaxMemoryMB:     64,   // 64MB max memory
		XRefCacheSize:   1000, // Cache 1000 xref entries
		ObjectCacheSize: 500,  // Cache 500 objects
		GCTrigger:       0.8,  // Trigger GC at 80% memory usage
		BufferPoolSize:  10,   // Pool of 10 buffers
	}
}

// NewStreamParser creates a new streaming PDF parser
func NewStreamParser(reader io.ReadSeeker, opts ...StreamOptions) (*StreamParser, error) {
	if reader == nil {
		return nil, fmt.Errorf("reader cannot be nil")
	}

	options := DefaultStreamOptions()
	if len(opts) > 0 {
		options = opts[0]
	}

	// Validate options
	if options.ChunkSizeMB <= 0 {
		options.ChunkSizeMB = 1
	}
	if options.MaxMemoryMB <= 0 {
		options.MaxMemoryMB = 64
	}
	if options.GCTrigger <= 0 || options.GCTrigger > 1 {
		options.GCTrigger = 0.8
	}

	chunkSize := int64(options.ChunkSizeMB * 1024 * 1024)
	maxMemory := int64(options.MaxMemoryMB * 1024 * 1024)

	parser := &StreamParser{
		reader:      reader,
		buffReader:  bufio.NewReaderSize(reader, int(chunkSize/4)), // Use 1/4 chunk size for buffer
		chunkSize:   chunkSize,
		maxMemory:   maxMemory,
		gcTrigger:   options.GCTrigger,
		xrefCache:   NewLRUCache(options.XRefCacheSize),
		objectCache: NewLRUCache(options.ObjectCacheSize),
		options:     options,
	}

	// Initialize buffer pool
	parser.bufferPool = sync.Pool{
		New: func() interface{} {
			return make([]byte, chunkSize)
		},
	}

	return parser, nil
}

// Seek moves the parser to a specific position in the stream
func (sp *StreamParser) Seek(offset int64, whence int) (int64, error) {
	newOffset, err := sp.reader.Seek(offset, whence)
	if err != nil {
		return 0, fmt.Errorf("seek failed: %w", err)
	}

	sp.offset = newOffset
	sp.buffReader.Reset(sp.reader)
	return newOffset, nil
}

// ReadChunk reads a chunk of data from the current position
func (sp *StreamParser) ReadChunk() ([]byte, error) {
	// Get buffer from pool
	buffer := sp.getBuffer()
	defer sp.putBuffer(buffer)

	// Check memory usage before allocation - only check for the data copy size
	n, err := sp.buffReader.Read(buffer)
	if err != nil && err != io.EOF {
		return nil, fmt.Errorf("failed to read chunk: %w", err)
	}

	if n == 0 {
		return nil, io.EOF
	}

	// Check if we can allocate memory for the actual data read
	if err := sp.checkMemoryUsage(n); err != nil {
		return nil, err
	}

	// Create a copy of the actual data read
	data := make([]byte, n)
	copy(data, buffer[:n])

	sp.offset += int64(n)
	sp.trackMemoryUsage(n)

	return data, nil
}

// ProcessInChunks processes the entire stream in chunks
func (sp *StreamParser) ProcessInChunks(processor func([]byte, int64) error) error {
	sp.Seek(0, io.SeekStart)

	for {
		startOffset := sp.offset
		chunk, err := sp.ReadChunk()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("error reading chunk at offset %d: %w", startOffset, err)
		}

		if err := processor(chunk, startOffset); err != nil {
			return fmt.Errorf("processor failed at offset %d: %w", startOffset, err)
		}

		// Release memory after processing chunk
		sp.releaseMemory(len(chunk))

		// Check if we should trigger GC
		sp.maybeRunGC()
	}

	return nil
}

// FindPattern searches for a byte pattern in the stream
func (sp *StreamParser) FindPattern(pattern []byte) ([]int64, error) {
	if len(pattern) == 0 {
		return nil, fmt.Errorf("pattern cannot be empty")
	}

	var positions []int64
	sp.Seek(0, io.SeekStart)

	var overlap []byte

	for {
		startOffset := sp.offset
		chunk, err := sp.ReadChunk()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		// Combine with overlap from previous chunk
		searchData := chunk
		if len(overlap) > 0 {
			searchData = append(overlap, chunk...)
		}

		// Find all occurrences in this chunk
		for i := 0; i <= len(searchData)-len(pattern); i++ {
			if sp.bytesEqual(searchData[i:i+len(pattern)], pattern) {
				actualOffset := startOffset + int64(i) - int64(len(overlap))
				positions = append(positions, actualOffset)
			}
		}

		// Keep overlap for next iteration (pattern length - 1)
		if len(pattern) > 1 && len(searchData) >= len(pattern)-1 {
			overlapSize := len(pattern) - 1
			overlap = make([]byte, overlapSize)
			copy(overlap, searchData[len(searchData)-overlapSize:])
		} else {
			overlap = nil
		}
	}

	return positions, nil
}

// GetCurrentOffset returns the current stream position
func (sp *StreamParser) GetCurrentOffset() int64 {
	return sp.offset
}

// GetMemoryUsage returns current memory usage statistics
func (sp *StreamParser) GetMemoryUsage() MemoryStats {
	sp.memMutex.RLock()
	defer sp.memMutex.RUnlock()

	return MemoryStats{
		CurrentBytes:    sp.currentMem,
		MaxBytes:        sp.maxMemory,
		UsagePercent:    float64(sp.currentMem) / float64(sp.maxMemory) * 100,
		XRefCacheSize:   sp.xrefCache.Len(),
		ObjectCacheSize: sp.objectCache.Len(),
	}
}

// ClearCaches clears all internal caches to free memory
func (sp *StreamParser) ClearCaches() {
	sp.xrefCache.Clear()
	sp.objectCache.Clear()
	sp.memMutex.Lock()
	sp.currentMem = 0
	sp.memMutex.Unlock()
}

// Close releases all resources held by the parser
func (sp *StreamParser) Close() error {
	sp.ClearCaches()
	if closer, ok := sp.reader.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}

// Internal helper methods

func (sp *StreamParser) getBuffer() []byte {
	return sp.bufferPool.Get().([]byte)
}

func (sp *StreamParser) putBuffer(buffer []byte) {
	sp.bufferPool.Put(buffer)
}

func (sp *StreamParser) checkMemoryUsage(additionalBytes int) error {
	sp.memMutex.RLock()
	defer sp.memMutex.RUnlock()

	if sp.currentMem+int64(additionalBytes) > sp.maxMemory {
		return fmt.Errorf("operation would exceed memory limit: %d + %d > %d",
			sp.currentMem, additionalBytes, sp.maxMemory)
	}

	return nil
}

func (sp *StreamParser) trackMemoryUsage(bytes int) {
	sp.memMutex.Lock()
	sp.currentMem += int64(bytes)
	sp.memMutex.Unlock()
}

func (sp *StreamParser) releaseMemory(bytes int) {
	sp.memMutex.Lock()
	sp.currentMem -= int64(bytes)
	if sp.currentMem < 0 {
		sp.currentMem = 0
	}
	sp.memMutex.Unlock()
}

func (sp *StreamParser) maybeRunGC() {
	sp.memMutex.RLock()
	shouldGC := float64(sp.currentMem)/float64(sp.maxMemory) > sp.gcTrigger
	sp.memMutex.RUnlock()

	if shouldGC {
		runtime.GC()
	}
}

func (sp *StreamParser) bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// MemoryStats provides information about memory usage
type MemoryStats struct {
	CurrentBytes    int64   `json:"current_bytes"`
	MaxBytes        int64   `json:"max_bytes"`
	UsagePercent    float64 `json:"usage_percent"`
	XRefCacheSize   int     `json:"xref_cache_size"`
	ObjectCacheSize int     `json:"object_cache_size"`
}
