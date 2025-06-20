package streaming

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
	"runtime"
	"strconv"
	"strings"
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

	// XRef table for object resolution
	xrefTable map[int]XRefEntry
	xrefMutex sync.RWMutex

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
		xrefTable:   make(map[int]XRefEntry),
		options:     options,
	}

	// Initialize buffer pool
	parser.bufferPool = sync.Pool{
		New: func() interface{} {
			return make([]byte, chunkSize)
		},
	}

	// Parse XRef table during initialization
	if err := parser.ParseXRefTable(); err != nil {
		// Don't fail completely if XRef parsing fails - some PDFs might be recoverable
		// Try to build a basic XRef table by scanning for objects
		parser.buildBasicXRefTable()
	}

	// Reset reader position to beginning after XRef parsing
	parser.reader.Seek(0, io.SeekStart)
	parser.buffReader.Reset(parser.reader)
	parser.offset = 0

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

// GetObject retrieves a PDF object by its ID and generation
func (sp *StreamParser) GetObject(objID, generation int) (*PDFObject, error) {
	// Check cache first
	cacheKey := fmt.Sprintf("%d_%d", objID, generation)
	if cached, found := sp.objectCache.Get(cacheKey); found && cached != nil {
		if obj, ok := cached.(*PDFObject); ok {
			return obj, nil
		}
	}

	// Look up in XRef table
	sp.xrefMutex.RLock()
	entry, exists := sp.xrefTable[objID]
	sp.xrefMutex.RUnlock()

	if !exists {
		return nil, fmt.Errorf("object %d %d not found", objID, generation)
	}

	if !entry.InUse {
		return nil, fmt.Errorf("object %d %d is not in use", objID, generation)
	}

	// Parse object at the specified offset
	obj, err := sp.parseObjectAt(entry.Offset, objID, generation)
	if err != nil {
		return nil, fmt.Errorf("failed to parse object %d %d: %w", objID, generation, err)
	}

	// Cache the result
	sp.objectCache.Put(cacheKey, obj)
	return obj, nil
}

// parseObjectAt parses a PDF object at the given offset
func (sp *StreamParser) parseObjectAt(offset int64, expectedObjID, expectedGeneration int) (*PDFObject, error) {
	// Save current position
	currentPos, _ := sp.reader.Seek(0, io.SeekCurrent)
	defer sp.reader.Seek(currentPos, io.SeekStart)

	// Seek to object position
	_, err := sp.reader.Seek(offset, io.SeekStart)
	if err != nil {
		return nil, fmt.Errorf("failed to seek to offset %d: %w", offset, err)
	}

	// Read a chunk containing the object
	sp.buffReader.Reset(sp.reader)
	buffer := make([]byte, 4096) // Read up to 4KB
	n, err := sp.buffReader.Read(buffer)
	if err != nil && err != io.EOF {
		return nil, fmt.Errorf("failed to read object data: %w", err)
	}

	content := string(buffer[:n])

	// Parse object header and content (use (?s) flag to make . match newlines)
	objRegex := regexp.MustCompile(fmt.Sprintf(`(?s)%d\s+%d\s+obj\s*(.*?)\s*endobj`, expectedObjID, expectedGeneration))
	matches := objRegex.FindStringSubmatch(content)
	if len(matches) < 2 {
		return nil, fmt.Errorf("object %d %d not found at offset %d", expectedObjID, expectedGeneration, offset)
	}

	objContent := strings.TrimSpace(matches[1])

	return &PDFObject{
		Number:     expectedObjID,
		Generation: expectedGeneration,
		Offset:     offset,
		Length:     int64(len(objContent)),
		Content:    objContent,
	}, nil
}

// ParseXRefTable parses the cross-reference table from the PDF
func (sp *StreamParser) ParseXRefTable() error {
	// Find startxref
	startxrefOffset, err := sp.findStartXRef()
	if err != nil {
		return fmt.Errorf("failed to find startxref: %w", err)
	}

	// Parse XRef table at the found offset
	return sp.parseXRefAt(startxrefOffset)
}

// findStartXRef finds the startxref offset in the PDF
func (sp *StreamParser) findStartXRef() (int64, error) {
	// Read the last 1024 bytes of the file to find startxref
	fileSize, err := sp.reader.Seek(0, io.SeekEnd)
	if err != nil {
		return 0, fmt.Errorf("failed to seek to end: %w", err)
	}

	readSize := int64(1024)
	if fileSize < readSize {
		readSize = fileSize
	}

	_, err = sp.reader.Seek(fileSize-readSize, io.SeekStart)
	if err != nil {
		return 0, fmt.Errorf("failed to seek to read position: %w", err)
	}

	buffer := make([]byte, readSize)
	n, err := sp.reader.Read(buffer)
	if err != nil {
		return 0, fmt.Errorf("failed to read end of file: %w", err)
	}

	content := string(buffer[:n])

	// Find startxref keyword
	startxrefRegex := regexp.MustCompile(`startxref\s+(\d+)`)
	matches := startxrefRegex.FindStringSubmatch(content)
	if len(matches) < 2 {
		return 0, fmt.Errorf("startxref not found")
	}

	offset, err := strconv.ParseInt(matches[1], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid startxref offset: %w", err)
	}

	return offset, nil
}

// parseXRefAt parses the XRef table at the given offset
func (sp *StreamParser) parseXRefAt(offset int64) error {
	// Seek to XRef table
	_, err := sp.reader.Seek(offset, io.SeekStart)
	if err != nil {
		return fmt.Errorf("failed to seek to xref at %d: %w", offset, err)
	}

	sp.buffReader.Reset(sp.reader)

	// Read xref keyword
	line, err := sp.buffReader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read xref keyword: %w", err)
	}

	if !strings.HasPrefix(strings.TrimSpace(line), "xref") {
		return fmt.Errorf("expected 'xref' keyword, got: %s", strings.TrimSpace(line))
	}

	// Parse xref entries
	for {
		line, err := sp.buffReader.ReadString('\n')
		if err != nil {
			break
		}

		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "trailer") {
			break
		}

		// Parse subsection header (start count)
		parts := strings.Fields(line)
		if len(parts) == 2 {
			start, err1 := strconv.Atoi(parts[0])
			count, err2 := strconv.Atoi(parts[1])
			if err1 == nil && err2 == nil {
				// Parse entries for this subsection
				for i := 0; i < count; i++ {
					entryLine, err := sp.buffReader.ReadString('\n')
					if err != nil {
						break
					}
					sp.parseXRefEntry(start+i, strings.TrimSpace(entryLine))
				}
			}
		}
	}

	return nil
}

// parseXRefEntry parses a single XRef entry
func (sp *StreamParser) parseXRefEntry(objID int, line string) {
	parts := strings.Fields(line)
	if len(parts) >= 3 {
		offset, err1 := strconv.ParseInt(parts[0], 10, 64)
		generation, err2 := strconv.Atoi(parts[1])
		flag := parts[2]

		if err1 == nil && err2 == nil {
			entry := XRefEntry{
				Offset:     offset,
				Generation: generation,
				InUse:      flag == "n",
			}

			sp.xrefMutex.Lock()
			sp.xrefTable[objID] = entry
			sp.xrefMutex.Unlock()
		}
	}
}

// buildBasicXRefTable builds a basic XRef table by scanning for objects
func (sp *StreamParser) buildBasicXRefTable() error {
	// Save current position
	currentPos, _ := sp.reader.Seek(0, io.SeekCurrent)
	defer sp.reader.Seek(currentPos, io.SeekStart)

	// Seek to beginning
	_, err := sp.reader.Seek(0, io.SeekStart)
	if err != nil {
		return err
	}

	sp.buffReader.Reset(sp.reader)

	// Scan through the file looking for object headers
	var buffer []byte
	var offset int64 = 0
	var foundObjects = false

	for {
		chunk := make([]byte, 4096)
		n, err := sp.buffReader.Read(chunk)
		if err != nil && err != io.EOF {
			break
		}
		if n == 0 {
			break
		}

		buffer = append(buffer, chunk[:n]...)

		// Look for object patterns in the buffer
		content := string(buffer)
		objRegex := regexp.MustCompile(`(\d+)\s+(\d+)\s+obj`)
		matches := objRegex.FindAllStringSubmatchIndex(content, -1)

		for _, match := range matches {
			if len(match) >= 6 {
				objIDStr := content[match[2]:match[3]]
				generationStr := content[match[4]:match[5]]

				objID, err1 := strconv.Atoi(objIDStr)
				generation, err2 := strconv.Atoi(generationStr)

				if err1 == nil && err2 == nil {
					objOffset := offset + int64(match[0])

					entry := XRefEntry{
						Offset:     objOffset,
						Generation: generation,
						InUse:      true,
					}

					sp.xrefMutex.Lock()
					sp.xrefTable[objID] = entry
					sp.xrefMutex.Unlock()
					foundObjects = true
				}
			}
		}

		// Keep only the last 1KB of buffer to handle objects spanning chunks
		if len(buffer) > 5120 {
			offset += int64(len(buffer) - 1024)
			buffer = buffer[len(buffer)-1024:]
		}

		if err == io.EOF {
			break
		}
	}

	// If no objects were found, this is likely not a valid PDF
	if !foundObjects {
		return fmt.Errorf("no PDF objects found in file")
	}

	return nil
}

// SetXRefTable allows external setting of the XRef table
func (sp *StreamParser) SetXRefTable(xrefTable map[int]XRefEntry) {
	sp.xrefMutex.Lock()
	sp.xrefTable = xrefTable
	sp.xrefMutex.Unlock()
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
