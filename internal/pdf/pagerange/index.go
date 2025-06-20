package pagerange

import (
	"fmt"
	"io"
	"regexp"
	"sort"
	"strconv"
	"strings"
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
	index       *PageIndex
	objectCache map[string]string
	patterns    *IndexPatterns
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

	// Step 1: Find and parse the cross-reference table
	if err := pib.buildXRefTable(reader); err != nil {
		return nil, fmt.Errorf("failed to build xref table: %w", err)
	}

	// Step 2: Find the document catalog
	catalog, err := pib.findCatalog(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to find catalog: %w", err)
	}

	// Step 3: Build page tree from catalog
	if err := pib.buildPageTree(reader, catalog); err != nil {
		return nil, fmt.Errorf("failed to build page tree: %w", err)
	}

	// Step 4: Extract page-specific resources
	if err := pib.extractPageResources(reader); err != nil {
		return nil, fmt.Errorf("failed to extract page resources: %w", err)
	}

	pib.index.BuildTime = getCurrentTimeMillis() - startTime
	return pib.index, nil
}

// buildXRefTable builds the cross-reference table
func (pib *PageIndexBuilder) buildXRefTable(reader io.ReadSeeker) error {
	// Seek to end of file to find startxref
	_, err := reader.Seek(-1024, io.SeekEnd)
	if err != nil {
		return fmt.Errorf("failed to seek to end: %w", err)
	}

	// Read last part of file
	buffer := make([]byte, 1024)
	n, err := reader.Read(buffer)
	if err != nil && err != io.EOF {
		return fmt.Errorf("failed to read end of file: %w", err)
	}

	content := string(buffer[:n])

	// Find startxref offset
	startxrefRegex := regexp.MustCompile(`startxref\s*(\d+)`)
	matches := startxrefRegex.FindStringSubmatch(content)
	if len(matches) < 2 {
		return fmt.Errorf("startxref not found")
	}

	xrefOffset, err := strconv.ParseInt(matches[1], 10, 64)
	if err != nil {
		return fmt.Errorf("invalid xref offset: %w", err)
	}

	// Read xref table
	_, err = reader.Seek(xrefOffset, io.SeekStart)
	if err != nil {
		return fmt.Errorf("failed to seek to xref: %w", err)
	}

	xrefBuffer := make([]byte, 4096)
	n, err = reader.Read(xrefBuffer)
	if err != nil && err != io.EOF {
		return fmt.Errorf("failed to read xref: %w", err)
	}

	return pib.parseXRefTable(string(xrefBuffer[:n]))
}

// parseXRefTable parses the cross-reference table content
func (pib *PageIndexBuilder) parseXRefTable(content string) error {
	lines := strings.Split(content, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if line == "xref" {
			continue
		}

		if strings.HasPrefix(line, "trailer") {
			break
		}

		// Parse subsection header (start count)
		if strings.Contains(line, " ") && !strings.Contains(line, " n") && !strings.Contains(line, " f") {
			continue
		}

		// Parse xref entry
		if len(line) == 18 { // Standard xref entry length
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				offset, err := strconv.ParseInt(parts[0], 10, 64)
				if err != nil {
					continue
				}

				generation, err := strconv.Atoi(parts[1])
				if err != nil {
					continue
				}

				flag := parts[2]
				objID := len(pib.index.XRefTable)

				pib.index.XRefTable[objID] = XRefEntry{
					ObjectID:   objID,
					Generation: generation,
					Offset:     offset,
					InUse:      flag == "n",
					Type:       "normal",
				}
			}
		}
	}

	return nil
}

// findCatalog finds the document catalog object
func (pib *PageIndexBuilder) findCatalog(reader io.ReadSeeker) (ObjectRef, error) {
	// Look for catalog reference in trailer
	_, err := reader.Seek(-2048, io.SeekEnd)
	if err != nil {
		return ObjectRef{}, fmt.Errorf("failed to seek to trailer: %w", err)
	}

	buffer := make([]byte, 2048)
	n, err := reader.Read(buffer)
	if err != nil && err != io.EOF {
		return ObjectRef{}, fmt.Errorf("failed to read trailer: %w", err)
	}

	content := string(buffer[:n])

	// Find Root reference in trailer
	rootRegex := regexp.MustCompile(`/Root\s+(\d+)\s+(\d+)\s+R`)
	matches := rootRegex.FindStringSubmatch(content)
	if len(matches) >= 3 {
		objID, _ := strconv.Atoi(matches[1])
		generation, _ := strconv.Atoi(matches[2])

		// Find offset from xref table
		var offset int64
		if entry, exists := pib.index.XRefTable[objID]; exists {
			offset = entry.Offset
		}

		return ObjectRef{
			ObjectID:   objID,
			Generation: generation,
			Offset:     offset,
		}, nil
	}

	return ObjectRef{}, fmt.Errorf("catalog not found in trailer")
}

// buildPageTree builds the page tree structure
func (pib *PageIndexBuilder) buildPageTree(reader io.ReadSeeker, catalog ObjectRef) error {
	// Parse catalog object
	catalogContent, err := pib.parseObjectAt(reader, catalog)
	if err != nil {
		return fmt.Errorf("failed to parse catalog: %w", err)
	}

	// Find Pages reference
	pagesRegex := regexp.MustCompile(`/Pages\s+(\d+)\s+(\d+)\s+R`)
	matches := pagesRegex.FindStringSubmatch(catalogContent)
	if len(matches) < 3 {
		return fmt.Errorf("Pages reference not found in catalog")
	}

	pagesObjID, _ := strconv.Atoi(matches[1])
	pagesGeneration, _ := strconv.Atoi(matches[2])

	var pagesOffset int64
	if entry, exists := pib.index.XRefTable[pagesObjID]; exists {
		pagesOffset = entry.Offset
	}

	pagesRef := ObjectRef{
		ObjectID:   pagesObjID,
		Generation: pagesGeneration,
		Offset:     pagesOffset,
	}

	// Build page tree starting from root
	pageNum := 1
	pib.index.PageTree, err = pib.buildPageTreeNode(reader, pagesRef, nil, &pageNum)
	if err != nil {
		return fmt.Errorf("failed to build page tree: %w", err)
	}

	pib.index.TotalPages = pageNum - 1
	return nil
}

// buildPageTreeNode recursively builds a page tree node
func (pib *PageIndexBuilder) buildPageTreeNode(reader io.ReadSeeker, objRef ObjectRef, parent *PageTreeNode, pageNum *int) (*PageTreeNode, error) {
	// Parse the object
	content, err := pib.parseObjectAt(reader, objRef)
	if err != nil {
		return nil, fmt.Errorf("failed to parse object %d: %w", objRef.ObjectID, err)
	}

	node := &PageTreeNode{
		ObjectRef: objRef,
		Parent:    parent,
		Kids:      []*PageTreeNode{},
	}

	// Determine if this is a Pages or Page object
	if strings.Contains(content, "/Type /Pages") {
		node.Type = "Pages"

		// Find Kids array
		kidsRegex := regexp.MustCompile(`/Kids\s*\[\s*((?:\d+\s+\d+\s+R\s*)+)\]`)
		matches := kidsRegex.FindStringSubmatch(content)
		if len(matches) > 1 {
			// Parse kid references
			kidRefs := parseObjectReferences(matches[1])

			for _, kidRef := range kidRefs {
				// Find offset for kid
				if entry, exists := pib.index.XRefTable[kidRef.ObjectID]; exists {
					kidRef.Offset = entry.Offset
				}

				kidNode, err := pib.buildPageTreeNode(reader, kidRef, node, pageNum)
				if err != nil {
					continue // Skip problematic kids
				}
				node.Kids = append(node.Kids, kidNode)
			}
		}

	} else if strings.Contains(content, "/Type /Page") {
		node.Type = "Page"
		node.PageNum = *pageNum

		// Add to index
		pib.index.PageOffsets[*pageNum] = objRef.Offset
		pib.index.PageObjects[*pageNum] = objRef

		*pageNum++
	}

	return node, nil
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

// parseObjectAt parses an object at a specific location
func (pib *PageIndexBuilder) parseObjectAt(reader io.ReadSeeker, objRef ObjectRef) (string, error) {
	// Check cache first
	cacheKey := fmt.Sprintf("%d_%d", objRef.ObjectID, objRef.Generation)
	if cached, exists := pib.objectCache[cacheKey]; exists {
		return cached, nil
	}

	// Seek to object
	if objRef.Offset > 0 {
		_, err := reader.Seek(objRef.Offset, io.SeekStart)
		if err != nil {
			return "", fmt.Errorf("failed to seek to object: %w", err)
		}
	}

	// Read object content
	buffer := make([]byte, 8192)
	n, err := reader.Read(buffer)
	if err != nil && err != io.EOF {
		return "", fmt.Errorf("failed to read object: %w", err)
	}

	content := string(buffer[:n])

	// Extract object content between obj and endobj
	objRegex := regexp.MustCompile(fmt.Sprintf(`%d\s+%d\s+obj\s*(.*?)\s*endobj`, objRef.ObjectID, objRef.Generation))
	matches := objRegex.FindStringSubmatch(content)
	if len(matches) > 1 {
		result := matches[1]
		pib.objectCache[cacheKey] = result // Cache for future use
		return result, nil
	}

	return "", fmt.Errorf("object %d %d not found", objRef.ObjectID, objRef.Generation)
}

// Helper functions

// parseObjectReferences parses object references from a string
func parseObjectReferences(content string) []ObjectRef {
	var refs []ObjectRef

	refRegex := regexp.MustCompile(`(\d+)\s+(\d+)\s+R`)
	matches := refRegex.FindAllStringSubmatch(content, -1)

	for _, match := range matches {
		if len(match) >= 3 {
			objID, _ := strconv.Atoi(match[1])
			generation, _ := strconv.Atoi(match[2])

			refs = append(refs, ObjectRef{
				ObjectID:   objID,
				Generation: generation,
			})
		}
	}

	return refs
}

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
