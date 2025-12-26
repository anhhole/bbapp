package auth

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestManager_SaveAndLoadCredentials(t *testing.T) {
	// Create temp directory for test
	tmpDir := t.TempDir()

	mgr := NewManager("test-device-hash")
	mgr.storageDir = tmpDir

	// Create test credentials
	creds := &Credentials{
		AccessToken:  "test-access-token",
		RefreshToken: "test-refresh-token",
		ExpiresAt:    time.Now().Add(24 * time.Hour),
		User: User{
			ID:       123,
			Username: "testuser",
			Email:    "test@example.com",
		},
		Agency: Agency{
			ID:       456,
			Name:     "Test Agency",
			Plan:     "TRIAL",
			MaxRooms: 1,
		},
	}

	// Test save
	if err := mgr.SaveCredentials(creds); err != nil {
		t.Fatalf("SaveCredentials() error = %v", err)
	}

	// Verify file exists
	credPath := filepath.Join(tmpDir, "credentials.enc")
	if _, err := os.Stat(credPath); os.IsNotExist(err) {
		t.Error("credentials file was not created")
	}

	// Test load
	loaded, err := mgr.LoadCredentials()
	if err != nil {
		t.Fatalf("LoadCredentials() error = %v", err)
	}

	// Verify loaded data
	if loaded.AccessToken != creds.AccessToken {
		t.Errorf("AccessToken = %s, want %s", loaded.AccessToken, creds.AccessToken)
	}
	if loaded.User.Username != creds.User.Username {
		t.Errorf("Username = %s, want %s", loaded.User.Username, creds.User.Username)
	}
	if loaded.Agency.Plan != creds.Agency.Plan {
		t.Errorf("Plan = %s, want %s", loaded.Agency.Plan, creds.Agency.Plan)
	}
}

func TestManager_LoadCredentials_NotFound(t *testing.T) {
	tmpDir := t.TempDir()

	mgr := NewManager("test-device-hash")
	mgr.storageDir = tmpDir

	_, err := mgr.LoadCredentials()
	if err == nil {
		t.Error("LoadCredentials() should return error when file doesn't exist")
	}
}
