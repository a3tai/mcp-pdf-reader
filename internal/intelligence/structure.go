package intelligence

import (
	"fmt"
	"math"
	"regexp"
	"sort"
	"strings"

	"github.com/a3tai/mcp-pdf-reader/internal/pdf/extraction"
)

// StructureType represents the type of document structure element
type StructureType string

const (
	StructureTypeDocument  StructureType = "document"
	StructureTypeSection   StructureType = "section"
	StructureTypeHeader    StructureType = "header"
	StructureTypeParagraph StructureType = "paragraph"
	StructureTypeList      StructureType = "list"
	StructureTypeListItem  StructureType = "list_item"
	StructureTypeTable     StructureType = "table"
	StructureTypeImage     StructureType = "image"
	StructureTypeFootnote  StructureType = "footnote"
	StructureTypeCaption   StructureType = "caption"
)

// StructureNode represents a node in the document structure tree
type StructureNode struct {
	ID           string                      `json:"id"`
	Type         StructureType               `json:"type"`
	Level        int                         `json:"level"` // For headers: 1-6
	Content      string                      `json:"content"`
	Elements     []extraction.ContentElement `json:"elements"`
	Children     []*StructureNode            `json:"children"`
	Parent       *StructureNode              `json:"-"` // Avoid circular JSON
	BoundingBox  *extraction.BoundingBox     `json:"bounding_box,omitempty"`
	Style        *TextStyle                  `json:"style,omitempty"`
	Confidence   float64                     `json:"confidence"`
	PageNumber   int                         `json:"page_number"`
	ReadingOrder int                         `json:"reading_order"`
}

// TextStyle represents text styling information
type TextStyle struct {
	FontName   string  `json:"font_name"`
	FontSize   float64 `json:"font_size"`
	IsBold     bool    `json:"is_bold"`
	IsItalic   bool    `json:"is_italic"`
	Color      string  `json:"color,omitempty"`
	Alignment  string  `json:"alignment"` // left, center, right, justify
	LineHeight float64 `json:"line_height"`
}

// DocumentStructure represents the analyzed document structure
type DocumentStructure struct {
	Root          *StructureNode           `json:"root"`
	Sections      []*StructureNode         `json:"sections"`
	Headers       map[int][]*StructureNode `json:"headers_by_level"`
	ReadingOrder  []*StructureNode         `json:"reading_order"`
	PageStructure map[int][]*StructureNode `json:"page_structure"`
	Statistics    *StructureStatistics     `json:"statistics"`
}

// StructureStatistics provides statistics about the document structure
type StructureStatistics struct {
	TotalNodes     int            `json:"total_nodes"`
	NodesByType    map[string]int `json:"nodes_by_type"`
	MaxDepth       int            `json:"max_depth"`
	AverageDepth   float64        `json:"average_depth"`
	HeaderCount    int            `json:"header_count"`
	ParagraphCount int            `json:"paragraph_count"`
	ListCount      int            `json:"list_count"`
	TableCount     int            `json:"table_count"`
	ImageCount     int            `json:"image_count"`
}

// StructureDetector analyzes document content to detect structure
type StructureDetector struct {
	debugMode bool
	config    StructureDetectionConfig
}

// StructureDetectionConfig configures structure detection behavior
type StructureDetectionConfig struct {
	MinHeaderFontSizeRatio   float64  // Minimum font size ratio to consider as header
	MaxHeaderLength          int      // Maximum length for header text
	MinParagraphLength       int      // Minimum length for paragraph text
	ListItemPatterns         []string // Patterns to detect list items
	IndentationThreshold     float64  // Threshold for detecting indentation
	LineSpacingThreshold     float64  // Threshold for paragraph separation
	SectionBreakThreshold    float64  // Vertical gap for section breaks
	EnableSmartGrouping      bool     // Enable smart content grouping
	EnableReadingOrderDetect bool     // Enable reading order detection
}

// DefaultStructureDetectionConfig returns default configuration
func DefaultStructureDetectionConfig() StructureDetectionConfig {
	return StructureDetectionConfig{
		MinHeaderFontSizeRatio:   1.2,
		MaxHeaderLength:          200,
		MinParagraphLength:       20,
		ListItemPatterns:         []string{`^\d+\.`, `^[•·●○▪▫◦‣⁃]`, `^[a-zA-Z]\)`, `^-\s`, `^\*\s`},
		IndentationThreshold:     10.0,
		LineSpacingThreshold:     1.5,
		SectionBreakThreshold:    30.0,
		EnableSmartGrouping:      true,
		EnableReadingOrderDetect: true,
	}
}

// NewStructureDetector creates a new structure detector
func NewStructureDetector(debugMode bool) *StructureDetector {
	return &StructureDetector{
		debugMode: debugMode,
		config:    DefaultStructureDetectionConfig(),
	}
}

// NewStructureDetectorWithConfig creates a new structure detector with custom config
func NewStructureDetectorWithConfig(config StructureDetectionConfig, debugMode bool) *StructureDetector {
	return &StructureDetector{
		debugMode: debugMode,
		config:    config,
	}
}

// DetectStructure analyzes content elements to detect document structure
func (sd *StructureDetector) DetectStructure(elements []extraction.ContentElement) (*DocumentStructure, error) {
	if len(elements) == 0 {
		return nil, fmt.Errorf("no content elements provided")
	}

	// Group elements by page
	pageElements := sd.groupByPage(elements)

	// Detect text styles and classify elements
	classifiedElements := sd.classifyElements(elements)

	// Build initial structure nodes
	nodes := sd.buildStructureNodes(classifiedElements)

	// Detect reading order
	if sd.config.EnableReadingOrderDetect {
		sd.detectReadingOrder(nodes)
	}

	// Build hierarchical structure
	root := sd.buildHierarchy(nodes)

	// Extract sections
	sections := sd.extractSections(root)

	// Group headers by level
	headersByLevel := sd.groupHeadersByLevel(root)

	// Build final structure
	structure := &DocumentStructure{
		Root:          root,
		Sections:      sections,
		Headers:       headersByLevel,
		ReadingOrder:  sd.getReadingOrder(root),
		PageStructure: sd.buildPageStructure(pageElements, root),
		Statistics:    sd.calculateStatistics(root),
	}

	return structure, nil
}

// groupByPage groups elements by page number
func (sd *StructureDetector) groupByPage(elements []extraction.ContentElement) map[int][]extraction.ContentElement {
	pageElements := make(map[int][]extraction.ContentElement)
	for _, elem := range elements {
		pageElements[elem.PageNumber] = append(pageElements[elem.PageNumber], elem)
	}
	return pageElements
}

// classifyElements analyzes elements to determine their structure type
func (sd *StructureDetector) classifyElements(elements []extraction.ContentElement) []classifiedElement {
	var classified []classifiedElement

	// Calculate average font size
	avgFontSize := sd.calculateAverageFontSize(elements)

	for _, elem := range elements {
		ce := classifiedElement{
			Element: elem,
			Style:   sd.extractStyle(elem),
		}

		// Classify based on content type and style
		switch elem.Type {
		case extraction.ContentTypeText:
			ce.StructureType = sd.classifyTextElement(elem, avgFontSize)
		case extraction.ContentTypeImage:
			ce.StructureType = StructureTypeImage
		default:
			ce.StructureType = StructureTypeParagraph
		}

		classified = append(classified, ce)
	}

	// Apply smart grouping if enabled
	if sd.config.EnableSmartGrouping {
		classified = sd.applySmartGrouping(classified)
	}

	return classified
}

// classifiedElement holds an element with its classified structure type
type classifiedElement struct {
	Element       extraction.ContentElement
	StructureType StructureType
	Style         *TextStyle
}

// extractStyle extracts text style from content element
func (sd *StructureDetector) extractStyle(elem extraction.ContentElement) *TextStyle {
	style := &TextStyle{
		Alignment: "left", // Default
	}

	// Extract style based on content type
	if elem.Type == extraction.ContentTypeText {
		if textElem, ok := elem.Content.(extraction.TextElement); ok {
			style.FontName = textElem.Properties.FontName
			style.FontSize = textElem.Properties.FontSize
			style.IsBold = strings.Contains(strings.ToLower(textElem.Properties.FontName), "bold")
			style.IsItalic = strings.Contains(strings.ToLower(textElem.Properties.FontName), "italic")

			// Detect alignment based on position
			if elem.BoundingBox.LowerLeft.X > 0 {
				// Simple alignment detection - can be enhanced
				pageWidth := 612.0 // Assume standard page width
				centerX := (elem.BoundingBox.LowerLeft.X + elem.BoundingBox.UpperRight.X) / 2
				if math.Abs(centerX-pageWidth/2) < 20 {
					style.Alignment = "center"
				} else if elem.BoundingBox.UpperRight.X > pageWidth-100 {
					style.Alignment = "right"
				}
			}
		}
	}

	return style
}

// calculateAverageFontSize calculates the average font size of text elements
func (sd *StructureDetector) calculateAverageFontSize(elements []extraction.ContentElement) float64 {
	var totalSize float64
	var count int

	for _, elem := range elements {
		if elem.Type == extraction.ContentTypeText {
			if textElem, ok := elem.Content.(extraction.TextElement); ok && textElem.Properties.FontSize > 0 {
				totalSize += textElem.Properties.FontSize
				count++
			}
		}
	}

	if count == 0 {
		return 12.0 // Default font size
	}

	return totalSize / float64(count)
}

// classifyTextElement classifies a text element based on its properties
func (sd *StructureDetector) classifyTextElement(elem extraction.ContentElement, avgFontSize float64) StructureType {
	textElem, ok := elem.Content.(extraction.TextElement)
	if !ok {
		return StructureTypeParagraph
	}

	text := strings.TrimSpace(textElem.Text)

	// Check for list items
	if sd.isListItem(text) {
		return StructureTypeListItem
	}

	// Check for headers based on font size and length
	// Also check for bold font as an indicator of headers
	isBold := strings.Contains(strings.ToLower(textElem.Properties.FontName), "bold")
	isLargeFont := textElem.Properties.FontSize > avgFontSize*sd.config.MinHeaderFontSizeRatio

	if (isLargeFont || isBold) && len(text) < sd.config.MaxHeaderLength {
		return StructureTypeHeader
	}

	// Check for captions (contains typical caption keywords, usually with smaller font)
	lowerText := strings.ToLower(text)
	if len(text) < 150 && (strings.Contains(lowerText, "figure") || strings.Contains(lowerText, "table") ||
		strings.Contains(lowerText, "fig.") || strings.Contains(lowerText, "image") ||
		strings.Contains(lowerText, "chart") || strings.Contains(lowerText, "diagram")) {
		// If it has caption keywords and is not too large, it's likely a caption
		// Also check if font is not larger than average (captions shouldn't be headers)
		if textElem.Properties.FontSize <= avgFontSize*1.1 {
			return StructureTypeCaption
		}
	}

	// Check for footnotes (small font size, bottom of page)
	if textElem.Properties.FontSize < avgFontSize*0.8 && elem.BoundingBox.LowerLeft.Y < 100 {
		return StructureTypeFootnote
	}

	// Default to paragraph
	return StructureTypeParagraph
}

// isListItem checks if text matches list item patterns
func (sd *StructureDetector) isListItem(text string) bool {
	// Check for numbered lists (1., 2., etc.)
	if matched, _ := regexp.MatchString(`^\d+\.`, text); matched {
		return true
	}

	// Check for lettered lists (a), b), etc.)
	if matched, _ := regexp.MatchString(`^[a-zA-Z]\)`, text); matched {
		return true
	}

	// Check for bullet points
	bulletPatterns := []string{"•", "·", "●", "○", "▪", "▫", "◦", "‣", "⁃", "-", "*"}
	for _, bullet := range bulletPatterns {
		if strings.HasPrefix(text, bullet+" ") || strings.HasPrefix(text, bullet+"\t") {
			return true
		}
	}

	return false
}

// buildStructureNodes creates structure nodes from classified elements
func (sd *StructureDetector) buildStructureNodes(classified []classifiedElement) []*StructureNode {
	var nodes []*StructureNode
	nodeID := 0

	for _, ce := range classified {
		nodeID++
		node := &StructureNode{
			ID:          fmt.Sprintf("node_%d", nodeID),
			Type:        ce.StructureType,
			Elements:    []extraction.ContentElement{ce.Element},
			Style:       ce.Style,
			Confidence:  ce.Element.Confidence,
			PageNumber:  ce.Element.PageNumber,
			BoundingBox: &ce.Element.BoundingBox,
		}

		// Extract content based on element type
		switch ce.Element.Type {
		case extraction.ContentTypeText:
			if textElem, ok := ce.Element.Content.(extraction.TextElement); ok {
				node.Content = textElem.Text

				// Determine header level based on font size
				if ce.StructureType == StructureTypeHeader {
					node.Level = sd.determineHeaderLevel(textElem.Properties.FontSize, classified)
				}
			}
		case extraction.ContentTypeImage:
			node.Content = "[Image]"
		}

		nodes = append(nodes, node)
	}

	return nodes
}

// determineHeaderLevel determines the header level (1-6) based on font size
func (sd *StructureDetector) determineHeaderLevel(fontSize float64, allElements []classifiedElement) int {
	// Collect all header font sizes
	var headerSizes []float64
	for _, ce := range allElements {
		if ce.StructureType == StructureTypeHeader {
			if textElem, ok := ce.Element.Content.(extraction.TextElement); ok {
				headerSizes = append(headerSizes, textElem.Properties.FontSize)
			}
		}
	}

	// Sort unique sizes in descending order
	uniqueSizes := sd.getUniqueSizes(headerSizes)
	sort.Float64s(uniqueSizes)
	for i, j := 0, len(uniqueSizes)-1; i < j; i, j = i+1, j-1 {
		uniqueSizes[i], uniqueSizes[j] = uniqueSizes[j], uniqueSizes[i]
	}

	// Find level based on size ranking
	for i, size := range uniqueSizes {
		if math.Abs(fontSize-size) < 0.1 {
			level := i + 1
			if level > 6 {
				return 6
			}
			return level
		}
	}

	return 3 // Default to level 3
}

// getUniqueSizes returns unique font sizes
func (sd *StructureDetector) getUniqueSizes(sizes []float64) []float64 {
	sizeMap := make(map[float64]bool)
	for _, size := range sizes {
		sizeMap[size] = true
	}

	var unique []float64
	for size := range sizeMap {
		unique = append(unique, size)
	}
	return unique
}

// detectReadingOrder determines the reading order of nodes
func (sd *StructureDetector) detectReadingOrder(nodes []*StructureNode) {
	// Sort by page, then by Y position (top to bottom), then by X position (left to right)
	sort.Slice(nodes, func(i, j int) bool {
		if nodes[i].PageNumber != nodes[j].PageNumber {
			return nodes[i].PageNumber < nodes[j].PageNumber
		}

		// Compare Y positions with threshold
		if nodes[i].BoundingBox != nil && nodes[j].BoundingBox != nil {
			yDiff := math.Abs(nodes[i].BoundingBox.UpperRight.Y - nodes[j].BoundingBox.UpperRight.Y)
			if yDiff > 5 { // 5 point threshold
				return nodes[i].BoundingBox.UpperRight.Y > nodes[j].BoundingBox.UpperRight.Y
			}
			// Same line, sort by X
			return nodes[i].BoundingBox.LowerLeft.X < nodes[j].BoundingBox.LowerLeft.X
		}

		return false
	})

	// Assign reading order
	for i, node := range nodes {
		node.ReadingOrder = i + 1
	}
}

// buildHierarchy builds a hierarchical structure from nodes
func (sd *StructureDetector) buildHierarchy(nodes []*StructureNode) *StructureNode {
	// Create root node
	root := &StructureNode{
		ID:       "root",
		Type:     StructureTypeDocument,
		Children: []*StructureNode{},
	}

	// Build hierarchy based on headers and sections
	currentSection := root
	headerStack := []*StructureNode{root}

	for _, node := range nodes {
		if node.Type == StructureTypeHeader {
			// Find appropriate parent based on header level
			for len(headerStack) > node.Level {
				headerStack = headerStack[:len(headerStack)-1]
			}

			parent := headerStack[len(headerStack)-1]
			parent.Children = append(parent.Children, node)
			node.Parent = parent

			// Update header stack
			if len(headerStack) == node.Level {
				headerStack[len(headerStack)-1] = node
			} else {
				headerStack = append(headerStack, node)
			}

			currentSection = node
		} else {
			// Add non-header nodes to current section
			currentSection.Children = append(currentSection.Children, node)
			node.Parent = currentSection
		}
	}

	return root
}

// applySmartGrouping groups related elements together
func (sd *StructureDetector) applySmartGrouping(elements []classifiedElement) []classifiedElement {
	// Group consecutive list items
	var grouped []classifiedElement
	var currentList *classifiedElement

	for i, elem := range elements {
		if elem.StructureType == StructureTypeListItem {
			if currentList == nil {
				// Start new list
				currentList = &classifiedElement{
					StructureType: StructureTypeList,
					Element: extraction.ContentElement{
						Type:        extraction.ContentTypeText,
						PageNumber:  elem.Element.PageNumber,
						BoundingBox: elem.Element.BoundingBox,
						Confidence:  elem.Element.Confidence,
						Content: extraction.TextElement{
							Text:       "",
							Properties: elem.Element.Content.(extraction.TextElement).Properties,
						},
					},
				}
			}
			// Always add to current list
			if textElem, ok := currentList.Element.Content.(extraction.TextElement); ok {
				if elemText, ok := elem.Element.Content.(extraction.TextElement); ok {
					if textElem.Text != "" {
						textElem.Text += "\n"
					}
					textElem.Text += elemText.Text
					currentList.Element.Content = textElem
				}
				// Update bounding box
				currentList.Element.BoundingBox = sd.mergeBoundingBoxes(
					currentList.Element.BoundingBox,
					elem.Element.BoundingBox,
				)
			}
		} else {
			// End current list if exists
			if currentList != nil {
				grouped = append(grouped, *currentList)
				currentList = nil
			}
			grouped = append(grouped, elem)
		}

		// Handle last element
		if i == len(elements)-1 && currentList != nil {
			grouped = append(grouped, *currentList)
		}
	}

	return grouped
}

// mergeBoundingBoxes merges two bounding boxes
func (sd *StructureDetector) mergeBoundingBoxes(box1, box2 extraction.BoundingBox) extraction.BoundingBox {
	return extraction.BoundingBox{
		LowerLeft: extraction.Coordinate{
			X: math.Min(box1.LowerLeft.X, box2.LowerLeft.X),
			Y: math.Min(box1.LowerLeft.Y, box2.LowerLeft.Y),
		},
		UpperRight: extraction.Coordinate{
			X: math.Max(box1.UpperRight.X, box2.UpperRight.X),
			Y: math.Max(box1.UpperRight.Y, box2.UpperRight.Y),
		},
		Width:  math.Max(box1.UpperRight.X, box2.UpperRight.X) - math.Min(box1.LowerLeft.X, box2.LowerLeft.X),
		Height: math.Max(box1.UpperRight.Y, box2.UpperRight.Y) - math.Min(box1.LowerLeft.Y, box2.LowerLeft.Y),
	}
}

// extractSections extracts top-level sections from the hierarchy
func (sd *StructureDetector) extractSections(root *StructureNode) []*StructureNode {
	var sections []*StructureNode

	for _, child := range root.Children {
		if child.Type == StructureTypeHeader && child.Level <= 2 {
			sections = append(sections, child)
		}
	}

	return sections
}

// groupHeadersByLevel groups headers by their level
func (sd *StructureDetector) groupHeadersByLevel(root *StructureNode) map[int][]*StructureNode {
	headers := make(map[int][]*StructureNode)

	sd.walkTree(root, func(node *StructureNode) {
		if node.Type == StructureTypeHeader {
			headers[node.Level] = append(headers[node.Level], node)
		}
	})

	return headers
}

// walkTree walks the structure tree and applies a function to each node
func (sd *StructureDetector) walkTree(node *StructureNode, fn func(*StructureNode)) {
	fn(node)
	for _, child := range node.Children {
		sd.walkTree(child, fn)
	}
}

// getReadingOrder extracts nodes in reading order
func (sd *StructureDetector) getReadingOrder(root *StructureNode) []*StructureNode {
	var ordered []*StructureNode

	// walkTree walks the structure tree and applies a function to each node
	sd.walkTree(root, func(node *StructureNode) {
		if node.Type != StructureTypeDocument && node.ReadingOrder > 0 {
			ordered = append(ordered, node)
		}
	})

	// Sort by reading order
	if len(ordered) > 0 {
		sort.Slice(ordered, func(i, j int) bool {
			return ordered[i].ReadingOrder < ordered[j].ReadingOrder
		})
	}

	return ordered
}

// buildPageStructure builds structure organized by page
func (sd *StructureDetector) buildPageStructure(pageElements map[int][]extraction.ContentElement, root *StructureNode) map[int][]*StructureNode {
	pageStructure := make(map[int][]*StructureNode)

	sd.walkTree(root, func(node *StructureNode) {
		if node.Type != StructureTypeDocument {
			pageStructure[node.PageNumber] = append(pageStructure[node.PageNumber], node)
		}
	})

	return pageStructure
}

// calculateStatistics calculates structure statistics
func (sd *StructureDetector) calculateStatistics(root *StructureNode) *StructureStatistics {
	stats := &StructureStatistics{
		NodesByType: make(map[string]int),
	}

	depths := []int{}
	sd.walkTreeWithDepth(root, 0, func(node *StructureNode, depth int) {
		stats.TotalNodes++
		stats.NodesByType[string(node.Type)]++
		depths = append(depths, depth)

		// Count specific types
		switch node.Type {
		case StructureTypeHeader:
			stats.HeaderCount++
		case StructureTypeParagraph:
			stats.ParagraphCount++
		case StructureTypeList:
			stats.ListCount++
		case StructureTypeTable:
			stats.TableCount++
		case StructureTypeImage:
			stats.ImageCount++
		}
	})

	// Calculate depth statistics
	if len(depths) > 0 {
		maxDepth := 0
		totalDepth := 0
		for _, d := range depths {
			if d > maxDepth {
				maxDepth = d
			}
			totalDepth += d
		}
		stats.MaxDepth = maxDepth
		stats.AverageDepth = float64(totalDepth) / float64(len(depths))
	}

	return stats
}

// walkTreeWithDepth walks the tree tracking depth
func (sd *StructureDetector) walkTreeWithDepth(node *StructureNode, depth int, fn func(*StructureNode, int)) {
	if node != nil {
		fn(node, depth)
		for _, child := range node.Children {
			sd.walkTreeWithDepth(child, depth+1, fn)
		}
	}
}

// GetTableOfContents generates a table of contents from the structure
func (sd *StructureDetector) GetTableOfContents(structure *DocumentStructure) []TOCEntry {
	var toc []TOCEntry

	// Walk the tree to find all headers
	sd.walkTree(structure.Root, func(node *StructureNode) {
		if node.Type == StructureTypeHeader && node.Content != "" {
			toc = append(toc, TOCEntry{
				Level:      node.Level,
				Title:      node.Content,
				PageNumber: node.PageNumber,
				NodeID:     node.ID,
			})
		}
	})

	// Sort by reading order if available, otherwise by page number
	sort.Slice(toc, func(i, j int) bool {
		// First compare by page number
		if toc[i].PageNumber != toc[j].PageNumber {
			return toc[i].PageNumber < toc[j].PageNumber
		}
		// Then by reading order (implied by the order we found them)
		return false
	})

	return toc
}

// TOCEntry represents a table of contents entry
type TOCEntry struct {
	Level      int    `json:"level"`
	Title      string `json:"title"`
	PageNumber int    `json:"page_number"`
	NodeID     string `json:"node_id"`
}
