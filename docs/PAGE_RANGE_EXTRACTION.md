# Page Range Extraction

This document describes the page range extraction functionality in the MCP PDF Reader, which enables efficient extraction of content from specific page ranges in large PDF documents without loading the entire file into memory.

## Overview

Page range extraction is designed for scenarios where you need to extract content from specific pages or page ranges in large PDF documents. It builds upon the streaming infrastructure to provide efficient, memory-conscious processing of targeted page content.

## Key Features

- **Selective Processing**: Extract only the pages you need
- **Memory Efficient**: Builds minimal page index without parsing all content
- **Streaming Based**: Leverages streaming parser for large file support
- **Object Caching**: Intelligent caching of PDF objects for performance
- **Multiple Content Types**: Extract text, images, forms, and metadata
- **Range Validation**: Automatic validation and normalization of page ranges

## When to Use Page Range Extraction

### Use Page Range Extraction When:
- You need specific pages from a large document (e.g., pages 100-110 from a 1000-page PDF)
- Processing entire document would be memory-intensive
- You know the exact page ranges you need
- Working with structured documents where specific sections are on known pages
- Batch processing scenarios where different ranges go to different systems

### Use Regular Extraction When:
- You need the entire document
- Document is small (<50MB)
- You need to analyze document structure across all pages
- Random access to all pages is required

### Use Streaming When:
- You need to process the entire large document sequentially
- Memory is extremely constrained
- You're doing full-document analysis

## API Reference

### Basic Page Range Extraction

```go
// Service method
req := PDFExtractPageRangeRequest{
    Path: "large-document.pdf",
    Ranges: []PageRangeSpec{
        {Start: 10, End: 20},
        {Start: 50, End: 55},
    },
    ContentTypes: []string{"text", "images"},
    ExtractImages: true,
    ExtractForms: false,
    IncludeMetadata: true,
}

result, err := service.ExtractPageRange(req)
if err != nil {
    return fmt.Errorf("page range extraction failed: %w", err)
}

// Access extracted content
for pageNum, pageContent := range result.Pages {
    fmt.Printf("Page %d: %s\n", pageNum, pageContent.Text)
    fmt.Printf("  Images: %d\n", len(pageContent.Images))
    fmt.Printf("  MediaBox: %+v\n", pageContent.Metadata.MediaBox)
}
```

### Direct Extractor Usage

```go
// Create extractor with custom configuration
config := pagerange.ExtractorConfig{
    MaxCacheSize:    100 * 1024 * 1024, // 100MB cache
    EnableCaching:   true,
    PreloadObjects:  true,
    ParallelEnabled: false,
}

extractor := pagerange.NewPageRangeExtractor(config)

// Define page ranges
ranges := []pagerange.PageRange{
    {Start: 1, End: 5},
    {Start: 10, End: 15},
}

// Configure extraction options
options := pagerange.ExtractOptions{
    ContentTypes:       []string{"text", "metadata"},
    PreserveFormatting: true,
    IncludeMetadata:    true,
    ExtractImages:      false,
    ExtractForms:       false,
    OutputFormat:       "json",
}

// Extract from file
result, err := extractor.ExtractFromFile("document.pdf", ranges, options)
if err != nil {
    return fmt.Errorf("extraction failed: %w", err)
}

// Process results
fmt.Printf("Total pages in document: %d\n", result.TotalPages)
fmt.Printf("Processing time: %d ms\n", result.Metadata.ProcessingTime)
fmt.Printf("Cache hit rate: %.1f%%\n", 
    float64(result.Metadata.CacheHits) / 
    float64(result.Metadata.CacheHits + result.Metadata.CacheMisses) * 100)
```

### Text-Only Extraction with Formatting

```go
options := pagerange.ExtractOptions{
    ContentTypes:       []string{"text"},
    PreserveFormatting: true,
    IncludeMetadata:    false,
    ExtractImages:      false,
    ExtractForms:       false,
}

result, err := extractor.ExtractRange(reader, ranges, options)
if err != nil {
    return err
}

// Access formatted text blocks
for pageNum, pageContent := range result.Pages {
    fmt.Printf("=== Page %d ===\n", pageNum)
    
    if options.PreserveFormatting {
        for _, block := range pageContent.TextBlocks {
            fmt.Printf("Text: %s\n", block.Text)
            fmt.Printf("  Position: (%.1f, %.1f)\n", block.X, block.Y)
            fmt.Printf("  Font: %s, Size: %.1f\n", block.FontName, block.FontSize)
        }
    } else {
        fmt.Printf("Text: %s\n", pageContent.Text)
    }
}
```

## Configuration Options

### ExtractorConfig

```go
type ExtractorConfig struct {
    MaxCacheSize    int64 // Maximum cache size in bytes (default: 50MB)
    EnableCaching   bool  // Whether to enable object caching (default: true)
    PreloadObjects  bool  // Whether to preload required objects (default: true)
    ParallelEnabled bool  // Whether to enable parallel processing (default: false)
}
```

### ExtractOptions

```go
type ExtractOptions struct {
    ContentTypes       []string // Content to extract: "text", "images", "forms", "metadata"
    PreserveFormatting bool     // Whether to preserve text formatting and positioning
    IncludeMetadata    bool     // Whether to include page metadata
    ExtractImages      bool     // Whether to extract image references
    ExtractForms       bool     // Whether to extract form fields
    OutputFormat       string   // Output format: "json", "xml", "plain"
}
```

## Page Range Specifications

### Single Page
```go
ranges := []pagerange.PageRange{
    {Start: 5, End: 5}, // Extract only page 5
}
```

### Continuous Range
```go
ranges := []pagerange.PageRange{
    {Start: 10, End: 20}, // Extract pages 10 through 20
}
```

### Multiple Ranges
```go
ranges := []pagerange.PageRange{
    {Start: 1, End: 5},   // First 5 pages
    {Start: 50, End: 55}, // Pages 50-55
    {Start: 100, End: 100}, // Just page 100
}
```

### Range Validation
- Invalid ranges (start > end) are automatically filtered out
- Ranges beyond document bounds are automatically clipped
- Start pages < 1 are normalized to 1
- End pages > total pages are normalized to total pages

## Architecture

### Components

1. **PageRangeExtractor**: High-level API for page range extraction
2. **PageIndex**: Efficient index of page locations and metadata
3. **PageObjectCache**: LRU cache for PDF objects with size limits
4. **StreamParser Integration**: Uses streaming parser for memory efficiency

### Processing Flow

1. **Index Building**: Create minimal page index without parsing all content
2. **Range Validation**: Validate and normalize requested page ranges
3. **Object Calculation**: Determine which PDF objects are needed for requested pages
4. **Object Preloading**: Optionally preload required objects into cache
5. **Content Extraction**: Extract content from specific pages based on options
6. **Result Assembly**: Combine extracted content into structured result

## Performance Considerations

### Memory Usage

The page range extractor uses significantly less memory than loading entire documents:

- **Page Index**: Minimal metadata about page locations (~1KB per page)
- **Object Cache**: Configurable cache for frequently accessed objects
- **Working Memory**: Temporary memory for processing current page
- **Total**: Typically 10-50MB regardless of document size

### Caching Strategy

```go
// Configure cache for optimal performance
config := pagerange.ExtractorConfig{
    MaxCacheSize:    100 * 1024 * 1024, // 100MB cache
    EnableCaching:   true,               // Enable caching
    PreloadObjects:  true,               // Preload for better performance
}

// Monitor cache performance
result, _ := extractor.ExtractFromFile(path, ranges, options)
hitRate := float64(result.Metadata.CacheHits) / 
          float64(result.Metadata.CacheHits + result.Metadata.CacheMisses) * 100
fmt.Printf("Cache hit rate: %.1f%%\n", hitRate)
```

### Performance Tips

1. **Enable Caching**: Always enable caching for better performance
2. **Optimize Cache Size**: Larger caches improve performance for complex documents
3. **Preload Objects**: Enable preloading when processing multiple ranges
4. **Batch Ranges**: Process multiple ranges in single call when possible
5. **Content Type Selection**: Only extract the content types you need

## Error Handling

### Common Errors

```go
result, err := extractor.ExtractFromFile(path, ranges, options)
if err != nil {
    if strings.Contains(err.Error(), "file does not exist") {
        return fmt.Errorf("PDF file not found: %w", err)
    }
    if strings.Contains(err.Error(), "failed to build page index") {
        return fmt.Errorf("PDF structure invalid: %w", err)
    }
    if strings.Contains(err.Error(), "failed to extract page") {
        return fmt.Errorf("page extraction failed: %w", err)
    }
    return fmt.Errorf("unexpected error: %w", err)
}
```

### Graceful Degradation

```go
// Handle partial failures gracefully
result, err := extractor.ExtractFromFile(path, ranges, options)
if err != nil {
    log.Printf("Extraction failed: %v", err)
    return err
}

// Check which pages were successfully extracted
for _, r := range ranges {
    for pageNum := r.Start; pageNum <= r.End; pageNum++ {
        if pageContent, exists := result.Pages[pageNum]; exists {
            log.Printf("Successfully extracted page %d", pageNum)
        } else {
            log.Printf("Failed to extract page %d", pageNum)
        }
    }
}
```

## Best Practices

### 1. Resource Management

```go
// Always use appropriate cache sizes
func extractWithOptimalCache(docSize int64) pagerange.ExtractorConfig {
    cacheSize := docSize / 10 // Use 10% of document size as cache
    if cacheSize < 10*1024*1024 {
        cacheSize = 10 * 1024 * 1024 // Minimum 10MB
    }
    if cacheSize > 200*1024*1024 {
        cacheSize = 200 * 1024 * 1024 // Maximum 200MB
    }
    
    return pagerange.ExtractorConfig{
        MaxCacheSize:    cacheSize,
        EnableCaching:   true,
        PreloadObjects:  true,
        ParallelEnabled: false,
    }
}
```

### 2. Range Optimization

```go
// Combine adjacent ranges for better performance
func optimizeRanges(ranges []pagerange.PageRange) []pagerange.PageRange {
    if len(ranges) <= 1 {
        return ranges
    }
    
    var optimized []pagerange.PageRange
    current := ranges[0]
    
    for i := 1; i < len(ranges); i++ {
        next := ranges[i]
        
        // If ranges are adjacent or overlapping, combine them
        if next.Start <= current.End + 1 {
            current.End = max(current.End, next.End)
        } else {
            optimized = append(optimized, current)
            current = next
        }
    }
    
    optimized = append(optimized, current)
    return optimized
}
```

### 3. Content Type Selection

```go
// Only extract what you need for better performance
func extractTextOnly(path string, ranges []pagerange.PageRange) (*pagerange.ExtractedContent, error) {
    options := pagerange.ExtractOptions{
        ContentTypes:       []string{"text"},    // Only text
        PreserveFormatting: false,               // No positioning needed
        IncludeMetadata:    false,               // No metadata needed
        ExtractImages:      false,               // Skip images
        ExtractForms:       false,               // Skip forms
    }
    
    extractor := pagerange.NewPageRangeExtractor()
    return extractor.ExtractFromFile(path, ranges, options)
}
```

### 4. Progress Monitoring

```go
// Monitor extraction progress for long operations
func extractWithProgress(path string, ranges []pagerange.PageRange) error {
    extractor := pagerange.NewPageRangeExtractor()
    
    start := time.Now()
    result, err := extractor.ExtractFromFile(path, ranges, options)
    duration := time.Since(start)
    
    if err != nil {
        return err
    }
    
    // Log performance metrics
    log.Printf("Extraction completed in %v", duration)
    log.Printf("Pages extracted: %d", len(result.Pages))
    log.Printf("Cache efficiency: %.1f%%", 
        float64(result.Metadata.CacheHits) / 
        float64(result.Metadata.CacheHits + result.Metadata.CacheMisses) * 100)
    log.Printf("Memory usage: %.1f MB", 
        float64(result.Metadata.MemoryUsage) / (1024*1024))
    
    return nil
}
```

## Integration with Other Features

### With Streaming Processing

```go
// Use page range for targeted extraction, streaming for full document
if len(targetPages) > 0 && len(targetPages) < totalPages/2 {
    // Use page range extraction for specific pages
    return service.ExtractPageRange(pageRangeRequest)
} else {
    // Use streaming for full or majority extraction
    return service.StreamProcessFile(streamRequest)
}
```

### With Regular Extraction

```go
// Fallback strategy
result, err := service.ExtractPageRange(pageRangeRequest)
if err != nil {
    log.Printf("Page range extraction failed, falling back to regular extraction: %v", err)
    return service.PDFReadFile(regularRequest)
}
```

## Limitations

### Current Limitations

- **Complex PDF Structures**: Some complex PDF structures may not be fully supported
- **Encrypted PDFs**: Limited support for encrypted documents
- **XFA Forms**: Advanced XFA forms may not be fully extracted
- **Compressed Objects**: Some compression formats may impact performance

### Workarounds

```go
// Validate document before processing
func validateDocumentForPageRange(path string) error {
    // Try to build a basic index
    extractor := pagerange.NewPageRangeExtractor()
    
    file, err := os.Open(path)
    if err != nil {
        return err
    }
    defer file.Close()
    
    // Test with single page to validate structure
    testRanges := []pagerange.PageRange{{Start: 1, End: 1}}
    testOptions := pagerange.ExtractOptions{
        ContentTypes: []string{"metadata"},
    }
    
    _, err = extractor.ExtractRange(file, testRanges, testOptions)
    return err
}
```

## Examples

### Complete Example: Report Section Extraction

```go
package main

import (
    "fmt"
    "log"
    
    "github.com/a3tai/mcp-pdf-reader/internal/pdf/pagerange"
)

func extractReportSections(reportPath string) error {
    // Configure for report processing
    config := pagerange.ExtractorConfig{
        MaxCacheSize:    50 * 1024 * 1024, // 50MB cache
        EnableCaching:   true,
        PreloadObjects:  true,
        ParallelEnabled: false,
    }
    
    extractor := pagerange.NewPageRangeExtractor(config)
    
    // Define report sections
    sections := map[string]pagerange.PageRange{
        "Executive Summary": {Start: 1, End: 3},
        "Financial Data":    {Start: 15, End: 25},
        "Appendix":         {Start: 50, End: 60},
    }
    
    // Extract each section
    for sectionName, pageRange := range sections {
        fmt.Printf("Extracting %s (pages %d-%d)...\n", 
            sectionName, pageRange.Start, pageRange.End)
        
        ranges := []pagerange.PageRange{pageRange}
        options := pagerange.ExtractOptions{
            ContentTypes:       []string{"text", "metadata"},
            PreserveFormatting: true,
            IncludeMetadata:    true,
            ExtractImages:      false,
            ExtractForms:       false,
        }
        
        result, err := extractor.ExtractFromFile(reportPath, ranges, options)
        if err != nil {
            log.Printf("Failed to extract %s: %v", sectionName, err)
            continue
        }
        
        // Process extracted content
        fmt.Printf("  Successfully extracted %d pages\n", len(result.Pages))
        
        for pageNum, pageContent := range result.Pages {
            fmt.Printf("  Page %d: %d characters\n", pageNum, len(pageContent.Text))
        }
        
        fmt.Printf("  Processing time: %d ms\n", result.Metadata.ProcessingTime)
    }
    
    return nil
}
```

This page range extraction system provides efficient, targeted content extraction from large PDF documents while maintaining low memory usage and high performance.