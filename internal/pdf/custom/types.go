package custom

import (
	"fmt"
	"io"
	"strconv"
	"strings"
)

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

func (t ObjectType) String() string {
	switch t {
	case TypeNull:
		return "null"
	case TypeBool:
		return "bool"
	case TypeNumber:
		return "number"
	case TypeString:
		return "string"
	case TypeName:
		return "name"
	case TypeArray:
		return "array"
	case TypeDictionary:
		return "dictionary"
	case TypeStream:
		return "stream"
	case TypeIndirectRef:
		return "indirect_ref"
	case TypeKeyword:
		return "keyword"
	default:
		return "unknown"
	}
}

// PDFObject is the base interface for all PDF objects
type PDFObject interface {
	Type() ObjectType
	String() string
}

// ObjectID represents a PDF object identifier
type ObjectID struct {
	Number     int64 // Object number
	Generation int64 // Generation number
}

func (id ObjectID) String() string {
	return fmt.Sprintf("%d %d", id.Number, id.Generation)
}

func (id ObjectID) IsValid() bool {
	return id.Number > 0 && id.Generation >= 0
}

// Null represents a PDF null object
type Null struct{}

func (n *Null) Type() ObjectType { return TypeNull }
func (n *Null) String() string   { return "null" }

// Bool represents a PDF boolean object
type Bool struct {
	Value bool
}

func (b *Bool) Type() ObjectType { return TypeBool }
func (b *Bool) String() string {
	if b.Value {
		return "true"
	}
	return "false"
}

// Number represents a PDF numeric object (integer or real)
type Number struct {
	Value interface{} // int64 or float64
}

func (n *Number) Type() ObjectType { return TypeNumber }
func (n *Number) String() string {
	switch v := n.Value.(type) {
	case int64:
		return strconv.FormatInt(v, 10)
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	default:
		return "0"
	}
}

func (n *Number) Int() int64 {
	switch v := n.Value.(type) {
	case int64:
		return v
	case float64:
		return int64(v)
	default:
		return 0
	}
}

func (n *Number) Float() float64 {
	switch v := n.Value.(type) {
	case int64:
		return float64(v)
	case float64:
		return v
	default:
		return 0.0
	}
}

// String represents a PDF string object
type String struct {
	Value    string
	IsHex    bool // true for hex strings, false for literal strings
	Encoding string
}

func (s *String) Type() ObjectType { return TypeString }
func (s *String) String() string {
	if s.IsHex {
		return fmt.Sprintf("<%s>", s.Value)
	}
	return fmt.Sprintf("(%s)", s.Value)
}

// Name represents a PDF name object
type Name struct {
	Value string
}

func (n *Name) Type() ObjectType { return TypeName }
func (n *Name) String() string   { return "/" + n.Value }

// Array represents a PDF array object
type Array struct {
	Elements []PDFObject
}

func (a *Array) Type() ObjectType { return TypeArray }
func (a *Array) String() string {
	var parts []string
	for _, elem := range a.Elements {
		parts = append(parts, elem.String())
	}
	return "[" + strings.Join(parts, " ") + "]"
}

func (a *Array) Len() int {
	return len(a.Elements)
}

func (a *Array) Get(index int) PDFObject {
	if index >= 0 && index < len(a.Elements) {
		return a.Elements[index]
	}
	return &Null{}
}

func (a *Array) Add(obj PDFObject) {
	a.Elements = append(a.Elements, obj)
}

// Dictionary represents a PDF dictionary object
type Dictionary struct {
	Keys   []Name // Maintains insertion order
	Values map[string]PDFObject
}

func NewDictionary() *Dictionary {
	return &Dictionary{
		Keys:   make([]Name, 0),
		Values: make(map[string]PDFObject),
	}
}

func (d *Dictionary) Type() ObjectType { return TypeDictionary }
func (d *Dictionary) String() string {
	var parts []string
	for _, key := range d.Keys {
		value := d.Values[key.Value]
		parts = append(parts, key.String()+" "+value.String())
	}
	return "<<" + strings.Join(parts, " ") + ">>"
}

func (d *Dictionary) Get(key string) PDFObject {
	if obj, exists := d.Values[key]; exists {
		return obj
	}
	return &Null{}
}

func (d *Dictionary) Set(key string, value PDFObject) {
	if _, exists := d.Values[key]; !exists {
		d.Keys = append(d.Keys, Name{Value: key})
	}
	d.Values[key] = value
}

func (d *Dictionary) Has(key string) bool {
	_, exists := d.Values[key]
	return exists
}

func (d *Dictionary) Remove(key string) {
	if _, exists := d.Values[key]; exists {
		delete(d.Values, key)
		// Remove from keys slice
		for i, k := range d.Keys {
			if k.Value == key {
				d.Keys = append(d.Keys[:i], d.Keys[i+1:]...)
				break
			}
		}
	}
}

func (d *Dictionary) Len() int {
	return len(d.Keys)
}

// Convenience methods for common types
func (d *Dictionary) GetString(key string) string {
	if obj := d.Get(key); obj.Type() == TypeString {
		return obj.(*String).Value
	}
	return ""
}

func (d *Dictionary) GetInt(key string) int64 {
	if obj := d.Get(key); obj.Type() == TypeNumber {
		return obj.(*Number).Int()
	}
	return 0
}

func (d *Dictionary) GetFloat(key string) float64 {
	if obj := d.Get(key); obj.Type() == TypeNumber {
		return obj.(*Number).Float()
	}
	return 0.0
}

func (d *Dictionary) GetBool(key string) bool {
	if obj := d.Get(key); obj.Type() == TypeBool {
		return obj.(*Bool).Value
	}
	return false
}

func (d *Dictionary) GetName(key string) string {
	if obj := d.Get(key); obj.Type() == TypeName {
		return obj.(*Name).Value
	}
	return ""
}

func (d *Dictionary) GetArray(key string) *Array {
	if obj := d.Get(key); obj.Type() == TypeArray {
		return obj.(*Array)
	}
	return &Array{}
}

func (d *Dictionary) GetDictionary(key string) *Dictionary {
	if obj := d.Get(key); obj.Type() == TypeDictionary {
		return obj.(*Dictionary)
	}
	return NewDictionary()
}

// Stream represents a PDF stream object
type Stream struct {
	Dict   *Dictionary
	Data   []byte
	Offset int64 // File offset where stream data starts
	Length int64 // Length of stream data
}

func (s *Stream) Type() ObjectType { return TypeStream }
func (s *Stream) String() string {
	return fmt.Sprintf("%s\nstream\n[%d bytes]\nendstream", s.Dict.String(), len(s.Data))
}

func (s *Stream) GetFilter() []string {
	filterObj := s.Dict.Get("Filter")
	if filterObj.Type() == TypeNull {
		return nil
	}

	var filters []string
	if filterObj.Type() == TypeName {
		filters = append(filters, filterObj.(*Name).Value)
	} else if filterObj.Type() == TypeArray {
		arr := filterObj.(*Array)
		for _, elem := range arr.Elements {
			if elem.Type() == TypeName {
				filters = append(filters, elem.(*Name).Value)
			}
		}
	}
	return filters
}

func (s *Stream) GetLength() int64 {
	if s.Length > 0 {
		return s.Length
	}
	lengthObj := s.Dict.Get("Length")
	if lengthObj.Type() == TypeNumber {
		return lengthObj.(*Number).Int()
	}
	return int64(len(s.Data))
}

// IndirectRef represents an indirect object reference
type IndirectRef struct {
	ObjectID ObjectID
}

func (r *IndirectRef) Type() ObjectType { return TypeIndirectRef }
func (r *IndirectRef) String() string   { return fmt.Sprintf("%s R", r.ObjectID.String()) }

// Keyword represents a PDF keyword/operator
type Keyword struct {
	Value string
}

func (k *Keyword) Type() ObjectType { return TypeKeyword }
func (k *Keyword) String() string   { return k.Value }

// Token represents a lexical token in PDF content
type Token struct {
	Type  TokenType
	Value string
	Pos   int64 // Position in stream
}

// TokenType represents the type of a lexical token
type TokenType int

const (
	TokenEOF TokenType = iota
	TokenError
	TokenWhitespace
	TokenComment
	TokenNumber
	TokenString
	TokenHexString
	TokenName
	TokenKeyword
	TokenDelimiter
	TokenArrayStart     // [
	TokenArrayEnd       // ]
	TokenDictStart      // <<
	TokenDictEnd        // >>
	TokenStreamStart    // stream
	TokenStreamEnd      // endstream
	TokenObjStart       // obj
	TokenObjEnd         // endobj
	TokenIndirectRef    // R
	TokenXRefKeyword    // xref
	TokenTrailerKeyword // trailer
	TokenStartXRef      // startxref
)

func (t TokenType) String() string {
	switch t {
	case TokenEOF:
		return "EOF"
	case TokenError:
		return "ERROR"
	case TokenWhitespace:
		return "WHITESPACE"
	case TokenComment:
		return "COMMENT"
	case TokenNumber:
		return "NUMBER"
	case TokenString:
		return "STRING"
	case TokenHexString:
		return "HEXSTRING"
	case TokenName:
		return "NAME"
	case TokenKeyword:
		return "KEYWORD"
	case TokenDelimiter:
		return "DELIMITER"
	case TokenArrayStart:
		return "ARRAY_START"
	case TokenArrayEnd:
		return "ARRAY_END"
	case TokenDictStart:
		return "DICT_START"
	case TokenDictEnd:
		return "DICT_END"
	case TokenStreamStart:
		return "STREAM_START"
	case TokenStreamEnd:
		return "STREAM_END"
	case TokenObjStart:
		return "OBJ_START"
	case TokenObjEnd:
		return "OBJ_END"
	case TokenIndirectRef:
		return "INDIRECT_REF"
	case TokenXRefKeyword:
		return "XREF"
	case TokenTrailerKeyword:
		return "TRAILER"
	case TokenStartXRef:
		return "STARTXREF"
	default:
		return "UNKNOWN"
	}
}

// XRefEntry represents an entry in the cross-reference table
type XRefEntry struct {
	ObjectID  ObjectID
	Offset    int64    // File offset of the object
	InUse     bool     // Whether the object is in use
	EntryType XRefType // Type of xref entry (normal, compressed, free)
	StreamNum int64    // For compressed objects, the object stream number
	StreamIdx int64    // For compressed objects, the index within the stream
}

// XRefType represents the type of cross-reference entry
type XRefType int

const (
	XRefTypeFree XRefType = iota
	XRefTypeNormal
	XRefTypeCompressed
)

func (t XRefType) String() string {
	switch t {
	case XRefTypeFree:
		return "free"
	case XRefTypeNormal:
		return "normal"
	case XRefTypeCompressed:
		return "compressed"
	default:
		return "unknown"
	}
}

// CrossReferenceTable represents the PDF cross-reference table
type CrossReferenceTable struct {
	Entries map[ObjectID]*XRefEntry
	MaxObj  int64 // Highest object number
}

func NewCrossReferenceTable() *CrossReferenceTable {
	return &CrossReferenceTable{
		Entries: make(map[ObjectID]*XRefEntry),
		MaxObj:  0,
	}
}

func (xref *CrossReferenceTable) AddEntry(entry *XRefEntry) {
	xref.Entries[entry.ObjectID] = entry
	if entry.ObjectID.Number > xref.MaxObj {
		xref.MaxObj = entry.ObjectID.Number
	}
}

func (xref *CrossReferenceTable) GetEntry(objID ObjectID) *XRefEntry {
	return xref.Entries[objID]
}

func (xref *CrossReferenceTable) HasEntry(objID ObjectID) bool {
	_, exists := xref.Entries[objID]
	return exists
}

func (xref *CrossReferenceTable) Count() int {
	return len(xref.Entries)
}

// IndirectObject represents an indirect object with its ID and content
type IndirectObject struct {
	ID     ObjectID
	Object PDFObject
}

func (io *IndirectObject) String() string {
	return fmt.Sprintf("%s obj\n%s\nendobj", io.ID.String(), io.Object.String())
}

// PDFReader interface for reading PDF content
type PDFReader interface {
	io.ReadSeeker
	Peek(n int) ([]byte, error)
	ReadByte() (byte, error)
	UnreadByte() error
}

// Error types for PDF parsing
type ParseError struct {
	Message  string
	Position int64
	Context  string
}

func (e *ParseError) Error() string {
	if e.Position >= 0 {
		return fmt.Sprintf("PDF parse error at position %d: %s", e.Position, e.Message)
	}
	return fmt.Sprintf("PDF parse error: %s", e.Message)
}

func NewParseError(msg string, pos int64) *ParseError {
	return &ParseError{
		Message:  msg,
		Position: pos,
	}
}

func NewParseErrorWithContext(msg string, pos int64, context string) *ParseError {
	return &ParseError{
		Message:  msg,
		Position: pos,
		Context:  context,
	}
}

// Constants for PDF parsing
const (
	// PDF version patterns
	PDFHeaderPattern = "%PDF-"
	PDFVersion10     = "1.0"
	PDFVersion11     = "1.1"
	PDFVersion12     = "1.2"
	PDFVersion13     = "1.3"
	PDFVersion14     = "1.4"
	PDFVersion15     = "1.5"
	PDFVersion16     = "1.6"
	PDFVersion17     = "1.7"

	// PDF delimiters and keywords
	ObjKeyword       = "obj"
	EndObjKeyword    = "endobj"
	StreamKeyword    = "stream"
	EndStreamKeyword = "endstream"
	XRefKeyword      = "xref"
	TrailerKeyword   = "trailer"
	StartXRefKeyword = "startxref"

	// PDF whitespace characters
	NullChar           = '\000'
	TabChar            = '\t'
	LineFeedChar       = '\n'
	FormFeedChar       = '\f'
	CarriageReturnChar = '\r'
	SpaceChar          = ' '

	// PDF delimiters
	LeftParen   = '('
	RightParen  = ')'
	LeftAngle   = '<'
	RightAngle  = '>'
	LeftSquare  = '['
	RightSquare = ']'
	LeftCurly   = '{'
	RightCurly  = '}'
	Solidus     = '/'
	PercentSign = '%'
)

// IsWhitespace checks if a character is PDF whitespace
func IsWhitespace(ch byte) bool {
	return ch == NullChar || ch == TabChar || ch == LineFeedChar ||
		ch == FormFeedChar || ch == CarriageReturnChar || ch == SpaceChar
}

// IsDelimiter checks if a character is a PDF delimiter
func IsDelimiter(ch byte) bool {
	return ch == LeftParen || ch == RightParen || ch == LeftAngle || ch == RightAngle ||
		ch == LeftSquare || ch == RightSquare || ch == LeftCurly || ch == RightCurly ||
		ch == Solidus || ch == PercentSign
}

// IsRegular checks if a character is a regular character (not whitespace or delimiter)
func IsRegular(ch byte) bool {
	return !IsWhitespace(ch) && !IsDelimiter(ch)
}
