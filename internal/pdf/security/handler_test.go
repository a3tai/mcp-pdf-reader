package security

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"testing"
)

// Test data and constants
var (
	testFileID = []byte{0x01, 0x23, 0x45, 0x67, 0x89, 0xAB, 0xCD, 0xEF, 0xFE, 0xDC, 0xBA, 0x98, 0x76, 0x54, 0x32, 0x10}

	// Sample encryption dictionary for V=2, R=3 (40-bit RC4)
	testEncryptDictV2 = &EncryptionDictionary{
		Filter:          "Standard",
		V:               2,
		Length:          40,
		R:               3,
		O:               []byte{0x28, 0xBF, 0x4E, 0x5E, 0x4E, 0x75, 0x8A, 0x41, 0x64, 0x00, 0x4E, 0x56, 0xFF, 0xFA, 0x01, 0x08, 0x2E, 0x2E, 0x00, 0xB6, 0xD0, 0x68, 0x3E, 0x80, 0x2F, 0x0C, 0xA9, 0xFE, 0x64, 0x53, 0x69, 0x7A},
		U:               []byte{0x44, 0x6D, 0x8D, 0x99, 0x90, 0xE7, 0x23, 0x4F, 0x8C, 0x86, 0x8C, 0x8F, 0x63, 0x9B, 0x2C, 0x12, 0x28, 0xBF, 0x4E, 0x5E, 0x4E, 0x75, 0x8A, 0x41, 0x64, 0x00, 0x4E, 0x56, 0xFF, 0xFA, 0x01, 0x08},
		P:               -44,
		EncryptMetadata: true,
	}

	// Sample encryption dictionary for V=4, R=4 (128-bit AES)
	testEncryptDictV4 = &EncryptionDictionary{
		Filter:          "Standard",
		V:               4,
		Length:          128,
		R:               4,
		O:               []byte{0x28, 0xBF, 0x4E, 0x5E, 0x4E, 0x75, 0x8A, 0x41, 0x64, 0x00, 0x4E, 0x56, 0xFF, 0xFA, 0x01, 0x08, 0x2E, 0x2E, 0x00, 0xB6, 0xD0, 0x68, 0x3E, 0x80, 0x2F, 0x0C, 0xA9, 0xFE, 0x64, 0x53, 0x69, 0x7A},
		U:               []byte{0x44, 0x6D, 0x8D, 0x99, 0x90, 0xE7, 0x23, 0x4F, 0x8C, 0x86, 0x8C, 0x8F, 0x63, 0x9B, 0x2C, 0x12, 0x28, 0xBF, 0x4E, 0x5E, 0x4E, 0x75, 0x8A, 0x41, 0x64, 0x00, 0x4E, 0x56, 0xFF, 0xFA, 0x01, 0x08},
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
	}

	testPassword      = []byte("user")
	testOwnerPassword = []byte("owner")
	emptyPassword     = []byte("")
	longPassword      = []byte("this_is_a_very_long_password_that_exceeds_32_bytes_and_should_be_truncated")
)

func TestNewStandardSecurityHandler(t *testing.T) {
	tests := []struct {
		name        string
		encryptDict *EncryptionDictionary
		fileID      []byte
		wantErr     bool
	}{
		{
			name:        "Valid V=2 encryption dictionary",
			encryptDict: testEncryptDictV2,
			fileID:      testFileID,
			wantErr:     false,
		},
		{
			name:        "Valid V=4 encryption dictionary",
			encryptDict: testEncryptDictV4,
			fileID:      testFileID,
			wantErr:     false,
		},
		{
			name:        "Nil encryption dictionary",
			encryptDict: nil,
			fileID:      testFileID,
			wantErr:     false, // Constructor doesn't validate, just stores
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewStandardSecurityHandler(tt.encryptDict, tt.fileID)

			if handler == nil {
				t.Fatal("NewStandardSecurityHandler returned nil")
			}

			if tt.encryptDict != nil {
				if handler.encryptDict != tt.encryptDict {
					t.Error("Handler does not store encryption dictionary correctly")
				}

				if !bytes.Equal(handler.fileID, tt.fileID) {
					t.Error("Handler does not store file ID correctly")
				}

				if handler.authenticated {
					t.Error("Handler should not be authenticated initially")
				}
			}
		})
	}
}

func TestStandardSecurityHandler_IsEncrypted(t *testing.T) {
	tests := []struct {
		name        string
		encryptDict *EncryptionDictionary
		want        bool
	}{
		{
			name:        "With encryption dictionary",
			encryptDict: testEncryptDictV2,
			want:        true,
		},
		{
			name:        "Without encryption dictionary",
			encryptDict: nil,
			want:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewStandardSecurityHandler(tt.encryptDict, testFileID)

			if got := handler.IsEncrypted(); got != tt.want {
				t.Errorf("IsEncrypted() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStandardSecurityHandler_IsAuthenticated(t *testing.T) {
	handler := NewStandardSecurityHandler(testEncryptDictV2, testFileID)

	if handler.IsAuthenticated() {
		t.Error("Handler should not be authenticated initially")
	}

	// Manually set authenticated for testing
	handler.authenticated = true

	if !handler.IsAuthenticated() {
		t.Error("Handler should be authenticated after setting flag")
	}
}

func TestStandardSecurityHandler_padPassword(t *testing.T) {
	handler := NewStandardSecurityHandler(testEncryptDictV2, testFileID)

	tests := []struct {
		name     string
		password []byte
		wantLen  int
	}{
		{
			name:     "Empty password",
			password: []byte{},
			wantLen:  32,
		},
		{
			name:     "Short password",
			password: []byte("test"),
			wantLen:  32,
		},
		{
			name:     "32-byte password",
			password: make([]byte, 32),
			wantLen:  32,
		},
		{
			name:     "Long password",
			password: make([]byte, 50),
			wantLen:  32,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			padded := handler.padPassword(tt.password)

			if len(padded) != tt.wantLen {
				t.Errorf("padPassword() length = %d, want %d", len(padded), tt.wantLen)
			}

			// Check that short passwords are padded with the standard padding
			if len(tt.password) < 32 {
				expectedPadding := passwordPadding[:32-len(tt.password)]
				actualPadding := padded[len(tt.password):]

				if !bytes.Equal(actualPadding, expectedPadding) {
					t.Error("Password not padded correctly with standard padding")
				}
			}
		})
	}
}

func TestStandardSecurityHandler_computeObjectKey(t *testing.T) {
	handler := NewStandardSecurityHandler(testEncryptDictV2, testFileID)
	handler.encryptionKey = []byte{0x01, 0x02, 0x03, 0x04, 0x05}

	tests := []struct {
		name   string
		objNum int
		genNum int
	}{
		{"Object 1 Gen 0", 1, 0},
		{"Object 100 Gen 5", 100, 5},
		{"Object 65535 Gen 65535", 65535, 65535},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key1 := handler.computeObjectKey(tt.objNum, tt.genNum)
			key2 := handler.computeObjectKey(tt.objNum, tt.genNum)

			// Should be deterministic
			if !bytes.Equal(key1, key2) {
				t.Error("computeObjectKey is not deterministic")
			}

			// Should be different for different objects
			if tt.objNum != 1 || tt.genNum != 0 {
				keyDifferent := handler.computeObjectKey(1, 0)
				if bytes.Equal(key1, keyDifferent) {
					t.Error("computeObjectKey should produce different keys for different objects")
				}
			}

			// Key length should be appropriate
			expectedLen := len(handler.encryptionKey) + 5
			if expectedLen > 16 {
				expectedLen = 16
			}

			if len(key1) != expectedLen {
				t.Errorf("Object key length = %d, want %d", len(key1), expectedLen)
			}
		})
	}
}

func TestStandardSecurityHandler_decryptRC4(t *testing.T) {
	handler := NewStandardSecurityHandler(testEncryptDictV2, testFileID)

	tests := []struct {
		name      string
		key       []byte
		plaintext []byte
	}{
		{
			name:      "Simple text",
			key:       []byte{0x01, 0x02, 0x03, 0x04, 0x05},
			plaintext: []byte("Hello, World!"),
		},
		{
			name:      "Empty data",
			key:       []byte{0x01, 0x02, 0x03, 0x04, 0x05},
			plaintext: []byte{},
		},
		{
			name:      "Binary data",
			key:       []byte{0x01, 0x02, 0x03, 0x04, 0x05},
			plaintext: []byte{0x00, 0x01, 0x02, 0x03, 0xFF, 0xFE, 0xFD},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// First encrypt the data (RC4 is symmetric)
			encrypted, err := handler.decryptRC4(tt.key, tt.plaintext)
			if err != nil {
				t.Fatalf("Failed to encrypt data: %v", err)
			}

			// Then decrypt it back
			decrypted, err := handler.decryptRC4(tt.key, encrypted)
			if err != nil {
				t.Fatalf("Failed to decrypt data: %v", err)
			}

			if !bytes.Equal(decrypted, tt.plaintext) {
				t.Errorf("Decrypted data does not match original: got %v, want %v", decrypted, tt.plaintext)
			}
		})
	}
}

func TestStandardSecurityHandler_decryptAES(t *testing.T) {
	handler := NewStandardSecurityHandler(testEncryptDictV4, testFileID)

	tests := []struct {
		name      string
		keySize   int
		plaintext []byte
	}{
		{
			name:      "AES-128 simple text",
			keySize:   16,
			plaintext: []byte("Hello, World! This is a test message for AES encryption."),
		},
		{
			name:      "AES-256 simple text",
			keySize:   32,
			plaintext: []byte("Hello, World! This is a test message for AES encryption."),
		},
		{
			name:      "AES block-aligned data",
			keySize:   16,
			plaintext: make([]byte, 32), // Exactly 2 blocks
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a key of the specified size
			key := make([]byte, tt.keySize)
			if _, err := rand.Read(key); err != nil {
				t.Fatalf("Failed to generate random key: %v", err)
			}

			// Create test data by encrypting with AES-CBC
			block, err := aes.NewCipher(key)
			if err != nil {
				t.Fatalf("Failed to create AES cipher: %v", err)
			}

			// Add PKCS7 padding
			padded := addPKCS7Padding(tt.plaintext, aes.BlockSize)

			// Generate random IV
			iv := make([]byte, aes.BlockSize)
			if _, err := rand.Read(iv); err != nil {
				t.Fatalf("Failed to generate random IV: %v", err)
			}

			// Encrypt
			encrypted := make([]byte, len(padded))
			mode := cipher.NewCBCEncrypter(block, iv)
			mode.CryptBlocks(encrypted, padded)

			// Prepend IV to create the format expected by decryptAES
			ciphertext := append(iv, encrypted...)

			// Test decryption
			decrypted, err := handler.decryptAES(key, ciphertext)
			if err != nil {
				t.Fatalf("Failed to decrypt AES data: %v", err)
			}

			if !bytes.Equal(decrypted, tt.plaintext) {
				t.Errorf("Decrypted data does not match original")
				t.Logf("Original:  %x", tt.plaintext)
				t.Logf("Decrypted: %x", decrypted)
			}
		})
	}
}

func TestStandardSecurityHandler_removePKCS7Padding(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected []byte
		wantErr  bool
	}{
		{
			name:     "Valid padding",
			data:     []byte{0x01, 0x02, 0x03, 0x04, 0x04, 0x04, 0x04, 0x04},
			expected: []byte{0x01, 0x02, 0x03, 0x04},
			wantErr:  false,
		},
		{
			name:     "Single byte padding",
			data:     []byte{0x01, 0x02, 0x03, 0x01},
			expected: []byte{0x01, 0x02, 0x03},
			wantErr:  false,
		},
		{
			name:     "Full block padding",
			data:     []byte{0x10, 0x10, 0x10, 0x10, 0x10, 0x10, 0x10, 0x10, 0x10, 0x10, 0x10, 0x10, 0x10, 0x10, 0x10, 0x10},
			expected: []byte{},
			wantErr:  false,
		},
		{
			name:     "Invalid padding - wrong values",
			data:     []byte{0x01, 0x02, 0x03, 0x04, 0x04, 0x04, 0x04, 0x03},
			expected: []byte{0x01, 0x02, 0x03, 0x04, 0x04, 0x04, 0x04, 0x03}, // Return as-is
			wantErr:  false,
		},
		{
			name:     "Empty data",
			data:     []byte{},
			expected: []byte{},
			wantErr:  false,
		},
		{
			name:     "Zero padding length",
			data:     []byte{0x01, 0x02, 0x03, 0x00},
			expected: []byte{0x01, 0x02, 0x03, 0x00}, // Return as-is
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := removePKCS7Padding(tt.data)

			if (err != nil) != tt.wantErr {
				t.Errorf("removePKCS7Padding() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !bytes.Equal(result, tt.expected) {
				t.Errorf("removePKCS7Padding() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestStandardSecurityHandler_GetPermissions(t *testing.T) {
	handler := NewStandardSecurityHandler(testEncryptDictV2, testFileID)
	handler.permissions = uint32(testEncryptDictV2.P)

	perms := handler.GetPermissions()

	// Test specific permission bits based on P = -44
	// -44 in binary (as uint32): 11111111111111111111111111010100
	// This should allow print (bit 3) but deny modify (bit 4) and annotate (bit 6)
	if !perms.Print {
		t.Error("Print permission should be allowed")
	}
	if perms.Modify {
		t.Error("Modify permission should be denied")
	}
	if !perms.Copy {
		t.Error("Copy permission should be allowed")
	}
	if perms.Annotate {
		t.Error("Annotate permission should be denied")
	}
}

func TestStandardSecurityHandler_DecryptObject_NotAuthenticated(t *testing.T) {
	handler := NewStandardSecurityHandler(testEncryptDictV2, testFileID)

	data := []byte("test data")
	_, err := handler.DecryptObject(1, 0, data)

	if err == nil {
		t.Error("DecryptObject should fail when not authenticated")
	}

	if err.Error() != "not authenticated" {
		t.Errorf("Expected 'not authenticated' error, got: %v", err)
	}
}

func TestStandardSecurityHandler_DecryptObject_EmptyData(t *testing.T) {
	handler := NewStandardSecurityHandler(testEncryptDictV2, testFileID)
	handler.authenticated = true
	handler.encryptionKey = []byte{0x01, 0x02, 0x03, 0x04, 0x05}

	result, err := handler.DecryptObject(1, 0, []byte{})
	if err != nil {
		t.Errorf("DecryptObject should not fail with empty data: %v", err)
	}

	if len(result) != 0 {
		t.Error("DecryptObject should return empty data for empty input")
	}
}

func TestStandardSecurityHandler_UnsupportedVersion(t *testing.T) {
	unsupportedDict := &EncryptionDictionary{
		Filter: "Standard",
		V:      6, // Unsupported version
		R:      7,
		O:      make([]byte, 32),
		U:      make([]byte, 32),
		P:      -44,
	}

	handler := NewStandardSecurityHandler(unsupportedDict, testFileID)
	handler.authenticated = true
	handler.encryptionKey = []byte{0x01, 0x02, 0x03, 0x04, 0x05}

	_, err := handler.DecryptObject(1, 0, []byte("test"))

	if err == nil {
		t.Error("DecryptObject should fail with unsupported version")
	}

	if err.Error() != "unsupported encryption version: 6" {
		t.Errorf("Expected unsupported version error, got: %v", err)
	}
}

// Helper function to add PKCS7 padding for testing
func addPKCS7Padding(data []byte, blockSize int) []byte {
	padding := blockSize - len(data)%blockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(data, padtext...)
}

// Benchmark tests
func BenchmarkStandardSecurityHandler_computeObjectKey(b *testing.B) {
	handler := NewStandardSecurityHandler(testEncryptDictV2, testFileID)
	handler.encryptionKey = []byte{0x01, 0x02, 0x03, 0x04, 0x05}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		handler.computeObjectKey(i, 0)
	}
}

func BenchmarkStandardSecurityHandler_decryptRC4(b *testing.B) {
	handler := NewStandardSecurityHandler(testEncryptDictV2, testFileID)
	key := []byte{0x01, 0x02, 0x03, 0x04, 0x05}
	data := make([]byte, 1024) // 1KB of data

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		handler.decryptRC4(key, data)
	}
}

func BenchmarkStandardSecurityHandler_decryptAES(b *testing.B) {
	handler := NewStandardSecurityHandler(testEncryptDictV4, testFileID)
	key := make([]byte, 16)
	data := make([]byte, 1024+16) // 1KB + IV

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		handler.decryptAES(key, data)
	}
}

func BenchmarkStandardSecurityHandler_padPassword(b *testing.B) {
	handler := NewStandardSecurityHandler(testEncryptDictV2, testFileID)
	password := []byte("testpassword")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		handler.padPassword(password)
	}
}

// Test error conditions and edge cases
func TestStandardSecurityHandler_ErrorConditions(t *testing.T) {
	t.Run("Authenticate with nil encryption dictionary", func(t *testing.T) {
		handler := NewStandardSecurityHandler(nil, testFileID)

		err := handler.Authenticate(testPassword)
		if err == nil {
			t.Error("Authenticate should fail with nil encryption dictionary")
		}
	})

	t.Run("Decrypt AES with short ciphertext", func(t *testing.T) {
		handler := NewStandardSecurityHandler(testEncryptDictV4, testFileID)
		handler.authenticated = true
		handler.encryptionKey = make([]byte, 16)

		// Ciphertext shorter than AES block size
		shortData := make([]byte, 8)
		_, err := handler.decryptAES(handler.encryptionKey, shortData)

		if err == nil {
			t.Error("decryptAES should fail with short ciphertext")
		}
	})

	t.Run("Decrypt AES with non-block-aligned ciphertext", func(t *testing.T) {
		handler := NewStandardSecurityHandler(testEncryptDictV4, testFileID)
		handler.authenticated = true
		handler.encryptionKey = make([]byte, 16)

		// Ciphertext not aligned to block size (after IV removal)
		data := make([]byte, 16+10) // IV + 10 bytes (not block-aligned)
		_, err := handler.decryptAES(handler.encryptionKey, data)

		if err == nil {
			t.Error("decryptAES should fail with non-block-aligned ciphertext")
		}
	})
}

// Test various password scenarios
func TestStandardSecurityHandler_PasswordScenarios(t *testing.T) {
	tests := []struct {
		name     string
		password []byte
	}{
		{"Empty password", []byte("")},
		{"Short password", []byte("abc")},
		{"Exactly 32-byte password", make([]byte, 32)},
		{"Long password", make([]byte, 64)},
		{"Unicode password", []byte("πάσσωορδ")},
		{"Binary password", []byte{0x00, 0x01, 0x02, 0xFF, 0xFE, 0xFD}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewStandardSecurityHandler(testEncryptDictV2, testFileID)

			// Test password padding
			padded := handler.padPassword(tt.password)
			if len(padded) != 32 {
				t.Errorf("Padded password length should be 32, got %d", len(padded))
			}

			// Test that padding is deterministic
			padded2 := handler.padPassword(tt.password)
			if !bytes.Equal(padded, padded2) {
				t.Error("Password padding should be deterministic")
			}
		})
	}
}
