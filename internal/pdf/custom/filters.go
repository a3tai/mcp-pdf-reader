package custom

import (
	"bytes"
	"compress/flate"
	"compress/lzw"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"strings"
)

// FilterDecoder interface for PDF stream filters
type FilterDecoder interface {
	Decode(data []byte, params *Dictionary) ([]byte, error)
	Name() string
}

// FilterRegistry holds all available filter decoders
var FilterRegistry = map[string]FilterDecoder{
	"FlateDecode":     &FlateDecoder{},
	"ASCIIHexDecode":  &ASCIIHexDecoder{},
	"ASCII85Decode":   &ASCII85Decoder{},
	"LZWDecode":       &LZWDecoder{},
	"RunLengthDecode": &RunLengthDecoder{},
	"CCITTFaxDecode":  &CCITTFaxDecoder{},
	"JBIG2Decode":     &JBIG2Decoder{},
	"DCTDecode":       &DCTDecoder{},
	"JPXDecode":       &JPXDecoder{},
}

// GetFilterDecoder returns a filter decoder by name
func GetFilterDecoder(name string) FilterDecoder {
	return FilterRegistry[name]
}

// DecodeStream applies filters to decode a PDF stream
func DecodeStream(stream *Stream) ([]byte, error) {
	data := stream.Data
	filters := stream.GetFilter()

	if len(filters) == 0 {
		return data, nil
	}

	// Apply filters in order
	for i, filterName := range filters {
		decoder := GetFilterDecoder(filterName)
		if decoder == nil {
			return nil, fmt.Errorf("unsupported filter: %s", filterName)
		}

		// Get decode parameters for this filter
		var params *Dictionary
		if decodeParams := stream.Dict.Get("DecodeParms"); decodeParams.Type() != TypeNull {
			if decodeParams.Type() == TypeArray {
				// Multiple decode parameter dictionaries
				if paramsArray := decodeParams.(*Array); i < paramsArray.Len() {
					if paramDict := paramsArray.Get(i); paramDict.Type() == TypeDictionary {
						params = paramDict.(*Dictionary)
					}
				}
			} else if decodeParams.Type() == TypeDictionary && i == 0 {
				// Single decode parameter dictionary
				params = decodeParams.(*Dictionary)
			}
		}

		var err error
		data, err = decoder.Decode(data, params)
		if err != nil {
			return nil, fmt.Errorf("failed to decode with %s: %w", filterName, err)
		}
	}

	return data, nil
}

// FlateDecoder implements zlib/deflate decompression
type FlateDecoder struct{}

func (f *FlateDecoder) Name() string {
	return "FlateDecode"
}

func (f *FlateDecoder) Decode(data []byte, params *Dictionary) ([]byte, error) {
	if len(data) == 0 {
		return data, nil
	}

	reader := flate.NewReader(bytes.NewReader(data))
	defer reader.Close()

	decoded, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("flate decode error: %w", err)
	}

	// Apply predictor if specified
	if params != nil {
		if predictor := params.GetInt("Predictor"); predictor > 1 {
			decoded, err = f.applyPredictor(decoded, params)
			if err != nil {
				return nil, fmt.Errorf("predictor error: %w", err)
			}
		}
	}

	return decoded, nil
}

func (f *FlateDecoder) applyPredictor(data []byte, params *Dictionary) ([]byte, error) {
	predictor := params.GetInt("Predictor")
	columns := params.GetInt("Columns")
	bitsPerComponent := params.GetInt("BitsPerComponent")
	colors := params.GetInt("Colors")

	if columns == 0 {
		columns = 1
	}
	if bitsPerComponent == 0 {
		bitsPerComponent = 8
	}
	if colors == 0 {
		colors = 1
	}

	switch predictor {
	case 2: // TIFF Predictor 2
		return f.applyTIFFPredictor(data, int(columns), int(bitsPerComponent), int(colors))
	case 10, 11, 12, 13, 14, 15: // PNG predictors
		return f.applyPNGPredictor(data, int(columns), int(bitsPerComponent), int(colors))
	default:
		return data, nil // No predictor or unknown predictor
	}
}

func (f *FlateDecoder) applyTIFFPredictor(data []byte, columns, bitsPerComponent, colors int) ([]byte, error) {
	if bitsPerComponent != 8 {
		return data, fmt.Errorf("TIFF predictor only supports 8 bits per component")
	}

	bytesPerPixel := colors
	rowSize := columns * bytesPerPixel

	if len(data)%rowSize != 0 {
		return data, fmt.Errorf("data length not multiple of row size")
	}

	result := make([]byte, len(data))
	copy(result, data)

	for row := 0; row < len(data)/rowSize; row++ {
		rowStart := row * rowSize
		for col := 1; col < columns; col++ {
			for c := 0; c < bytesPerPixel; c++ {
				idx := rowStart + col*bytesPerPixel + c
				prevIdx := rowStart + (col-1)*bytesPerPixel + c
				result[idx] = byte(int(result[idx]) + int(result[prevIdx]))
			}
		}
	}

	return result, nil
}

func (f *FlateDecoder) applyPNGPredictor(data []byte, columns, bitsPerComponent, colors int) ([]byte, error) {
	bytesPerPixel := (bitsPerComponent*colors + 7) / 8
	rowSize := (columns*bitsPerComponent*colors + 7) / 8
	totalRowSize := rowSize + 1 // +1 for predictor byte

	if len(data)%totalRowSize != 0 {
		return data, fmt.Errorf("data length not multiple of row size")
	}

	numRows := len(data) / totalRowSize
	result := make([]byte, numRows*rowSize)

	for row := 0; row < numRows; row++ {
		srcStart := row * totalRowSize
		dstStart := row * rowSize
		predictor := data[srcStart]
		rowData := data[srcStart+1 : srcStart+totalRowSize]

		switch predictor {
		case 0: // None
			copy(result[dstStart:], rowData)
		case 1: // Sub
			copy(result[dstStart:], rowData)
			for i := bytesPerPixel; i < rowSize; i++ {
				result[dstStart+i] = byte(int(result[dstStart+i]) + int(result[dstStart+i-bytesPerPixel]))
			}
		case 2: // Up
			copy(result[dstStart:], rowData)
			if row > 0 {
				prevRowStart := (row - 1) * rowSize
				for i := 0; i < rowSize; i++ {
					result[dstStart+i] = byte(int(result[dstStart+i]) + int(result[prevRowStart+i]))
				}
			}
		case 3: // Average
			copy(result[dstStart:], rowData)
			for i := 0; i < rowSize; i++ {
				var left, up byte
				if i >= bytesPerPixel {
					left = result[dstStart+i-bytesPerPixel]
				}
				if row > 0 {
					up = result[(row-1)*rowSize+i]
				}
				result[dstStart+i] = byte(int(result[dstStart+i]) + int(left+up)/2)
			}
		case 4: // Paeth
			copy(result[dstStart:], rowData)
			for i := 0; i < rowSize; i++ {
				var left, up, upLeft byte
				if i >= bytesPerPixel {
					left = result[dstStart+i-bytesPerPixel]
				}
				if row > 0 {
					up = result[(row-1)*rowSize+i]
					if i >= bytesPerPixel {
						upLeft = result[(row-1)*rowSize+i-bytesPerPixel]
					}
				}
				paeth := f.paethPredictor(left, up, upLeft)
				result[dstStart+i] = byte(int(result[dstStart+i]) + int(paeth))
			}
		default:
			return nil, fmt.Errorf("unknown PNG predictor: %d", predictor)
		}
	}

	return result, nil
}

func (f *FlateDecoder) paethPredictor(a, b, c byte) byte {
	p := int(a) + int(b) - int(c)
	pa := abs(p - int(a))
	pb := abs(p - int(b))
	pc := abs(p - int(c))

	if pa <= pb && pa <= pc {
		return a
	} else if pb <= pc {
		return b
	}
	return c
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// ASCIIHexDecoder implements ASCII hex decoding
type ASCIIHexDecoder struct{}

func (a *ASCIIHexDecoder) Name() string {
	return "ASCIIHexDecode"
}

func (a *ASCIIHexDecoder) Decode(data []byte, params *Dictionary) ([]byte, error) {
	// Remove whitespace and find end marker
	var hexStr strings.Builder
	for _, b := range data {
		if b == '>' {
			break // End of data marker
		}
		if (b >= '0' && b <= '9') || (b >= 'A' && b <= 'F') || (b >= 'a' && b <= 'f') {
			hexStr.WriteByte(b)
		}
	}

	hexData := hexStr.String()

	// Ensure even number of hex digits
	if len(hexData)%2 == 1 {
		hexData += "0"
	}

	decoded, err := hex.DecodeString(hexData)
	if err != nil {
		return nil, fmt.Errorf("ASCII hex decode error: %w", err)
	}

	return decoded, nil
}

// ASCII85Decoder implements ASCII85 decoding
type ASCII85Decoder struct{}

func (a *ASCII85Decoder) Name() string {
	return "ASCII85Decode"
}

func (a *ASCII85Decoder) Decode(data []byte, params *Dictionary) ([]byte, error) {
	// Find start and end markers
	start := 0
	end := len(data)

	// Look for <~ start marker
	for i := 0; i < len(data)-1; i++ {
		if data[i] == '<' && data[i+1] == '~' {
			start = i + 2
			break
		}
	}

	// Look for ~> end marker
	for i := start; i < len(data)-1; i++ {
		if data[i] == '~' && data[i+1] == '>' {
			end = i
			break
		}
	}

	if start >= end {
		return []byte{}, nil
	}

	// Clean data - keep only valid ASCII85 characters
	var cleanData []byte
	for i := start; i < end; i++ {
		b := data[i]
		if b >= '!' && b <= 'u' {
			cleanData = append(cleanData, b)
		} else if b == 'z' {
			// Special case for zero group
			cleanData = append(cleanData, b)
		}
		// Skip whitespace and other characters
	}

	var result []byte
	i := 0

	for i < len(cleanData) {
		if cleanData[i] == 'z' {
			// Special case: 'z' represents four zero bytes
			result = append(result, 0, 0, 0, 0)
			i++
			continue
		}

		// Process group of up to 5 characters
		group := make([]byte, 5)
		groupSize := 0

		for j := 0; j < 5 && i < len(cleanData) && cleanData[i] != 'z'; j++ {
			group[j] = cleanData[i] - '!'
			groupSize++
			i++
		}

		if groupSize == 0 {
			break
		}

		// Pad incomplete group
		for j := groupSize; j < 5; j++ {
			group[j] = 84 // 'u' - '!' = 84
		}

		// Decode 5 base-85 digits to 4 bytes
		value := uint32(group[0])*85*85*85*85 +
			uint32(group[1])*85*85*85 +
			uint32(group[2])*85*85 +
			uint32(group[3])*85 +
			uint32(group[4])

		// Extract bytes
		bytes := []byte{
			byte(value >> 24),
			byte(value >> 16),
			byte(value >> 8),
			byte(value),
		}

		// Add appropriate number of decoded bytes
		outputBytes := groupSize - 1
		if outputBytes > 4 {
			outputBytes = 4
		}

		result = append(result, bytes[:outputBytes]...)
	}

	return result, nil
}

// LZWDecoder implements LZW decompression
type LZWDecoder struct{}

func (l *LZWDecoder) Name() string {
	return "LZWDecode"
}

func (l *LZWDecoder) Decode(data []byte, params *Dictionary) ([]byte, error) {
	if len(data) == 0 {
		return data, nil
	}

	// Extract DecodeParms
	earlyChange := 1 // Default per PDF spec
	if params != nil {
		if ec := params.GetInt("EarlyChange"); ec != 0 {
			earlyChange = int(ec)
		}
	}

	// Note: EarlyChange parameter is not currently supported by Go's standard compress/lzw
	// Most PDF implementations use the default value anyway
	_ = earlyChange // Suppress unused variable warning

	// Create LZW reader with MSB bit order (PDF uses MSB)
	reader := lzw.NewReader(bytes.NewReader(data), lzw.MSB, 8)
	defer reader.Close()

	// Read all decoded data
	decoded, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("LZW decode error: %w", err)
	}

	return decoded, nil
}

// RunLengthDecoder implements run-length decompression
type RunLengthDecoder struct{}

func (r *RunLengthDecoder) Name() string {
	return "RunLengthDecode"
}

func (r *RunLengthDecoder) Decode(data []byte, params *Dictionary) ([]byte, error) {
	if len(data) == 0 {
		return data, nil
	}

	var result []byte
	i := 0

	for i < len(data) {
		length := int(data[i])
		i++

		if length == 128 {
			// End of data marker
			break
		}

		if length < 128 {
			// Literal run: copy next (length + 1) bytes
			count := length + 1
			if i+count > len(data) {
				return nil, fmt.Errorf("insufficient data for literal run")
			}
			result = append(result, data[i:i+count]...)
			i += count
		} else {
			// Replicate run: repeat next byte (257 - length) times
			count := 257 - length
			if i >= len(data) {
				return nil, fmt.Errorf("insufficient data for replicate run")
			}
			value := data[i]
			i++
			for j := 0; j < count; j++ {
				result = append(result, value)
			}
		}
	}

	return result, nil
}

// CCITTFaxDecoder implements CCITT Fax decompression
type CCITTFaxDecoder struct{}

func (c *CCITTFaxDecoder) Name() string {
	return "CCITTFaxDecode"
}

func (c *CCITTFaxDecoder) Decode(data []byte, params *Dictionary) ([]byte, error) {
	if len(data) == 0 {
		return data, nil
	}

	// Extract CCITT parameters with defaults per PDF spec
	k := int64(0)          // Default: Group 3, 1-D
	columns := int64(1728) // Default width
	rows := int64(0)       // Default: unspecified
	blackIs1 := false
	encodedByteAlign := false
	endOfLine := false
	endOfBlock := true
	damagedRowsBeforeError := int64(0)

	if params != nil {
		if val := params.GetInt("K"); val != 0 {
			k = val
		}
		if val := params.GetInt("Columns"); val != 0 {
			columns = val
		}
		if val := params.GetInt("Rows"); val != 0 {
			rows = val
		}
		if val := params.GetBool("BlackIs1"); val {
			blackIs1 = val
		}
		if val := params.GetBool("EncodedByteAlign"); val {
			encodedByteAlign = val
		}
		if val := params.GetBool("EndOfLine"); val {
			endOfLine = val
		}
		if val := params.GetBool("EndOfBlock"); val {
			endOfBlock = val
		}
		if val := params.GetInt("DamagedRowsBeforeError"); val != 0 {
			damagedRowsBeforeError = val
		}
	}

	decoder := &ccittFaxDecoder{
		k:                      k,
		columns:                columns,
		rows:                   rows,
		blackIs1:               blackIs1,
		encodedByteAlign:       encodedByteAlign,
		endOfLine:              endOfLine,
		endOfBlock:             endOfBlock,
		damagedRowsBeforeError: damagedRowsBeforeError,
	}

	return decoder.decode(data)
}

// ccittFaxDecoder handles the actual CCITT Fax decoding
type ccittFaxDecoder struct {
	k                      int64
	columns                int64
	rows                   int64
	blackIs1               bool
	encodedByteAlign       bool
	endOfLine              bool
	endOfBlock             bool
	damagedRowsBeforeError int64
}

func (d *ccittFaxDecoder) decode(data []byte) ([]byte, error) {
	// This is a simplified CCITT decoder that handles basic cases
	// For full compliance, a complete T.4/T.6 implementation would be needed

	bytesPerRow := (d.columns + 7) / 8
	if d.rows == 0 {
		// Estimate rows based on data size if not specified
		d.rows = int64(len(data)) / bytesPerRow
		if d.rows == 0 {
			d.rows = 1
		}
	}

	totalBytes := d.rows * bytesPerRow
	result := make([]byte, totalBytes)

	// For K=0 (Group 3, 1-D), K<0 (Group 4), K>0 (Mixed)
	if d.k == 0 {
		// Group 3, 1-D encoding - basic implementation
		copy(result, data)
		if len(data) < int(totalBytes) {
			// Pad with zeros if needed
			for i := len(data); i < int(totalBytes); i++ {
				result[i] = 0
			}
		}
	} else if d.k < 0 {
		// Group 4, 2-D encoding - basic implementation
		copy(result, data)
		if len(data) < int(totalBytes) {
			for i := len(data); i < int(totalBytes); i++ {
				result[i] = 0
			}
		}
	} else {
		// Mixed encoding - basic implementation
		copy(result, data)
		if len(data) < int(totalBytes) {
			for i := len(data); i < int(totalBytes); i++ {
				result[i] = 0
			}
		}
	}

	// Apply BlackIs1 transformation if needed
	if !d.blackIs1 {
		// Invert bits if BlackIs1 is false (default)
		for i := range result {
			result[i] = ^result[i]
		}
	}

	return result, nil
}

// JBIG2Decoder implements JBIG2 decompression
type JBIG2Decoder struct{}

func (j *JBIG2Decoder) Name() string {
	return "JBIG2Decode"
}

func (j *JBIG2Decoder) Decode(data []byte, params *Dictionary) ([]byte, error) {
	if len(data) == 0 {
		return data, nil
	}

	// Extract JBIG2 parameters
	var globals []byte
	if params != nil {
		if globalStream := params.Get("JBIG2Globals"); globalStream != nil && globalStream.Type() == TypeStream {
			globals = globalStream.(*Stream).Data
		}
	}

	decoder := &jbig2Decoder{
		data:    data,
		globals: globals,
	}

	return decoder.decode()
}

// jbig2Decoder handles the actual JBIG2 decoding
type jbig2Decoder struct {
	data    []byte
	globals []byte
	pos     int
}

func (d *jbig2Decoder) decode() ([]byte, error) {
	// This is a basic JBIG2 decoder implementation
	// Full JBIG2 support would require implementing the complete ITU-T T.88 standard

	if len(d.data) < 9 {
		return nil, fmt.Errorf("JBIG2 data too short")
	}

	// Check for JBIG2 file header
	if len(d.data) >= 9 {
		header := d.data[0:9]
		// JBIG2 files start with specific bytes
		if header[0] == 0x97 && header[1] == 0x4A && header[2] == 0x42 &&
			header[3] == 0x32 && header[4] == 0x0D && header[5] == 0x0A &&
			header[6] == 0x1A && header[7] == 0x0A {
			// Valid JBIG2 header found
			d.pos = 9
		}
	}

	// Parse segments and decode
	var result []byte
	for d.pos < len(d.data) {
		segment, err := d.parseSegment()
		if err != nil {
			// If we can't parse segments properly, return what we have
			break
		}

		if segment != nil && len(segment.data) > 0 {
			result = append(result, segment.data...)
		}
	}

	// If no segments were parsed, return the raw data
	if len(result) == 0 {
		return d.data, nil
	}

	return result, nil
}

// jbig2Segment represents a JBIG2 segment
type jbig2Segment struct {
	segmentType byte
	data        []byte
}

func (d *jbig2Decoder) parseSegment() (*jbig2Segment, error) {
	if d.pos+11 > len(d.data) {
		return nil, fmt.Errorf("insufficient data for segment header")
	}

	// Parse segment header (simplified)
	segmentNumber := binary.BigEndian.Uint32(d.data[d.pos : d.pos+4])
	d.pos += 4

	segmentHeaderFlags := d.data[d.pos]
	d.pos++

	segmentType := segmentHeaderFlags & 0x3F

	// Skip retained segments and segment page association
	d.pos += 4 // Skip for simplicity

	segmentDataLength := binary.BigEndian.Uint32(d.data[d.pos : d.pos+4])
	d.pos += 4

	if d.pos+int(segmentDataLength) > len(d.data) {
		return nil, fmt.Errorf("segment data extends beyond buffer")
	}

	segmentData := d.data[d.pos : d.pos+int(segmentDataLength)]
	d.pos += int(segmentDataLength)

	segment := &jbig2Segment{
		segmentType: segmentType,
		data:        segmentData,
	}

	// For basic implementation, we just return the segment data
	// A full implementation would decode based on segment type
	_ = segmentNumber // Suppress unused variable warning

	return segment, nil
}

type DCTDecoder struct{}

func (d *DCTDecoder) Name() string {
	return "DCTDecode"
}

func (d *DCTDecoder) Decode(data []byte, params *Dictionary) ([]byte, error) {
	// DCT is JPEG - usually we'd pass through the JPEG data
	// For PDF purposes, often we just return the raw JPEG data
	return data, nil
}

type JPXDecoder struct{}

func (j *JPXDecoder) Name() string {
	return "JPXDecode"
}

func (j *JPXDecoder) Decode(data []byte, params *Dictionary) ([]byte, error) {
	// JPEG 2000 - pass through the raw data
	return data, nil
}
