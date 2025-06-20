package security

import (
	"errors"
	"fmt"
)

// ObjectResolver defines the interface for resolving PDF object references
type ObjectResolver interface {
	ResolveObject(ref interface{}) (interface{}, error)
	GetObject(objNum, genNum int) (interface{}, error)
}

// SecurityParser handles parsing of encryption-related PDF objects
type SecurityParser struct {
	resolver ObjectResolver
}

// NewSecurityParser creates a new security parser with the given object resolver
func NewSecurityParser(resolver ObjectResolver) *SecurityParser {
	return &SecurityParser{
		resolver: resolver,
	}
}

// ParseEncryptDict parses a PDF encryption dictionary into an EncryptionDictionary struct
func (p *SecurityParser) ParseEncryptDict(dict map[string]interface{}) (*EncryptionDictionary, error) {
	if dict == nil {
		return nil, errors.New("encryption dictionary is nil")
	}

	enc := &EncryptionDictionary{
		EncryptMetadata: true, // Default value per PDF spec
		CF:              make(map[string]CryptFilter),
	}

	// Parse Filter (required)
	if filter, ok := dict["Filter"]; ok {
		if filterStr, ok := filter.(string); ok {
			enc.Filter = filterStr
		} else {
			return nil, errors.New("Filter must be a string")
		}
	} else {
		return nil, errors.New("missing required Filter entry in encryption dictionary")
	}

	// Parse SubFilter (optional)
	if subFilter, ok := dict["SubFilter"]; ok {
		if subFilterStr, ok := subFilter.(string); ok {
			enc.SubFilter = subFilterStr
		}
	}

	// Parse V (version) - required
	if v, ok := dict["V"]; ok {
		if vInt, ok := v.(int); ok {
			enc.V = vInt
		} else if vFloat, ok := v.(float64); ok {
			enc.V = int(vFloat)
		} else {
			return nil, errors.New("V must be an integer")
		}
	} else {
		return nil, errors.New("missing required V entry in encryption dictionary")
	}

	// Parse Length (optional, algorithm-specific)
	if length, ok := dict["Length"]; ok {
		if lengthInt, ok := length.(int); ok {
			enc.Length = lengthInt
		} else if lengthFloat, ok := length.(float64); ok {
			enc.Length = int(lengthFloat)
		}
	} else {
		// Set default length based on version
		switch enc.V {
		case 1:
			enc.Length = 40
		case 2:
			enc.Length = 40
		case 4:
			enc.Length = 128
		case 5:
			enc.Length = 256
		default:
			enc.Length = 40
		}
	}

	// Parse R (revision) - required for Standard security handler
	if r, ok := dict["R"]; ok {
		if rInt, ok := r.(int); ok {
			enc.R = rInt
		} else if rFloat, ok := r.(float64); ok {
			enc.R = int(rFloat)
		} else {
			return nil, errors.New("R must be an integer")
		}
	} else {
		return nil, errors.New("missing required R entry in encryption dictionary")
	}

	// Parse O (owner password hash) - required
	if o, ok := dict["O"]; ok {
		if oBytes, err := p.parseByteString(o); err != nil {
			return nil, fmt.Errorf("failed to parse O entry: %w", err)
		} else {
			enc.O = oBytes
		}
	} else {
		return nil, errors.New("missing required O entry in encryption dictionary")
	}

	// Parse U (user password hash) - required
	if u, ok := dict["U"]; ok {
		if uBytes, err := p.parseByteString(u); err != nil {
			return nil, fmt.Errorf("failed to parse U entry: %w", err)
		} else {
			enc.U = uBytes
		}
	} else {
		return nil, errors.New("missing required U entry in encryption dictionary")
	}

	// Parse OE (owner encryption key) - for revision 6
	if oe, ok := dict["OE"]; ok {
		if oeBytes, err := p.parseByteString(oe); err != nil {
			return nil, fmt.Errorf("failed to parse OE entry: %w", err)
		} else {
			enc.OE = oeBytes
		}
	}

	// Parse UE (user encryption key) - for revision 6
	if ue, ok := dict["UE"]; ok {
		if ueBytes, err := p.parseByteString(ue); err != nil {
			return nil, fmt.Errorf("failed to parse UE entry: %w", err)
		} else {
			enc.UE = ueBytes
		}
	}

	// Parse P (permissions) - required
	if pVal, ok := dict["P"]; ok {
		if pInt, ok := pVal.(int); ok {
			enc.P = int32(pInt)
		} else if pFloat, ok := pVal.(float64); ok {
			enc.P = int32(pFloat)
		} else if pInt32, ok := pVal.(int32); ok {
			enc.P = pInt32
		} else {
			return nil, errors.New("P must be an integer")
		}
	} else {
		return nil, errors.New("missing required P entry in encryption dictionary")
	}

	// Parse EncryptMetadata (optional, default true)
	if encryptMetadata, ok := dict["EncryptMetadata"]; ok {
		if encryptBool, ok := encryptMetadata.(bool); ok {
			enc.EncryptMetadata = encryptBool
		}
	}

	// Parse StmF (stream filter) - for V >= 4
	if stmF, ok := dict["StmF"]; ok {
		if stmFStr, ok := stmF.(string); ok {
			enc.StmF = stmFStr
		}
	}

	// Parse StrF (string filter) - for V >= 4
	if strF, ok := dict["StrF"]; ok {
		if strFStr, ok := strF.(string); ok {
			enc.StrF = strFStr
		}
	}

	// Parse CF (crypt filter dictionary) - for V >= 4
	if cf, ok := dict["CF"]; ok {
		if cfDict, ok := cf.(map[string]interface{}); ok {
			parsedCF, err := p.parseCryptFilters(cfDict)
			if err != nil {
				return nil, fmt.Errorf("failed to parse CF entry: %w", err)
			}
			enc.CF = parsedCF
		}
	}

	// Validate the parsed encryption dictionary
	if err := p.validateEncryptionDictionary(enc); err != nil {
		return nil, fmt.Errorf("encryption dictionary validation failed: %w", err)
	}

	return enc, nil
}

// parseByteString converts various PDF string representations to byte arrays
func (p *SecurityParser) parseByteString(obj interface{}) ([]byte, error) {
	switch v := obj.(type) {
	case string:
		return []byte(v), nil
	case []byte:
		return v, nil
	case []interface{}:
		// Handle arrays of integers representing bytes
		bytes := make([]byte, len(v))
		for i, item := range v {
			if intVal, ok := item.(int); ok {
				bytes[i] = byte(intVal)
			} else if floatVal, ok := item.(float64); ok {
				bytes[i] = byte(floatVal)
			} else {
				return nil, fmt.Errorf("array element at index %d is not a number", i)
			}
		}
		return bytes, nil
	default:
		return nil, fmt.Errorf("unsupported type for byte string: %T", obj)
	}
}

// parseCryptFilters parses the CF (crypt filter) dictionary
func (p *SecurityParser) parseCryptFilters(cfDict map[string]interface{}) (map[string]CryptFilter, error) {
	filters := make(map[string]CryptFilter)

	for name, filterObj := range cfDict {
		if filterDict, ok := filterObj.(map[string]interface{}); ok {
			filter, err := p.parseCryptFilter(filterDict)
			if err != nil {
				return nil, fmt.Errorf("failed to parse crypt filter '%s': %w", name, err)
			}
			filters[name] = filter
		} else {
			return nil, fmt.Errorf("crypt filter '%s' is not a dictionary", name)
		}
	}

	return filters, nil
}

// parseCryptFilter parses a single crypt filter dictionary
func (p *SecurityParser) parseCryptFilter(filterDict map[string]interface{}) (CryptFilter, error) {
	filter := CryptFilter{
		Type:      "CryptFilter", // Default
		AuthEvent: "DocOpen",     // Default
	}

	// Parse Type (should always be "CryptFilter")
	if typeVal, ok := filterDict["Type"]; ok {
		if typeStr, ok := typeVal.(string); ok {
			filter.Type = typeStr
		}
	}

	// Parse CFM (crypt filter method) - required
	if cfm, ok := filterDict["CFM"]; ok {
		if cfmStr, ok := cfm.(string); ok {
			filter.CFM = cfmStr
		} else {
			return CryptFilter{}, errors.New("CFM must be a string")
		}
	} else {
		return CryptFilter{}, errors.New("missing required CFM entry in crypt filter")
	}

	// Parse AuthEvent (authorization event)
	if authEvent, ok := filterDict["AuthEvent"]; ok {
		if authEventStr, ok := authEvent.(string); ok {
			filter.AuthEvent = authEventStr
		}
	}

	// Parse Length (key length in bytes)
	if length, ok := filterDict["Length"]; ok {
		if lengthInt, ok := length.(int); ok {
			filter.Length = lengthInt
		} else if lengthFloat, ok := length.(float64); ok {
			filter.Length = int(lengthFloat)
		}
	}

	return filter, nil
}

// validateEncryptionDictionary performs validation on the parsed encryption dictionary
func (p *SecurityParser) validateEncryptionDictionary(enc *EncryptionDictionary) error {
	// Validate Filter
	if enc.Filter != "Standard" {
		return fmt.Errorf("unsupported security handler: %s", enc.Filter)
	}

	// Validate V (version)
	if enc.V < 1 || enc.V > 5 {
		return fmt.Errorf("unsupported encryption version: %d", enc.V)
	}

	// Validate R (revision)
	if enc.R < 2 || enc.R > 6 {
		return fmt.Errorf("unsupported security handler revision: %d", enc.R)
	}

	// Validate consistency between V and R
	switch enc.V {
	case 1:
		if enc.R != 2 {
			return fmt.Errorf("V=1 requires R=2, got R=%d", enc.R)
		}
	case 2:
		if enc.R != 3 {
			return fmt.Errorf("V=2 requires R=3, got R=%d", enc.R)
		}
	case 4:
		if enc.R != 4 {
			return fmt.Errorf("V=4 requires R=4, got R=%d", enc.R)
		}
	case 5:
		if enc.R != 5 && enc.R != 6 {
			return fmt.Errorf("V=5 requires R=5 or R=6, got R=%d", enc.R)
		}
	}

	// Validate key length
	switch enc.V {
	case 1:
		if enc.Length != 40 {
			return fmt.Errorf("V=1 requires Length=40, got Length=%d", enc.Length)
		}
	case 2:
		if enc.Length < 40 || enc.Length > 128 || enc.Length%8 != 0 {
			return fmt.Errorf("V=2 requires Length between 40-128 (multiple of 8), got Length=%d", enc.Length)
		}
	case 4:
		if enc.Length != 128 {
			return fmt.Errorf("V=4 requires Length=128, got Length=%d", enc.Length)
		}
	case 5:
		if enc.Length != 256 {
			return fmt.Errorf("V=5 requires Length=256, got Length=%d", enc.Length)
		}
	}

	// Validate required byte arrays
	if len(enc.O) == 0 {
		return errors.New("O (owner password hash) cannot be empty")
	}
	if len(enc.U) == 0 {
		return errors.New("U (user password hash) cannot be empty")
	}

	// Validate revision-specific entries
	if enc.R == 6 {
		if len(enc.OE) == 0 {
			return errors.New("OE (owner encryption key) required for R=6")
		}
		if len(enc.UE) == 0 {
			return errors.New("UE (user encryption key) required for R=6")
		}
	}

	// Validate crypt filters for V >= 4
	if enc.V >= 4 {
		if enc.StmF != "" {
			if _, ok := enc.CF[enc.StmF]; !ok {
				return fmt.Errorf("StmF references undefined crypt filter: %s", enc.StmF)
			}
		}
		if enc.StrF != "" {
			if _, ok := enc.CF[enc.StrF]; !ok {
				return fmt.Errorf("StrF references undefined crypt filter: %s", enc.StrF)
			}
		}
	}

	return nil
}

// GetEncryptionInfo returns a human-readable summary of the encryption settings
func (p *SecurityParser) GetEncryptionInfo(enc *EncryptionDictionary) map[string]interface{} {
	info := map[string]interface{}{
		"filter":           enc.Filter,
		"version":          enc.V,
		"revision":         enc.R,
		"key_length_bits":  enc.Length,
		"encrypt_metadata": enc.EncryptMetadata,
	}

	// Add algorithm information
	switch enc.V {
	case 1, 2:
		info["algorithm"] = "RC4"
	case 4:
		if enc.StmF == "StdCF" || enc.StrF == "StdCF" {
			if cf, ok := enc.CF["StdCF"]; ok {
				switch cf.CFM {
				case "AESV2":
					info["algorithm"] = "AES-128"
				case "V2":
					info["algorithm"] = "RC4"
				default:
					info["algorithm"] = cf.CFM
				}
			}
		} else {
			info["algorithm"] = "RC4"
		}
	case 5:
		info["algorithm"] = "AES-256"
	}

	// Add permissions info
	permissions := NewPermissions(enc.P)
	info["permissions"] = map[string]interface{}{
		"print":              permissions.Print,
		"modify":             permissions.Modify,
		"copy":               permissions.Copy,
		"annotate":           permissions.Annotate,
		"fill_forms":         permissions.FillForms,
		"extract":            permissions.Extract,
		"assemble":           permissions.Assemble,
		"print_high_quality": permissions.PrintHighQuality,
	}

	return info
}
