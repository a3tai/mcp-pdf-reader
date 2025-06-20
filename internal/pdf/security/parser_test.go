package security

import (
	"reflect"
	"testing"
)

// Mock ObjectResolver for testing
type mockObjectResolver struct {
	objects map[string]interface{}
}

func (m *mockObjectResolver) ResolveObject(ref interface{}) (interface{}, error) {
	if refStr, ok := ref.(string); ok {
		if obj, exists := m.objects[refStr]; exists {
			return obj, nil
		}
	}
	return nil, nil
}

func (m *mockObjectResolver) GetObject(objNum, genNum int) (interface{}, error) {
	return nil, nil
}

func newMockResolver() *mockObjectResolver {
	return &mockObjectResolver{
		objects: make(map[string]interface{}),
	}
}

func TestSecurityParser_ParseEncryptDict(t *testing.T) {
	parser := NewSecurityParser(newMockResolver())

	tests := []struct {
		name    string
		dict    map[string]interface{}
		want    *EncryptionDictionary
		wantErr bool
	}{
		{
			name: "Valid V=2 R=3 dictionary",
			dict: map[string]interface{}{
				"Filter": "Standard",
				"V":      2,
				"R":      3,
				"Length": 40,
				"O":      []byte{0x28, 0xBF, 0x4E, 0x5E, 0x4E, 0x75, 0x8A, 0x41, 0x64, 0x00, 0x4E, 0x56, 0xFF, 0xFA, 0x01, 0x08, 0x2E, 0x2E, 0x00, 0xB6, 0xD0, 0x68, 0x3E, 0x80, 0x2F, 0x0C, 0xA9, 0xFE, 0x64, 0x53, 0x69, 0x7A},
				"U":      []byte{0x44, 0x6D, 0x8D, 0x99, 0x90, 0xE7, 0x23, 0x4F, 0x8C, 0x86, 0x8C, 0x8F, 0x63, 0x9B, 0x2C, 0x12, 0x28, 0xBF, 0x4E, 0x5E, 0x4E, 0x75, 0x8A, 0x41, 0x64, 0x00, 0x4E, 0x56, 0xFF, 0xFA, 0x01, 0x08},
				"P":      int32(-44),
			},
			want: &EncryptionDictionary{
				Filter:          "Standard",
				V:               2,
				R:               3,
				Length:          40,
				O:               []byte{0x28, 0xBF, 0x4E, 0x5E, 0x4E, 0x75, 0x8A, 0x41, 0x64, 0x00, 0x4E, 0x56, 0xFF, 0xFA, 0x01, 0x08, 0x2E, 0x2E, 0x00, 0xB6, 0xD0, 0x68, 0x3E, 0x80, 0x2F, 0x0C, 0xA9, 0xFE, 0x64, 0x53, 0x69, 0x7A},
				U:               []byte{0x44, 0x6D, 0x8D, 0x99, 0x90, 0xE7, 0x23, 0x4F, 0x8C, 0x86, 0x8C, 0x8F, 0x63, 0x9B, 0x2C, 0x12, 0x28, 0xBF, 0x4E, 0x5E, 0x4E, 0x75, 0x8A, 0x41, 0x64, 0x00, 0x4E, 0x56, 0xFF, 0xFA, 0x01, 0x08},
				P:               -44,
				EncryptMetadata: true,
				CF:              make(map[string]CryptFilter),
			},
			wantErr: false,
		},
		{
			name: "Valid V=4 R=4 dictionary with AES",
			dict: map[string]interface{}{
				"Filter": "Standard",
				"V":      4,
				"R":      4,
				"Length": 128,
				"O":      "owner_hash_32_bytes_long_string",
				"U":      "user_hash_32_bytes_long_string_",
				"P":      int32(-44),
				"StmF":   "StdCF",
				"StrF":   "StdCF",
				"CF": map[string]interface{}{
					"StdCF": map[string]interface{}{
						"Type":      "CryptFilter",
						"CFM":       "AESV2",
						"AuthEvent": "DocOpen",
						"Length":    16,
					},
				},
			},
			want: &EncryptionDictionary{
				Filter:          "Standard",
				V:               4,
				R:               4,
				Length:          128,
				O:               []byte("owner_hash_32_bytes_long_string"),
				U:               []byte("user_hash_32_bytes_long_string_"),
				P:               -44,
				EncryptMetadata: true,
				StmF:            "StdCF",
				StrF:            "StdCF",
				CF: map[string]CryptFilter{
					"StdCF": {
						Type:      "CryptFilter",
						CFM:       "AESV2",
						AuthEvent: "DocOpen",
						Length:    16,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Valid V=1 R=2 dictionary (default length)",
			dict: map[string]interface{}{
				"Filter": "Standard",
				"V":      1,
				"R":      2,
				"O":      "owner_hash",
				"U":      "user_hash_",
				"P":      int32(-4),
			},
			want: &EncryptionDictionary{
				Filter:          "Standard",
				V:               1,
				R:               2,
				Length:          40, // Default for V=1
				O:               []byte("owner_hash"),
				U:               []byte("user_hash_"),
				P:               -4,
				EncryptMetadata: true,
				CF:              make(map[string]CryptFilter),
			},
			wantErr: false,
		},
		{
			name: "Dictionary with float values",
			dict: map[string]interface{}{
				"Filter": "Standard",
				"V":      float64(2),
				"R":      float64(3),
				"Length": float64(40),
				"O":      "owner_hash",
				"U":      "user_hash_",
				"P":      float64(-44),
			},
			want: &EncryptionDictionary{
				Filter:          "Standard",
				V:               2,
				R:               3,
				Length:          40,
				O:               []byte("owner_hash"),
				U:               []byte("user_hash_"),
				P:               -44,
				EncryptMetadata: true,
				CF:              make(map[string]CryptFilter),
			},
			wantErr: false,
		},
		{
			name: "Dictionary with byte arrays",
			dict: map[string]interface{}{
				"Filter": "Standard",
				"V":      2,
				"R":      3,
				"O":      []interface{}{40, 191, 78, 94, 78, 117, 138, 65, 100, 0, 78, 86, 255, 250, 1, 8},
				"U":      []interface{}{68, 109, 141, 153, 144, 231, 35, 79, 140, 134, 140, 143, 99, 155, 44, 18},
				"P":      int32(-44),
			},
			want: &EncryptionDictionary{
				Filter:          "Standard",
				V:               2,
				R:               3,
				Length:          40,
				O:               []byte{40, 191, 78, 94, 78, 117, 138, 65, 100, 0, 78, 86, 255, 250, 1, 8},
				U:               []byte{68, 109, 141, 153, 144, 231, 35, 79, 140, 134, 140, 143, 99, 155, 44, 18},
				P:               -44,
				EncryptMetadata: true,
				CF:              make(map[string]CryptFilter),
			},
			wantErr: false,
		},
		{
			name: "Dictionary with EncryptMetadata false",
			dict: map[string]interface{}{
				"Filter":          "Standard",
				"V":               4,
				"R":               4,
				"Length":          128,
				"O":               "owner_hash",
				"U":               "user_hash_",
				"P":               int32(-44),
				"EncryptMetadata": false,
			},
			want: &EncryptionDictionary{
				Filter:          "Standard",
				V:               4,
				R:               4,
				Length:          128,
				O:               []byte("owner_hash"),
				U:               []byte("user_hash_"),
				P:               -44,
				EncryptMetadata: false,
				CF:              make(map[string]CryptFilter),
			},
			wantErr: false,
		},
		{
			name:    "Nil dictionary",
			dict:    nil,
			want:    nil,
			wantErr: true,
		},
		{
			name: "Missing Filter",
			dict: map[string]interface{}{
				"V": 2,
				"R": 3,
				"O": "owner_hash",
				"U": "user_hash_",
				"P": int32(-44),
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "Missing V",
			dict: map[string]interface{}{
				"Filter": "Standard",
				"R":      3,
				"O":      "owner_hash",
				"U":      "user_hash_",
				"P":      int32(-44),
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "Missing R",
			dict: map[string]interface{}{
				"Filter": "Standard",
				"V":      2,
				"O":      "owner_hash",
				"U":      "user_hash_",
				"P":      int32(-44),
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "Missing O",
			dict: map[string]interface{}{
				"Filter": "Standard",
				"V":      2,
				"R":      3,
				"U":      "user_hash_",
				"P":      int32(-44),
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "Missing U",
			dict: map[string]interface{}{
				"Filter": "Standard",
				"V":      2,
				"R":      3,
				"O":      "owner_hash",
				"P":      int32(-44),
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "Missing P",
			dict: map[string]interface{}{
				"Filter": "Standard",
				"V":      2,
				"R":      3,
				"O":      "owner_hash",
				"U":      "user_hash_",
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "Invalid Filter type",
			dict: map[string]interface{}{
				"Filter": 123,
				"V":      2,
				"R":      3,
				"O":      "owner_hash",
				"U":      "user_hash_",
				"P":      int32(-44),
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "Invalid V type",
			dict: map[string]interface{}{
				"Filter": "Standard",
				"V":      "invalid",
				"R":      3,
				"O":      "owner_hash",
				"U":      "user_hash_",
				"P":      int32(-44),
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "Unsupported Filter",
			dict: map[string]interface{}{
				"Filter": "Custom",
				"V":      2,
				"R":      3,
				"O":      "owner_hash",
				"U":      "user_hash_",
				"P":      int32(-44),
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "Unsupported Version",
			dict: map[string]interface{}{
				"Filter": "Standard",
				"V":      6,
				"R":      7,
				"O":      "owner_hash",
				"U":      "user_hash_",
				"P":      int32(-44),
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "Inconsistent V and R",
			dict: map[string]interface{}{
				"Filter": "Standard",
				"V":      1,
				"R":      3, // Should be 2 for V=1
				"O":      "owner_hash",
				"U":      "user_hash_",
				"P":      int32(-44),
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "Invalid Length for V=1",
			dict: map[string]interface{}{
				"Filter": "Standard",
				"V":      1,
				"R":      2,
				"Length": 128, // Should be 40 for V=1
				"O":      "owner_hash",
				"U":      "user_hash_",
				"P":      int32(-44),
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parser.ParseEncryptDict(tt.dict)

			if (err != nil) != tt.wantErr {
				t.Errorf("ParseEncryptDict() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseEncryptDict() got = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestSecurityParser_parseByteString(t *testing.T) {
	parser := NewSecurityParser(newMockResolver())

	tests := []struct {
		name    string
		obj     interface{}
		want    []byte
		wantErr bool
	}{
		{
			name:    "String input",
			obj:     "hello",
			want:    []byte("hello"),
			wantErr: false,
		},
		{
			name:    "Byte array input",
			obj:     []byte{0x01, 0x02, 0x03},
			want:    []byte{0x01, 0x02, 0x03},
			wantErr: false,
		},
		{
			name:    "Integer array input",
			obj:     []interface{}{1, 2, 3, 255},
			want:    []byte{1, 2, 3, 255},
			wantErr: false,
		},
		{
			name:    "Float array input",
			obj:     []interface{}{1.0, 2.0, 3.0, 255.0},
			want:    []byte{1, 2, 3, 255},
			wantErr: false,
		},
		{
			name:    "Empty string",
			obj:     "",
			want:    []byte{},
			wantErr: false,
		},
		{
			name:    "Empty array",
			obj:     []interface{}{},
			want:    []byte{},
			wantErr: false,
		},
		{
			name:    "Invalid array element",
			obj:     []interface{}{1, "invalid", 3},
			want:    nil,
			wantErr: true,
		},
		{
			name:    "Unsupported type",
			obj:     123,
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parser.parseByteString(tt.obj)

			if (err != nil) != tt.wantErr {
				t.Errorf("parseByteString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseByteString() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSecurityParser_parseCryptFilters(t *testing.T) {
	parser := NewSecurityParser(newMockResolver())

	tests := []struct {
		name    string
		cfDict  map[string]interface{}
		want    map[string]CryptFilter
		wantErr bool
	}{
		{
			name: "Valid crypt filters",
			cfDict: map[string]interface{}{
				"StdCF": map[string]interface{}{
					"Type":      "CryptFilter",
					"CFM":       "AESV2",
					"AuthEvent": "DocOpen",
					"Length":    16,
				},
				"CustomCF": map[string]interface{}{
					"CFM":       "V2",
					"AuthEvent": "EFOpen",
					"Length":    8,
				},
			},
			want: map[string]CryptFilter{
				"StdCF": {
					Type:      "CryptFilter",
					CFM:       "AESV2",
					AuthEvent: "DocOpen",
					Length:    16,
				},
				"CustomCF": {
					Type:      "CryptFilter", // Default
					CFM:       "V2",
					AuthEvent: "EFOpen",
					Length:    8,
				},
			},
			wantErr: false,
		},
		{
			name:    "Empty crypt filters",
			cfDict:  map[string]interface{}{},
			want:    map[string]CryptFilter{},
			wantErr: false,
		},
		{
			name: "Invalid filter - not a dictionary",
			cfDict: map[string]interface{}{
				"BadFilter": "not a dict",
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "Filter missing CFM",
			cfDict: map[string]interface{}{
				"BadFilter": map[string]interface{}{
					"Type": "CryptFilter",
				},
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parser.parseCryptFilters(tt.cfDict)

			if (err != nil) != tt.wantErr {
				t.Errorf("parseCryptFilters() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseCryptFilters() got = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestSecurityParser_parseCryptFilter(t *testing.T) {
	parser := NewSecurityParser(newMockResolver())

	tests := []struct {
		name       string
		filterDict map[string]interface{}
		want       CryptFilter
		wantErr    bool
	}{
		{
			name: "Complete crypt filter",
			filterDict: map[string]interface{}{
				"Type":      "CryptFilter",
				"CFM":       "AESV2",
				"AuthEvent": "DocOpen",
				"Length":    16,
			},
			want: CryptFilter{
				Type:      "CryptFilter",
				CFM:       "AESV2",
				AuthEvent: "DocOpen",
				Length:    16,
			},
			wantErr: false,
		},
		{
			name: "Minimal crypt filter",
			filterDict: map[string]interface{}{
				"CFM": "V2",
			},
			want: CryptFilter{
				Type:      "CryptFilter", // Default
				CFM:       "V2",
				AuthEvent: "DocOpen", // Default
				Length:    0,
			},
			wantErr: false,
		},
		{
			name: "Crypt filter with float length",
			filterDict: map[string]interface{}{
				"CFM":    "AESV2",
				"Length": float64(16),
			},
			want: CryptFilter{
				Type:      "CryptFilter",
				CFM:       "AESV2",
				AuthEvent: "DocOpen",
				Length:    16,
			},
			wantErr: false,
		},
		{
			name:       "Missing CFM",
			filterDict: map[string]interface{}{},
			want:       CryptFilter{},
			wantErr:    true,
		},
		{
			name: "Invalid CFM type",
			filterDict: map[string]interface{}{
				"CFM": 123,
			},
			want:    CryptFilter{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parser.parseCryptFilter(tt.filterDict)

			if (err != nil) != tt.wantErr {
				t.Errorf("parseCryptFilter() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if got != tt.want {
				t.Errorf("parseCryptFilter() got = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestSecurityParser_validateEncryptionDictionary(t *testing.T) {
	parser := NewSecurityParser(newMockResolver())

	tests := []struct {
		name    string
		enc     *EncryptionDictionary
		wantErr bool
	}{
		{
			name: "Valid V=2 R=3",
			enc: &EncryptionDictionary{
				Filter: "Standard",
				V:      2,
				R:      3,
				Length: 40,
				O:      []byte("owner"),
				U:      []byte("user_"),
				P:      -44,
			},
			wantErr: false,
		},
		{
			name: "Valid V=4 R=4 with crypt filters",
			enc: &EncryptionDictionary{
				Filter: "Standard",
				V:      4,
				R:      4,
				Length: 128,
				O:      []byte("owner"),
				U:      []byte("user_"),
				P:      -44,
				StmF:   "StdCF",
				StrF:   "StdCF",
				CF: map[string]CryptFilter{
					"StdCF": {CFM: "AESV2"},
				},
			},
			wantErr: false,
		},
		{
			name: "Unsupported filter",
			enc: &EncryptionDictionary{
				Filter: "Custom",
				V:      2,
				R:      3,
				O:      []byte("owner"),
				U:      []byte("user_"),
				P:      -44,
			},
			wantErr: true,
		},
		{
			name: "Invalid version",
			enc: &EncryptionDictionary{
				Filter: "Standard",
				V:      0,
				R:      3,
				O:      []byte("owner"),
				U:      []byte("user_"),
				P:      -44,
			},
			wantErr: true,
		},
		{
			name: "Invalid revision",
			enc: &EncryptionDictionary{
				Filter: "Standard",
				V:      2,
				R:      1,
				O:      []byte("owner"),
				U:      []byte("user_"),
				P:      -44,
			},
			wantErr: true,
		},
		{
			name: "V=1 with wrong R",
			enc: &EncryptionDictionary{
				Filter: "Standard",
				V:      1,
				R:      3,
				Length: 40,
				O:      []byte("owner"),
				U:      []byte("user_"),
				P:      -44,
			},
			wantErr: true,
		},
		{
			name: "V=1 with wrong length",
			enc: &EncryptionDictionary{
				Filter: "Standard",
				V:      1,
				R:      2,
				Length: 128,
				O:      []byte("owner"),
				U:      []byte("user_"),
				P:      -44,
			},
			wantErr: true,
		},
		{
			name: "Empty O",
			enc: &EncryptionDictionary{
				Filter: "Standard",
				V:      2,
				R:      3,
				Length: 40,
				O:      []byte{},
				U:      []byte("user_"),
				P:      -44,
			},
			wantErr: true,
		},
		{
			name: "Empty U",
			enc: &EncryptionDictionary{
				Filter: "Standard",
				V:      2,
				R:      3,
				Length: 40,
				O:      []byte("owner"),
				U:      []byte{},
				P:      -44,
			},
			wantErr: true,
		},
		{
			name: "Invalid length for V=2",
			enc: &EncryptionDictionary{
				Filter: "Standard",
				V:      2,
				R:      3,
				Length: 33, // Not multiple of 8
				O:      []byte("owner"),
				U:      []byte("user_"),
				P:      -44,
			},
			wantErr: true,
		},
		{
			name: "StmF references undefined filter",
			enc: &EncryptionDictionary{
				Filter: "Standard",
				V:      4,
				R:      4,
				Length: 128,
				O:      []byte("owner"),
				U:      []byte("user_"),
				P:      -44,
				StmF:   "UndefinedCF",
				CF:     map[string]CryptFilter{},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := parser.validateEncryptionDictionary(tt.enc)

			if (err != nil) != tt.wantErr {
				t.Errorf("validateEncryptionDictionary() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSecurityParser_GetEncryptionInfo(t *testing.T) {
	parser := NewSecurityParser(newMockResolver())

	tests := []struct {
		name string
		enc  *EncryptionDictionary
		want map[string]interface{}
	}{
		{
			name: "V=2 R=3 RC4",
			enc: &EncryptionDictionary{
				Filter:          "Standard",
				V:               2,
				R:               3,
				Length:          40,
				P:               -44,
				EncryptMetadata: true,
			},
			want: map[string]interface{}{
				"filter":           "Standard",
				"version":          2,
				"revision":         3,
				"key_length_bits":  40,
				"encrypt_metadata": true,
				"algorithm":        "RC4",
				"permissions": map[string]interface{}{
					"print":              true,
					"modify":             false,
					"copy":               true,
					"annotate":           false,
					"fill_forms":         true,
					"extract":            true,
					"assemble":           true,
					"print_high_quality": true,
				},
			},
		},
		{
			name: "V=4 R=4 AES",
			enc: &EncryptionDictionary{
				Filter:          "Standard",
				V:               4,
				R:               4,
				Length:          128,
				P:               -4,
				EncryptMetadata: false,
				StmF:            "StdCF",
				StrF:            "StdCF",
				CF: map[string]CryptFilter{
					"StdCF": {CFM: "AESV2"},
				},
			},
			want: map[string]interface{}{
				"filter":           "Standard",
				"version":          4,
				"revision":         4,
				"key_length_bits":  128,
				"encrypt_metadata": false,
				"algorithm":        "AES-128",
				"permissions": map[string]interface{}{
					"print":              true,
					"modify":             true,
					"copy":               true,
					"annotate":           true,
					"fill_forms":         true,
					"extract":            true,
					"assemble":           true,
					"print_high_quality": true,
				},
			},
		},
		{
			name: "V=5 AES-256",
			enc: &EncryptionDictionary{
				Filter:          "Standard",
				V:               5,
				R:               5,
				Length:          256,
				P:               -44,
				EncryptMetadata: true,
			},
			want: map[string]interface{}{
				"filter":           "Standard",
				"version":          5,
				"revision":         5,
				"key_length_bits":  256,
				"encrypt_metadata": true,
				"algorithm":        "AES-256",
				"permissions": map[string]interface{}{
					"print":              true,
					"modify":             false,
					"copy":               true,
					"annotate":           false,
					"fill_forms":         true,
					"extract":            true,
					"assemble":           true,
					"print_high_quality": true,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parser.GetEncryptionInfo(tt.enc)

			// Check all expected fields
			for key, expectedValue := range tt.want {
				if gotValue, exists := got[key]; !exists {
					t.Errorf("GetEncryptionInfo() missing key %q", key)
				} else if !reflect.DeepEqual(gotValue, expectedValue) {
					t.Errorf("GetEncryptionInfo() key %q = %v, want %v", key, gotValue, expectedValue)
				}
			}
		})
	}
}

// Benchmark tests
func BenchmarkSecurityParser_ParseEncryptDict(b *testing.B) {
	parser := NewSecurityParser(newMockResolver())

	dict := map[string]interface{}{
		"Filter": "Standard",
		"V":      2,
		"R":      3,
		"Length": 40,
		"O":      []byte("owner_hash_32_bytes_long_string"),
		"U":      []byte("user_hash_32_bytes_long_string_"),
		"P":      int32(-44),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parser.ParseEncryptDict(dict)
	}
}

func BenchmarkSecurityParser_parseByteString(b *testing.B) {
	parser := NewSecurityParser(newMockResolver())
	data := []interface{}{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parser.parseByteString(data)
	}
}

func BenchmarkSecurityParser_GetEncryptionInfo(b *testing.B) {
	parser := NewSecurityParser(newMockResolver())

	enc := &EncryptionDictionary{
		Filter:          "Standard",
		V:               4,
		R:               4,
		Length:          128,
		P:               -44,
		EncryptMetadata: true,
		StmF:            "StdCF",
		StrF:            "StdCF",
		CF: map[string]CryptFilter{
			"StdCF": {CFM: "AESV2"},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parser.GetEncryptionInfo(enc)
	}
}
