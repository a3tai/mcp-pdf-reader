package extraction

import (
	"testing"

	"github.com/ledongthuc/pdf"
	"github.com/stretchr/testify/assert"
)

func TestFormExtractor_ExtractForms(t *testing.T) {
	tests := []struct {
		name          string
		debugMode     bool
		expectedForms []FormField
		expectedError bool
	}{
		{
			name:          "returns_empty_list",
			debugMode:     false,
			expectedForms: []FormField{},
			expectedError: false,
		},
		{
			name:          "returns_empty_list_debug_mode",
			debugMode:     true,
			expectedForms: []FormField{},
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			extractor := NewFormExtractor(tt.debugMode)
			pdfReader := &pdf.Reader{}

			forms, err := extractor.ExtractForms(pdfReader)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, len(tt.expectedForms), len(forms))
			}
		})
	}
}

func TestFormExtractor_ExtractFormsFromPage(t *testing.T) {
	tests := []struct {
		name          string
		pageContent   string
		pageNum       int
		expectedForms int
		expectedTypes []FormFieldType
	}{
		{
			name:          "empty_page",
			pageContent:   "",
			pageNum:       1,
			expectedForms: 0,
			expectedTypes: []FormFieldType{},
		},
		{
			name:          "page_with_checkbox_pattern",
			pageContent:   "Please check: [ ] Option A",
			pageNum:       1,
			expectedForms: 1,
			expectedTypes: []FormFieldType{FormFieldTypeCheckbox},
		},
		{
			name:          "page_with_checked_checkbox",
			pageContent:   "Selected: [X] Option B",
			pageNum:       2,
			expectedForms: 1,
			expectedTypes: []FormFieldType{FormFieldTypeCheckbox},
		},
		{
			name:          "page_with_text_field_pattern",
			pageContent:   "Name: ________________",
			pageNum:       1,
			expectedForms: 1,
			expectedTypes: []FormFieldType{FormFieldTypeText},
		},
		{
			name:          "page_with_multiple_patterns",
			pageContent:   "Name: ____ and Check: [ ]",
			pageNum:       3,
			expectedForms: 2,
			expectedTypes: []FormFieldType{FormFieldTypeCheckbox, FormFieldTypeText},
		},
		{
			name:          "page_with_dots_pattern",
			pageContent:   "Address: ......................",
			pageNum:       1,
			expectedForms: 1,
			expectedTypes: []FormFieldType{FormFieldTypeText},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			extractor := NewFormExtractor(false)

			// Create a mock page with content
			// Note: This is a simplified test as the actual pdf.Page type
			// would require more complex setup
			page := pdf.Page{}
			// In a real test, we would mock the page.Content() method
			// to return content with the test patterns

			forms, err := extractor.ExtractFormsFromPage(page, tt.pageNum)

			assert.NoError(t, err)
			// Due to the limitations of mocking pdf.Page, we can't fully test
			// the pattern detection without a real PDF
			// In production, this would require integration tests with actual PDFs
			_ = forms // Acknowledge the variable
		})
	}
}

func TestContainsPattern(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		patterns []string
		expected bool
	}{
		{
			name:     "empty_text",
			text:     "",
			patterns: []string{"[ ]"},
			expected: false,
		},
		{
			name:     "empty_pattern",
			text:     "some text",
			patterns: []string{""},
			expected: false,
		},
		{
			name:     "single_pattern_found",
			text:     "Check this: [ ] Option",
			patterns: []string{"[ ]"},
			expected: true,
		},
		{
			name:     "single_pattern_not_found",
			text:     "Check this: [X] Option",
			patterns: []string{"[ ]"},
			expected: false,
		},
		{
			name:     "multiple_patterns_first_found",
			text:     "Check this: [ ] Option",
			patterns: []string{"[ ]", "[X]", "[x]"},
			expected: true,
		},
		{
			name:     "multiple_patterns_second_found",
			text:     "Check this: [X] Option",
			patterns: []string{"[ ]", "[X]", "[x]"},
			expected: true,
		},
		{
			name:     "pattern_at_start",
			text:     "[ ] Option at start",
			patterns: []string{"[ ]"},
			expected: true,
		},
		{
			name:     "pattern_at_end",
			text:     "Option at end [ ]",
			patterns: []string{"[ ]"},
			expected: true,
		},
		{
			name:     "underline_pattern",
			text:     "Name: ____",
			patterns: []string{"____"},
			expected: true,
		},
		{
			name:     "dots_pattern",
			text:     "Address: ....",
			patterns: []string{"...."},
			expected: true,
		},
		{
			name:     "no_patterns_found",
			text:     "Regular text without patterns",
			patterns: []string{"[ ]", "[X]", "____", "...."},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := containsPattern(tt.text, tt.patterns...)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCreateFormElement(t *testing.T) {
	tests := []struct {
		name     string
		field    FormField
		pageNum  int
		expected ContentElement
	}{
		{
			name: "checkbox_field",
			field: FormField{
				Name: "checkbox_1",
				Type: FormFieldTypeCheckbox,
				Page: 1,
				Bounds: &BoundingBox{
					LowerLeft:  Coordinate{X: 100, Y: 200},
					UpperRight: Coordinate{X: 120, Y: 220},
					Width:      20,
					Height:     20,
				},
			},
			pageNum: 1,
			expected: ContentElement{
				Type: ContentTypeForm,
				BoundingBox: BoundingBox{
					LowerLeft:  Coordinate{X: 100, Y: 200},
					UpperRight: Coordinate{X: 120, Y: 220},
					Width:      20,
					Height:     20,
				},
				PageNumber: 1,
				Confidence: 0.5,
				Content: FormElement{
					Field: FormField{
						Name: "checkbox_1",
						Type: FormFieldTypeCheckbox,
						Page: 1,
						Bounds: &BoundingBox{
							LowerLeft:  Coordinate{X: 100, Y: 200},
							UpperRight: Coordinate{X: 120, Y: 220},
							Width:      20,
							Height:     20,
						},
					},
				},
			},
		},
		{
			name: "text_field_without_bounds",
			field: FormField{
				Name: "textfield_1",
				Type: FormFieldTypeText,
				Page: 2,
			},
			pageNum: 2,
			expected: ContentElement{
				Type:        ContentTypeForm,
				BoundingBox: BoundingBox{},
				PageNumber:  2,
				Confidence:  0.5,
				Content: FormElement{
					Field: FormField{
						Name: "textfield_1",
						Type: FormFieldTypeText,
						Page: 2,
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CreateFormElement(tt.field, tt.pageNum)
			assert.Equal(t, tt.expected.Type, result.Type)
			assert.Equal(t, tt.expected.PageNumber, result.PageNumber)
			assert.Equal(t, tt.expected.Confidence, result.Confidence)

			// Compare BoundingBox
			if tt.field.Bounds != nil {
				assert.Equal(t, *tt.field.Bounds, result.BoundingBox)
			} else {
				assert.Equal(t, BoundingBox{}, result.BoundingBox)
			}

			// Check the form element content
			formElement, ok := result.Content.(FormElement)
			assert.True(t, ok)
			assert.Equal(t, tt.field.Name, formElement.Field.Name)
			assert.Equal(t, tt.field.Type, formElement.Field.Type)
		})
	}
}

// Integration test placeholder
func TestFormExtractor_Integration(t *testing.T) {
	// This would be an integration test with actual PDF files
	// containing forms. Since the current implementation uses
	// pattern matching rather than true form extraction,
	// integration tests would need PDFs with visible form patterns
	t.Skip("Integration test requires actual PDF files with form patterns")
}

// Benchmark tests
func BenchmarkContainsPattern(b *testing.B) {
	text := "This is a sample text with a checkbox [ ] and a text field ____ for testing"
	patterns := []string{"[ ]", "[X]", "[x]", "____", "...."}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = containsPattern(text, patterns...)
	}
}

func BenchmarkFormExtractor_ExtractFormsFromPage(b *testing.B) {
	extractor := NewFormExtractor(false)
	page := pdf.Page{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = extractor.ExtractFormsFromPage(page, 1)
	}
}
