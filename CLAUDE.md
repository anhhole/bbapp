# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

BBapp is a Wails v2 desktop application that intercepts Bigo Live WebSocket messages via hidden Chrome browsers and forwards gift/chat events to BB-Core via STOMP messaging.

**Architecture:** Go backend (chromedp for browser automation, STOMP client) + React TypeScript frontend (connection management UI)

## Tech Stack

- **Wails v2:** Go + React framework for desktop apps
- **chromedp:** Chrome DevTools Protocol for browser control
- **go-stomp/stomp:** STOMP messaging client
- **React + TypeScript:** UI layer

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
# Run all tests
go test ./...

# Run tests with verbose output
go test ./... -v

# Run specific package tests
go test ./internal/browser -v
go test ./internal/listener -v
go test ./internal/stomp -v
go test ./internal/logger -v

# Run single test
go test ./internal/browser -v -run TestManager_CreateBrowser

# Skip integration tests (requires external services)
go test ./... -short
```

## Architecture

### Core Components

**internal/browser/manager.go**
- Manages headless Chrome instances using chromedp
- Creates isolated browser contexts per Bigo room
- Each browser instance navigates to a Bigo Live room and intercepts WebSocket frames
- Browser lifecycle tied to monitoring session

**internal/listener/bigo.go**
- Listens to WebSocket frames from Bigo Live via Chrome DevTools Protocol
- Parses gift and chat events from intercepted WebSocket messages
- Uses chromedp's `ListenTarget` with `network.EventWebSocketFrameReceived`
- Invokes registered handlers when events are detected

**internal/listener/websocket.go**
- Generic WebSocket frame handler abstraction
- Supports multiple registered handlers per listener
- Thread-safe handler registration and invocation

**internal/stomp/client.go**
- STOMP client wrapper for BB-Core communication
- Connects to BB-Core message broker
- Publishes gift/chat events to `/app/room/{roomId}/bigo` destination
- JSON serialization of event payloads

**internal/logger/logger.go**
- File-based activity logging (JSONL format)
- Creates daily log files: `bbapp_YYYY-MM-DD.jsonl`
- Logs all gift events with timestamp, type, and metadata
- Stored in `./logs` directory

**app.go (Wails App)**
- Main application orchestration layer
- Manages lifecycle of browsers, listeners, STOMP client, and logger
- Exposes methods to React frontend via Wails bindings:
  - `ConnectToCore(url, username, password)` - Connect to BB-Core STOMP
  - `AddStreamer(bigoRoomId, teamId, roomId)` - Start monitoring a Bigo room
  - `RemoveStreamer(bigoRoomId)` - Stop monitoring a room
  - `GetConnections()` - List active monitoring sessions

### Data Flow

1. Hidden Chrome browser navigates to Bigo room URL (`https://www.bigo.tv/{roomId}`)
2. chromedp intercepts WebSocket frames via DevTools Protocol
3. BigoListener parses gift events from JSON payloads
4. Gift handler logs event to file AND forwards to BB-Core via STOMP
5. BB-Core receives message on `/app/room/{roomId}/bigo` destination

### Event Message Format

Gift events forwarded to BB-Core:
```json
{
  "type": "GIFT",
  "bigoId": "string",
  "nickname": "string",
  "giftName": "string",
  "giftValue": 123
}
```

Activity log entries:
```json
{
  "timestamp": 1234567890,
  "type": "GIFT",
  "bigoRoomId": "12345",
  "nickname": "username",
  "giftName": "Rose",
  "value": 100
}
```

## Testing Strategy

This project follows Test-Driven Development (TDD):

1. **Write failing test first** - Define expected behavior
2. **Run test to verify failure** - Confirm test is valid
3. **Implement minimal code** - Make test pass
4. **Run test to verify pass** - Confirm implementation works
5. **Commit** - Save working increment

Integration tests that require external services (STOMP server, live Bigo streams) should use `testing.Short()` check and skip when unavailable.

## Implementation Notes

- Each Bigo room gets its own isolated browser context
- Browsers run headless with `--no-sandbox --disable-gpu` flags
- STOMP connection is shared across all monitored rooms
- Logger creates new file daily, appends to existing file within same day
- Frontend communicates with Go backend via Wails IPC (auto-generated bindings in `frontend/wailsjs/`)
- Context cancellation cascades to clean up browser resources

## Development Workflow

When implementing new features:

1. Create package structure in `internal/`
2. Write test file (`*_test.go`) with expected behavior
3. Run test to verify failure
4. Implement minimal code to pass test
5. Verify with `go test ./...`
6. Build with `wails build` to ensure integration works
7. Test in dev mode with `wails dev`

## Known Constraints

- Requires Chrome/Chromium installed on system (chromedp dependency)
- STOMP connection requires BB-Core to be running and accessible
- Bigo Live WebSocket message format is reverse-engineered and may change
- No automatic reconnection on WebSocket disconnect (MVP limitation)
