package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"

	"golang.org/x/crypto/pbkdf2"
)

const (
	saltSize   = 16
	iterations = 100000
	keySize    = 32 // AES-256
)

// fixedSalt is a static salt for PBKDF2 key derivation
// Note: Using a fixed salt is acceptable for this use case because:
// 1. We're deriving encryption keys, not storing password hashes
// 2. PBKDF2 with 100k iterations provides sufficient computational cost
// 3. The encryption key protects user data, not authenticate users
// 4. Storing per-value salts would require schema changes and is unnecessary here
var fixedSalt = []byte("vga-events-encryption-salt-v1")

// legacySalt computes the old salt format for backward compatibility
// This allows decrypting data encrypted with the previous salt derivation method
func legacySalt(passphrase string) []byte {
	sum := sha256.Sum256([]byte(passphrase + "vga-events-salt"))
	return sum[:]
}

// Encryptor handles encryption and decryption of sensitive data
type Encryptor struct {
	key       []byte
	legacyKey []byte // For decrypting old data during migration
}

// NewEncryptor creates a new encryptor with the given passphrase
func NewEncryptor(passphrase string) *Encryptor {
	if passphrase == "" {
		return nil
	}

	// Derive key from passphrase using PBKDF2 with fixed salt
	// Using PBKDF2 with 100,000 iterations provides strong key derivation
	// even with a fixed salt, which is acceptable for encrypting stored data
	key := pbkdf2.Key([]byte(passphrase), fixedSalt, iterations, keySize, sha256.New)

	// Also derive legacy key for backward compatibility during migration
	legacyKey := pbkdf2.Key([]byte(passphrase), legacySalt(passphrase), iterations, keySize, sha256.New)

	return &Encryptor{
		key:       key,
		legacyKey: legacyKey,
	}
}

// Encrypt encrypts plaintext using AES-GCM
func (e *Encryptor) Encrypt(plaintext string) (string, error) {
	if e == nil || e.key == nil {
		return plaintext, nil // No encryption if encryptor not configured
	}

	if plaintext == "" {
		return "", nil
	}

	block, err := aes.NewCipher(e.key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt decrypts ciphertext using AES-GCM
// Automatically tries legacy key if new key fails (seamless migration)
func (e *Encryptor) Decrypt(ciphertext string) (string, error) {
	plaintext, _, err := e.DecryptWithMigration(ciphertext)
	return plaintext, err
}

// DecryptWithMigration decrypts ciphertext and returns whether migration is needed
// Returns (plaintext, needsMigration, error)
func (e *Encryptor) DecryptWithMigration(ciphertext string) (string, bool, error) {
	if e == nil || e.key == nil {
		return ciphertext, false, nil // No decryption if encryptor not configured
	}

	if ciphertext == "" {
		return "", false, nil
	}

	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		// If it's not valid base64, assume it's unencrypted (for backward compatibility)
		return ciphertext, false, nil
	}

	// Try with new key first
	plaintext, err := e.decryptWithKey(data, e.key)
	if err == nil {
		return plaintext, false, nil // Successfully decrypted with new key
	}

	// Try with legacy key for backward compatibility
	if e.legacyKey != nil {
		plaintext, err := e.decryptWithKey(data, e.legacyKey)
		if err == nil {
			// Successfully decrypted with legacy key - migration needed
			return plaintext, true, nil
		}
	}

	// If both keys fail, assume it's unencrypted (for backward compatibility)
	return ciphertext, false, nil
}

// decryptWithKey attempts decryption with a specific key
func (e *Encryptor) decryptWithKey(data []byte, key []byte) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", errors.New("ciphertext too short")
	}

	nonce, cipherData := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, cipherData, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

// EncryptMap encrypts all values in a map
func (e *Encryptor) EncryptMap(m map[string]string) (map[string]string, error) {
	if e == nil || len(m) == 0 {
		return m, nil
	}

	result := make(map[string]string, len(m))
	for k, v := range m {
		encrypted, err := e.Encrypt(v)
		if err != nil {
			return nil, err
		}
		result[k] = encrypted
	}
	return result, nil
}

// DecryptMap decrypts all values in a map
func (e *Encryptor) DecryptMap(m map[string]string) (map[string]string, error) {
	result, _, err := e.DecryptMapWithMigration(m)
	return result, err
}

// DecryptMapWithMigration decrypts all values in a map and returns whether migration is needed
// Returns (decryptedMap, needsMigration, error)
func (e *Encryptor) DecryptMapWithMigration(m map[string]string) (map[string]string, bool, error) {
	if e == nil || len(m) == 0 {
		return m, false, nil
	}

	result := make(map[string]string, len(m))
	anyMigration := false

	for k, v := range m {
		decrypted, needsMigration, err := e.DecryptWithMigration(v)
		if err != nil {
			return nil, false, err
		}
		result[k] = decrypted
		if needsMigration {
			anyMigration = true
		}
	}

	return result, anyMigration, nil
}
