package descriptions

// Comprehensive tool descriptions with practical examples and use cases

const (
	// Basic Tools
	PDFReadFileDescription = `Extract readable text from PDF documents quickly and efficiently.

**When to use:** Need to get the actual text content from a PDF document for analysis, search, or conversion.

**Why it's useful:** Automatically handles different PDF text encodings and extracts clean, searchable text while identifying document structure.

**Examples:**
• Extract text from a research paper: "Get all text from research-paper.pdf to analyze methodology"
• Process invoice data: "Read invoice-2024-001.pdf to extract line items and totals"
• Convert PDF to text: "Get clean text from manual.pdf for documentation system"

**Common workflows:**
1. Research & Analysis: Extract text → Analyze content → Generate summaries
2. Document Processing: Read PDF → Extract data → Feed to downstream systems
3. Content Migration: Read PDF → Clean text → Convert to other formats

**Best practices:** Always validate the file first, check content_type in response to understand document structure.`

	PDFAssetsFileDescription = `Extract images, graphics, and visual assets from PDF documents.

**When to use:** Need to get images from PDFs, especially for scanned documents, presentations, or image-heavy reports.

**Why it's useful:** Recovers high-quality images in their original formats (JPEG, PNG, TIFF) that would be lost in text-only extraction.

**Examples:**
• Extract charts from reports: "Get all graphs and charts from quarterly-report.pdf for presentation"
• Recover scanned images: "Extract images from scanned-invoice.pdf when text extraction fails"
• Asset extraction: "Get all logos and photos from marketing-brochure.pdf"

**Common workflows:**
1. Scanned Document Processing: pdf_read_file → check content_type → if "scanned_images" → pdf_assets_file
2. Visual Analysis: Extract images → Run OCR or image analysis → Combine with text
3. Asset Recovery: Extract assets → Catalog images → Use in other documents

**Best practices:** Use after pdf_read_file shows content_type as "scanned_images" or "mixed", supports JPEG, PNG, TIFF, JBIG2.`

	PDFValidateFileDescription = `Verify PDF file integrity and readability before processing.

**When to use:** Before attempting to read or process any PDF file, especially in automated workflows or when handling user uploads.

**Why it's useful:** Prevents processing errors, identifies corrupted files early, and ensures compatibility with extraction tools.

**Examples:**
• Batch processing safety: "Validate all PDFs in /invoices/ before bulk text extraction"
• Upload verification: "Check user-uploaded contract.pdf is valid before processing"
• Quality control: "Verify exported-report.pdf is readable before sending to client"

**Common workflows:**
1. Automated Processing: Validate → Process if valid → Handle errors gracefully
2. File Quality Check: Validate → Report issues → Fix or reject bad files
3. Pre-processing Pipeline: Validate → Route to appropriate extraction method

**Best practices:** Always run this first in automated workflows, essential for production systems handling unknown PDFs.`

	PDFStatsFileDescription = `Get comprehensive metadata and statistics about PDF documents.

**When to use:** Need document properties, page count, file size, creation info, or to understand document structure before processing.

**Why it's useful:** Provides essential metadata for document management, helps choose processing strategies, and offers insights into document origin.

**Examples:**
• Document management: "Get creation date and author from legal-contract.pdf for filing system"
• Processing decisions: "Check page count of manual.pdf to estimate processing time"
• Audit trail: "Get metadata from signed-agreement.pdf for compliance records"

**Common workflows:**
1. Document Cataloging: Get stats → Store metadata → Index for search
2. Processing Planning: Check stats → Choose extraction method → Allocate resources
3. Compliance & Audit: Extract metadata → Verify properties → Log for records

**Best practices:** Useful for document management systems, helps estimate processing requirements for large files.`

	// Search and Discovery Tools
	PDFSearchDirectoryDescription = `Discover and filter PDF files across directories with intelligent search.

**When to use:** Need to find specific PDFs by name patterns, explore unknown directories, or build file inventories.

**Why it's useful:** Quickly locates relevant documents without manual browsing, supports fuzzy matching for partial names.

**Examples:**
• Find invoices: "Search /documents/ for files containing 'invoice' or '2024'"
• Locate reports: "Find all PDF files with 'quarterly' in /reports/ directory"
• Inventory building: "List all PDFs in /archive/ to understand content scope"

**Common workflows:**
1. Targeted Processing: Search for specific patterns → Process matching files → Generate reports
2. Content Discovery: Explore directory → Identify document types → Plan extraction strategy
3. Batch Operations: Find files → Validate each → Process in sequence

**Best practices:** Use fuzzy search for partial matches, combine with pdf_stats_directory for comprehensive overview.`

	PDFStatsDirectoryDescription = `Analyze PDF collections and get comprehensive directory statistics.

**When to use:** Need overview of PDF collection size, total file count, storage usage, or to assess processing requirements.

**Why it's useful:** Provides high-level insights for capacity planning, identifies largest files, and helps prioritize processing efforts.

**Examples:**
• Capacity planning: "Analyze /archive/ to understand storage usage and processing load"
• Collection overview: "Get statistics on /contracts/ to plan migration strategy"
• Resource allocation: "Check /invoices/ stats to estimate batch processing time"

**Common workflows:**
1. Migration Planning: Get directory stats → Estimate resources → Plan migration phases
2. Storage Management: Analyze usage → Identify large files → Optimize storage
3. Processing Strategy: Review collection → Plan batch sizes → Allocate processing time

**Best practices:** Essential for understanding large document collections before bulk processing operations.`

	// Utility Tools
	PDFServerInfoDescription = `Get real-time server status, available tools, and system capabilities.

**When to use:** Starting work with the PDF server, troubleshooting issues, or checking available functionality.

**Why it's useful:** Provides complete overview of server capabilities, current configuration, and directory contents for informed decision-making.

**Examples:**
• System check: "Verify server is ready and all tools are available before batch processing"
• Troubleshooting: "Check server info to diagnose why files aren't being found"
• Capability discovery: "See all available tools and their descriptions for new projects"

**Common workflows:**
1. Session Startup: Check server info → Verify capabilities → Plan processing approach
2. Debugging: Review server status → Check directory paths → Verify tool availability
3. Planning: Review available tools → Choose appropriate methods → Execute workflow

**Best practices:** Run at start of sessions, provides cached directory contents for quick overview.`

	// Advanced Extraction Tools
	PDFExtractStructuredDescription = `Extract text with detailed layout, positioning, and formatting information.

**When to use:** Need precise text positioning, formatting details, or to preserve document layout in extraction.

**Why it's useful:** Maintains spatial relationships between text elements, preserves formatting context, enables layout-aware processing.

**Examples:**
• Form processing: "Extract structured data from tax-form.pdf preserving field positions"
• Layout analysis: "Get formatted text from newsletter.pdf maintaining column structure"
• Template matching: "Extract structured content from invoice.pdf for data mapping"

**Common workflows:**
1. Form Processing: Extract structured → Map to fields → Validate data → Store results
2. Layout Preservation: Extract with positioning → Reconstruct layout → Convert to other formats
3. Data Mining: Extract structured → Analyze patterns → Extract business rules

**Best practices:** Use when layout matters, provides coordinates and formatting that basic text extraction loses.`

	PDFExtractCompleteDescription = `Perform comprehensive extraction of all content types in a single operation.

**When to use:** Need complete document analysis including text, images, tables, forms, and annotations in one pass.

**Why it's useful:** Most efficient for full document analysis, provides unified view of all content types, reduces multiple API calls.

**Examples:**
• Full document analysis: "Complete extraction of annual-report.pdf for comprehensive analysis"
• Content inventory: "Extract everything from contract.pdf to understand all embedded content"
• Migration preparation: "Full extraction of legacy-document.pdf for system migration"

**Common workflows:**
1. Complete Analysis: Full extraction → Categorize content → Process each type appropriately
2. Document Migration: Extract all → Map content types → Reconstruct in target format
3. Comprehensive Audit: Extract everything → Analyze completeness → Generate detailed reports

**Best practices:** Most resource-intensive but most comprehensive, ideal for critical documents requiring full analysis.`

	PDFExtractTablesDescription = `Extract tabular data with preserved row/column structure and relationships.

**When to use:** PDFs contain tables, spreadsheet data, or structured information that needs to maintain relationships.

**Why it's useful:** Preserves table structure that general text extraction would flatten, enables direct data analysis and database import.

**Examples:**
• Financial reports: "Extract budget tables from annual-report.pdf for spreadsheet analysis"
• Data import: "Get structured pricing table from catalog.pdf for database update"
• Comparison analysis: "Extract performance tables from multiple quarterly reports"

**Common workflows:**
1. Data Analysis: Extract tables → Import to spreadsheet → Perform analysis → Generate insights
2. Database Import: Extract structured data → Validate formats → Import to database
3. Report Generation: Extract tables → Format for presentation → Include in new documents

**Best practices:** Ideal for financial reports, catalogs, and any document with structured data relationships.`

	PDFExtractFormsDescription = `Extract interactive form fields including values, field types, and properties.

**When to use:** Processing fillable PDF forms, extracting user input, or analyzing form structure.

**Why it's useful:** Recovers form data that text extraction misses, identifies field types and validation rules, preserves form relationships.

**Examples:**
• Application processing: "Extract filled application.pdf to get user responses for review"
• Form analysis: "Get field structure from template.pdf to understand data collection"
• Data migration: "Extract form data from legacy-forms/ for new system import"

**Common workflows:**
1. Form Processing: Extract form data → Validate responses → Route for approval → Store results
2. Form Analytics: Extract field structure → Analyze completion patterns → Optimize forms
3. System Migration: Extract form data → Map to new fields → Import to new system

**Best practices:** Essential for any PDF forms processing, works with AcroForms and some XFA forms.`

	PDFExtractSemanticDescription = `Extract content with intelligent grouping and relationship detection.

**When to use:** Need to understand document structure, identify content relationships, or perform intelligent content analysis.

**Why it's useful:** Groups related content intelligently, identifies document sections, provides context for better content understanding.

**Examples:**
• Document analysis: "Semantic extraction of research-paper.pdf to identify sections and citations"
• Content organization: "Group related content from manual.pdf by topic and importance"
• Information extraction: "Identify key concepts and relationships in legal-contract.pdf"

**Common workflows:**
1. Content Analysis: Semantic extraction → Identify themes → Generate summaries → Create insights
2. Document Classification: Extract semantic structure → Analyze patterns → Classify document types
3. Knowledge Extraction: Identify relationships → Build knowledge graphs → Enable intelligent search

**Best practices:** Advanced feature for AI-powered document analysis, ideal for research and knowledge extraction.`

	// Query and Analysis Tools
	PDFQueryContentDescription = `Search and filter extracted PDF content using structured query criteria.

**When to use:** Need to find specific information within documents, perform targeted content searches, or filter extracted data.

**Why it's useful:** Enables precise content searches without re-reading files, supports complex filtering, provides contextual results.

**Examples:**
• Information retrieval: "Find all mentions of 'liability' in legal-contract.pdf with context"
• Data extraction: "Search invoice.pdf for line items over $1000 with descriptions"
• Research assistance: "Find methodology sections in research-papers/ directory"

**Common workflows:**
1. Targeted Search: Query specific content → Review results → Extract relevant data → Use in analysis
2. Compliance Check: Search for required terms → Verify presence → Generate compliance reports
3. Research Support: Query documents → Collect relevant passages → Compile research materials

**Best practices:** Powerful for large documents, use specific query terms for best results, supports regex patterns.`

	PDFAnalyzeDocumentDescription = `Perform deep document analysis including classification, structure detection, and quality assessment.

**When to use:** Need to understand document type, assess quality, detect structure, or perform comprehensive document intelligence.

**Why it's useful:** Automatically classifies documents, identifies structural elements, assesses readability, provides processing recommendations.

**Examples:**
• Document classification: "Analyze unknown-document.pdf to determine if it's a contract, invoice, or report"
• Quality assessment: "Analyze scanned-doc.pdf to check readability and recommend processing approach"
• Structure analysis: "Understand organization of complex-manual.pdf for automated processing"

**Common workflows:**
1. Automated Classification: Analyze document → Classify type → Route to appropriate processing → Apply type-specific extraction
2. Quality Control: Analyze document → Assess quality → Recommend improvements → Flag issues
3. Processing Optimization: Analyze structure → Choose best extraction method → Configure processing parameters

**Best practices:** Most advanced analysis tool, provides AI-powered insights for optimal document processing strategies.`

	// Metadata Tools
	PDFGetPageInfoDescription = `Get detailed page dimensions, layout properties, and structural information.

**When to use:** Need page specifications for layout analysis, conversion planning, or understanding document structure.

**Why it's useful:** Provides precise page measurements, rotation info, and layout data essential for accurate document reconstruction.

**Examples:**
• Layout analysis: "Get page dimensions from brochure.pdf for template creation"
• Conversion planning: "Check page properties of manual.pdf before format conversion"
• Print preparation: "Verify page specs of document.pdf for printing requirements"

**Common workflows:**
1. Format Conversion: Get page info → Plan layout → Convert preserving structure → Validate output
2. Template Creation: Analyze page structure → Design templates → Apply to similar documents
3. Print Production: Check specifications → Verify compatibility → Prepare for printing

**Best practices:** Essential for any layout-sensitive operations, provides technical specifications for accurate processing.`

	PDFGetMetadataDescription = `Extract comprehensive document metadata including creation info, security settings, and properties.

**When to use:** Need document provenance, creation details, author information, or security properties for compliance or analysis.

**Why it's useful:** Provides document history, creation context, and technical properties often required for legal, compliance, or management purposes.

**Examples:**
• Compliance audit: "Get metadata from contracts/ to verify creation dates and authors"
• Document management: "Extract metadata from reports/ for cataloging and search indexing"
• Security analysis: "Check metadata of received-document.pdf for security settings and origins"

**Common workflows:**
1. Compliance Management: Extract metadata → Verify requirements → Document compliance → Store records
2. Document Cataloging: Get metadata → Index properties → Enable advanced search → Maintain inventory
3. Security Assessment: Check metadata → Identify risks → Apply security policies → Monitor compliance

**Best practices:** Critical for document management systems, provides essential data for compliance and security workflows.`
)

// ToolDescriptions maps tool names to their comprehensive descriptions
var ToolDescriptions = map[string]string{
	"pdf_read_file":          PDFReadFileDescription,
	"pdf_assets_file":        PDFAssetsFileDescription,
	"pdf_validate_file":      PDFValidateFileDescription,
	"pdf_stats_file":         PDFStatsFileDescription,
	"pdf_search_directory":   PDFSearchDirectoryDescription,
	"pdf_stats_directory":    PDFStatsDirectoryDescription,
	"pdf_server_info":        PDFServerInfoDescription,
	"pdf_extract_structured": PDFExtractStructuredDescription,
	"pdf_extract_complete":   PDFExtractCompleteDescription,
	"pdf_extract_tables":     PDFExtractTablesDescription,
	"pdf_extract_forms":      PDFExtractFormsDescription,
	"pdf_extract_semantic":   PDFExtractSemanticDescription,
	"pdf_query_content":      PDFQueryContentDescription,
	"pdf_analyze_document":   PDFAnalyzeDocumentDescription,
	"pdf_get_page_info":      PDFGetPageInfoDescription,
	"pdf_get_metadata":       PDFGetMetadataDescription,
}

// GetToolDescription returns the comprehensive description for a tool
func GetToolDescription(toolName string) string {
	if desc, exists := ToolDescriptions[toolName]; exists {
		return desc
	}
	return "Tool description not available"
}

// GetAllToolNames returns a list of all available tool names
func GetAllToolNames() []string {
	var names []string
	for name := range ToolDescriptions {
		names = append(names, name)
	}
	return names
}
