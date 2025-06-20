package custom

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/a3tai/mcp-pdf-reader/internal/pdf/xref"
)

// CustomPDFParser represents a PDF parser that handles PDF structure parsing
type CustomPDFParser struct {
	reader      io.ReadSeeker
	lexer       *PDFLexer
	version     string
	xrefTable   *CrossReferenceTable
	xrefParser  *xref.XRefParser
	trailer     *Dictionary
	catalog     *Dictionary
	objectCache map[ObjectID]PDFObject
	fileSize    int64
}

// NewCustomPDFParser creates a new PDF parser
func NewCustomPDFParser(reader io.ReadSeeker) *CustomPDFParser {
	return &CustomPDFParser{
		reader:      reader,
		objectCache: make(map[ObjectID]PDFObject),
		xrefTable:   NewCrossReferenceTable(),
		xrefParser:  xref.NewXRefParser(reader),
	}
}

// Parse parses the PDF document structure
func (p *CustomPDFParser) Parse() error {
	// Get file size
	if size, err := p.reader.Seek(0, io.SeekEnd); err != nil {
		return fmt.Errorf("failed to get file size: %w", err)
	} else {
		p.fileSize = size
	}

	// Reset to beginning
	if _, err := p.reader.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("failed to seek to start: %w", err)
	}

	// 1. Parse PDF header
	if err := p.parseHeader(); err != nil {
		return fmt.Errorf("header parse failed: %w", err)
	}

	// 2. Find and parse cross-reference table
	if err := p.parseXRefTable(); err != nil {
		return fmt.Errorf("xref parse failed: %w", err)
	}

	// 3. Parse trailer dictionary
	if err := p.parseTrailer(); err != nil {
		return fmt.Errorf("trailer parse failed: %w", err)
	}

	// 4. Load document catalog
	if err := p.loadCatalog(); err != nil {
		return fmt.Errorf("catalog load failed: %w", err)
	}

	return nil
}

// parseHeader parses the PDF header and extracts version information
func (p *CustomPDFParser) parseHeader() error {
	// Read first line for PDF header
	scanner := bufio.NewScanner(p.reader)
	if !scanner.Scan() {
		return NewParseError("failed to read PDF header", 0)
	}

	headerLine := scanner.Text()
	if !strings.HasPrefix(headerLine, PDFHeaderPattern) {
		return NewParseError("invalid PDF header", 0)
	}

	// Extract version
	p.version = strings.TrimPrefix(headerLine, PDFHeaderPattern)
	if p.version == "" {
		p.version = PDFVersion14 // Default to 1.4
	}

	return nil
}

// parseXRefTable locates and parses the cross-reference table using the new comprehensive parser
func (p *CustomPDFParser) parseXRefTable() error {
	// Find startxref keyword by reading from the end of file
	startXRefOffset, err := p.findStartXRef()
	if err != nil {
		return fmt.Errorf("failed to find startxref: %w", err)
	}

	// Use the new comprehensive xref parser
	err = p.xrefParser.ParseXRef(startXRefOffset)
	if err != nil {
		return fmt.Errorf("failed to parse xref table: %w", err)
	}

	// Validate the parsed xref structure
	err = p.xrefParser.ValidateConsistency()
	if err != nil {
		// Be liberal - warn but don't fail on consistency issues
		fmt.Printf("Warning: xref consistency check failed: %v\n", err)
	}

	// Convert xref entries to legacy format for compatibility
	p.convertXRefEntries()

	return nil
}

// convertXRefEntries converts new xref parser entries to legacy format for compatibility
func (p *CustomPDFParser) convertXRefEntries() {
	// Clear existing entries
	p.xrefTable = NewCrossReferenceTable()

	// Convert entries from new parser to legacy format
	for _, objNum := range p.xrefParser.GetObjectNumbers() {
		if entry := p.xrefParser.GetLatestEntry(objNum); entry != nil {
			legacyEntry := &XRefEntry{
				ObjectID:  ObjectID{Number: int64(objNum), Generation: int64(entry.Generation)},
				Offset:    entry.Offset,
				InUse:     entry.Type == xref.EntryInUse,
				EntryType: p.convertEntryType(entry.Type),
			}

			if entry.Type == xref.EntryCompressed {
				legacyEntry.StreamNum = entry.Offset
				legacyEntry.StreamIdx = int64(entry.StreamIndex)
			}

			p.xrefTable.AddEntry(legacyEntry)
		}
	}
}

// convertEntryType converts new xref entry type to legacy format
func (p *CustomPDFParser) convertEntryType(entryType xref.EntryType) XRefType {
	switch entryType {
	case xref.EntryFree:
		return XRefTypeFree
	case xref.EntryInUse:
		return XRefTypeNormal
	case xref.EntryCompressed:
		return XRefTypeCompressed
	default:
		return XRefTypeFree
	}
}

// findStartXRef finds the startxref offset by reading from the end of the file
func (p *CustomPDFParser) findStartXRef() (int64, error) {
	// Read last 1024 bytes of file
	readSize := int64(1024)
	if readSize > p.fileSize {
		readSize = p.fileSize
	}

	startPos := p.fileSize - readSize
	if _, err := p.reader.Seek(startPos, io.SeekStart); err != nil {
		return 0, fmt.Errorf("failed to seek to end of file: %w", err)
	}

	data := make([]byte, readSize)
	if _, err := io.ReadFull(p.reader, data); err != nil {
		return 0, fmt.Errorf("failed to read end of file: %w", err)
	}

	// Find "startxref" keyword
	content := string(data)
	startXRefIndex := strings.LastIndex(content, StartXRefKeyword)
	if startXRefIndex == -1 {
		return 0, NewParseError("startxref keyword not found", p.fileSize)
	}

	// Find the number after startxref
	afterKeyword := content[startXRefIndex+len(StartXRefKeyword):]
	lines := strings.Split(afterKeyword, "\n")
	if len(lines) < 2 {
		return 0, NewParseError("missing offset after startxref", p.fileSize)
	}

	offsetStr := strings.TrimSpace(lines[1])
	offset, err := strconv.ParseInt(offsetStr, 10, 64)
	if err != nil {
		return 0, NewParseError("invalid startxref offset", p.fileSize)
	}

	return offset, nil
}

// parseTrailer parses the trailer dictionary
func (p *CustomPDFParser) parseTrailer() error {
	// The "trailer" keyword has already been consumed in parseXRefTable
	// Parse trailer dictionary directly
	trailerObj, err := p.parseObject()
	if err != nil {
		return fmt.Errorf("failed to parse trailer dictionary: %w", err)
	}

	if trailerObj.Type() != TypeDictionary {
		return NewParseError("trailer must be a dictionary", p.lexer.GetPosition())
	}

	p.trailer = trailerObj.(*Dictionary)
	return nil
}

// loadCatalog loads the document catalog from the trailer
func (p *CustomPDFParser) loadCatalog() error {
	if p.trailer == nil {
		return NewParseError("trailer not parsed", 0)
	}

	// Get Root reference from trailer
	rootObj := p.trailer.Get("Root")
	if rootObj.Type() != TypeIndirectRef {
		return NewParseError("trailer Root must be an indirect reference", 0)
	}

	// Resolve the catalog object
	catalogObj, err := p.resolveIndirectObject(rootObj)
	if err != nil {
		return fmt.Errorf("failed to resolve catalog: %w", err)
	}

	if catalogObj.Type() != TypeDictionary {
		return NewParseError("catalog must be a dictionary", 0)
	}

	p.catalog = catalogObj.(*Dictionary)

	// Validate catalog type
	if p.catalog.GetName("Type") != "Catalog" {
		return NewParseError("invalid catalog type", 0)
	}

	return nil
}

// resolveIndirectObject resolves an indirect object reference
func (p *CustomPDFParser) resolveIndirectObject(obj PDFObject) (PDFObject, error) {
	if obj.Type() != TypeIndirectRef {
		return obj, nil // Not an indirect reference
	}

	ref := obj.(*IndirectRef)

	// Check cache first
	if cached, exists := p.objectCache[ref.ObjectID]; exists {
		return cached, nil
	}

	// Find object in xref table
	entry := p.xrefTable.GetEntry(ref.ObjectID)
	if entry == nil {
		return nil, NewParseError(fmt.Sprintf("object %s not found in xref table", ref.ObjectID), 0)
	}

	if !entry.InUse {
		return nil, NewParseError(fmt.Sprintf("object %s is not in use", ref.ObjectID), 0)
	}

	// Seek to object location
	if _, err := p.reader.Seek(entry.Offset, io.SeekStart); err != nil {
		return nil, fmt.Errorf("failed to seek to object %s: %w", ref.ObjectID, err)
	}

	// Create new lexer for object parsing
	p.lexer = NewPDFLexer(p.reader)

	// Parse indirect object
	indirectObj, err := p.parseIndirectObject()
	if err != nil {
		return nil, fmt.Errorf("failed to parse indirect object %s: %w", ref.ObjectID, err)
	}

	// Cache the resolved object
	p.objectCache[ref.ObjectID] = indirectObj.Object

	return indirectObj.Object, nil
}

// parseIndirectObject parses an indirect object definition
func (p *CustomPDFParser) parseIndirectObject() (*IndirectObject, error) {
	// Read object number
	numToken, err := p.lexer.NextToken()
	if err != nil {
		return nil, fmt.Errorf("failed to read object number: %w", err)
	}
	if numToken.Type != TokenNumber {
		return nil, NewParseError("expected object number", numToken.Pos)
	}

	objNum, err := strconv.ParseInt(numToken.Value, 10, 64)
	if err != nil {
		return nil, NewParseError("invalid object number", numToken.Pos)
	}

	// Read generation number
	genToken, err := p.lexer.NextToken()
	if err != nil {
		return nil, fmt.Errorf("failed to read generation number: %w", err)
	}
	if genToken.Type != TokenNumber {
		return nil, NewParseError("expected generation number", genToken.Pos)
	}

	generation, err := strconv.ParseInt(genToken.Value, 10, 64)
	if err != nil {
		return nil, NewParseError("invalid generation number", genToken.Pos)
	}

	// Expect "obj" keyword
	objToken, err := p.lexer.NextToken()
	if err != nil {
		return nil, fmt.Errorf("failed to read obj keyword: %w", err)
	}
	if objToken.Type != TokenObjStart {
		return nil, NewParseError("expected 'obj' keyword", objToken.Pos)
	}

	// Parse the object content
	obj, err := p.parseObject()
	if err != nil {
		return nil, fmt.Errorf("failed to parse object content: %w", err)
	}

	// Expect "endobj" keyword
	endObjToken, err := p.lexer.NextToken()
	if err != nil {
		return nil, fmt.Errorf("failed to read endobj keyword: %w", err)
	}
	if endObjToken.Type != TokenObjEnd {
		return nil, NewParseError("expected 'endobj' keyword", endObjToken.Pos)
	}

	return &IndirectObject{
		ID:     ObjectID{Number: objNum, Generation: generation},
		Object: obj,
	}, nil
}

// parseObject parses a PDF object of any type
func (p *CustomPDFParser) parseObject() (PDFObject, error) {
	token, err := p.lexer.NextToken()
	if err != nil {
		return nil, fmt.Errorf("failed to read token: %w", err)
	}

	switch token.Type {
	case TokenKeyword:
		switch token.Value {
		case "null":
			return &Null{}, nil
		case "true":
			return &Bool{Value: true}, nil
		case "false":
			return &Bool{Value: false}, nil
		default:
			return &Keyword{Value: token.Value}, nil
		}

	case TokenNumber:
		// Check if this is part of an indirect reference
		return p.parseNumberOrRef(token)

	case TokenString:
		return &String{Value: token.Value, IsHex: false}, nil

	case TokenHexString:
		return &String{Value: token.Value, IsHex: true}, nil

	case TokenName:
		return &Name{Value: token.Value}, nil

	case TokenArrayStart:
		return p.parseArray()

	case TokenDictStart:
		return p.parseDictionary()

	default:
		return nil, NewParseError(fmt.Sprintf("unexpected token type: %s", token.Type), token.Pos)
	}
}

// parseNumber parses a numeric object
func (p *CustomPDFParser) parseNumber(token Token) (PDFObject, error) {
	if strings.Contains(token.Value, ".") {
		// Real number
		val, err := strconv.ParseFloat(token.Value, 64)
		if err != nil {
			return nil, NewParseError("invalid real number", token.Pos)
		}
		return &Number{Value: val}, nil
	} else {
		// Integer
		val, err := strconv.ParseInt(token.Value, 10, 64)
		if err != nil {
			return nil, NewParseError("invalid integer", token.Pos)
		}
		return &Number{Value: val}, nil
	}
}

// parseArray parses a PDF array object
func (p *CustomPDFParser) parseArray() (PDFObject, error) {
	array := &Array{Elements: make([]PDFObject, 0)}

	for {
		token, err := p.lexer.NextToken()
		if err != nil {
			return nil, fmt.Errorf("failed to read array token: %w", err)
		}

		if token.Type == TokenArrayEnd {
			break
		}

		// Parse the token directly as an object
		obj, err := p.parseTokenAsObject(token)
		if err != nil {
			return nil, fmt.Errorf("failed to parse array element: %w", err)
		}

		array.Add(obj)
	}

	return array, nil
}

// parseDictionary parses a PDF dictionary object
func (p *CustomPDFParser) parseDictionary() (PDFObject, error) {
	dict := NewDictionary()

	for {
		token, err := p.lexer.NextToken()
		if err != nil {
			return nil, fmt.Errorf("failed to read dictionary token: %w", err)
		}

		if token.Type == TokenDictEnd {
			break
		}

		// Should be a name for the key
		if token.Type != TokenName {
			return nil, NewParseError("expected name for dictionary key", token.Pos)
		}

		key := token.Value

		// Parse the value
		value, err := p.parseObject()
		if err != nil {
			return nil, fmt.Errorf("failed to parse dictionary value for key %s: %w", key, err)
		}

		dict.Set(key, value)
	}

	// Check if this is a stream dictionary
	return p.checkForStream(dict)
}

// parseNumberOrRef parses a number or indirect reference
func (p *CustomPDFParser) parseNumberOrRef(numToken Token) (PDFObject, error) {
	// Parse the number first
	num, err := p.parseNumber(numToken)
	if err != nil {
		return nil, err
	}

	// Save current reader position
	pos, _ := p.reader.Seek(0, io.SeekCurrent)

	// Try to read next token
	token2, err := p.lexer.NextToken()
	if err != nil {
		// Just a number
		return num, nil
	}

	// Check if it's another number (generation)
	if token2.Type == TokenNumber {
		// Could be indirect reference, check for 'R'
		token3, err := p.lexer.NextToken()
		if err == nil && token3.Type == TokenIndirectRef {
			// It's an indirect reference!
			objNum := num.(*Number).Int()
			generation, _ := strconv.ParseInt(token2.Value, 10, 64)
			return &IndirectRef{
				ObjectID: ObjectID{Number: objNum, Generation: generation},
			}, nil
		}
	}

	// Not an indirect reference, restore position
	p.reader.Seek(pos, io.SeekStart)
	p.lexer = NewPDFLexer(p.reader)
	return num, nil
}

// checkForStream checks if a dictionary is followed by stream data
func (p *CustomPDFParser) checkForStream(dict *Dictionary) (PDFObject, error) {
	// Look ahead to see if next token is "stream"
	currentPos, _ := p.reader.Seek(0, io.SeekCurrent)

	token, err := p.lexer.NextToken()
	if err != nil || token.Type != TokenStreamStart {
		// Not a stream, restore position and return dictionary
		p.reader.Seek(currentPos, io.SeekStart)
		p.lexer = NewPDFLexer(p.reader)
		return dict, nil
	}

	// This is a stream, parse stream data
	length := dict.GetInt("Length")
	if length <= 0 {
		return nil, NewParseError("stream missing or invalid Length", token.Pos)
	}

	// Skip whitespace after "stream"
	// Create a buffered reader for ReadByte functionality
	bufReader := bufio.NewReader(p.reader)
	for {
		ch, err := bufReader.ReadByte()
		if err != nil {
			return nil, fmt.Errorf("failed to read after stream: %w", err)
		}
		if ch == '\n' {
			break
		}
		if ch == '\r' {
			// Check for CRLF
			if next, err := bufReader.ReadByte(); err == nil && next != '\n' {
				// Put back the byte if it's not LF
				bufReader.UnreadByte()
			}
			break
		}
	}

	// Read stream data
	data := make([]byte, length)
	if _, err := io.ReadFull(bufReader, data); err != nil {
		return nil, fmt.Errorf("failed to read stream data: %w", err)
	}

	// Expect "endstream"
	p.lexer = NewPDFLexer(bufReader)
	endToken, err := p.lexer.NextToken()
	if err != nil {
		return nil, fmt.Errorf("failed to read endstream: %w", err)
	}
	if endToken.Type != TokenStreamEnd {
		return nil, NewParseError("expected 'endstream'", endToken.Pos)
	}

	return &Stream{
		Dict:   dict,
		Data:   data,
		Length: length,
	}, nil
}

// GetVersion returns the PDF version
func (p *CustomPDFParser) GetVersion() string {
	return p.version
}

// GetCatalog returns the document catalog
func (p *CustomPDFParser) GetCatalog() *Dictionary {
	return p.catalog
}

// GetTrailer returns the trailer dictionary
func (p *CustomPDFParser) GetTrailer() *Dictionary {
	return p.trailer
}

// GetXRefTable returns the cross-reference table
func (p *CustomPDFParser) GetXRefTable() *CrossReferenceTable {
	return p.xrefTable
}

// GetObjectCache returns the object cache
func (p *CustomPDFParser) GetObjectCache() map[ObjectID]PDFObject {
	return p.objectCache
}

// ResolveIndirectObject resolves an indirect object reference (public method)
func (p *CustomPDFParser) ResolveIndirectObject(obj PDFObject) (PDFObject, error) {
	return p.resolveIndirectObject(obj)
}

// parseTokenAsObject converts a pre-read token into a PDF object
func (p *CustomPDFParser) parseTokenAsObject(token Token) (PDFObject, error) {
	switch token.Type {
	case TokenKeyword:
		switch token.Value {
		case "null":
			return &Null{}, nil
		case "true":
			return &Bool{Value: true}, nil
		case "false":
			return &Bool{Value: false}, nil
		default:
			return &Keyword{Value: token.Value}, nil
		}

	case TokenNumber:
		// Pre-read tokens can't be indirect references
		return p.parseNumber(token)

	case TokenString:
		return &String{Value: token.Value, IsHex: false}, nil

	case TokenHexString:
		return &String{Value: token.Value, IsHex: true}, nil

	case TokenName:
		return &Name{Value: token.Value}, nil

	case TokenArrayStart:
		return p.parseArray()

	case TokenDictStart:
		return p.parseDictionary()

	default:
		return nil, NewParseError(fmt.Sprintf("unexpected token type: %s", token.Type), token.Pos)
	}
}
