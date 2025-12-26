# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

BBapp is a Wails v2 desktop application that intercepts Bigo Live WebSocket messages via headless Chrome browsers and forwards gift/chat events to BB-Core via STOMP messaging with session-based lifecycle management.

**Architecture:** Go backend (chromedp for browser automation, REST API client, session management, STOMP with auto-reconnection) + React TypeScript frontend (simplified session-based UI)

## Tech Stack

- **Wails v2:** Go + React framework for desktop apps
- **chromedp:** Chrome DevTools Protocol for browser control
- **go-stomp/stomp:** STOMP messaging client with auto-reconnection
- **React + TypeScript:** UI layer with real-time status polling
- **net/http:** REST API client for BB-Core integration

## Development Commands

### Initial Setup

```bash
# Install Wails CLI (first time only)
go install github.com/wailsapp/wails/v2/cmd/wails@latest

# Install dependencies
go get github.com/chromedp/chromedp@latest
go get github.com/chromedp/cdproto@latest
go get github.com/go-stomp/stomp/v3@latest
npm install --prefix frontend
```

### Development

```bash
# Run in development mode (hot reload)
wails dev

# Build production executable
wails build

# Clean build
wails build -clean
```

### Testing

```bash
# Run all tests (skip integration tests)
go test ./... -v -short

# Run specific package tests
go test ./internal/api -v
go test ./internal/session -v
go test ./internal/fingerprint -v
go test ./internal/config -v
go test ./internal/profile -v
go test ./internal/message -v

# Run integration tests (requires mock BB-Core)
go test ./internal/session -v
```

## Architecture (Post-Advanced Integration)

### Session-Based Workflow

1. User enters Room ID → BB-Core fetches config
2. Config provides list of streamers → Browsers created automatically
3. POST /pk/start-from-bbapp called with device hash
4. Gift/chat events forwarded with complete metadata via STOMP
5. Heartbeat sent every 30s with connection status
6. Auto-reconnection for STOMP and browsers
7. POST /pk/stop-from-bbapp called on session end

### Core Components

**internal/api/client.go**
- BB-Core REST API client with retry logic (exponential backoff)
- Authentication: Login, Register, RefreshToken (JWT tokens)
- Configuration: GetConfig, SaveConfig
- Session: StartSession, StopSession, SendHeartbeat
- Trial: ValidateTrial (anti-abuse checking)
- 10s timeout, 3 retries with 1s, 2s, 4s delays
- Auto token refresh on 401 responses
- Bearer token authentication

**internal/profile/manager.go**
- Local profile management with JSON file storage
- CRUD operations: CreateProfile, LoadProfile, UpdateProfile, DeleteProfile
- UUID-based profile IDs
- Atomic file writes (temp → rename pattern)
- LastUsedAt tracking for MRU sorting
- Thread-safe with RWMutex

**internal/message/types.go**
- Standardized message types for STOMP
- GiftMessage: All 17 fields per BB-Core spec
- ChatMessage: All 10 fields per BB-Core spec
- Comprehensive serialization tests (6 test cases)

**internal/fingerprint/device.go**
- Generates deterministic device hash for trial validation
- SHA-256 hash based on hostname + OS + architecture
- Cached for performance (consistent across app lifetime)

**internal/config/manager.go**
- Fast streamer lookup with O(1) indexes
- Maps BigoRoomId → Streamer and StreamerId → Streamer
- GetAllBigoRoomIds() returns all configured rooms

**internal/session/manager.go**
- Session lifecycle orchestration with 4-step flow:
  1. Validate trial (final check before session start)
  2. Start session at BB-Core (POST /pk/start-from-bbapp)
  3. Establish STOMP connection (moved from app startup)
  4. Start heartbeat service (30s interval)
- Stop(reason) - Graceful cleanup: heartbeat → STOMP → BB-Core notification
- UpdateConnectionStatus() - Tracks per-streamer connection health
- GetStatus() - Returns session state and connection list
- Thread-safe with RWMutex
- Rollback on STOMP failure (stops session at BB-Core)

**internal/session/heartbeat.go**
- 30s interval heartbeat to BB-Core
- Sends connection status for all streamers
- Goroutine-based with graceful shutdown

**internal/listener/bigo.go**
- Enhanced protocol parsing (gifts + chat + all fields)
- BigoGift: sender info, receiver info, gift details, metadata
- BigoChat: sender info, message, timestamp
- Debug mode with frame capture to file
- Health monitoring (IsHealthy checks frame reception)

**internal/stomp/client.go**
- Auto-reconnecting STOMP client
- Health monitoring every 10s
- Reconnection with exponential backoff (max 5 attempts)
- Thread-safe with RWMutex
- Supports both TCP and WebSocket transport

**app.go (Wails App)**
- Authentication bindings:
  - `Login(username, password)` - JWT authentication
  - `Register(...)` - User registration
  - `RefreshAuthToken(refreshToken)` - Token refresh
- Profile management bindings:
  - `CreateProfile(name, roomID, config)` - Create new profile
  - `LoadProfile(id)` - Load profile by ID
  - `UpdateProfile(id, config)` - Update profile config
  - `DeleteProfile(id)` - Delete profile
  - `ListProfiles()` - Get all profiles (sorted by lastUsedAt)
- API bindings:
  - `FetchConfig(roomID)` - Get BB-Core config
  - `ValidateTrial(streamers)` - Validate trial eligibility
- Session-based workflow methods:
  - `StartPKSession(bbCoreUrl, authToken, roomId, config)` - Complete session init
  - `StopPKSession(reason)` - Graceful session teardown
  - `GetSessionStatus()` - Returns current session state
- Legacy methods (backward compatible):
  - `ConnectToCore()`, `AddStreamer()`, `RemoveStreamer()`

### Frontend Components (Wizard UI)

**frontend/src/components/wizard/WizardContainer.tsx**
- Main wizard orchestration component
- 4-step state machine with progress tracking
- Step validation before navigation
- Toast notification management

**frontend/src/components/wizard/ProfileSelectionStep.tsx**
- Load existing profiles or create new
- Profile list with last used sorting
- Form validation for new profiles

**frontend/src/components/wizard/RoomConfigStep.tsx**
- Room ID input with BB-Core config fetch
- Configuration preview display
- Error handling with persistent toasts

**frontend/src/components/wizard/StreamerConfigStep.tsx**
- Streamer table display grouped by teams
- Live trial validation with ValidateTrial API
- Blocks progression if validation fails
- Shows blocked Bigo IDs with reason

**frontend/src/components/wizard/ReviewStep.tsx**
- Configuration summary (teams, streamers, counts)
- Save profile and/or start session options
- Calls StartPKSession with complete config

**frontend/src/components/wizard/ToastNotification.tsx**
- Auto-dismiss notifications (5s default)
- Persistent toast support for critical errors
- Manual dismiss capability

### Data Flow

1. User clicks "Start Session" with Room ID
2. BBapp calls BB-Core GET /bbapp-config/{roomId}
3. BBapp calls BB-Core POST /pk/start-from-bbapp/{roomId} with device hash
4. For each streamer in config:
   - Create headless Chrome browser
   - Navigate to https://www.bigo.tv/{bigoRoomId}
   - Intercept WebSocket frames via chromedp
5. Parse gift/chat events from frames
6. Forward to BB-Core via STOMP with complete payload
7. Send heartbeat every 30s with connection status
8. Auto-reconnect STOMP/browsers if disconnected
9. User clicks "Stop Session" → cleanup all resources

### Enhanced Message Formats

**Gift Event (STOMP):**
```json
{
  "type": "GIFT",
  "roomId": "abc123",
  "bigoRoomId": "room123",
  "senderId": "sender1",
  "senderName": "Bob",
  "senderAvatar": "https://...",
  "senderLevel": 25,
  "streamerId": "s1",
  "streamerName": "Alice",
  "streamerAvatar": "https://...",
  "giftId": "g1",
  "giftName": "Rose",
  "giftCount": 1,
  "diamonds": 100,
  "giftImageUrl": "https://...",
  "timestamp": 1234567890,
  "deviceHash": "abc123..."
}
```

**Chat Message (STOMP):**
```json
{
  "type": "CHAT",
  "roomId": "abc123",
  "bigoRoomId": "room123",
  "senderId": "sender1",
  "senderName": "Bob",
  "senderAvatar": "https://...",
  "senderLevel": 25,
  "message": "Hello!",
  "timestamp": 1234567890,
  "deviceHash": "abc123..."
}
```

## Testing Strategy

This project follows Test-Driven Development (TDD):

1. **Write failing test first** - Define expected behavior
2. **Run test to verify failure** - Confirm test is valid
3. **Implement minimal code** - Make test pass
4. **Run test to verify pass** - Confirm implementation works
5. **Commit** - Save working increment

Integration tests require BB-Core running. Use `-short` flag to skip them during development.

## Implementation Notes

- **Session-based:** Single StartPKSession creates all browsers from config
- **Device fingerprinting:** Stable hash for trial validation
- **Auto-reconnection:** STOMP monitors health every 10s, reconnects automatically
- **Heartbeat:** Every 30s to BB-Core with connection status
- **Enhanced parsing:** Full gift/chat metadata captured
- **Thread-safe:** All managers use RWMutex for concurrent access
- **Frontend bindings:** Auto-generated by Wails in `frontend/wailsjs/`
- **JSON tags:** Required on Go structs for proper TypeScript generation

## Development Workflow

When implementing new features:

1. Check if plan exists in `docs/plans/`
2. Follow TDD: write test → run (fail) → implement → run (pass) → commit
3. For REST endpoints, add to `internal/api/client.go`
4. For session logic, add to `internal/session/manager.go`
5. Build with `wails build` to ensure frontend bindings generate
6. Test in dev mode with `wails dev`

## Recent Enhancements (BB-Core Official API Integration)

**Phase 1: Authentication Foundation**
- ✅ JWT authentication (Login, Register, RefreshToken)
- ✅ Auto token refresh on 401 responses
- ✅ 36 comprehensive API tests (100% passing)

**Phase 2: API Client Refactor**
- ✅ Migrated to official BB-Core endpoints (/api/v1/external/*)
- ✅ GetConfig, SaveConfig, ValidateTrial endpoints
- ✅ Enhanced error handling (APIError, ValidationError types)
- ✅ Retry logic with exponential backoff

**Phase 3: Profile Management**
- ✅ Local profile storage (JSON files with atomic writes)
- ✅ CRUD operations with UUID-based IDs
- ✅ 19 profile manager tests (100% passing)

**Phase 4: Wizard UI**
- ✅ 4-step configuration wizard (Profile → Config → Streamers → Review)
- ✅ React components with real-time validation
- ✅ Toast notification system
- ✅ Wails bindings for all backend functionality

**Phase 5: Session Manager Enhancement**
- ✅ 4-step session start (trial validation → start → STOMP → heartbeat)
- ✅ STOMP lifecycle tied to session (not app startup)
- ✅ Graceful cleanup with rollback on failures
- ✅ DeviceHash in all STOMP messages

**Phase 6: Message Format Alignment**
- ✅ Standardized GiftMessage and ChatMessage types
- ✅ 6 serialization tests (100% passing)
- ✅ Full BB-Core spec compliance

**Phase 7: Integration Testing**
- ✅ Mock BB-Core server framework
- ✅ 9 comprehensive integration test suites
- ✅ 19 integration test scenarios (100% passing)
- ✅ Session lifecycle, error recovery, multi-streamer testing

## Known Features

- STOMP auto-reconnects with exponential backoff
- Browser health monitoring (recreates if no frames for 30s)
- Device fingerprinting for trial validation
- Complete gift/chat metadata forwarding
- Real-time connection status in UI
