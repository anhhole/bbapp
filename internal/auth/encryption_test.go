package auth

import (
	"testing"
)

func TestEncryptDecrypt(t *testing.T) {
	key := []byte("test-key-32-bytes-long-padding!!")
	plaintext := []byte(`{"accessToken":"test","refreshToken":"refresh"}`)

	// Test encryption
	encrypted, err := encrypt(plaintext, key)
	if err != nil {
		t.Fatalf("encrypt() error = %v", err)
	}

	if len(encrypted) == 0 {
		t.Error("encrypted data is empty")
	}

	// Test decryption
	decrypted, err := decrypt(encrypted, key)
	if err != nil {
		t.Fatalf("decrypt() error = %v", err)
	}

	if string(decrypted) != string(plaintext) {
		t.Errorf("decrypted = %s, want %s", decrypted, plaintext)
	}
}

func TestDecryptInvalidData(t *testing.T) {
	key := []byte("test-key-32-bytes-long-padding!!")
	invalid := []byte("invalid-encrypted-data")

	_, err := decrypt(invalid, key)
	if err == nil {
		t.Error("decrypt() should fail with invalid data")
	}
}
