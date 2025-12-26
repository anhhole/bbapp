# BB-Core Official API Integration - Design Document

**Date:** December 26, 2025
**Status:** Design Complete - Ready for Implementation
**Author:** AI-Assisted Design Session

## Overview

This document describes the complete redesign of BBapp to align with BB-Core's official API specification as documented in `docs/BBAPP_INTEGRATION_GUIDE.md`. The integration includes full authentication flow, configuration wizard with named profiles, early trial validation, and proper JWT token lifecycle management.

## Design Decisions Summary

| Decision Point | Choice | Rationale |
|----------------|--------|-----------|
| **Integration Approach** | Align with BB-Core's official API | Full compatibility, proper authentication, trial validation |
| **Authentication Flow** | Full registration + login | Complete user onboarding experience |
| **User Journey** | Configuration wizard (guided setup) | Educational for new users, prevents errors |
| **Configuration Management** | Named profiles (local storage) | Power user friendly, multiple scenarios |
| **Trial Validation** | Early validation in wizard | Prevent wasted setup time |
| **Token Storage** | Wails app storage + encryption | Simple, framework-native, secure |
| **Wizard Steps** | 4-step balanced wizard | Good separation of concerns, validation feedback |
| **Token Refresh** | Automatic background refresh | Uninterrupted sessions, best UX |
| **WebSocket Connection** | Connect at session start | Clean lifecycle tied to PK session |
| **Error Display** | Toast notifications (all errors) | Consistent error experience |

---

## Section 1: Architecture Overview

### High-Level Structure

BBapp implements BB-Core's official API contract with a wizard-driven interface. The architecture consists of four main layers:

### 1. Authentication Layer (`internal/auth/`)

- JWT token management with automatic refresh
- Secure credential storage using encrypted Wails app storage
- Background goroutine monitors token expiry (check every 5 minutes)
- Implements `/api/v1/auth/login`, `/api/v1/auth/register`, `/api/v1/auth/refresh-token`

### 2. API Client Layer (`internal/api/client.go` - **refactored**)

- Migrate from current custom endpoints to BB-Core standard endpoints
- **Change:** `GET /api/v1/stream/rooms/{roomId}/bbapp-config` → `GET /api/v1/external/config`
- **Add:** `POST /api/v1/external/validate-trial` for early validation
- **Add:** `POST /api/v1/external/heartbeat` for connection monitoring
- Maintain retry logic (exponential backoff) but add token refresh on 401

### 3. Profile Manager (`internal/profile/`)

- Local storage of named configuration profiles
- CRUD operations: Create, Load, Update, Delete profiles
- Profile structure contains: name, roomId, timestamp, streamers (cached from BB-Core)
- Storage: Encrypted JSON file in Wails app data directory

### 4. Session Manager (`internal/session/manager.go` - **enhanced**)

- Orchestrates PK session lifecycle with STOMP connection tied to session
- Integrates trial validation before starting browsers
- Coordinates: API calls → STOMP connect → Browser creation → Heartbeat start

---

## Section 2: Authentication System

### Token Management Service (`internal/auth/manager.go`)

The authentication manager handles the complete JWT lifecycle.

### Login/Register Flow

1. User provides credentials → API call to `/api/v1/auth/login` or `/api/v1/auth/register`
2. Response contains: `accessToken`, `refreshToken`, `expiresAt`, `user`, `agency`
3. Store tokens encrypted in Wails app storage (using AES-256)
4. Cache agency info for plan limit display (maxRooms, currentRooms, plan type)

### Automatic Token Refresh

- Background goroutine runs every 5 minutes checking token expiry
- If access token expires within 10 minutes → call `/api/v1/auth/refresh-token`
- Update stored tokens atomically (write new, then delete old)
- On refresh failure → emit event to UI prompting re-login
- Thread-safe with RWMutex protecting token state

### Secure Storage

- Use Wails runtime to get app data directory
- File: `{appDataDir}/bbapp/credentials.enc`
- Encryption key derived from: device hash (existing `internal/fingerprint`) + static salt
- JSON structure: `{"accessToken": "...", "refreshToken": "...", "expiresAt": 1234567890, "user": {...}, "agency": {...}}`
- Decrypt on app startup, keep in memory during runtime

### API Client Integration

- Inject current access token into all API requests via `Authorization: Bearer {token}`
- On 401 response → attempt refresh once → retry original request
- If retry fails → bubble error to UI for re-authentication

---

## Section 3: Configuration Wizard Flow

### 4-Step Wizard Implementation (`frontend/src/components/Wizard/`)

### Step 1: Profile Selection

- Display list of saved profiles (load from `internal/profile/manager.go`)
- Each profile shows: name, roomId, last used date, streamer count
- Options: "Load Profile" or "Create New Profile"
- If creating new → prompt for profile name (validation: 3-50 chars, unique)
- User selection determines next steps: load populates wizard, new starts fresh

### Step 2: Room Configuration

- Input field: Room ID (required, alphanumeric)
- On blur/continue → call `GET /api/v1/external/config` with room ID
- Display fetched data: agency name, plan type, limits
- Show plan limits prominently: "Plan: TRIAL | Max Rooms: 1 | Current: 0/1"
- Parse teams and streamers from config response
- **Error handling:** 404 = "Room not found", 403 = "Access denied", show via toast
- Cache fetched config in wizard state for Step 3

### Step 3: Streamer Configuration & Validation

- Display teams/streamers from Step 2 in editable table
- Columns: Team Name, Streamer Name, Bigo ID, Bigo Room ID, Binding Gift
- Allow add/remove streamers (respects plan limits)
- **Live validation:** On any change → call `POST /api/v1/external/validate-trial`
- Request body: `{"streamers": [{"bigoId": "...", "bigoRoomId": "..."}]}`
- Show validation status inline: ✓ green checkmark or ✗ red error with reason
- Block "Next" button if validation fails or exceeds plan limits

### Step 4: Review & Start

- Summary view: Profile name, Room ID, Team count, Streamer count, Plan usage
- Actions: "Save Profile" (persist to local storage), "Start Session" (or both)
- On "Start Session" → call session manager with validated configuration
- Show loading spinner during session initialization

---

## Section 4: Profile Management

### Profile Manager Service (`internal/profile/manager.go`)

### Profile Data Structure

```go
type Profile struct {
    ID          string    `json:"id"`          // UUID
    Name        string    `json:"name"`        // User-friendly name
    RoomID      string    `json:"roomId"`      // BB-Core room ID
    CreatedAt   time.Time `json:"createdAt"`
    UpdatedAt   time.Time `json:"updatedAt"`
    LastUsedAt  *time.Time `json:"lastUsedAt"`
    Config      Config    `json:"config"`      // Cached BB-Core config
}

type Config struct {
    Teams []Team `json:"teams"`
    // Matches api.Config structure from integration guide
}
```

### Storage Implementation

- File location: `{appDataDir}/bbapp/profiles.json`
- Structure: Array of Profile objects
- Load all profiles on app startup into memory
- Write to disk on: Create, Update, Delete operations
- Atomic writes: write to temp file → rename (prevents corruption)

### CRUD Operations

- `CreateProfile(name, roomId, config)` → validates uniqueness, generates UUID, saves
- `LoadProfile(id)` → returns profile by ID, updates lastUsedAt
- `UpdateProfile(id, config)` → overwrites config, updates updatedAt timestamp
- `DeleteProfile(id)` → removes from array, rewrites file
- `ListProfiles()` → returns all profiles sorted by lastUsedAt desc

### Profile-Config Relationship

- Profiles cache BB-Core config for offline viewing/quick loading
- Wizard Step 2 always fetches fresh config from BB-Core (source of truth)
- User can choose: "Use cached" (skip fetch) or "Refresh from BB-Core"
- On save profile → store current wizard state as cached config
- Validation always uses fresh data from BB-Core API

---

## Section 5: Session Lifecycle

### Enhanced Session Manager (`internal/session/manager.go` - **refactored**)

### Session Start Flow

```
1. User clicks "Start Session" from wizard Step 4
2. Session manager receives: roomId, config, deviceHash, authToken
3. Call POST /api/v1/external/validate-trial (final validation)
   - If rejected → return error, show toast, don't proceed
4. Call POST /pk/start-from-bbapp/{roomId} with deviceHash
   - Notifies BB-Core that BBapp session is starting
5. Establish STOMP connection to ws://{bbCoreUrl}/ws?token={accessToken}
   - Test connection, subscribe to /topic/room/{roomId}/pk
   - On connection failure → call stop-from-bbapp, return error
6. Create headless browsers for each streamer's bigoRoomId
   - Parallel creation with error collection
   - Track per-streamer connection status
7. Start heartbeat service (30s interval)
   - POST /api/v1/external/heartbeat with connection statuses
8. Session state → ACTIVE, emit status event to UI
```

### Running Session

- Gift/chat messages flow through existing listeners → STOMP publish to `/app/room/{roomId}/bigo`
- Include `deviceHash` in every message payload (as per integration guide)
- Heartbeat goroutine sends status updates with connection health
- STOMP auto-reconnection handled by existing `internal/stomp/client.go`
- Browser health monitoring (existing) recreates browsers if frames stop
- Session manager tracks: streamer statuses, message counts, last activity time

### Session Stop Flow

```
1. User clicks "Stop Session" or app shutdown initiated
2. Stop heartbeat service (cancel goroutine)
3. Stop all browser listeners and close browsers
4. Disconnect STOMP connection
5. Call POST /pk/stop-from-bbapp/{roomId} with reason
   - Notifies BB-Core that session ended gracefully
6. Session state → INACTIVE, emit status event to UI
7. Clean up in-memory session data
```

### Error Recovery

- STOMP disconnect during session → auto-reconnect (existing logic)
- Browser crash → recreate browser (existing health monitoring)
- API errors during heartbeat → log warning, continue session (don't crash)
- Manual stop always succeeds even if BB-Core API fails

---

## Section 6: Error Handling

### Standardized Error Response Handling (`internal/api/errors.go`)

### BB-Core Error Format

```go
type APIError struct {
    Timestamp  string            `json:"timestamp"`
    Status     string            `json:"status"`
    ErrorCode  int               `json:"errorCode"`
    Message    string            `json:"message"`
    Details    string            `json:"details,omitempty"`
    SubErrors  []ValidationError `json:"subErrors,omitempty"`
}

type ValidationError struct {
    Object        string      `json:"object"`
    Field         string      `json:"field"`
    RejectedValue interface{} `json:"rejectedValue"`
    Message       string      `json:"message"`
}
```

### Error Handling Strategy by Context

#### Wizard Errors (User-facing)

- Validation errors (400) → Toast with field-specific messages
- Entity not found (404) → Toast: "Room not found. Please check the Room ID."
- Unauthorized (401) → Trigger token refresh, retry once, then prompt re-login
- Trial validation rejected → Toast showing blocked Bigo IDs with reason
- Network errors → Toast: "Unable to connect to BB-Core. Check your connection."

#### Session Errors (Runtime)

- STOMP connection failure → Toast notification, retry with exponential backoff (5s, 10s, 20s)
- Browser creation failure → Log error, continue with other browsers, show warning toast
- Heartbeat API failure → Log warning, don't interrupt session (non-critical)
- Gift/chat publish failure → Log error, attempt republish once, then drop message

#### Authentication Errors

- Invalid credentials (2002) → Toast: "Invalid username or password"
- Token expired (2003) → Automatic refresh attempt → toast prompt login if refresh fails
- User not found (2001) → Toast: "Account not found. Please register."

### Error Recovery Actions

- **Retry with backoff:** Network errors, 5xx server errors
- **Token refresh:** 401 errors (once per request)
- **User notification:** Validation errors, trial rejection, critical failures
- **Silent logging:** Heartbeat failures, non-critical API errors
- **Graceful degradation:** Continue session if heartbeat fails, log for debugging

### UI Error Display

- **Toast notifications for all errors:** validation, network, session, authentication
- **Toast types:** Error (red), Warning (yellow), Success (green), Info (blue)
- **Auto-dismiss:** 5 seconds, user can dismiss manually
- **Stack multiple toasts** if errors occur rapidly
- **Critical errors:** Persistent toast requiring manual dismiss (trial rejection, session start failure)

---

## Section 7: Data Structures & Message Formats

### API Request/Response Models (`internal/api/types.go` - **refactored**)

### Authentication

```go
type LoginRequest struct {
    Username string `json:"username"`
    Password string `json:"password"`
}

type RegisterRequest struct {
    Username   string `json:"username"`
    Email      string `json:"email"`
    Password   string `json:"password"`
    FirstName  string `json:"firstName,omitempty"`
    LastName   string `json:"lastName,omitempty"`
    AgencyName string `json:"agencyName"`
}

type AuthResponse struct {
    AccessToken  string    `json:"accessToken"`
    RefreshToken string    `json:"refreshToken"`
    TokenType    string    `json:"tokenType"`
    ExpiresIn    int64     `json:"expiresIn"`
    ExpiresAt    time.Time `json:"expiresAt"`
    User         User      `json:"user"`
    Agency       Agency    `json:"agency"`
}

type Agency struct {
    ID          int64     `json:"id"`
    Name        string    `json:"name"`
    Plan        string    `json:"plan"` // TRIAL, PAID, PROFESSIONAL, ENTERPRISE
    Status      string    `json:"status"`
    MaxRooms    int       `json:"maxRooms"`
    CurrentRooms int      `json:"currentRooms"`
    ExpiresAt   time.Time `json:"expiresAt"`
}
```

### Configuration & Validation

```go
type ConfigResponse struct {
    RoomID   string   `json:"roomId"`
    AgencyID int64    `json:"agencyId"`
    Session  Session  `json:"session"`
    Teams    []Team   `json:"teams"`
}

type ValidateTrialRequest struct {
    Streamers []StreamerValidation `json:"streamers"`
}

type ValidateTrialResponse struct {
    Allowed        bool     `json:"allowed"`
    Message        string   `json:"message"`
    BlockedBigoIds []string `json:"blockedBigoIds"`
    Reason         string   `json:"reason,omitempty"`
}
```

### STOMP Message Payloads

Matching integration guide specification:

```go
// Gift message - all fields from integration guide
type GiftMessage struct {
    Type           string `json:"type"` // "GIFT"
    RoomID         string `json:"roomId"`
    BigoRoomID     string `json:"bigoRoomId"`
    SenderID       string `json:"senderId"`
    SenderName     string `json:"senderName"`
    SenderAvatar   string `json:"senderAvatar,omitempty"`
    SenderLevel    int    `json:"senderLevel,omitempty"`
    StreamerID     string `json:"streamerId"`
    StreamerName   string `json:"streamerName"`
    StreamerAvatar string `json:"streamerAvatar,omitempty"`
    GiftID         string `json:"giftId"`
    GiftName       string `json:"giftName"`
    GiftCount      int    `json:"giftCount"`
    Diamonds       int64  `json:"diamonds"`
    GiftImageURL   string `json:"giftImageUrl,omitempty"`
    Timestamp      int64  `json:"timestamp"`
    DeviceHash     string `json:"deviceHash"`
}

// Chat message
type ChatMessage struct {
    Type         string `json:"type"` // "CHAT"
    RoomID       string `json:"roomId"`
    BigoRoomID   string `json:"bigoRoomId"`
    SenderID     string `json:"senderId"`
    SenderName   string `json:"senderName"`
    SenderAvatar string `json:"senderAvatar,omitempty"`
    SenderLevel  int    `json:"senderLevel,omitempty"`
    Message      string `json:"message"`
    Timestamp    int64  `json:"timestamp"`
    DeviceHash   string `json:"deviceHash"`
}
```

---

## Section 8: Implementation Plan

### Phased Implementation Approach

Following TDD workflow from CLAUDE.md: write test → run (fail) → implement → run (pass) → commit

### Phase 1: Authentication Foundation (Days 1-2)

- Create `internal/auth/manager.go` with token storage and refresh logic
- Implement encryption/decryption for credential storage
- Add background token refresh goroutine
- Update `internal/api/client.go` to use new auth endpoints
- **Tests:** Token encryption, refresh logic, storage persistence
- **Wails bindings:** `Login()`, `Register()`, `RefreshToken()`, `GetAuthStatus()`

### Phase 2: API Client Refactor (Days 3-4)

- Migrate endpoints from `/api/v1/stream/rooms/*` to `/api/v1/external/*`
- Implement: `GetConfig()`, `ValidateTrial()`, `SendHeartbeat()`
- Add 401 handling with automatic token refresh
- Update error parsing to match BB-Core error format
- **Tests:** API calls, retry logic, error handling
- Keep backward compatibility during transition (feature flag)

### Phase 3: Profile Management (Day 5)

- Create `internal/profile/manager.go` with CRUD operations
- Implement encrypted JSON file storage
- Add profile validation and uniqueness checks
- **Tests:** Profile CRUD, file persistence, concurrent access
- **Wails bindings:** `CreateProfile()`, `LoadProfile()`, `ListProfiles()`, `DeleteProfile()`

### Phase 4: Wizard UI (Days 6-8)

- Build React wizard components (4 steps)
  - Step 1: Profile selection UI with list view
  - Step 2: Room ID input + agency info display
  - Step 3: Streamer table with live validation
  - Step 4: Review summary + start button
- Integrate with Wails bindings
- Toast notification system for all errors
- **Tests:** Component tests, wizard flow, validation display

### Phase 5: Session Manager Enhancement (Days 9-10)

- Refactor `internal/session/manager.go` for new flow
- Add trial validation before session start
- Integrate STOMP connection at session start (not at app start)
- Update heartbeat to use `/api/v1/external/heartbeat`
- Add `deviceHash` to all STOMP messages
- **Tests:** Session lifecycle, validation integration, error recovery
- **Wails bindings:** Update `StartPKSession()` and `StopPKSession()`

### Phase 6: Message Format Alignment (Day 11)

- Update gift/chat handlers to include all required fields
- Ensure `deviceHash` is included in every message
- Verify STOMP destinations match: `/app/room/{roomId}/bigo`
- Update `internal/listener/bigo.go` to populate new fields
- **Tests:** Message serialization, field validation

### Phase 7: Integration Testing (Days 12-13)

- End-to-end testing with real BB-Core instance
- Test full wizard → session → stop flow
- Verify trial validation rejection scenarios
- Test token refresh during long sessions
- Test STOMP reconnection and browser recreation
- Load testing: Multiple streamers, extended sessions

### Phase 8: Migration & Cleanup (Day 14)

- Remove old custom endpoints (feature flag off)
- Clean up deprecated code paths
- Update documentation (CLAUDE.md, README)
- Create migration guide for existing users
- Final testing and bug fixes

### Rollout Strategy

- Feature flag: `USE_OFFICIAL_API` (default: false initially)
- Gradual rollout: Internal testing → beta users → full release
- Maintain backward compatibility for 1 release cycle
- Deprecation warnings in logs for old endpoints

---

## Appendix: API Endpoint Mapping

### Current → New Endpoints

| Current Endpoint | New Endpoint | Notes |
|------------------|--------------|-------|
| N/A | `POST /api/v1/auth/login` | New authentication |
| N/A | `POST /api/v1/auth/register` | New user registration |
| N/A | `POST /api/v1/auth/refresh-token` | New token refresh |
| `GET /api/v1/stream/rooms/{roomId}/bbapp-config` | `GET /api/v1/external/config` | Standard config endpoint |
| `POST /api/v1/stream/rooms/{roomId}/bbapp-config` | `POST /api/v1/external/config` | Standard save endpoint |
| N/A | `POST /api/v1/external/validate-trial` | New trial validation |
| N/A | `POST /api/v1/external/heartbeat` | New heartbeat endpoint |
| `POST /pk/start-from-bbapp/{roomId}` | Same | Keep existing |
| `POST /pk/stop-from-bbapp/{roomId}` | Same | Keep existing |

### WebSocket/STOMP

- Connection: `ws://{bbCoreUrl}/ws?token={accessToken}`
- Send destination: `/app/room/{roomId}/bigo`
- Subscribe: `/topic/room/{roomId}/pk`

---

## Success Criteria

1. ✅ Full authentication flow (register, login, token refresh) working
2. ✅ Configuration wizard completes 4-step flow without errors
3. ✅ Trial validation prevents unauthorized Bigo IDs
4. ✅ Named profiles save/load/delete successfully
5. ✅ Sessions start with proper BB-Core API calls
6. ✅ STOMP messages include all required fields + deviceHash
7. ✅ Token auto-refresh works during long sessions
8. ✅ All errors display via toast notifications
9. ✅ Integration tests pass with real BB-Core instance
10. ✅ Documentation updated (CLAUDE.md, README)

---

## Next Steps

**Ready for implementation?**

1. Use `superpowers:using-git-worktrees` to create isolated workspace
2. Use `superpowers:writing-plans` to create detailed implementation plan
3. Follow TDD workflow: test → fail → implement → pass → commit
4. Track progress with `TodoWrite` tool

---

**End of Design Document**
