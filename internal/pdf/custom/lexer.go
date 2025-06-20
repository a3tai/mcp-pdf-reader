package custom

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strconv"
	"unicode"
)

// PDFLexer tokenizes PDF content
type PDFLexer struct {
	reader   *bufio.Reader
	position int64
	current  byte
	hasNext  bool
	error    error
}

// NewPDFLexer creates a new PDF lexer
func NewPDFLexer(reader io.Reader) *PDFLexer {
	bufReader := bufio.NewReader(reader)
	lexer := &PDFLexer{
		reader:   bufReader,
		position: -1, // Will be 0 after first advance
		hasNext:  true,
	}
	lexer.advance() // Read first character
	return lexer
}

// advance reads the next character from the input
func (l *PDFLexer) advance() {
	if !l.hasNext {
		return
	}

	ch, err := l.reader.ReadByte()
	if err != nil {
		if err == io.EOF {
			l.hasNext = false
			l.current = 0
		} else {
			l.error = err
			l.hasNext = false
		}
		return
	}

	l.current = ch
	l.position++
}

// peek looks at the next character without advancing
func (l *PDFLexer) peek() byte {
	if !l.hasNext {
		return 0
	}

	next, err := l.reader.Peek(1)
	if err != nil || len(next) == 0 {
		return 0
	}
	return next[0]
}

// peekN looks at the next N characters without advancing
func (l *PDFLexer) peekN(n int) []byte {
	if !l.hasNext {
		return nil
	}

	next, err := l.reader.Peek(n)
	if err != nil {
		return nil
	}
	return next
}

// skipWhitespace skips all whitespace characters
func (l *PDFLexer) skipWhitespace() {
	for l.hasNext && IsWhitespace(l.current) {
		l.advance()
	}
}

// skipComment skips a comment line starting with %
func (l *PDFLexer) skipComment() {
	if l.current != PercentSign {
		return
	}

	// Skip until end of line
	for l.hasNext && l.current != LineFeedChar && l.current != CarriageReturnChar {
		l.advance()
	}

	// Skip the line ending
	if l.hasNext && (l.current == LineFeedChar || l.current == CarriageReturnChar) {
		if l.current == CarriageReturnChar && l.peek() == LineFeedChar {
			l.advance() // Skip CR
		}
		l.advance() // Skip LF or remaining CR
	}
}

// NextToken returns the next token from the input
func (l *PDFLexer) NextToken() (Token, error) {
	if l.error != nil {
		return Token{Type: TokenError, Value: l.error.Error(), Pos: l.position}, l.error
	}

	// Skip whitespace and comments
	for l.hasNext {
		if IsWhitespace(l.current) {
			l.skipWhitespace()
		} else if l.current == PercentSign {
			l.skipComment()
		} else {
			break
		}
	}

	if !l.hasNext {
		return Token{Type: TokenEOF, Pos: l.position}, nil
	}

	startPos := l.position

	switch l.current {
	case LeftParen:
		return l.readLiteralString()
	case LeftAngle:
		if l.peek() == LeftAngle {
			return l.readDictionaryStart()
		}
		return l.readHexString()
	case RightAngle:
		if l.peek() == RightAngle {
			return l.readDictionaryEnd()
		}
		return Token{Type: TokenDelimiter, Value: string(l.current), Pos: startPos}, nil
	case LeftSquare:
		l.advance()
		return Token{Type: TokenArrayStart, Value: "[", Pos: startPos}, nil
	case RightSquare:
		l.advance()
		return Token{Type: TokenArrayEnd, Value: "]", Pos: startPos}, nil
	case Solidus:
		return l.readName()
	default:
		if unicode.IsDigit(rune(l.current)) || l.current == '+' || l.current == '-' || l.current == '.' {
			return l.readNumber()
		}
		return l.readKeyword()
	}
}

// readLiteralString reads a literal string enclosed in parentheses
func (l *PDFLexer) readLiteralString() (Token, error) {
	startPos := l.position
	var buffer bytes.Buffer

	l.advance() // Skip opening parenthesis
	parenCount := 1

	for l.hasNext && parenCount > 0 {
		ch := l.current

		if ch == LeftParen {
			parenCount++
			buffer.WriteByte(ch)
		} else if ch == RightParen {
			parenCount--
			if parenCount > 0 {
				buffer.WriteByte(ch)
			}
		} else if ch == '\\' {
			// Handle escape sequences
			l.advance()
			if !l.hasNext {
				break
			}

			switch l.current {
			case 'n':
				buffer.WriteByte('\n')
			case 'r':
				buffer.WriteByte('\r')
			case 't':
				buffer.WriteByte('\t')
			case 'b':
				buffer.WriteByte('\b')
			case 'f':
				buffer.WriteByte('\f')
			case '(':
				buffer.WriteByte('(')
			case ')':
				buffer.WriteByte(')')
			case '\\':
				buffer.WriteByte('\\')
			case LineFeedChar, CarriageReturnChar:
				// Line continuation - skip the line break
				if l.current == CarriageReturnChar && l.peek() == LineFeedChar {
					l.advance()
				}
			default:
				if unicode.IsDigit(rune(l.current)) {
					// Octal escape sequence
					octal := string(l.current)
					for i := 0; i < 2 && l.peek() != 0 && unicode.IsDigit(rune(l.peek())); i++ {
						l.advance()
						octal += string(l.current)
					}
					if val, err := strconv.ParseInt(octal, 8, 8); err == nil {
						buffer.WriteByte(byte(val))
					} else {
						buffer.WriteByte(l.current)
					}
				} else {
					buffer.WriteByte(l.current)
				}
			}
		} else {
			buffer.WriteByte(ch)
		}

		l.advance()
	}

	return Token{Type: TokenString, Value: buffer.String(), Pos: startPos}, nil
}

// readHexString reads a hexadecimal string enclosed in angle brackets
func (l *PDFLexer) readHexString() (Token, error) {
	startPos := l.position
	var buffer bytes.Buffer

	l.advance() // Skip opening angle bracket

	for l.hasNext && l.current != RightAngle {
		if !IsWhitespace(l.current) {
			if unicode.Is(unicode.ASCII_Hex_Digit, rune(l.current)) {
				buffer.WriteByte(l.current)
			} else {
				return Token{Type: TokenError, Value: "invalid hex digit", Pos: l.position},
					NewParseError("invalid hex digit in hex string", l.position)
			}
		}
		l.advance()
	}

	if l.hasNext && l.current == RightAngle {
		l.advance() // Skip closing angle bracket
	}

	// Ensure even number of hex digits
	hexStr := buffer.String()
	if len(hexStr)%2 == 1 {
		hexStr += "0"
	}

	return Token{Type: TokenHexString, Value: hexStr, Pos: startPos}, nil
}

// readDictionaryStart reads the << dictionary start delimiter
func (l *PDFLexer) readDictionaryStart() (Token, error) {
	startPos := l.position
	l.advance() // Skip first <
	l.advance() // Skip second <
	return Token{Type: TokenDictStart, Value: "<<", Pos: startPos}, nil
}

// readDictionaryEnd reads the >> dictionary end delimiter
func (l *PDFLexer) readDictionaryEnd() (Token, error) {
	startPos := l.position
	l.advance() // Skip first >
	l.advance() // Skip second >
	return Token{Type: TokenDictEnd, Value: ">>", Pos: startPos}, nil
}

// readName reads a name object starting with /
func (l *PDFLexer) readName() (Token, error) {
	startPos := l.position
	var buffer bytes.Buffer

	l.advance() // Skip the solidus

	for l.hasNext && IsRegular(l.current) {
		if l.current == '#' {
			// Hex escape in name
			l.advance()
			if l.hasNext && unicode.Is(unicode.ASCII_Hex_Digit, rune(l.current)) {
				hex1 := l.current
				l.advance()
				if l.hasNext && unicode.Is(unicode.ASCII_Hex_Digit, rune(l.current)) {
					hex2 := l.current
					if val, err := strconv.ParseInt(string([]byte{hex1, hex2}), 16, 8); err == nil {
						buffer.WriteByte(byte(val))
					} else {
						buffer.WriteByte('#')
						buffer.WriteByte(hex1)
						buffer.WriteByte(hex2)
					}
					l.advance()
				} else {
					buffer.WriteByte('#')
					buffer.WriteByte(hex1)
				}
			} else {
				buffer.WriteByte('#')
			}
		} else {
			buffer.WriteByte(l.current)
			l.advance()
		}
	}

	return Token{Type: TokenName, Value: buffer.String(), Pos: startPos}, nil
}

// readNumber reads a numeric value (integer or real)
func (l *PDFLexer) readNumber() (Token, error) {
	startPos := l.position
	var buffer bytes.Buffer

	// Handle sign
	if l.current == '+' || l.current == '-' {
		buffer.WriteByte(l.current)
		l.advance()
	}

	// Read digits before decimal point
	for l.hasNext && unicode.IsDigit(rune(l.current)) {
		buffer.WriteByte(l.current)
		l.advance()
	}

	// Check for decimal point
	if l.hasNext && l.current == '.' {
		buffer.WriteByte(l.current)
		l.advance()

		// Read digits after decimal point
		for l.hasNext && unicode.IsDigit(rune(l.current)) {
			buffer.WriteByte(l.current)
			l.advance()
		}
	}

	return Token{Type: TokenNumber, Value: buffer.String(), Pos: startPos}, nil
}

// readKeyword reads a keyword or identifier
func (l *PDFLexer) readKeyword() (Token, error) {
	startPos := l.position
	var buffer bytes.Buffer

	for l.hasNext && IsRegular(l.current) {
		buffer.WriteByte(l.current)
		l.advance()
	}

	keyword := buffer.String()

	// Check for special keywords
	switch keyword {
	case "true", "false":
		return Token{Type: TokenKeyword, Value: keyword, Pos: startPos}, nil
	case "null":
		return Token{Type: TokenKeyword, Value: keyword, Pos: startPos}, nil
	case "R":
		return Token{Type: TokenIndirectRef, Value: keyword, Pos: startPos}, nil
	case ObjKeyword:
		return Token{Type: TokenObjStart, Value: keyword, Pos: startPos}, nil
	case EndObjKeyword:
		return Token{Type: TokenObjEnd, Value: keyword, Pos: startPos}, nil
	case StreamKeyword:
		return Token{Type: TokenStreamStart, Value: keyword, Pos: startPos}, nil
	case EndStreamKeyword:
		return Token{Type: TokenStreamEnd, Value: keyword, Pos: startPos}, nil
	case XRefKeyword:
		return Token{Type: TokenXRefKeyword, Value: keyword, Pos: startPos}, nil
	case TrailerKeyword:
		return Token{Type: TokenTrailerKeyword, Value: keyword, Pos: startPos}, nil
	case StartXRefKeyword:
		return Token{Type: TokenStartXRef, Value: keyword, Pos: startPos}, nil
	default:
		return Token{Type: TokenKeyword, Value: keyword, Pos: startPos}, nil
	}
}

// GetPosition returns the current position in the input
func (l *PDFLexer) GetPosition() int64 {
	return l.position
}

// HasNext returns true if there are more characters to read
func (l *PDFLexer) HasNext() bool {
	return l.hasNext
}

// GetError returns any error that occurred during lexing
func (l *PDFLexer) GetError() error {
	return l.error
}

// Reset resets the lexer to read from a new input
func (l *PDFLexer) Reset(reader io.Reader) {
	l.reader = bufio.NewReader(reader)
	l.position = -1
	l.hasNext = true
	l.error = nil
	l.advance()
}

// ReadUntil reads characters until a specific delimiter is found
func (l *PDFLexer) ReadUntil(delimiter byte) ([]byte, error) {
	var buffer bytes.Buffer

	for l.hasNext && l.current != delimiter {
		buffer.WriteByte(l.current)
		l.advance()
	}

	if l.hasNext && l.current == delimiter {
		l.advance() // Skip the delimiter
	}

	return buffer.Bytes(), l.error
}

// ReadBytes reads exactly n bytes
func (l *PDFLexer) ReadBytes(n int) ([]byte, error) {
	data := make([]byte, n)

	for i := 0; i < n && l.hasNext; i++ {
		data[i] = l.current
		l.advance()
	}

	return data, l.error
}

// SkipBytes skips exactly n bytes
func (l *PDFLexer) SkipBytes(n int) error {
	for i := 0; i < n && l.hasNext; i++ {
		l.advance()
	}
	return l.error
}

// ExpectKeyword checks if the next token is the expected keyword
func (l *PDFLexer) ExpectKeyword(expected string) error {
	token, err := l.NextToken()
	if err != nil {
		return err
	}

	if token.Type != TokenKeyword || token.Value != expected {
		return NewParseError(fmt.Sprintf("expected keyword '%s', got '%s'", expected, token.Value), token.Pos)
	}

	return nil
}

// ExpectToken checks if the next token is of the expected type
func (l *PDFLexer) ExpectToken(expectedType TokenType) (Token, error) {
	token, err := l.NextToken()
	if err != nil {
		return token, err
	}

	if token.Type != expectedType {
		return token, NewParseError(fmt.Sprintf("expected token type %s, got %s", expectedType, token.Type), token.Pos)
	}

	return token, nil
}
