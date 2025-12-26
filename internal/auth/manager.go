package auth

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// Manager handles authentication and credential storage
type Manager struct {
	deviceHash  string
	storageDir  string
	credentials *Credentials
	mu          sync.RWMutex
}

// NewManager creates a new auth manager
func NewManager(deviceHash string) *Manager {
	return &Manager{
		deviceHash: deviceHash,
		storageDir: "", // Will be set by SetStorageDir
	}
}

// SetStorageDir sets the storage directory for credentials
func (m *Manager) SetStorageDir(dir string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("create storage dir: %w", err)
	}

	m.storageDir = dir
	return nil
}

// SaveCredentials encrypts and saves credentials to disk
func (m *Manager) SaveCredentials(creds *Credentials) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Marshal to JSON
	jsonData, err := json.Marshal(creds)
	if err != nil {
		return fmt.Errorf("marshal credentials: %w", err)
	}

	// Encrypt
	key := deriveKey(m.deviceHash)
	encrypted, err := encrypt(jsonData, key)
	if err != nil {
		return fmt.Errorf("encrypt: %w", err)
	}

	// Write to file (atomic)
	credPath := filepath.Join(m.storageDir, "credentials.enc")
	tempPath := credPath + ".tmp"

	if err := os.WriteFile(tempPath, encrypted, 0600); err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	if err := os.Rename(tempPath, credPath); err != nil {
		return fmt.Errorf("rename file: %w", err)
	}

	// Cache in memory
	m.credentials = creds
	return nil
}

// LoadCredentials loads and decrypts credentials from disk
func (m *Manager) LoadCredentials() (*Credentials, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	credPath := filepath.Join(m.storageDir, "credentials.enc")

	// Read encrypted file
	encrypted, err := os.ReadFile(credPath)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	// Decrypt
	key := deriveKey(m.deviceHash)
	decrypted, err := decrypt(encrypted, key)
	if err != nil {
		return nil, fmt.Errorf("decrypt: %w", err)
	}

	// Unmarshal
	var creds Credentials
	if err := json.Unmarshal(decrypted, &creds); err != nil {
		return nil, fmt.Errorf("unmarshal: %w", err)
	}

	return &creds, nil
}

// GetCredentials returns cached credentials (if any)
func (m *Manager) GetCredentials() *Credentials {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.credentials
}

// ClearCredentials removes stored credentials
func (m *Manager) ClearCredentials() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	credPath := filepath.Join(m.storageDir, "credentials.enc")
	if err := os.Remove(credPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove file: %w", err)
	}

	m.credentials = nil
	return nil
}
