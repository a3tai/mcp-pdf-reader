package mcp

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/a3tai/mcp-pdf-reader/internal/config"
	"github.com/a3tai/mcp-pdf-reader/internal/descriptions"
	"github.com/a3tai/mcp-pdf-reader/internal/intelligence"
	"github.com/a3tai/mcp-pdf-reader/internal/pdf"
	"github.com/a3tai/mcp-pdf-reader/internal/pdf/extraction"
	"github.com/a3tai/mcp-pdf-reader/internal/pdf/stability"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// Server represents the MCP server instance
type Server struct {
	config           *config.Config
	pdfService       *pdf.Service
	mcpServer        *server.MCPServer
	documentAnalyzer *intelligence.DocumentAnalyzer
	stableExtraction *stability.StableExtractionService
}

// convertExtractionConfig converts pdf.ExtractionConfig to pdf.ExtractConfig
func convertExtractionConfig(config pdf.ExtractionConfig) pdf.ExtractConfig {
	return pdf.ExtractConfig{
		ExtractText:        config.ExtractText,
		ExtractImages:      config.ExtractImages,
		ExtractTables:      config.ExtractTables,
		ExtractForms:       config.ExtractForms,
		ExtractAnnotations: config.ExtractAnnotations,
		IncludeCoordinates: config.IncludeCoordinates,
		IncludeFormatting:  config.IncludeFormatting,
		Pages:              config.Pages,
		MinConfidence:      config.MinConfidence,
	}
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
		config:           cfg,
		pdfService:       pdfService,
		mcpServer:        mcpServer,
		documentAnalyzer: intelligence.NewDocumentAnalyzer(),
		stableExtraction: stability.NewStableExtractionService(100 * 1024 * 1024), // 100MB limit
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
		mcp.WithDescription(descriptions.GetToolDescription("pdf_read_file")),
		mcp.WithString("path",
			mcp.Required(),
			mcp.Description("Full path to the PDF file (supports both absolute and relative paths)"),
		),
	)
	s.mcpServer.AddTool(pdfReadFileTool, s.handlePDFReadFile)

	// Register PDF assets file tool
	pdfAssetsFileTool := mcp.NewTool(
		"pdf_assets_file",
		mcp.WithDescription(descriptions.GetToolDescription("pdf_assets_file")),
		mcp.WithString("path",
			mcp.Required(),
			mcp.Description("Full path to the PDF file (supports both absolute and relative paths)"),
		),
	)
	s.mcpServer.AddTool(pdfAssetsFileTool, s.handlePDFAssetsFile)

	// Register PDF validate file tool
	pdfValidateFileTool := mcp.NewTool(
		"pdf_validate_file",
		mcp.WithDescription(descriptions.GetToolDescription("pdf_validate_file")),
		mcp.WithString("path",
			mcp.Required(),
			mcp.Description("Full path to the PDF file (supports both absolute and relative paths)"),
		),
	)
	s.mcpServer.AddTool(pdfValidateFileTool, s.handlePDFValidateFile)

	// Register PDF stats file tool
	pdfStatsFileTool := mcp.NewTool(
		"pdf_stats_file",
		mcp.WithDescription(descriptions.GetToolDescription("pdf_stats_file")),
		mcp.WithString("path",
			mcp.Required(),
			mcp.Description("Full path to the PDF file (supports both absolute and relative paths)"),
		),
	)
	s.mcpServer.AddTool(pdfStatsFileTool, s.handlePDFStatsFile)
}

// registerExtractionTools registers structured extraction tools
func (s *Server) registerExtractionTools() {
	// Register PDF extract structured tool
	pdfExtractStructuredTool := mcp.NewTool(
		"pdf_extract_structured",
		mcp.WithDescription(descriptions.GetToolDescription("pdf_extract_structured")),
		mcp.WithString("path",
			mcp.Required(),
			mcp.Description("Full path to the PDF file (supports both absolute and relative paths)"),
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
		mcp.WithDescription(descriptions.GetToolDescription("pdf_extract_tables")),
		mcp.WithString("path",
			mcp.Required(),
			mcp.Description("Full path to the PDF file (supports both absolute and relative paths)"),
		),
		mcp.WithString("config",
			mcp.Description("JSON string with extraction configuration options"),
		),
	)
	s.mcpServer.AddTool(pdfExtractTablesTool, s.handlePDFExtractTables)

	// Register PDF extract forms tool
	pdfExtractFormsTool := mcp.NewTool(
		"pdf_extract_forms",
		mcp.WithDescription(descriptions.GetToolDescription("pdf_extract_forms")),
		mcp.WithString("path",
			mcp.Required(),
			mcp.Description("Full path to the PDF file (supports both absolute and relative paths)"),
		),
		mcp.WithString("config",
			mcp.Description("JSON string with extraction configuration options"),
		),
	)
	s.mcpServer.AddTool(pdfExtractFormsTool, s.handlePDFExtractForms)

	// Register PDF extract semantic tool
	pdfExtractSemanticTool := mcp.NewTool(
		"pdf_extract_semantic",
		mcp.WithDescription(descriptions.GetToolDescription("pdf_extract_semantic")),
		mcp.WithString("path",
			mcp.Required(),
			mcp.Description("Full path to the PDF file (supports both absolute and relative paths)"),
		),
		mcp.WithString("config",
			mcp.Description("JSON string with extraction configuration options"),
		),
	)
	s.mcpServer.AddTool(pdfExtractSemanticTool, s.handlePDFExtractSemantic)

	// Register PDF extract complete tool
	pdfExtractCompleteTool := mcp.NewTool(
		"pdf_extract_complete",
		mcp.WithDescription(descriptions.GetToolDescription("pdf_extract_complete")),
		mcp.WithString("path",
			mcp.Required(),
			mcp.Description("Full path to the PDF file (supports both absolute and relative paths)"),
		),
		mcp.WithString("config",
			mcp.Description("JSON string with extraction configuration options"),
		),
	)
	s.mcpServer.AddTool(pdfExtractCompleteTool, s.handlePDFExtractComplete)

	// Register PDF query content tool
	pdfQueryContentTool := mcp.NewTool(
		"pdf_query_content",
		mcp.WithDescription(descriptions.GetToolDescription("pdf_query_content")),
		mcp.WithString("path",
			mcp.Required(),
			mcp.Description("Full path to the PDF file (supports both absolute and relative paths)"),
		),
		mcp.WithString("query",
			mcp.Required(),
			mcp.Description("JSON string with query criteria for filtering content"),
		),
	)
	s.mcpServer.AddTool(pdfQueryContentTool, s.handlePDFQueryContent)

	// Register PDF analyze document tool
	pdfAnalyzeDocumentTool := mcp.NewTool(
		"pdf_analyze_document",
		mcp.WithDescription(descriptions.GetToolDescription("pdf_analyze_document")),
		mcp.WithString("path",
			mcp.Required(),
			mcp.Description("Full path to the PDF file (supports both absolute and relative paths)"),
		),
		mcp.WithString("config",
			mcp.Description("JSON string with analysis configuration options"),
		),
	)
	s.mcpServer.AddTool(pdfAnalyzeDocumentTool, s.handlePDFAnalyzeDocument)
}

// registerUtilityTools registers utility and information tools
func (s *Server) registerUtilityTools() {
	// Register PDF search directory tool
	pdfSearchDirectoryTool := mcp.NewTool(
		"pdf_search_directory",
		mcp.WithDescription(descriptions.GetToolDescription("pdf_search_directory")),
		mcp.WithString("directory",
			mcp.Description("Directory path to search (uses current directory if empty, supports relative paths)"),
		),
		mcp.WithString("query",
			mcp.Description("Optional search query for fuzzy matching"),
		),
	)
	s.mcpServer.AddTool(pdfSearchDirectoryTool, s.handlePDFSearchDirectory)

	// Register PDF stats directory tool
	pdfStatsDirectoryTool := mcp.NewTool(
		"pdf_stats_directory",
		mcp.WithDescription(descriptions.GetToolDescription("pdf_stats_directory")),
		mcp.WithString("directory",
			mcp.Description("Directory path to analyze (uses current directory if empty, supports relative paths)"),
		),
	)
	s.mcpServer.AddTool(pdfStatsDirectoryTool, s.handlePDFStatsDirectory)

	// Register PDF server info tool
	pdfServerInfoTool := mcp.NewTool(
		"pdf_server_info",
		mcp.WithDescription(descriptions.GetToolDescription("pdf_server_info")),
	)
	s.mcpServer.AddTool(pdfServerInfoTool, s.handlePDFServerInfo)

	// Register PDF get page info tool
	pdfGetPageInfoTool := mcp.NewTool(
		"pdf_get_page_info",
		mcp.WithDescription(descriptions.GetToolDescription("pdf_get_page_info")),
		mcp.WithString("path",
			mcp.Required(),
			mcp.Description("Full path to the PDF file (supports both absolute and relative paths)"),
		),
	)
	s.mcpServer.AddTool(pdfGetPageInfoTool, s.handlePDFGetPageInfo)

	// Register PDF get metadata tool
	pdfGetMetadataTool := mcp.NewTool(
		"pdf_get_metadata",
		mcp.WithDescription(descriptions.GetToolDescription("pdf_get_metadata")),
		mcp.WithString("path",
			mcp.Required(),
			mcp.Description("Full path to the PDF file (supports both absolute and relative paths)"),
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
		responseText += "\n### Recommendation\n\nThis PDF appears to contain scanned images with little or no " +
			"extractable text. Consider using `pdf_assets_file` to extract the images.\n"
	case "mixed":
		responseText += "\n### Info\n\nThis PDF contains both text and images. You may want to use " +
			"`pdf_assets_file` to extract the images as well.\n"
	case "no_content":
		responseText += "\n### Warning\n\nThis PDF appears to have no readable content or images.\n"
	}

	responseText += "\n## Content\n\n"
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
	result, err := s.pdfService.PDFServerInfo(ctx, req, s.config.ServerName, s.config.Version, s.config.PDFDirectory)
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

	// Convert PDFExtractStructuredRequest to PDFExtractRequest
	extractReq := pdf.PDFExtractRequest{
		Path:   req.Path,
		Mode:   req.Mode,
		Config: convertExtractionConfig(req.Config),
		Query:  req.Query,
	}
	result, err := s.stableExtraction.ExtractStructured(extractReq)
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

func (s *Server) handlePDFExtractForms(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return s.handleExtractionRequest(request,
		func(path string, config pdf.ExtractionConfig) (*pdf.PDFExtractResult, error) {
			// Convert ExtractionConfig to ExtractConfig for PDFExtractRequest
			extractConfig := pdf.ExtractConfig{
				ExtractText:        config.ExtractText,
				ExtractImages:      config.ExtractImages,
				ExtractTables:      config.ExtractTables,
				ExtractForms:       config.ExtractForms,
				ExtractAnnotations: config.ExtractAnnotations,
				IncludeCoordinates: config.IncludeCoordinates,
				IncludeFormatting:  config.IncludeFormatting,
				Pages:              config.Pages,
				MinConfidence:      config.MinConfidence,
			}
			return s.pdfService.ExtractForms(pdf.PDFExtractRequest{Path: path, Config: extractConfig})
		}, pdf.ExtractionConfig{
			ExtractForms:       true,
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

	// Convert PDFExtractCompleteRequest to PDFExtractRequest
	extractReq := pdf.PDFExtractRequest{
		Path:   req.Path,
		Mode:   "complete",
		Config: convertExtractionConfig(req.Config),
	}
	result, err := s.stableExtraction.ExtractComplete(extractReq)
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

	// Convert PDFQueryContentRequest to PDFQueryRequest
	queryReq := pdf.PDFQueryRequest{
		Path:  req.Path,
		Query: req.Query,
	}
	result, err := s.stableExtraction.QueryContent(queryReq)
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

	result, err := s.stableExtraction.GetPageInfo(path)
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

	result, err := s.stableExtraction.GetMetadata(path)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Convert DocumentMetadata to PDFMetadataResult for formatting
	metadataResult := &pdf.PDFMetadataResult{
		FilePath: path,
		Metadata: *result,
	}
	responseText := s.formatPDFMetadataResult(metadataResult)
	return mcp.NewToolResultText(responseText), nil
}

func (s *Server) handlePDFAnalyzeDocument(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	path, err := request.RequireString("path")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Get optional config
	args := request.GetArguments()
	config := pdf.ExtractionConfig{}
	if configStr, ok := args["config"].(string); ok && configStr != "" {
		// For now, just use default config
		// TODO: Parse JSON config string when needed
		config = pdf.ExtractionConfig{}
	}

	// Extract structured content from the PDF first
	extractReq := pdf.PDFExtractStructuredRequest{
		Path:   path,
		Mode:   string(extraction.ModeComplete),
		Config: config,
	}

	// Convert PDFExtractStructuredRequest to PDFExtractRequest
	convertedReq := pdf.PDFExtractRequest{
		Path:   extractReq.Path,
		Mode:   extractReq.Mode,
		Config: convertExtractionConfig(extractReq.Config),
		Query:  extractReq.Query,
	}
	extractResult, err := s.stableExtraction.ExtractStructured(convertedReq)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to extract content for analysis: %v\n\nTroubleshooting:\n• Check if the PDF file exists and is readable\n• Verify the PDF is not corrupted or password-protected\n• Try using pdf_validate_file to check document integrity", err)), nil
	}

	// Convert extraction result to content elements
	elements, err := s.convertToContentElements(extractResult)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to convert extracted content: %v", err)), nil
	}

	// Add debugging information about extracted content
	fmt.Fprintf(os.Stderr, "[AnalyzeDocument] Extracted %d elements from %s\n", len(elements), path)
	if len(elements) == 0 {
		fmt.Fprintf(os.Stderr, "[AnalyzeDocument] WARNING: No content elements extracted - document may be image-based or corrupted\n")
	}

	// Perform document analysis (now handles empty content gracefully)
	analysis, err := s.documentAnalyzer.Analyze(elements)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Document analysis failed: %v", err)), nil
	}

	// Format and return the analysis results
	responseText := s.formatDocumentAnalysisResult(analysis)
	return mcp.NewToolResultText(responseText), nil
}

// convertToContentElements converts PDF extraction result to content elements for analysis
func (s *Server) convertToContentElements(extractResult *pdf.PDFExtractResult) ([]extraction.ContentElement, error) {
	var elements []extraction.ContentElement

	// Convert each element from the extraction result
	for _, elem := range extractResult.Elements {
		// Convert PDF Rectangle to extraction BoundingBox
		boundingBox := extraction.BoundingBox{
			LowerLeft: extraction.Coordinate{
				X: elem.BoundingBox.X,
				Y: elem.BoundingBox.Y,
			},
			UpperRight: extraction.Coordinate{
				X: elem.BoundingBox.X + elem.BoundingBox.Width,
				Y: elem.BoundingBox.Y + elem.BoundingBox.Height,
			},
			Width:  elem.BoundingBox.Width,
			Height: elem.BoundingBox.Height,
		}

		contentElement := extraction.ContentElement{
			ID:          elem.ID,
			Type:        extraction.ContentType(elem.Type),
			PageNumber:  elem.PageNumber,
			BoundingBox: boundingBox,
			Content:     elem.Content,
			Properties:  elem.Properties,
			Confidence:  elem.Confidence,
		}
		elements = append(elements, contentElement)
	}

	return elements, nil
}

// Formatting methods
func (s *Server) formatDocumentAnalysisResult(analysis *intelligence.DocumentAnalysis) string {
	text := "# Document Analysis Report\n\n"

	// Document classification
	text += fmt.Sprintf("**Document Type:** %s\n", analysis.Type)
	text += fmt.Sprintf("**Analysis Version:** %s\n", analysis.Metadata.AnalysisVersion)
	text += fmt.Sprintf("**Processing Time:** %v\n\n", analysis.Metadata.ProcessingTime)

	// Content statistics
	stats := analysis.Statistics
	text += "## Content Statistics\n\n"
	text += fmt.Sprintf("- **Pages:** %d\n", stats.PageCount)
	text += fmt.Sprintf("- **Words:** %d\n", stats.WordCount)
	text += fmt.Sprintf("- **Characters:** %d\n", stats.CharacterCount)
	text += fmt.Sprintf("- **Paragraphs:** %d\n", stats.ParagraphCount)
	text += fmt.Sprintf("- **Sentences:** %d\n", stats.SentenceCount)
	text += fmt.Sprintf("- **Reading Time:** %.1f minutes\n", stats.ReadingTime)

	if stats.ImageCount > 0 {
		text += fmt.Sprintf("- **Images:** %d\n", stats.ImageCount)
	}
	if stats.TableCount > 0 {
		text += fmt.Sprintf("- **Tables:** %d\n", stats.TableCount)
	}
	if stats.FormFieldCount > 0 {
		text += fmt.Sprintf("- **Form Fields:** %d\n", stats.FormFieldCount)
	}
	if stats.HeaderCount > 0 {
		text += fmt.Sprintf("- **Headers:** %d\n", stats.HeaderCount)
	}
	if stats.ListCount > 0 {
		text += fmt.Sprintf("- **Lists:** %d\n", stats.ListCount)
	}

	// Content density
	if len(stats.ContentDensity) > 0 {
		text += "\n### Content Density\n"
		for metric, value := range stats.ContentDensity {
			text += fmt.Sprintf("- **%s:** %.1f\n", metric, value)
		}
	}

	// Document structure
	if len(analysis.Sections) > 0 {
		text += "\n## Document Structure\n\n"
		for i, section := range analysis.Sections {
			text += fmt.Sprintf("### %d. %s\n", i+1, section.Title)
			if section.Type != "" {
				text += fmt.Sprintf("**Type:** %s  \n", section.Type)
			}
			text += fmt.Sprintf("**Level:** %d  \n", section.Level)
			if section.PageStart == section.PageEnd {
				text += fmt.Sprintf("**Page:** %d  \n", section.PageStart)
			} else {
				text += fmt.Sprintf("**Pages:** %d-%d  \n", section.PageStart, section.PageEnd)
			}

			// Show content preview
			if section.Content != "" {
				preview := section.Content
				if len(preview) > 200 {
					preview = preview[:200] + "..."
				}
				text += fmt.Sprintf("**Content:** %s\n", preview)
			}

			// Show subsections
			if len(section.Subsections) > 0 {
				text += fmt.Sprintf("**Subsections:** %d\n", len(section.Subsections))
			}
			text += "\n"
		}
	}

	// Quality metrics
	quality := analysis.Quality
	text += "## Quality Assessment\n\n"
	text += fmt.Sprintf("- **Overall Score:** %.2f/1.0\n", quality.OverallScore)
	text += fmt.Sprintf("- **Readability:** %.2f/1.0\n", quality.ReadabilityScore)
	text += fmt.Sprintf("- **Completeness:** %.2f/1.0\n", quality.CompletenessScore)
	text += fmt.Sprintf("- **Consistency:** %.2f/1.0\n", quality.ConsistencyScore)
	text += fmt.Sprintf("- **Structure:** %.2f/1.0\n", quality.StructureScore)
	text += fmt.Sprintf("- **Accessibility:** %.2f/1.0\n", quality.AccessibilityScore)

	// Quality issues
	if len(quality.IssuesFound) > 0 {
		text += "\n### Quality Issues\n"
		for _, issue := range quality.IssuesFound {
			text += fmt.Sprintf("- **%s** (%s): %s\n", issue.Type, issue.Severity, issue.Description)
			if issue.Suggestion != "" {
				text += fmt.Sprintf("  *Suggestion: %s*\n", issue.Suggestion)
			}
		}
	}

	// Positive indicators
	if len(quality.PositiveIndicators) > 0 {
		text += "\n### Positive Aspects\n"
		for _, indicator := range quality.PositiveIndicators {
			text += fmt.Sprintf("- %s\n", indicator)
		}
	}

	// Suggestions for improvement
	if len(analysis.Suggestions) > 0 {
		text += "\n## Suggestions for Improvement\n\n"
		for i, suggestion := range analysis.Suggestions {
			text += fmt.Sprintf("%d. %s\n", i+1, suggestion)
		}
	}

	// Processing metadata
	if len(analysis.Metadata.ComponentsUsed) > 0 {
		text += "\n## Analysis Details\n\n"
		text += fmt.Sprintf("**Components Used:** %s\n", strings.Join(analysis.Metadata.ComponentsUsed, ", "))
	}

	if len(analysis.Metadata.Warnings) > 0 {
		text += "\n### Warnings\n"
		for _, warning := range analysis.Metadata.Warnings {
			text += fmt.Sprintf("- %s\n", warning)
		}
	}

	if len(analysis.Metadata.Errors) > 0 {
		text += "\n### Errors\n"
		for _, error := range analysis.Metadata.Errors {
			text += fmt.Sprintf("- %s\n", error)
		}
	}

	return text
}

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
	text := fmt.Sprintf("# %s v%s - Server Information\n\n", result.ServerName, result.Version)
	text += fmt.Sprintf("**Default Directory:** %s\n", result.DefaultDirectory)
	text += fmt.Sprintf("**Max File Size:** %d MB\n\n", result.MaxFileSize/(1024*1024))

	// Directory contents
	if len(result.DirectoryContents) > 0 {
		text += fmt.Sprintf("## Directory Contents (%d PDF files found)\n\n", len(result.DirectoryContents))
		for i, file := range result.DirectoryContents {
			if i >= 10 { // Limit to first 10 files for readability
				text += fmt.Sprintf("*... and %d more files*\n", len(result.DirectoryContents)-10)
				break
			}
			text += fmt.Sprintf("- %s (%d bytes)\n", file.Name, file.Size)
		}
		text += "\n"
	} else {
		text += "## Directory Contents\n\n*No PDF files found in default directory*\n\n"
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
	text := fmt.Sprintf("# PDF Extraction Results\n\n**File:** %s\n", result.FilePath)
	text += fmt.Sprintf("**Mode:** %s\n", result.Mode)
	text += fmt.Sprintf("**Pages:** %d (processed: %v)\n", result.TotalPages, result.ProcessedPages)
	text += fmt.Sprintf("**Quality:** %s\n", result.Summary.Quality)
	text += fmt.Sprintf("**Total Elements:** %d\n\n", result.Summary.TotalElements)

	// Content type breakdown
	text += "## Content Types Found\n\n"
	for contentType, count := range result.Summary.ContentTypes {
		text += fmt.Sprintf("- **%s:** %d\n", contentType, count)
	}
	text += "\n"

	// Tables if found
	if len(result.Tables) > 0 {
		text += fmt.Sprintf("## Tables Found: %d\n\n", len(result.Tables))
		for i, table := range result.Tables {
			text += fmt.Sprintf("### Table %d\n\n", i+1)
			text += fmt.Sprintf("- **Dimensions:** %d rows × %d columns (%d cells)\n",
				len(table.Rows), len(table.Columns), table.CellCount)
			if table.HasHeaders {
				text += "- **Headers:** Yes\n"
			}
			text += fmt.Sprintf("- **Confidence:** %.2f\n\n", table.Confidence)
		}
	}

	// Page breakdown
	if len(result.Summary.PageBreakdown) > 0 {
		text += "## Page Breakdown\n\n"
		for _, page := range result.Summary.PageBreakdown {
			text += fmt.Sprintf("- **Page %d:** %d elements\n", page.Page, page.Elements)
		}
		text += "\n"
	}

	// Suggestions
	if len(result.Summary.Suggestions) > 0 {
		text += "## Suggestions\n\n"
		for _, suggestion := range result.Summary.Suggestions {
			text += fmt.Sprintf("- %s\n", suggestion)
		}
		text += "\n"
	}

	// Warnings and errors
	if len(result.Warnings) > 0 {
		text += "## Warnings\n\n"
		for _, warning := range result.Warnings {
			text += fmt.Sprintf("- %s\n", warning)
		}
		text += "\n"
	}

	if len(result.Errors) > 0 {
		text += "## Errors\n\n"
		for _, error := range result.Errors {
			text += fmt.Sprintf("- %s\n", error)
		}
		text += "\n"
	}

	// Show first few elements as examples
	if len(result.Elements) > 0 {
		text += fmt.Sprintf("## Content Elements (showing first %d)\n\n", minInt(5, len(result.Elements)))
		for i, element := range result.Elements {
			if i >= 5 {
				text += fmt.Sprintf("*... and %d more elements*\n", len(result.Elements)-5)
				break
			}
			text += fmt.Sprintf("### Element %d\n\n", i+1)
			text += fmt.Sprintf("- **Type:** %s\n", element.Type)
			text += fmt.Sprintf("- **Page:** %d\n", element.PageNumber)
			text += fmt.Sprintf("- **Confidence:** %.2f\n", element.Confidence)

			// Show content preview for text elements
			if element.Type == "text" {
				if contentStr, ok := element.Content.(string); ok {
					preview := contentStr
					if len(preview) > 100 {
						preview = preview[:100] + "..."
					}
					text += fmt.Sprintf("- **Content:** %s\n", preview)
				}
			}
			text += "\n"
		}
	}

	return text
}

func (s *Server) formatPDFQueryResult(result *pdf.PDFQueryResult) string {
	text := fmt.Sprintf("# Query Results\n\n**File:** %s\n", result.FilePath)
	text += fmt.Sprintf("**Matches Found:** %d\n", result.MatchCount)
	text += fmt.Sprintf("**Average Confidence:** %.2f\n\n", result.Summary.Confidence)

	// Query details
	text += "## Query Details\n\n"
	if len(result.Query.ContentTypes) > 0 {
		text += fmt.Sprintf("**Content Types:** %v\n", result.Query.ContentTypes)
	}
	if len(result.Query.Pages) > 0 {
		text += fmt.Sprintf("**Pages:** %v\n", result.Query.Pages)
	}
	if result.Query.TextQuery != "" {
		text += fmt.Sprintf("**Text Query:** %s\n", result.Query.TextQuery)
	}
	if result.Query.MinConfidence > 0 {
		text += fmt.Sprintf("**Min Confidence:** %.2f\n", result.Query.MinConfidence)
	}
	text += "\n"

	// Result breakdown
	// Matching elements breakdown
	if len(result.Summary.TypeBreakdown) > 0 {
		text += "## Match Breakdown by Type\n\n"
		for contentType, count := range result.Summary.TypeBreakdown {
			text += fmt.Sprintf("- **%s:** %d matches\n", contentType, count)
		}
		text += "\n"
	}

	if len(result.Summary.PageBreakdown) > 0 {
		text += "## Match Breakdown by Page\n\n"
		for page, count := range result.Summary.PageBreakdown {
			text += fmt.Sprintf("- **Page %d:** %d matches\n", page, count)
		}
		text += "\n"
	}

	// Show matching elements
	// Show first few elements as examples
	if len(result.Elements) > 0 {
		text += fmt.Sprintf("## Content Elements (showing first %d)\n\n", minInt(5, len(result.Elements)))
		for i, element := range result.Elements {
			if i >= 5 {
				text += fmt.Sprintf("*... and %d more elements*\n", len(result.Elements)-5)
				break
			}
			text += fmt.Sprintf("### Match %d\n\n", i+1)
			text += fmt.Sprintf("- **Type:** %s\n", element.Type)
			text += fmt.Sprintf("- **Page:** %d\n", element.PageNumber)
			text += fmt.Sprintf("- **Confidence:** %.2f\n", element.Confidence)

			// Show content preview for matches
			if element.Type == "text" {
				if contentStr, ok := element.Content.(string); ok {
					preview := contentStr
					if len(preview) > 150 {
						preview = preview[:150] + "..."
					}
					text += fmt.Sprintf("- **Content:** %s\n", preview)
				}
			}
			text += "\n"
		}
	}

	return text
}

func (s *Server) formatPDFPageInfoResult(result *pdf.PDFPageInfoResult) string {
	text := fmt.Sprintf("# Page Information\n\n**File:** %s\n", result.FilePath)
	text += fmt.Sprintf("**Total Pages:** %d\n\n", len(result.Pages))

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
	text := fmt.Sprintf("# Document Metadata\n\n**File:** %s\n\n", result.FilePath)

	metadata := result.Metadata

	if metadata.Title != "" {
		text += fmt.Sprintf("**Title:** %s\n", metadata.Title)
	}
	if metadata.Author != "" {
		text += fmt.Sprintf("**Author:** %s\n", metadata.Author)
	}
	if metadata.Subject != "" {
		text += fmt.Sprintf("**Subject:** %s\n", metadata.Subject)
	}
	if metadata.Creator != "" {
		text += fmt.Sprintf("**Creator:** %s\n", metadata.Creator)
	}
	if metadata.Producer != "" {
		text += fmt.Sprintf("**Producer:** %s\n", metadata.Producer)
	}
	if metadata.CreationDate != "" {
		text += fmt.Sprintf("**Created:** %s\n", metadata.CreationDate)
	}
	if metadata.ModificationDate != "" {
		text += fmt.Sprintf("**Modified:** %s\n", metadata.ModificationDate)
	}
	if len(metadata.Keywords) > 0 {
		text += fmt.Sprintf("**Keywords:** %v\n", metadata.Keywords)
	}
	if metadata.Version != "" {
		text += fmt.Sprintf("**PDF Version:** %s\n", metadata.Version)
	}
	if metadata.PageLayout != "" {
		text += fmt.Sprintf("**Page Layout:** %s\n", metadata.PageLayout)
	}
	if metadata.PageMode != "" {
		text += fmt.Sprintf("**Page Mode:** %s\n", metadata.PageMode)
	}
	if metadata.Encrypted {
		text += "\n**Security:** Document is encrypted\n"
	}

	if len(metadata.CustomProperties) > 0 {
		text += "\n## Custom Properties\n\n"
		for key, value := range metadata.CustomProperties {
			text += fmt.Sprintf("- **%s:** %v\n", key, value)
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
