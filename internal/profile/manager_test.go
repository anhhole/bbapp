package profile

import (
	"os"
	"path/filepath"
	"testing"
	"bbapp/internal/api"
)

func TestNewManager(t *testing.T) {
	tmpDir := t.TempDir()

	mgr := NewManager(tmpDir)

	if mgr == nil {
		t.Fatal("NewManager() returned nil")
	}

	if mgr.storageDir != tmpDir {
		t.Errorf("storageDir = %s, want %s", mgr.storageDir, tmpDir)
	}

	if mgr.profiles == nil {
		t.Error("profiles map should be initialized")
	}
}

func TestManager_StorageFilePath(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)

	expectedPath := filepath.Join(tmpDir, "profiles.json")
	actualPath := mgr.getStoragePath()

	if actualPath != expectedPath {
		t.Errorf("getStoragePath() = %s, want %s", actualPath, expectedPath)
	}
}

func TestManager_LoadProfiles_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)

	// When file doesn't exist, should initialize empty profiles
	if err := mgr.loadProfiles(); err != nil {
		t.Fatalf("loadProfiles() should not error on missing file: %v", err)
	}

	if len(mgr.profiles) != 0 {
		t.Errorf("profiles should be empty, got %d profiles", len(mgr.profiles))
	}
}

func TestManager_SaveProfiles_CreatesFile(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)

	// Save empty profiles
	if err := mgr.saveProfiles(); err != nil {
		t.Fatalf("saveProfiles() error = %v", err)
	}

	// Verify file was created
	storagePath := mgr.getStoragePath()
	if _, err := os.Stat(storagePath); os.IsNotExist(err) {
		t.Error("profiles.json file should have been created")
	}
}

func TestManager_CreateProfile_Success(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)

	profile, err := mgr.CreateProfile("Test Profile", "room-123", testConfig())
	if err != nil {
		t.Fatalf("CreateProfile() error = %v", err)
	}

	// Verify ID is generated (UUID format)
	if profile.ID == "" {
		t.Error("Profile ID should not be empty")
	}

	// Verify name and roomID
	if profile.Name != "Test Profile" {
		t.Errorf("Name = %s, want Test Profile", profile.Name)
	}
	if profile.RoomID != "room-123" {
		t.Errorf("RoomID = %s, want room-123", profile.RoomID)
	}

	// Verify timestamps are set
	if profile.CreatedAt.IsZero() {
		t.Error("CreatedAt should be set")
	}
	if profile.UpdatedAt.IsZero() {
		t.Error("UpdatedAt should be set")
	}

	// Verify LastUsedAt is nil for new profile
	if profile.LastUsedAt != nil {
		t.Error("LastUsedAt should be nil for new profile")
	}

	// Verify profile was saved to file
	storagePath := mgr.getStoragePath()
	if _, err := os.Stat(storagePath); os.IsNotExist(err) {
		t.Error("profiles.json should have been created")
	}
}

func TestManager_CreateProfile_DuplicateName(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)

	// Create first profile
	_, err := mgr.CreateProfile("Duplicate Name", "room-123", testConfig())
	if err != nil {
		t.Fatalf("CreateProfile() first call error = %v", err)
	}

	// Try to create profile with same name
	_, err = mgr.CreateProfile("Duplicate Name", "room-456", testConfig())
	if err == nil {
		t.Error("CreateProfile() should return error for duplicate name")
	}
}

func TestManager_CreateProfile_EmptyName(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)

	// Try to create profile with empty name
	_, err := mgr.CreateProfile("", "room-123", testConfig())
	if err == nil {
		t.Error("CreateProfile() should return error for empty name")
	}
}

// Helper function for test config
func testConfig() api.Config {
	return api.Config{
		RoomId:   "room-123",
		AgencyId: 789,
	}
}
