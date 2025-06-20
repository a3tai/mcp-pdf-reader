package wrapper

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// PDFLibraryFactory creates PDF library instances with unified interface
type PDFLibraryFactory struct {
	defaultLibrary LibraryType
	config         FactoryConfig
}

// FactoryConfig contains configuration options for the factory
type FactoryConfig struct {
	// PreferredLibrary is the default library to use when LibraryAuto is specified
	PreferredLibrary LibraryType `json:"preferred_library"`

	// EnableAutoSelection allows the factory to automatically choose the best library
	// based on the operation type and file characteristics
	EnableAutoSelection bool `json:"enable_auto_selection"`

	// MaxFileSize limits the file size for certain operations (in bytes)
	MaxFileSize int64 `json:"max_file_size"`

	// DebugMode enables debug logging for library operations
	DebugMode bool `json:"debug_mode"`

	// LibrarySpecificConfigs holds configuration for each library type
	LibrarySpecificConfigs map[LibraryType]interface{} `json:"library_configs,omitempty"`
}

// NewPDFLibraryFactory creates a new factory with default configuration
func NewPDFLibraryFactory() *PDFLibraryFactory {
	return &PDFLibraryFactory{
		defaultLibrary: LibraryAuto,
		config: FactoryConfig{
			PreferredLibrary:       LibraryPDFCPU,
			EnableAutoSelection:    true,
			MaxFileSize:            100 * 1024 * 1024, // 100MB
			DebugMode:              false,
			LibrarySpecificConfigs: make(map[LibraryType]interface{}),
		},
	}
}

// NewPDFLibraryFactoryWithConfig creates a factory with custom configuration
func NewPDFLibraryFactoryWithConfig(config FactoryConfig) *PDFLibraryFactory {
	return &PDFLibraryFactory{
		defaultLibrary: config.PreferredLibrary,
		config:         config,
	}
}

// Create instantiates a PDF library of the specified type
func (f *PDFLibraryFactory) Create(libType LibraryType) (PDFLibrary, error) {
	switch libType {
	case LibraryCustom:
		return f.createCustomLibrary()
	case LibraryPDFCPU:
		return f.createPDFCPULibrary()
	case LibraryLedongthuc:
		return f.createLedongthucLibrary()
	case LibraryAuto:
		return f.createAutoLibrary()
	default:
		return nil, &WrapperError{
			Library: libType,
			Op:      "create",
			Err:     fmt.Errorf("unknown library type: %s", libType),
		}
	}
}

// CreateForFile creates the best library instance for a specific file
func (f *PDFLibraryFactory) CreateForFile(filePath string) (PDFLibrary, error) {
	if !f.config.EnableAutoSelection {
		return f.Create(f.defaultLibrary)
	}

	// Analyze file to determine best library
	libType, err := f.analyzeFile(filePath)
	if err != nil {
		// Fall back to default library if analysis fails
		if f.config.DebugMode {
			fmt.Printf("File analysis failed, using default library: %v\n", err)
		}
		return f.Create(f.config.PreferredLibrary)
	}

	return f.Create(libType)
}

// CreateForReader creates the best library instance for reading from io.Reader
func (f *PDFLibraryFactory) CreateForReader(reader io.Reader) (PDFLibrary, error) {
	// For readers, we can't easily analyze the content, so use preferred library
	return f.Create(f.config.PreferredLibrary)
}

// CreateForOperation creates the best library for a specific operation type
func (f *PDFLibraryFactory) CreateForOperation(operation OperationType) (PDFLibrary, error) {
	if !f.config.EnableAutoSelection {
		return f.Create(f.defaultLibrary)
	}

	libType := f.selectLibraryForOperation(operation)
	return f.Create(libType)
}

// OperationType represents different types of PDF operations
type OperationType string

const (
	OperationTextExtraction  OperationType = "text_extraction"
	OperationFormExtraction  OperationType = "form_extraction"
	OperationImageExtraction OperationType = "image_extraction"
	OperationTableExtraction OperationType = "table_extraction"
	OperationMetadata        OperationType = "metadata"
	OperationValidation      OperationType = "validation"
	OperationSecurity        OperationType = "security"
	OperationGeneral         OperationType = "general"
)

// selectLibraryForOperation chooses the best library for specific operations
func (f *PDFLibraryFactory) selectLibraryForOperation(operation OperationType) LibraryType {
	switch operation {
	case OperationFormExtraction:
		// pdfcpu is best for form operations
		return LibraryPDFCPU
	case OperationSecurity:
		// pdfcpu has better security/encryption support
		return LibraryPDFCPU
	case OperationTextExtraction:
		// ledongthuc is lightweight and good for text
		return LibraryLedongthuc
	case OperationImageExtraction:
		// pdfcpu has better image extraction capabilities
		return LibraryPDFCPU
	case OperationTableExtraction:
		// Custom implementation might be better for complex table detection
		return LibraryCustom
	case OperationMetadata, OperationValidation:
		// ledongthuc is sufficient for basic operations
		return LibraryLedongthuc
	default:
		return f.config.PreferredLibrary
	}
}

// analyzeFile examines a PDF file to determine the best library to use
func (f *PDFLibraryFactory) analyzeFile(filePath string) (LibraryType, error) {
	// Check if file exists and is readable
	info, err := os.Stat(filePath)
	if err != nil {
		return "", &WrapperError{
			Library: LibraryAuto,
			Op:      "analyze",
			Err:     fmt.Errorf("cannot access file: %w", err),
		}
	}

	// Check file size
	if info.Size() > f.config.MaxFileSize {
		return "", &WrapperError{
			Library: LibraryAuto,
			Op:      "analyze",
			Err:     fmt.Errorf("file size %d exceeds maximum %d", info.Size(), f.config.MaxFileSize),
		}
	}

	// Check file extension
	ext := strings.ToLower(filepath.Ext(filePath))
	if ext != ".pdf" {
		return "", &WrapperError{
			Library: LibraryAuto,
			Op:      "analyze",
			Err:     fmt.Errorf("file does not have .pdf extension: %s", ext),
		}
	}

	// For now, use a simple heuristic based on file size
	// Larger files might benefit from pdfcpu's more robust parsing
	if info.Size() > 10*1024*1024 { // 10MB
		return LibraryPDFCPU, nil
	}

	// For smaller files, ledongthuc might be faster
	return LibraryLedongthuc, nil
}

// createCustomLibrary creates an instance of the custom PDF library
func (f *PDFLibraryFactory) createCustomLibrary() (PDFLibrary, error) {
	return NewCustomPDFLibrary(f.config), nil
}

// createPDFCPULibrary creates an instance of the pdfcpu library wrapper
func (f *PDFLibraryFactory) createPDFCPULibrary() (PDFLibrary, error) {
	return NewPDFCPULibrary(f.config), nil
}

// createLedongthucLibrary creates an instance of the ledongthuc library wrapper
func (f *PDFLibraryFactory) createLedongthucLibrary() (PDFLibrary, error) {
	return NewLedongthucLibrary(f.config), nil
}

// createAutoLibrary creates an instance using automatic selection
func (f *PDFLibraryFactory) createAutoLibrary() (PDFLibrary, error) {
	// Auto defaults to the preferred library
	return f.Create(f.config.PreferredLibrary)
}

// SetDefaultLibrary changes the default library type
func (f *PDFLibraryFactory) SetDefaultLibrary(libType LibraryType) {
	f.defaultLibrary = libType
	f.config.PreferredLibrary = libType
}

// GetDefaultLibrary returns the current default library type
func (f *PDFLibraryFactory) GetDefaultLibrary() LibraryType {
	return f.defaultLibrary
}

// SetConfig updates the factory configuration
func (f *PDFLibraryFactory) SetConfig(config FactoryConfig) {
	f.config = config
	f.defaultLibrary = config.PreferredLibrary
}

// GetConfig returns the current factory configuration
func (f *PDFLibraryFactory) GetConfig() FactoryConfig {
	return f.config
}

// GetSupportedLibraries returns a list of all supported library types
func (f *PDFLibraryFactory) GetSupportedLibraries() []LibraryType {
	return []LibraryType{
		LibraryCustom,
		LibraryPDFCPU,
		LibraryLedongthuc,
		LibraryAuto,
	}
}

// ValidateLibraryType checks if a library type is supported
func (f *PDFLibraryFactory) ValidateLibraryType(libType LibraryType) error {
	for _, supported := range f.GetSupportedLibraries() {
		if libType == supported {
			return nil
		}
	}
	return &WrapperError{
		Library: libType,
		Op:      "validate",
		Err:     fmt.Errorf("unsupported library type: %s", libType),
	}
}

// GetLibraryCapabilities returns the capabilities of each library
func (f *PDFLibraryFactory) GetLibraryCapabilities() map[LibraryType]LibraryCapabilities {
	return map[LibraryType]LibraryCapabilities{
		LibraryCustom: {
			TextExtraction:  true,
			ImageExtraction: true,
			FormExtraction:  false,
			TableExtraction: true,
			Encryption:      false,
			Validation:      true,
			Performance:     "medium",
			PurGo:           true,
		},
		LibraryPDFCPU: {
			TextExtraction:  true,
			ImageExtraction: true,
			FormExtraction:  true,
			TableExtraction: true,
			Encryption:      true,
			Validation:      true,
			Performance:     "high",
			PurGo:           true,
		},
		LibraryLedongthuc: {
			TextExtraction:  true,
			ImageExtraction: false,
			FormExtraction:  false,
			TableExtraction: false,
			Encryption:      false,
			Validation:      true,
			Performance:     "fast",
			PurGo:           true,
		},
	}
}

// LibraryCapabilities describes what each library can do
type LibraryCapabilities struct {
	TextExtraction  bool   `json:"text_extraction"`
	ImageExtraction bool   `json:"image_extraction"`
	FormExtraction  bool   `json:"form_extraction"`
	TableExtraction bool   `json:"table_extraction"`
	Encryption      bool   `json:"encryption"`
	Validation      bool   `json:"validation"`
	Performance     string `json:"performance"` // "fast", "medium", "high"
	PurGo           bool   `json:"pure_go"`
}

// GetRecommendedLibrary returns the recommended library for given requirements
func (f *PDFLibraryFactory) GetRecommendedLibrary(requirements LibraryRequirements) LibraryType {
	capabilities := f.GetLibraryCapabilities()

	for _, libType := range []LibraryType{LibraryPDFCPU, LibraryCustom, LibraryLedongthuc} {
		caps := capabilities[libType]

		if requirements.FormExtraction && !caps.FormExtraction {
			continue
		}
		if requirements.ImageExtraction && !caps.ImageExtraction {
			continue
		}
		if requirements.TableExtraction && !caps.TableExtraction {
			continue
		}
		if requirements.Encryption && !caps.Encryption {
			continue
		}
		if requirements.PureGo && !caps.PurGo {
			continue
		}

		// Found a suitable library
		return libType
	}

	// Fallback to preferred library
	return f.config.PreferredLibrary
}

// LibraryRequirements specifies requirements for library selection
type LibraryRequirements struct {
	TextExtraction  bool `json:"text_extraction"`
	ImageExtraction bool `json:"image_extraction"`
	FormExtraction  bool `json:"form_extraction"`
	TableExtraction bool `json:"table_extraction"`
	Encryption      bool `json:"encryption"`
	PureGo          bool `json:"pure_go"`
}
