package wrapper

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPDFLibraryFactory_Creation(t *testing.T) {
	factory := NewPDFLibraryFactory()
	assert.NotNil(t, factory)
	assert.Equal(t, LibraryAuto, factory.GetDefaultLibrary())
	assert.True(t, factory.GetConfig().EnableAutoSelection)
}

func TestPDFLibraryFactory_CreateWithConfig(t *testing.T) {
	config := FactoryConfig{
		PreferredLibrary:    LibraryPDFCPU,
		EnableAutoSelection: false,
		MaxFileSize:         50 * 1024 * 1024,
		DebugMode:           true,
	}

	factory := NewPDFLibraryFactoryWithConfig(config)
	assert.NotNil(t, factory)
	assert.Equal(t, LibraryPDFCPU, factory.GetDefaultLibrary())
	assert.False(t, factory.GetConfig().EnableAutoSelection)
	assert.True(t, factory.GetConfig().DebugMode)
}

func TestPDFLibraryFactory_CreateLibraries(t *testing.T) {
	factory := NewPDFLibraryFactory()

	tests := []struct {
		name        string
		libType     LibraryType
		expectError bool
	}{
		{
			name:        "create_custom_library",
			libType:     LibraryCustom,
			expectError: false,
		},
		{
			name:        "create_pdfcpu_library",
			libType:     LibraryPDFCPU,
			expectError: false,
		},
		{
			name:        "create_ledongthuc_library",
			libType:     LibraryLedongthuc,
			expectError: false,
		},
		{
			name:        "create_auto_library",
			libType:     LibraryAuto,
			expectError: false,
		},
		{
			name:        "create_invalid_library",
			libType:     LibraryType("invalid"),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lib, err := factory.Create(tt.libType)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, lib)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, lib)

				if tt.libType != LibraryAuto {
					assert.Equal(t, tt.libType, lib.GetLibraryType())
				}

				// Test basic operations
				assert.NoError(t, lib.Validate())
				assert.NotEmpty(t, lib.GetVersion())
				assert.NoError(t, lib.Close())
			}
		})
	}
}

func TestPDFLibraryFactory_ValidateLibraryType(t *testing.T) {
	factory := NewPDFLibraryFactory()

	validTypes := []LibraryType{
		LibraryCustom,
		LibraryPDFCPU,
		LibraryLedongthuc,
		LibraryAuto,
	}

	for _, libType := range validTypes {
		assert.NoError(t, factory.ValidateLibraryType(libType))
	}

	// Test invalid type
	assert.Error(t, factory.ValidateLibraryType(LibraryType("invalid")))
}

func TestPDFLibraryFactory_GetSupportedLibraries(t *testing.T) {
	factory := NewPDFLibraryFactory()
	supported := factory.GetSupportedLibraries()

	assert.Contains(t, supported, LibraryCustom)
	assert.Contains(t, supported, LibraryPDFCPU)
	assert.Contains(t, supported, LibraryLedongthuc)
	assert.Contains(t, supported, LibraryAuto)
	assert.Len(t, supported, 4)
}

func TestPDFLibraryFactory_GetLibraryCapabilities(t *testing.T) {
	factory := NewPDFLibraryFactory()
	capabilities := factory.GetLibraryCapabilities()

	// Test pdfcpu capabilities
	pdfcpuCaps := capabilities[LibraryPDFCPU]
	assert.True(t, pdfcpuCaps.TextExtraction)
	assert.True(t, pdfcpuCaps.ImageExtraction)
	assert.True(t, pdfcpuCaps.FormExtraction)
	assert.True(t, pdfcpuCaps.TableExtraction)
	assert.True(t, pdfcpuCaps.Encryption)
	assert.True(t, pdfcpuCaps.PurGo)

	// Test ledongthuc capabilities
	ledongthucCaps := capabilities[LibraryLedongthuc]
	assert.True(t, ledongthucCaps.TextExtraction)
	assert.False(t, ledongthucCaps.ImageExtraction)
	assert.False(t, ledongthucCaps.FormExtraction)
	assert.False(t, ledongthucCaps.TableExtraction)
	assert.False(t, ledongthucCaps.Encryption)
	assert.True(t, ledongthucCaps.PurGo)

	// Test custom capabilities
	customCaps := capabilities[LibraryCustom]
	assert.True(t, customCaps.TextExtraction)
	assert.True(t, customCaps.ImageExtraction)
	assert.False(t, customCaps.FormExtraction)
	assert.True(t, customCaps.TableExtraction)
	assert.False(t, customCaps.Encryption)
	assert.True(t, customCaps.PurGo)
}

func TestPDFLibraryFactory_GetRecommendedLibrary(t *testing.T) {
	factory := NewPDFLibraryFactory()

	tests := []struct {
		name         string
		requirements LibraryRequirements
		expected     LibraryType
	}{
		{
			name: "form_extraction_required",
			requirements: LibraryRequirements{
				FormExtraction: true,
			},
			expected: LibraryPDFCPU,
		},
		{
			name: "encryption_required",
			requirements: LibraryRequirements{
				Encryption: true,
			},
			expected: LibraryPDFCPU,
		},
		{
			name: "text_only",
			requirements: LibraryRequirements{
				TextExtraction: true,
			},
			expected: LibraryPDFCPU, // First in the list that satisfies
		},
		{
			name: "impossible_requirements",
			requirements: LibraryRequirements{
				FormExtraction: true,
				PureGo:         false, // No library satisfies this
			},
			expected: LibraryPDFCPU, // Falls back to preferred
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recommended := factory.GetRecommendedLibrary(tt.requirements)
			assert.Equal(t, tt.expected, recommended)
		})
	}
}

func TestPDFLibraryFactory_SelectLibraryForOperation(t *testing.T) {
	factory := NewPDFLibraryFactory()

	tests := []struct {
		operation OperationType
		expected  LibraryType
	}{
		{OperationFormExtraction, LibraryPDFCPU},
		{OperationSecurity, LibraryPDFCPU},
		{OperationTextExtraction, LibraryLedongthuc},
		{OperationImageExtraction, LibraryPDFCPU},
		{OperationTableExtraction, LibraryCustom},
		{OperationMetadata, LibraryLedongthuc},
		{OperationValidation, LibraryLedongthuc},
		{OperationGeneral, LibraryPDFCPU}, // Falls back to preferred
	}

	for _, tt := range tests {
		t.Run(string(tt.operation), func(t *testing.T) {
			lib, err := factory.CreateForOperation(tt.operation)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, lib.GetLibraryType())
		})
	}
}

func TestPDFLibraryFactory_AnalyzeFile(t *testing.T) {
	factory := NewPDFLibraryFactory()

	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// Test non-existent file
	nonExistentFile := filepath.Join(tempDir, "nonexistent.pdf")
	_, err := factory.CreateForFile(nonExistentFile)
	assert.Error(t, err)

	// Test non-PDF file
	txtFile := filepath.Join(tempDir, "test.txt")
	err = os.WriteFile(txtFile, []byte("not a pdf"), 0644)
	require.NoError(t, err)

	_, err = factory.CreateForFile(txtFile)
	assert.Error(t, err)

	// Test empty PDF file (should fail)
	emptyPDF := filepath.Join(tempDir, "empty.pdf")
	err = os.WriteFile(emptyPDF, []byte(""), 0644)
	require.NoError(t, err)

	_, err = factory.CreateForFile(emptyPDF)
	// Should fall back to preferred library even if analysis fails
	assert.NoError(t, err)

	// Test file size limits
	factory.SetConfig(FactoryConfig{
		PreferredLibrary:    LibraryPDFCPU,
		EnableAutoSelection: true,
		MaxFileSize:         10, // Very small limit
		DebugMode:           false,
	})

	largePDF := filepath.Join(tempDir, "large.pdf")
	err = os.WriteFile(largePDF, make([]byte, 100), 0644) // Larger than limit
	require.NoError(t, err)

	_, err = factory.CreateForFile(largePDF)
	// Should fall back to preferred library
	assert.NoError(t, err)
}

func TestPDFCPULibrary_Basic(t *testing.T) {
	lib := NewPDFCPULibrary(FactoryConfig{})

	assert.Equal(t, LibraryPDFCPU, lib.GetLibraryType())
	assert.NotEmpty(t, lib.GetVersion())
	assert.NoError(t, lib.Validate())

	// Test close
	assert.NoError(t, lib.Close())

	// Operations after close should fail
	assert.Error(t, lib.Validate())
	_, err := lib.Open(strings.NewReader("test"))
	assert.Error(t, err)
}

func TestLedongthucLibrary_Basic(t *testing.T) {
	lib := NewLedongthucLibrary(FactoryConfig{})

	assert.Equal(t, LibraryLedongthuc, lib.GetLibraryType())
	assert.NotEmpty(t, lib.GetVersion())
	assert.NoError(t, lib.Validate())

	// Test close
	assert.NoError(t, lib.Close())

	// Operations after close should fail
	assert.Error(t, lib.Validate())
	_, err := lib.Open(strings.NewReader("test"))
	assert.Error(t, err)
}

func TestCustomLibrary_Basic(t *testing.T) {
	lib := NewCustomPDFLibrary(FactoryConfig{})

	assert.Equal(t, LibraryCustom, lib.GetLibraryType())
	assert.NotEmpty(t, lib.GetVersion())
	assert.NoError(t, lib.Validate())

	// Test close
	assert.NoError(t, lib.Close())

	// Operations after close should fail
	assert.Error(t, lib.Validate())
	_, err := lib.Open(strings.NewReader("test"))
	assert.Error(t, err)
}

func TestWrapperError(t *testing.T) {
	err := &WrapperError{
		Library: LibraryPDFCPU,
		Op:      "test_operation",
		Err:     assert.AnError,
	}

	assert.Contains(t, err.Error(), "pdfcpu")
	assert.Contains(t, err.Error(), "test_operation")
	assert.Equal(t, assert.AnError, err.Unwrap())
}

func TestFactoryConfig_SetAndGet(t *testing.T) {
	factory := NewPDFLibraryFactory()

	newConfig := FactoryConfig{
		PreferredLibrary:    LibraryLedongthuc,
		EnableAutoSelection: false,
		MaxFileSize:         200 * 1024 * 1024,
		DebugMode:           true,
	}

	factory.SetConfig(newConfig)
	assert.Equal(t, newConfig, factory.GetConfig())
	assert.Equal(t, LibraryLedongthuc, factory.GetDefaultLibrary())

	// Test SetDefaultLibrary
	factory.SetDefaultLibrary(LibraryCustom)
	assert.Equal(t, LibraryCustom, factory.GetDefaultLibrary())
	assert.Equal(t, LibraryCustom, factory.GetConfig().PreferredLibrary)
}

func TestLibraryRequirements(t *testing.T) {
	requirements := LibraryRequirements{
		TextExtraction:  true,
		FormExtraction:  false,
		ImageExtraction: true,
		TableExtraction: false,
		Encryption:      false,
		PureGo:          true,
	}

	factory := NewPDFLibraryFactory()
	recommended := factory.GetRecommendedLibrary(requirements)

	// Should recommend a library that supports text, images, and is pure Go
	capabilities := factory.GetLibraryCapabilities()
	caps := capabilities[recommended]

	assert.True(t, caps.TextExtraction)
	assert.True(t, caps.ImageExtraction)
	assert.True(t, caps.PurGo)
}

func TestOperationType_Constants(t *testing.T) {
	// Test that all operation types are defined
	operations := []OperationType{
		OperationTextExtraction,
		OperationFormExtraction,
		OperationImageExtraction,
		OperationTableExtraction,
		OperationMetadata,
		OperationValidation,
		OperationSecurity,
		OperationGeneral,
	}

	for _, op := range operations {
		assert.NotEmpty(t, string(op))
	}
}

func TestLibraryType_Constants(t *testing.T) {
	// Test that all library types are defined
	types := []LibraryType{
		LibraryCustom,
		LibraryPDFCPU,
		LibraryLedongthuc,
		LibraryAuto,
	}

	for _, libType := range types {
		assert.NotEmpty(t, string(libType))
	}
}

func BenchmarkPDFLibraryFactory_Create(b *testing.B) {
	factory := NewPDFLibraryFactory()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		lib, err := factory.Create(LibraryPDFCPU)
		if err != nil {
			b.Fatal(err)
		}
		lib.Close()
	}
}

func BenchmarkPDFLibraryFactory_GetRecommendedLibrary(b *testing.B) {
	factory := NewPDFLibraryFactory()
	requirements := LibraryRequirements{
		TextExtraction:  true,
		FormExtraction:  true,
		ImageExtraction: false,
		TableExtraction: false,
		Encryption:      false,
		PureGo:          true,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = factory.GetRecommendedLibrary(requirements)
	}
}

// Example tests demonstrating usage
func ExamplePDFLibraryFactory_Create() {
	factory := NewPDFLibraryFactory()

	// Create a pdfcpu library instance
	lib, err := factory.Create(LibraryPDFCPU)
	if err != nil {
		panic(err)
	}
	defer lib.Close()

	// Use the library
	// doc, err := lib.OpenFile("example.pdf")
	// ...
}

func ExamplePDFLibraryFactory_CreateForOperation() {
	factory := NewPDFLibraryFactory()

	// Create the best library for form extraction
	lib, err := factory.CreateForOperation(OperationFormExtraction)
	if err != nil {
		panic(err)
	}
	defer lib.Close()

	// This will automatically select pdfcpu since it's best for forms
	// doc, err := lib.OpenFile("form.pdf")
	// forms, err := doc.ExtractForms()
	// ...
}

func ExamplePDFLibraryFactory_GetRecommendedLibrary() {
	factory := NewPDFLibraryFactory()

	requirements := LibraryRequirements{
		FormExtraction: true,
		Encryption:     true,
		PureGo:         true,
	}

	recommended := factory.GetRecommendedLibrary(requirements)

	lib, err := factory.Create(recommended)
	if err != nil {
		panic(err)
	}
	defer lib.Close()

	// Use the recommended library
	// ...
}
