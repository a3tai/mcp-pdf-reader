package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

const (
	// Mode constants
	ModeStdio  = "stdio"
	ModeServer = "server"

	// Default values
	DefaultPort        = 8080
	DefaultHost        = "127.0.0.1"
	DefaultLogLevel    = "info"
	DefaultMaxFileSize = 100 * 1024 * 1024 // 100MB

	// Directory permissions
	DefaultDirPerm = 0o750
)

// Config holds all configuration for the PDF MCP server
type Config struct {
	// Server configuration
	Mode string // "server" or "stdio"
	Host string
	Port int

	// PDF configuration
	PDFDirectory string

	// Application configuration
	Version     string
	ServerName  string
	LogLevel    string
	MaxFileSize int64 // Maximum PDF file size in bytes
}

// DefaultConfig returns a configuration with sensible defaults
func DefaultConfig() *Config {
	currentDir, err := os.Getwd()
	if err != nil {
		// Fallback to current directory if working directory cannot be determined
		currentDir = "."
	}

	return &Config{
		Mode:         ModeStdio, // Default to stdio mode for MCP compatibility
		Host:         DefaultHost,
		Port:         DefaultPort,
		PDFDirectory: currentDir,
		Version:      "1.0.0",
		ServerName:   "mcp-pdf-reader",
		LogLevel:     DefaultLogLevel,
		MaxFileSize:  DefaultMaxFileSize,
	}
}

// LoadFromFlags parses command line flags and returns a configuration
func LoadFromFlags() (*Config, error) {
	cfg := DefaultConfig()

	// Set environment variable prefix
	viper.SetEnvPrefix("MCP_PDF")
	viper.AutomaticEnv()

	// Define flags with Viper
	viper.SetDefault("mode", cfg.Mode)
	viper.SetDefault("host", cfg.Host)
	viper.SetDefault("port", cfg.Port)
	viper.SetDefault("dir", cfg.PDFDirectory)
	viper.SetDefault("loglevel", cfg.LogLevel)
	viper.SetDefault("maxfilesize", cfg.MaxFileSize)

	// Define command line flags
	pflag.String("mode", cfg.Mode, "Server mode: 'stdio' for MCP standard I/O, 'server' for HTTP server")
	pflag.String("host", cfg.Host, "Server host address (server mode only)")
	pflag.Int("port", cfg.Port, "Server port (server mode only)")
	pflag.String("dir", cfg.PDFDirectory, "Directory containing PDF files")
	pflag.String("loglevel", cfg.LogLevel, "Log level (debug, info, warn, error)")
	pflag.Int64("maxfilesize", cfg.MaxFileSize, "Maximum PDF file size in bytes")

	// Bind command line flags
	viper.BindPFlag("mode", pflag.Lookup("mode"))
	viper.BindPFlag("host", pflag.Lookup("host"))
	viper.BindPFlag("port", pflag.Lookup("port"))
	viper.BindPFlag("dir", pflag.Lookup("dir"))
	viper.BindPFlag("loglevel", pflag.Lookup("loglevel"))
	viper.BindPFlag("maxfilesize", pflag.Lookup("maxfilesize"))

	// Custom usage message
	pflag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\nMCP PDF Reader - A Model Context Protocol server for reading PDF files\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		pflag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  %s                                         # stdio mode, current directory (default)\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s --dir=/path/to/pdfs                     # stdio mode with custom directory\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s --mode=server --dir=/path/to/pdfs       # server mode\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s --mode=server --host=0.0.0.0 --port=8081 # server on all interfaces\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\nEnvironment Variables:\n")
		fmt.Fprintf(os.Stderr, "  MCP_PDF_MODE        Server mode\n")
		fmt.Fprintf(os.Stderr, "  MCP_PDF_HOST        Server host\n")
		fmt.Fprintf(os.Stderr, "  MCP_PDF_PORT        Server port\n")
		fmt.Fprintf(os.Stderr, "  MCP_PDF_DIR         PDF directory\n")
		fmt.Fprintf(os.Stderr, "  MCP_PDF_LOGLEVEL    Log level\n")
		fmt.Fprintf(os.Stderr, "  MCP_PDF_MAXFILESIZE Maximum file size\n")
	}

	// Check for version flag before parsing
	for _, arg := range os.Args[1:] {
		if arg == "-version" || arg == "--version" || arg == "-v" {
			return nil, fmt.Errorf("version requested")
		}
	}

	pflag.Parse()

	// Populate config from Viper
	cfg.Mode = viper.GetString("mode")
	cfg.Host = viper.GetString("host")
	cfg.Port = viper.GetInt("port")
	cfg.PDFDirectory = viper.GetString("dir")
	cfg.LogLevel = viper.GetString("loglevel")
	cfg.MaxFileSize = viper.GetInt64("maxfilesize")

	// Expand paths if needed
	if cfg.PDFDirectory != "" {
		if expandedPath, err := filepath.Abs(cfg.PDFDirectory); err == nil {
			cfg.PDFDirectory = expandedPath
		}
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return cfg, nil
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	// Validate mode
	if c.Mode != ModeStdio && c.Mode != ModeServer {
		return errors.New("mode must be either 'stdio' or 'server'")
	}

	// Validate port range (only for server mode)
	if c.Mode == ModeServer && (c.Port < 1 || c.Port > 65535) {
		return errors.New("port must be between 1 and 65535")
	}

	// Validate PDF directory
	if c.PDFDirectory == "" {
		return errors.New("PDF directory cannot be empty")
	}

	// Check if PDF directory exists, create if it doesn't
	if _, err := os.Stat(c.PDFDirectory); os.IsNotExist(err) {
		if err := os.MkdirAll(c.PDFDirectory, DefaultDirPerm); err != nil {
			return fmt.Errorf("cannot create PDF directory %s: %w", c.PDFDirectory, err)
		}
	} else if err != nil {
		return fmt.Errorf("cannot access PDF directory %s: %w", c.PDFDirectory, err)
	}

	// Validate max file size
	if c.MaxFileSize <= 0 {
		return errors.New("maximum file size must be positive")
	}

	// Validate log level
	validLogLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}
	if !validLogLevels[c.LogLevel] {
		return fmt.Errorf("invalid log level: %s (must be one of: debug, info, warn, error)", c.LogLevel)
	}

	return nil
}

// Address returns the server address as host:port
func (c *Config) Address() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

// IsDebug returns true if debug logging is enabled
func (c *Config) IsDebug() bool {
	return c.LogLevel == "debug"
}

// String returns a string representation of the configuration
func (c *Config) String() string {
	return fmt.Sprintf("Config{Mode: %s, Host: %s, Port: %d, PDFDirectory: %s, LogLevel: %s, MaxFileSize: %d}",
		c.Mode, c.Host, c.Port, c.PDFDirectory, c.LogLevel, c.MaxFileSize)
}

// IsServerMode returns true if the server is running in HTTP server mode
func (c *Config) IsServerMode() bool {
	return c.Mode == ModeServer
}

// IsStdioMode returns true if the server is running in stdio mode
func (c *Config) IsStdioMode() bool {
	return c.Mode == ModeStdio
}
