package extraction

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

// PDFCPUFormExtractor implements form extraction using the pdfcpu library
type PDFCPUFormExtractor struct {
	debugMode bool
}

// NewPDFCPUFormExtractor creates a new form extractor using pdfcpu
func NewPDFCPUFormExtractor(debugMode bool) *PDFCPUFormExtractor {
	return &PDFCPUFormExtractor{
		debugMode: debugMode,
	}
}

// ExtractFormsFromFile extracts all form fields from a PDF file
func (fe *PDFCPUFormExtractor) ExtractFormsFromFile(filePath string) ([]FormField, error) {
	if fe.debugMode {
		fmt.Printf("Extracting forms from: %s using pdfcpu\n", filePath)
	}

	// Open the PDF file
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open PDF file: %w", err)
	}
	defer file.Close()

	// Read configuration
	conf := model.NewDefaultConfiguration()
	conf.ValidationMode = model.ValidationRelaxed

	// Create context from reader
	ctx, err := api.ReadContext(file, conf)
	if err != nil {
		return nil, fmt.Errorf("failed to read PDF context: %w", err)
	}

	// Ensure the context is valid
	if err := ctx.EnsurePageCount(); err != nil {
		return nil, fmt.Errorf("failed to ensure page count: %w", err)
	}

	// Extract forms from the context
	return fe.extractFormsFromContext(ctx)
}

// ExtractFormsFromReader extracts forms from an io.Reader
func (fe *PDFCPUFormExtractor) ExtractFormsFromReader(reader io.ReadSeeker) ([]FormField, error) {
	// Read configuration
	conf := model.NewDefaultConfiguration()
	conf.ValidationMode = model.ValidationRelaxed

	// Create context from reader
	ctx, err := api.ReadContext(reader, conf)
	if err != nil {
		return nil, fmt.Errorf("failed to read PDF context: %w", err)
	}

	// Ensure the context is valid
	if err := ctx.EnsurePageCount(); err != nil {
		return nil, fmt.Errorf("failed to ensure page count: %w", err)
	}

	return fe.extractFormsFromContext(ctx)
}

// extractFormsFromContext extracts form fields from a pdfcpu context
func (fe *PDFCPUFormExtractor) extractFormsFromContext(ctx *model.Context) ([]FormField, error) {
	var forms []FormField

	// Get the AcroForm dictionary from the catalog
	rootDict, err := ctx.Catalog()
	if err != nil {
		return nil, fmt.Errorf("failed to get catalog: %w", err)
	}

	acroFormObj, found := rootDict.Find("AcroForm")
	if !found {
		if fe.debugMode {
			fmt.Println("No AcroForm dictionary found in document")
		}
		return forms, nil
	}

	// Resolve indirect reference if needed
	acroFormDict, err := ctx.DereferenceDict(acroFormObj)
	if err != nil {
		return nil, fmt.Errorf("failed to dereference AcroForm: %w", err)
	}

	if acroFormDict == nil {
		return forms, nil
	}

	// Get the Fields array
	fieldsObj, found := acroFormDict.Find("Fields")
	if !found {
		if fe.debugMode {
			fmt.Println("No Fields array found in AcroForm")
		}
		return forms, nil
	}

	fieldsArray, err := ctx.DereferenceArray(fieldsObj)
	if err != nil {
		return nil, fmt.Errorf("failed to dereference Fields array: %w", err)
	}

	// Process each field
	for i, fieldRef := range fieldsArray {
		field, err := fe.processField(ctx, fieldRef, i)
		if err != nil {
			if fe.debugMode {
				fmt.Printf("Error processing field %d: %v\n", i, err)
			}
			continue
		}
		if field != nil {
			forms = append(forms, *field)
		}
	}

	return forms, nil
}

// processField processes a single field dictionary
func (fe *PDFCPUFormExtractor) processField(ctx *model.Context, fieldObj types.Object, index int) (*FormField, error) {
	fieldDict, err := ctx.DereferenceDict(fieldObj)
	if err != nil {
		return nil, fmt.Errorf("failed to dereference field: %w", err)
	}

	if fieldDict == nil {
		return nil, nil
	}

	field := &FormField{}

	// Extract field name (T entry)
	if nameObj, found := fieldDict.Find("T"); found {
		if name, err := ctx.DereferenceStringOrHexLiteral(nameObj, model.V10, nil); err == nil {
			field.Name = name
		}
	}

	// If no name, generate one
	if field.Name == "" {
		field.Name = fmt.Sprintf("field_%d", index)
	}

	// Extract field type (FT entry)
	fieldType := fe.extractFieldType(ctx, fieldDict)
	field.Type = fieldType

	// Extract field value (V entry)
	if valueObj, found := fieldDict.Find("V"); found {
		field.Value = fe.extractFieldValue(ctx, valueObj, fieldType)
	}

	// Extract default value (DV entry)
	if defaultObj, found := fieldDict.Find("DV"); found {
		field.DefaultValue = fe.extractFieldValue(ctx, defaultObj, fieldType)
	}

	// Extract field flags (Ff entry)
	if flagsObj, found := fieldDict.Find("Ff"); found {
		if flags, err := ctx.DereferenceInteger(flagsObj); err == nil && flags != nil {
			flagValue := *flags
			field.ReadOnly = (flagValue & 1) != 0 // Bit 1
			field.Required = (flagValue & 2) != 0 // Bit 2
		}
	}

	// Extract options for choice fields
	if fieldType == FormFieldTypeSelect || fieldType == FormFieldTypeRadio {
		field.Options = fe.extractFieldOptions(ctx, fieldDict)
	}

	// Extract field bounds from widget annotation
	field.Bounds, field.Page = fe.extractFieldBounds(ctx, fieldDict)

	// Extract appearance properties
	field.Appearance = fe.extractFieldAppearance(ctx, fieldDict)

	// Extract validation rules
	field.Validation = fe.extractFieldValidation(ctx, fieldDict)

	if fe.debugMode {
		fmt.Printf("Extracted field: %s (type: %s)\n", field.Name, field.Type)
	}

	return field, nil
}

// extractFieldType determines the field type from the FT entry
func (fe *PDFCPUFormExtractor) extractFieldType(ctx *model.Context, fieldDict types.Dict) FormFieldType {
	ftObj, found := fieldDict.Find("FT")
	if !found {
		// Check parent for inherited FT
		if parentObj, found := fieldDict.Find("Parent"); found {
			if parentDict, err := ctx.DereferenceDict(parentObj); err == nil && parentDict != nil {
				return fe.extractFieldType(ctx, parentDict)
			}
		}
		return FormFieldTypeUnknown
	}

	ftName, err := ctx.DereferenceName(ftObj, model.V10, nil)
	if err != nil {
		return FormFieldTypeUnknown
	}

	switch ftName {
	case "Btn":
		// Check if it's a checkbox, radio, or button
		if flagsObj, found := fieldDict.Find("Ff"); found {
			if flags, err := ctx.DereferenceInteger(flagsObj); err == nil && flags != nil {
				flagValue := *flags
				if (flagValue & (1 << 15)) != 0 { // Bit 16: Radio
					return FormFieldTypeRadio
				} else if (flagValue & (1 << 16)) != 0 { // Bit 17: Pushbutton
					return FormFieldTypeButton
				}
			}
		}
		return FormFieldTypeCheckbox
	case "Tx":
		return FormFieldTypeText
	case "Ch":
		return FormFieldTypeSelect
	case "Sig":
		return FormFieldTypeSignature
	default:
		return FormFieldTypeUnknown
	}
}

// extractFieldValue extracts the value based on field type
func (fe *PDFCPUFormExtractor) extractFieldValue(ctx *model.Context, valueObj types.Object, fieldType FormFieldType) interface{} {
	switch fieldType {
	case FormFieldTypeText:
		if val, err := ctx.DereferenceStringOrHexLiteral(valueObj, model.V10, nil); err == nil {
			return val
		}
	case FormFieldTypeCheckbox:
		if name, err := ctx.DereferenceName(valueObj, model.V10, nil); err == nil {
			return name == "Yes" || name == "On"
		}
	case FormFieldTypeRadio:
		if name, err := ctx.DereferenceName(valueObj, model.V10, nil); err == nil {
			return name
		}
	case FormFieldTypeSelect:
		// Can be string or array of strings
		if val, err := ctx.DereferenceStringOrHexLiteral(valueObj, model.V10, nil); err == nil {
			return val
		}
		if arr, err := ctx.DereferenceArray(valueObj); err == nil {
			var values []string
			for _, item := range arr {
				if str, err := ctx.DereferenceStringOrHexLiteral(item, model.V10, nil); err == nil {
					values = append(values, str)
				}
			}
			return values
		}
	}
	return nil
}

// extractFieldOptions extracts options for choice fields
func (fe *PDFCPUFormExtractor) extractFieldOptions(ctx *model.Context, fieldDict types.Dict) []string {
	var options []string

	optObj, found := fieldDict.Find("Opt")
	if !found {
		return options
	}

	optArray, err := ctx.DereferenceArray(optObj)
	if err != nil {
		return options
	}

	for _, opt := range optArray {
		// Options can be strings or arrays of [export_value, display_value]
		if str, err := ctx.DereferenceStringOrHexLiteral(opt, model.V10, nil); err == nil {
			options = append(options, str)
		} else if arr, err := ctx.DereferenceArray(opt); err == nil && len(arr) >= 2 {
			// Use display value (second element)
			if displayVal, err := ctx.DereferenceStringOrHexLiteral(arr[1], model.V10, nil); err == nil {
				options = append(options, displayVal)
			}
		}
	}

	return options
}

// extractFieldBounds extracts the field position from widget annotation
func (fe *PDFCPUFormExtractor) extractFieldBounds(ctx *model.Context, fieldDict types.Dict) (*BoundingBox, int) {
	// Try to find Rect in the field dictionary (for merged widget annotation)
	if rectObj, found := fieldDict.Find("Rect"); found {
		if rect, page := fe.parseRectAndPage(ctx, rectObj, fieldDict); rect != nil {
			return rect, page
		}
	}

	// Look for Kids array (for fields with separate widget annotations)
	if kidsObj, found := fieldDict.Find("Kids"); found {
		if kidsArray, err := ctx.DereferenceArray(kidsObj); err == nil && len(kidsArray) > 0 {
			// Get bounds from first widget annotation
			if widgetDict, err := ctx.DereferenceDict(kidsArray[0]); err == nil && widgetDict != nil {
				if rectObj, found := widgetDict.Find("Rect"); found {
					if rect, page := fe.parseRectAndPage(ctx, rectObj, widgetDict); rect != nil {
						return rect, page
					}
				}
			}
		}
	}

	return nil, 0
}

// parseRectAndPage parses rectangle coordinates and page number
func (fe *PDFCPUFormExtractor) parseRectAndPage(ctx *model.Context, rectObj types.Object, annotDict types.Dict) (*BoundingBox, int) {
	rectArray, err := ctx.DereferenceArray(rectObj)
	if err != nil || len(rectArray) != 4 {
		return nil, 0
	}

	// Parse coordinates
	coords := make([]float64, 4)
	for i, coord := range rectArray {
		if f, err := ctx.DereferenceNumber(coord); err == nil {
			coords[i] = f
		}
	}

	bounds := &BoundingBox{
		LowerLeft:  Coordinate{X: coords[0], Y: coords[1]},
		UpperRight: Coordinate{X: coords[2], Y: coords[3]},
		Width:      coords[2] - coords[0],
		Height:     coords[3] - coords[1],
	}

	// Get page number
	// TODO: Implement proper page detection from annotation reference
	// For now, default to page 1 as form fields are often on the first page
	pageNum := 1

	return bounds, pageNum
}

// extractFieldAppearance extracts appearance properties
func (fe *PDFCPUFormExtractor) extractFieldAppearance(ctx *model.Context, fieldDict types.Dict) *FieldAppearance {
	appearance := &FieldAppearance{}

	// Extract default appearance string (DA)
	if daObj, found := fieldDict.Find("DA"); found {
		if daStr, err := ctx.DereferenceStringOrHexLiteral(daObj, model.V10, nil); err == nil {
			// Parse DA string for font and color information
			appearance.parseDAString(daStr)
		}
	}

	// Extract border style
	if bsObj, found := fieldDict.Find("BS"); found {
		if bsDict, err := ctx.DereferenceDict(bsObj); err == nil && bsDict != nil {
			if wObj, found := bsDict.Find("W"); found {
				if width, err := ctx.DereferenceNumber(wObj); err == nil {
					appearance.BorderWidth = width
				}
			}
		}
	}

	return appearance
}

// parseDAString parses the default appearance string
func (fa *FieldAppearance) parseDAString(da string) {
	// Simple parsing of common appearance commands
	parts := strings.Fields(da)
	for i := 0; i < len(parts); i++ {
		switch parts[i] {
		case "Tf": // Font
			if i >= 2 {
				fa.FontName = parts[i-2]
				if size, err := parseFloat(parts[i-1]); err == nil {
					fa.FontSize = size
				}
			}
		case "rg", "RG": // RGB color
			if i >= 3 {
				r, _ := parseFloat(parts[i-3])
				g, _ := parseFloat(parts[i-2])
				b, _ := parseFloat(parts[i-1])
				fa.TextColor = fmt.Sprintf("rgb(%.0f,%.0f,%.0f)", r*255, g*255, b*255)
			}
		case "g", "G": // Gray color
			if i >= 1 {
				gray, _ := parseFloat(parts[i-1])
				fa.TextColor = fmt.Sprintf("gray(%.0f)", gray*255)
			}
		}
	}
}

// extractFieldValidation extracts validation rules
func (fe *PDFCPUFormExtractor) extractFieldValidation(ctx *model.Context, fieldDict types.Dict) *FieldValidation {
	validation := &FieldValidation{}

	// Check field flags for required
	if flagsObj, found := fieldDict.Find("Ff"); found {
		if flags, err := ctx.DereferenceInteger(flagsObj); err == nil && flags != nil {
			flagValue := *flags
			validation.Required = (flagValue & 2) != 0 // Bit 2
		}
	}

	// Extract max length for text fields
	if maxLenObj, found := fieldDict.Find("MaxLen"); found {
		if maxLen, err := ctx.DereferenceInteger(maxLenObj); err == nil && maxLen != nil {
			validation.MaxLength = int(*maxLen)
		}
	}

	// Additional validation from JavaScript actions could be extracted here
	// but requires JavaScript parsing which is complex

	return validation
}

// parseFloat is a helper to parse float from string
func parseFloat(s string) (float64, error) {
	var f float64
	_, err := fmt.Sscanf(s, "%f", &f)
	return f, err
}
