package fingerprint_test

import (
	"testing"
	"bbapp/internal/fingerprint"
)

func TestGenerateDeviceHash(t *testing.T) {
	hash1, err := fingerprint.GenerateDeviceHash()
	if err != nil {
		t.Fatalf("GenerateDeviceHash failed: %v", err)
	}

	if len(hash1) == 0 {
		t.Fatal("Expected non-empty device hash")
	}

	// Should be deterministic
	hash2, err := fingerprint.GenerateDeviceHash()
	if err != nil {
		t.Fatalf("GenerateDeviceHash failed: %v", err)
	}

	if hash1 != hash2 {
		t.Errorf("Expected same hash, got %s and %s", hash1, hash2)
	}
}
