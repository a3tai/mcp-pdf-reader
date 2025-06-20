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
	s.registerBasicTools()
	s.registerExtractionTools()
	s.registerUtilityTools()
}

// registerBasicTools registers basic PDF manipulation tools
func (s *Server) registerBasicTools() {
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
}

// registerExtractionTools registers structured extraction tools
func (s *Server) registerExtractionTools() {
	// Register PDF extract structured tool
	pdfExtractStructuredTool := mcp.NewTool(
		"pdf_extract_structured",
		mcp.WithDescription("Extract structured content with positioning and formatting information"),
		mcp.WithString("path",
			mcp.Required(),
			mcp.Description("Full path to the PDF file"),
		),
		mcp.WithString("mode",
			mcp.Description("Extraction mode: raw, structured, semantic, table, complete (default: structured)"),
		),
		mcp.WithString("config",
			mcp.Description("JSON string with extraction configuration options"),
		),
	)
	s.mcpServer.AddTool(pdfExtractStructuredTool, s.handlePDFExtractStructured)

	// Register PDF extract tables tool
	pdfExtractTablesTool := mcp.NewTool(
		"pdf_extract_tables",
		mcp.WithDescription("Extract tabular data from PDF with structure preservation"),
		mcp.WithString("path",
			mcp.Required(),
			mcp.Description("Full path to the PDF file"),
		),
		mcp.WithString("config",
			mcp.Description("JSON string with extraction configuration options"),
		),
	)
	s.mcpServer.AddTool(pdfExtractTablesTool, s.handlePDFExtractTables)

	// Register PDF extract semantic tool
	pdfExtractSemanticTool := mcp.NewTool(
		"pdf_extract_semantic",
		mcp.WithDescription("Extract content with semantic grouping and relationship detection"),
		mcp.WithString("path",
			mcp.Required(),
			mcp.Description("Full path to the PDF file"),
		),
		mcp.WithString("config",
			mcp.Description("JSON string with extraction configuration options"),
		),
	)
	s.mcpServer.AddTool(pdfExtractSemanticTool, s.handlePDFExtractSemantic)

	// Register PDF extract complete tool
	pdfExtractCompleteTool := mcp.NewTool(
		"pdf_extract_complete",
		mcp.WithDescription("Comprehensive extraction of all content types (text, images, tables, forms, annotations)"),
		mcp.WithString("path",
			mcp.Required(),
			mcp.Description("Full path to the PDF file"),
		),
		mcp.WithString("config",
			mcp.Description("JSON string with extraction configuration options"),
		),
	)
	s.mcpServer.AddTool(pdfExtractCompleteTool, s.handlePDFExtractComplete)

	// Register PDF query content tool
	pdfQueryContentTool := mcp.NewTool(
		"pdf_query_content",
		mcp.WithDescription("Query and filter extracted PDF content using search criteria"),
		mcp.WithString("path",
			mcp.Required(),
			mcp.Description("Full path to the PDF file"),
		),
		mcp.WithString("query",
			mcp.Required(),
			mcp.Description("JSON string with query criteria for filtering content"),
		),
	)
	s.mcpServer.AddTool(pdfQueryContentTool, s.handlePDFQueryContent)
}

// registerUtilityTools registers utility and information tools
func (s *Server) registerUtilityTools() {
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

	// Register PDF get page info tool
	pdfGetPageInfoTool := mcp.NewTool(
		"pdf_get_page_info",
		mcp.WithDescription("Get detailed information about PDF pages (dimensions, layout, etc.)"),
		mcp.WithString("path",
			mcp.Required(),
			mcp.Description("Full path to the PDF file"),
		),
	)
	s.mcpServer.AddTool(pdfGetPageInfoTool, s.handlePDFGetPageInfo)

	// Register PDF get metadata tool
	pdfGetMetadataTool := mcp.NewTool(
		"pdf_get_metadata",
		mcp.WithDescription("Extract comprehensive document metadata and properties"),
		mcp.WithString("path",
			mcp.Required(),
			mcp.Description("Full path to the PDF file"),
		),
	)
	s.mcpServer.AddTool(pdfGetMetadataTool, s.handlePDFGetMetadata)
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
		responseText += "\n🔍 RECOMMENDATION: This PDF appears to contain scanned images with little or no " +
			"extractable text. Consider using 'pdf_assets_file' to extract the images.\n"
	case "mixed":
		responseText += "\n💡 INFO: This PDF contains both text and images. You may want to use " +
			"'pdf_assets_file' to extract the images as well.\n"
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

// New structured extraction handlers

func (s *Server) handlePDFExtractStructured(
	ctx context.Context, request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	path, err := request.RequireString("path")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	args := request.GetArguments()

	req := pdf.PDFExtractStructuredRequest{
		Path: path,
	}

	// Handle optional mode parameter
	if mode, ok := args["mode"].(string); ok {
		req.Mode = mode
	}

	// Handle optional config parameter (simplified for now)
	if configStr, ok := args["config"].(string); ok && configStr != "" {
		// For now, just use default config
		// TODO: Parse JSON config string when needed
		req.Config = pdf.ExtractionConfig{
			ExtractText:        true,
			IncludeCoordinates: true,
			IncludeFormatting:  true,
		}
	}

	result, err := s.pdfService.ExtractStructured(req)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	responseText := s.formatPDFExtractResult(result)
	return mcp.NewToolResultText(responseText), nil
}

func (s *Server) handlePDFExtractTables(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return s.handleExtractionRequest(request,
		func(path string, config pdf.ExtractionConfig) (*pdf.PDFExtractResult, error) {
			return s.pdfService.ExtractTables(pdf.PDFExtractTablesRequest{Path: path, Config: config})
		}, pdf.ExtractionConfig{
			ExtractText:        true,
			ExtractTables:      true,
			IncludeCoordinates: true,
		})
}

func (s *Server) handlePDFExtractSemantic(
	ctx context.Context, request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	return s.handleExtractionRequest(request,
		func(path string, config pdf.ExtractionConfig) (*pdf.PDFExtractResult, error) {
			return s.pdfService.ExtractSemantic(pdf.PDFExtractSemanticRequest{Path: path, Config: config})
		}, pdf.ExtractionConfig{
			ExtractText:        true,
			IncludeCoordinates: true,
			IncludeFormatting:  true,
		})
}

// handleExtractionRequest is a common handler for extraction requests
func (s *Server) handleExtractionRequest(
	request mcp.CallToolRequest,
	handler func(string, pdf.ExtractionConfig) (*pdf.PDFExtractResult, error),
	defaultConfig pdf.ExtractionConfig,
) (*mcp.CallToolResult, error) {
	path, err := request.RequireString("path")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	args := request.GetArguments()
	config := defaultConfig

	// Handle optional config parameter (simplified for now)
	if configStr, ok := args["config"].(string); ok && configStr != "" {
		// For now, just use default config
		// TODO: Parse JSON config string when needed
		config = defaultConfig
	}

	result, err := handler(path, config)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	responseText := s.formatPDFExtractResult(result)
	return mcp.NewToolResultText(responseText), nil
}

func (s *Server) handlePDFExtractComplete(
	ctx context.Context, request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	path, err := request.RequireString("path")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	args := request.GetArguments()

	req := pdf.PDFExtractCompleteRequest{
		Path: path,
	}

	// Handle optional config parameter (simplified for now)
	if configStr, ok := args["config"].(string); ok && configStr != "" {
		// For now, just use default config for complete extraction
		req.Config = pdf.ExtractionConfig{
			ExtractText:        true,
			ExtractImages:      true,
			ExtractTables:      true,
			ExtractForms:       true,
			ExtractAnnotations: true,
			IncludeCoordinates: true,
			IncludeFormatting:  true,
		}
	}

	result, err := s.pdfService.ExtractComplete(req)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	responseText := s.formatPDFExtractResult(result)
	return mcp.NewToolResultText(responseText), nil
}

func (s *Server) handlePDFQueryContent(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	path, err := request.RequireString("path")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	queryStr, err := request.RequireString("query")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// For now, create a simple query based on the query string
	// TODO: Parse JSON query string when needed
	query := pdf.ContentQuery{
		TextQuery: queryStr,
	}

	req := pdf.PDFQueryContentRequest{
		Path:  path,
		Query: query,
	}

	result, err := s.pdfService.QueryContent(req)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	responseText := s.formatPDFQueryResult(result)
	return mcp.NewToolResultText(responseText), nil
}

func (s *Server) handlePDFGetPageInfo(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	path, err := request.RequireString("path")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	req := pdf.PDFGetPageInfoRequest{Path: path}
	result, err := s.pdfService.GetPageInfo(req)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	responseText := s.formatPDFPageInfoResult(result)
	return mcp.NewToolResultText(responseText), nil
}

func (s *Server) handlePDFGetMetadata(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	path, err := request.RequireString("path")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	req := pdf.PDFGetMetadataRequest{Path: path}
	result, err := s.pdfService.GetMetadata(req)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	responseText := s.formatPDFMetadataResult(result)
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

// New formatting methods for structured extraction results

func (s *Server) formatPDFExtractResult(result *pdf.PDFExtractResult) string {
	text := fmt.Sprintf("📄 PDF Extraction Results: %s\n", result.FilePath)
	text += fmt.Sprintf("🔧 Mode: %s\n", result.Mode)
	text += fmt.Sprintf("📖 Pages: %d (processed: %v)\n", result.TotalPages, result.ProcessedPages)
	text += fmt.Sprintf("🎯 Quality: %s\n", result.Summary.Quality)
	text += fmt.Sprintf("📊 Total Elements: %d\n\n", result.Summary.TotalElements)

	// Content type breakdown
	text += "📋 Content Types Found:\n"
	for contentType, count := range result.Summary.ContentTypes {
		text += fmt.Sprintf("  • %s: %d\n", contentType, count)
	}
	text += "\n"

	// Tables if found
	if len(result.Tables) > 0 {
		text += fmt.Sprintf("📊 Tables Found: %d\n", len(result.Tables))
		for i, table := range result.Tables {
			text += fmt.Sprintf("  Table %d: %d rows × %d columns (%d cells)\n",
				i+1, len(table.Rows), len(table.Columns), table.CellCount)
			if table.HasHeaders {
				text += "    - Has headers\n"
			}
			text += fmt.Sprintf("    - Confidence: %.2f\n", table.Confidence)
		}
		text += "\n"
	}

	// Page breakdown
	if len(result.Summary.PageBreakdown) > 0 {
		text += "📄 Page Breakdown:\n"
		for _, page := range result.Summary.PageBreakdown {
			text += fmt.Sprintf("  Page %d: %d elements\n", page.Page, page.Elements)
		}
		text += "\n"
	}

	// Suggestions
	if len(result.Summary.Suggestions) > 0 {
		text += "💡 Suggestions:\n"
		for _, suggestion := range result.Summary.Suggestions {
			text += fmt.Sprintf("  • %s\n", suggestion)
		}
		text += "\n"
	}

	// Warnings and errors
	if len(result.Warnings) > 0 {
		text += "⚠️  Warnings:\n"
		for _, warning := range result.Warnings {
			text += fmt.Sprintf("  • %s\n", warning)
		}
		text += "\n"
	}

	if len(result.Errors) > 0 {
		text += "❌ Errors:\n"
		for _, error := range result.Errors {
			text += fmt.Sprintf("  • %s\n", error)
		}
		text += "\n"
	}

	// Show first few elements as examples
	if len(result.Elements) > 0 {
		text += fmt.Sprintf("🔍 Content Elements (showing first %d):\n", minInt(5, len(result.Elements)))
		for i, element := range result.Elements {
			if i >= 5 {
				text += fmt.Sprintf("  ... and %d more elements\n", len(result.Elements)-5)
				break
			}
			text += fmt.Sprintf("  %d. %s on page %d (confidence: %.2f)\n",
				i+1, element.Type, element.PageNumber, element.Confidence)

			// Show content preview for text elements
			if element.Type == "text" {
				if contentStr, ok := element.Content.(string); ok {
					preview := contentStr
					if len(preview) > 100 {
						preview = preview[:100] + "..."
					}
					text += fmt.Sprintf("     Content: %s\n", preview)
				}
			}
		}
	}

	return text
}

func (s *Server) formatPDFQueryResult(result *pdf.PDFQueryResult) string {
	text := fmt.Sprintf("🔍 Query Results: %s\n", result.FilePath)
	text += fmt.Sprintf("📊 Matches Found: %d\n", result.MatchCount)
	text += fmt.Sprintf("🎯 Average Confidence: %.2f\n\n", result.Summary.Confidence)

	// Query details
	text += "🔎 Query Details:\n"
	if len(result.Query.ContentTypes) > 0 {
		text += fmt.Sprintf("  Content Types: %v\n", result.Query.ContentTypes)
	}
	if len(result.Query.Pages) > 0 {
		text += fmt.Sprintf("  Pages: %v\n", result.Query.Pages)
	}
	if result.Query.TextQuery != "" {
		text += fmt.Sprintf("  Text Query: %s\n", result.Query.TextQuery)
	}
	if result.Query.MinConfidence > 0 {
		text += fmt.Sprintf("  Min Confidence: %.2f\n", result.Query.MinConfidence)
	}
	text += "\n"

	// Result breakdown
	if len(result.Summary.TypeBreakdown) > 0 {
		text += "📋 Result Breakdown by Type:\n"
		for contentType, count := range result.Summary.TypeBreakdown {
			text += fmt.Sprintf("  • %s: %d\n", contentType, count)
		}
		text += "\n"
	}

	if len(result.Summary.PageBreakdown) > 0 {
		text += "📄 Result Breakdown by Page:\n"
		for page, count := range result.Summary.PageBreakdown {
			text += fmt.Sprintf("  • Page %d: %d\n", page, count)
		}
		text += "\n"
	}

	// Show matching elements
	if len(result.Elements) > 0 {
		text += fmt.Sprintf("🎯 Matching Elements (showing first %d):\n", minInt(10, len(result.Elements)))
		for i, element := range result.Elements {
			if i >= 10 {
				text += fmt.Sprintf("  ... and %d more matches\n", len(result.Elements)-10)
				break
			}
			text += fmt.Sprintf("  %d. %s on page %d (confidence: %.2f)\n",
				i+1, element.Type, element.PageNumber, element.Confidence)
		}
	}

	return text
}

func (s *Server) formatPDFPageInfoResult(result *pdf.PDFPageInfoResult) string {
	text := fmt.Sprintf("📄 Page Information: %s\n", result.FilePath)
	text += fmt.Sprintf("📖 Total Pages: %d\n\n", len(result.Pages))

	for _, page := range result.Pages {
		text += fmt.Sprintf("Page %d:\n", page.Number)
		text += fmt.Sprintf("  Dimensions: %.1f × %.1f pts\n", page.Width, page.Height)
		if page.Rotation != 0 {
			text += fmt.Sprintf("  Rotation: %d°\n", page.Rotation)
		}
		text += fmt.Sprintf("  Media Box: (%.1f, %.1f) to (%.1f, %.1f)\n",
			page.MediaBox.X, page.MediaBox.Y,
			page.MediaBox.X+page.MediaBox.Width, page.MediaBox.Y+page.MediaBox.Height)
		text += "\n"
	}

	return text
}

func (s *Server) formatPDFMetadataResult(result *pdf.PDFMetadataResult) string {
	text := fmt.Sprintf("📋 Document Metadata: %s\n\n", result.FilePath)

	metadata := result.Metadata

	if metadata.Title != "" {
		text += fmt.Sprintf("📖 Title: %s\n", metadata.Title)
	}
	if metadata.Author != "" {
		text += fmt.Sprintf("👤 Author: %s\n", metadata.Author)
	}
	if metadata.Subject != "" {
		text += fmt.Sprintf("📝 Subject: %s\n", metadata.Subject)
	}
	if metadata.Creator != "" {
		text += fmt.Sprintf("🛠️ Creator: %s\n", metadata.Creator)
	}
	if metadata.Producer != "" {
		text += fmt.Sprintf("🏭 Producer: %s\n", metadata.Producer)
	}
	if metadata.CreationDate != "" {
		text += fmt.Sprintf("📅 Created: %s\n", metadata.CreationDate)
	}
	if metadata.ModificationDate != "" {
		text += fmt.Sprintf("📅 Modified: %s\n", metadata.ModificationDate)
	}
	if len(metadata.Keywords) > 0 {
		text += fmt.Sprintf("🏷️ Keywords: %v\n", metadata.Keywords)
	}
	if metadata.Version != "" {
		text += fmt.Sprintf("📄 PDF Version: %s\n", metadata.Version)
	}
	if metadata.PageLayout != "" {
		text += fmt.Sprintf("📐 Page Layout: %s\n", metadata.PageLayout)
	}
	if metadata.PageMode != "" {
		text += fmt.Sprintf("🖥️ Page Mode: %s\n", metadata.PageMode)
	}
	if metadata.Encrypted {
		text += "🔒 Document is encrypted\n"
	}

	if len(metadata.CustomProperties) > 0 {
		text += "\n🏷️ Custom Properties:\n"
		for key, value := range metadata.CustomProperties {
			text += fmt.Sprintf("  • %s: %s\n", key, value)
		}
	}

	return text
}

// Helper function for minimum of two integers
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
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
