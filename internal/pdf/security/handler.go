package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rc4"
	"errors"
	"fmt"
)

// SecurityHandler defines the interface for PDF security handlers
type SecurityHandler interface {
	Authenticate(password []byte) error
	DecryptObject(objNum, genNum int, data []byte) ([]byte, error)
	GetPermissions() Permissions
	IsEncrypted() bool
	IsAuthenticated() bool
}

// StandardSecurityHandler implements the Standard Security Handler (PDF 1.4/1.7 section 7.6)
type StandardSecurityHandler struct {
	encryptDict   *EncryptionDictionary
	encryptionKey []byte
	authenticated bool
	permissions   uint32
	revision      int
	keyLength     int
	fileID        []byte
}

// EncryptionDictionary represents the PDF encryption dictionary
type EncryptionDictionary struct {
	Filter          string                 // Standard
	SubFilter       string                 // Optional
	V               int                    // Version (1-5)
	Length          int                    // Key length in bits
	R               int                    // Revision (2-6)
	O               []byte                 // Owner password hash
	U               []byte                 // User password hash
	OE              []byte                 // Owner encryption key (R=6)
	UE              []byte                 // User encryption key (R=6)
	P               int32                  // Permissions
	EncryptMetadata bool                   // Whether to encrypt metadata
	StmF            string                 // Stream filter
	StrF            string                 // String filter
	CF              map[string]CryptFilter // Crypt filters
}

// CryptFilter represents a crypt filter dictionary
type CryptFilter struct {
	Type      string // Type (always "CryptFilter")
	CFM       string // Crypt filter method ("V2", "AESV2", "AESV3")
	AuthEvent string // Authorization event ("DocOpen", "EFOpen")
	Length    int    // Key length in bytes
}

// NewStandardSecurityHandler creates a new Standard Security Handler
func NewStandardSecurityHandler(encryptDict *EncryptionDictionary, fileID []byte) *StandardSecurityHandler {
	handler := &StandardSecurityHandler{
		encryptDict:   encryptDict,
		authenticated: false,
		fileID:        fileID,
	}

	if encryptDict != nil {
		handler.revision = encryptDict.R

		// Set key length based on version and revision
		if encryptDict.V == 1 {
			handler.keyLength = 40 // 40-bit RC4 for V=1
		} else if encryptDict.Length > 0 {
			handler.keyLength = encryptDict.Length
		} else {
			handler.keyLength = 40 // Default
		}
	} else {
		// Default values for nil encryption dictionary
		handler.revision = 0
		handler.keyLength = 40
	}

	return handler
}

// IsEncrypted returns true if the document is encrypted
func (h *StandardSecurityHandler) IsEncrypted() bool {
	return h.encryptDict != nil
}

// IsAuthenticated returns true if the handler has been authenticated
func (h *StandardSecurityHandler) IsAuthenticated() bool {
	return h.authenticated
}

// Authenticate attempts to authenticate with the given password
func (h *StandardSecurityHandler) Authenticate(password []byte) error {
	if h.encryptDict == nil {
		return errors.New("no encryption dictionary available")
	}

	// Try user password first
	if h.authenticateUserPassword(password) {
		h.authenticated = true
		return nil
	}

	// Try owner password
	if h.authenticateOwnerPassword(password) {
		h.authenticated = true
		return nil
	}

	return errors.New("invalid password")
}

// authenticateUserPassword attempts to authenticate as user
func (h *StandardSecurityHandler) authenticateUserPassword(password []byte) bool {
	// Compute encryption key
	key := h.computeEncryptionKey(password)
	if key == nil {
		return false
	}

	// Compute expected U value and compare with stored U
	expectedU := h.computeUserPassword(key)

	// For revision 2, compare all 32 bytes
	// For revision 3+, compare first 16 bytes
	compareLength := 32
	if h.revision >= 3 {
		compareLength = 16
	}

	if len(expectedU) >= compareLength && len(h.encryptDict.U) >= compareLength {
		for i := 0; i < compareLength; i++ {
			if expectedU[i] != h.encryptDict.U[i] {
				return false
			}
		}
		h.encryptionKey = key
		h.permissions = uint32(h.encryptDict.P)
		return true
	}

	return false
}

// authenticateOwnerPassword attempts to authenticate as owner
func (h *StandardSecurityHandler) authenticateOwnerPassword(password []byte) bool {
	// Algorithm 3 from PDF spec - recover user password from owner password
	userPassword := h.computeUserPasswordFromOwner(password)
	if userPassword == nil {
		return false
	}

	// Try authenticating with the recovered user password
	return h.authenticateUserPassword(userPassword)
}

// computeUserPasswordFromOwner implements Algorithm 3 from PDF spec
func (h *StandardSecurityHandler) computeUserPasswordFromOwner(ownerPassword []byte) []byte {
	// Step 1: Pad the owner password
	paddedOwner := h.padPassword(ownerPassword)

	// Step 2: Compute MD5 hash
	hash := md5.Sum(paddedOwner)

	// Step 3: For revision 3+, do additional hashing
	if h.revision >= 3 {
		for i := 0; i < 50; i++ {
			hash = md5.Sum(hash[:])
		}
	}

	// Step 4: Create RC4 key
	keyLen := h.keyLength / 8
	if keyLen > 16 {
		keyLen = 16
	}
	rc4Key := hash[:keyLen]

	// Step 5: Decrypt the O value to get user password
	encrypted := make([]byte, len(h.encryptDict.O))
	copy(encrypted, h.encryptDict.O)

	if h.revision >= 3 {
		// For revision 3+, iterate with different keys
		for i := 19; i >= 0; i-- {
			// Create new key by XORing with iteration count
			newKey := make([]byte, len(rc4Key))
			for j := range rc4Key {
				newKey[j] = rc4Key[j] ^ byte(i)
			}

			// Decrypt with RC4
			cipher, err := rc4.NewCipher(newKey)
			if err != nil {
				return nil
			}
			cipher.XORKeyStream(encrypted, encrypted)
		}
	} else {
		// For revision 2, single RC4 decryption
		cipher, err := rc4.NewCipher(rc4Key)
		if err != nil {
			return nil
		}
		cipher.XORKeyStream(encrypted, encrypted)
	}

	return encrypted
}

// GetPermissions returns the document permissions
func (h *StandardSecurityHandler) GetPermissions() Permissions {
	return Permissions{}.FromInt32(int32(h.permissions))
}

// DecryptObject decrypts an object's data
func (h *StandardSecurityHandler) DecryptObject(objNum, genNum int, data []byte) ([]byte, error) {
	if !h.authenticated {
		return nil, errors.New("not authenticated")
	}

	if len(data) == 0 {
		return data, nil
	}

	// Compute object-specific key
	objKey := h.computeObjectKey(objNum, genNum)

	switch h.encryptDict.V {
	case 1, 2: // RC4
		return h.decryptRC4(objKey, data)
	case 4: // AES-128
		if h.encryptDict.StmF == "StdCF" || h.encryptDict.StrF == "StdCF" {
			// Check if using AES
			if cf, ok := h.encryptDict.CF["StdCF"]; ok && cf.CFM == "AESV2" {
				return h.decryptAES128(objKey, data)
			}
		}
		// Default to RC4 for V=4
		return h.decryptRC4(objKey, data)
	case 5: // AES-256
		return h.decryptAES256(objKey, data)
	default:
		return nil, fmt.Errorf("unsupported encryption version: %d", h.encryptDict.V)
	}
}

// computeObjectKey computes the encryption key for a specific object
func (h *StandardSecurityHandler) computeObjectKey(objNum, genNum int) []byte {
	// Algorithm 1 from PDF spec
	hash := md5.New()
	hash.Write(h.encryptionKey)

	// Write object number (3 bytes, little-endian)
	hash.Write([]byte{byte(objNum), byte(objNum >> 8), byte(objNum >> 16)})

	// Write generation number (2 bytes, little-endian)
	hash.Write([]byte{byte(genNum), byte(genNum >> 8)})

	// For AES, append "sAlT"
	if h.encryptDict.V >= 4 {
		if h.usesAES() {
			hash.Write([]byte("sAlT"))
		}
	}

	digest := hash.Sum(nil)

	// Key length is min(n+5, 16) where n is the base key length
	keyLen := len(h.encryptionKey) + 5
	if keyLen > 16 {
		keyLen = 16
	}

	return digest[:keyLen]
}

// usesAES determines if AES encryption is used
func (h *StandardSecurityHandler) usesAES() bool {
	if h.encryptDict.V < 4 {
		return false
	}

	// Check crypt filters
	if cf, ok := h.encryptDict.CF["StdCF"]; ok {
		return cf.CFM == "AESV2" || cf.CFM == "AESV3"
	}

	return h.encryptDict.V == 5 // V=5 always uses AES
}

// decryptRC4 decrypts data using RC4
func (h *StandardSecurityHandler) decryptRC4(key, data []byte) ([]byte, error) {
	cipher, err := rc4.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create RC4 cipher: %w", err)
	}

	decrypted := make([]byte, len(data))
	cipher.XORKeyStream(decrypted, data)
	return decrypted, nil
}

// decryptAES128 decrypts data using AES-128 in CBC mode
func (h *StandardSecurityHandler) decryptAES128(key, data []byte) ([]byte, error) {
	return h.decryptAES(key, data)
}

// decryptAES256 decrypts data using AES-256 in CBC mode
func (h *StandardSecurityHandler) decryptAES256(key, data []byte) ([]byte, error) {
	return h.decryptAES(key, data)
}

// decryptAES decrypts data using AES in CBC mode
func (h *StandardSecurityHandler) decryptAES(key, data []byte) ([]byte, error) {
	if len(data) < aes.BlockSize {
		return nil, errors.New("ciphertext too short")
	}

	// Ensure key is the right length for AES
	if len(key) < 16 {
		// Pad key to 16 bytes
		paddedKey := make([]byte, 16)
		copy(paddedKey, key)
		key = paddedKey
	} else if len(key) > 32 {
		key = key[:32] // Truncate to max AES-256 key size
	} else if len(key) > 16 && len(key) < 32 {
		// Pad to 32 bytes for AES-256
		paddedKey := make([]byte, 32)
		copy(paddedKey, key)
		key = paddedKey
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}

	// Extract IV (first 16 bytes)
	iv := data[:aes.BlockSize]
	ciphertext := data[aes.BlockSize:]

	// Ensure ciphertext length is multiple of block size
	if len(ciphertext)%aes.BlockSize != 0 {
		return nil, errors.New("ciphertext is not a multiple of block size")
	}

	mode := cipher.NewCBCDecrypter(block, iv)
	decrypted := make([]byte, len(ciphertext))
	mode.CryptBlocks(decrypted, ciphertext)

	// Remove PKCS7 padding
	return removePKCS7Padding(decrypted)
}

// removePKCS7Padding removes PKCS7 padding from decrypted data
func removePKCS7Padding(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return data, nil
	}

	paddingLen := int(data[len(data)-1])
	if paddingLen == 0 || paddingLen > len(data) {
		// Invalid padding, return as-is
		return data, nil
	}

	// Verify padding bytes
	for i := len(data) - paddingLen; i < len(data); i++ {
		if data[i] != byte(paddingLen) {
			// Invalid padding, return as-is
			return data, nil
		}
	}

	return data[:len(data)-paddingLen], nil
}
