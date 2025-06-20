package custom

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/a3tai/mcp-pdf-reader/internal/pdf/extraction"
)

// AcroFormParser handles parsing of PDF AcroForm structures
type AcroFormParser struct {
	parser     *CustomPDFParser
	formDict   *Dictionary
	fieldCache map[string]*FormField
}

// FormField represents a PDF form field with all its properties
type FormField struct {
	Name         string                 `json:"name"`
	Type         string                 `json:"type"`    // Tx, Btn, Ch, Sig
	SubType      string                 `json:"subtype"` // For buttons: radio, check, push
	Value        interface{}            `json:"value"`
	DefaultValue interface{}            `json:"default_value"`
	Flags        int64                  `json:"flags"`
	Rect         []float64              `json:"rect"` // [x1 y1 x2 y2]
	Page         int                    `json:"page"`
	ReadOnly     bool                   `json:"read_only"`
	Required     bool                   `json:"required"`
	NoExport     bool                   `json:"no_export"`
	Options      []FormFieldOption      `json:"options,omitempty"`     // For choice fields
	Kids         []*FormField           `json:"kids,omitempty"`        // Child fields
	Parent       *FormField             `json:"parent,omitempty"`      // Parent field
	Annotations  []*WidgetAnnotation    `json:"annotations,omitempty"` // Widget annotations
	Properties   map[string]interface{} `json:"properties,omitempty"`  // Additional properties
}

// FormFieldOption represents an option in a choice field
type FormFieldOption struct {
	Value   string `json:"value"`
	Display string `json:"display"`
}

// WidgetAnnotation represents a widget annotation associated with a form field
type WidgetAnnotation struct {
	Rect        []float64              `json:"rect"`
	Page        int                    `json:"page"`
	Appearance  map[string]interface{} `json:"appearance,omitempty"`
	BorderStyle map[string]interface{} `json:"border_style,omitempty"`
	Background  map[string]interface{} `json:"background,omitempty"`
	Properties  map[string]interface{} `json:"properties,omitempty"`
}

// AcroForm represents the complete AcroForm structure
type AcroForm struct {
	Fields          []*FormField           `json:"fields"`
	NeedAppearances bool                   `json:"need_appearances"`
	SigFlags        int64                  `json:"sig_flags,omitempty"`
	CO              []string               `json:"co,omitempty"`
	DR              map[string]interface{} `json:"dr,omitempty"`
	DA              string                 `json:"da,omitempty"`
	Q               int64                  `json:"q,omitempty"`
}

// NewAcroFormParser creates a new AcroForm parser
func NewAcroFormParser(parser *CustomPDFParser) *AcroFormParser {
	return &AcroFormParser{
		parser:     parser,
		fieldCache: make(map[string]*FormField),
	}
}

// ParseAcroForm parses the AcroForm dictionary from the document catalog
func (a *AcroFormParser) ParseAcroForm(catalog *Dictionary) (*AcroForm, error) {
	// Extract AcroForm dictionary
	acroFormObj := catalog.Get("AcroForm")
	if acroFormObj.Type() == TypeNull {
		return nil, nil // No forms in this document
	}

	// Resolve indirect reference if necessary
	formDict, err := a.resolveObject(acroFormObj)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve AcroForm dictionary: %w", err)
	}

	if formDict.Type() != TypeDictionary {
		return nil, fmt.Errorf("AcroForm must be a dictionary, got %s", formDict.Type())
	}

	a.formDict = formDict.(*Dictionary)

	// Parse field tree
	fieldsObj := a.formDict.Get("Fields")
	fields, err := a.parseFieldTree(fieldsObj, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to parse field tree: %w", err)
	}

	// Create AcroForm structure
	acroForm := &AcroForm{
		Fields:          fields,
		NeedAppearances: a.formDict.GetBool("NeedAppearances"),
		SigFlags:        a.formDict.GetInt("SigFlags"),
		CO:              a.parseCO(a.formDict.Get("CO")),
		DR:              a.parseResources(a.formDict.Get("DR")),
		DA:              a.formDict.GetString("DA"),
		Q:               a.formDict.GetInt("Q"),
	}

	return acroForm, nil
}

// parseFieldTree parses the hierarchical field tree structure
func (a *AcroFormParser) parseFieldTree(fieldsObj PDFObject, parent *FormField) ([]*FormField, error) {
	if fieldsObj.Type() == TypeNull {
		return nil, nil
	}

	// Resolve indirect reference
	resolved, err := a.resolveObject(fieldsObj)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve fields object: %w", err)
	}

	if resolved.Type() != TypeArray {
		return nil, fmt.Errorf("fields must be an array, got %s", resolved.Type())
	}

	fieldsArray := resolved.(*Array)
	var fields []*FormField

	for i, fieldObj := range fieldsArray.Elements {
		field, err := a.parseField(fieldObj, parent)
		if err != nil {
			return nil, fmt.Errorf("failed to parse field %d: %w", i, err)
		}
		if field != nil {
			fields = append(fields, field)
		}
	}

	return fields, nil
}

// parseField parses a single form field
func (a *AcroFormParser) parseField(fieldObj PDFObject, parent *FormField) (*FormField, error) {
	// Resolve indirect reference
	resolved, err := a.resolveObject(fieldObj)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve field object: %w", err)
	}

	if resolved.Type() != TypeDictionary {
		return nil, fmt.Errorf("field must be a dictionary, got %s", resolved.Type())
	}

	fieldDict := resolved.(*Dictionary)

	// Create field structure
	field := &FormField{
		Parent:     parent,
		Properties: make(map[string]interface{}),
	}

	// Parse basic field properties
	if err := a.parseBasicFieldProperties(field, fieldDict); err != nil {
		return nil, fmt.Errorf("failed to parse basic field properties: %w", err)
	}

	// Handle field inheritance from parent
	if parent != nil {
		a.inheritFieldProperties(field, parent)
	}

	// Parse field type-specific properties
	if err := a.parseFieldTypeProperties(field, fieldDict); err != nil {
		return nil, fmt.Errorf("failed to parse field type properties: %w", err)
	}

	// Parse widget annotations
	if err := a.parseWidgetAnnotations(field, fieldDict); err != nil {
		return nil, fmt.Errorf("failed to parse widget annotations: %w", err)
	}

	// Parse child fields (Kids)
	kidsObj := fieldDict.Get("Kids")
	if kidsObj.Type() != TypeNull {
		kids, err := a.parseFieldTree(kidsObj, field)
		if err != nil {
			return nil, fmt.Errorf("failed to parse child fields: %w", err)
		}
		field.Kids = kids
	}

	// Cache field for lookups
	if field.Name != "" {
		a.fieldCache[field.Name] = field
	}

	return field, nil
}

// parseBasicFieldProperties parses basic field properties common to all field types
func (a *AcroFormParser) parseBasicFieldProperties(field *FormField, dict *Dictionary) error {
	// Field name (T)
	field.Name = dict.GetString("T")

	// Field type (FT)
	field.Type = dict.GetName("FT")

	// Field flags (Ff)
	field.Flags = dict.GetInt("Ff")

	// Parse flags into boolean properties
	field.ReadOnly = (field.Flags & 1) != 0
	field.Required = (field.Flags & 2) != 0
	field.NoExport = (field.Flags & 4) != 0

	// Field value (V)
	valueObj := dict.Get("V")
	field.Value = a.parseFieldValue(valueObj)

	// Default value (DV)
	defaultObj := dict.Get("DV")
	field.DefaultValue = a.parseFieldValue(defaultObj)

	// Rectangle (Rect) - inherited from widget annotation
	rectObj := dict.Get("Rect")
	if rectObj.Type() != TypeNull {
		field.Rect = a.parseRect(rectObj)
	}

	return nil
}

// parseFieldTypeProperties parses properties specific to field types
func (a *AcroFormParser) parseFieldTypeProperties(field *FormField, dict *Dictionary) error {
	switch field.Type {
	case "Tx": // Text field
		return a.parseTextFieldProperties(field, dict)
	case "Btn": // Button field
		return a.parseButtonFieldProperties(field, dict)
	case "Ch": // Choice field
		return a.parseChoiceFieldProperties(field, dict)
	case "Sig": // Signature field
		return a.parseSignatureFieldProperties(field, dict)
	default:
		// Unknown field type, just store additional properties
		return a.parseGenericFieldProperties(field, dict)
	}
}

// parseTextFieldProperties parses text field specific properties
func (a *AcroFormParser) parseTextFieldProperties(field *FormField, dict *Dictionary) error {
	// Maximum length (MaxLen)
	if maxLen := dict.GetInt("MaxLen"); maxLen > 0 {
		field.Properties["max_length"] = maxLen
	}

	// Text field flags
	isMultiline := (field.Flags & 0x1000) != 0
	isPassword := (field.Flags & 0x2000) != 0
	isFileSelect := (field.Flags & 0x100000) != 0
	isRichText := (field.Flags & 0x2000000) != 0

	field.Properties["multiline"] = isMultiline
	field.Properties["password"] = isPassword
	field.Properties["file_select"] = isFileSelect
	field.Properties["rich_text"] = isRichText

	return nil
}

// parseButtonFieldProperties parses button field specific properties
func (a *AcroFormParser) parseButtonFieldProperties(field *FormField, dict *Dictionary) error {
	// Button type flags
	isRadio := (field.Flags & 0x8000) != 0
	isPushButton := (field.Flags & 0x10000) != 0

	if isPushButton {
		field.SubType = "push"
	} else if isRadio {
		field.SubType = "radio"
	} else {
		field.SubType = "check"
	}

	field.Properties["push_button"] = isPushButton
	field.Properties["radio"] = isRadio

	// For radio buttons, parse options
	if isRadio {
		// Radio buttons typically have their values in the widget annotations
		// or in the field value directly
		field.Properties["radio_group"] = true
	}

	return nil
}

// parseChoiceFieldProperties parses choice field specific properties
func (a *AcroFormParser) parseChoiceFieldProperties(field *FormField, dict *Dictionary) error {
	// Choice field flags
	isCombo := (field.Flags & 0x20000) != 0
	isEditable := (field.Flags & 0x40000) != 0
	isSort := (field.Flags & 0x80000) != 0
	isMultiSelect := (field.Flags & 0x200000) != 0

	field.SubType = "list"
	if isCombo {
		field.SubType = "combo"
	}

	field.Properties["combo"] = isCombo
	field.Properties["editable"] = isEditable
	field.Properties["sort"] = isSort
	field.Properties["multi_select"] = isMultiSelect

	// Parse options (Opt)
	optObj := dict.Get("Opt")
	if optObj.Type() != TypeNull {
		options, err := a.parseFieldOptions(optObj)
		if err != nil {
			return fmt.Errorf("failed to parse field options: %w", err)
		}
		field.Options = options
	}

	// Parse top index (TI)
	if topIndex := dict.GetInt("TI"); topIndex > 0 {
		field.Properties["top_index"] = topIndex
	}

	return nil
}

// parseSignatureFieldProperties parses signature field specific properties
func (a *AcroFormParser) parseSignatureFieldProperties(field *FormField, dict *Dictionary) error {
	field.Properties["signature"] = true

	// Lock dictionary (Lock)
	lockObj := dict.Get("Lock")
	if lockObj.Type() != TypeNull {
		field.Properties["lock"] = a.parseDictionaryToMap(lockObj)
	}

	// Seed value dictionary (SV)
	svObj := dict.Get("SV")
	if svObj.Type() != TypeNull {
		field.Properties["seed_value"] = a.parseDictionaryToMap(svObj)
	}

	return nil
}

// parseGenericFieldProperties parses properties for unknown field types
func (a *AcroFormParser) parseGenericFieldProperties(field *FormField, dict *Dictionary) error {
	// Store all additional properties
	for _, key := range dict.Keys {
		keyName := key.Value
		if !isStandardFieldKey(keyName) {
			obj := dict.Get(keyName)
			field.Properties[keyName] = a.parseFieldValue(obj)
		}
	}
	return nil
}

// parseWidgetAnnotations parses widget annotations associated with the field
func (a *AcroFormParser) parseWidgetAnnotations(field *FormField, dict *Dictionary) error {
	// Check if this field dictionary is also a widget annotation
	if dict.Has("Subtype") && dict.GetName("Subtype") == "Widget" {
		widget := a.parseWidgetAnnotation(dict)
		field.Annotations = append(field.Annotations, widget)

		// If field doesn't have a rect, use widget rect
		if len(field.Rect) == 0 && len(widget.Rect) > 0 {
			field.Rect = widget.Rect
		}
	}

	// Parse separate widget annotations (Kids that are widgets)
	kidsObj := dict.Get("Kids")
	if kidsObj.Type() != TypeNull {
		if resolved, err := a.resolveObject(kidsObj); err == nil {
			if resolved.Type() == TypeArray {
				kidsArray := resolved.(*Array)
				for _, kidObj := range kidsArray.Elements {
					if kidResolved, err := a.resolveObject(kidObj); err == nil {
						if kidResolved.Type() == TypeDictionary {
							kidDict := kidResolved.(*Dictionary)
							if kidDict.GetName("Subtype") == "Widget" {
								widget := a.parseWidgetAnnotation(kidDict)
								field.Annotations = append(field.Annotations, widget)
							}
						}
					}
				}
			}
		}
	}

	return nil
}

// parseWidgetAnnotation parses a single widget annotation
func (a *AcroFormParser) parseWidgetAnnotation(dict *Dictionary) *WidgetAnnotation {
	widget := &WidgetAnnotation{
		Properties: make(map[string]interface{}),
	}

	// Rectangle
	widget.Rect = a.parseRect(dict.Get("Rect"))

	// Page (P) - resolve page reference
	pageObj := dict.Get("P")
	if pageObj.Type() != TypeNull {
		// This would require page tree traversal to get page number
		// For now, we'll leave it as 0 and implement page resolution later
		widget.Page = 0
	}

	// Appearance (AP)
	apObj := dict.Get("AP")
	if apObj.Type() != TypeNull {
		widget.Appearance = a.parseDictionaryToMap(apObj)
	}

	// Border style (BS)
	bsObj := dict.Get("BS")
	if bsObj.Type() != TypeNull {
		widget.BorderStyle = a.parseDictionaryToMap(bsObj)
	}

	// Background (BG)
	bgObj := dict.Get("BG")
	if bgObj.Type() != TypeNull {
		widget.Background = a.parseDictionaryToMap(bgObj)
	}

	return widget
}

// parseFieldOptions parses the options array for choice fields
func (a *AcroFormParser) parseFieldOptions(optObj PDFObject) ([]FormFieldOption, error) {
	resolved, err := a.resolveObject(optObj)
	if err != nil {
		return nil, err
	}

	if resolved.Type() != TypeArray {
		return nil, fmt.Errorf("options must be an array")
	}

	optArray := resolved.(*Array)
	var options []FormFieldOption

	for _, elem := range optArray.Elements {
		elemResolved, err := a.resolveObject(elem)
		if err != nil {
			continue
		}

		if elemResolved.Type() == TypeString {
			// Simple string option
			value := elemResolved.(*String).Value
			options = append(options, FormFieldOption{
				Value:   value,
				Display: value,
			})
		} else if elemResolved.Type() == TypeArray {
			// Array with [export_value, display_text]
			elemArray := elemResolved.(*Array)
			if elemArray.Len() >= 2 {
				valueObj, _ := a.resolveObject(elemArray.Get(0))
				displayObj, _ := a.resolveObject(elemArray.Get(1))

				var value, display string
				if valueObj.Type() == TypeString {
					value = valueObj.(*String).Value
				}
				if displayObj.Type() == TypeString {
					display = displayObj.(*String).Value
				}

				options = append(options, FormFieldOption{
					Value:   value,
					Display: display,
				})
			}
		}
	}

	return options, nil
}

// parseFieldValue parses a field value object into appropriate Go type
func (a *AcroFormParser) parseFieldValue(obj PDFObject) interface{} {
	if obj.Type() == TypeNull {
		return nil
	}

	resolved, err := a.resolveObject(obj)
	if err != nil {
		return nil
	}

	switch resolved.Type() {
	case TypeString:
		return resolved.(*String).Value
	case TypeNumber:
		num := resolved.(*Number)
		if val, ok := num.Value.(int64); ok {
			return val
		}
		return num.Value
	case TypeBool:
		return resolved.(*Bool).Value
	case TypeName:
		return resolved.(*Name).Value
	case TypeArray:
		// Convert array to slice
		arr := resolved.(*Array)
		var result []interface{}
		for _, elem := range arr.Elements {
			result = append(result, a.parseFieldValue(elem))
		}
		return result
	default:
		return resolved.String()
	}
}

// parseRect parses a rectangle array [x1 y1 x2 y2]
func (a *AcroFormParser) parseRect(obj PDFObject) []float64 {
	if obj.Type() == TypeNull {
		return nil
	}

	resolved, err := a.resolveObject(obj)
	if err != nil || resolved.Type() != TypeArray {
		return nil
	}

	arr := resolved.(*Array)
	if arr.Len() < 4 {
		return nil
	}

	rect := make([]float64, 4)
	for i := 0; i < 4; i++ {
		elem, err := a.resolveObject(arr.Get(i))
		if err != nil || elem.Type() != TypeNumber {
			return nil
		}
		rect[i] = elem.(*Number).Float()
	}

	return rect
}

// parseCO parses the calculation order array
func (a *AcroFormParser) parseCO(obj PDFObject) []string {
	if obj.Type() == TypeNull {
		return nil
	}

	resolved, err := a.resolveObject(obj)
	if err != nil || resolved.Type() != TypeArray {
		return nil
	}

	arr := resolved.(*Array)
	var co []string
	for _, elem := range arr.Elements {
		if elemResolved, err := a.resolveObject(elem); err == nil {
			if elemResolved.Type() == TypeIndirectRef {
				// This is a field reference, we'd need to resolve the field name
				co = append(co, elemResolved.String())
			}
		}
	}

	return co
}

// parseResources parses the default resources dictionary
func (a *AcroFormParser) parseResources(obj PDFObject) map[string]interface{} {
	return a.parseDictionaryToMap(obj)
}

// parseDictionaryToMap converts a PDF dictionary to a Go map
func (a *AcroFormParser) parseDictionaryToMap(obj PDFObject) map[string]interface{} {
	if obj.Type() == TypeNull {
		return nil
	}

	resolved, err := a.resolveObject(obj)
	if err != nil || resolved.Type() != TypeDictionary {
		return nil
	}

	dict := resolved.(*Dictionary)
	result := make(map[string]interface{})

	for _, key := range dict.Keys {
		keyName := key.Value
		value := dict.Get(keyName)
		result[keyName] = a.parseFieldValue(value)
	}

	return result
}

// inheritFieldProperties handles field property inheritance from parent
func (a *AcroFormParser) inheritFieldProperties(field *FormField, parent *FormField) {
	// Inherit name if not set
	if field.Name == "" && parent.Name != "" {
		field.Name = parent.Name
	} else if field.Name != "" && parent.Name != "" {
		field.Name = parent.Name + "." + field.Name
	}

	// Inherit type if not set
	if field.Type == "" {
		field.Type = parent.Type
	}

	// Inherit flags
	if field.Flags == 0 {
		field.Flags = parent.Flags
	}

	// Inherit value if not set
	if field.Value == nil {
		field.Value = parent.Value
	}

	// Inherit default value if not set
	if field.DefaultValue == nil {
		field.DefaultValue = parent.DefaultValue
	}
}

// resolveObject resolves indirect object references
func (a *AcroFormParser) resolveObject(obj PDFObject) (PDFObject, error) {
	return a.parser.resolveIndirectObject(obj)
}

// isStandardFieldKey checks if a key is a standard PDF field key
func isStandardFieldKey(key string) bool {
	standardKeys := map[string]bool{
		"Type": true, "Parent": true, "Kids": true, "T": true, "TU": true, "TM": true,
		"Ff": true, "V": true, "DV": true, "AA": true, "FT": true, "MaxLen": true,
		"Opt": true, "TI": true, "I": true, "Lock": true, "SV": true,
		// Annotation keys
		"Subtype": true, "Rect": true, "Contents": true, "P": true, "NM": true,
		"M": true, "F": true, "AP": true, "AS": true, "Border": true, "C": true,
		"StructParent": true, "OC": true, "H": true, "MK": true, "A": true,
		"BS": true, "BE": true, "RD": true, "BG": true,
	}
	return standardKeys[key]
}

// ConvertToExtractionFormFields converts internal FormField to extraction.FormField
func (a *AcroFormParser) ConvertToExtractionFormFields(fields []*FormField) ([]extraction.FormField, error) {
	var result []extraction.FormField

	for _, field := range fields {
		extractionField := a.convertFormField(field)
		result = append(result, extractionField)

		// Add child fields
		if len(field.Kids) > 0 {
			childFields, err := a.ConvertToExtractionFormFields(field.Kids)
			if err != nil {
				return nil, err
			}
			result = append(result, childFields...)
		}
	}

	return result, nil
}

// convertFormField converts a single FormField to extraction.FormField
func (a *AcroFormParser) convertFormField(field *FormField) extraction.FormField {
	extractionField := extraction.FormField{
		Name:     field.Name,
		Type:     a.mapFieldType(field.Type, field.SubType),
		Value:    a.formatFieldValue(field.Value),
		Required: field.Required,
		ReadOnly: field.ReadOnly,
	}

	// Set position if available
	if len(field.Rect) >= 4 {
		extractionField.Bounds = &extraction.BoundingBox{
			LowerLeft: extraction.Coordinate{
				X: field.Rect[0],
				Y: field.Rect[1],
			},
			UpperRight: extraction.Coordinate{
				X: field.Rect[2],
				Y: field.Rect[3],
			},
			Width:  field.Rect[2] - field.Rect[0],
			Height: field.Rect[3] - field.Rect[1],
		}
	}

	// Set page if available from annotations
	if len(field.Annotations) > 0 {
		extractionField.Page = field.Annotations[0].Page
	}

	// Set options for choice fields
	if len(field.Options) > 0 {
		for _, opt := range field.Options {
			extractionField.Options = append(extractionField.Options, opt.Display)
		}
	}

	return extractionField
}

// mapFieldType maps PDF field types to extraction field types
func (a *AcroFormParser) mapFieldType(fieldType, subType string) extraction.FormFieldType {
	switch fieldType {
	case "Tx":
		return extraction.FormFieldTypeText
	case "Btn":
		switch subType {
		case "push":
			return extraction.FormFieldTypeButton
		case "radio":
			return extraction.FormFieldTypeRadio
		case "check":
			return extraction.FormFieldTypeCheckbox
		default:
			return extraction.FormFieldTypeCheckbox
		}
	case "Ch":
		if subType == "combo" {
			return extraction.FormFieldTypeSelect
		}
		return extraction.FormFieldTypeSelect
	case "Sig":
		return extraction.FormFieldTypeSignature
	default:
		return extraction.FormFieldTypeText
	}
}

// formatFieldValue formats field value for extraction interface
func (a *AcroFormParser) formatFieldValue(value interface{}) string {
	if value == nil {
		return ""
	}

	switch v := value.(type) {
	case string:
		return v
	case int64:
		return strconv.FormatInt(v, 10)
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case bool:
		if v {
			return "true"
		}
		return "false"
	case []interface{}:
		var parts []string
		for _, item := range v {
			parts = append(parts, a.formatFieldValue(item))
		}
		return strings.Join(parts, ", ")
	default:
		return fmt.Sprintf("%v", v)
	}
}
