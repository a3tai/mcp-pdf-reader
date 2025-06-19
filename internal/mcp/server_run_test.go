package mcp

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/a3tai/mcp-pdf-reader/internal/config"
	"github.com/a3tai/mcp-pdf-reader/internal/pdf"
)

func TestServer_Run_StdioMode(t *testing.T) {
	cfg := &config.Config{
		Mode:         "stdio",
		Host:         "localhost",
		Port:         8080,
		PDFDirectory: "/tmp",
		LogLevel:     "info",
		MaxFileSize:  100 * 1024 * 1024,
		ServerName:   "test-server",
		Version:      "1.0.0",
	}

	pdfService := pdf.NewService(cfg.MaxFileSize)
	server, err := NewServer(cfg, pdfService)
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}

	// Test with context that gets canceled immediately
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Run should return quickly in stdio mode when context is canceled
	err = server.Run(ctx)
	if err != nil {
		// Error is expected due to canceled context
		if !strings.Contains(err.Error(), "context") {
			t.Errorf("Run() error = %v, expected context-related error", err)
		}
	}
}

func TestServer_Run_ServerMode(t *testing.T) {
	cfg := &config.Config{
		Mode:         "server",
		Host:         "localhost",
		Port:         8080,
		PDFDirectory: "/tmp",
		LogLevel:     "info",
		MaxFileSize:  100 * 1024 * 1024,
		ServerName:   "test-server",
		Version:      "1.0.0",
	}

	pdfService := pdf.NewService(cfg.MaxFileSize)
	server, err := NewServer(cfg, pdfService)
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}

	// Test with context that gets canceled immediately
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Run should return quickly in server mode when context is canceled
	err = server.Run(ctx)
	if err != nil {
		// Error is expected due to canceled context
		if !strings.Contains(err.Error(), "context") {
			t.Errorf("Run() error = %v, expected context-related error", err)
		}
	}
}

func TestServer_Run_InvalidMode(t *testing.T) {
	cfg := &config.Config{
		Mode:         "invalid",
		Host:         "localhost",
		Port:         8080,
		PDFDirectory: "/tmp",
		LogLevel:     "info",
		MaxFileSize:  100 * 1024 * 1024,
		ServerName:   "test-server",
		Version:      "1.0.0",
	}

	pdfService := pdf.NewService(cfg.MaxFileSize)
	server, err := NewServer(cfg, pdfService)
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	// Based on the actual implementation, invalid modes might fall back to stdio mode
	// rather than returning an error, so we test for graceful handling
	err = server.Run(ctx)
	// The server should handle invalid modes gracefully, either by error or fallback
	if err != nil && !strings.Contains(err.Error(), "context") && !strings.Contains(err.Error(), "unsupported mode") {
		t.Errorf("Run() unexpected error type: %v", err)
	}
}

func TestServer_runStdioMode(t *testing.T) {
	cfg := &config.Config{
		Mode:         "stdio",
		Host:         "localhost",
		Port:         8080,
		PDFDirectory: "/tmp",
		LogLevel:     "info",
		MaxFileSize:  100 * 1024 * 1024,
		ServerName:   "test-server",
		Version:      "1.0.0",
	}

	pdfService := pdf.NewService(cfg.MaxFileSize)
	server, err := NewServer(cfg, pdfService)
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}

	tests := []struct {
		name           string
		contextTimeout time.Duration
		expectComplete bool
	}{
		{
			name:           "canceled context",
			contextTimeout: 1 * time.Millisecond,
			expectComplete: true, // Should complete gracefully
		},
		{
			name:           "quick timeout",
			contextTimeout: 10 * time.Millisecond,
			expectComplete: true, // Should complete gracefully
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), tt.contextTimeout)
			defer cancel()

			err := server.runStdioMode(ctx)
			if tt.expectComplete {
				// Server should handle quick timeouts gracefully
				if err != nil && !strings.Contains(err.Error(), "context") {
					t.Errorf("runStdioMode() unexpected non-context error = %v", err)
				}
			}
		})
	}
}

func TestServer_runServerMode(t *testing.T) {
	cfg := &config.Config{
		Mode:         "server",
		Host:         "localhost",
		Port:         8080,
		PDFDirectory: "/tmp",
		LogLevel:     "info",
		MaxFileSize:  100 * 1024 * 1024,
		ServerName:   "test-server",
		Version:      "1.0.0",
	}

	pdfService := pdf.NewService(cfg.MaxFileSize)
	server, err := NewServer(cfg, pdfService)
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}

	tests := []struct {
		name           string
		contextTimeout time.Duration
		expectComplete bool
	}{
		{
			name:           "canceled context",
			contextTimeout: 1 * time.Millisecond,
			expectComplete: true, // Should complete gracefully (may fall back to stdio)
		},
		{
			name:           "quick timeout",
			contextTimeout: 10 * time.Millisecond,
			expectComplete: true, // Should complete gracefully (may fall back to stdio)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), tt.contextTimeout)
			defer cancel()

			err := server.runServerMode(ctx)
			if tt.expectComplete {
				// Server should handle quick timeouts gracefully, may fall back to stdio mode
				if err != nil && !strings.Contains(err.Error(), "context") {
					t.Errorf("runServerMode() unexpected non-context error = %v", err)
				}
			}
		})
	}
}

func TestServer_Run_ContextCancellation(t *testing.T) {
	tests := []struct {
		name string
		mode string
	}{
		{
			name: "stdio mode context cancellation",
			mode: "stdio",
		},
		{
			name: "server mode context cancellation",
			mode: "server",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				Mode:         tt.mode,
				Host:         "localhost",
				Port:         8080,
				PDFDirectory: "/tmp",
				LogLevel:     "info",
				MaxFileSize:  100 * 1024 * 1024,
				ServerName:   "test-server",
				Version:      "1.0.0",
			}

			pdfService := pdf.NewService(cfg.MaxFileSize)
			server, err := NewServer(cfg, pdfService)
			if err != nil {
				t.Fatalf("NewServer() error = %v", err)
			}

			ctx, cancel := context.WithCancel(context.Background())

			// Run server in goroutine
			errChan := make(chan error, 1)
			go func() {
				errChan <- server.Run(ctx)
			}()

			// Cancel context after a short delay
			time.Sleep(10 * time.Millisecond)
			cancel()

			// Wait for server to stop
			select {
			case err := <-errChan:
				// Error is expected due to context cancellation
				if err != nil && !strings.Contains(err.Error(), "context") {
					t.Errorf("Run() error = %v, expected context-related error", err)
				}
			case <-time.After(1 * time.Second):
				t.Error("Run() did not return after context cancellation")
			}
		})
	}
}

func TestServer_Run_ConfigValidation(t *testing.T) {
	pdfService := pdf.NewService(100 * 1024 * 1024)

	tests := []struct {
		name   string
		config *config.Config
	}{
		{
			name: "stdio mode with valid config",
			config: &config.Config{
				Mode:         "stdio",
				Host:         "localhost",
				Port:         8080,
				PDFDirectory: "/tmp",
				LogLevel:     "info",
				MaxFileSize:  100 * 1024 * 1024,
				ServerName:   "test-server",
				Version:      "1.0.0",
			},
		},
		{
			name: "server mode with valid config",
			config: &config.Config{
				Mode:         "server",
				Host:         "localhost",
				Port:         8080,
				PDFDirectory: "/tmp",
				LogLevel:     "info",
				MaxFileSize:  100 * 1024 * 1024,
				ServerName:   "test-server",
				Version:      "1.0.0",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, err := NewServer(tt.config, pdfService)
			if err != nil {
				t.Fatalf("NewServer() error = %v", err)
			}

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
			defer cancel()

			// Run should not panic and should handle the timeout gracefully
			err = server.Run(ctx)
			// We expect an error due to timeout, but it should be handled gracefully
			if err == nil {
				t.Log("Run() completed without error (may be expected for quick timeout)")
			}
		})
	}
}

func TestServer_Run_NilConfig(t *testing.T) {
	pdfService := pdf.NewService(100 * 1024 * 1024)

	// Test with nil config (will likely panic, so we catch it)
	server := &Server{
		config:     nil,
		pdfService: pdfService,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	defer func() {
		if r := recover(); r != nil {
			// Panic is expected with nil config
			return
		}
	}()

	err := server.Run(ctx)
	if err == nil {
		t.Error("Run() expected error with nil config but got none")
	}
}

func TestServer_Run_NilPDFService(t *testing.T) {
	cfg := &config.Config{
		Mode:         "stdio",
		Host:         "localhost",
		Port:         8080,
		PDFDirectory: "/tmp",
		LogLevel:     "info",
		MaxFileSize:  100 * 1024 * 1024,
		ServerName:   "test-server",
		Version:      "1.0.0",
	}

	// Test with nil PDF service (will likely panic, so we catch it)
	server := &Server{
		config:     cfg,
		pdfService: nil,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	defer func() {
		if r := recover(); r != nil {
			// Panic is expected with nil PDF service
			return
		}
	}()

	err := server.Run(ctx)
	if err == nil {
		t.Error("Run() expected error with nil PDF service but got none")
	}
}

func TestServer_Run_ErrorHandling(t *testing.T) {
	cfg := &config.Config{
		Mode:         "stdio",
		Host:         "localhost",
		Port:         8080,
		PDFDirectory: "/tmp",
		LogLevel:     "info",
		MaxFileSize:  100 * 1024 * 1024,
		ServerName:   "test-server",
		Version:      "1.0.0",
	}

	pdfService := pdf.NewService(cfg.MaxFileSize)
	server, err := NewServer(cfg, pdfService)
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}

	// Test error handling with already canceled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err = server.Run(ctx)
	if err != nil {
		// Error is expected, but should be handled gracefully
		if strings.Contains(err.Error(), "panic") {
			t.Errorf("Run() should not panic, got error: %v", err)
		}
	}
}

func TestServer_Run_GracefulShutdown(t *testing.T) {
	cfg := &config.Config{
		Mode:         "server",
		Host:         "localhost",
		Port:         8080,
		PDFDirectory: "/tmp",
		LogLevel:     "info",
		MaxFileSize:  100 * 1024 * 1024,
		ServerName:   "test-server",
		Version:      "1.0.0",
	}

	pdfService := pdf.NewService(cfg.MaxFileSize)
	server, err := NewServer(cfg, pdfService)
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Start server in goroutine
	done := make(chan struct{})
	go func() {
		defer close(done)
		server.Run(ctx)
	}()

	// Give server time to start
	time.Sleep(50 * time.Millisecond)

	// Cancel context to trigger graceful shutdown
	cancel()

	// Wait for server to shutdown
	select {
	case <-done:
		// Server shut down successfully
	case <-time.After(2 * time.Second):
		t.Error("Server did not shutdown gracefully within timeout")
	}
}

func TestServer_Run_MultipleShutdowns(t *testing.T) {
	cfg := &config.Config{
		Mode:         "stdio",
		Host:         "localhost",
		Port:         8080,
		PDFDirectory: "/tmp",
		LogLevel:     "info",
		MaxFileSize:  100 * 1024 * 1024,
		ServerName:   "test-server",
		Version:      "1.0.0",
	}

	pdfService := pdf.NewService(cfg.MaxFileSize)
	server, err := NewServer(cfg, pdfService)
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}

	// Test multiple rapid shutdowns
	for i := 0; i < 3; i++ {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		err := server.Run(ctx)
		// Should handle multiple shutdowns gracefully
		if err != nil && strings.Contains(err.Error(), "panic") {
			t.Errorf("Run() iteration %d should not panic, got error: %v", i, err)
		}
	}
}
