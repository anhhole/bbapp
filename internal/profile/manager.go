package profile

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"bbapp/internal/api"
	"github.com/google/uuid"
)

// Manager handles profile CRUD operations with file-based storage
type Manager struct {
	storageDir string
	profiles   map[string]*Profile // key: profile ID
	mu         sync.RWMutex
}

// NewManager creates a new profile manager
func NewManager(storageDir string) *Manager {
	mgr := &Manager{
		storageDir: storageDir,
		profiles:   make(map[string]*Profile),
	}

	// Load existing profiles from disk
	_ = mgr.loadProfiles() // Ignore error on initialization (file may not exist)

	return mgr
}

// getStoragePath returns the full path to the profiles storage file
func (m *Manager) getStoragePath() string {
	return filepath.Join(m.storageDir, "profiles.json")
}

// loadProfiles loads all profiles from disk into memory
func (m *Manager) loadProfiles() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	storagePath := m.getStoragePath()

	// If file doesn't exist, start with empty profiles
	if _, err := os.Stat(storagePath); os.IsNotExist(err) {
		return nil
	}

	// Read file
	data, err := os.ReadFile(storagePath)
	if err != nil {
		return fmt.Errorf("read profiles file: %w", err)
	}

	// Parse JSON array
	var profiles []*Profile
	if err := json.Unmarshal(data, &profiles); err != nil {
		return fmt.Errorf("unmarshal profiles: %w", err)
	}

	// Build map from array
	m.profiles = make(map[string]*Profile)
	for _, p := range profiles {
		m.profiles[p.ID] = p
	}

	return nil
}

// saveProfiles saves all profiles to disk atomically
func (m *Manager) saveProfiles() error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Convert map to array
	profiles := make([]*Profile, 0, len(m.profiles))
	for _, p := range m.profiles {
		profiles = append(profiles, p)
	}

	// Marshal to JSON
	data, err := json.MarshalIndent(profiles, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal profiles: %w", err)
	}

	// Ensure directory exists
	if err := os.MkdirAll(m.storageDir, 0755); err != nil {
		return fmt.Errorf("create storage directory: %w", err)
	}

	storagePath := m.getStoragePath()

	// Write to temp file first (atomic write pattern)
	tempPath := storagePath + ".tmp"
	if err := os.WriteFile(tempPath, data, 0644); err != nil {
		return fmt.Errorf("write temp file: %w", err)
	}

	// Rename temp file to actual file (atomic operation)
	if err := os.Rename(tempPath, storagePath); err != nil {
		return fmt.Errorf("rename temp file: %w", err)
	}

	return nil
}

// CreateProfile creates a new profile with a unique name
func (m *Manager) CreateProfile(name, roomID string, config api.Config) (*Profile, error) {
	// Validate name
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("profile name cannot be empty")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Check for duplicate name
	for _, p := range m.profiles {
		if p.Name == name {
			return nil, fmt.Errorf("profile with name '%s' already exists", name)
		}
	}

	// Create new profile
	now := time.Now()
	profile := &Profile{
		ID:         uuid.New().String(),
		Name:       name,
		RoomID:     roomID,
		CreatedAt:  now,
		UpdatedAt:  now,
		LastUsedAt: nil,
		Config:     config,
	}

	// Add to map
	m.profiles[profile.ID] = profile

	// Save to disk
	if err := m.saveProfilesLocked(); err != nil {
		// Rollback: remove from map
		delete(m.profiles, profile.ID)
		return nil, fmt.Errorf("save profiles: %w", err)
	}

	return profile, nil
}

// saveProfilesLocked saves profiles without acquiring lock (caller must hold lock)
func (m *Manager) saveProfilesLocked() error {
	// Convert map to array
	profiles := make([]*Profile, 0, len(m.profiles))
	for _, p := range m.profiles {
		profiles = append(profiles, p)
	}

	// Marshal to JSON
	data, err := json.MarshalIndent(profiles, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal profiles: %w", err)
	}

	// Ensure directory exists
	if err := os.MkdirAll(m.storageDir, 0755); err != nil {
		return fmt.Errorf("create storage directory: %w", err)
	}

	storagePath := m.getStoragePath()

	// Write to temp file first (atomic write pattern)
	tempPath := storagePath + ".tmp"
	if err := os.WriteFile(tempPath, data, 0644); err != nil {
		return fmt.Errorf("write temp file: %w", err)
	}

	// Rename temp file to actual file (atomic operation)
	if err := os.Rename(tempPath, storagePath); err != nil {
		return fmt.Errorf("rename temp file: %w", err)
	}

	return nil
}

// LoadProfile loads a profile by ID and updates its LastUsedAt timestamp
func (m *Manager) LoadProfile(id string) (*Profile, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Find profile
	profile, exists := m.profiles[id]
	if !exists {
		return nil, fmt.Errorf("profile not found: %s", id)
	}

	// Update LastUsedAt timestamp
	now := time.Now()
	profile.LastUsedAt = &now

	// Save to disk
	if err := m.saveProfilesLocked(); err != nil {
		return nil, fmt.Errorf("save profiles: %w", err)
	}

	return profile, nil
}

// UpdateProfile updates a profile's config and updatedAt timestamp
func (m *Manager) UpdateProfile(id string, config api.Config) (*Profile, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Find profile
	profile, exists := m.profiles[id]
	if !exists {
		return nil, fmt.Errorf("profile not found: %s", id)
	}

	// Update config and timestamp
	profile.Config = config
	profile.UpdatedAt = time.Now()

	// Save to disk
	if err := m.saveProfilesLocked(); err != nil {
		return nil, fmt.Errorf("save profiles: %w", err)
	}

	return profile, nil
}
