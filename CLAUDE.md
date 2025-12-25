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
- Methods: GetConfig, StartSession, StopSession, SendHeartbeat
- 10s timeout, 3 retries with 1s, 2s, 4s delays
- Bearer token authentication

**internal/fingerprint/device.go**
- Generates deterministic device hash for trial validation
- SHA-256 hash based on hostname + OS + architecture
- Cached for performance (consistent across app lifetime)

**internal/config/manager.go**
- Fast streamer lookup with O(1) indexes
- Maps BigoRoomId → Streamer and StreamerId → Streamer
- GetAllBigoRoomIds() returns all configured rooms

**internal/session/manager.go**
- Session lifecycle orchestration
- Start(roomId) - Fetches config, calls BB-Core start-session API
- Stop(reason) - Gracefully stops session
- UpdateConnectionStatus() - Tracks per-streamer health
- Thread-safe with RWMutex

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
- Session-based workflow methods:
  - `StartPKSession(bbCoreUrl, authToken, roomId)` - Complete session init
  - `StopPKSession(reason)` - Graceful session teardown
  - `GetSessionStatus()` - Returns current session state
- Legacy methods (backward compatible):
  - `ConnectToCore()`, `AddStreamer()`, `RemoveStreamer()`

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

## Recent Enhancements

- ✅ REST API client for BB-Core
- ✅ Device fingerprinting
- ✅ Configuration manager
- ✅ Enhanced BigoGift struct (all fields)
- ✅ Chat message support
- ✅ Session manager
- ✅ Heartbeat service (30s)
- ✅ STOMP auto-reconnection
- ✅ Session-based UI workflow
- ✅ Connection health dashboard

## Known Features

- STOMP auto-reconnects with exponential backoff
- Browser health monitoring (recreates if no frames for 30s)
- Device fingerprinting for trial validation
- Complete gift/chat metadata forwarding
- Real-time connection status in UI
