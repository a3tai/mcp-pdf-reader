package xref

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// XRefParser handles comprehensive cross-reference table parsing for random access
type XRefParser struct {
	reader      io.ReadSeeker
	entries     map[int]map[int]*XRefEntry // objNum -> generation -> entry
	trailers    []*TrailerDict             // Chain of trailers for incremental updates
	startxref   int64                      // Offset to main xref
	prevChain   []int64                    // Chain of previous xref offsets
	objectCache map[ObjectID]PDFObject     // Cache for resolved objects
}

// XRefEntry represents an entry in the cross-reference table
type XRefEntry struct {
	Type        EntryType // Free, InUse, Compressed
	Offset      int64     // Byte offset for InUse, or object stream number for Compressed
	Generation  int       // Generation number (0 for compressed)
	StreamIndex int       // Index within object stream (for compressed objects)
}

// EntryType represents the type of cross-reference entry
type EntryType int

const (
	EntryFree EntryType = iota
	EntryInUse
	EntryCompressed
)

func (t EntryType) String() string {
	switch t {
	case EntryFree:
		return "free"
	case EntryInUse:
		return "in-use"
	case EntryCompressed:
		return "compressed"
	default:
		return "unknown"
	}
}

// TrailerDict represents a PDF trailer dictionary
type TrailerDict struct {
	Size    int          // Total number of entries
	Prev    *int64       // Offset to previous xref (for incremental updates)
	Root    *IndirectRef // Catalog dictionary
	Encrypt *IndirectRef // Encryption dictionary
	Info    *IndirectRef // Info dictionary
	ID      [][]byte     // File identifiers
	XRefStm *int64       // Cross-reference stream object number
}

// IndirectRef represents an indirect object reference for the xref package
type IndirectRef struct {
	ObjectNumber     int64
	GenerationNumber int64
}

func (r *IndirectRef) String() string {
	return fmt.Sprintf("%d %d R", r.ObjectNumber, r.GenerationNumber)
}

// Local types to avoid circular import

// ObjectID represents a PDF object identifier
type ObjectID struct {
	Number     int64 // Object number
	Generation int64 // Generation number
}

// PDFObject interface for local use
type PDFObject interface {
	Type() ObjectType
	String() string
}

// ObjectType represents the type of a PDF object
type ObjectType int

const (
	TypeNull ObjectType = iota
	TypeBool
	TypeNumber
	TypeString
	TypeName
	TypeArray
	TypeDictionary
	TypeStream
	TypeIndirectRef
	TypeKeyword
)

// Null represents a PDF null object
type Null struct{}

func (n *Null) Type() ObjectType { return TypeNull }
func (n *Null) String() string   { return "null" }

// Keyword represents a PDF keyword/operator
type Keyword struct {
	Value string
}

func (k *Keyword) Type() ObjectType { return TypeKeyword }
func (k *Keyword) String() string   { return k.Value }

// NewXRefParser creates a new cross-reference parser
func NewXRefParser(reader io.ReadSeeker) *XRefParser {
	return &XRefParser{
		reader:      reader,
		entries:     make(map[int]map[int]*XRefEntry),
		trailers:    make([]*TrailerDict, 0),
		objectCache: make(map[ObjectID]PDFObject),
	}
}

// min helper function
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// ParseXRef parses all cross-reference sections starting from startxref
func (p *XRefParser) ParseXRef(startxref int64) error {
	p.startxref = startxref
	p.entries = make(map[int]map[int]*XRefEntry)
	p.trailers = p.trailers[:0] // Clear but keep capacity
	p.prevChain = p.prevChain[:0]

	// Follow the Prev chain to parse all xref sections
	offset := startxref
	parsedSections := 0
	for offset >= 0 {
		p.prevChain = append(p.prevChain, offset)

		if _, err := p.reader.Seek(offset, io.SeekStart); err != nil {
			return fmt.Errorf("seek to xref at %d: %w", offset, err)
		}

		// Read a larger buffer to detect the format
		buf := make([]byte, 100)
		n, _ := p.reader.Read(buf)
		if _, err := p.reader.Seek(offset, io.SeekStart); err != nil { // Reset position
			return fmt.Errorf("reset seek position: %w", err)
		}

		var trailer *TrailerDict
		var err error

		content := string(buf[:n])
		// Check if it starts with "xref" keyword (traditional xref table)
		if strings.HasPrefix(strings.TrimSpace(content), "xref") {
			trailer, err = p.parseXRefTable()
		} else {
			// For now, assume it's a traditional table if we can't detect xref streams
			// In the future, this should parse xref streams properly
			trailer, err = p.parseXRefTable()
		}

		if err != nil {
			// Be liberal in what we accept - don't fail completely on parse errors
			fmt.Printf("Warning: parse xref at %d: %v (content: %q)\n", offset, err, string(buf[:min(n, 50)]))
			break
		}

		if trailer != nil {
			p.trailers = append(p.trailers, trailer)
			parsedSections++

			// Continue with previous xref if exists
			if trailer.Prev != nil {
				offset = *trailer.Prev
			} else {
				offset = -1 // Terminate loop
			}
		} else {
			break
		}
	}

	if len(p.trailers) == 0 {
		return fmt.Errorf("no valid xref sections found (parsed %d sections, started at offset %d)", parsedSections, startxref)
	}

	return nil
}

// parseXRefTable handles traditional cross-reference tables
func (p *XRefParser) parseXRefTable() (*TrailerDict, error) {
	scanner := bufio.NewScanner(p.reader)

	// Read "xref" keyword
	if !scanner.Scan() {
		return nil, fmt.Errorf("failed to read xref keyword: %v", scanner.Err())
	}
	line := strings.TrimSpace(scanner.Text())
	if line != "xref" {
		return nil, fmt.Errorf("expected 'xref' keyword, got '%s' (position may be incorrect)", line)
	}

	// Parse xref subsections
	for scanner.Scan() {
		line = strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if line == "trailer" {
			break
		}

		// Parse subsection header: start_num count
		parts := strings.Fields(line)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid xref subsection header: %s (expected 2 parts, got %d)", line, len(parts))
		}

		startNum, err := strconv.ParseInt(parts[0], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid start number '%s' in xref subsection: %w", parts[0], err)
		}

		count, err := strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid count '%s' in xref subsection: %w", parts[1], err)
		}

		// Parse xref entries for this subsection
		for i := int64(0); i < count; i++ {
			if !scanner.Scan() {
				return nil, fmt.Errorf("unexpected end of xref entries")
			}
			entryLine := scanner.Text()

			objNum := int(startNum + i)
			entry, err := p.parseXRefEntryLine(entryLine, objNum)
			if err != nil {
				// Be liberal - skip malformed entries but continue parsing
				fmt.Printf("Warning: skipping malformed xref entry %d (line: %q): %v\n", objNum, entryLine, err)
				continue
			}

			p.addEntry(objNum, entry)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading xref table: %w", err)
	}

	// Parse trailer dictionary directly here since we already consumed "trailer"
	trailer, err := p.parseTrailerDictFromScanner(scanner)
	if err != nil {
		return nil, fmt.Errorf("failed to parse trailer: %w", err)
	}

	return trailer, nil
}

// parseXRefEntryLine parses a single xref entry line
func (p *XRefParser) parseXRefEntryLine(line string, objNum int) (*XRefEntry, error) {
	// Each entry is exactly 20 bytes: offset(10) generation(5) flag(1) + 2 spaces + newline
	if len(line) < 18 {
		return nil, fmt.Errorf("xref entry too short (len=%d): %q", len(line), line)
	}

	// Extract components (be liberal with whitespace)
	parts := strings.Fields(line)
	if len(parts) < 3 {
		return nil, fmt.Errorf("invalid xref entry format (expected 3 parts, got %d): %q", len(parts), line)
	}

	offset, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid offset '%s': %w", parts[0], err)
	}

	generation, err := strconv.ParseInt(parts[1], 10, 32)
	if err != nil {
		return nil, fmt.Errorf("invalid generation '%s': %w", parts[1], err)
	}

	flag := parts[2]

	entry := &XRefEntry{
		Offset:     offset,
		Generation: int(generation),
	}

	switch flag {
	case "n":
		entry.Type = EntryInUse
	case "f":
		entry.Type = EntryFree
	default:
		// Be liberal - treat unknown flags as free
		fmt.Printf("Warning: unknown xref flag '%s' for object %d, treating as free\n", flag, objNum)
		entry.Type = EntryFree
	}

	return entry, nil
}

// parseXRefStream handles cross-reference streams (PDF 1.5+)
func (p *XRefParser) parseXRefStream() (*TrailerDict, error) {
	// For now, provide a basic implementation that can be extended
	// This would typically involve parsing an object stream with compressed xref data
	return nil, fmt.Errorf("xref streams not yet implemented - using fallback")
}

// parseTrailerDict parses the trailer dictionary
func (p *XRefParser) parseTrailerDict() (*TrailerDict, error) {
	scanner := bufio.NewScanner(p.reader)

	// Look for "trailer" keyword first
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "trailer" {
			break
		}
		if line != "" {
			return nil, fmt.Errorf("expected 'trailer' keyword, got '%s'", line)
		}
	}

	return p.parseTrailerDictFromScanner(scanner)
}

// parseTrailerDictFromScanner parses the trailer dictionary using an existing scanner
func (p *XRefParser) parseTrailerDictFromScanner(scanner *bufio.Scanner) (*TrailerDict, error) {
	var dictContent strings.Builder

	// Read the dictionary content
	inDict := false
	openBrackets := 0

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		if strings.Contains(line, "<<") {
			inDict = true
			openBrackets += strings.Count(line, "<<")
			openBrackets -= strings.Count(line, ">>")
			dictContent.WriteString(line + "\n")
			if openBrackets == 0 {
				break
			}
			continue
		}

		if inDict {
			openBrackets += strings.Count(line, "<<")
			openBrackets -= strings.Count(line, ">>")
			dictContent.WriteString(line + "\n")
			if openBrackets == 0 {
				break
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading trailer: %w", err)
	}

	// Parse the dictionary content into a TrailerDict
	return p.parseTrailerContent(dictContent.String())
}

// parseTrailerContent parses the trailer dictionary content
func (p *XRefParser) parseTrailerContent(content string) (*TrailerDict, error) {
	trailer := &TrailerDict{}

	// Basic parsing - extract key values
	// This is a simplified parser, in production this should use the full PDF lexer
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || line == "<<" || line == ">>" {
			continue
		}

		// Extract key-value pairs
		if strings.Contains(line, "/Size") {
			if parts := strings.Fields(line); len(parts) >= 2 {
				if size, err := strconv.ParseInt(parts[1], 10, 32); err == nil {
					trailer.Size = int(size)
				}
			}
		} else if strings.Contains(line, "/Prev") {
			if parts := strings.Fields(line); len(parts) >= 2 {
				if prev, err := strconv.ParseInt(parts[1], 10, 64); err == nil {
					trailer.Prev = &prev
				}
			}
		} else if strings.Contains(line, "/Root") {
			// Parse indirect reference
			if ref := p.parseIndirectRef(line); ref != nil {
				trailer.Root = ref
			}
		} else if strings.Contains(line, "/Info") {
			// Parse indirect reference
			if ref := p.parseIndirectRef(line); ref != nil {
				trailer.Info = ref
			}
		} else if strings.Contains(line, "/Encrypt") {
			// Parse indirect reference
			if ref := p.parseIndirectRef(line); ref != nil {
				trailer.Encrypt = ref
			}
		}
	}

	return trailer, nil
}

// parseIndirectRef extracts an indirect reference from a line
func (p *XRefParser) parseIndirectRef(line string) *IndirectRef {
	parts := strings.Fields(line)
	for i := 0; i < len(parts)-2; i++ {
		if parts[i+2] == "R" {
			if objNum, err := strconv.ParseInt(parts[i], 10, 64); err == nil {
				if genNum, err := strconv.ParseInt(parts[i+1], 10, 64); err == nil {
					return &IndirectRef{
						ObjectNumber:     objNum,
						GenerationNumber: genNum,
					}
				}
			}
		}
	}
	return nil
}

// addEntry adds an xref entry to the table
func (p *XRefParser) addEntry(objNum int, entry *XRefEntry) {
	if p.entries[objNum] == nil {
		p.entries[objNum] = make(map[int]*XRefEntry)
	}
	p.entries[objNum][entry.Generation] = entry
}

// GetEntry retrieves an xref entry for the given object number and generation
func (p *XRefParser) GetEntry(objNum int, generation int) *XRefEntry {
	if genMap := p.entries[objNum]; genMap != nil {
		return genMap[generation]
	}
	return nil
}

// GetLatestEntry retrieves the latest xref entry for the given object number
func (p *XRefParser) GetLatestEntry(objNum int) *XRefEntry {
	if genMap := p.entries[objNum]; genMap != nil {
		// Find the highest generation number
		maxGen := -1
		var latestEntry *XRefEntry
		for gen, entry := range genMap {
			if gen > maxGen && entry.Type == EntryInUse {
				maxGen = gen
				latestEntry = entry
			}
		}
		return latestEntry
	}
	return nil
}

// HasEntry checks if an entry exists for the given object number and generation
func (p *XRefParser) HasEntry(objNum int, generation int) bool {
	return p.GetEntry(objNum, generation) != nil
}

// GetTrailer returns the main (latest) trailer dictionary
func (p *XRefParser) GetTrailer() *TrailerDict {
	if len(p.trailers) > 0 {
		return p.trailers[0] // First trailer is the latest (most recent update)
	}
	return nil
}

// GetAllTrailers returns all trailer dictionaries in the Prev chain
func (p *XRefParser) GetAllTrailers() []*TrailerDict {
	return p.trailers
}

// GetEntryCount returns the total number of xref entries
func (p *XRefParser) GetEntryCount() int {
	count := 0
	for _, genMap := range p.entries {
		count += len(genMap)
	}
	return count
}

// GetObjectNumbers returns all object numbers that have xref entries
func (p *XRefParser) GetObjectNumbers() []int {
	numbers := make([]int, 0, len(p.entries))
	for objNum := range p.entries {
		numbers = append(numbers, objNum)
	}
	return numbers
}

// ValidateConsistency performs basic consistency checks on the xref table
func (p *XRefParser) ValidateConsistency() error {
	trailer := p.GetTrailer()
	if trailer == nil {
		return fmt.Errorf("no trailer found")
	}

	// Check if the Size field matches the number of entries
	actualCount := p.GetEntryCount()
	if trailer.Size > 0 && actualCount > trailer.Size {
		// This might be OK due to incremental updates, just warn
		fmt.Printf("Warning: xref entry count (%d) exceeds trailer Size (%d)\n",
			actualCount, trailer.Size)
	}

	// Check for essential references
	if trailer.Root == nil {
		return fmt.Errorf("missing Root reference in trailer")
	}

	return nil
}

// ResolveObject resolves an indirect object reference using the xref table
func (p *XRefParser) ResolveObject(objNum int, generation int) (PDFObject, error) {
	objID := ObjectID{Number: int64(objNum), Generation: int64(generation)}

	// Check cache first
	if cached, exists := p.objectCache[objID]; exists {
		return cached, nil
	}

	entry := p.GetEntry(objNum, generation)
	if entry == nil {
		return nil, fmt.Errorf("no xref entry found for object %d %d", objNum, generation)
	}

	switch entry.Type {
	case EntryFree:
		return &Null{}, nil

	case EntryInUse:
		// Seek to object location and parse
		if _, err := p.reader.Seek(entry.Offset, io.SeekStart); err != nil {
			return nil, fmt.Errorf("failed to seek to object %d %d at offset %d: %w",
				objNum, generation, entry.Offset, err)
		}

		// This would typically use the main PDF parser to read the object
		// For now, return a placeholder
		obj := &Keyword{Value: fmt.Sprintf("Object_%d_%d", objNum, generation)}
		p.objectCache[objID] = obj
		return obj, nil

	case EntryCompressed:
		// Handle compressed objects (object streams)
		return nil, fmt.Errorf("compressed object resolution not yet implemented for object %d %d",
			objNum, generation)

	default:
		return nil, fmt.Errorf("unknown entry type %d for object %d %d",
			entry.Type, objNum, generation)
	}
}

// GetStartXRef returns the startxref offset
func (p *XRefParser) GetStartXRef() int64 {
	return p.startxref
}

// GetPrevChain returns the chain of previous xref offsets
func (p *XRefParser) GetPrevChain() []int64 {
	return p.prevChain
}

// ClearCache clears the object resolution cache
func (p *XRefParser) ClearCache() {
	p.objectCache = make(map[ObjectID]PDFObject)
}

// GetCacheSize returns the number of cached objects
func (p *XRefParser) GetCacheSize() int {
	return len(p.objectCache)
}
