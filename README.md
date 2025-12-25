# BBapp - Bigo Live PK Session Manager

Desktop app for intercepting Bigo Live WebSocket messages and forwarding to BB-Core.

## Features

- ✅ Session-based BB-Core integration
- ✅ Automatic configuration fetching
- ✅ Gift and chat event forwarding
- ✅ Device fingerprinting for trial validation
- ✅ Heartbeat monitoring (30s intervals)
- ✅ Auto-reconnection for STOMP and browsers
- ✅ Real-time connection health dashboard

## Quick Start

1. Install BBapp
2. Start BB-Core at `http://localhost:8080`
3. Launch BBapp
4. Enter:
   - BB-Core URL
   - Authentication token
   - Room ID
5. Click "Start Session"
6. Monitor connection health in dashboard
7. Click "Stop Session" when done

## Development

```bash
# Install dependencies
go get ./...
npm install --prefix frontend

# Run in dev mode
wails dev

# Build production
wails build

# Run tests
go test ./... -v -short
```

## Architecture (Session-Based)

**Workflow:**

1. User enters Room ID → BB-Core fetches config
2. Config provides list of streamers → Browsers created automatically
3. POST /pk/start-from-bbapp called with device hash
4. Gift/chat events forwarded with complete metadata via STOMP
5. Heartbeat sent every 30s with connection status
6. Auto-reconnection for STOMP and browsers
7. POST /pk/stop-from-bbapp called on session end

**Key Components:**

- `internal/api/` - BB-Core REST API client
- `internal/session/` - Session and heartbeat management
- `internal/config/` - Configuration with streamer lookup
- `internal/fingerprint/` - Device hash generation
- `internal/listener/bigo.go` - Enhanced protocol parsing (gifts + chat)
- `internal/stomp/client.go` - Auto-reconnecting STOMP client

## Documentation

- [BB-CORE-INTEGRATION.md](BB-CORE-INTEGRATION.md) - BB-Core integration guide
- [TEST_INTEGRATION.md](TEST_INTEGRATION.md) - Integration testing
- [TEST_RESULTS.md](TEST_RESULTS.md) - Test status and manual testing procedures
- [CLAUDE.md](CLAUDE.md) - Technical architecture for Claude Code
- [Implementation Plans](docs/plans/) - Detailed implementation plans
