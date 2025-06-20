package xref

import (
	"strings"
	"testing"
)

// createTestXRefTable creates a sample xref table for testing
func createTestXRefTable() string {
	return `xref
0 6
0000000000 65535 f
0000000009 00000 n
0000000074 00000 n
0000000173 00000 n
0000000301 00000 n
0000000380 00000 n
trailer
<<
/Size 6
/Root 1 0 R
/Info 5 0 R
>>
startxref
0
%%EOF`
}

// createTestXRefTableWithPrev creates a xref table with Prev pointer
func createTestXRefTableWithPrev() string {
	return `xref
0 3
0000000000 65535 f
0000000009 00000 n
0000000074 00000 n
3 2
0000000173 00000 n
0000000301 00000 n
trailer
<<
/Size 5
/Root 1 0 R
/Prev 100
>>
startxref
0
%%EOF`
}

// createMalformedXRefTable creates a malformed xref table for error testing
func createMalformedXRefTable() string {
	return `xref
0 3
invalid entry here
0000000009 00000 n
0000000074 00000 n
trailer
<<
/Size 3
/Root 1 0 R
>>
startxref
0
%%EOF`
}

func TestNewXRefParser(t *testing.T) {
	reader := strings.NewReader("test")
	parser := NewXRefParser(reader)

	if parser == nil {
		t.Fatal("NewXRefParser returned nil")
	}

	if parser.reader != reader {
		t.Error("Reader not set correctly")
	}

	if parser.entries == nil {
		t.Error("Entries map not initialized")
	}

	if parser.trailers == nil {
		t.Error("Trailers slice not initialized")
	}

	if parser.objectCache == nil {
		t.Error("Object cache not initialized")
	}
}

func TestParseXRefTable_Basic(t *testing.T) {
	testData := createTestXRefTable()
	reader := strings.NewReader(testData)
	parser := NewXRefParser(reader)

	err := parser.ParseXRef(0)
	if err != nil {
		t.Fatalf("ParseXRef failed: %v", err)
	}

	// Check entries
	if parser.GetEntryCount() != 6 {
		t.Errorf("Expected 6 entries, got %d", parser.GetEntryCount())
	}

	// Check specific entries
	entry := parser.GetEntry(0, 65535)
	if entry == nil {
		t.Error("Entry 0 65535 not found")
	} else {
		if entry.Type != EntryFree {
			t.Errorf("Entry 0 should be free, got %v", entry.Type)
		}
	}

	entry = parser.GetEntry(1, 0)
	if entry == nil {
		t.Error("Entry 1 0 not found")
	} else {
		if entry.Type != EntryInUse {
			t.Errorf("Entry 1 should be in-use, got %v", entry.Type)
		}
		if entry.Offset != 9 {
			t.Errorf("Entry 1 offset should be 9, got %d", entry.Offset)
		}
	}

	// Check trailer
	trailer := parser.GetTrailer()
	if trailer == nil {
		t.Fatal("No trailer found")
	}

	if trailer.Size != 6 {
		t.Errorf("Expected trailer size 6, got %d", trailer.Size)
	}

	if trailer.Root == nil {
		t.Error("Root reference not found in trailer")
	} else {
		if trailer.Root.ObjectNumber != 1 || trailer.Root.GenerationNumber != 0 {
			t.Errorf("Expected Root 1 0 R, got %v", trailer.Root)
		}
	}
}

func TestParseXRefTable_WithPrev(t *testing.T) {
	testData := createTestXRefTableWithPrev()
	reader := strings.NewReader(testData)
	parser := NewXRefParser(reader)

	err := parser.ParseXRef(0)
	if err != nil {
		t.Fatalf("ParseXRef failed: %v", err)
	}

	// Check entries from both subsections
	entry := parser.GetEntry(0, 65535)
	if entry == nil || entry.Type != EntryFree {
		t.Error("Entry 0 should be free")
	}

	entry = parser.GetEntry(3, 0)
	if entry == nil || entry.Type != EntryInUse {
		t.Error("Entry 3 should be in-use")
	}

	// Check trailer with Prev
	trailer := parser.GetTrailer()
	if trailer == nil {
		t.Fatal("No trailer found")
	}

	if trailer.Prev == nil || *trailer.Prev != 100 {
		t.Errorf("Expected Prev 100, got %v", trailer.Prev)
	}
}

func TestParseXRefTable_Malformed(t *testing.T) {
	testData := createMalformedXRefTable()
	reader := strings.NewReader(testData)
	parser := NewXRefParser(reader)

	// Should not fail completely due to liberal parsing
	err := parser.ParseXRef(0)
	if err != nil {
		t.Logf("ParseXRef returned error (expected for malformed data): %v", err)
	}

	// Should still have some valid entries
	validEntries := 0
	for _, objNum := range parser.GetObjectNumbers() {
		if entry := parser.GetLatestEntry(objNum); entry != nil && entry.Type == EntryInUse {
			validEntries++
		}
	}

	t.Logf("Found %d valid entries despite malformed input", validEntries)
}

func TestXRefEntry_Methods(t *testing.T) {
	testData := createTestXRefTable()
	reader := strings.NewReader(testData)
	parser := NewXRefParser(reader)

	err := parser.ParseXRef(0)
	if err != nil {
		t.Fatalf("ParseXRef failed: %v", err)
	}

	// Test HasEntry
	if !parser.HasEntry(1, 0) {
		t.Error("HasEntry should return true for existing entry")
	}

	if parser.HasEntry(999, 0) {
		t.Error("HasEntry should return false for non-existing entry")
	}

	// Test GetLatestEntry
	latestEntry := parser.GetLatestEntry(1)
	if latestEntry == nil {
		t.Error("GetLatestEntry should return entry for object 1")
	}

	// Test GetObjectNumbers
	objNumbers := parser.GetObjectNumbers()
	if len(objNumbers) == 0 {
		t.Error("GetObjectNumbers should return object numbers")
	}

	expectedNumbers := map[int]bool{0: true, 1: true, 2: true, 3: true, 4: true, 5: true}
	for _, num := range objNumbers {
		if !expectedNumbers[num] {
			t.Errorf("Unexpected object number: %d", num)
		}
	}
}

func TestTrailerDict_Parsing(t *testing.T) {
	testData := createTestXRefTable()
	reader := strings.NewReader(testData)
	parser := NewXRefParser(reader)

	err := parser.ParseXRef(0)
	if err != nil {
		t.Fatalf("ParseXRef failed: %v", err)
	}

	trailer := parser.GetTrailer()
	if trailer == nil {
		t.Fatal("No trailer found")
	}

	// Test all trailer fields
	if trailer.Size != 6 {
		t.Errorf("Expected Size 6, got %d", trailer.Size)
	}

	if trailer.Root == nil {
		t.Error("Root should not be nil")
	}

	if trailer.Info == nil {
		t.Error("Info should not be nil")
	}

	if trailer.Encrypt != nil {
		t.Error("Encrypt should be nil for unencrypted document")
	}

	if trailer.Prev != nil {
		t.Error("Prev should be nil for single xref table")
	}
}

func TestValidateConsistency(t *testing.T) {
	testData := createTestXRefTable()
	reader := strings.NewReader(testData)
	parser := NewXRefParser(reader)

	err := parser.ParseXRef(0)
	if err != nil {
		t.Fatalf("ParseXRef failed: %v", err)
	}

	err = parser.ValidateConsistency()
	if err != nil {
		t.Errorf("ValidateConsistency failed: %v", err)
	}
}

func TestValidateConsistency_MissingRoot(t *testing.T) {
	// Create xref table without Root reference
	testData := `xref
0 2
0000000000 65535 f
0000000009 00000 n
trailer
<<
/Size 2
>>
startxref
0
%%EOF`

	reader := strings.NewReader(testData)
	parser := NewXRefParser(reader)

	err := parser.ParseXRef(0)
	if err != nil {
		t.Fatalf("ParseXRef failed: %v", err)
	}

	err = parser.ValidateConsistency()
	if err == nil {
		t.Error("ValidateConsistency should fail when Root is missing")
	}
}

func TestResolveObject(t *testing.T) {
	testData := createTestXRefTable()
	reader := strings.NewReader(testData)
	parser := NewXRefParser(reader)

	err := parser.ParseXRef(0)
	if err != nil {
		t.Fatalf("ParseXRef failed: %v", err)
	}

	// Test resolving an in-use object
	obj, err := parser.ResolveObject(1, 0)
	if err != nil {
		t.Errorf("ResolveObject failed: %v", err)
	}
	if obj == nil {
		t.Error("ResolveObject returned nil object")
	}

	// Test resolving a free object
	obj, err = parser.ResolveObject(0, 65535)
	if err != nil {
		t.Errorf("ResolveObject failed for free object: %v", err)
	}
	if obj.Type() != TypeNull {
		t.Errorf("Free object should resolve to null, got %v", obj.Type())
	}

	// Test resolving non-existent object
	_, err = parser.ResolveObject(999, 0)
	if err == nil {
		t.Error("ResolveObject should fail for non-existent object")
	}
}

func TestObjectCache(t *testing.T) {
	testData := createTestXRefTable()
	reader := strings.NewReader(testData)
	parser := NewXRefParser(reader)

	err := parser.ParseXRef(0)
	if err != nil {
		t.Fatalf("ParseXRef failed: %v", err)
	}

	// Initial cache should be empty
	if parser.GetCacheSize() != 0 {
		t.Errorf("Initial cache size should be 0, got %d", parser.GetCacheSize())
	}

	// Resolve an object to populate cache
	_, err = parser.ResolveObject(1, 0)
	if err != nil {
		t.Fatalf("ResolveObject failed: %v", err)
	}

	// Cache should now have one entry
	if parser.GetCacheSize() != 1 {
		t.Errorf("Cache size should be 1 after resolution, got %d", parser.GetCacheSize())
	}

	// Resolve the same object again (should hit cache)
	_, err = parser.ResolveObject(1, 0)
	if err != nil {
		t.Fatalf("ResolveObject failed on second call: %v", err)
	}

	// Cache size should still be 1
	if parser.GetCacheSize() != 1 {
		t.Errorf("Cache size should still be 1, got %d", parser.GetCacheSize())
	}

	// Clear cache
	parser.ClearCache()
	if parser.GetCacheSize() != 0 {
		t.Errorf("Cache size should be 0 after clear, got %d", parser.GetCacheSize())
	}
}

func TestEntryType_String(t *testing.T) {
	tests := []struct {
		entryType EntryType
		expected  string
	}{
		{EntryFree, "free"},
		{EntryInUse, "in-use"},
		{EntryCompressed, "compressed"},
		{EntryType(999), "unknown"},
	}

	for _, test := range tests {
		result := test.entryType.String()
		if result != test.expected {
			t.Errorf("EntryType(%d).String() = %s, expected %s",
				test.entryType, result, test.expected)
		}
	}
}

func TestIndirectRef_String(t *testing.T) {
	ref := &IndirectRef{
		ObjectNumber:     123,
		GenerationNumber: 45,
	}

	expected := "123 45 R"
	result := ref.String()
	if result != expected {
		t.Errorf("IndirectRef.String() = %s, expected %s", result, expected)
	}
}

func TestParseXRefEntryLine(t *testing.T) {
	parser := NewXRefParser(strings.NewReader(""))

	tests := []struct {
		line        string
		objNum      int
		expectError bool
		expected    *XRefEntry
	}{
		{
			line:   "0000000009 00000 n ",
			objNum: 1,
			expected: &XRefEntry{
				Type:       EntryInUse,
				Offset:     9,
				Generation: 0,
			},
		},
		{
			line:   "0000000000 65535 f ",
			objNum: 0,
			expected: &XRefEntry{
				Type:       EntryFree,
				Offset:     0,
				Generation: 65535,
			},
		},
		{
			line:        "invalid entry",
			objNum:      1,
			expectError: true,
		},
		{
			line:        "123",
			objNum:      1,
			expectError: true,
		},
	}

	for i, test := range tests {
		entry, err := parser.parseXRefEntryLine(test.line, test.objNum)

		if test.expectError {
			if err == nil {
				t.Errorf("Test %d: expected error for line '%s'", i, test.line)
			}
			continue
		}

		if err != nil {
			t.Errorf("Test %d: unexpected error: %v", i, err)
			continue
		}

		if entry.Type != test.expected.Type {
			t.Errorf("Test %d: Type = %v, expected %v", i, entry.Type, test.expected.Type)
		}

		if entry.Offset != test.expected.Offset {
			t.Errorf("Test %d: Offset = %d, expected %d", i, entry.Offset, test.expected.Offset)
		}

		if entry.Generation != test.expected.Generation {
			t.Errorf("Test %d: Generation = %d, expected %d", i, entry.Generation, test.expected.Generation)
		}
	}
}

func TestParseIndirectRef(t *testing.T) {
	parser := NewXRefParser(strings.NewReader(""))

	tests := []struct {
		line     string
		expected *IndirectRef
	}{
		{
			line: "/Root 1 0 R",
			expected: &IndirectRef{
				ObjectNumber:     1,
				GenerationNumber: 0,
			},
		},
		{
			line: "/Info 5 2 R",
			expected: &IndirectRef{
				ObjectNumber:     5,
				GenerationNumber: 2,
			},
		},
		{
			line:     "/Size 6",
			expected: nil,
		},
		{
			line:     "invalid reference",
			expected: nil,
		},
	}

	for i, test := range tests {
		result := parser.parseIndirectRef(test.line)

		if test.expected == nil {
			if result != nil {
				t.Errorf("Test %d: expected nil, got %v", i, result)
			}
			continue
		}

		if result == nil {
			t.Errorf("Test %d: expected %v, got nil", i, test.expected)
			continue
		}

		if result.ObjectNumber != test.expected.ObjectNumber {
			t.Errorf("Test %d: ObjectNumber = %d, expected %d",
				i, result.ObjectNumber, test.expected.ObjectNumber)
		}

		if result.GenerationNumber != test.expected.GenerationNumber {
			t.Errorf("Test %d: GenerationNumber = %d, expected %d",
				i, result.GenerationNumber, test.expected.GenerationNumber)
		}
	}
}

func TestGetStartXRefAndPrevChain(t *testing.T) {
	testData := createTestXRefTable()
	reader := strings.NewReader(testData)
	parser := NewXRefParser(reader)

	startxref := int64(0)
	err := parser.ParseXRef(startxref)
	if err != nil {
		t.Fatalf("ParseXRef failed: %v", err)
	}

	if parser.GetStartXRef() != startxref {
		t.Errorf("GetStartXRef() = %d, expected %d", parser.GetStartXRef(), startxref)
	}

	prevChain := parser.GetPrevChain()
	if len(prevChain) == 0 {
		t.Error("PrevChain should not be empty")
	}

	if prevChain[0] != startxref {
		t.Errorf("First entry in PrevChain should be %d, got %d", startxref, prevChain[0])
	}
}

func TestParseXRefStream(t *testing.T) {
	parser := NewXRefParser(strings.NewReader(""))

	// Test that xref streams return appropriate error for now
	trailer, err := parser.parseXRefStream()
	if err == nil {
		t.Error("parseXRefStream should return error (not implemented)")
	}
	if trailer != nil {
		t.Error("parseXRefStream should return nil trailer when not implemented")
	}
}

// Benchmark tests
func BenchmarkParseXRefTable(b *testing.B) {
	testData := createTestXRefTable()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reader := strings.NewReader(testData)
		parser := NewXRefParser(reader)
		parser.ParseXRef(0)
	}
}

func BenchmarkResolveObject(b *testing.B) {
	testData := createTestXRefTable()
	reader := strings.NewReader(testData)
	parser := NewXRefParser(reader)
	parser.ParseXRef(0)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parser.ResolveObject(1, 0)
	}
}
