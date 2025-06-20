package pdf

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestServerInfo(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "pdf-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a test PDF file (empty file for this test)
	testPDFPath := filepath.Join(tempDir, "test.pdf")
	if err := os.WriteFile(testPDFPath, []byte("%PDF-1.4\n1 0 obj\n<<\n/Type /Catalog\n/Pages 2 0 R\n>>\nendobj\n2 0 obj\n<<\n/Type /Pages\n/Kids [3 0 R]\n/Count 1\n>>\nendobj\n3 0 obj\n<<\n/Type /Page\n/Parent 2 0 R\n/MediaBox [0 0 612 792]\n>>\nendobj\nxref\n0 4\n0000000000 65535 f \n0000000009 00000 n \n0000000074 00000 n \n0000000120 00000 n \ntrailer\n<<\n/Size 4\n/Root 1 0 R\n>>\nstartxref\n197\n%%EOF"), 0o644); err != nil {
		t.Fatalf("Failed to create test PDF: %v", err)
	}

	// Test configuration values
	maxFileSize := int64(100 * 1024 * 1024) // 100MB
	serverName := "test-pdf-server"
	version := "1.0.0-test"

	// Create PDF service
	pdfService, err := NewService(maxFileSize, tempDir)
	if err != nil {
		t.Fatalf("Failed to create PDF service: %v", err)
	}

	// Test server info functionality
	req := PDFServerInfoRequest{}
	result, err := pdfService.PDFServerInfo(context.Background(), req, serverName, version, tempDir)
	if err != nil {
		t.Fatalf("Server info failed: %v", err)
	}

	// Verify the result contains expected information
	if result.ServerName != serverName {
		t.Errorf("Expected server name %s, got %s", serverName, result.ServerName)
	}

	if result.Version != version {
		t.Errorf("Expected version %s, got %s", version, result.Version)
	}

	if result.DefaultDirectory != tempDir {
		t.Errorf("Expected directory %s, got %s", tempDir, result.DefaultDirectory)
	}

	if result.MaxFileSize != maxFileSize {
		t.Errorf("Expected max file size %d, got %d", maxFileSize, result.MaxFileSize)
	}

	// Check that we have the expected tools
	expectedTools := []string{
		"pdf_read_file",
		"pdf_assets_file",
		"pdf_validate_file",
		"pdf_stats_file",
		"pdf_search_directory",
		"pdf_stats_directory",
		"pdf_server_info",
		"pdf_get_page_info",
		"pdf_get_metadata",
		"pdf_extract_structured",
		"pdf_extract_complete",
		"pdf_extract_tables",
		"pdf_extract_forms",
		"pdf_extract_semantic",
		"pdf_query_content",
		"pdf_analyze_document",
	}

	if len(result.AvailableTools) != len(expectedTools) {
		t.Errorf("Expected %d tools, got %d", len(expectedTools), len(result.AvailableTools))
	}

	// Verify each expected tool is present
	toolNames := make(map[string]bool)
	for _, tool := range result.AvailableTools {
		toolNames[tool.Name] = true

		// Verify each tool has required fields
		if tool.Name == "" {
			t.Error("Tool name should not be empty")
		}
		if tool.Description == "" {
			t.Error("Tool description should not be empty")
		}
		if tool.Usage == "" {
			t.Error("Tool usage should not be empty")
		}
		if tool.Parameters == "" {
			t.Error("Tool parameters should not be empty")
		}
	}

	for _, expectedTool := range expectedTools {
		if !toolNames[expectedTool] {
			t.Errorf("Expected tool %s not found in available tools", expectedTool)
		}
	}

	// Verify usage guidance is provided
	if result.UsageGuidance == "" {
		t.Error("Usage guidance should not be empty")
	}

	// Verify supported formats are provided
	if len(result.SupportedFormats) == 0 {
		t.Error("Should have at least one supported format")
	}

	// Verify directory contents are scanned (should find our test PDF, but won't be valid)
	// Note: Our test PDF is minimal and may not be detected as valid, that's okay for this test
	t.Logf("Found %d files in directory", len(result.DirectoryContents))
	t.Logf("Usage guidance: %s", result.UsageGuidance[:100]+"...")
}

func TestServerInfoWithEmptyDirectory(t *testing.T) {
	// Create a temporary empty directory for testing
	tempDir, err := os.MkdirTemp("", "pdf-test-empty")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Test configuration values
	maxFileSize := int64(100 * 1024 * 1024) // 100MB
	serverName := "test-pdf-server"
	version := "1.0.0-test"

	// Create PDF service
	pdfService, err := NewService(maxFileSize, tempDir)
	if err != nil {
		t.Fatalf("Failed to create PDF service: %v", err)
	}

	// Test server info with empty directory
	req := PDFServerInfoRequest{}
	result, err := pdfService.PDFServerInfo(context.Background(), req, serverName, version, tempDir)
	if err != nil {
		t.Fatalf("Server info failed: %v", err)
	}

	// Verify empty directory handling
	if len(result.DirectoryContents) != 0 {
		t.Errorf("Expected empty directory contents, got %d files", len(result.DirectoryContents))
	}

	// Should still have all other information
	if len(result.AvailableTools) == 0 {
		t.Error("Should still have tools available even with empty directory")
	}

	if result.UsageGuidance == "" {
		t.Error("Should still have usage guidance even with empty directory")
	}
}
