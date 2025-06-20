# Streaming PDF Processing

This document describes the streaming PDF processing functionality in the MCP PDF Reader, which enables efficient processing of large PDF files without loading them entirely into memory.

## Overview

The streaming PDF processor is designed to handle arbitrarily large PDF files by processing them in chunks, implementing efficient memory management and progressive parsing techniques. This is particularly useful for:

- Large PDF files (>100MB)
- Memory-constrained environments
- Batch processing scenarios
- Server environments with multiple concurrent requests

## When to Use Streaming

### Use Streaming When:
- File size exceeds available memory
- Processing multiple large files concurrently
- Memory usage needs to be predictable and limited
- Processing files of unknown size
- Working in resource-constrained environments

### Use Regular Processing When:
- Files are small (<50MB)
- Full document analysis is needed upfront
- Random access to pages is required
- Memory is abundant and speed is priority

## Architecture

The streaming system consists of several key components:

### StreamParser
- Handles low-level chunk reading and seeking
- Manages memory buffers and caching
- Provides pattern matching across chunk boundaries

### ChunkProcessor
- Processes individual chunks for PDF objects
- Extracts content progressively
- Manages content buffers (text, images, forms)

### StreamingExtractor
- High-level API for streaming operations
- Orchestrates parser and processor
- Provides progress reporting and error handling

### PageStreamer
- Specialized for page-by-page processing
- Discovers and processes pages individually
- Supports selective page range processing

## API Reference

### Basic Streaming Extraction

```go
// Create streaming extractor with default configuration
extractor := streaming.NewStreamingExtractor()

// Process file
ctx := context.Background()
result, err := extractor.ExtractFromFile(ctx, "large-document.pdf")
if err != nil {
    return fmt.Errorf("streaming failed: %w", err)
}

// Access extracted content
text := result.Content.Text
images := result.Content.Images
forms := result.Content.Forms
```

### Text-Only Streaming

```go
// Stream text to writer (memory efficient)
file, _ := os.Open("document.pdf")
defer file.Close()

var output bytes.Buffer
err := extractor.ExtractTextStream(ctx, file, &output)
if err != nil {
    return err
}

extractedText := output.String()
```

### Page-by-Page Processing

```go
// Create page streamer
parser, _ := streaming.NewStreamParser(file)
streamer := streaming.NewPageStreamer(parser)

// Process each page individually
err := streamer.StreamPages(ctx, func(page *streaming.StreamPage) error {
    fmt.Printf("Page %d: %d characters\n", page.Number, len(page.Content.Text))
    
    // Process page content as needed
    if len(page.Content.Images) > 0 {
        fmt.Printf("  Found %d images\n", len(page.Content.Images))
    }
    
    return nil
})
```

### Configuration Options

```go
// Custom configuration
config := streaming.StreamingConfig{
    ChunkSize:      2 * 1024 * 1024, // 2MB chunks
    MaxMemory:      64 * 1024 * 1024, // 64MB max memory
    ExtractText:    true,
    ExtractImages:  true,
    ExtractForms:   true,
    PreserveFormat: false,
    EnableCaching:  true,
    CacheSize:      1000,
    BufferPoolSize: 10,
}

extractor := streaming.NewStreamingExtractor(config)
```

### Builder Pattern Configuration

```go
extractor, err := streaming.NewStreamingOptions().
    WithChunkSize(2 * 1024 * 1024).        // 2MB chunks
    WithMaxMemory(64 * 1024 * 1024).       // 64MB max memory
    WithTextExtraction(true).
    WithImageExtraction(false).             // Skip images for speed
    WithFormExtraction(true).
    WithCaching(true, 1000).               // Enable caching with 1000 entries
    Build()
```

### Progress Reporting

```go
result, err := extractor.ExtractWithProgress(ctx, file, fileSize,
    func(progress streaming.ProcessingProgress) {
        percent := float64(progress.CurrentPage) / float64(progress.TotalPages) * 100
        fmt.Printf("Progress: %.1f%% (Page %d/%d)\n", 
            percent, progress.CurrentPage, progress.TotalPages)
        fmt.Printf("  Text: %d bytes, Images: %d, Forms: %d\n",
            progress.TextSize, progress.ImageCount, progress.FormCount)
    })
```

## Service Integration

The streaming functionality is integrated into the main PDF service:

### Stream Process File

```go
req := PDFStreamProcessRequest{
    Path:          "large-document.pdf",
    ExtractText:   true,
    ExtractImages: true,
    ExtractForms:  true,
    Config: &StreamingConfig{
        ChunkSizeMB: 2,
        MaxMemoryMB: 64,
        CacheSize:   1000,
    },
}

result, err := service.StreamProcessFile(req)
```

### Stream Process Pages

```go
req := PDFStreamPageRequest{
    Path:        "document.pdf",
    StartPage:   10,
    EndPage:     20,
    ExtractText: true,
}

result, err := service.StreamProcessPages(req)
```

### Stream Extract Text

```go
req := PDFStreamTextRequest{
    Path:       "document.pdf",
    OutputPath: "extracted-text.txt", // Optional: stream to file
}

result, err := service.StreamExtractText(req)
```

## Memory Management

### Memory Limits
The streaming system respects configured memory limits:

```go
// Estimate memory usage for a file
estimate := extractor.EstimateMemoryUsage(fileSize)
fmt.Printf("Estimated memory usage: %d bytes\n", estimate.TotalEstimate)
fmt.Printf("Recommendation: %s\n", estimate.Recommendation)
```

### Memory Components
- **Parser Memory**: Buffers and chunk processing (typically 25% of max)
- **Content Buffers**: Text, image, and form buffers (typically 25% of max)
- **Cache Memory**: Object and xref caches (typically 5-10% of max)
- **Working Memory**: Temporary allocations during processing

### Garbage Collection
The system automatically triggers garbage collection when memory usage exceeds the configured threshold (default: 80%).

## Performance Considerations

### Chunk Size Selection
- **Small chunks (64KB-256KB)**: Lower memory usage, more overhead
- **Medium chunks (1MB-4MB)**: Balanced performance (recommended)
- **Large chunks (8MB+)**: Higher memory usage, fewer seek operations

### Caching Strategy
- **XRef Cache**: Speeds up object lookups
- **Object Cache**: Reduces re-parsing of frequently accessed objects
- **Buffer Pool**: Reuses memory allocations

### Optimization Tips

1. **Disable unused extraction**: Skip images/forms if only text is needed
2. **Adjust chunk size**: Larger chunks for sequential access, smaller for random access
3. **Use page streaming**: For page-specific processing
4. **Monitor memory usage**: Use progress callbacks to track resource usage

## Error Handling

### Common Errors
- **Memory limit exceeded**: Reduce chunk size or increase memory limit
- **Invalid PDF structure**: Some PDFs may not be streamable
- **IO errors**: Handle network/disk issues gracefully

### Error Recovery
```go
result, err := extractor.ExtractFromFile(ctx, filePath)
if err != nil {
    if strings.Contains(err.Error(), "memory limit") {
        // Retry with smaller chunk size
        config := extractor.GetConfiguration()
        config.ChunkSize = config.ChunkSize / 2
        extractor.UpdateConfiguration(config)
        return extractor.ExtractFromFile(ctx, filePath)
    }
    return fmt.Errorf("extraction failed: %w", err)
}
```

## Best Practices

### 1. Resource Management
```go
// Always close resources
parser, err := streaming.NewStreamParser(file)
if err != nil {
    return err
}
defer parser.Close()
```

### 2. Context Usage
```go
// Use context for cancellation
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
defer cancel()

result, err := extractor.ExtractFromReader(ctx, file, fileSize)
```

### 3. Configuration Validation
```go
// Validate configuration before use
if err := extractor.ValidateConfiguration(); err != nil {
    return fmt.Errorf("invalid configuration: %w", err)
}
```

### 4. Progress Monitoring
```go
// Monitor progress for long-running operations
extractor.ExtractWithProgress(ctx, file, fileSize,
    func(progress streaming.ProcessingProgress) {
        if progress.CurrentPage%100 == 0 {
            log.Printf("Processed %d pages", progress.CurrentPage)
        }
    })
```

### 5. Error Logging
```go
// Log detailed error information
if err != nil {
    log.Printf("Streaming failed: %v", err)
    log.Printf("Memory stats: %+v", parser.GetMemoryUsage())
    return err
}
```

## Limitations

### Current Limitations
- **Encrypted PDFs**: Limited support for encrypted documents
- **Complex Forms**: Some XFA forms may not be fully supported
- **Compressed Streams**: Some compression formats may require full decompression
- **Cross-References**: Damaged xref tables may cause issues

### Workarounds
- **Pre-process**: Use PDF repair tools for damaged files
- **Fallback**: Fall back to regular processing for unsupported files
- **Validation**: Always validate PDFs before streaming

## Performance Benchmarks

### Memory Usage
- **Regular processing**: ~2-3x file size in memory
- **Streaming processing**: Configurable, typically 64-128MB regardless of file size

### Processing Speed
- **Small files (<10MB)**: Regular processing ~2x faster
- **Large files (>100MB)**: Streaming processing enables processing of otherwise impossible files
- **Very large files (>1GB)**: Only viable with streaming

## Examples

### Complete Example: Large File Processing
```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"
    "time"

    "github.com/a3tai/mcp-pdf-reader/internal/pdf/streaming"
)

func processLargeFile(filePath string) error {
    // Open file
    file, err := os.Open(filePath)
    if err != nil {
        return fmt.Errorf("failed to open file: %w", err)
    }
    defer file.Close()

    // Get file size
    info, err := file.Stat()
    if err != nil {
        return fmt.Errorf("failed to get file info: %w", err)
    }

    // Configure streaming for large files
    config := streaming.StreamingConfig{
        ChunkSize:      4 * 1024 * 1024, // 4MB chunks
        MaxMemory:      128 * 1024 * 1024, // 128MB max memory
        ExtractText:    true,
        ExtractImages:  false, // Skip images for speed
        ExtractForms:   true,
        EnableCaching:  true,
        CacheSize:      2000,
        BufferPoolSize: 20,
    }

    extractor := streaming.NewStreamingExtractor(config)

    // Create context with timeout
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
    defer cancel()

    // Process with progress reporting
    result, err := extractor.ExtractWithProgress(ctx, file, info.Size(),
        func(progress streaming.ProcessingProgress) {
            if progress.CurrentPage%50 == 0 {
                log.Printf("Progress: Page %d, Text: %d KB, Memory: %.1f%%",
                    progress.CurrentPage,
                    progress.TextSize/1024,
                    float64(progress.ObjectsFound)/1000*100)
            }
        })

    if err != nil {
        return fmt.Errorf("streaming extraction failed: %w", err)
    }

    // Log results
    log.Printf("Extraction completed:")
    log.Printf("  Text length: %d characters", len(result.Content.Text))
    log.Printf("  Images found: %d", len(result.Content.Images))
    log.Printf("  Forms found: %d", len(result.Content.Forms))
    log.Printf("  Processing time: %d ms", result.ProcessingStats.ProcessingTime)
    log.Printf("  Memory peak: %.1f MB", 
        float64(result.MemoryStats.CurrentBytes)/(1024*1024))

    return nil
}
```

This streaming system enables the MCP PDF Reader to handle large PDF files efficiently while maintaining predictable memory usage and providing detailed progress feedback.