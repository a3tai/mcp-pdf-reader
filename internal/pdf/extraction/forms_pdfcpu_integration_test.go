package extraction

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPDFCPUFormExtractor_IntegrationWithGeneratedPDFs(t *testing.T) {
	// Skip if running short tests
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tests := []struct {
		name           string
		pdfFile        string
		expectedFields map[string]struct {
			fieldType  FormFieldType
			hasValue   bool
			hasOptions bool
			required   bool
			readOnly   bool
			maxLength  int
		}
	}{
		{
			name:    "basic_form",
			pdfFile: "basic-form.pdf",
			expectedFields: map[string]struct {
				fieldType  FormFieldType
				hasValue   bool
				hasOptions bool
				required   bool
				readOnly   bool
				maxLength  int
			}{
				"name": {
					fieldType: FormFieldTypeText,
					maxLength: 100,
				},
				"email": {
					fieldType: FormFieldTypeText,
					maxLength: 100,
				},
				"subscribe": {
					fieldType: FormFieldTypeCheckbox,
					required:  true,
				},
				"gender": {
					fieldType: FormFieldTypeRadio,
					required:  true,
				},
				"country": {
					fieldType:  FormFieldTypeSelect,
					hasValue:   true,
					hasOptions: true,
				},
			},
		},
		{
			name:    "text_fields",
			pdfFile: "text-fields.pdf",
			expectedFields: map[string]struct {
				fieldType  FormFieldType
				hasValue   bool
				hasOptions bool
				required   bool
				readOnly   bool
				maxLength  int
			}{
				"regularText": {
					fieldType: FormFieldTypeText,
					maxLength: 100,
				},
				"requiredField": {
					fieldType: FormFieldTypeText,
					required:  true,
					maxLength: 100,
				},
				"maxLengthField": {
					fieldType: FormFieldTypeText,
					maxLength: 10,
				},
				"comments": {
					fieldType: FormFieldTypeText,
					maxLength: 100,
				},
				"password": {
					fieldType: FormFieldTypeText,
					maxLength: 100,
				},
				"readOnly": {
					fieldType: FormFieldTypeText,
					hasValue:  true,
					readOnly:  true,
					maxLength: 100,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Build path to test PDF
			pdfPath := filepath.Join("..", "..", "..", "docs", "test-forms", tt.pdfFile)

			// Check if test file exists
			if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
				t.Skipf("Test PDF %s not found", tt.pdfFile)
			}

			// Create extractor
			extractor := NewPDFCPUFormExtractor(false)

			// Extract forms
			fields, err := extractor.ExtractFormsFromFile(pdfPath)
			require.NoError(t, err, "Form extraction should succeed")

			// Create a map of fields by name for easier testing
			fieldMap := make(map[string]FormField)
			for _, field := range fields {
				fieldMap[field.Name] = field
			}

			// Verify expected fields
			for fieldName, expected := range tt.expectedFields {
				field, found := fieldMap[fieldName]
				assert.True(t, found, "Field %s should be found", fieldName)

				if found {
					// Check field type
					assert.Equal(t, expected.fieldType, field.Type,
						"Field %s should have correct type", fieldName)

					// Check required flag
					assert.Equal(t, expected.required, field.Required,
						"Field %s required flag should match", fieldName)

					// Check read-only flag
					assert.Equal(t, expected.readOnly, field.ReadOnly,
						"Field %s read-only flag should match", fieldName)

					// Check max length for text fields
					if field.Type == FormFieldTypeText && field.Validation != nil {
						assert.Equal(t, expected.maxLength, field.Validation.MaxLength,
							"Field %s should have correct max length", fieldName)
					}

					// Check if field has value
					if expected.hasValue {
						assert.NotNil(t, field.Value, "Field %s should have a value", fieldName)
					}

					// Check if field has options
					if expected.hasOptions {
						assert.NotEmpty(t, field.Options, "Field %s should have options", fieldName)
					}

					// Verify field has bounds
					assert.NotNil(t, field.Bounds, "Field %s should have bounds", fieldName)
					if field.Bounds != nil {
						assert.Greater(t, field.Bounds.Width, 0.0, "Field width should be positive")
						assert.Greater(t, field.Bounds.Height, 0.0, "Field height should be positive")
					}

					// Verify page number
					assert.Equal(t, 1, field.Page, "Field should be on page 1")
				}
			}

			// Log summary
			t.Logf("Found %d fields in %s", len(fields), tt.pdfFile)
		})
	}
}

func TestPDFCPUFormExtractor_FieldDetails(t *testing.T) {
	// Skip if running short tests
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	pdfPath := filepath.Join("..", "..", "..", "docs", "test-forms", "basic-form.pdf")

	// Check if test file exists
	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		t.Skip("Test PDF basic-form.pdf not found")
	}

	extractor := NewPDFCPUFormExtractor(false)
	fields, err := extractor.ExtractFormsFromFile(pdfPath)
	require.NoError(t, err)
	require.NotEmpty(t, fields, "Should find form fields")

	// Test specific field properties
	for _, field := range fields {
		switch field.Name {
		case "name", "email":
			// Text fields should have appearance info
			assert.NotNil(t, field.Appearance, "Text field should have appearance")
			if field.Appearance != nil {
				assert.NotEmpty(t, field.Appearance.FontName, "Should have font name")
				assert.Greater(t, field.Appearance.FontSize, 0.0, "Should have font size")
				assert.NotEmpty(t, field.Appearance.TextColor, "Should have text color")
			}

		case "country":
			// Choice field should have options
			assert.NotEmpty(t, field.Options, "Choice field should have options")
			assert.Contains(t, field.Options, "us", "Should have US option")
			assert.Contains(t, field.Options, "ca", "Should have Canada option")
			assert.Contains(t, field.Options, "uk", "Should have UK option")

		case "subscribe":
			// Checkbox should be detected correctly
			assert.Equal(t, FormFieldTypeCheckbox, field.Type, "Subscribe should be checkbox")
			assert.False(t, field.Value.(bool), "Checkbox should be unchecked by default")
		}
	}
}

func TestPDFCPUFormExtractor_JSONOutput(t *testing.T) {
	// Skip if running short tests
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	pdfPath := filepath.Join("..", "..", "..", "docs", "test-forms", "text-fields.pdf")

	// Check if test file exists
	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		t.Skip("Test PDF text-fields.pdf not found")
	}

	extractor := NewPDFCPUFormExtractor(false)
	fields, err := extractor.ExtractFormsFromFile(pdfPath)
	require.NoError(t, err)
	require.NotEmpty(t, fields, "Should find form fields")

	// Test JSON serialization
	jsonData, err := json.MarshalIndent(fields, "", "  ")
	require.NoError(t, err, "Should serialize to JSON")

	// Verify JSON can be parsed back
	var parsedFields []FormField
	err = json.Unmarshal(jsonData, &parsedFields)
	require.NoError(t, err, "Should parse JSON back")
	assert.Equal(t, len(fields), len(parsedFields), "Should have same number of fields")

	// Log JSON for debugging (only first field to keep output manageable)
	if len(fields) > 0 {
		firstFieldJSON, _ := json.MarshalIndent(fields[0], "", "  ")
		t.Logf("Example field JSON:\n%s", string(firstFieldJSON))
	}
}

func TestPDFCPUFormExtractor_ErrorHandling(t *testing.T) {
	extractor := NewPDFCPUFormExtractor(false)

	// Test with non-existent file
	_, err := extractor.ExtractFormsFromFile("/non/existent/file.pdf")
	assert.Error(t, err, "Should error on non-existent file")
	assert.Contains(t, err.Error(), "failed to open PDF file")

	// Test with invalid PDF
	tempFile := filepath.Join(t.TempDir(), "invalid.pdf")
	err = os.WriteFile(tempFile, []byte("not a pdf"), 0644)
	require.NoError(t, err)

	_, err = extractor.ExtractFormsFromFile(tempFile)
	assert.Error(t, err, "Should error on invalid PDF")
}

func TestPDFCPUFormExtractor_NoForms(t *testing.T) {
	// Test with PDFs that don't have forms
	testPDFs := []string{
		"basic-text.pdf",
		"image-doc.pdf",
		"sample-report.pdf",
	}

	extractor := NewPDFCPUFormExtractor(false)

	for _, pdfFile := range testPDFs {
		t.Run(pdfFile, func(t *testing.T) {
			pdfPath := filepath.Join("..", "..", "..", "docs", "examples", pdfFile)

			// Check if test file exists
			if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
				t.Skipf("Test PDF %s not found", pdfFile)
			}

			fields, err := extractor.ExtractFormsFromFile(pdfPath)
			require.NoError(t, err, "Should not error on PDFs without forms")
			assert.Empty(t, fields, "Should return empty slice for PDFs without forms")
		})
	}
}

func TestPDFCPUFormExtractor_Performance(t *testing.T) {
	// Skip if running short tests
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	pdfPath := filepath.Join("..", "..", "..", "docs", "test-forms", "basic-form.pdf")

	// Check if test file exists
	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		t.Skip("Test PDF basic-form.pdf not found")
	}

	extractor := NewPDFCPUFormExtractor(false)

	// Benchmark form extraction
	result := testing.Benchmark(func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = extractor.ExtractFormsFromFile(pdfPath)
		}
	})

	t.Logf("Form extraction performance: %s per operation", result.T)
	t.Logf("Memory allocated: %d bytes per operation", result.MemBytes)
}

// TestCreateFormElement verifies the CreateFormElement helper function
func TestCreateFormElement_Integration(t *testing.T) {
	field := FormField{
		Name:     "testField",
		Type:     FormFieldTypeText,
		Value:    "test value",
		Required: true,
		Bounds: &BoundingBox{
			LowerLeft:  Coordinate{X: 100, Y: 200},
			UpperRight: Coordinate{X: 300, Y: 220},
			Width:      200,
			Height:     20,
		},
		Page: 2,
		Validation: &FieldValidation{
			MaxLength: 50,
			Required:  true,
		},
		Appearance: &FieldAppearance{
			FontName:  "Helvetica",
			FontSize:  12,
			TextColor: "rgb(0,0,0)",
		},
	}

	element := CreateFormElement(field, 2)

	// Verify element properties
	assert.Equal(t, ContentTypeForm, element.Type)
	assert.Equal(t, 2, element.PageNumber)
	assert.Equal(t, 0.5, element.Confidence) // Default confidence for forms

	// Verify form content
	formContent, ok := element.Content.(FormElement)
	require.True(t, ok, "Content should be FormElement")
	assert.Equal(t, field, formContent.Field)

	// Verify bounds
	assert.Equal(t, field.Bounds.LowerLeft, element.BoundingBox.LowerLeft)
	assert.Equal(t, field.Bounds.Width, element.BoundingBox.Width)
	assert.Equal(t, field.Bounds.Height, element.BoundingBox.Height)
}
