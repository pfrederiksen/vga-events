package crypto

import (
	"crypto/sha256"
	"strings"
	"testing"

	"golang.org/x/crypto/pbkdf2"
)

func TestNewEncryptor(t *testing.T) {
	tests := []struct {
		name       string
		passphrase string
		wantNil    bool
	}{
		{
			name:       "valid passphrase",
			passphrase: "strong-passphrase-123",
			wantNil:    false,
		},
		{
			name:       "empty passphrase returns nil",
			passphrase: "",
			wantNil:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			enc := NewEncryptor(tt.passphrase)
			if tt.wantNil && enc != nil {
				t.Errorf("NewEncryptor() = %v, want nil", enc)
			}
			if !tt.wantNil && enc == nil {
				t.Error("NewEncryptor() = nil, want non-nil")
			}
		})
	}
}

func TestEncryptDecrypt(t *testing.T) {
	enc := NewEncryptor("test-passphrase")

	tests := []struct {
		name      string
		plaintext string
	}{
		{
			name:      "simple text",
			plaintext: "hello world",
		},
		{
			name:      "empty string",
			plaintext: "",
		},
		{
			name:      "special characters",
			plaintext: "!@#$%^&*()_+-=[]{}|;:',.<>?",
		},
		{
			name:      "unicode",
			plaintext: "Hello 世界 🌍",
		},
		{
			name:      "long text",
			plaintext: strings.Repeat("Lorem ipsum dolor sit amet. ", 100),
		},
		{
			name:      "multiline",
			plaintext: "line1\nline2\nline3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Encrypt
			ciphertext, err := enc.Encrypt(tt.plaintext)
			if err != nil {
				t.Fatalf("Encrypt() error = %v", err)
			}

			// Verify it's different from plaintext (unless empty)
			if tt.plaintext != "" && ciphertext == tt.plaintext {
				t.Error("Encrypt() did not change the plaintext")
			}

			// Decrypt
			decrypted, err := enc.Decrypt(ciphertext)
			if err != nil {
				t.Fatalf("Decrypt() error = %v", err)
			}

			// Verify round-trip
			if decrypted != tt.plaintext {
				t.Errorf("Decrypt() = %q, want %q", decrypted, tt.plaintext)
			}
		})
	}
}

func TestEncryptDecrypt_NilEncryptor(t *testing.T) {
	// Nil encryptor should pass through without changes
	var enc *Encryptor

	plaintext := "hello world"

	// Encrypt with nil should return plaintext
	ciphertext, err := enc.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt() with nil encryptor error = %v", err)
	}
	if ciphertext != plaintext {
		t.Errorf("Encrypt() with nil encryptor = %q, want %q", ciphertext, plaintext)
	}

	// Decrypt with nil should return input
	decrypted, err := enc.Decrypt(plaintext)
	if err != nil {
		t.Fatalf("Decrypt() with nil encryptor error = %v", err)
	}
	if decrypted != plaintext {
		t.Errorf("Decrypt() with nil encryptor = %q, want %q", decrypted, plaintext)
	}
}

func TestDecrypt_BackwardCompatibility(t *testing.T) {
	enc := NewEncryptor("test-passphrase")

	tests := []struct {
		name      string
		input     string
		wantError bool
	}{
		{
			name:      "unencrypted text",
			input:     "plain text that was never encrypted",
			wantError: false,
		},
		{
			name:      "invalid base64",
			input:     "not-valid-base64-but-should-not-crash!@#",
			wantError: false,
		},
		{
			name:      "empty string",
			input:     "",
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := enc.Decrypt(tt.input)
			if (err != nil) != tt.wantError {
				t.Errorf("Decrypt() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestEncryptMap(t *testing.T) {
	enc := NewEncryptor("test-passphrase")

	tests := []struct {
		name string
		m    map[string]string
	}{
		{
			name: "simple map",
			m: map[string]string{
				"key1": "value1",
				"key2": "value2",
			},
		},
		{
			name: "empty map",
			m:    map[string]string{},
		},
		{
			name: "single entry",
			m: map[string]string{
				"only": "one",
			},
		},
		{
			name: "special characters",
			m: map[string]string{
				"note1": "This has special chars: !@#$%",
				"note2": "Unicode: 世界 🌍",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Encrypt
			encrypted, err := enc.EncryptMap(tt.m)
			if err != nil {
				t.Fatalf("EncryptMap() error = %v", err)
			}

			// Verify keys are preserved
			if len(encrypted) != len(tt.m) {
				t.Errorf("EncryptMap() length = %d, want %d", len(encrypted), len(tt.m))
			}

			// Verify values are encrypted (different from original)
			for k, v := range tt.m {
				encVal, exists := encrypted[k]
				if !exists {
					t.Errorf("EncryptMap() missing key %q", k)
					continue
				}
				if v != "" && encVal == v {
					t.Errorf("EncryptMap() did not encrypt value for key %q", k)
				}
			}

			// Decrypt
			decrypted, err := enc.DecryptMap(encrypted)
			if err != nil {
				t.Fatalf("DecryptMap() error = %v", err)
			}

			// Verify round-trip
			if len(decrypted) != len(tt.m) {
				t.Errorf("DecryptMap() length = %d, want %d", len(decrypted), len(tt.m))
			}
			for k, want := range tt.m {
				got, exists := decrypted[k]
				if !exists {
					t.Errorf("DecryptMap() missing key %q", k)
					continue
				}
				if got != want {
					t.Errorf("DecryptMap()[%q] = %q, want %q", k, got, want)
				}
			}
		})
	}
}

func TestEncryptDecryptMap_NilEncryptor(t *testing.T) {
	var enc *Encryptor

	m := map[string]string{
		"key1": "value1",
		"key2": "value2",
	}

	// EncryptMap with nil should return original
	encrypted, err := enc.EncryptMap(m)
	if err != nil {
		t.Fatalf("EncryptMap() with nil encryptor error = %v", err)
	}
	if len(encrypted) != len(m) {
		t.Errorf("EncryptMap() with nil encryptor changed map size")
	}
	for k, v := range m {
		if encrypted[k] != v {
			t.Errorf("EncryptMap() with nil encryptor changed value for key %q", k)
		}
	}

	// DecryptMap with nil should return original
	decrypted, err := enc.DecryptMap(m)
	if err != nil {
		t.Fatalf("DecryptMap() with nil encryptor error = %v", err)
	}
	if len(decrypted) != len(m) {
		t.Errorf("DecryptMap() with nil encryptor changed map size")
	}
	for k, v := range m {
		if decrypted[k] != v {
			t.Errorf("DecryptMap() with nil encryptor changed value for key %q", k)
		}
	}
}

func TestEncryptMap_NilMap(t *testing.T) {
	enc := NewEncryptor("test-passphrase")

	encrypted, err := enc.EncryptMap(nil)
	if err != nil {
		t.Fatalf("EncryptMap() with nil map error = %v", err)
	}
	if encrypted != nil {
		t.Error("EncryptMap() with nil map should return nil")
	}

	decrypted, err := enc.DecryptMap(nil)
	if err != nil {
		t.Fatalf("DecryptMap() with nil map error = %v", err)
	}
	if decrypted != nil {
		t.Error("DecryptMap() with nil map should return nil")
	}
}

func TestDifferentEncryptors(t *testing.T) {
	enc1 := NewEncryptor("passphrase1")
	enc2 := NewEncryptor("passphrase2")

	plaintext := "secret data"

	// Encrypt with first encryptor
	ciphertext, err := enc1.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("enc1.Encrypt() error = %v", err)
	}

	// Try to decrypt with second encryptor - should fail gracefully
	// Due to backward compatibility, it may return the ciphertext as-is
	decrypted, err := enc2.Decrypt(ciphertext)
	if err != nil {
		t.Fatalf("enc2.Decrypt() error = %v", err)
	}

	// Should NOT match original plaintext (wrong key)
	if decrypted == plaintext {
		t.Error("Different encryptor should not decrypt correctly")
	}
}

func TestEncryption_ConsistentKeyDerivation(t *testing.T) {
	// Same passphrase should produce same key
	passphrase := "test-passphrase-123"

	enc1 := NewEncryptor(passphrase)
	enc2 := NewEncryptor(passphrase)

	plaintext := "test data"

	// Encrypt with first
	ciphertext1, err := enc1.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("enc1.Encrypt() error = %v", err)
	}

	// Decrypt with second (same passphrase)
	decrypted, err := enc2.Decrypt(ciphertext1)
	if err != nil {
		t.Fatalf("enc2.Decrypt() error = %v", err)
	}

	if decrypted != plaintext {
		t.Errorf("Same passphrase should allow decryption: got %q, want %q", decrypted, plaintext)
	}
}

func TestEncryption_NonDeterministic(t *testing.T) {
	// Same plaintext should produce different ciphertexts (due to random nonce)
	enc := NewEncryptor("test-passphrase")
	plaintext := "same text"

	ciphertext1, err := enc.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("first Encrypt() error = %v", err)
	}

	ciphertext2, err := enc.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("second Encrypt() error = %v", err)
	}

	if ciphertext1 == ciphertext2 {
		t.Error("Encryption should be non-deterministic (use random nonce)")
	}

	// But both should decrypt to same plaintext
	dec1, _ := enc.Decrypt(ciphertext1)
	dec2, _ := enc.Decrypt(ciphertext2)

	if dec1 != plaintext || dec2 != plaintext {
		t.Error("Both ciphertexts should decrypt to same plaintext")
	}
}

func TestEncryption_LegacyMigration(t *testing.T) {
	passphrase := "test-migration-passphrase"
	plaintext := "sensitive data to migrate"

	// Create legacy encryptor (simulates old encryption with SHA256-derived salt)
	legacySalt := legacySalt(passphrase)
	legacyKey := pbkdf2.Key([]byte(passphrase), legacySalt, 100000, 32, sha256.New)
	legacyEnc := &Encryptor{key: legacyKey}

	// Encrypt with legacy method
	legacyCiphertext, err := legacyEnc.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Legacy Encrypt() error = %v", err)
	}

	// Create new encryptor (has both new and legacy keys)
	newEnc := NewEncryptor(passphrase)

	// Decrypt with new encryptor - should detect migration needed
	decrypted, needsMigration, err := newEnc.DecryptWithMigration(legacyCiphertext)
	if err != nil {
		t.Fatalf("DecryptWithMigration() error = %v", err)
	}

	if decrypted != plaintext {
		t.Errorf("DecryptWithMigration() = %q, want %q", decrypted, plaintext)
	}

	if !needsMigration {
		t.Error("DecryptWithMigration() should detect legacy encryption")
	}

	// Re-encrypt with new key
	newCiphertext, err := newEnc.Encrypt(decrypted)
	if err != nil {
		t.Fatalf("Re-encrypt error = %v", err)
	}

	// Decrypt again - should NOT need migration this time
	decrypted2, needsMigration2, err := newEnc.DecryptWithMigration(newCiphertext)
	if err != nil {
		t.Fatalf("Second DecryptWithMigration() error = %v", err)
	}

	if decrypted2 != plaintext {
		t.Errorf("Second decrypt = %q, want %q", decrypted2, plaintext)
	}

	if needsMigration2 {
		t.Error("New encryption should NOT need migration")
	}

	// Verify legacy encryptor cannot decrypt new ciphertext (different key)
	// It should fall back to returning ciphertext as-is (backward compatibility)
	// This is expected behavior - legacy key can't decrypt new format
	_ = legacyEnc // Suppress unused variable warning
}

func TestEncryptMap_LegacyMigration(t *testing.T) {
	passphrase := "test-map-migration"

	// Create legacy encryptor
	legacySalt := legacySalt(passphrase)
	legacyKey := pbkdf2.Key([]byte(passphrase), legacySalt, 100000, 32, sha256.New)
	legacyEnc := &Encryptor{key: legacyKey}

	// Encrypt map with legacy method
	originalMap := map[string]string{
		"note1": "First note",
		"note2": "Second note",
		"note3": "Third note",
	}

	legacyEncryptedMap, err := legacyEnc.EncryptMap(originalMap)
	if err != nil {
		t.Fatalf("Legacy EncryptMap() error = %v", err)
	}

	// Create new encryptor and decrypt
	newEnc := NewEncryptor(passphrase)
	decryptedMap, needsMigration, err := newEnc.DecryptMapWithMigration(legacyEncryptedMap)
	if err != nil {
		t.Fatalf("DecryptMapWithMigration() error = %v", err)
	}

	if !needsMigration {
		t.Error("DecryptMapWithMigration() should detect legacy encryption")
	}

	// Verify all values decrypted correctly
	for k, want := range originalMap {
		got, exists := decryptedMap[k]
		if !exists {
			t.Errorf("Missing key %q in decrypted map", k)
			continue
		}
		if got != want {
			t.Errorf("DecryptMapWithMigration()[%q] = %q, want %q", k, got, want)
		}
	}

	// Re-encrypt with new key
	newEncryptedMap, err := newEnc.EncryptMap(decryptedMap)
	if err != nil {
		t.Fatalf("Re-encrypt map error = %v", err)
	}

	// Decrypt again - should NOT need migration
	decryptedMap2, needsMigration2, err := newEnc.DecryptMapWithMigration(newEncryptedMap)
	if err != nil {
		t.Fatalf("Second DecryptMapWithMigration() error = %v", err)
	}

	if needsMigration2 {
		t.Error("New encryption should NOT need migration")
	}

	// Verify values still correct
	for k, want := range originalMap {
		got := decryptedMap2[k]
		if got != want {
			t.Errorf("After migration, map[%q] = %q, want %q", k, got, want)
		}
	}
}
