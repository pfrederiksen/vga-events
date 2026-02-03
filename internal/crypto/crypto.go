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

// Encryptor handles encryption and decryption of sensitive data
type Encryptor struct {
	key []byte
}

// NewEncryptor creates a new encryptor with the given passphrase
func NewEncryptor(passphrase string) *Encryptor {
	if passphrase == "" {
		return nil
	}

	// Derive key from passphrase using PBKDF2
	// Note: In production, salt should be stored alongside encrypted data
	// For this use case, we use a fixed salt derived from the passphrase itself
	salt := sha256.Sum256([]byte(passphrase + "vga-events-salt"))

	key := pbkdf2.Key([]byte(passphrase), salt[:], iterations, keySize, sha256.New)

	return &Encryptor{key: key}
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
func (e *Encryptor) Decrypt(ciphertext string) (string, error) {
	if e == nil || e.key == nil {
		return ciphertext, nil // No decryption if encryptor not configured
	}

	if ciphertext == "" {
		return "", nil
	}

	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		// If it's not valid base64, assume it's unencrypted (for backward compatibility)
		return ciphertext, nil
	}

	block, err := aes.NewCipher(e.key)
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
		// If decryption fails, assume it's unencrypted (for backward compatibility)
		return ciphertext, nil
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
	if e == nil || len(m) == 0 {
		return m, nil
	}

	result := make(map[string]string, len(m))
	for k, v := range m {
		decrypted, err := e.Decrypt(v)
		if err != nil {
			return nil, err
		}
		result[k] = decrypted
	}
	return result, nil
}
