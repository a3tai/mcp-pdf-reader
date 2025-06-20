package mcp

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/a3tai/mcp-pdf-reader/internal/config"
	"github.com/a3tai/mcp-pdf-reader/internal/pdf"
)

func TestNewServer(t *testing.T) {
	// Create temp directory for test
	tempDir, err := os.MkdirTemp("", "mcp_server_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	maxFileSize := int64(1024 * 1024)
	pdfService, err := pdf.NewService(maxFileSize, tempDir)
	if err != nil {
		t.Fatalf("Failed to create PDF service: %v", err)
	}

	tests := []struct {
		name        string
		config      *config.Config
		expectError bool
	}{
		{
			name: "valid stdio mode config",
			config: &config.Config{
				Mode:         "stdio",
				Host:         "127.0.0.1",
				Port:         8080,
				PDFDirectory: "/tmp",
				Version:      "1.0.0",
				ServerName:   "test-server",
				LogLevel:     "info",
				MaxFileSize:  maxFileSize,
			},
			expectError: false,
		},
		{
			name: "valid server mode config",
			config: &config.Config{
				Mode:         "server",
				Host:         "127.0.0.1",
				Port:         8080,
				PDFDirectory: "/tmp",
				Version:      "1.0.0",
				ServerName:   "test-server",
				LogLevel:     "info",
				MaxFileSize:  maxFileSize,
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, err := NewServer(tt.config, pdfService)

			if tt.expectError && err == nil {
				t.Errorf("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if !tt.expectError {
				if server == nil {
					t.Fatal("server should not be nil")
				}
				if server.config != tt.config {
					t.Error("server config not set correctly")
				}
				if server.pdfService != pdfService {
					t.Error("server pdfService not set correctly")
				}
				if server.mcpServer == nil {
					t.Error("mcpServer should be initialized")
				}
			}
		})
	}
}

func TestServer_HandlePDFValidateFile(t *testing.T) {
	// Create temp directory for test files
	tempDir, err := os.MkdirTemp("", "mcp_server_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test file
	testFile := filepath.Join(tempDir, "test.pdf")
	if err := os.WriteFile(testFile, make([]byte, 1024), 0o644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Setup server
	cfg := &config.Config{
		Mode:         "stdio",
		PDFDirectory: tempDir,
		Version:      "1.0.0",
		ServerName:   "test-server",
		MaxFileSize:  1024 * 1024,
	}
	pdfService, err := pdf.NewService(cfg.MaxFileSize, cfg.PDFDirectory)
	if err != nil {
		t.Fatalf("Failed to create PDF service: %v", err)
	}
	server, err := NewServer(cfg, pdfService)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	// Create request with real CallToolRequest
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"path": testFile,
			},
		},
	}

	// Test the handler
	result, err := server.handlePDFValidateFile(context.Background(), request)
	if err != nil {
		t.Fatalf("handler failed: %v", err)
	}

	if result == nil {
		t.Fatal("result should not be nil")
	}

	// The file should be invalid since it's not a real PDF
	resultText := extractTextFromResult(result)
	if !strings.Contains(resultText, "PDF validation failed") {
		t.Errorf("expected validation to fail, got: %s", resultText)
	}
}

func TestServer_HandlePDFSearchDirectory(t *testing.T) {
	// Create temp directory with PDF files
	tempDir, err := os.MkdirTemp("", "mcp_search_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test PDF files
	testFiles := []string{"doc1.pdf", "doc2.pdf", "report.txt"}
	for _, filename := range testFiles {
		filePath := filepath.Join(tempDir, filename)
		if err := os.WriteFile(filePath, make([]byte, 1024), 0o644); err != nil {
			t.Fatalf("failed to create test file %s: %v", filename, err)
		}
	}

	// Setup server
	cfg := &config.Config{
		Mode:         "stdio",
		PDFDirectory: tempDir,
		Version:      "1.0.0",
		ServerName:   "test-server",
		MaxFileSize:  1024 * 1024,
	}
	pdfService, err := pdf.NewService(cfg.MaxFileSize, cfg.PDFDirectory)
	if err != nil {
		t.Fatalf("Failed to create PDF service: %v", err)
	}
	server, err := NewServer(cfg, pdfService)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	// Create request with real CallToolRequest
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"directory": tempDir,
				"query":     "",
			},
		},
	}

	// Test the handler
	result, err := server.handlePDFSearchDirectory(context.Background(), request)
	if err != nil {
		t.Fatalf("handler failed: %v", err)
	}

	if result == nil {
		t.Fatal("result should not be nil")
	}

	// Verify content mentions the found PDF files
	resultText := extractTextFromResult(result)
	if !strings.Contains(resultText, "Found 2 PDF file(s)") {
		t.Errorf("content should mention 2 PDF files, got: %s", resultText)
	}
}

func TestServer_HandlePDFStatsDirectory(t *testing.T) {
	// Create temp directory with PDF files
	tempDir, err := os.MkdirTemp("", "mcp_stats_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test PDF files with different sizes
	testFiles := map[string]int{
		"small.pdf":  512,
		"medium.pdf": 1024,
		"large.pdf":  2048,
	}

	for filename, size := range testFiles {
		filePath := filepath.Join(tempDir, filename)
		if err := os.WriteFile(filePath, make([]byte, size), 0o644); err != nil {
			t.Fatalf("failed to create test file %s: %v", filename, err)
		}
	}

	// Setup server
	cfg := &config.Config{
		Mode:         "stdio",
		PDFDirectory: tempDir,
		Version:      "1.0.0",
		ServerName:   "test-server",
		MaxFileSize:  1024 * 1024,
	}
	pdfService, err := pdf.NewService(cfg.MaxFileSize, cfg.PDFDirectory)
	if err != nil {
		t.Fatalf("Failed to create PDF service: %v", err)
	}
	server, err := NewServer(cfg, pdfService)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	// Create request with real CallToolRequest
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"directory": tempDir,
			},
		},
	}

	// Test the handler
	result, err := server.handlePDFStatsDirectory(context.Background(), request)
	if err != nil {
		t.Fatalf("handler failed: %v", err)
	}

	if result == nil {
		t.Fatal("result should not be nil")
	}

	// Verify content mentions the statistics
	resultText := extractTextFromResult(result)
	if !strings.Contains(resultText, "Total PDF files: 3") {
		t.Errorf("content should mention 3 PDF files, got: %s", resultText)
	}
}

func TestServer_DefaultDirectory(t *testing.T) {
	// Create temp directory
	tempDir, err := os.MkdirTemp("", "pdf-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cfg := &config.Config{
		Mode:         "stdio",
		PDFDirectory: tempDir,
		Version:      "1.0.0",
		ServerName:   "test-server",
		MaxFileSize:  1024 * 1024,
	}
	pdfService, err := pdf.NewService(cfg.MaxFileSize, cfg.PDFDirectory)
	if err != nil {
		t.Fatalf("Failed to create PDF service: %v", err)
	}
	server, err := NewServer(cfg, pdfService)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	// Create request without directory (should use default)
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"query": "",
			},
		},
	}

	// Test search directory handler
	result, err := server.handlePDFSearchDirectory(context.Background(), request)
	if err != nil {
		t.Fatalf("handler failed: %v", err)
	}

	if result == nil {
		t.Fatal("result should not be nil")
	}

	// Verify it used the default directory
	resultText := extractTextFromResult(result)
	if !strings.Contains(resultText, tempDir) {
		t.Errorf("content should mention default directory %s, got: %s", tempDir, resultText)
	}
}

func TestServer_InvalidArguments(t *testing.T) {
	// Setup server
	cfg := &config.Config{
		Mode:         "stdio",
		PDFDirectory: "/tmp",
		Version:      "1.0.0",
		ServerName:   "test-server",
		MaxFileSize:  1024 * 1024,
	}
	pdfService, err := pdf.NewService(cfg.MaxFileSize, cfg.PDFDirectory)
	if err != nil {
		t.Fatalf("Failed to create PDF service: %v", err)
	}
	server, err := NewServer(cfg, pdfService)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	// Test with missing required arguments
	emptyRequest := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{},
		},
	}

	// Test each handler that requires arguments
	handlers := []struct {
		name    string
		handler func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error)
	}{
		{"PDFValidateFile", server.handlePDFValidateFile},
		{"PDFReadFile", server.handlePDFReadFile},
		{"PDFAssetsFile", server.handlePDFAssetsFile},
		{"PDFStatsFile", server.handlePDFStatsFile},
	}

	for _, h := range handlers {
		t.Run(h.name, func(t *testing.T) {
			result, err := h.handler(context.Background(), emptyRequest)
			if err != nil {
				t.Errorf("handler should not return error, got: %v", err)
			}
			if result == nil {
				t.Fatal("result should not be nil")
			}

			// Check if it's an error result
			resultText := extractTextFromResult(result)
			if !strings.Contains(resultText, "required") && !strings.Contains(resultText, "missing") && !strings.Contains(resultText, "error") {
				t.Errorf("expected error message for missing arguments, got: %s", resultText)
			}
		})
	}
}

func TestFormatMethods(t *testing.T) {
	// Setup server
	cfg := &config.Config{
		Mode:         "stdio",
		PDFDirectory: "/tmp",
		Version:      "1.0.0",
		ServerName:   "test-server",
		MaxFileSize:  1024 * 1024,
	}
	pdfService, err := pdf.NewService(cfg.MaxFileSize, cfg.PDFDirectory)
	if err != nil {
		t.Fatalf("Failed to create PDF service: %v", err)
	}
	server, err := NewServer(cfg, pdfService)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	// Test formatPDFSearchDirectoryResult
	searchResult := &pdf.PDFSearchDirectoryResult{
		Files: []pdf.FileInfo{
			{
				Name:         "test.pdf",
				Path:         "/tmp/test.pdf",
				Size:         1024,
				ModifiedTime: "2023-01-01 12:00:00",
			},
		},
		TotalCount:  1,
		Directory:   "/tmp",
		SearchQuery: "test",
	}

	formatted := server.formatPDFSearchDirectoryResult(searchResult)
	if !strings.Contains(formatted, "Found 1 PDF file(s)") {
		t.Error("formatted result should contain file count")
	}
	if !strings.Contains(formatted, "test.pdf") {
		t.Error("formatted result should contain filename")
	}

	// Test formatPDFStatsDirectoryResult
	statsResult := &pdf.PDFStatsDirectoryResult{
		Directory:        "/tmp",
		TotalFiles:       2,
		TotalSize:        2048,
		LargestFileSize:  1024,
		LargestFileName:  "large.pdf",
		SmallestFileSize: 512,
		SmallestFileName: "small.pdf",
		AverageFileSize:  1024,
	}

	formatted = server.formatPDFStatsDirectoryResult(statsResult)
	if !strings.Contains(formatted, "Total PDF files: 2") {
		t.Error("formatted result should contain total files")
	}
	if !strings.Contains(formatted, "large.pdf") {
		t.Error("formatted result should contain largest filename")
	}

	// Test formatPDFStatsFileResult
	fileStatsResult := &pdf.PDFStatsFileResult{
		Path:         "/tmp/test.pdf",
		Size:         1024,
		Pages:        5,
		ModifiedDate: "2023-01-01 12:00:00",
		Title:        "Test Document",
		Author:       "Test Author",
	}

	formatted = server.formatPDFStatsFileResult(fileStatsResult)
	if !strings.Contains(formatted, "Pages: 5") {
		t.Error("formatted result should contain page count")
	}
	if !strings.Contains(formatted, "Test Document") {
		t.Error("formatted result should contain title")
	}

	// Test formatPDFAssetsFileResult
	assetsResult := &pdf.PDFAssetsFileResult{
		Path: "/tmp/test.pdf",
		Images: []pdf.ImageInfo{
			{
				PageNumber: 1,
				Width:      800,
				Height:     600,
				Format:     "JPEG",
				Size:       50000,
			},
		},
		TotalCount: 1,
	}

	formatted = server.formatPDFAssetsFileResult(assetsResult)
	if !strings.Contains(formatted, "Total images found: 1") {
		t.Error("formatted result should contain image count")
	}
	if !strings.Contains(formatted, "800x600") {
		t.Error("formatted result should contain image dimensions")
	}
}

// Helper function to extract text from a CallToolResult
func extractTextFromResult(result *mcp.CallToolResult) string {
	if result == nil || len(result.Content) == 0 {
		return ""
	}

	// Try to extract text content
	for _, content := range result.Content {
		if textContent, ok := content.(mcp.TextContent); ok {
			return textContent.Text
		}
		// Handle pointer to TextContent as well
		if textContentPtr, ok := content.(*mcp.TextContent); ok {
			return textContentPtr.Text
		}
	}

	return ""
}
