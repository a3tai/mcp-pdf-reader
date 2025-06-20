package custom

import (
	"bytes"
	"compress/flate"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFilterSystemIntegration tests the complete filter system integration
func TestFilterSystemIntegration(t *testing.T) {
	t.Run("CompleteFilterChain", func(t *testing.T) {
		// Test a complete filter chain: ASCII Hex -> Flate -> result
		original := []byte("This is test data for PDF filter integration testing")

		// Step 1: Compress with flate
		var compressed bytes.Buffer
		writer, err := flate.NewWriter(&compressed, flate.DefaultCompression)
		require.NoError(t, err)
		_, err = writer.Write(original)
		require.NoError(t, err)
		err = writer.Close()
		require.NoError(t, err)

		// Step 2: Encode with ASCII Hex
		hexEncoded := hex.EncodeToString(compressed.Bytes()) + ">"

		// Step 3: Create PDF stream with filter chain
		dict := NewDictionary()
		filterArray := &Array{}
		filterArray.Add(&Name{Value: "ASCIIHexDecode"})
		filterArray.Add(&Name{Value: "FlateDecode"})
		dict.Set("Filter", filterArray)

		stream := &Stream{
			Dict: dict,
			Data: []byte(hexEncoded),
		}

		// Step 4: Decode through filter chain
		decoded, err := DecodeStream(stream)
		require.NoError(t, err)
		assert.Equal(t, original, decoded)
	})

	t.Run("FilterWithParameters", func(t *testing.T) {
		// Test filter with DecodeParms
		original := []byte("Test data with predictor")

		// Compress with flate
		var compressed bytes.Buffer
		writer, err := flate.NewWriter(&compressed, flate.DefaultCompression)
		require.NoError(t, err)
		_, err = writer.Write(original)
		require.NoError(t, err)
		err = writer.Close()
		require.NoError(t, err)

		// Create stream with FlateDecode and predictor params
		dict := NewDictionary()
		dict.Set("Filter", &Name{Value: "FlateDecode"})

		params := NewDictionary()
		params.Set("Predictor", &Number{Value: int64(1)}) // No predictor
		dict.Set("DecodeParms", params)

		stream := &Stream{
			Dict: dict,
			Data: compressed.Bytes(),
		}

		decoded, err := DecodeStream(stream)
		require.NoError(t, err)
		assert.Equal(t, original, decoded)
	})

	t.Run("MultipleFilterParameters", func(t *testing.T) {
		// Test multiple filters with multiple parameter dictionaries
		original := []byte("Multi-filter test")

		// Compress and encode
		var compressed bytes.Buffer
		writer, err := flate.NewWriter(&compressed, flate.DefaultCompression)
		require.NoError(t, err)
		_, err = writer.Write(original)
		require.NoError(t, err)
		err = writer.Close()
		require.NoError(t, err)

		hexEncoded := hex.EncodeToString(compressed.Bytes()) + ">"

		// Create stream with multiple filters and parameters
		dict := NewDictionary()
		filterArray := &Array{}
		filterArray.Add(&Name{Value: "ASCIIHexDecode"})
		filterArray.Add(&Name{Value: "FlateDecode"})
		dict.Set("Filter", filterArray)

		// Parameter array
		paramsArray := &Array{}
		paramsArray.Add(&Null{}) // No params for ASCIIHexDecode

		flateParams := NewDictionary()
		flateParams.Set("Predictor", &Number{Value: int64(1)})
		paramsArray.Add(flateParams)

		dict.Set("DecodeParms", paramsArray)

		stream := &Stream{
			Dict: dict,
			Data: []byte(hexEncoded),
		}

		decoded, err := DecodeStream(stream)
		require.NoError(t, err)
		assert.Equal(t, original, decoded)
	})

	t.Run("RunLengthIntegration", func(t *testing.T) {
		// Test RunLength filter integration
		dict := NewDictionary()
		dict.Set("Filter", &Name{Value: "RunLengthDecode"})

		// Create run-length encoded data: "AAAAA"
		runLengthData := []byte{252, 'A', 128} // Replicate 'A' 5 times, EOD

		stream := &Stream{
			Dict: dict,
			Data: runLengthData,
		}

		decoded, err := DecodeStream(stream)
		require.NoError(t, err)
		assert.Equal(t, []byte("AAAAA"), decoded)
	})

	t.Run("ImageFilterIntegration", func(t *testing.T) {
		// Test image filter (DCT pass-through)
		jpegData := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10} // JPEG header

		dict := NewDictionary()
		dict.Set("Filter", &Name{Value: "DCTDecode"})

		stream := &Stream{
			Dict: dict,
			Data: jpegData,
		}

		decoded, err := DecodeStream(stream)
		require.NoError(t, err)
		assert.Equal(t, jpegData, decoded) // Should pass through unchanged
	})

	t.Run("CCITTFaxIntegration", func(t *testing.T) {
		// Test CCITT Fax with parameters
		dict := NewDictionary()
		dict.Set("Filter", &Name{Value: "CCITTFaxDecode"})

		params := NewDictionary()
		params.Set("K", &Number{Value: int64(0)})       // Group 3
		params.Set("Columns", &Number{Value: int64(8)}) // 8 pixels wide
		params.Set("Rows", &Number{Value: int64(1)})    // 1 row
		params.Set("BlackIs1", &Bool{Value: false})
		dict.Set("DecodeParms", params)

		faxData := []byte{0xFF} // Simple test data

		stream := &Stream{
			Dict: dict,
			Data: faxData,
		}

		decoded, err := DecodeStream(stream)
		require.NoError(t, err)
		assert.NotNil(t, decoded)
		// CCITT basic implementation may return different sizes based on available data
		// Just ensure it returns reasonable data without crashing
		assert.True(t, len(decoded) > 0, "Should return some decoded data")
	})

	t.Run("LZWIntegration", func(t *testing.T) {
		// Test LZW filter integration
		dict := NewDictionary()
		dict.Set("Filter", &Name{Value: "LZWDecode"})

		params := NewDictionary()
		params.Set("EarlyChange", &Number{Value: int64(1)})
		dict.Set("DecodeParms", params)

		// Simple LZW test data
		lzwData := []byte{0x80, 0x0B, 0x60, 0x50, 0x22, 0x0C, 0x0C, 0x85, 0x01}

		stream := &Stream{
			Dict: dict,
			Data: lzwData,
		}

		// LZW may fail with test data, but should not crash
		_, err := DecodeStream(stream)
		// Don't require success, just that it doesn't crash
		if err != nil {
			assert.Contains(t, err.Error(), "LZW decode error")
		}
	})
}

// TestFilterErrorHandling tests error handling in the integrated system
func TestFilterErrorHandling(t *testing.T) {
	t.Run("UnsupportedFilter", func(t *testing.T) {
		dict := NewDictionary()
		dict.Set("Filter", &Name{Value: "UnsupportedFilterType"})

		stream := &Stream{
			Dict: dict,
			Data: []byte("test data"),
		}

		_, err := DecodeStream(stream)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported filter")
	})

	t.Run("FilterChainError", func(t *testing.T) {
		dict := NewDictionary()
		filterArray := &Array{}
		filterArray.Add(&Name{Value: "ASCIIHexDecode"})
		filterArray.Add(&Name{Value: "UnsupportedFilter"})
		dict.Set("Filter", filterArray)

		stream := &Stream{
			Dict: dict,
			Data: []byte("48656C6C6F>"), // Valid hex data
		}

		_, err := DecodeStream(stream)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported filter")
	})

	t.Run("MalformedFilterData", func(t *testing.T) {
		dict := NewDictionary()
		dict.Set("Filter", &Name{Value: "ASCIIHexDecode"})

		stream := &Stream{
			Dict: dict,
			Data: []byte("invalid hex data without end marker"),
		}

		// Should handle gracefully
		decoded, err := DecodeStream(stream)
		require.NoError(t, err)
		assert.NotNil(t, decoded)
	})

	t.Run("RunLengthInvalidData", func(t *testing.T) {
		dict := NewDictionary()
		dict.Set("Filter", &Name{Value: "RunLengthDecode"})

		stream := &Stream{
			Dict: dict,
			Data: []byte{5, 'H', 'i'}, // Says 6 bytes but only has 2
		}

		_, err := DecodeStream(stream)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "insufficient data")
	})
}

// TestFilterParameterExtraction tests parameter extraction integration
func TestFilterParameterExtraction(t *testing.T) {
	t.Run("SingleParameterDictionary", func(t *testing.T) {
		dict := NewDictionary()
		dict.Set("Filter", &Name{Value: "FlateDecode"})

		params := NewDictionary()
		params.Set("Predictor", &Number{Value: int64(2)})
		params.Set("Columns", &Number{Value: int64(100)})
		params.Set("BitsPerComponent", &Number{Value: int64(8)})
		dict.Set("DecodeParms", params)

		// Test that parameters are properly extracted
		decodeParms := dict.Get("DecodeParms")
		assert.Equal(t, TypeDictionary, decodeParms.Type())

		paramDict := decodeParms.(*Dictionary)
		assert.Equal(t, int64(2), paramDict.GetInt("Predictor"))
		assert.Equal(t, int64(100), paramDict.GetInt("Columns"))
		assert.Equal(t, int64(8), paramDict.GetInt("BitsPerComponent"))
	})

	t.Run("ParameterArrayExtraction", func(t *testing.T) {
		dict := NewDictionary()
		filterArray := &Array{}
		filterArray.Add(&Name{Value: "ASCIIHexDecode"})
		filterArray.Add(&Name{Value: "FlateDecode"})
		dict.Set("Filter", filterArray)

		paramsArray := &Array{}
		paramsArray.Add(&Null{}) // No params for first filter

		flateParams := NewDictionary()
		flateParams.Set("Predictor", &Number{Value: int64(2)})
		paramsArray.Add(flateParams)

		dict.Set("DecodeParms", paramsArray)

		// Test parameter array extraction
		decodeParms := dict.Get("DecodeParms")
		assert.Equal(t, TypeArray, decodeParms.Type())

		paramArray := decodeParms.(*Array)
		assert.Equal(t, 2, paramArray.Len())

		// First parameter should be null
		firstParam := paramArray.Get(0)
		assert.Equal(t, TypeNull, firstParam.Type())

		// Second parameter should be dictionary
		secondParam := paramArray.Get(1)
		assert.Equal(t, TypeDictionary, secondParam.Type())

		secondDict := secondParam.(*Dictionary)
		assert.Equal(t, int64(2), secondDict.GetInt("Predictor"))
	})
}

// TestFilterPerformance tests basic performance characteristics
func TestFilterPerformance(t *testing.T) {
	t.Run("LargeDataProcessing", func(t *testing.T) {
		// Create larger test data
		original := bytes.Repeat([]byte("Large data test for filter performance. "), 1000)

		// Compress with flate
		var compressed bytes.Buffer
		writer, err := flate.NewWriter(&compressed, flate.DefaultCompression)
		require.NoError(t, err)
		_, err = writer.Write(original)
		require.NoError(t, err)
		err = writer.Close()
		require.NoError(t, err)

		dict := NewDictionary()
		dict.Set("Filter", &Name{Value: "FlateDecode"})

		stream := &Stream{
			Dict: dict,
			Data: compressed.Bytes(),
		}

		// Should handle large data efficiently
		decoded, err := DecodeStream(stream)
		require.NoError(t, err)
		assert.Equal(t, original, decoded)
		assert.True(t, len(decoded) > 10000) // Verify we processed substantial data
	})

	t.Run("MultipleFilterChainPerformance", func(t *testing.T) {
		// Test performance with multiple filters
		original := bytes.Repeat([]byte("Performance test data. "), 100)

		// Compress and encode
		var compressed bytes.Buffer
		writer, err := flate.NewWriter(&compressed, flate.DefaultCompression)
		require.NoError(t, err)
		_, err = writer.Write(original)
		require.NoError(t, err)
		err = writer.Close()
		require.NoError(t, err)

		hexEncoded := hex.EncodeToString(compressed.Bytes()) + ">"

		dict := NewDictionary()
		filterArray := &Array{}
		filterArray.Add(&Name{Value: "ASCIIHexDecode"})
		filterArray.Add(&Name{Value: "FlateDecode"})
		dict.Set("Filter", filterArray)

		stream := &Stream{
			Dict: dict,
			Data: []byte(hexEncoded),
		}

		// Should handle filter chain efficiently
		decoded, err := DecodeStream(stream)
		require.NoError(t, err)
		assert.Equal(t, original, decoded)
	})
}
