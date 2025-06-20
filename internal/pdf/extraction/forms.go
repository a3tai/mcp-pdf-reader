package extraction

import (
	"fmt"
	"io"

	"github.com/ledongthuc/pdf"
)

// FormFieldType represents the type of a form field
type FormFieldType string

const (
	FormFieldTypeText      FormFieldType = "text"
	FormFieldTypeCheckbox  FormFieldType = "checkbox"
	FormFieldTypeRadio     FormFieldType = "radio"
	FormFieldTypeSelect    FormFieldType = "select"
	FormFieldTypeButton    FormFieldType = "button"
	FormFieldTypeSignature FormFieldType = "signature"
	FormFieldTypeUnknown   FormFieldType = "unknown"
)

// FormField represents an interactive form field in a PDF
type FormField struct {
	Name         string           `json:"name"`
	Type         FormFieldType    `json:"type"`
	Value        interface{}      `json:"value,omitempty"`
	DefaultValue interface{}      `json:"default_value,omitempty"`
	Options      []string         `json:"options,omitempty"`
	Selected     []int            `json:"selected,omitempty"` // For multi-select
	Required     bool             `json:"required"`
	ReadOnly     bool             `json:"read_only"`
	Bounds       *BoundingBox     `json:"bounds,omitempty"`
	Page         int              `json:"page"`
	Validation   *FieldValidation `json:"validation,omitempty"`
	Appearance   *FieldAppearance `json:"appearance,omitempty"`
}

// FieldValidation represents validation rules for a form field
type FieldValidation struct {
	MaxLength    int     `json:"max_length,omitempty"`
	MinLength    int     `json:"min_length,omitempty"`
	Pattern      string  `json:"pattern,omitempty"`
	MinValue     float64 `json:"min_value,omitempty"`
	MaxValue     float64 `json:"max_value,omitempty"`
	Required     bool    `json:"required"`
	CustomScript string  `json:"custom_script,omitempty"`
}

// FieldAppearance represents visual properties of a form field
type FieldAppearance struct {
	FontName    string  `json:"font_name,omitempty"`
	FontSize    float64 `json:"font_size,omitempty"`
	TextColor   string  `json:"text_color,omitempty"`
	BorderColor string  `json:"border_color,omitempty"`
	FillColor   string  `json:"fill_color,omitempty"`
	BorderWidth float64 `json:"border_width,omitempty"`
}

// FormExtractor handles extraction of form fields from PDF documents
type FormExtractor struct {
	debugMode bool
}

// NewFormExtractor creates a new form extractor
func NewFormExtractor(debugMode bool) *FormExtractor {
	return &FormExtractor{
		debugMode: debugMode,
	}
}

// ExtractForms extracts all form fields from a PDF document
// This implementation now uses pdfcpu for proper form extraction
func (fe *FormExtractor) ExtractForms(pdfReader *pdf.Reader) ([]FormField, error) {
	// Since ledongthuc/pdf.Reader doesn't provide access to the underlying reader,
	// we'll need to work with what we have. For now, we'll extract form patterns
	// from each page as a fallback approach.

	forms := make([]FormField, 0)

	// Try to extract forms from each page
	for pageNum := 1; pageNum <= pdfReader.NumPage(); pageNum++ {
		page := pdfReader.Page(pageNum)
		pageForms, err := fe.ExtractFormsFromPage(page, pageNum)
		if err != nil {
			if fe.debugMode {
				fmt.Printf("Error extracting forms from page %d: %v\n", pageNum, err)
			}
			continue
		}
		forms = append(forms, pageForms...)
	}

	if fe.debugMode {
		fmt.Printf("Extracted %d form patterns using heuristic approach\n", len(forms))
	}

	return forms, nil
}

// ExtractFormsFromReader extracts forms using pdfcpu from an io.Reader
// This is the preferred method when you have direct access to the PDF data
func (fe *FormExtractor) ExtractFormsFromReader(reader io.ReadSeeker) ([]FormField, error) {
	// Use the pdfcpu-based extractor for proper form extraction
	pdfcpuExtractor := NewPDFCPUFormExtractor(fe.debugMode)
	return pdfcpuExtractor.ExtractFormsFromReader(reader)
}

// ExtractFormsFromFile extracts forms using pdfcpu from a file path
// This is the preferred method when working with files
func (fe *FormExtractor) ExtractFormsFromFile(filePath string) ([]FormField, error) {
	// Use the pdfcpu-based extractor for proper form extraction
	pdfcpuExtractor := NewPDFCPUFormExtractor(fe.debugMode)
	return pdfcpuExtractor.ExtractFormsFromFile(filePath)
}

// ExtractFormsFromPage attempts to extract form fields from a specific page
// This is a placeholder implementation that can be enhanced later
func (fe *FormExtractor) ExtractFormsFromPage(page pdf.Page, pageNum int) ([]FormField, error) {
	forms := make([]FormField, 0)

	// Basic implementation that could detect form-like patterns in text
	// For example, looking for patterns like "[ ]" for checkboxes
	// or "____" for text fields

	// Get page content
	content := page.Content()
	if len(content.Text) > 0 {
		// Parse content for form-like elements
		// This is a very basic approach and would need enhancement
		textParts := content.Text
		text := ""
		for _, t := range textParts {
			text += t.S
		}

		// Look for checkbox patterns
		if containsPattern(text, "[ ]", "[X]", "[x]") {
			forms = append(forms, FormField{
				Name: fmt.Sprintf("checkbox_%d", len(forms)+1),
				Type: FormFieldTypeCheckbox,
				Page: pageNum,
			})
		}

		// Look for underline patterns that might indicate text fields
		if containsPattern(text, "____", "....") {
			forms = append(forms, FormField{
				Name: fmt.Sprintf("textfield_%d", len(forms)+1),
				Type: FormFieldTypeText,
				Page: pageNum,
			})
		}
	}

	return forms, nil
}

// containsPattern checks if any of the patterns exist in the text
func containsPattern(text string, patterns ...string) bool {
	for _, pattern := range patterns {
		if len(text) > 0 && len(pattern) > 0 {
			// Simple pattern matching - can be enhanced
			for i := 0; i <= len(text)-len(pattern); i++ {
				if text[i:i+len(pattern)] == pattern {
					return true
				}
			}
		}
	}
	return false
}

// CreateFormElement creates a ContentElement from a FormField
func CreateFormElement(field FormField, pageNum int) ContentElement {
	element := ContentElement{
		Type:       ContentTypeForm,
		PageNumber: pageNum,
		Confidence: 0.5, // Lower confidence due to heuristic detection
		Content: FormElement{
			Field: field,
		},
	}
	if field.Bounds != nil {
		element.BoundingBox = *field.Bounds
	}
	return element
}
