package security

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/binary"
)

// PDF password padding string as specified in PDF spec
var passwordPadding = []byte{
	0x28, 0xBF, 0x4E, 0x5E, 0x4E, 0x75, 0x8A, 0x41,
	0x64, 0x00, 0x4E, 0x56, 0xFF, 0xFA, 0x01, 0x08,
	0x2E, 0x2E, 0x00, 0xB6, 0xD0, 0x68, 0x3E, 0x80,
	0x2F, 0x0C, 0xA9, 0xFE, 0x64, 0x53, 0x69, 0x7A,
}

// computeEncryptionKey computes the encryption key based on the revision
func (h *StandardSecurityHandler) computeEncryptionKey(password []byte) []byte {
	switch h.revision {
	case 2, 3, 4:
		return h.computeRC4Key(password)
	case 5, 6:
		return h.computeAESKey(password)
	default:
		return nil
	}
}

// computeRC4Key implements Algorithm 2 from PDF specification
// Used for Standard Security Handler revisions 2, 3, and 4
func (h *StandardSecurityHandler) computeRC4Key(password []byte) []byte {
	// Step 1: Pad the password using the standard padding string
	padded := h.padPassword(password)

	// Step 2: Initialize MD5 hash and add padded password
	hash := md5.New()
	hash.Write(padded)

	// Step 3: Add owner password string (O entry from encryption dictionary)
	hash.Write(h.encryptDict.O)

	// Step 4: Add permission value (P entry) as 4 bytes, low-order byte first
	hash.Write(intToBytes(h.encryptDict.P))

	// Step 5: Add file identifier (first element of ID array from trailer)
	hash.Write(h.fileID)

	// Step 6: (Revision 4 or greater) If document metadata is not encrypted,
	// add 4 bytes with value 0xFFFFFFFF
	if h.revision >= 4 && !h.encryptDict.EncryptMetadata {
		hash.Write([]byte{0xff, 0xff, 0xff, 0xff})
	}

	// Step 7: Finish the hash
	digest := hash.Sum(nil)

	// Step 8: (Revision 3 or greater) Do the following 50 times:
	// Take the output from the previous MD5 hash and pass the first n bytes
	// as input to a new MD5 hash, where n is the key length in bytes
	if h.revision >= 3 {
		keyLength := h.keyLength / 8
		for i := 0; i < 50; i++ {
			hash.Reset()
			hash.Write(digest[:keyLength])
			digest = hash.Sum(nil)
		}
	}

	// Step 9: Set the encryption key to the first n bytes of the output
	// from the final MD5 hash, where n is the key length in bytes
	keyLength := h.keyLength / 8
	if keyLength > len(digest) {
		keyLength = len(digest)
	}

	return digest[:keyLength]
}

// computeAESKey implements key computation for AES encryption (revisions 5 and 6)
func (h *StandardSecurityHandler) computeAESKey(password []byte) []byte {
	// For revision 5 and 6, the encryption key is derived differently
	// This is a simplified implementation - full revision 6 support would require
	// more complex algorithms involving SHA-256, random salts, etc.

	if h.revision == 5 {
		// For revision 5, use SHA-256 instead of MD5
		padded := h.padPassword(password)
		hash := sha256.New()
		hash.Write(padded)
		hash.Write(h.encryptDict.O)
		hash.Write(intToBytes(h.encryptDict.P))
		hash.Write(h.fileID)

		digest := hash.Sum(nil)
		return digest[:32] // AES-256 key
	}

	// For revision 6, this would require more complex implementation
	// involving iterative hashing with random data
	return h.computeRC4Key(password) // Fallback for now
}

// computeUserPassword implements Algorithm 4 (revision 2) and Algorithm 5 (revision 3+)
// from the PDF specification to compute the expected U value
func (h *StandardSecurityHandler) computeUserPassword(encryptionKey []byte) []byte {
	if h.revision == 2 {
		// Algorithm 4: Encrypt the padding string using RC4 with the encryption key
		cipher, err := newRC4Cipher(encryptionKey)
		if err != nil {
			return nil
		}

		result := make([]byte, 32)
		cipher.XORKeyStream(result, passwordPadding)
		return result
	}

	// Algorithm 5 for revision 3 and above
	// Step 1: Create MD5 hash of the padding string and file identifier
	hash := md5.New()
	hash.Write(passwordPadding)
	hash.Write(h.fileID)
	digest := hash.Sum(nil)

	// Step 2: Encrypt the 16-byte result using RC4 with the encryption key
	cipher, err := newRC4Cipher(encryptionKey)
	if err != nil {
		return nil
	}

	encrypted := make([]byte, 16)
	cipher.XORKeyStream(encrypted, digest)

	// Step 3: Do the following 19 times: Take the output from the previous
	// invocation and encrypt it using RC4 with a key formed by taking each
	// byte of the original encryption key and performing an XOR operation
	// between that byte and the single-byte value of the iteration counter
	for i := 1; i <= 19; i++ {
		// Create new key by XORing with iteration counter
		newKey := make([]byte, len(encryptionKey))
		for j := range encryptionKey {
			newKey[j] = encryptionKey[j] ^ byte(i)
		}

		// Encrypt with the new key
		cipher, err := newRC4Cipher(newKey)
		if err != nil {
			return nil
		}
		cipher.XORKeyStream(encrypted, encrypted)
	}

	// Step 4: Append 16 bytes of arbitrary padding to complete 32-byte result
	result := make([]byte, 32)
	copy(result, encrypted)
	// Fill remaining bytes with arbitrary data (we'll use zeros)
	for i := 16; i < 32; i++ {
		result[i] = 0
	}

	return result
}

// padPassword pads a password to exactly 32 bytes using the standard padding string
func (h *StandardSecurityHandler) padPassword(password []byte) []byte {
	result := make([]byte, 32)

	if len(password) >= 32 {
		// If password is 32 bytes or longer, use first 32 bytes
		copy(result, password[:32])
	} else {
		// If password is shorter, copy it and pad with standard padding
		copy(result, password)
		copy(result[len(password):], passwordPadding[:32-len(password)])
	}

	return result
}

// intToBytes converts an int32 to a 4-byte array in little-endian format
func intToBytes(value int32) []byte {
	bytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(bytes, uint32(value))
	return bytes
}

// newRC4Cipher creates a new RC4 cipher with error handling
func newRC4Cipher(key []byte) (streamCipher, error) {
	// Simple RC4 implementation for PDF encryption
	return &rc4Cipher{key: key}, nil
}

// rc4Cipher is a simple RC4 implementation
type rc4Cipher struct {
	key  []byte
	s    [256]byte
	i, j byte
}

// streamCipher interface for RC4
type streamCipher interface {
	XORKeyStream(dst, src []byte)
}

// XORKeyStream implements the RC4 key stream XOR operation
func (c *rc4Cipher) XORKeyStream(dst, src []byte) {
	if len(dst) < len(src) {
		panic("dst buffer too small")
	}

	// Initialize S-box if not already done
	if c.s[0] == 0 && c.s[1] == 0 {
		c.initSBox()
	}

	// Generate keystream and XOR with source
	for k := 0; k < len(src); k++ {
		c.i++
		c.j += c.s[c.i]
		c.s[c.i], c.s[c.j] = c.s[c.j], c.s[c.i]
		keyByte := c.s[byte(c.s[c.i]+c.s[c.j])]
		dst[k] = src[k] ^ keyByte
	}
}

// initSBox initializes the RC4 S-box
func (c *rc4Cipher) initSBox() {
	// Initialize S-box
	for i := 0; i < 256; i++ {
		c.s[i] = byte(i)
	}

	// Key-scheduling algorithm
	j := byte(0)
	for i := 0; i < 256; i++ {
		j += c.s[i] + c.key[i%len(c.key)]
		c.s[i], c.s[j] = c.s[j], c.s[i]
	}

	c.i = 0
	c.j = 0
}

// validateKeyLength ensures the encryption key length is valid
func (h *StandardSecurityHandler) validateKeyLength() bool {
	switch h.encryptDict.V {
	case 1:
		return h.keyLength == 40
	case 2:
		return h.keyLength >= 40 && h.keyLength <= 128 && h.keyLength%8 == 0
	case 4:
		return h.keyLength == 128
	case 5:
		return h.keyLength == 256
	default:
		return false
	}
}

// computeOwnerKey implements Algorithm 3 from PDF specification
// Used to compute the O entry in the encryption dictionary
func (h *StandardSecurityHandler) computeOwnerKey(ownerPassword, userPassword []byte) []byte {
	// Step 1: Pad the owner password
	paddedOwner := h.padPassword(ownerPassword)

	// Step 2: Initialize MD5 hash and add padded owner password
	hash := md5.New()
	hash.Write(paddedOwner)
	digest := hash.Sum(nil)

	// Step 3: (Revision 3 or greater) Do the following 50 times
	if h.revision >= 3 {
		for i := 0; i < 50; i++ {
			hash.Reset()
			hash.Write(digest)
			digest = hash.Sum(nil)
		}
	}

	// Step 4: Create RC4 encryption key from first n bytes of hash
	keyLength := h.keyLength / 8
	if keyLength > 16 {
		keyLength = 16
	}
	rc4Key := digest[:keyLength]

	// Step 5: Pad the user password
	paddedUser := h.padPassword(userPassword)

	// Step 6: Encrypt the padded user password using RC4
	encrypted := make([]byte, len(paddedUser))
	copy(encrypted, paddedUser)

	if h.revision >= 3 {
		// For revision 3+, iterate 20 times with different keys
		for i := 0; i < 20; i++ {
			// Create new key by XORing with iteration counter
			newKey := make([]byte, len(rc4Key))
			for j := range rc4Key {
				newKey[j] = rc4Key[j] ^ byte(i)
			}

			// Encrypt with RC4
			cipher, err := newRC4Cipher(newKey)
			if err != nil {
				return nil
			}
			cipher.XORKeyStream(encrypted, encrypted)
		}
	} else {
		// For revision 2, single RC4 encryption
		cipher, err := newRC4Cipher(rc4Key)
		if err != nil {
			return nil
		}
		cipher.XORKeyStream(encrypted, encrypted)
	}

	return encrypted
}
