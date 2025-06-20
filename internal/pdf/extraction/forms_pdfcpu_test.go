package extraction

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPDFCPUFormExtractor_ExtractFormsFromFile(t *testing.T) {
	tests := []struct {
		name           string
		fileName       string
		expectedFields int
		expectedTypes  map[FormFieldType]int
		checkFields    func(t *testing.T, fields []FormField)
	}{
		{
			name:           "fillable_form_pdf",
			fileName:       "fillable-form.pdf",
			expectedFields: 0, // Will be updated based on actual form
			expectedTypes: map[FormFieldType]int{
				FormFieldTypeText:     0,
				FormFieldTypeCheckbox: 0,
				FormFieldTypeRadio:    0,
				FormFieldTypeSelect:   0,
			},
			checkFields: func(t *testing.T, fields []FormField) {
				// Basic validation - will be enhanced based on actual form structure
				for _, field := range fields {
					assert.NotEmpty(t, field.Name, "Field should have a name")
					assert.NotEqual(t, FormFieldTypeUnknown, field.Type, "Field type should be detected")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Check if test file exists
			testPath := filepath.Join("..", "..", "..", "docs", "examples", tt.fileName)
			if _, err := os.Stat(testPath); os.IsNotExist(err) {
				t.Skipf("Test file %s not found", testPath)
			}

			extractor := NewPDFCPUFormExtractor(false)
			fields, err := extractor.ExtractFormsFromFile(testPath)

			require.NoError(t, err)

			// If we expect specific field count
			if tt.expectedFields > 0 {
				assert.Len(t, fields, tt.expectedFields)
			}

			// Count field types
			typeCounts := make(map[FormFieldType]int)
			for _, field := range fields {
				typeCounts[field.Type]++
			}

			// Verify expected type counts
			for fieldType, expectedCount := range tt.expectedTypes {
				if expectedCount > 0 {
					assert.Equal(t, expectedCount, typeCounts[fieldType],
						"Expected %d fields of type %s", expectedCount, fieldType)
				}
			}

			// Run custom field checks
			if tt.checkFields != nil {
				tt.checkFields(t, fields)
			}
		})
	}
}

func TestPDFCPUFormExtractor_ExtractFieldTypes(t *testing.T) {
	// Test with a mock PDF that has various field types
	// This would require creating test PDFs or mocking pdfcpu context

	extractor := NewPDFCPUFormExtractor(true) // Enable debug mode

	// For now, just verify the extractor can be created
	assert.NotNil(t, extractor)
}

func TestPDFCPUFormExtractor_FieldProperties(t *testing.T) {
	tests := []struct {
		name          string
		setupField    func() FormField
		validateField func(t *testing.T, field FormField)
	}{
		{
			name: "text_field_with_value",
			setupField: func() FormField {
				return FormField{
					Name:         "fullName",
					Type:         FormFieldTypeText,
					Value:        "John Doe",
					DefaultValue: "",
					Required:     true,
					ReadOnly:     false,
				}
			},
			validateField: func(t *testing.T, field FormField) {
				assert.Equal(t, "fullName", field.Name)
				assert.Equal(t, FormFieldTypeText, field.Type)
				assert.Equal(t, "John Doe", field.Value)
				assert.True(t, field.Required)
				assert.False(t, field.ReadOnly)
			},
		},
		{
			name: "checkbox_field",
			setupField: func() FormField {
				return FormField{
					Name:     "agreement",
					Type:     FormFieldTypeCheckbox,
					Value:    true,
					Required: false,
					ReadOnly: false,
				}
			},
			validateField: func(t *testing.T, field FormField) {
				assert.Equal(t, "agreement", field.Name)
				assert.Equal(t, FormFieldTypeCheckbox, field.Type)
				assert.Equal(t, true, field.Value)
			},
		},
		{
			name: "select_field_with_options",
			setupField: func() FormField {
				return FormField{
					Name:    "country",
					Type:    FormFieldTypeSelect,
					Value:   "USA",
					Options: []string{"USA", "Canada", "Mexico"},
				}
			},
			validateField: func(t *testing.T, field FormField) {
				assert.Equal(t, "country", field.Name)
				assert.Equal(t, FormFieldTypeSelect, field.Type)
				assert.Equal(t, "USA", field.Value)
				assert.Len(t, field.Options, 3)
				assert.Contains(t, field.Options, "USA")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			field := tt.setupField()
			tt.validateField(t, field)
		})
	}
}

func TestPDFCPUFormExtractor_Bounds(t *testing.T) {
	// Test field positioning extraction
	bounds := &BoundingBox{
		LowerLeft:  Coordinate{X: 100, Y: 200},
		UpperRight: Coordinate{X: 300, Y: 250},
		Width:      200,
		Height:     50,
	}

	field := FormField{
		Name:   "textField",
		Type:   FormFieldTypeText,
		Bounds: bounds,
		Page:   1,
	}

	assert.NotNil(t, field.Bounds)
	assert.Equal(t, 100.0, field.Bounds.LowerLeft.X)
	assert.Equal(t, 200.0, field.Bounds.LowerLeft.Y)
	assert.Equal(t, 200.0, field.Bounds.Width)
	assert.Equal(t, 50.0, field.Bounds.Height)
	assert.Equal(t, 1, field.Page)
}

func TestPDFCPUFormExtractor_Appearance(t *testing.T) {
	appearance := &FieldAppearance{
		FontName:    "Helvetica",
		FontSize:    12.0,
		TextColor:   "rgb(0,0,0)",
		BorderColor: "rgb(128,128,128)",
		BorderWidth: 1.0,
	}

	// Test parseDAString
	da := "1 0 0 rg /Helv 12 Tf"
	appearance.parseDAString(da)

	assert.Equal(t, "/Helv", appearance.FontName)
	assert.Equal(t, 12.0, appearance.FontSize)
	assert.Equal(t, "rgb(255,0,0)", appearance.TextColor)
}

func TestPDFCPUFormExtractor_Validation(t *testing.T) {
	validation := &FieldValidation{
		MaxLength: 50,
		MinLength: 3,
		Required:  true,
		Pattern:   "[A-Za-z]+",
	}

	field := FormField{
		Name:       "username",
		Type:       FormFieldTypeText,
		Validation: validation,
	}

	assert.NotNil(t, field.Validation)
	assert.Equal(t, 50, field.Validation.MaxLength)
	assert.Equal(t, 3, field.Validation.MinLength)
	assert.True(t, field.Validation.Required)
	assert.Equal(t, "[A-Za-z]+", field.Validation.Pattern)
}

func TestPDFCPUFormExtractor_ErrorCases(t *testing.T) {
	extractor := NewPDFCPUFormExtractor(false)

	// Test with non-existent file
	_, err := extractor.ExtractFormsFromFile("/non/existent/file.pdf")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to open PDF file")

	// Test with invalid PDF
	tempFile := filepath.Join(t.TempDir(), "invalid.pdf")
	err = os.WriteFile(tempFile, []byte("not a pdf"), 0644)
	require.NoError(t, err)

	_, err = extractor.ExtractFormsFromFile(tempFile)
	assert.Error(t, err)
}

func TestPDFCPUFormExtractor_Integration(t *testing.T) {
	// Test integration with FormExtractor
	formExtractor := NewFormExtractor(false)

	// Test file-based extraction with a PDF that has actual form fields
	testPath := filepath.Join("..", "..", "..", "docs", "test-forms", "basic-form.pdf")
	if _, err := os.Stat(testPath); os.IsNotExist(err) {
		// Fallback to fillable-form.pdf which doesn't have real form fields
		testPath = filepath.Join("..", "..", "..", "docs", "examples", "fillable-form.pdf")
		if _, err := os.Stat(testPath); os.IsNotExist(err) {
			t.Skip("Test file not found")
		}
		// This file doesn't have actual AcroForm fields, so we expect empty results
		fields, err := formExtractor.ExtractFormsFromFile(testPath)
		assert.NoError(t, err)
		assert.NotNil(t, fields)
		assert.Empty(t, fields, "fillable-form.pdf has no AcroForm fields")
		return
	}

	fields, err := formExtractor.ExtractFormsFromFile(testPath)
	assert.NoError(t, err)
	assert.NotNil(t, fields)
	assert.NotEmpty(t, fields, "basic-form.pdf should have form fields")

	// Verify that we're using the pdfcpu implementation
	// by checking that we get actual form fields (not heuristic-based)
	for _, field := range fields {
		// Real form fields should have proper names (not generated ones)
		assert.NotContains(t, field.Name, "field_")
		assert.NotContains(t, field.Name, "checkbox_")
		assert.NotContains(t, field.Name, "textfield_")
	}
}

func TestPDFCPUCreateFormElement(t *testing.T) {
	field := FormField{
		Name:     "email",
		Type:     FormFieldTypeText,
		Value:    "test@example.com",
		Required: true,
		Bounds: &BoundingBox{
			LowerLeft:  Coordinate{X: 100, Y: 200},
			UpperRight: Coordinate{X: 300, Y: 220},
			Width:      200,
			Height:     20,
		},
		Page: 2,
	}

	element := CreateFormElement(field, 2)

	assert.Equal(t, ContentTypeForm, element.Type)
	assert.Equal(t, 2, element.PageNumber)
	assert.Equal(t, 0.5, element.Confidence) // Default confidence

	formContent, ok := element.Content.(FormElement)
	assert.True(t, ok)
	assert.Equal(t, field, formContent.Field)

	assert.Equal(t, field.Bounds.LowerLeft, element.BoundingBox.LowerLeft)
	assert.Equal(t, field.Bounds.Width, element.BoundingBox.Width)
}

// Benchmark tests
func BenchmarkPDFCPUFormExtractor_ExtractFormsFromFile(b *testing.B) {
	testPath := filepath.Join("..", "..", "..", "docs", "examples", "fillable-form.pdf")
	if _, err := os.Stat(testPath); os.IsNotExist(err) {
		b.Skip("Test file not found")
	}

	extractor := NewPDFCPUFormExtractor(false)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = extractor.ExtractFormsFromFile(testPath)
	}
}
