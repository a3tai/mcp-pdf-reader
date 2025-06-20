package config

import (
	"os"
	"testing"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// Helper function to reset pflag.CommandLine for testing
func resetFlags() {
	pflag.CommandLine = pflag.NewFlagSet(os.Args[0], pflag.ExitOnError)
	viper.Reset()
}

// Helper function to set os.Args for testing
func setArgs(args []string) {
	os.Args = args
}

// Helper function to clear environment variables
func clearEnvVars() {
	os.Unsetenv("MCP_PDF_MODE")
	os.Unsetenv("MCP_PDF_HOST")
	os.Unsetenv("MCP_PDF_PORT")
	os.Unsetenv("MCP_PDF_DIR")
	os.Unsetenv("MCP_PDF_LOG_LEVEL")
	os.Unsetenv("MCP_PDF_MAX_FILE_SIZE")
}

func TestLoadFromFlags_DefaultConfig(t *testing.T) {
	// Save original args and environment
	originalArgs := os.Args
	defer func() {
		os.Args = originalArgs
		resetFlags()
		clearEnvVars()
	}()

	// Set minimal args (just program name)
	setArgs([]string{"mcp-pdf-reader"})
	resetFlags()
	clearEnvVars()

	cfg, err := LoadFromFlags()
	if err != nil {
		t.Fatalf("LoadFromFlags() unexpected error: %v", err)
	}

	// Verify default values
	if cfg.Mode != "stdio" {
		t.Errorf("LoadFromFlags() Mode = %v, want %v", cfg.Mode, "stdio")
	}
	if cfg.Host != "127.0.0.1" {
		t.Errorf("LoadFromFlags() Host = %v, want %v", cfg.Host, "127.0.0.1")
	}
	if cfg.Port != 8080 {
		t.Errorf("LoadFromFlags() Port = %v, want %v", cfg.Port, 8080)
	}
	if cfg.LogLevel != "info" {
		t.Errorf("LoadFromFlags() LogLevel = %v, want %v", cfg.LogLevel, "info")
	}
	if cfg.MaxFileSize != 100*1024*1024 {
		t.Errorf("LoadFromFlags() MaxFileSize = %v, want %v", cfg.MaxFileSize, 100*1024*1024)
	}
	// PDFDirectory should be current working directory
	if cfg.PDFDirectory == "" {
		t.Error("LoadFromFlags() PDFDirectory should not be empty")
	}
}

func TestLoadFromFlags_ValidFlags(t *testing.T) {
	tests := []struct {
		name            string
		argsTemplate    []string
		wantMode        string
		wantHost        string
		wantPort        int
		wantLogLevel    string
		wantMaxFileSize int64
	}{
		{
			name:            "stdio mode with custom directory",
			argsTemplate:    []string{"mcp-pdf-reader", "--dir=%s"},
			wantMode:        "stdio",
			wantHost:        "127.0.0.1",
			wantPort:        8080,
			wantLogLevel:    "info",
			wantMaxFileSize: 100 * 1024 * 1024,
		},
		{
			name:            "server mode",
			argsTemplate:    []string{"mcp-pdf-reader", "--mode=server", "--dir=%s"},
			wantMode:        "server",
			wantHost:        "127.0.0.1",
			wantPort:        8080,
			wantLogLevel:    "info",
			wantMaxFileSize: 100 * 1024 * 1024,
		},
		{
			name:            "server mode with custom host and port",
			argsTemplate:    []string{"mcp-pdf-reader", "--mode=server", "--host=0.0.0.0", "--port=9090", "--dir=%s"},
			wantMode:        "server",
			wantHost:        "0.0.0.0",
			wantPort:        9090,
			wantLogLevel:    "info",
			wantMaxFileSize: 100 * 1024 * 1024,
		},
		{
			name:            "debug logging",
			argsTemplate:    []string{"mcp-pdf-reader", "--log-level=debug", "--dir=%s"},
			wantMode:        "stdio",
			wantHost:        "127.0.0.1",
			wantPort:        8080,
			wantLogLevel:    "debug",
			wantMaxFileSize: 100 * 1024 * 1024,
		},
		{
			name:            "custom max file size",
			argsTemplate:    []string{"mcp-pdf-reader", "--max-file-size=50000000", "--dir=%s"},
			wantMode:        "stdio",
			wantHost:        "127.0.0.1",
			wantPort:        8080,
			wantLogLevel:    "info",
			wantMaxFileSize: 50000000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original args and environment
			originalArgs := os.Args
			defer func() {
				os.Args = originalArgs
				resetFlags()
				clearEnvVars()
			}()

			// Create temp directory for this test
			tempDir := t.TempDir()

			// Build args with temp directory
			args := make([]string, len(tt.argsTemplate))
			for i, arg := range tt.argsTemplate {
				if arg == "--dir=%s" {
					args[i] = "--dir=" + tempDir
				} else {
					args[i] = arg
				}
			}

			setArgs(args)
			resetFlags()
			clearEnvVars()

			cfg, err := LoadFromFlags()
			if err != nil {
				t.Fatalf("LoadFromFlags() unexpected error: %v", err)
			}

			if cfg.Mode != tt.wantMode {
				t.Errorf("LoadFromFlags() Mode = %v, want %v", cfg.Mode, tt.wantMode)
			}
			if cfg.Host != tt.wantHost {
				t.Errorf("LoadFromFlags() Host = %v, want %v", cfg.Host, tt.wantHost)
			}
			if cfg.Port != tt.wantPort {
				t.Errorf("LoadFromFlags() Port = %v, want %v", cfg.Port, tt.wantPort)
			}
			if cfg.LogLevel != tt.wantLogLevel {
				t.Errorf("LoadFromFlags() LogLevel = %v, want %v", cfg.LogLevel, tt.wantLogLevel)
			}
			if cfg.MaxFileSize != tt.wantMaxFileSize {
				t.Errorf("LoadFromFlags() MaxFileSize = %v, want %v", cfg.MaxFileSize, tt.wantMaxFileSize)
			}
			// PDFDirectory should be expanded to absolute path
			if cfg.PDFDirectory == "" {
				t.Error("LoadFromFlags() PDFDirectory should not be empty")
			}
		})
	}
}

func TestLoadFromFlags_EnvironmentVariables(t *testing.T) {
	// Save original args and environment
	originalArgs := os.Args
	defer func() {
		os.Args = originalArgs
		resetFlags()
		clearEnvVars()
	}()

	// Create temp directory for testing
	tempDir := t.TempDir()

	// Set environment variables
	os.Setenv("MCP_PDF_MODE", "server")
	os.Setenv("MCP_PDF_HOST", "192.168.1.1")
	os.Setenv("MCP_PDF_PORT", "3000")
	os.Setenv("MCP_PDF_DIR", tempDir)
	os.Setenv("MCP_PDF_LOG_LEVEL", "warn")
	os.Setenv("MCP_PDF_MAX_FILE_SIZE", "200000000")

	setArgs([]string{"mcp-pdf-reader"})
	resetFlags()

	cfg, err := LoadFromFlags()
	if err != nil {
		t.Fatalf("LoadFromFlags() unexpected error: %v", err)
	}

	if cfg.Mode != "server" {
		t.Errorf("LoadFromFlags() Mode = %v, want %v", cfg.Mode, "server")
	}
	if cfg.Host != "192.168.1.1" {
		t.Errorf("LoadFromFlags() Host = %v, want %v", cfg.Host, "192.168.1.1")
	}
	if cfg.Port != 3000 {
		t.Errorf("LoadFromFlags() Port = %v, want %v", cfg.Port, 3000)
	}
	if cfg.LogLevel != "warn" {
		t.Errorf("LoadFromFlags() LogLevel = %v, want %v", cfg.LogLevel, "warn")
	}
	if cfg.MaxFileSize != 200000000 {
		t.Errorf("LoadFromFlags() MaxFileSize = %v, want %v", cfg.MaxFileSize, 200000000)
	}
}

func TestLoadFromFlags_FlagOverridesEnvironment(t *testing.T) {
	// Save original args and environment
	originalArgs := os.Args
	defer func() {
		os.Args = originalArgs
		resetFlags()
		clearEnvVars()
	}()

	// Set environment variables
	os.Setenv("MCP_PDF_MODE", "server")
	os.Setenv("MCP_PDF_HOST", "192.168.1.1")
	os.Setenv("MCP_PDF_PORT", "3000")

	// Set args that should override environment
	setArgs([]string{"mcp-pdf-reader", "--mode=stdio", "--host=localhost", "--port=8888"})
	resetFlags()

	cfg, err := LoadFromFlags()
	if err != nil {
		t.Fatalf("LoadFromFlags() unexpected error: %v", err)
	}

	// Flags should override environment variables
	if cfg.Mode != "stdio" {
		t.Errorf("LoadFromFlags() Mode = %v, want %v (should override env)", cfg.Mode, "stdio")
	}
	if cfg.Host != "localhost" {
		t.Errorf("LoadFromFlags() Host = %v, want %v (should override env)", cfg.Host, "localhost")
	}
	if cfg.Port != 8888 {
		t.Errorf("LoadFromFlags() Port = %v, want %v (should override env)", cfg.Port, 8888)
	}
}

func TestLoadFromFlags_InvalidMode(t *testing.T) {
	// Save original args
	originalArgs := os.Args
	defer func() {
		os.Args = originalArgs
		resetFlags()
		clearEnvVars()
	}()

	tempDir := t.TempDir()
	setArgs([]string{"mcp-pdf-reader", "--mode=invalid", "--dir=" + tempDir})
	resetFlags()
	clearEnvVars()

	_, err := LoadFromFlags()
	if err == nil {
		t.Error("LoadFromFlags() expected error for invalid mode")
	}
	if err != nil && !containsString(err.Error(), "mode must be either 'stdio' or 'server'") {
		t.Errorf("LoadFromFlags() error = %v, want error about invalid mode", err)
	}
}

func TestLoadFromFlags_InvalidPort(t *testing.T) {
	// Save original args
	originalArgs := os.Args
	defer func() {
		os.Args = originalArgs
		resetFlags()
		clearEnvVars()
	}()

	tempDir := t.TempDir()
	setArgs([]string{"mcp-pdf-reader", "--mode=server", "--port=99999", "--dir=" + tempDir})
	resetFlags()
	clearEnvVars()

	_, err := LoadFromFlags()
	if err == nil {
		t.Error("LoadFromFlags() expected error for invalid port")
	}
	if err != nil && !containsString(err.Error(), "port must be between 1 and 65535") {
		t.Errorf("LoadFromFlags() error = %v, want error about invalid port", err)
	}
}

func TestLoadFromFlags_InvalidLogLevel(t *testing.T) {
	// Save original args
	originalArgs := os.Args
	defer func() {
		os.Args = originalArgs
		resetFlags()
		clearEnvVars()
	}()

	tempDir := t.TempDir()
	setArgs([]string{"mcp-pdf-reader", "--log-level=invalid", "--dir=" + tempDir})
	resetFlags()
	clearEnvVars()

	_, err := LoadFromFlags()
	if err == nil {
		t.Error("LoadFromFlags() expected error for invalid log level")
	}
	if err != nil && !containsString(err.Error(), "invalid log level") {
		t.Errorf("LoadFromFlags() error = %v, want error about invalid log level", err)
	}
}

func TestLoadFromFlags_VersionFlag(t *testing.T) {
	// Save original args
	originalArgs := os.Args
	defer func() {
		os.Args = originalArgs
		resetFlags()
		clearEnvVars()
	}()

	setArgs([]string{"mcp-pdf-reader", "--version"})
	resetFlags()
	clearEnvVars()

	_, err := LoadFromFlags()
	if err == nil {
		t.Error("LoadFromFlags() expected version error")
	}
	if err != nil && err.Error() != "version requested" {
		t.Errorf("LoadFromFlags() error = %v, want 'version requested'", err)
	}
}

// Helper function to check if a string contains a substring
func containsString(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
			(len(s) > len(substr) &&
				(s[:len(substr)] == substr ||
					s[len(s)-len(substr):] == substr ||
					findSubstring(s, substr))))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
