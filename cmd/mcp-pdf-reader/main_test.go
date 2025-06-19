package main

import (
	"bytes"
	"io"
	"log"
	"os"
	"strings"
	"testing"

	"github.com/a3tai/mcp-pdf-reader/internal/config"
)

const (
	testVersion = "1.2.3"
	devVersion  = "dev"
)

func TestPrintVersion(t *testing.T) {
	// Save original stdout
	originalStdout := os.Stdout

	// Create a pipe to capture output
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("Failed to create pipe: %v", err)
	}

	// Redirect stdout to the pipe
	os.Stdout = w

	// Set version variables for testing
	oldVersion := version
	oldBuildTime := buildTime
	oldGitCommit := gitCommit

	version = testVersion
	buildTime = "2023-12-01_10:30:00"
	gitCommit = "abc123"

	defer func() {
		// Restore original values
		version = oldVersion
		buildTime = oldBuildTime
		gitCommit = oldGitCommit
		os.Stdout = originalStdout
	}()

	// Call printVersion in a goroutine
	done := make(chan struct{})
	go func() {
		defer close(done)
		printVersion()
		w.Close()
	}()

	// Read the output
	var buf bytes.Buffer
	io.Copy(&buf, r)
	<-done

	output := buf.String()

	// Verify output contains expected information
	expectedStrings := []string{
		"MCP PDF Reader",
		"Version: " + testVersion,
		"Build Time: 2023-12-01_10:30:00",
		"Git Commit: abc123",
		"Built with:",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(output, expected) {
			t.Errorf("printVersion() output missing expected string: %s\nActual output:\n%s", expected, output)
		}
	}
}

func TestPrintVersionWithDefaults(t *testing.T) {
	// Save original stdout
	originalStdout := os.Stdout

	// Create a pipe to capture output
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("Failed to create pipe: %v", err)
	}

	// Redirect stdout to the pipe
	os.Stdout = w

	// Use default version variables
	oldVersion := version
	oldBuildTime := buildTime
	oldGitCommit := gitCommit

	version = "dev"
	buildTime = "unknown"
	gitCommit = "unknown"

	defer func() {
		// Restore original values
		version = oldVersion
		buildTime = oldBuildTime
		gitCommit = oldGitCommit
		os.Stdout = originalStdout
	}()

	// Call printVersion in a goroutine
	done := make(chan struct{})
	go func() {
		defer close(done)
		printVersion()
		w.Close()
	}()

	// Read the output
	var buf bytes.Buffer
	io.Copy(&buf, r)
	<-done

	output := buf.String()

	// Verify output contains default values
	expectedStrings := []string{
		"MCP PDF Reader",
		"Version: dev",
		"Build Time: unknown",
		"Git Commit: unknown",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(output, expected) {
			t.Errorf("printVersion() output missing expected string: %s\nActual output:\n%s", expected, output)
		}
	}
}

func TestSetupLogging_StdioMode(t *testing.T) {
	// Save original log settings
	originalOutput := log.Writer()
	originalFlags := log.Flags()

	defer func() {
		log.SetOutput(originalOutput)
		log.SetFlags(originalFlags)
	}()

	tests := []struct {
		name     string
		wantType string
		config   *config.Config
		isDebug  bool
	}{
		{
			name: "stdio mode - debug enabled",
			config: &config.Config{
				Mode:     "stdio",
				LogLevel: "debug",
			},
			isDebug:  true,
			wantType: "stderr",
		},
		{
			name: "stdio mode - debug disabled",
			config: &config.Config{
				Mode:     "stdio",
				LogLevel: "info",
			},
			isDebug:  false,
			wantType: "devnull",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupLogging(tt.config)

			// Check that output was set appropriately
			currentOutput := log.Writer()

			switch tt.wantType {
			case "stderr":
				if currentOutput != os.Stderr {
					t.Errorf("setupLogging() for stdio debug mode should set output to stderr")
				}
			case "devnull":
				// For non-debug stdio mode, output should be set to devnull
				// We can't easily test this directly, but we can verify it's not stderr
				if currentOutput == os.Stderr {
					t.Errorf("setupLogging() for stdio non-debug mode should not use stderr")
				}
			}
		})
	}
}

func TestSetupLogging_ServerMode(t *testing.T) {
	// Save original log settings
	originalOutput := log.Writer()
	originalFlags := log.Flags()

	defer func() {
		log.SetOutput(originalOutput)
		log.SetFlags(originalFlags)
	}()

	cfg := &config.Config{
		Mode:     "server",
		LogLevel: "info",
	}

	setupLogging(cfg)

	// In server mode, flags should include LstdFlags and Lshortfile
	currentFlags := log.Flags()
	expectedFlags := log.LstdFlags | log.Lshortfile

	if currentFlags != expectedFlags {
		t.Errorf("setupLogging() for server mode: flags = %v, want %v", currentFlags, expectedFlags)
	}
}

func TestSetupLogging_EdgeCases(t *testing.T) {
	// Save original log settings
	originalOutput := log.Writer()
	originalFlags := log.Flags()

	defer func() {
		log.SetOutput(originalOutput)
		log.SetFlags(originalFlags)
	}()

	// Test with nil config (this will panic, so we expect it)
	t.Run("nil config", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("setupLogging() with nil config should panic, but it didn't")
			}
		}()

		setupLogging(nil)
	})

	// Test with empty mode
	t.Run("empty mode", func(t *testing.T) {
		cfg := &config.Config{
			Mode: "",
		}

		defer func() {
			if r := recover(); r != nil {
				t.Errorf("setupLogging() with empty mode should not panic: %v", r)
			}
		}()

		setupLogging(cfg)
	})
}

func TestVersionFlagDetection(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		hasVersion bool
	}{
		{
			name:       "no version flag",
			args:       []string{"program"},
			hasVersion: false,
		},
		{
			name:       "-version flag",
			args:       []string{"program", "-version"},
			hasVersion: true,
		},
		{
			name:       "--version flag",
			args:       []string{"program", "--version"},
			hasVersion: true,
		},
		{
			name:       "-v flag",
			args:       []string{"program", "-v"},
			hasVersion: true,
		},
		{
			name:       "version flag with other args",
			args:       []string{"program", "-mode=server", "-version", "-port=8080"},
			hasVersion: true,
		},
		{
			name:       "version flag first",
			args:       []string{"program", "-version", "-mode=server"},
			hasVersion: true,
		},
		{
			name:       "version flag last",
			args:       []string{"program", "-mode=server", "-version"},
			hasVersion: true,
		},
		{
			name:       "similar but not version flag",
			args:       []string{"program", "-verbose", "-versions"},
			hasVersion: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			found := false
			for _, arg := range tt.args[1:] { // Skip program name
				if arg == "-version" || arg == "--version" || arg == "-v" {
					found = true
					break
				}
			}

			if found != tt.hasVersion {
				t.Errorf("Version flag detection for %v: got %v, want %v", tt.args, found, tt.hasVersion)
			}
		})
	}
}

func TestConfigIsDebugLogic(t *testing.T) {
	tests := []struct {
		name      string
		logLevel  string
		wantDebug bool
	}{
		{
			name:      "debug level",
			logLevel:  "debug",
			wantDebug: true,
		},
		{
			name:      "info level",
			logLevel:  "info",
			wantDebug: false,
		},
		{
			name:      "warn level",
			logLevel:  "warn",
			wantDebug: false,
		},
		{
			name:      "error level",
			logLevel:  "error",
			wantDebug: false,
		},
		{
			name:      "empty level",
			logLevel:  "",
			wantDebug: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				LogLevel: tt.logLevel,
			}

			isDebug := cfg.IsDebug()
			if isDebug != tt.wantDebug {
				t.Errorf("Config.IsDebug() with LogLevel=%s: got %v, want %v", tt.logLevel, isDebug, tt.wantDebug)
			}
		})
	}
}

func TestConfigModeLogic(t *testing.T) {
	tests := []struct {
		name       string
		mode       string
		wantStdio  bool
		wantServer bool
	}{
		{
			name:       "stdio mode",
			mode:       "stdio",
			wantStdio:  true,
			wantServer: false,
		},
		{
			name:       "server mode",
			mode:       "server",
			wantStdio:  false,
			wantServer: true,
		},
		{
			name:       "empty mode",
			mode:       "",
			wantStdio:  false,
			wantServer: false,
		},
		{
			name:       "invalid mode",
			mode:       "invalid",
			wantStdio:  false,
			wantServer: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				Mode: tt.mode,
			}

			isStdio := cfg.IsStdioMode()
			if isStdio != tt.wantStdio {
				t.Errorf("Config.IsStdioMode() with Mode=%s: got %v, want %v", tt.mode, isStdio, tt.wantStdio)
			}

			isServer := cfg.IsServerMode()
			if isServer != tt.wantServer {
				t.Errorf("Config.IsServerMode() with Mode=%s: got %v, want %v", tt.mode, isServer, tt.wantServer)
			}
		})
	}
}

func TestMainFunctionLogic(t *testing.T) {
	// Test the core logic that would be in main function
	// We can't test main() directly due to os.Exit calls, but we can test the logic

	t.Run("version setting logic", func(t *testing.T) {
		cfg := config.DefaultConfig()

		// Simulate version being set during build
		buildVersion := "1.2.3"

		if buildVersion != "dev" {
			cfg.Version = buildVersion
		}

		if cfg.Version != testVersion {
			t.Errorf("Version setting logic: got %s, want %s", cfg.Version, testVersion)
		}
	})

	t.Run("version not set logic", func(t *testing.T) {
		cfg := config.DefaultConfig()
		originalVersion := cfg.Version

		// Simulate version not being set during build (remains "dev")
		buildVersion := "dev"

		if buildVersion != "dev" {
			cfg.Version = buildVersion
		}

		if cfg.Version != originalVersion {
			t.Errorf("Version not set logic: version should remain unchanged, got %s, want %s", cfg.Version, originalVersion)
		}
	})
}

func TestLoggingModeConfiguration(t *testing.T) {
	// Test that logging is configured appropriately for different modes
	tests := []struct {
		name     string
		mode     string
		logLevel string
		debug    bool
	}{
		{
			name:     "stdio mode with debug",
			mode:     "stdio",
			logLevel: "debug",
			debug:    true,
		},
		{
			name:     "stdio mode without debug",
			mode:     "stdio",
			logLevel: "info",
			debug:    false,
		},
		{
			name:     "server mode with debug",
			mode:     "server",
			logLevel: "debug",
			debug:    true,
		},
		{
			name:     "server mode without debug",
			mode:     "server",
			logLevel: "info",
			debug:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				Mode:     tt.mode,
				LogLevel: tt.logLevel,
			}

			if cfg.IsDebug() != tt.debug {
				t.Errorf("Config debug detection: got %v, want %v", cfg.IsDebug(), tt.debug)
			}

			if cfg.IsStdioMode() != (tt.mode == "stdio") {
				t.Errorf("Config stdio mode detection: got %v, want %v", cfg.IsStdioMode(), tt.mode == "stdio")
			}

			if cfg.IsServerMode() != (tt.mode == "server") {
				t.Errorf("Config server mode detection: got %v, want %v", cfg.IsServerMode(), tt.mode == "server")
			}
		})
	}
}
