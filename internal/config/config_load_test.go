package config

import (
	"flag"
	"os"
	"strings"
	"testing"
)

// Helper function to reset flag.CommandLine for testing
func resetFlags() {
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
}

// Helper function to set os.Args for testing
func setArgs(args []string) {
	os.Args = args
}

func TestLoadFromFlags_DefaultConfig(t *testing.T) {
	// Save original args
	originalArgs := os.Args
	defer func() {
		os.Args = originalArgs
		resetFlags()
	}()

	// Set minimal args (just program name)
	setArgs([]string{"mcp-pdf-reader"})
	resetFlags()

	cfg, err := LoadFromFlags()
	if err != nil {
		t.Fatalf("LoadFromFlags() unexpected error: %v", err)
	}

	// Verify default values
	if cfg.Mode != "stdio" {
		t.Errorf("LoadFromFlags() Mode = %v, want %v", cfg.Mode, "stdio")
	}
	if cfg.Host != "localhost" {
		t.Errorf("LoadFromFlags() Host = %v, want %v", cfg.Host, "localhost")
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
}

func TestLoadFromFlags_ValidFlags(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want Config
	}{
		{
			name: "stdio mode with custom PDF directory",
			args: []string{"mcp-pdf-reader", "-mode=stdio", "-pdfdir=/custom/path"},
			want: Config{
				Mode:         "stdio",
				Host:         "localhost",
				Port:         8080,
				PDFDirectory: "/custom/path",
				LogLevel:     "info",
				MaxFileSize:  100 * 1024 * 1024,
				ServerName:   "mcp-pdf-reader",
				Version:      "1.0.0",
			},
		},
		{
			name: "server mode with custom host and port",
			args: []string{"mcp-pdf-reader", "-mode=server", "-host=0.0.0.0", "-port=9090"},
			want: Config{
				Mode:         "server",
				Host:         "0.0.0.0",
				Port:         9090,
				PDFDirectory: "",
				LogLevel:     "info",
				MaxFileSize:  100 * 1024 * 1024,
				ServerName:   "mcp-pdf-reader",
				Version:      "1.0.0",
			},
		},
		{
			name: "debug log level",
			args: []string{"mcp-pdf-reader", "-loglevel=debug"},
			want: Config{
				Mode:         "stdio",
				Host:         "localhost",
				Port:         8080,
				PDFDirectory: "",
				LogLevel:     "debug",
				MaxFileSize:  100 * 1024 * 1024,
				ServerName:   "mcp-pdf-reader",
				Version:      "1.0.0",
			},
		},
		{
			name: "custom max file size",
			args: []string{"mcp-pdf-reader", "-maxfilesize=50000000"},
			want: Config{
				Mode:         "stdio",
				Host:         "localhost",
				Port:         8080,
				PDFDirectory: "",
				LogLevel:     "info",
				MaxFileSize:  50000000,
				ServerName:   "mcp-pdf-reader",
				Version:      "1.0.0",
			},
		},
		{
			name: "all flags combined",
			args: []string{
				"mcp-pdf-reader",
				"-mode=server",
				"-host=192.168.1.1",
				"-port=3000",
				"-pdfdir=/home/user/documents",
				"-loglevel=error",
				"-maxfilesize=200000000",
			},
			want: Config{
				Mode:         "server",
				Host:         "192.168.1.1",
				Port:         3000,
				PDFDirectory: "/home/user/documents",
				LogLevel:     "error",
				MaxFileSize:  200000000,
				ServerName:   "mcp-pdf-reader",
				Version:      "1.0.0",
			},
		},
	}

	originalArgs := os.Args
	defer func() {
		os.Args = originalArgs
	}()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setArgs(tt.args)
			resetFlags()

			cfg, err := LoadFromFlags()
			if err != nil {
				t.Fatalf("LoadFromFlags() unexpected error: %v", err)
			}

			if cfg.Mode != tt.want.Mode {
				t.Errorf("LoadFromFlags() Mode = %v, want %v", cfg.Mode, tt.want.Mode)
			}
			if cfg.Host != tt.want.Host {
				t.Errorf("LoadFromFlags() Host = %v, want %v", cfg.Host, tt.want.Host)
			}
			if cfg.Port != tt.want.Port {
				t.Errorf("LoadFromFlags() Port = %v, want %v", cfg.Port, tt.want.Port)
			}
			if cfg.PDFDirectory != tt.want.PDFDirectory {
				t.Errorf("LoadFromFlags() PDFDirectory = %v, want %v", cfg.PDFDirectory, tt.want.PDFDirectory)
			}
			if cfg.LogLevel != tt.want.LogLevel {
				t.Errorf("LoadFromFlags() LogLevel = %v, want %v", cfg.LogLevel, tt.want.LogLevel)
			}
			if cfg.MaxFileSize != tt.want.MaxFileSize {
				t.Errorf("LoadFromFlags() MaxFileSize = %v, want %v", cfg.MaxFileSize, tt.want.MaxFileSize)
			}
		})
	}
}

func TestLoadFromFlags_InvalidFlags(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		wantError string
	}{
		{
			name:      "invalid mode",
			args:      []string{"mcp-pdf-reader", "-mode=invalid"},
			wantError: "invalid mode",
		},
		{
			name:      "invalid port - too low",
			args:      []string{"mcp-pdf-reader", "-mode=server", "-port=0"},
			wantError: "port must be between 1 and 65535",
		},
		{
			name:      "invalid port - too high",
			args:      []string{"mcp-pdf-reader", "-mode=server", "-port=70000"},
			wantError: "port must be between 1 and 65535",
		},
		{
			name:      "invalid log level",
			args:      []string{"mcp-pdf-reader", "-loglevel=invalid"},
			wantError: "invalid log level",
		},
		{
			name:      "invalid max file size - negative",
			args:      []string{"mcp-pdf-reader", "-maxfilesize=-1"},
			wantError: "maxFileSize must be positive",
		},
		{
			name:      "invalid max file size - zero",
			args:      []string{"mcp-pdf-reader", "-maxfilesize=0"},
			wantError: "maxFileSize must be positive",
		},
		{
			name:      "invalid max file size - too large",
			args:      []string{"mcp-pdf-reader", "-maxfilesize=1073741825"}, // 1GB + 1 byte
			wantError: "maxFileSize cannot exceed 1GB",
		},
	}

	originalArgs := os.Args
	defer func() {
		os.Args = originalArgs
	}()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setArgs(tt.args)
			resetFlags()

			cfg, err := LoadFromFlags()
			if err == nil {
				t.Errorf("LoadFromFlags() expected error but got none")
			}
			if cfg != nil {
				t.Errorf("LoadFromFlags() expected nil config on error, got %v", cfg)
			}
			if !strings.Contains(err.Error(), tt.wantError) {
				t.Errorf("LoadFromFlags() error = %v, want error containing %v", err, tt.wantError)
			}
		})
	}
}

func TestLoadFromFlags_EdgeCases(t *testing.T) {
	originalArgs := os.Args
	defer func() {
		os.Args = originalArgs
	}()

	t.Run("empty PDF directory in stdio mode", func(t *testing.T) {
		setArgs([]string{"mcp-pdf-reader", "-mode=stdio", "-pdfdir="})
		resetFlags()

		cfg, err := LoadFromFlags()
		if err == nil {
			t.Error("LoadFromFlags() expected error for empty PDF directory in stdio mode")
		}
		if cfg != nil {
			t.Error("LoadFromFlags() expected nil config on error")
		}
		if !strings.Contains(err.Error(), "PDFDirectory cannot be empty") {
			t.Errorf("LoadFromFlags() error = %v, want error containing 'PDFDirectory cannot be empty'", err)
		}
	})

	t.Run("port validation ignored in stdio mode", func(t *testing.T) {
		setArgs([]string{"mcp-pdf-reader", "-mode=stdio", "-port=70000"})
		resetFlags()

		cfg, err := LoadFromFlags()
		if err != nil {
			t.Errorf("LoadFromFlags() unexpected error in stdio mode: %v", err)
		}
		if cfg == nil {
			t.Error("LoadFromFlags() expected config but got nil")
		}
		if cfg != nil && cfg.Port != 70000 {
			t.Errorf("LoadFromFlags() Port = %v, want %v", cfg.Port, 70000)
		}
	})

	t.Run("host validation in server mode", func(t *testing.T) {
		setArgs([]string{"mcp-pdf-reader", "-mode=server", "-host=", "-port=8080"})
		resetFlags()

		cfg, err := LoadFromFlags()
		if err != nil {
			t.Errorf("LoadFromFlags() unexpected error: %v", err)
		}
		if cfg == nil {
			t.Error("LoadFromFlags() expected config but got nil")
		}
		// Empty host should be allowed (defaults to localhost behavior)
	})
}

func TestLoadFromFlags_BoundaryValues(t *testing.T) {
	originalArgs := os.Args
	defer func() {
		os.Args = originalArgs
	}()

	tests := []struct {
		name      string
		args      []string
		wantError bool
	}{
		{
			name:      "minimum valid port",
			args:      []string{"mcp-pdf-reader", "-mode=server", "-port=1"},
			wantError: false,
		},
		{
			name:      "maximum valid port",
			args:      []string{"mcp-pdf-reader", "-mode=server", "-port=65535"},
			wantError: false,
		},
		{
			name:      "minimum valid max file size",
			args:      []string{"mcp-pdf-reader", "-maxfilesize=1"},
			wantError: false,
		},
		{
			name:      "maximum valid max file size",
			args:      []string{"mcp-pdf-reader", "-maxfilesize=1073741824"}, // 1GB exactly
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setArgs(tt.args)
			resetFlags()

			cfg, err := LoadFromFlags()
			if tt.wantError {
				if err == nil {
					t.Errorf("LoadFromFlags() expected error but got none")
				}
				if cfg != nil {
					t.Errorf("LoadFromFlags() expected nil config on error")
				}
			} else {
				if err != nil {
					t.Errorf("LoadFromFlags() unexpected error: %v", err)
				}
				if cfg == nil {
					t.Errorf("LoadFromFlags() expected config but got nil")
				}
			}
		})
	}
}

func TestLoadFromFlags_FlagAliases(t *testing.T) {
	originalArgs := os.Args
	defer func() {
		os.Args = originalArgs
	}()

	tests := []struct {
		name string
		args []string
	}{
		{
			name: "short flag style",
			args: []string{"mcp-pdf-reader", "-mode", "server", "-host", "0.0.0.0", "-port", "9090"},
		},
		{
			name: "equals flag style",
			args: []string{"mcp-pdf-reader", "-mode=server", "-host=0.0.0.0", "-port=9090"},
		},
		{
			name: "mixed flag styles",
			args: []string{"mcp-pdf-reader", "-mode=server", "-host", "0.0.0.0", "-port=9090"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setArgs(tt.args)
			resetFlags()

			cfg, err := LoadFromFlags()
			if err != nil {
				t.Errorf("LoadFromFlags() unexpected error: %v", err)
			}
			if cfg == nil {
				t.Error("LoadFromFlags() expected config but got nil")
				return
			}

			// Verify the values were parsed correctly regardless of flag style
			if cfg.Mode != "server" {
				t.Errorf("LoadFromFlags() Mode = %v, want %v", cfg.Mode, "server")
			}
			if cfg.Host != "0.0.0.0" {
				t.Errorf("LoadFromFlags() Host = %v, want %v", cfg.Host, "0.0.0.0")
			}
			if cfg.Port != 9090 {
				t.Errorf("LoadFromFlags() Port = %v, want %v", cfg.Port, 9090)
			}
		})
	}
}

func TestLoadFromFlags_ValidationIntegration(t *testing.T) {
	// Test that LoadFromFlags properly calls Validate()
	originalArgs := os.Args
	defer func() {
		os.Args = originalArgs
	}()

	// This should pass validation
	t.Run("valid config passes validation", func(t *testing.T) {
		setArgs([]string{"mcp-pdf-reader", "-mode=stdio", "-pdfdir=/tmp"})
		resetFlags()

		cfg, err := LoadFromFlags()
		if err != nil {
			t.Errorf("LoadFromFlags() unexpected error: %v", err)
		}
		if cfg == nil {
			t.Error("LoadFromFlags() expected config but got nil")
		}
	})

	// This should fail validation
	t.Run("invalid config fails validation", func(t *testing.T) {
		setArgs([]string{"mcp-pdf-reader", "-mode=invalid"})
		resetFlags()

		cfg, err := LoadFromFlags()
		if err == nil {
			t.Error("LoadFromFlags() expected validation error but got none")
		}
		if cfg != nil {
			t.Error("LoadFromFlags() expected nil config on validation error")
		}
		if !strings.Contains(err.Error(), "invalid configuration") {
			t.Errorf("LoadFromFlags() error = %v, want error containing 'invalid configuration'", err)
		}
	})
}
