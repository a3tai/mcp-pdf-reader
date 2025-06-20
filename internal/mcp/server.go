package mcp

import (
	"context"
	"fmt"
	"log"

	"github.com/a3tai/mcp-pdf-reader/internal/config"
	"github.com/a3tai/mcp-pdf-reader/internal/pdf"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// Server represents the MCP server instance
type Server struct {
	config     *config.Config
	pdfService *pdf.Service
	mcpServer  *server.MCPServer
}

// NewServer creates a new MCP server instance
func NewServer(cfg *config.Config, pdfService *pdf.Service) (*Server, error) {
	if pdfService == nil {
		return nil, fmt.Errorf("pdfService cannot be nil")
	}

	// Create MCP server
	mcpServer := server.NewMCPServer(
		cfg.ServerName,
		cfg.Version,
		server.WithToolCapabilities(false), // We don't support dynamic tool capabilities
	)

	s := &Server{
		config:     cfg,
		pdfService: pdfService,
		mcpServer:  mcpServer,
	}

	// Register tools
	s.registerTools()

	return s, nil
}

// registerTools registers all available MCP tools
func (s *Server) registerTools() {
	// Register PDF read file tool
	pdfReadFileTool := mcp.NewTool(
		"pdf_read_file",
		mcp.WithDescription("Read and extract text content from a PDF file"),
		mcp.WithString("path",
			mcp.Required(),
			mcp.Description("Full path to the PDF file"),
		),
	)
	s.mcpServer.AddTool(pdfReadFileTool, s.handlePDFReadFile)

	// Register PDF assets file tool
	pdfAssetsFileTool := mcp.NewTool(
		"pdf_assets_file",
		mcp.WithDescription("Extract visual assets like images from a PDF file"),
		mcp.WithString("path",
			mcp.Required(),
			mcp.Description("Full path to the PDF file"),
		),
	)
	s.mcpServer.AddTool(pdfAssetsFileTool, s.handlePDFAssetsFile)

	// Register PDF validate file tool
	pdfValidateFileTool := mcp.NewTool(
		"pdf_validate_file",
		mcp.WithDescription("Validate if a file is a readable PDF"),
		mcp.WithString("path",
			mcp.Required(),
			mcp.Description("Full path to the PDF file"),
		),
	)
	s.mcpServer.AddTool(pdfValidateFileTool, s.handlePDFValidateFile)

	// Register PDF stats file tool
	pdfStatsFileTool := mcp.NewTool(
		"pdf_stats_file",
		mcp.WithDescription("Get detailed statistics about a PDF file"),
		mcp.WithString("path",
			mcp.Required(),
			mcp.Description("Full path to the PDF file"),
		),
	)
	s.mcpServer.AddTool(pdfStatsFileTool, s.handlePDFStatsFile)

	// Register PDF search directory tool
	pdfSearchDirectoryTool := mcp.NewTool(
		"pdf_search_directory",
		mcp.WithDescription("Search for PDF files in a directory with optional fuzzy search"),
		mcp.WithString("directory",
			mcp.Description("Directory path to search (uses default if empty)"),
		),
		mcp.WithString("query",
			mcp.Description("Optional search query for fuzzy matching"),
		),
	)
	s.mcpServer.AddTool(pdfSearchDirectoryTool, s.handlePDFSearchDirectory)

	// Register PDF stats directory tool
	pdfStatsDirectoryTool := mcp.NewTool(
		"pdf_stats_directory",
		mcp.WithDescription("Get statistics about PDF files in a directory"),
		mcp.WithString("directory",
			mcp.Description("Directory path to analyze (uses default if empty)"),
		),
	)
	s.mcpServer.AddTool(pdfStatsDirectoryTool, s.handlePDFStatsDirectory)

	// Register PDF server info tool
	pdfServerInfoTool := mcp.NewTool(
		"pdf_server_info",
		mcp.WithDescription("Get server information, available tools, directory contents, and usage guidance"),
	)
	s.mcpServer.AddTool(pdfServerInfoTool, s.handlePDFServerInfo)
}

// Handler functions
func (s *Server) handlePDFReadFile(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	path, err := request.RequireString("path")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	req := pdf.PDFReadFileRequest{Path: path}
	result, err := s.pdfService.PDFReadFile(req)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	responseText := fmt.Sprintf("Successfully read PDF: %s\n", result.Path)
	responseText += fmt.Sprintf("Pages: %d\n", result.Pages)
	responseText += fmt.Sprintf("Size: %d bytes\n", result.Size)
	responseText += fmt.Sprintf("Content Type: %s\n", result.ContentType)
	responseText += fmt.Sprintf("Has Images: %t\n", result.HasImages)
	if result.HasImages {
		responseText += fmt.Sprintf("Image Count: %d\n", result.ImageCount)
	}

	// Add guidance based on content type
	switch result.ContentType {
	case "scanned_images":
		responseText += "\n🔍 RECOMMENDATION: This PDF appears to contain scanned images with little or no extractable text. Consider using 'pdf_assets_file' to extract the images.\n"
	case "mixed":
		responseText += "\n💡 INFO: This PDF contains both text and images. You may want to use 'pdf_assets_file' to extract the images as well.\n"
	case "no_content":
		responseText += "\n⚠️  WARNING: This PDF appears to have no readable content or images.\n"
	}

	responseText += "\nContent:\n"
	responseText += result.Content

	return mcp.NewToolResultText(responseText), nil
}

func (s *Server) handlePDFAssetsFile(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	path, err := request.RequireString("path")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	req := pdf.PDFAssetsFileRequest{Path: path}
	result, err := s.pdfService.PDFAssetsFile(req)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	responseText := s.formatPDFAssetsFileResult(result)
	return mcp.NewToolResultText(responseText), nil
}

func (s *Server) handlePDFValidateFile(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	path, err := request.RequireString("path")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	req := pdf.PDFValidateFileRequest{Path: path}
	result, err := s.pdfService.PDFValidateFile(req)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	var responseText string
	if result.Valid {
		responseText = fmt.Sprintf("PDF file %s is valid and readable", result.Path)
	} else {
		responseText = fmt.Sprintf("PDF validation failed for %s: %s", result.Path, result.Message)
	}

	return mcp.NewToolResultText(responseText), nil
}

func (s *Server) handlePDFStatsFile(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	path, err := request.RequireString("path")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	req := pdf.PDFStatsFileRequest{Path: path}
	result, err := s.pdfService.PDFStatsFile(req)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	responseText := s.formatPDFStatsFileResult(result)
	return mcp.NewToolResultText(responseText), nil
}

func (s *Server) handlePDFSearchDirectory(ctx context.Context, request mcp.CallToolRequest) (
	*mcp.CallToolResult, error,
) {
	args := request.GetArguments()

	directory := s.config.PDFDirectory // default
	if dir, ok := args["directory"].(string); ok && dir != "" {
		directory = dir
	}

	query := ""
	if q, ok := args["query"].(string); ok {
		query = q
	}

	req := pdf.PDFSearchDirectoryRequest{
		Directory: directory,
		Query:     query,
	}

	result, err := s.pdfService.PDFSearchDirectory(req)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	var responseText string
	if result.TotalCount == 0 {
		responseText = fmt.Sprintf("No PDF files found in directory: %s", result.Directory)
		if result.SearchQuery != "" {
			responseText += fmt.Sprintf(" (searched for: %s)", result.SearchQuery)
		}
	} else {
		responseText = s.formatPDFSearchDirectoryResult(result)
	}

	return mcp.NewToolResultText(responseText), nil
}

func (s *Server) handlePDFStatsDirectory(ctx context.Context, request mcp.CallToolRequest) (
	*mcp.CallToolResult, error,
) {
	args := request.GetArguments()

	directory := s.config.PDFDirectory // default
	if dir, ok := args["directory"].(string); ok && dir != "" {
		directory = dir
	}

	req := pdf.PDFStatsDirectoryRequest{Directory: directory}
	result, err := s.pdfService.PDFStatsDirectory(req)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	responseText := s.formatPDFStatsDirectoryResult(result)
	return mcp.NewToolResultText(responseText), nil
}

func (s *Server) handlePDFServerInfo(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	req := pdf.PDFServerInfoRequest{}
	result, err := s.pdfService.PDFServerInfo(req, s.config.ServerName, s.config.Version, s.config.PDFDirectory)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	responseText := s.formatPDFServerInfoResult(result)
	return mcp.NewToolResultText(responseText), nil
}

// Formatting methods
func (s *Server) formatPDFSearchDirectoryResult(result *pdf.PDFSearchDirectoryResult) string {
	text := fmt.Sprintf("Found %d PDF file(s) in directory: %s\n", result.TotalCount, result.Directory)
	if result.SearchQuery != "" {
		text += fmt.Sprintf("Search query: %s\n", result.SearchQuery)
	}
	text += "\nFiles:\n"

	for i, file := range result.Files {
		text += fmt.Sprintf("%d. %s\n", i+1, file.Name)
		text += fmt.Sprintf("   Path: %s\n", file.Path)
		text += fmt.Sprintf("   Size: %d bytes\n", file.Size)
		text += fmt.Sprintf("   Modified: %s\n", file.ModifiedTime)
		if i < len(result.Files)-1 {
			text += "\n"
		}
	}

	return text
}

func (s *Server) formatPDFStatsDirectoryResult(result *pdf.PDFStatsDirectoryResult) string {
	text := "PDF Directory Statistics\n"
	text += fmt.Sprintf("Directory: %s\n", result.Directory)
	text += fmt.Sprintf("Total PDF files: %d\n", result.TotalFiles)
	text += fmt.Sprintf("Total size: %d bytes\n", result.TotalSize)

	if result.TotalFiles > 0 {
		text += fmt.Sprintf("Average file size: %d bytes\n", result.AverageFileSize)
		if result.LargestFileName != "" {
			text += fmt.Sprintf("Largest file: %s (%d bytes)\n", result.LargestFileName, result.LargestFileSize)
		}
		if result.SmallestFileName != "" {
			text += fmt.Sprintf("Smallest file: %s (%d bytes)\n", result.SmallestFileName, result.SmallestFileSize)
		}
	}

	return text
}

func (s *Server) formatPDFStatsFileResult(result *pdf.PDFStatsFileResult) string {
	text := "PDF File Statistics\n"
	text += fmt.Sprintf("File: %s\n", result.Path)
	text += fmt.Sprintf("Size: %d bytes\n", result.Size)
	text += fmt.Sprintf("Pages: %d\n", result.Pages)
	text += fmt.Sprintf("Modified: %s\n", result.ModifiedDate)

	if result.Title != "" {
		text += fmt.Sprintf("Title: %s\n", result.Title)
	}
	if result.Author != "" {
		text += fmt.Sprintf("Author: %s\n", result.Author)
	}
	if result.Subject != "" {
		text += fmt.Sprintf("Subject: %s\n", result.Subject)
	}
	if result.Producer != "" {
		text += fmt.Sprintf("Producer: %s\n", result.Producer)
	}
	if result.CreatedDate != "" {
		text += fmt.Sprintf("Created: %s\n", result.CreatedDate)
	}

	return text
}

func (s *Server) formatPDFAssetsFileResult(result *pdf.PDFAssetsFileResult) string {
	text := fmt.Sprintf("PDF Assets for: %s\n", result.Path)
	text += fmt.Sprintf("Total images found: %d\n", result.TotalCount)

	if result.TotalCount > 0 {
		text += "\nImages:\n"
		for i, img := range result.Images {
			text += fmt.Sprintf("%d. Page %d: %dx%d pixels, Format: %s",
				i+1, img.PageNumber, img.Width, img.Height, img.Format)
			if img.Size > 0 {
				text += fmt.Sprintf(", Size: %d bytes", img.Size)
			}
			text += "\n"
		}
	}

	return text
}

func (s *Server) formatPDFServerInfoResult(result *pdf.PDFServerInfoResult) string {
	text := fmt.Sprintf("📋 %s v%s - Server Information\n", result.ServerName, result.Version)
	text += fmt.Sprintf("📁 Default Directory: %s\n", result.DefaultDirectory)
	text += fmt.Sprintf("📏 Max File Size: %d MB\n\n", result.MaxFileSize/(1024*1024))

	// Directory contents
	if len(result.DirectoryContents) > 0 {
		text += fmt.Sprintf("📂 Directory Contents (%d PDF files found):\n", len(result.DirectoryContents))
		for i, file := range result.DirectoryContents {
			if i >= 10 { // Limit to first 10 files for readability
				text += fmt.Sprintf("   ... and %d more files\n", len(result.DirectoryContents)-10)
				break
			}
			text += fmt.Sprintf("   %d. %s (%d bytes)\n", i+1, file.Name, file.Size)
		}
		text += "\n"
	} else {
		text += "📂 Directory Contents: No PDF files found in default directory\n\n"
	}

	// Available tools
	text += "🛠️  Available Tools:\n"
	for _, tool := range result.AvailableTools {
		text += fmt.Sprintf("\n• %s\n", tool.Name)
		text += fmt.Sprintf("  Description: %s\n", tool.Description)
		text += fmt.Sprintf("  Usage: %s\n", tool.Usage)
		text += fmt.Sprintf("  Parameters: %s\n", tool.Parameters)
	}

	// Supported formats
	if len(result.SupportedFormats) > 0 {
		text += "\n🖼️  Supported Image Formats:\n"
		for _, format := range result.SupportedFormats {
			text += fmt.Sprintf("  • %s\n", format)
		}
	}

	// Usage guidance
	text += "\n" + result.UsageGuidance

	return text
}

// Run starts the MCP server in the configured mode
func (s *Server) Run(ctx context.Context) error {
	if s.config.IsServerMode() {
		return s.runServerMode(ctx)
	} else {
		return s.runStdioMode(ctx)
	}
}

// runStdioMode runs the server in stdio mode
func (s *Server) runStdioMode(_ context.Context) error {
	if s.config.IsDebug() {
		log.Printf("Starting PDF MCP server in stdio mode")
		log.Printf("PDF directory: %s", s.config.PDFDirectory)
	}

	// Use the mark3labs/mcp-go server.ServeStdio function
	if err := server.ServeStdio(s.mcpServer); err != nil {
		return fmt.Errorf("failed to serve stdio: %w", err)
	}
	return nil
}

// runServerMode runs the server in HTTP server mode
func (s *Server) runServerMode(ctx context.Context) error {
	// For now, we'll just use stdio mode since the mark3labs library
	// handles the transport differently
	log.Printf("Server mode not yet implemented with mark3labs/mcp-go")
	log.Printf("Falling back to stdio mode")
	return s.runStdioMode(ctx)
}
