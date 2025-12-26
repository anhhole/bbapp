package profile

import (
	"os"
	"path/filepath"
	"testing"
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
