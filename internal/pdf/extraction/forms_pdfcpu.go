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

// FormDebugger provides detailed debugging for form extraction
type FormDebugger struct {
	enabled bool
}

func (fd *FormDebugger) log(format string, args ...interface{}) {
	if fd.enabled {
		fmt.Printf("[DEBUG] "+format+"\n", args...)
	}
}

func (fd *FormDebugger) TraceAcroFormLookup(catalog types.Dict) {
	if !fd.enabled {
		return
	}
	fd.log("Searching for AcroForm in catalog...")
	if acroForm, exists := catalog.Find("AcroForm"); exists {
		fd.log("AcroForm found: %+v", acroForm)
	} else {
		fd.log("WARNING: No AcroForm entry in catalog")
		keys := make([]string, 0, len(catalog))
		for key := range catalog {
			keys = append(keys, key)
		}
		fd.log("Catalog keys: %v", keys)
	}
}

func (fd *FormDebugger) TraceFieldParsing(field types.Dict, fieldName string) {
	if !fd.enabled {
		return
	}

	ftObj, _ := field.Find("FT")
	tObj, _ := field.Find("T")
	vObj, _ := field.Find("V")

	fd.log("Parsing field: Name=%s, FT=%v, T=%v, V=%v", fieldName, ftObj, tObj, vObj)

	// Check for widget annotations
	if kids, hasKids := field.Find("Kids"); hasKids {
		fd.log("Field has children: %v", kids)
	}
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
	debugger := &FormDebugger{enabled: fe.debugMode}

	// Get the AcroForm dictionary from the catalog
	rootDict, err := ctx.Catalog()
	if err != nil {
		return nil, fmt.Errorf("failed to get catalog: %w", err)
	}

	debugger.TraceAcroFormLookup(rootDict)

	// Fix 1: Try multiple methods to find forms
	forms = append(forms, fe.extractAcroForms(ctx, debugger)...)

	// Fix 2: Check for XFA forms (common in tax documents)
	if len(forms) == 0 {
		forms = append(forms, fe.extractXFAForms(ctx, debugger)...)
	}

	// Fix 3: Check for fields in page annotations as fallback
	if len(forms) == 0 {
		forms = append(forms, fe.searchPageAnnotations(ctx, debugger)...)
	}

	if fe.debugMode {
		fmt.Printf("Total forms extracted: %d\n", len(forms))
	}

	return forms, nil
}

// extractAcroForms extracts standard AcroForm fields with enhanced error handling
func (fe *PDFCPUFormExtractor) extractAcroForms(ctx *model.Context, debugger *FormDebugger) []FormField {
	var forms []FormField

	rootDict, err := ctx.Catalog()
	if err != nil {
		debugger.log("Failed to get catalog: %v", err)
		return forms
	}

	acroFormObj, found := rootDict.Find("AcroForm")
	if !found {
		debugger.log("No AcroForm dictionary found in document")
		return forms
	}

	// Fix 1: Handle indirect references properly
	acroFormDict, err := ctx.DereferenceDict(acroFormObj)
	if err != nil {
		debugger.log("Failed to dereference AcroForm: %v", err)
		// Try alternative dereferencing for indirect references
		if indRef, isIndirect := acroFormObj.(types.IndirectRef); isIndirect {
			debugger.log("Attempting to resolve indirect AcroForm reference: %v", indRef)
			if obj, err := ctx.Dereference(indRef); err == nil {
				if dict, ok := obj.(types.Dict); ok {
					acroFormDict = dict
					debugger.log("Successfully resolved indirect AcroForm reference")
				}
			}
		}
		if acroFormDict == nil {
			return forms
		}
	}

	if acroFormDict == nil {
		debugger.log("AcroForm dictionary is nil after dereferencing")
		return forms
	}

	// Get the Fields array
	fieldsObj, found := acroFormDict.Find("Fields")
	if !found {
		debugger.log("No Fields array found in AcroForm")
		return forms
	}

	fieldsArray, err := ctx.DereferenceArray(fieldsObj)
	if err != nil {
		debugger.log("Failed to dereference Fields array: %v", err)
		return forms
	}

	debugger.log("Found %d fields in AcroForm", len(fieldsArray))

	// Process each field with enhanced error handling
	for i, fieldRef := range fieldsArray {
		field, err := fe.processFieldWithDebugging(ctx, fieldRef, i, debugger)
		if err != nil {
			debugger.log("Error processing field %d: %v", i, err)
			continue
		}
		if field != nil {
			forms = append(forms, *field)
		}
	}

	return forms
}

// extractXFAForms handles XFA (XML Forms Architecture) forms common in tax documents
func (fe *PDFCPUFormExtractor) extractXFAForms(ctx *model.Context, debugger *FormDebugger) []FormField {
	var forms []FormField

	rootDict, err := ctx.Catalog()
	if err != nil {
		return forms
	}

	// Check for XFA in catalog
	if xfaObj, found := rootDict.Find("XFA"); found {
		debugger.log("XFA forms detected in catalog")
		return fe.parseXFAForms(ctx, xfaObj, debugger)
	}

	// Check for XFA in AcroForm
	if acroFormObj, found := rootDict.Find("AcroForm"); found {
		if acroFormDict, err := ctx.DereferenceDict(acroFormObj); err == nil && acroFormDict != nil {
			if xfaObj, found := acroFormDict.Find("XFA"); found {
				debugger.log("XFA forms detected in AcroForm")
				return fe.parseXFAForms(ctx, xfaObj, debugger)
			}
		}
	}

	return forms
}

// parseXFAForms parses XFA form data (placeholder for full implementation)
func (fe *PDFCPUFormExtractor) parseXFAForms(ctx *model.Context, xfaObj types.Object, debugger *FormDebugger) []FormField {
	// XFA forms require XML parsing which is complex
	// For now, we'll create placeholder fields to indicate XFA presence
	debugger.log("XFA form parsing not fully implemented - creating placeholder")

	var forms []FormField

	// Try to determine if it's an array or stream
	if xfaArray, err := ctx.DereferenceArray(xfaObj); err == nil {
		debugger.log("XFA is an array with %d elements", len(xfaArray))
		// Create placeholder fields based on array elements
		for i := 0; i < len(xfaArray); i += 2 { // XFA arrays come in pairs
			field := FormField{
				Name: fmt.Sprintf("xfa_field_%d", i/2),
				Type: FormFieldTypeText,
				Page: 1,
			}
			forms = append(forms, field)
		}
	} else if _, _, err := ctx.DereferenceStreamDict(xfaObj); err == nil {
		debugger.log("XFA is a stream")
		// Create a placeholder field for XFA stream
		field := FormField{
			Name: "xfa_stream_form",
			Type: FormFieldTypeText,
			Page: 1,
		}
		forms = append(forms, field)
	}

	return forms
}

// searchPageAnnotations searches for form fields in page annotations
func (fe *PDFCPUFormExtractor) searchPageAnnotations(ctx *model.Context, debugger *FormDebugger) []FormField {
	var forms []FormField

	debugger.log("Searching page annotations for form fields...")

	for pageNum := 1; pageNum <= ctx.PageCount; pageNum++ {
		pageDict, _, _, err := ctx.PageDict(pageNum, false)
		if err != nil {
			debugger.log("Error getting page %d: %v", pageNum, err)
			continue
		}

		if annotsObj, found := pageDict.Find("Annots"); found {
			if annotsArray, err := ctx.DereferenceArray(annotsObj); err == nil {
				debugger.log("Found %d annotations on page %d", len(annotsArray), pageNum)

				for i, annotObj := range annotsArray {
					if field := fe.parseWidgetAnnotation(ctx, annotObj, pageNum, i, debugger); field != nil {
						forms = append(forms, *field)
					}
				}
			}
		}
	}

	debugger.log("Found %d form fields in page annotations", len(forms))
	return forms
}

// parseWidgetAnnotation parses a widget annotation as a form field
func (fe *PDFCPUFormExtractor) parseWidgetAnnotation(ctx *model.Context, annotObj types.Object, pageNum, annotIndex int, debugger *FormDebugger) *FormField {
	annotDict, err := ctx.DereferenceDict(annotObj)
	if err != nil {
		return nil
	}

	if annotDict == nil {
		return nil
	}

	// Check if it's a widget annotation
	if subtypeObj, found := annotDict.Find("Subtype"); found {
		if subtype, err := ctx.DereferenceName(subtypeObj, model.V10, nil); err == nil {
			if subtype == "Widget" {
				debugger.log("Found widget annotation on page %d", pageNum)

				field := &FormField{
					Page: pageNum,
				}

				// Extract field name
				if nameObj, found := annotDict.Find("T"); found {
					if name, err := ctx.DereferenceStringOrHexLiteral(nameObj, model.V10, nil); err == nil {
						field.Name = name
					}
				}
				if field.Name == "" {
					field.Name = fmt.Sprintf("widget_field_%d_%d", pageNum, annotIndex)
				}

				// Extract field type
				field.Type = fe.extractFieldType(ctx, annotDict)

				// Extract bounds
				field.Bounds, _ = fe.extractFieldBounds(ctx, annotDict)

				debugger.log("Extracted widget field: %s (%s)", field.Name, field.Type)
				return field
			}
		}
	}

	return nil
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

// processFieldWithDebugging processes a field with enhanced debugging
func (fe *PDFCPUFormExtractor) processFieldWithDebugging(ctx *model.Context, fieldObj types.Object, index int, debugger *FormDebugger) (*FormField, error) {
	// Enhanced field processing with better error handling for indirect references
	fieldDict, err := ctx.DereferenceDict(fieldObj)
	if err != nil {
		debugger.log("Failed to dereference field %d: %v", index, err)
		// Try alternative dereferencing for indirect references
		if indRef, isIndirect := fieldObj.(types.IndirectRef); isIndirect {
			debugger.log("Attempting to resolve indirect field reference: %v", indRef)
			if obj, err := ctx.Dereference(indRef); err == nil {
				if dict, ok := obj.(types.Dict); ok {
					fieldDict = dict
					debugger.log("Successfully resolved indirect field reference")
				}
			}
		}
		if fieldDict == nil {
			return nil, fmt.Errorf("failed to dereference field: %w", err)
		}
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

	debugger.TraceFieldParsing(fieldDict, field.Name)

	// Extract field type (FT entry) with recursive search for inherited types
	fieldType := fe.extractFieldTypeWithInheritance(ctx, fieldDict, debugger)
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

	debugger.log("Successfully extracted field: %s (type: %s)", field.Name, field.Type)

	return field, nil
}

// extractFieldTypeWithInheritance determines field type with inheritance support
func (fe *PDFCPUFormExtractor) extractFieldTypeWithInheritance(ctx *model.Context, fieldDict types.Dict, debugger *FormDebugger) FormFieldType {
	// Try direct FT first
	if fieldType := fe.extractFieldType(ctx, fieldDict); fieldType != FormFieldTypeUnknown {
		return fieldType
	}

	// Recursive search through parent hierarchy for inherited FT
	debugger.log("Field type not found, searching parent hierarchy")
	return fe.recursiveFieldTypeSearch(ctx, fieldDict, debugger, 0)
}

// recursiveFieldTypeSearch searches parent hierarchy for field type
func (fe *PDFCPUFormExtractor) recursiveFieldTypeSearch(ctx *model.Context, fieldDict types.Dict, debugger *FormDebugger, depth int) FormFieldType {
	if depth > 5 { // Prevent infinite recursion
		debugger.log("Maximum recursion depth reached in field type search")
		return FormFieldTypeUnknown
	}

	// Check for Parent reference
	if parentObj, found := fieldDict.Find("Parent"); found {
		if parentDict, err := ctx.DereferenceDict(parentObj); err == nil && parentDict != nil {
			debugger.log("Checking parent at depth %d for field type", depth)
			if fieldType := fe.extractFieldType(ctx, parentDict); fieldType != FormFieldTypeUnknown {
				debugger.log("Found inherited field type: %s", fieldType)
				return fieldType
			}
			// Continue searching up the hierarchy
			return fe.recursiveFieldTypeSearch(ctx, parentDict, debugger, depth+1)
		}
	}

	return FormFieldTypeUnknown
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
