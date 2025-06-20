package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	// Test default values
	if cfg.Mode != "stdio" {
		t.Errorf("Expected default mode to be 'stdio', got '%s'", cfg.Mode)
	}

	if cfg.Host != "127.0.0.1" {
		t.Errorf("Expected default host to be '127.0.0.1', got '%s'", cfg.Host)
	}

	if cfg.Port != 8080 {
		t.Errorf("Expected default port to be 8080, got %d", cfg.Port)
	}

	if cfg.Version != "1.0.0" {
		t.Errorf("Expected default version to be '1.0.0', got '%s'", cfg.Version)
	}

	if cfg.ServerName != "mcp-pdf-reader" {
		t.Errorf("Expected default server name to be 'mcp-pdf-reader', got '%s'", cfg.ServerName)
	}

	if cfg.LogLevel != "info" {
		t.Errorf("Expected default log level to be 'info', got '%s'", cfg.LogLevel)
	}

	if cfg.MaxFileSize != 100*1024*1024 {
		t.Errorf("Expected default max file size to be 100MB, got %d", cfg.MaxFileSize)
	}

	// Test that PDF directory is set to current working directory by default
	currentDir, _ := os.Getwd()
	if cfg.PDFDirectory != currentDir {
		t.Errorf("Expected default PDF directory to be '%s', got '%s'", currentDir, cfg.PDFDirectory)
	}
}

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name:    "valid config - stdio mode",
			config:  DefaultConfig(),
			wantErr: false,
		},
		{
			name: "valid config - server mode",
			config: &Config{
				Mode:         "server",
				Host:         "127.0.0.1",
				Port:         8080,
				PDFDirectory: "/tmp/test",
				LogLevel:     "info",
				MaxFileSize:  1024,
			},
			wantErr: false,
		},
		{
			name: "invalid mode",
			config: &Config{
				Mode:         "invalid",
				Host:         "127.0.0.1",
				Port:         8080,
				PDFDirectory: "/tmp/test",
				LogLevel:     "info",
				MaxFileSize:  1024,
			},
			wantErr: true,
		},
		{
			name: "invalid port - too low (server mode)",
			config: &Config{
				Mode:         "server",
				Host:         "127.0.0.1",
				Port:         0,
				PDFDirectory: "/tmp/test",
				LogLevel:     "info",
				MaxFileSize:  1024,
			},
			wantErr: true,
		},
		{
			name: "invalid port - too high (server mode)",
			config: &Config{
				Mode:         "server",
				Host:         "127.0.0.1",
				Port:         70000,
				PDFDirectory: "/tmp/test",
				LogLevel:     "info",
				MaxFileSize:  1024,
			},
			wantErr: true,
		},
		{
			name: "invalid port ignored in stdio mode",
			config: &Config{
				Mode:         "stdio",
				Host:         "127.0.0.1",
				Port:         0,
				PDFDirectory: "/tmp/test",
				LogLevel:     "info",
				MaxFileSize:  1024,
			},
			wantErr: false,
		},
		{
			name: "empty PDF directory",
			config: &Config{
				Mode:         "stdio",
				Host:         "127.0.0.1",
				Port:         8080,
				PDFDirectory: "",
				LogLevel:     "info",
				MaxFileSize:  1024,
			},
			wantErr: true,
		},
		{
			name: "invalid log level",
			config: &Config{
				Mode:         "stdio",
				Host:         "127.0.0.1",
				Port:         8080,
				PDFDirectory: "/tmp/test",
				LogLevel:     "invalid",
				MaxFileSize:  1024,
			},
			wantErr: true,
		},
		{
			name: "invalid max file size",
			config: &Config{
				Mode:         "stdio",
				Host:         "127.0.0.1",
				Port:         8080,
				PDFDirectory: "/tmp/test",
				LogLevel:     "info",
				MaxFileSize:  0,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory for tests that need it
			if tt.config.PDFDirectory != "" && tt.config.PDFDirectory != "/tmp/test" {
				// Use the actual directory for default config test
			} else if tt.config.PDFDirectory == "/tmp/test" {
				// Create a temporary directory for testing
				tempDir, err := os.MkdirTemp("", "pdf-mcp-test-*")
				if err != nil {
					t.Fatalf("Failed to create temp dir: %v", err)
				}
				defer os.RemoveAll(tempDir)
				tt.config.PDFDirectory = tempDir
			}

			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfigAddress(t *testing.T) {
	cfg := &Config{
		Host: "192.168.1.1",
		Port: 9090,
	}

	expected := "192.168.1.1:9090"
	if got := cfg.Address(); got != expected {
		t.Errorf("Config.Address() = %v, want %v", got, expected)
	}
}

func TestConfigIsDebug(t *testing.T) {
	tests := []struct {
		name     string
		logLevel string
		want     bool
	}{
		{
			name:     "debug level",
			logLevel: "debug",
			want:     true,
		},
		{
			name:     "info level",
			logLevel: "info",
			want:     false,
		},
		{
			name:     "warn level",
			logLevel: "warn",
			want:     false,
		},
		{
			name:     "error level",
			logLevel: "error",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{LogLevel: tt.logLevel}
			if got := cfg.IsDebug(); got != tt.want {
				t.Errorf("Config.IsDebug() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfigString(t *testing.T) {
	cfg := &Config{
		Mode:         "server",
		Host:         "localhost",
		Port:         8080,
		PDFDirectory: "/home/user/pdfs",
		LogLevel:     "debug",
		MaxFileSize:  1024,
	}

	result := cfg.String()

	// Check that the string contains expected components
	expectedSubstrings := []string{
		"Mode: server",
		"Host: localhost",
		"Port: 8080",
		"PDFDirectory: /home/user/pdfs",
		"LogLevel: debug",
		"MaxFileSize: 1024",
	}

	for _, substr := range expectedSubstrings {
		if !contains(result, substr) {
			t.Errorf("Config.String() result doesn't contain expected substring: %s\nGot: %s", substr, result)
		}
	}
}

func TestConfigValidateDirectoryCreation(t *testing.T) {
	// Test that we no longer create directories automatically
	// This allows for placeholder paths like ${workspaceRoot}

	// Create a temporary parent directory
	tempParent, err := os.MkdirTemp("", "pdf-mcp-parent-*")
	if err != nil {
		t.Fatalf("Failed to create temp parent dir: %v", err)
	}
	defer os.RemoveAll(tempParent)

	// Use a non-existent subdirectory
	nonExistentDir := filepath.Join(tempParent, "non-existent", "pdfs")

	cfg := &Config{
		Mode:         "stdio",
		Host:         "127.0.0.1",
		Port:         8080,
		PDFDirectory: nonExistentDir,
		LogLevel:     "info",
		MaxFileSize:  1024,
	}

	// Validate should NOT create the directory anymore
	err = cfg.Validate()
	if err != nil {
		t.Errorf("Config.Validate() should not fail for non-existent directory, got error: %v", err)
	}

	// Check that directory was NOT created
	if _, err := os.Stat(nonExistentDir); !os.IsNotExist(err) {
		t.Errorf("Directory should NOT have been created: %s", nonExistentDir)
	}
}

func TestConfigValidateLogLevels(t *testing.T) {
	validLevels := []string{"debug", "info", "warn", "error"}
	invalidLevels := []string{"DEBUG", "INFO", "trace", "fatal", ""}

	tempDir, err := os.MkdirTemp("", "pdf-mcp-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Test valid log levels
	for _, level := range validLevels {
		t.Run("valid_"+level, func(t *testing.T) {
			cfg := &Config{
				Mode:         "stdio",
				Host:         "127.0.0.1",
				Port:         8080,
				PDFDirectory: tempDir,
				LogLevel:     level,
				MaxFileSize:  1024,
			}

			if err := cfg.Validate(); err != nil {
				t.Errorf("Config.Validate() should accept log level '%s', got error: %v", level, err)
			}
		})
	}

	// Test invalid log levels
	for _, level := range invalidLevels {
		t.Run("invalid_"+level, func(t *testing.T) {
			cfg := &Config{
				Mode:         "stdio",
				Host:         "127.0.0.1",
				Port:         8080,
				PDFDirectory: tempDir,
				LogLevel:     level,
				MaxFileSize:  1024,
			}

			if err := cfg.Validate(); err == nil {
				t.Errorf("Config.Validate() should reject log level '%s'", level)
			}
		})
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
			s[:len(substr)] == substr ||
			s[len(s)-len(substr):] == substr ||
			containsMiddle(s, substr))
}

func containsMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestConfigIsServerMode(t *testing.T) {
	tests := []struct {
		name string
		mode string
		want bool
	}{
		{
			name: "server mode",
			mode: "server",
			want: true,
		},
		{
			name: "stdio mode",
			mode: "stdio",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{Mode: tt.mode}
			if got := cfg.IsServerMode(); got != tt.want {
				t.Errorf("Config.IsServerMode() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfigIsStdioMode(t *testing.T) {
	tests := []struct {
		name string
		mode string
		want bool
	}{
		{
			name: "stdio mode",
			mode: "stdio",
			want: true,
		},
		{
			name: "server mode",
			mode: "server",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{Mode: tt.mode}
			if got := cfg.IsStdioMode(); got != tt.want {
				t.Errorf("Config.IsStdioMode() = %v, want %v", got, tt.want)
			}
		})
	}
}
