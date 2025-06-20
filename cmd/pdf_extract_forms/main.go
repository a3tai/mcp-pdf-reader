package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/a3tai/mcp-pdf-reader/internal/pdf/extraction"
)

var (
	diagnosticMode = flag.Bool("diagnostic", false, "Enable diagnostic output for form detection")
	outputFormat   = flag.String("format", "text", "Output format: text, json")
	verbose        = flag.Bool("verbose", false, "Enable verbose output")
	help           = flag.Bool("help", false, "Show help message")
)

func main() {
	flag.Parse()

	if *help {
		printHelp()
		return
	}

	if flag.NArg() == 0 {
		fmt.Fprintf(os.Stderr, "Error: PDF file path required\n\n")
		printUsage()
		os.Exit(1)
	}

	pdfPath := flag.Arg(0)
	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Error: File not found: %s\n", pdfPath)
		os.Exit(1)
	}

	// Set up logging for diagnostic mode
	if *diagnosticMode {
		log.SetLevel(DebugLevel)
		enableFormDiagnostics()
	}

	// Extract forms with enhanced debugging
	result, err := extractFormsWithDiagnostics(pdfPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error extracting forms: %v\n", err)
		os.Exit(1)
	}

	// Output results
	if err := outputResults(result); err != nil {
		fmt.Fprintf(os.Stderr, "Error outputting results: %v\n", err)
		os.Exit(1)
	}
}

func printHelp() {
	fmt.Println("PDF Extract Forms - Debug and extract form fields from PDF documents")
	fmt.Println()
	fmt.Println("This tool investigates and fixes form field detection issues in PDF documents,")
	fmt.Println("specifically designed to handle tax documents, W-2s, 1099s, and other complex forms.")
	fmt.Println()
	printUsage()
	fmt.Println()
	fmt.Println("OPTIONS:")
	fmt.Println("  -diagnostic    Enable comprehensive diagnostic output for form detection")
	fmt.Println("  -format        Output format: text (default), json")
	fmt.Println("  -verbose       Enable verbose output")
	fmt.Println("  -help          Show this help message")
	fmt.Println()
	fmt.Println("DIAGNOSTIC MODE:")
	fmt.Println("  When -diagnostic is enabled, the tool provides detailed debugging information:")
	fmt.Println("  • AcroForm dictionary analysis")
	fmt.Println("  • XFA form detection (XML Forms Architecture)")
	fmt.Println("  • Page annotation scanning for widget annotations")
	fmt.Println("  • Indirect reference resolution")
	fmt.Println("  • Recursive field hierarchy search")
	fmt.Println("  • Form field type inheritance detection")
	fmt.Println()
	fmt.Println("EXAMPLES:")
	fmt.Println("  pdf_extract_forms document.pdf")
	fmt.Println("  pdf_extract_forms -diagnostic tax-form.pdf")
	fmt.Println("  pdf_extract_forms -format json -verbose forms/w2.pdf")
	fmt.Println()
	fmt.Println("SUPPORTED FORM TYPES:")
	fmt.Println("  • AcroForms (standard PDF interactive forms)")
	fmt.Println("  • XFA Forms (XML Forms Architecture - common in tax documents)")
	fmt.Println("  • Widget Annotations (form fields stored as page annotations)")
	fmt.Println("  • Complex form hierarchies with inherited field properties")
}

func printUsage() {
	fmt.Println("USAGE:")
	fmt.Println("  pdf_extract_forms [OPTIONS] <pdf_file>")
}

// FormExtractionResult represents the complete result of form extraction
type FormExtractionResult struct {
	FilePath       string                 `json:"file_path"`
	Success        bool                   `json:"success"`
	FieldCount     int                    `json:"field_count"`
	Fields         []extraction.FormField `json:"fields"`
	Diagnostics    *DiagnosticInfo        `json:"diagnostics,omitempty"`
	Error          string                 `json:"error,omitempty"`
	ExtractionTime string                 `json:"extraction_time,omitempty"`
}

// DiagnosticInfo contains detailed diagnostic information
type DiagnosticInfo struct {
	PDFVersion       string   `json:"pdf_version"`
	PageCount        int      `json:"page_count"`
	Encrypted        bool     `json:"encrypted"`
	CatalogKeys      []string `json:"catalog_keys"`
	HasAcroForm      bool     `json:"has_acro_form"`
	HasXFA           bool     `json:"has_xfa"`
	HasAnnotations   bool     `json:"has_annotations"`
	MethodsAttempted []string `json:"methods_attempted"`
	Warnings         []string `json:"warnings"`
	DebugMessages    []string `json:"debug_messages"`
}

func extractFormsWithDiagnostics(pdfPath string) (*FormExtractionResult, error) {
	absPath, err := filepath.Abs(pdfPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	result := &FormExtractionResult{
		FilePath: absPath,
		Success:  false,
	}

	if *diagnosticMode {
		result.Diagnostics = &DiagnosticInfo{
			MethodsAttempted: []string{},
			Warnings:         []string{},
			DebugMessages:    []string{},
		}

		if *verbose {
			fmt.Printf("🔍 Analyzing PDF: %s\n", absPath)
			fmt.Println()
		}
	}

	// Create enhanced form extractor with diagnostic mode
	extractor := extraction.NewPDFCPUFormExtractor(*diagnosticMode || *verbose)

	// Perform comprehensive form extraction
	forms, err := extractor.ExtractFormsFromFile(absPath)
	if err != nil {
		result.Error = err.Error()
		return result, nil // Don't fail, return error in result
	}

	result.Success = true
	result.FieldCount = len(forms)
	result.Fields = forms

	if *diagnosticMode && *verbose {
		fmt.Printf("✅ Extraction completed successfully\n")
		fmt.Printf("📊 Found %d form fields\n", len(forms))
		fmt.Println()
	}

	return result, nil
}

func outputResults(result *FormExtractionResult) error {
	switch *outputFormat {
	case "json":
		return outputJSON(result)
	case "text":
		return outputText(result)
	default:
		return fmt.Errorf("unsupported output format: %s", *outputFormat)
	}
}

func outputJSON(result *FormExtractionResult) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(result)
}

func outputText(result *FormExtractionResult) error {
	if !result.Success {
		fmt.Printf("❌ Form extraction failed: %s\n", result.Error)
		return nil
	}

	if result.FieldCount == 0 {
		fmt.Println("⚠️  No form fields detected in the PDF")
		if *diagnosticMode {
			fmt.Println()
			fmt.Println("DIAGNOSTIC SUGGESTIONS:")
			fmt.Println("• This PDF may not contain interactive forms")
			fmt.Println("• Forms might use XFA (XML Forms Architecture) - not fully supported yet")
			fmt.Println("• Forms might be stored as non-standard annotations")
			fmt.Println("• The PDF might be scanned/image-based with visual form elements only")
			fmt.Println()
			fmt.Println("TRY:")
			fmt.Println("• Use pdf_analyze_document to understand the document structure")
			fmt.Println("• Check if the PDF has fillable fields in a PDF viewer")
			fmt.Println("• Consider using OCR for scanned forms")
		}
		return nil
	}

	fmt.Printf("✅ Successfully extracted %d form fields\n", result.FieldCount)
	fmt.Println()

	for i, field := range result.Fields {
		fmt.Printf("[%d] %s\n", i+1, field.Name)
		fmt.Printf("    Type: %s\n", field.Type)

		if field.Value != nil {
			fmt.Printf("    Value: %v\n", field.Value)
		}

		if field.DefaultValue != nil {
			fmt.Printf("    Default: %v\n", field.DefaultValue)
		}

		if field.Page > 0 {
			fmt.Printf("    Page: %d\n", field.Page)
		}

		if field.Bounds != nil {
			fmt.Printf("    Position: (%.1f, %.1f) to (%.1f, %.1f)\n",
				field.Bounds.LowerLeft.X, field.Bounds.LowerLeft.Y,
				field.Bounds.UpperRight.X, field.Bounds.UpperRight.Y)
		}

		properties := []string{}
		if field.Required {
			properties = append(properties, "Required")
		}
		if field.ReadOnly {
			properties = append(properties, "ReadOnly")
		}
		if len(properties) > 0 {
			fmt.Printf("    Properties: %v\n", properties)
		}

		if len(field.Options) > 0 {
			fmt.Printf("    Options: %v\n", field.Options)
		}

		if field.Validation != nil && field.Validation.MaxLength > 0 {
			fmt.Printf("    Max Length: %d\n", field.Validation.MaxLength)
		}

		fmt.Println()
	}

	if *diagnosticMode {
		printDiagnosticSummary(result)
	}

	return nil
}

func printDiagnosticSummary(result *FormExtractionResult) {
	if result.Diagnostics == nil {
		return
	}

	fmt.Println("📋 DIAGNOSTIC SUMMARY")
	fmt.Println("=====================")

	if result.Diagnostics.PDFVersion != "" {
		fmt.Printf("PDF Version: %s\n", result.Diagnostics.PDFVersion)
	}

	if result.Diagnostics.PageCount > 0 {
		fmt.Printf("Page Count: %d\n", result.Diagnostics.PageCount)
	}

	fmt.Printf("Encrypted: %t\n", result.Diagnostics.Encrypted)
	fmt.Printf("Has AcroForm: %t\n", result.Diagnostics.HasAcroForm)
	fmt.Printf("Has XFA: %t\n", result.Diagnostics.HasXFA)
	fmt.Printf("Has Annotations: %t\n", result.Diagnostics.HasAnnotations)

	if len(result.Diagnostics.MethodsAttempted) > 0 {
		fmt.Printf("Methods Attempted: %v\n", result.Diagnostics.MethodsAttempted)
	}

	if len(result.Diagnostics.Warnings) > 0 {
		fmt.Println("\nWarnings:")
		for _, warning := range result.Diagnostics.Warnings {
			fmt.Printf("  ⚠️  %s\n", warning)
		}
	}

	fmt.Println()
	fmt.Println("For more detailed analysis, consider using:")
	fmt.Println("  • pdf_server_info for server capabilities")
	fmt.Println("  • pdf_get_metadata for document properties")
	fmt.Println("  • pdf_analyze_document for comprehensive analysis")
	fmt.Println()
}

func enableFormDiagnostics() {
	if *verbose {
		fmt.Println("🔧 Form diagnostics enabled")
		fmt.Println("   • Enhanced error handling for indirect references")
		fmt.Println("   • XFA form detection (XML Forms Architecture)")
		fmt.Println("   • Recursive field search with inheritance")
		fmt.Println("   • Page annotation scanning for widget fields")
		fmt.Println()
	}
}

// Custom log level for debugging
type LogLevel int

const (
	InfoLevel LogLevel = iota
	DebugLevel
)

var currentLogLevel = InfoLevel

type Logger struct{}

func (l *Logger) SetLevel(level LogLevel) {
	currentLogLevel = level
}

func (l *Logger) Debug(format string, args ...interface{}) {
	if currentLogLevel >= DebugLevel {
		fmt.Printf("[DEBUG] "+format+"\n", args...)
	}
}

func (l *Logger) Info(format string, args ...interface{}) {
	fmt.Printf("[INFO] "+format+"\n", args...)
}

var log = &Logger{}

func init() {
	// Custom flag usage
	flag.Usage = func() {
		printHelp()
	}
}
