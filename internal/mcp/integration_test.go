package mcp

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/a3tai/mcp-pdf-reader/internal/config"
	"github.com/a3tai/mcp-pdf-reader/internal/pdf"
)

func TestServerIntegration(t *testing.T) {
	// Create temp directory for test files
	tempDir, err := os.MkdirTemp("", "mcp_integration_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test PDF files
	testFiles := []string{"doc1.pdf", "doc2.pdf"}
	for _, filename := range testFiles {
		filePath := filepath.Join(tempDir, filename)
		if err := os.WriteFile(filePath, make([]byte, 1024), 0o644); err != nil {
			t.Fatalf("failed to create test file %s: %v", filename, err)
		}
	}

	// Setup server configuration
	cfg := &config.Config{
		Mode:         "stdio",
		PDFDirectory: tempDir,
		Version:      "1.0.0",
		ServerName:   "integration-test-server",
		MaxFileSize:  1024 * 1024,
	}

	// Create PDF service
	pdfService := pdf.NewService(cfg.MaxFileSize)

	// Create MCP server
	server, err := NewServer(cfg, pdfService)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	// Verify server properties
	if server.config != cfg {
		t.Error("server config not set correctly")
	}
	if server.pdfService != pdfService {
		t.Error("server pdfService not set correctly")
	}
	if server.mcpServer == nil {
		t.Error("mcpServer should be initialized")
	}
}

func TestServerToolsRegistration(t *testing.T) {
	cfg := &config.Config{
		Mode:         "stdio",
		PDFDirectory: "/tmp",
		Version:      "1.0.0",
		ServerName:   "test-server",
		MaxFileSize:  1024 * 1024,
	}

	pdfService := pdf.NewService(cfg.MaxFileSize)
	server, err := NewServer(cfg, pdfService)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	// Test that tools are properly registered by checking the MCP server
	if server.mcpServer == nil {
		t.Fatal("MCP server should be initialized")
	}

	// The mark3labs library doesn't expose registered tools directly,
	// but we can verify the server was created successfully
	// which means tools were registered without errors
}

func TestServerRunStdio(t *testing.T) {
	cfg := &config.Config{
		Mode:         "stdio",
		PDFDirectory: "/tmp",
		Version:      "1.0.0",
		ServerName:   "test-server",
		MaxFileSize:  1024 * 1024,
	}

	pdfService := pdf.NewService(cfg.MaxFileSize)
	server, err := NewServer(cfg, pdfService)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	// Test that the server can start (and quickly stop)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Start server in a goroutine
	done := make(chan error, 1)
	go func() {
		done <- server.runStdioMode(ctx)
	}()

	// Wait for timeout or completion
	select {
	case err := <-done:
		// Server should have stopped due to context timeout
		// This is expected behavior
		if err != nil {
			t.Logf("Server stopped with: %v (expected due to timeout)", err)
		}
	case <-time.After(200 * time.Millisecond):
		t.Error("Server did not stop within expected time")
	}
}

func TestServerConfiguration(t *testing.T) {
	tests := []struct {
		name   string
		config *config.Config
		valid  bool
	}{
		{
			name: "valid stdio config",
			config: &config.Config{
				Mode:         "stdio",
				PDFDirectory: "/tmp",
				Version:      "1.0.0",
				ServerName:   "test-server",
				MaxFileSize:  1024 * 1024,
			},
			valid: true,
		},
		{
			name: "valid server config",
			config: &config.Config{
				Mode:         "server",
				Host:         "127.0.0.1",
				Port:         8080,
				PDFDirectory: "/tmp",
				Version:      "1.0.0",
				ServerName:   "test-server",
				MaxFileSize:  1024 * 1024,
			},
			valid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pdfService := pdf.NewService(tt.config.MaxFileSize)
			server, err := NewServer(tt.config, pdfService)

			if tt.valid && err != nil {
				t.Errorf("expected valid config to succeed, got error: %v", err)
			}
			if !tt.valid && err == nil {
				t.Error("expected invalid config to fail")
			}
			if tt.valid && server == nil {
				t.Error("expected server to be created for valid config")
			}
		})
	}
}

func TestServerErrorHandling(t *testing.T) {
	cfg := &config.Config{
		Mode:         "stdio",
		PDFDirectory: "/tmp",
		Version:      "1.0.0",
		ServerName:   "test-server",
		MaxFileSize:  1024 * 1024,
	}

	// Test with nil PDF service (should not panic)
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Server creation with nil service caused panic: %v", r)
		}
	}()

	_, err := NewServer(cfg, nil)
	if err == nil {
		t.Error("expected error with nil PDF service")
	}
}
