# BB-Core Official API Integration - Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Integrate BBapp with BB-Core's official API specification including authentication, profile management, wizard UI, and proper JWT token lifecycle

**Architecture:** Add authentication layer (internal/auth/), refactor API client for official endpoints, implement profile manager with encrypted storage, enhance session manager with trial validation

**Tech Stack:** Go (crypto/aes, encoding/json), Wails v2, React TypeScript, BB-Core REST API, JWT tokens

---

## Phase 1: Authentication Foundation

### Task 1: Create Authentication Types

**Files:**
- Create: `internal/auth/types.go`
- Test: `internal/auth/types_test.go`

**Step 1: Write types test**

```go
package auth

import (
	"testing"
	"time"
)

func TestCredentials_IsExpired(t *testing.T) {
	tests := []struct {
		name      string
		expiresAt time.Time
		want      bool
	}{
		{
			name:      "not expired",
			expiresAt: time.Now().Add(1 * time.Hour),
			want:      false,
		},
		{
			name:      "expired",
			expiresAt: time.Now().Add(-1 * time.Hour),
			want:      true,
		},
		{
			name:      "expires in 1 minute (not expired)",
			expiresAt: time.Now().Add(1 * time.Minute),
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Credentials{
				AccessToken:  "token",
				RefreshToken: "refresh",
				ExpiresAt:    tt.expiresAt,
			}
			if got := c.IsExpired(); got != tt.want {
				t.Errorf("IsExpired() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCredentials_NeedsRefresh(t *testing.T) {
	tests := []struct {
		name      string
		expiresAt time.Time
		want      bool
	}{
		{
			name:      "needs refresh in 5 minutes",
			expiresAt: time.Now().Add(5 * time.Minute),
			want:      true,
		},
		{
			name:      "needs refresh in 9 minutes",
			expiresAt: time.Now().Add(9 * time.Minute),
			want:      true,
		},
		{
			name:      "doesnt need refresh in 15 minutes",
			expiresAt: time.Now().Add(15 * time.Minute),
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Credentials{
				AccessToken:  "token",
				RefreshToken: "refresh",
				ExpiresAt:    tt.expiresAt,
			}
			if got := c.NeedsRefresh(); got != tt.want {
				t.Errorf("NeedsRefresh() = %v, want %v", got, tt.want)
			}
		})
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/auth -v`
Expected: FAIL with "no Go files in internal/auth"

**Step 3: Implement minimal types**

Create `internal/auth/types.go`:

```go
package auth

import (
	"time"
)

// Credentials represents stored authentication credentials
type Credentials struct {
	AccessToken  string    `json:"accessToken"`
	RefreshToken string    `json:"refreshToken"`
	ExpiresAt    time.Time `json:"expiresAt"`
	User         User      `json:"user"`
	Agency       Agency    `json:"agency"`
}

// User represents authenticated user info
type User struct {
	ID        int64  `json:"id"`
	Username  string `json:"username"`
	Email     string `json:"email"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	RoleCode  string `json:"roleCode"`
}

// Agency represents user's agency info
type Agency struct {
	ID           int64     `json:"id"`
	Name         string    `json:"name"`
	Plan         string    `json:"plan"` // TRIAL, PAID, PROFESSIONAL, ENTERPRISE
	Status       string    `json:"status"`
	MaxRooms     int       `json:"maxRooms"`
	CurrentRooms int       `json:"currentRooms"`
	ExpiresAt    time.Time `json:"expiresAt"`
}

// IsExpired checks if access token is expired
func (c *Credentials) IsExpired() bool {
	return time.Now().After(c.ExpiresAt)
}

// NeedsRefresh checks if token expires within 10 minutes
func (c *Credentials) NeedsRefresh() bool {
	return time.Until(c.ExpiresAt) < 10*time.Minute
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/auth -v`
Expected: PASS

**Step 5: Commit**

```bash
cd .worktrees/bbcore-api-integration
git add internal/auth/types.go internal/auth/types_test.go
git commit -m "feat(auth): add authentication types with expiry logic

- Add Credentials struct with JWT tokens and user/agency info
- Add IsExpired() and NeedsRefresh() helper methods
- Add User and Agency structs matching BB-Core API response
- Test token expiry and refresh logic

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

### Task 2: Create Encryption Utilities

**Files:**
- Create: `internal/auth/encryption.go`
- Test: `internal/auth/encryption_test.go`

**Step 1: Write encryption test**

```go
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
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/auth -v -run TestEncrypt`
Expected: FAIL with "undefined: encrypt"

**Step 3: Implement encryption functions**

Create `internal/auth/encryption.go`:

```go
package auth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"io"
)

// deriveKey creates a 32-byte key from input (device hash + salt)
func deriveKey(input string) []byte {
	hash := sha256.Sum256([]byte(input + "bbapp-salt-v1"))
	return hash[:]
}

// encrypt encrypts plaintext using AES-256-GCM
func encrypt(plaintext, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// decrypt decrypts ciphertext using AES-256-GCM
func decrypt(ciphertext, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypt: %w", err)
	}

	return plaintext, nil
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/auth -v -run TestEncrypt`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/auth/encryption.go internal/auth/encryption_test.go
git commit -m "feat(auth): add AES-256-GCM encryption utilities

- Implement encrypt/decrypt functions using AES-256-GCM
- Add deriveKey function using SHA-256 with salt
- Test encryption round-trip and invalid data handling

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

### Task 3: Create Auth Manager Storage

**Files:**
- Create: `internal/auth/manager.go`
- Test: `internal/auth/manager_test.go`

**Step 1: Write storage test**

```go
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
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/auth -v -run TestManager`
Expected: FAIL with "undefined: Manager"

**Step 3: Implement Manager storage methods**

Update `internal/auth/manager.go`:

```go
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
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/auth -v -run TestManager`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/auth/manager.go internal/auth/manager_test.go
git commit -m "feat(auth): implement credential storage with encryption

- Add Manager with SaveCredentials and LoadCredentials methods
- Implement atomic file writes (write to .tmp, then rename)
- Add in-memory caching of credentials
- Add ClearCredentials for logout
- Test save/load round-trip with encryption

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

### Task 4: Add API Auth Types

**Files:**
- Modify: `internal/api/types.go`
- Test: `internal/api/types_test.go`

**Step 1: Write auth types test**

Create `internal/api/types_test.go` (if doesn't exist):

```go
package api

import (
	"encoding/json"
	"testing"
	"time"
)

func TestAuthResponse_UnmarshalJSON(t *testing.T) {
	jsonData := `{
		"accessToken": "eyJhbGc...",
		"refreshToken": "eyJhbGc...",
		"tokenType": "Bearer",
		"expiresIn": 86400000,
		"expiresAt": "2025-12-27T12:00:00.000Z",
		"user": {
			"id": 123,
			"username": "testuser",
			"email": "test@example.com",
			"roleCode": "OWNER"
		},
		"agency": {
			"id": 456,
			"name": "Test Agency",
			"plan": "TRIAL",
			"status": "ACTIVE",
			"maxRooms": 1,
			"currentRooms": 0,
			"expiresAt": "2025-12-31T23:59:59.000Z"
		}
	}`

	var resp AuthResponse
	if err := json.Unmarshal([]byte(jsonData), &resp); err != nil {
		t.Fatalf("Unmarshal error = %v", err)
	}

	if resp.AccessToken != "eyJhbGc..." {
		t.Errorf("AccessToken = %s, want eyJhbGc...", resp.AccessToken)
	}
	if resp.User.Username != "testuser" {
		t.Errorf("Username = %s, want testuser", resp.User.Username)
	}
	if resp.Agency.Plan != "TRIAL" {
		t.Errorf("Plan = %s, want TRIAL", resp.Agency.Plan)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/api -v -run TestAuthResponse`
Expected: FAIL with "undefined: AuthResponse"

**Step 3: Add auth types to types.go**

Append to `internal/api/types.go`:

```go
// Authentication Types

// LoginRequest is the request body for login
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// RegisterRequest is the request body for registration
type RegisterRequest struct {
	Username   string `json:"username"`
	Email      string `json:"email"`
	Password   string `json:"password"`
	FirstName  string `json:"firstName,omitempty"`
	LastName   string `json:"lastName,omitempty"`
	AgencyName string `json:"agencyName"`
}

// RefreshTokenRequest is the request body for token refresh
type RefreshTokenRequest struct {
	RefreshToken string `json:"refreshToken"`
}

// AuthResponse is the response from login/register/refresh
type AuthResponse struct {
	AccessToken  string    `json:"accessToken"`
	RefreshToken string    `json:"refreshToken"`
	TokenType    string    `json:"tokenType"`
	ExpiresIn    int64     `json:"expiresIn"`
	ExpiresAt    time.Time `json:"expiresAt"`
	User         User      `json:"user"`
	Agency       Agency    `json:"agency"`
}

// User represents authenticated user info
type User struct {
	ID        int64  `json:"id"`
	Username  string `json:"username"`
	Email     string `json:"email"`
	FirstName string `json:"firstName,omitempty"`
	LastName  string `json:"lastName,omitempty"`
	RoleCode  string `json:"roleCode"`
}

// Agency represents user's agency info
type Agency struct {
	ID           int64     `json:"id"`
	Name         string    `json:"name"`
	Plan         string    `json:"plan"` // TRIAL, PAID, PROFESSIONAL, ENTERPRISE
	Status       string    `json:"status"`
	MaxRooms     int       `json:"maxRooms"`
	CurrentRooms int       `json:"currentRooms"`
	ExpiresAt    time.Time `json:"expiresAt"`
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/api -v -run TestAuthResponse`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/api/types.go internal/api/types_test.go
git commit -m "feat(api): add authentication request/response types

- Add LoginRequest, RegisterRequest, RefreshTokenRequest
- Add AuthResponse with User and Agency nested types
- Test JSON unmarshaling of auth response
- Types match BB-Core API specification

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Phase 2: API Client Refactor (Summary)

**Tasks:**
1. Add error handling types (APIError, ValidationError)
2. Implement Login endpoint
3. Implement Register endpoint
4. Implement RefreshToken endpoint
5. Add 401 retry logic with token refresh
6. Migrate GetConfig to /api/v1/external/config
7. Add ValidateTrial endpoint
8. Add SendHeartbeat endpoint

**Each task follows same TDD pattern**: test → fail → implement → pass → commit

---

## Phase 3: Profile Management (Summary)

**Tasks:**
1. Create Profile types (Profile, Team, Streamer)
2. Implement ProfileManager with CRUD operations
3. Add encrypted JSON file storage
4. Implement CreateProfile with uniqueness validation
5. Implement LoadProfile with lastUsedAt update
6. Implement UpdateProfile
7. Implement DeleteProfile
8. Implement ListProfiles sorted by lastUsedAt

**Each task follows TDD pattern with comprehensive tests**

---

## Phase 4: Wizard UI (Summary)

**Components to build:**
1. WizardContainer (4-step state machine)
2. ProfileSelectionStep (list saved profiles)
3. RoomConfigStep (fetch config, show agency info)
4. StreamerConfigStep (editable table, live validation)
5. ReviewStep (summary, save profile, start session)
6. ToastNotification system

**Testing:** React component tests, wizard flow integration tests

---

## Phase 5: Session Manager Enhancement (Summary)

**Tasks:**
1. Add trial validation call before session start
2. Move STOMP connection to session start (not app start)
3. Update heartbeat to use /api/v1/external/heartbeat
4. Add deviceHash to all STOMP messages
5. Enhance error handling with toast notifications
6. Update StartPKSession and StopPKSession methods

---

## Phase 6: Message Format Alignment (Summary)

**Tasks:**
1. Update gift handler to include deviceHash
2. Update chat handler to include deviceHash
3. Verify STOMP destinations match specification
4. Test message serialization matches BB-Core format

---

## Phase 7: Integration Testing

**Test Scenarios:**
1. Complete wizard flow end-to-end
2. Token refresh during long session
3. Trial validation rejection
4. STOMP reconnection
5. Browser recreation after failure
6. Multiple streamers simultaneously

---

## Phase 8: Migration & Cleanup

**Tasks:**
1. Remove old custom endpoints
2. Update CLAUDE.md documentation
3. Update README with new authentication flow
4. Create user migration guide
5. Final testing and bug fixes

---

## Success Criteria

- ✅ All unit tests passing (go test ./... -short)
- ✅ Integration tests passing with real BB-Core
- ✅ Authentication flow works (login, register, token refresh)
- ✅ Profile management CRUD works
- ✅ Wizard completes 4-step flow
- ✅ Trial validation prevents unauthorized IDs
- ✅ Sessions start/stop cleanly with proper API calls
- ✅ STOMP messages include all required fields
- ✅ Documentation updated

---

## Execution Notes

**TDD Discipline:**
- Write test FIRST (red)
- Implement MINIMAL code to pass (green)
- Commit IMMEDIATELY (save)
- Repeat for each small increment

**File Organization:**
- Tests in `*_test.go` files
- Keep functions small and focused
- Use helper functions for common test setup

**Commit Messages:**
- Format: `feat(component): description`
- Include test coverage info
- Reference design doc when relevant

---

**End of Implementation Plan**
