package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/a3tai/mcp-pdf-reader/internal/config"
	"github.com/a3tai/mcp-pdf-reader/internal/mcp"
	"github.com/a3tai/mcp-pdf-reader/internal/pdf"
)

var (
	version   = "dev"     // This will be set by build flags
	buildTime = "unknown" // This will be set by build flags
	gitCommit = "unknown" // This will be set by build flags
)

// setupLogging configures logging based on the server mode
func setupLogging(cfg *config.Config) {
	if cfg.IsStdioMode() {
		// In stdio mode, redirect log output to stderr to avoid interfering with MCP protocol
		log.SetOutput(os.Stderr)
		// Reduce log verbosity in stdio mode unless debug is enabled
		if !cfg.IsDebug() {
			log.SetOutput(os.NewFile(0, os.DevNull))
		}
	} else {
		// In server mode, use normal stdout logging with more detail
		log.SetFlags(log.LstdFlags | log.Lshortfile)
	}
}

// runServerMode handles server mode execution with signal handling
func runServerMode(ctx context.Context, cancel context.CancelFunc, server *mcp.Server) {
	// Set up signal handling for graceful shutdown
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	// Start server in a goroutine
	serverErrCh := make(chan error, 1)
	go func() {
		serverErrCh <- server.Run(ctx)
	}()

	// Wait for shutdown signal or server error
	select {
	case sig := <-signalCh:
		log.Printf("Received signal: %s", sig)
		log.Println("Initiating graceful shutdown...")
		cancel()

		// Wait for server to shutdown
		if err := <-serverErrCh; err != nil {
			log.Printf("Server shutdown with error: %v", err)
			os.Exit(1)
		}

	case err := <-serverErrCh:
		if err != nil {
			log.Printf("Server error: %v", err)
			os.Exit(1)
		}
	}

	log.Println("Server stopped successfully")
}

// runStdioMode handles stdio mode execution
func runStdioMode(ctx context.Context, _ context.CancelFunc, server *mcp.Server) {
	// In stdio mode, the parent process controls our lifecycle
	// We should exit cleanly when stdin is closed or we get an error

	// Start server and wait for it to complete
	if err := server.Run(ctx); err != nil {
		// Only log to stderr in debug mode to avoid protocol interference
		if os.Getenv("DEBUG") != "" {
			log.Printf("Server error: %v", err)
		}
		os.Exit(1)
	}
}

func main() {
	// Check for version flag before parsing other flags
	for _, arg := range os.Args[1:] {
		if arg == "-version" || arg == "--version" || arg == "-v" {
			printVersion()
			return
		}
	}

	// Load configuration from flags first
	cfg, err := config.LoadFromFlags()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Set up logging based on mode
	setupLogging(cfg)

	// Set version if it was provided during build
	if version != "dev" {
		cfg.Version = version
	}

	if cfg.IsDebug() && cfg.IsServerMode() {
		log.Printf("Starting with configuration: %s", cfg.String())
	}

	// Create PDF service
	pdfService := pdf.NewService(cfg.MaxFileSize)

	// Create MCP server
	server, err := mcp.NewServer(cfg, pdfService)
	if err != nil {
		log.Fatalf("Failed to create MCP server: %v", err)
	}

	// Set up context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle different modes
	if cfg.IsServerMode() {
		runServerMode(ctx, cancel, server)
	} else {
		runStdioMode(ctx, cancel, server)
	}
}

// printVersion prints version information
func printVersion() {
	fmt.Printf("MCP PDF Reader\n")
	fmt.Printf("Version: %s\n", version)
	fmt.Printf("Build Time: %s\n", buildTime)
	fmt.Printf("Git Commit: %s\n", gitCommit)
	fmt.Printf("Built with: %s\n", runtime.Version())
}
