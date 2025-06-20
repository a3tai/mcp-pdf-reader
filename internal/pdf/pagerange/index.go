package pagerange

import (
	"fmt"
	"io"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/a3tai/mcp-pdf-reader/internal/pdf/streaming"
)

// PageIndex represents an efficient index of page locations and metadata
type PageIndex struct {
	TotalPages   int                 `json:"total_pages"`
	PageOffsets  map[int]int64       `json:"page_offsets"`  // Page number -> file offset
	PageObjects  map[int]ObjectRef   `json:"page_objects"`  // Page number -> object reference
	Resources    map[int][]ObjectRef `json:"resources"`     // Page number -> required resources
	PageTree     *PageTreeNode       `json:"page_tree"`     // Page tree structure
	XRefTable    map[int]XRefEntry   `json:"xref_table"`    // Cross-reference table
	BuildTime    int64               `json:"build_time"`    // Time taken to build index
	IndexVersion string              `json:"index_version"` // Version of indexing algorithm
}

// PageTreeNode represents a node in the PDF page tree
type PageTreeNode struct {
	ObjectRef ObjectRef       `json:"object_ref"`
	Type      string          `json:"type"`      // "Pages" or "Page"
	Parent    *PageTreeNode   `json:"parent"`    // Parent node
	Kids      []*PageTreeNode `json:"kids"`      // Child nodes
	Count     int             `json:"count"`     // Number of leaf nodes
	PageNum   int             `json:"page_num"`  // Page number (for leaf nodes)
	Resources ObjectRef       `json:"resources"` // Resources for this page/subtree
}

// XRefEntry represents an entry in the cross-reference table
type XRefEntry struct {
	ObjectID   int    `json:"object_id"`
	Generation int    `json:"generation"`
	Offset     int64  `json:"offset"`
	InUse      bool   `json:"in_use"`
	Type       string `json:"type"` // "free" or "normal"
}

// PageIndexBuilder builds page indexes efficiently
type PageIndexBuilder struct {
	index        *PageIndex
	objectCache  map[string]string
	patterns     *IndexPatterns
	streamParser *streaming.StreamParser
}

// IndexPatterns contains compiled regex patterns for efficient parsing
type IndexPatterns struct {
	PageObject     *regexp.Regexp
	PageTreeObject *regexp.Regexp
	XRefEntry      *regexp.Regexp
	ObjectRef      *regexp.Regexp
	ResourceRef    *regexp.Regexp
	CatalogRef     *regexp.Regexp
}

// IndexStats provides statistics about the page index
type IndexStats struct {
	TotalPages      int     `json:"total_pages"`
	IndexedPages    int     `json:"indexed_pages"`
	ResourceRefs    int     `json:"resource_refs"`
	CachedObjects   int     `json:"cached_objects"`
	MemoryUsage     int     `json:"memory_usage"`
	BuildTimeMs     int64   `json:"build_time_ms"`
	IndexEfficiency float64 `json:"index_efficiency"`
}

// NewPageIndex creates a new empty page index
func NewPageIndex() *PageIndex {
	return &PageIndex{
		PageOffsets:  make(map[int]int64),
		PageObjects:  make(map[int]ObjectRef),
		Resources:    make(map[int][]ObjectRef),
		XRefTable:    make(map[int]XRefEntry),
		IndexVersion: "1.0",
	}
}

// NewPageIndexBuilder creates a new page index builder
func NewPageIndexBuilder() *PageIndexBuilder {
	return &PageIndexBuilder{
		index:       NewPageIndex(),
		objectCache: make(map[string]string),
		patterns:    compileIndexPatterns(),
	}
}

// BuildFromReader builds a page index from a PDF reader
func (pib *PageIndexBuilder) BuildFromReader(reader io.ReadSeeker) (*PageIndex, error) {
	startTime := getCurrentTimeMillis()

	// Reset reader to beginning
	_, err := reader.Seek(0, io.SeekStart)
	if err != nil {
		return nil, fmt.Errorf("failed to seek to start: %w", err)
	}

	// Create StreamParser for object resolution
	pib.streamParser, err = streaming.NewStreamParser(reader, streaming.StreamOptions{
		ChunkSizeMB:     1,   // 1MB chunks
		MaxMemoryMB:     100, // 100MB max
		XRefCacheSize:   1000,
		ObjectCacheSize: 1000,
		GCTrigger:       0.8,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create stream parser: %w", err)
	}
	defer pib.streamParser.Close()

	// Step 1: Build page tree from catalog (StreamParser handles XRef internally)
	if err := pib.buildPageTreeFromCatalog(); err != nil {
		return nil, fmt.Errorf("failed to build page tree: %w", err)
	}

	// Step 2: Extract page-specific resources
	if err := pib.extractPageResources(reader); err != nil {
		return nil, fmt.Errorf("failed to extract page resources: %w", err)
	}

	pib.index.BuildTime = getCurrentTimeMillis() - startTime
	return pib.index, nil
}

// extractPageResources extracts resource references for each page
func (pib *PageIndexBuilder) extractPageResources(reader io.ReadSeeker) error {
	for pageNum := 1; pageNum <= pib.index.TotalPages; pageNum++ {
		pageObj, exists := pib.index.PageObjects[pageNum]
		if !exists {
			continue
		}

		resources, err := pib.extractResourcesForPage(reader, pageObj)
		if err != nil {
			continue // Skip pages with errors
		}

		pib.index.Resources[pageNum] = resources
	}

	return nil
}

// extractResourcesForPage extracts resources for a specific page
func (pib *PageIndexBuilder) extractResourcesForPage(reader io.ReadSeeker, pageObj ObjectRef) ([]ObjectRef, error) {
	content, err := pib.parseObjectAt(reader, pageObj)
	if err != nil {
		return nil, err
	}

	var resources []ObjectRef

	// Extract Contents references
	contentRegex := regexp.MustCompile(`/Contents\s+(?:\[([^\]]+)\]|(\d+)\s+(\d+)\s+R)`)
	matches := contentRegex.FindAllStringSubmatch(content, -1)
	for _, match := range matches {
		if len(match) > 1 && match[1] != "" {
			// Array of content streams
			refs := parseObjectReferences(match[1])
			resources = append(resources, refs...)
		} else if len(match) >= 4 {
			// Single content stream
			objID, _ := strconv.Atoi(match[2])
			generation, _ := strconv.Atoi(match[3])

			var offset int64
			if entry, exists := pib.index.XRefTable[objID]; exists {
				offset = entry.Offset
			}

			resources = append(resources, ObjectRef{
				ObjectID:   objID,
				Generation: generation,
				Offset:     offset,
			})
		}
	}

	// Extract Resources reference
	resourceRegex := regexp.MustCompile(`/Resources\s+(\d+)\s+(\d+)\s+R`)
	matches = resourceRegex.FindAllStringSubmatch(content, -1)
	for _, match := range matches {
		if len(match) >= 3 {
			objID, _ := strconv.Atoi(match[1])
			generation, _ := strconv.Atoi(match[2])

			var offset int64
			if entry, exists := pib.index.XRefTable[objID]; exists {
				offset = entry.Offset
			}

			resources = append(resources, ObjectRef{
				ObjectID:   objID,
				Generation: generation,
				Offset:     offset,
			})
		}
	}

	return resources, nil
}

// buildPageTreeFromCatalog builds the page tree using the StreamParser
func (pib *PageIndexBuilder) buildPageTreeFromCatalog() error {
	// Get the catalog object (object 1 in most PDFs)
	catalogObj, err := pib.streamParser.GetObject(1, 0)
	if err != nil {
		return fmt.Errorf("failed to get catalog object: %w", err)
	}

	// Extract Pages reference from catalog
	pagesRegex := regexp.MustCompile(`/Pages\s+(\d+)\s+(\d+)\s+R`)
	matches := pagesRegex.FindStringSubmatch(catalogObj.Content)
	if len(matches) < 3 {
		return fmt.Errorf("Pages reference not found in catalog")
	}

	pagesObjID, _ := strconv.Atoi(matches[1])
	pagesGeneration, _ := strconv.Atoi(matches[2])

	// Get the Pages object
	pagesObj, err := pib.streamParser.GetObject(pagesObjID, pagesGeneration)
	if err != nil {
		return fmt.Errorf("failed to get Pages object %d %d: %w", pagesObjID, pagesGeneration, err)
	}

	// Parse the page tree
	pageNum := 1
	err = pib.parsePageTreeNode(pagesObj.Content, &pageNum, pagesObjID)
	if err != nil {
		return fmt.Errorf("failed to parse page tree: %w", err)
	}

	pib.index.TotalPages = pageNum - 1
	return nil
}

// parsePageTreeNode recursively parses the page tree structure
func (pib *PageIndexBuilder) parsePageTreeNode(content string, pageNum *int, objID int) error {
	// Check if this is a Pages node or a Page node
	// Be more specific: check for "/Type /Page" but not "/Type /Pages"
	if strings.Contains(content, "/Type /Page") && !strings.Contains(content, "/Type /Pages") {
		// This is a Page object - add to index
		objRef := ObjectRef{
			ObjectID:   objID,
			Generation: 0,
			Offset:     0, // Will be filled by XRef table
		}

		pib.index.PageObjects[*pageNum] = objRef
		pib.index.PageOffsets[*pageNum] = 0 // Will be filled by XRef table
		*pageNum++
		return nil
	}

	// This is a Pages node - find Kids array
	kidsRegex := regexp.MustCompile(`/Kids\s*\[\s*((?:\d+\s+\d+\s+R\s*)+)\]`)
	matches := kidsRegex.FindStringSubmatch(content)
	if len(matches) < 2 {
		return fmt.Errorf("Kids array not found in Pages object")
	}

	// Parse kid references
	kidRefs := parseObjectReferences(matches[1])
	for _, kidRef := range kidRefs {
		// Get the kid object
		kidObj, err := pib.streamParser.GetObject(kidRef.ObjectID, kidRef.Generation)
		if err != nil {
			continue // Skip problematic kids
		}

		// Recursively parse the kid
		err = pib.parsePageTreeNode(kidObj.Content, pageNum, kidRef.ObjectID)
		if err != nil {
			continue // Skip problematic kids
		}
	}

	return nil
}

// parseObjectAt parses an object at a specific location using StreamParser
func (pib *PageIndexBuilder) parseObjectAt(reader io.ReadSeeker, objRef ObjectRef) (string, error) {
	// Check cache first
	cacheKey := fmt.Sprintf("%d_%d", objRef.ObjectID, objRef.Generation)
	if cached, exists := pib.objectCache[cacheKey]; exists {
		return cached, nil
	}

	// Use StreamParser to get the object
	pdfObj, err := pib.streamParser.GetObject(objRef.ObjectID, objRef.Generation)
	if err != nil {
		return "", fmt.Errorf("failed to get object %d %d: %w", objRef.ObjectID, objRef.Generation, err)
	}

	// Cache for future use
	pib.objectCache[cacheKey] = pdfObj.Content
	return pdfObj.Content, nil
}

// Helper functions

// compileIndexPatterns compiles regex patterns for efficient parsing
func compileIndexPatterns() *IndexPatterns {
	return &IndexPatterns{
		PageObject:     regexp.MustCompile(`(\d+)\s+(\d+)\s+obj\s*<<[^>]*?/Type\s*/Page[^>]*?>>`),
		PageTreeObject: regexp.MustCompile(`(\d+)\s+(\d+)\s+obj\s*<<[^>]*?/Type\s*/Pages[^>]*?>>`),
		XRefEntry:      regexp.MustCompile(`(\d{10})\s+(\d{5})\s+([nf])`),
		ObjectRef:      regexp.MustCompile(`(\d+)\s+(\d+)\s+R`),
		ResourceRef:    regexp.MustCompile(`/Resources\s+(\d+)\s+(\d+)\s+R`),
		CatalogRef:     regexp.MustCompile(`/Root\s+(\d+)\s+(\d+)\s+R`),
	}
}

// Page index utility methods

// GetPage returns the page information for a specific page number
func (pi *PageIndex) GetPage(pageNum int) (ObjectRef, bool) {
	obj, exists := pi.PageObjects[pageNum]
	return obj, exists
}

// GetPageRange returns page objects for a range of pages
func (pi *PageIndex) GetPageRange(start, end int) map[int]ObjectRef {
	result := make(map[int]ObjectRef)

	for pageNum := start; pageNum <= end && pageNum <= pi.TotalPages; pageNum++ {
		if obj, exists := pi.PageObjects[pageNum]; exists {
			result[pageNum] = obj
		}
	}

	return result
}

// GetResourcesForPage returns resource references for a specific page
func (pi *PageIndex) GetResourcesForPage(pageNum int) ([]ObjectRef, bool) {
	resources, exists := pi.Resources[pageNum]
	return resources, exists
}

// GetStats returns statistics about the page index
func (pi *PageIndex) GetStats() IndexStats {
	resourceCount := 0
	for _, resources := range pi.Resources {
		resourceCount += len(resources)
	}

	return IndexStats{
		TotalPages:      pi.TotalPages,
		IndexedPages:    len(pi.PageObjects),
		ResourceRefs:    resourceCount,
		CachedObjects:   len(pi.XRefTable),
		BuildTimeMs:     pi.BuildTime,
		IndexEfficiency: float64(len(pi.PageObjects)) / float64(pi.TotalPages) * 100,
	}
}

// GetPagesInRange returns page numbers within a specific range
func (pi *PageIndex) GetPagesInRange(ranges []PageRange) []int {
	var pages []int
	pageSet := make(map[int]bool)

	for _, r := range ranges {
		for pageNum := r.Start; pageNum <= r.End && pageNum <= pi.TotalPages; pageNum++ {
			if !pageSet[pageNum] {
				pages = append(pages, pageNum)
				pageSet[pageNum] = true
			}
		}
	}

	sort.Ints(pages)
	return pages
}

// ValidateIndex validates the integrity of the page index
func (pi *PageIndex) ValidateIndex() []string {
	var issues []string

	// Check for missing pages
	for pageNum := 1; pageNum <= pi.TotalPages; pageNum++ {
		if _, exists := pi.PageObjects[pageNum]; !exists {
			issues = append(issues, fmt.Sprintf("Missing page object for page %d", pageNum))
		}

		if _, exists := pi.PageOffsets[pageNum]; !exists {
			issues = append(issues, fmt.Sprintf("Missing page offset for page %d", pageNum))
		}
	}

	// Check for orphaned entries
	for pageNum := range pi.PageObjects {
		if pageNum > pi.TotalPages {
			issues = append(issues, fmt.Sprintf("Page object exists for page %d but total pages is %d", pageNum, pi.TotalPages))
		}
	}

	return issues
}
