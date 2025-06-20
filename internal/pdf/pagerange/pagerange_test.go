package pagerange

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"testing"
)

// Mock reader for testing
type mockPageRangeReader struct {
	data   []byte
	offset int64
}

func newMockPageRangeReader(data []byte) *mockPageRangeReader {
	return &mockPageRangeReader{data: data}
}

func (mr *mockPageRangeReader) Read(p []byte) (n int, err error) {
	if mr.offset >= int64(len(mr.data)) {
		return 0, io.EOF
	}

	n = copy(p, mr.data[mr.offset:])
	mr.offset += int64(n)
	return n, nil
}

func (mr *mockPageRangeReader) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	case io.SeekStart:
		mr.offset = offset
	case io.SeekCurrent:
		mr.offset += offset
	case io.SeekEnd:
		mr.offset = int64(len(mr.data)) + offset
	}

	if mr.offset < 0 {
		mr.offset = 0
	}
	if mr.offset > int64(len(mr.data)) {
		mr.offset = int64(len(mr.data))
	}

	return mr.offset, nil
}

// Test PDF data with multiple pages
var testMultiPagePDF = []byte(generateMultiPagePDF())

func generateMultiPagePDF() string {
	pdf := "%PDF-1.4\n"

	// Object 1 - Catalog
	obj1Start := len(pdf)
	pdf += "1 0 obj\n<<\n/Type /Catalog\n/Pages 2 0 R\n>>\nendobj\n"

	// Object 2 - Pages
	obj2Start := len(pdf)
	pdf += "2 0 obj\n<<\n/Type /Pages\n/Kids [3 0 R 4 0 R 5 0 R]\n/Count 3\n>>\nendobj\n"

	// Object 3 - Page 1
	obj3Start := len(pdf)
	pdf += "3 0 obj\n<<\n/Type /Page\n/Parent 2 0 R\n/MediaBox [0 0 612 792]\n/Contents 6 0 R\n/Resources <<\n/Font <<\n/F1 <<\n/Type /Font\n/Subtype /Type1\n/BaseFont /Helvetica\n>>\n>>\n>>\n>>\nendobj\n"

	// Object 4 - Page 2
	obj4Start := len(pdf)
	pdf += "4 0 obj\n<<\n/Type /Page\n/Parent 2 0 R\n/MediaBox [0 0 612 792]\n/Contents 7 0 R\n/Resources <<\n/Font <<\n/F1 <<\n/Type /Font\n/Subtype /Type1\n/BaseFont /Helvetica\n>>\n>>\n>>\n>>\nendobj\n"

	// Object 5 - Page 3
	obj5Start := len(pdf)
	pdf += "5 0 obj\n<<\n/Type /Page\n/Parent 2 0 R\n/MediaBox [0 0 612 792]\n/Contents 8 0 R\n/Resources <<\n/Font <<\n/F1 <<\n/Type /Font\n/Subtype /Type1\n/BaseFont /Helvetica\n>>\n>>\n>>\n>>\nendobj\n"

	// Object 6 - Content Stream 1
	obj6Start := len(pdf)
	content1 := "BT\n/F1 12 Tf\n100 700 Td\n(Page 1 Text) Tj\nET\n"
	pdf += fmt.Sprintf("6 0 obj\n<<\n/Length %d\n>>\nstream\n%sendstream\nendobj\n", len(content1), content1)

	// Object 7 - Content Stream 2
	obj7Start := len(pdf)
	content2 := "BT\n/F1 12 Tf\n100 700 Td\n(Page 2 Text) Tj\nET\n"
	pdf += fmt.Sprintf("7 0 obj\n<<\n/Length %d\n>>\nstream\n%sendstream\nendobj\n", len(content2), content2)

	// Object 8 - Content Stream 3
	obj8Start := len(pdf)
	content3 := "BT\n/F1 12 Tf\n100 700 Td\n(Page 3 Text) Tj\nET\n"
	pdf += fmt.Sprintf("8 0 obj\n<<\n/Length %d\n>>\nstream\n%sendstream\nendobj\n", len(content3), content3)

	// Cross-reference table
	xrefStart := len(pdf)
	pdf += "xref\n0 9\n0000000000 65535 f \n"
	pdf += fmt.Sprintf("%010d 00000 n \n", obj1Start)
	pdf += fmt.Sprintf("%010d 00000 n \n", obj2Start)
	pdf += fmt.Sprintf("%010d 00000 n \n", obj3Start)
	pdf += fmt.Sprintf("%010d 00000 n \n", obj4Start)
	pdf += fmt.Sprintf("%010d 00000 n \n", obj5Start)
	pdf += fmt.Sprintf("%010d 00000 n \n", obj6Start)
	pdf += fmt.Sprintf("%010d 00000 n \n", obj7Start)
	pdf += fmt.Sprintf("%010d 00000 n \n", obj8Start)

	// Trailer
	pdf += "trailer\n<<\n/Size 9\n/Root 1 0 R\n>>\nstartxref\n"
	pdf += fmt.Sprintf("%d\n", xrefStart)
	pdf += "%%EOF"

	return pdf
}

func TestPageRangeExtractor_Basic(t *testing.T) {
	reader := newMockPageRangeReader(testMultiPagePDF)
	extractor := NewPageRangeExtractor()

	ranges := []PageRange{
		{Start: 1, End: 2},
	}

	options := ExtractOptions{
		ContentTypes:    []string{"text"},
		ExtractImages:   false,
		ExtractForms:    false,
		IncludeMetadata: true,
	}

	result, err := extractor.ExtractRange(reader, ranges, options)
	if err != nil {
		t.Fatalf("Failed to extract page range: %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	if len(result.Pages) == 0 {
		t.Error("Expected pages to be extracted")
	}

	if result.TotalPages == 0 {
		t.Error("Expected total pages to be set")
	}

	if len(result.Ranges) != 1 {
		t.Errorf("Expected 1 range, got %d", len(result.Ranges))
	}
}

func TestPageRangeExtractor_MultipleRanges(t *testing.T) {
	reader := newMockPageRangeReader(testMultiPagePDF)
	extractor := NewPageRangeExtractor()

	ranges := []PageRange{
		{Start: 1, End: 1},
		{Start: 3, End: 3},
	}

	options := ExtractOptions{
		ContentTypes: []string{"text"},
	}

	result, err := extractor.ExtractRange(reader, ranges, options)
	if err != nil {
		t.Fatalf("Failed to extract multiple ranges: %v", err)
	}

	// Should have pages 1 and 3
	if _, exists := result.Pages[1]; !exists {
		t.Error("Expected page 1 to be extracted")
	}

	if _, exists := result.Pages[3]; !exists {
		t.Error("Expected page 3 to be extracted")
	}

	if _, exists := result.Pages[2]; exists {
		t.Error("Page 2 should not be extracted")
	}
}

func TestPageRangeExtractor_InvalidRange(t *testing.T) {
	reader := newMockPageRangeReader(testMultiPagePDF)
	extractor := NewPageRangeExtractor()

	ranges := []PageRange{
		{Start: 10, End: 20}, // Beyond available pages
	}

	options := ExtractOptions{
		ContentTypes: []string{"text"},
	}

	result, err := extractor.ExtractRange(reader, ranges, options)
	if err != nil {
		t.Fatalf("Failed to handle invalid range: %v", err)
	}

	// Should have no pages extracted
	if len(result.Pages) != 0 {
		t.Errorf("Expected no pages for invalid range, got %d", len(result.Pages))
	}
}

func TestPageIndex_Build(t *testing.T) {
	reader := newMockPageRangeReader(testMultiPagePDF)
	builder := NewPageIndexBuilder()

	index, err := builder.BuildFromReader(reader)
	if err != nil {
		t.Fatalf("Failed to build page index: %v", err)
	}

	if index == nil {
		t.Fatal("Expected non-nil index")
	}

	if index.TotalPages <= 0 {
		t.Error("Expected positive total pages")
	}

	// Check that page objects were found
	if len(index.PageObjects) == 0 {
		t.Error("Expected page objects to be indexed")
	}

	// Check that page offsets were recorded
	if len(index.PageOffsets) == 0 {
		t.Error("Expected page offsets to be recorded")
	}
}

func TestPageIndex_GetPage(t *testing.T) {
	index := NewPageIndex()

	// Add test page
	testObj := ObjectRef{
		ObjectID:   3,
		Generation: 0,
		Offset:     100,
	}
	index.PageObjects[1] = testObj
	index.TotalPages = 1

	// Test getting existing page
	obj, exists := index.GetPage(1)
	if !exists {
		t.Error("Expected page 1 to exist")
	}

	if obj.ObjectID != 3 {
		t.Errorf("Expected object ID 3, got %d", obj.ObjectID)
	}

	// Test getting non-existing page
	_, exists = index.GetPage(999)
	if exists {
		t.Error("Expected page 999 to not exist")
	}
}

func TestPageIndex_GetPageRange(t *testing.T) {
	index := NewPageIndex()
	index.TotalPages = 5

	// Add test pages
	for i := 1; i <= 5; i++ {
		index.PageObjects[i] = ObjectRef{
			ObjectID:   i + 2,
			Generation: 0,
			Offset:     int64(i * 100),
		}
	}

	// Test getting range
	result := index.GetPageRange(2, 4)

	if len(result) != 3 {
		t.Errorf("Expected 3 pages in range, got %d", len(result))
	}

	for pageNum := 2; pageNum <= 4; pageNum++ {
		if _, exists := result[pageNum]; !exists {
			t.Errorf("Expected page %d in range result", pageNum)
		}
	}
}

func TestPageIndex_Stats(t *testing.T) {
	index := NewPageIndex()
	index.TotalPages = 3
	index.PageObjects[1] = ObjectRef{ObjectID: 1}
	index.PageObjects[2] = ObjectRef{ObjectID: 2}
	index.PageObjects[3] = ObjectRef{ObjectID: 3}
	index.Resources[1] = []ObjectRef{{ObjectID: 4}, {ObjectID: 5}}
	index.Resources[2] = []ObjectRef{{ObjectID: 6}}

	stats := index.GetStats()

	if stats.TotalPages != 3 {
		t.Errorf("Expected 3 total pages, got %d", stats.TotalPages)
	}

	if stats.IndexedPages != 3 {
		t.Errorf("Expected 3 indexed pages, got %d", stats.IndexedPages)
	}

	if stats.ResourceRefs != 3 {
		t.Errorf("Expected 3 resource refs, got %d", stats.ResourceRefs)
	}

	if stats.IndexEfficiency != 100.0 {
		t.Errorf("Expected 100%% efficiency, got %.1f", stats.IndexEfficiency)
	}
}

func TestPageObjectCache_Basic(t *testing.T) {
	cache := NewPageObjectCache(1024) // 1KB cache

	objRef := ObjectRef{
		ObjectID:   1,
		Generation: 0,
		Offset:     100,
	}

	content := "test content"

	// Test Put
	err := cache.Put(objRef, content)
	if err != nil {
		t.Fatalf("Failed to put object in cache: %v", err)
	}

	// Test Get
	retrieved := cache.Get(objRef)
	if retrieved == nil {
		t.Error("Expected to retrieve cached object")
	}

	if retrieved.(string) != content {
		t.Errorf("Expected '%s', got '%s'", content, retrieved.(string))
	}

	// Test Contains
	if !cache.Contains(objRef) {
		t.Error("Expected cache to contain object")
	}

	// Test non-existing object
	nonExistentRef := ObjectRef{ObjectID: 999, Generation: 0}
	if cache.Contains(nonExistentRef) {
		t.Error("Expected cache to not contain non-existent object")
	}
}

func TestPageObjectCache_LRUEviction(t *testing.T) {
	config := CacheConfig{
		MaxSizeBytes: 100, // Very small cache
		MaxObjects:   2,   // Only 2 objects max
	}
	cache := NewPageObjectCache(100, config)

	// Add objects that exceed cache capacity
	obj1 := ObjectRef{ObjectID: 1, Generation: 0}
	obj2 := ObjectRef{ObjectID: 2, Generation: 0}
	obj3 := ObjectRef{ObjectID: 3, Generation: 0}

	cache.Put(obj1, "content1")
	cache.Put(obj2, "content2")
	cache.Put(obj3, "content3") // Should evict obj1

	// obj1 should be evicted
	if cache.Contains(obj1) {
		t.Error("Expected obj1 to be evicted")
	}

	// obj2 and obj3 should still be there
	if !cache.Contains(obj2) {
		t.Error("Expected obj2 to still be in cache")
	}

	if !cache.Contains(obj3) {
		t.Error("Expected obj3 to be in cache")
	}
}

func TestPageObjectCache_Stats(t *testing.T) {
	cache := NewPageObjectCache(1024)

	objRef := ObjectRef{ObjectID: 1, Generation: 0}
	content := "test content"

	// Initial stats
	stats := cache.GetStats()
	if stats.ObjectCount != 0 {
		t.Errorf("Expected 0 objects initially, got %d", stats.ObjectCount)
	}

	// Add object
	cache.Put(objRef, content)

	// Get object (hit)
	cache.Get(objRef)

	// Get non-existent object (miss)
	nonExistentRef := ObjectRef{ObjectID: 999, Generation: 0}
	cache.Get(nonExistentRef)

	// Check stats
	stats = cache.GetStats()
	if stats.ObjectCount != 1 {
		t.Errorf("Expected 1 object, got %d", stats.ObjectCount)
	}

	if stats.Hits != 1 {
		t.Errorf("Expected 1 hit, got %d", stats.Hits)
	}

	if stats.Misses != 1 {
		t.Errorf("Expected 1 miss, got %d", stats.Misses)
	}

	if stats.HitRate != 50.0 {
		t.Errorf("Expected 50%% hit rate, got %.1f", stats.HitRate)
	}
}

func TestPageObjectCache_Clear(t *testing.T) {
	cache := NewPageObjectCache(1024)

	// Add some objects
	for i := 1; i <= 5; i++ {
		objRef := ObjectRef{ObjectID: i, Generation: 0}
		cache.Put(objRef, "content")
	}

	if cache.GetObjectCount() != 5 {
		t.Errorf("Expected 5 objects before clear, got %d", cache.GetObjectCount())
	}

	// Clear cache
	cache.Clear()

	if cache.GetObjectCount() != 0 {
		t.Errorf("Expected 0 objects after clear, got %d", cache.GetObjectCount())
	}

	if cache.GetSize() != 0 {
		t.Errorf("Expected 0 size after clear, got %d", cache.GetSize())
	}
}

func TestPageObjectCache_DetailedMetrics(t *testing.T) {
	cache := NewPageObjectCache(1024)

	// Add different types of content
	pageRef := ObjectRef{ObjectID: 1, Generation: 0}
	cache.Put(pageRef, "<< /Type /Page >>")

	imageRef := ObjectRef{ObjectID: 2, Generation: 0}
	cache.Put(imageRef, "<< /Subtype /Image >>")

	metrics := cache.GetDetailedMetrics()

	if len(metrics.ObjectTypes) == 0 {
		t.Error("Expected object types to be categorized")
	}

	if metrics.MemoryUsage.TotalBytes == 0 {
		t.Error("Expected non-zero memory usage")
	}
}

func TestExtractOptions_Validation(t *testing.T) {
	// Test default options
	options := ExtractOptions{
		ContentTypes: []string{"text", "images"},
	}

	if !containsString(options.ContentTypes, "text") {
		t.Error("Expected text to be in content types")
	}

	if !containsString(options.ContentTypes, "images") {
		t.Error("Expected images to be in content types")
	}

	if containsString(options.ContentTypes, "forms") {
		t.Error("Expected forms to not be in content types")
	}
}

func TestPageRange_Validation(t *testing.T) {
	reader := newMockPageRangeReader(testMultiPagePDF)
	extractor := NewPageRangeExtractor()

	// Build index first
	builder := NewPageIndexBuilder()
	index, err := builder.BuildFromReader(reader)
	if err != nil {
		t.Fatalf("Failed to build index: %v", err)
	}
	extractor.pageIndex = index

	// Test range validation
	ranges := []PageRange{
		{Start: -1, End: 2},  // Invalid start
		{Start: 1, End: 100}, // End beyond total pages
		{Start: 5, End: 2},   // Start > End
	}

	validRanges := extractor.validateRanges(ranges)

	// Should normalize and filter invalid ranges
	if len(validRanges) == 0 {
		t.Log("All ranges were invalid, which is expected for this test")
	}

	// Check that valid ranges are properly normalized
	for _, r := range validRanges {
		if r.Start < 1 {
			t.Errorf("Range start should be >= 1, got %d", r.Start)
		}
		if r.End > index.TotalPages {
			t.Errorf("Range end should be <= %d, got %d", index.TotalPages, r.End)
		}
		if r.Start > r.End {
			t.Errorf("Range start should be <= end, got %d > %d", r.Start, r.End)
		}
	}
}

// Benchmark tests
func BenchmarkPageRangeExtractor_ExtractRange(b *testing.B) {
	reader := newMockPageRangeReader(testMultiPagePDF)
	extractor := NewPageRangeExtractor()

	ranges := []PageRange{{Start: 1, End: 2}}
	options := ExtractOptions{ContentTypes: []string{"text"}}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reader.Seek(0, io.SeekStart)
		_, err := extractor.ExtractRange(reader, ranges, options)
		if err != nil {
			b.Fatalf("Extraction failed: %v", err)
		}
	}
}

func BenchmarkPageObjectCache_Put(b *testing.B) {
	cache := NewPageObjectCache(10 * 1024 * 1024) // 10MB cache

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		objRef := ObjectRef{ObjectID: i, Generation: 0}
		cache.Put(objRef, "test content")
	}
}

func BenchmarkPageObjectCache_Get(b *testing.B) {
	cache := NewPageObjectCache(10 * 1024 * 1024)

	// Pre-populate cache
	for i := 0; i < 1000; i++ {
		objRef := ObjectRef{ObjectID: i, Generation: 0}
		cache.Put(objRef, "test content")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		objRef := ObjectRef{ObjectID: i % 1000, Generation: 0}
		cache.Get(objRef)
	}
}

// Helper functions for testing

func createTestPageRange(start, end int) PageRange {
	return PageRange{Start: start, End: end}
}

func createTestExtractOptions(contentTypes []string) ExtractOptions {
	return ExtractOptions{
		ContentTypes:       contentTypes,
		PreserveFormatting: false,
		IncludeMetadata:    true,
		ExtractImages:      containsSlice(contentTypes, "images"),
		ExtractForms:       containsSlice(contentTypes, "forms"),
	}
}

func containsSlice(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// Test error conditions
func TestPageRangeExtractor_ErrorHandling(t *testing.T) {
	// Test with empty reader
	emptyReader := bytes.NewReader([]byte{})
	extractor := NewPageRangeExtractor()

	ranges := []PageRange{{Start: 1, End: 1}}
	options := ExtractOptions{ContentTypes: []string{"text"}}

	_, err := extractor.ExtractRange(emptyReader, ranges, options)
	if err == nil {
		t.Error("Expected error with empty reader")
	}
}

func TestPageIndex_Validation(t *testing.T) {
	index := NewPageIndex()
	index.TotalPages = 3
	index.PageObjects[1] = ObjectRef{ObjectID: 1}
	index.PageObjects[3] = ObjectRef{ObjectID: 3}
	// Missing page 2

	issues := index.ValidateIndex()
	if len(issues) == 0 {
		t.Error("Expected validation issues for incomplete index")
	}

	foundMissingPage := false
	for _, issue := range issues {
		if strings.Contains(issue, "Missing page object for page 2") {
			foundMissingPage = true
			break
		}
	}

	if !foundMissingPage {
		t.Error("Expected to find missing page 2 in validation issues")
	}
}
