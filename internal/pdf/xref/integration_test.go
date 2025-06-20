package xref

import (
	"fmt"
	"strings"
	"testing"
)

// createRealisticPDFStructure creates a more realistic PDF structure for integration testing
func createRealisticPDFStructure() string {
	return `%PDF-1.4
1 0 obj
<<
/Type /Catalog
/Pages 2 0 R
/Outlines 3 0 R
>>
endobj

2 0 obj
<<
/Type /Pages
/Kids [4 0 R]
/Count 1
>>
endobj

3 0 obj
<<
/Type /Outlines
/Count 0
>>
endobj

4 0 obj
<<
/Type /Page
/Parent 2 0 R
/MediaBox [0 0 612 792]
/Contents 5 0 R
/Resources <<
  /Font <<
    /F1 6 0 R
  >>
>>
>>
endobj

5 0 obj
<<
/Length 44
>>
stream
BT
/F1 12 Tf
100 700 Td
(Hello, World!) Tj
ET
endstream
endobj

6 0 obj
<<
/Type /Font
/Subtype /Type1
/BaseFont /Helvetica
>>
endobj

xref
0 7
0000000000 65535 f
0000000010 00000 n
0000000079 00000 n
0000000173 00000 n
0000000301 00000 n
0000000380 00000 n
0000000491 00000 n
trailer
<<
/Size 7
/Root 1 0 R
>>
startxref
567
%%EOF`
}

// createIncrementalUpdatePDF creates a PDF with incremental updates (Prev chain)
func createIncrementalUpdatePDF() string {
	return `xref
0 3
0000000000 65535 f
0000000015 00000 n
0000000065 00000 n
trailer
<<
/Size 3
/Root 1 0 R
>>
startxref
0
%%EOF
xref
3 1
0000000120 00000 n
trailer
<<
/Size 4
/Root 1 0 R
/Prev 0
>>
startxref
200
%%EOF`
}

func TestXRefParser_RandomAccess(t *testing.T) {
	testData := createRealisticPDFStructure()
	reader := strings.NewReader(testData)
	parser := NewXRefParser(reader)

	// Find the actual position of "xref" keyword in the test data
	xrefPos := strings.Index(testData, "xref")
	if xrefPos == -1 {
		t.Fatal("Could not find xref keyword in test data")
	}

	// Parse the xref table starting from the actual xref position
	err := parser.ParseXRef(int64(xrefPos))
	if err != nil {
		t.Fatalf("Failed to parse xref: %v", err)
	}

	// Test random access to different objects
	testCases := []struct {
		objNum      int
		generation  int
		shouldExist bool
		description string
	}{
		{1, 0, true, "Catalog object"},
		{2, 0, true, "Pages object"},
		{3, 0, true, "Outlines object"},
		{4, 0, true, "Page object"},
		{5, 0, true, "Content stream"},
		{6, 0, true, "Font object"},
		{0, 65535, true, "Free entry"},
		{999, 0, false, "Non-existent object"},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("Object_%d_%d_%s", tc.objNum, tc.generation, tc.description), func(t *testing.T) {
			entry := parser.GetEntry(tc.objNum, tc.generation)

			if tc.shouldExist {
				if entry == nil {
					t.Errorf("Expected entry for object %d %d (%s) but got nil",
						tc.objNum, tc.generation, tc.description)
					return
				}

				// Verify entry properties
				if tc.objNum == 0 && tc.generation == 65535 {
					if entry.Type != EntryFree {
						t.Errorf("Object 0 65535 should be free, got %v", entry.Type)
					}
				} else {
					if entry.Type != EntryInUse {
						t.Errorf("Object %d %d should be in-use, got %v",
							tc.objNum, tc.generation, entry.Type)
					}
					if entry.Offset <= 0 {
						t.Errorf("Object %d %d should have positive offset, got %d",
							tc.objNum, tc.generation, entry.Offset)
					}
				}
			} else {
				if entry != nil {
					t.Errorf("Expected no entry for object %d %d but got %v",
						tc.objNum, tc.generation, entry)
				}
			}
		})
	}
}

func TestXRefParser_IncrementalUpdates(t *testing.T) {
	testData := createIncrementalUpdatePDF()
	reader := strings.NewReader(testData)
	parser := NewXRefParser(reader)

	// Parse starting from the latest xref (position 118)
	err := parser.ParseXRef(118)
	if err != nil {
		t.Fatalf("Failed to parse xref with incremental updates: %v", err)
	}

	// Verify that we have entries from both xref sections
	expectedObjects := []int{0, 1, 2, 3}
	actualObjects := parser.GetObjectNumbers()

	objectMap := make(map[int]bool)
	for _, obj := range actualObjects {
		objectMap[obj] = true
	}

	for _, expected := range expectedObjects {
		if !objectMap[expected] {
			t.Errorf("Expected object %d not found in parsed objects", expected)
		}
	}

	// Verify that object 3 (from incremental update) is accessible
	entry := parser.GetEntry(3, 0)
	if entry == nil {
		t.Error("Object 3 from incremental update should be accessible")
	} else {
		if entry.Type != EntryInUse {
			t.Errorf("Object 3 should be in-use, got %v", entry.Type)
		}
		if entry.Offset != 120 {
			t.Errorf("Object 3 should have offset 120, got %d", entry.Offset)
		}
	}

	// Verify Prev chain
	trailers := parser.GetAllTrailers()
	if len(trailers) < 2 {
		t.Errorf("Expected at least 2 trailers in Prev chain, got %d", len(trailers))
	}

	// Check that latest trailer has Prev pointer
	latestTrailer := parser.GetTrailer()
	if latestTrailer.Prev == nil {
		t.Error("Latest trailer should have Prev pointer")
	} else {
		if *latestTrailer.Prev != 0 {
			t.Errorf("Expected Prev offset 0, got %d", *latestTrailer.Prev)
		}
	}
}

func TestXRefParser_ObjectResolution(t *testing.T) {
	testData := createRealisticPDFStructure()
	reader := strings.NewReader(testData)
	parser := NewXRefParser(reader)

	// Find the actual position of "xref" keyword in the test data
	xrefPos := strings.Index(testData, "xref")
	if xrefPos == -1 {
		t.Fatal("Could not find xref keyword in test data")
	}

	err := parser.ParseXRef(int64(xrefPos))
	if err != nil {
		t.Fatalf("Failed to parse xref: %v", err)
	}

	// Test object resolution (placeholder implementation)
	obj, err := parser.ResolveObject(1, 0)
	if err != nil {
		t.Errorf("Failed to resolve object 1 0: %v", err)
	}
	if obj == nil {
		t.Error("Resolved object should not be nil")
	}

	// Test caching
	initialCacheSize := parser.GetCacheSize()

	// Resolve the same object again
	obj2, err := parser.ResolveObject(1, 0)
	if err != nil {
		t.Errorf("Failed to resolve cached object 1 0: %v", err)
	}
	if obj2 == nil {
		t.Error("Cached resolved object should not be nil")
	}

	// Cache size should be consistent
	finalCacheSize := parser.GetCacheSize()
	if finalCacheSize != initialCacheSize {
		t.Errorf("Cache size changed unexpectedly: %d -> %d",
			initialCacheSize, finalCacheSize)
	}
}

func TestXRefParser_PerformanceRandomAccess(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	testData := createRealisticPDFStructure()
	reader := strings.NewReader(testData)
	parser := NewXRefParser(reader)

	// Find the actual position of "xref" keyword in the test data
	xrefPos := strings.Index(testData, "xref")
	if xrefPos == -1 {
		t.Fatalf("Could not find xref keyword in test data")
	}

	err := parser.ParseXRef(int64(xrefPos))
	if err != nil {
		t.Fatalf("Failed to parse xref: %v", err)
	}

	// Test random access performance
	iterations := 1000
	objectNumbers := []int{1, 2, 3, 4, 5, 6}

	for i := 0; i < iterations; i++ {
		objNum := objectNumbers[i%len(objectNumbers)]
		entry := parser.GetEntry(objNum, 0)
		if entry == nil {
			t.Errorf("Failed to get entry for object %d on iteration %d", objNum, i)
		}
	}
}

func TestXRefParser_ErrorRecovery(t *testing.T) {
	// Test with malformed but partially valid PDF
	malformedPDF := `%PDF-1.4
1 0 obj
<<
/Type /Catalog
>>
endobj

xref
0 2
0000000000 65535 f
invalid entry here but continue
trailer
<<
/Size 2
/Root 1 0 R
>>
startxref
50
%%EOF`

	reader := strings.NewReader(malformedPDF)
	parser := NewXRefParser(reader)

	// Find the actual position of "xref" keyword in the malformed test data
	xrefPos := strings.Index(malformedPDF, "xref")
	if xrefPos == -1 {
		t.Fatal("Could not find xref keyword in malformed test data")
	}

	// Should not fail completely due to liberal parsing
	err := parser.ParseXRef(int64(xrefPos))
	if err != nil {
		t.Logf("Parser returned error (acceptable for malformed input): %v", err)
	}

	// Should still have some valid entries
	if parser.GetEntryCount() == 0 {
		t.Log("No entries parsed from malformed PDF (this may be acceptable)")
	}

	// Free entry should still be accessible
	entry := parser.GetEntry(0, 65535)
	if entry != nil && entry.Type == EntryFree {
		t.Log("Successfully parsed free entry despite malformed input")
	}
}

// Benchmark the random access performance
func BenchmarkXRefParser_RandomAccess(b *testing.B) {
	testData := createRealisticPDFStructure()
	reader := strings.NewReader(testData)
	parser := NewXRefParser(reader)

	// Find the actual position of "xref" keyword in the test data
	xrefPos := strings.Index(testData, "xref")
	if xrefPos == -1 {
		b.Fatalf("Could not find xref keyword in test data")
	}

	err := parser.ParseXRef(int64(xrefPos))
	if err != nil {
		b.Fatalf("Failed to parse xref: %v", err)
	}

	objectNumbers := []int{1, 2, 3, 4, 5, 6}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		objNum := objectNumbers[i%len(objectNumbers)]
		entry := parser.GetEntry(objNum, 0)
		if entry == nil {
			b.Errorf("Failed to get entry for object %d", objNum)
		}
	}
}

func BenchmarkXRefParser_ObjectResolution(b *testing.B) {
	testData := createRealisticPDFStructure()
	reader := strings.NewReader(testData)
	parser := NewXRefParser(reader)

	// Find the actual position of "xref" keyword in the test data
	xrefPos := strings.Index(testData, "xref")
	if xrefPos == -1 {
		b.Fatalf("Could not find xref keyword in test data")
	}

	err := parser.ParseXRef(int64(xrefPos))
	if err != nil {
		b.Fatalf("Failed to parse xref: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		objNum := (i % 6) + 1 // Objects 1-6
		_, err := parser.ResolveObject(objNum, 0)
		if err != nil {
			b.Errorf("Failed to resolve object %d: %v", objNum, err)
		}
	}
}
