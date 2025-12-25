# PK Mode with Browser Overlay Design Document

**Date:** 2025-12-25
**Status:** Approved
**Author:** Claude Code (based on user requirements)

## Overview

Transform BBapp into a scene-based PK management platform with embedded HTTP server serving browser overlays for OBS integration. Remove Quick Start workflow in favor of comprehensive PK Mode with configuration management, session control, and real-time battle visualization overlays.

## Goals

1. Remove Quick Start workflow - focus on scene-based architecture
2. Implement login/authentication flow with BB-Core
3. Create PK Mode scene with team/streamer configuration UI
4. Embed HTTP server to serve overlay UI for OBS Browser Source
5. Connect overlay to BB-Core STOMP for real-time PK updates
6. Support multiple scenes (PK Mode now, future scenes later)
7. Each scene has its own overlay(s)

## Non-Goals

- Persist JWT tokens to disk (security concern)
- Manage overlay windows via chromedp (OBS handles display)
- SSE implementation (using STOMP instead)
- Multi-session support (one active session at a time)

---

## Section 1: Overall Architecture

### System Overview

BBapp becomes a **scene-based PK management platform** with:
- **Main Wails App**: Tab-based UI for scene selection and configuration
- **Embedded HTTP Server**: Auto-port HTTP server serving overlay UI
- **STOMP Integration**: Real-time updates from BB-Core to overlay
- **OBS Integration**: User copies URL to OBS Browser Source

### Key Components

**1. Scene Manager (Wails UI)**
- Tab navigation: [PK Mode] [Future Scene] [etc.]
- Each tab shows scene's configuration UI
- Prevents tab switching during active session
- Displays overlay URL when session starts

**2. Authentication Layer**
- Login page with username/password
- Connects to BB-Core: POST /api/v1/auth/login
- Stores tokens in memory (not persisted)
- Auto-refresh before expiration
- All API calls use Authorization header

**3. HTTP Overlay Server (Go)**
- Auto-selects available port (3000-3100 range)
- Serves React app at `/` and `/overlay`
- Static asset serving for overlay UI
- Starts with BBapp, runs continuously

**4. Overlay UI (React)**
- Single build, dynamic component loading
- URL params: `/overlay?scene=pk-mode&roomId={roomId}&bbCoreUrl={url}&token={jwt}`
- Connects to BB-Core STOMP for real-time updates
- Designed for OBS Browser Source (transparent background)

### Data Flow

```
Gift Event (Bigo) → BBapp (intercept) → STOMP → BB-Core
                     /app/room/{roomId}/bigo        ↓
                                              Business Logic
                                              (scoring, teams)
                                                     ↓
                                          /topic/room/{roomId}/pk
                                                     ↓
                                                Overlay (OBS)
                                                     ↓
                                               Stream viewers
```

### STOMP Message Flow

**BBapp → BB-Core** (already implemented):
- Destination: `/app/room/{roomId}/bigo`
- Types: GIFT, CHAT, STATUS

**BB-Core → Overlay** (new):
- Subscription: `/topic/room/{roomId}/pk`
- Type: `PK_SYNC`
- Contains: teamScores, leader, totalScore, activities

---

## Section 2: Frontend Structure & Scene Organization

### Directory Structure

```
frontend/src/
├── App.tsx                          # Main app with auth + scene tabs
├── components/
│   ├── LoginPage.tsx               # Authentication UI
│   ├── SceneTabs.tsx               # Tab navigation component
│   └── OverlayURLDisplay.tsx       # Shows copyable overlay URL
├── scenes/
│   ├── pk-mode/
│   │   ├── ui/
│   │   │   ├── PKModeScene.tsx    # Main PK mode component
│   │   │   ├── ConfigLoader.tsx   # Load/save config UI
│   │   │   ├── TeamEditor.tsx     # Team list editor
│   │   │   ├── TeamCard.tsx       # Team editor card
│   │   │   ├── StreamerCard.tsx   # Streamer editor card
│   │   │   ├── SessionControls.tsx # Start/stop buttons
│   │   │   └── SessionStatus.tsx  # Active session display
│   │   └── overlays/
│   │       └── battle/
│   │           ├── Overlay.tsx    # Battle visualization
│   │           └── useSTOMP.ts    # STOMP connection hook
│   └── [future-scene]/
│       ├── ui/
│       └── overlays/
├── overlay/
│   ├── OverlayApp.tsx             # Overlay entry point
│   └── OverlayRouter.tsx          # Routes to correct overlay component
└── shared/
    ├── types.ts                    # Shared TypeScript types
    ├── services/
    │   └── api.ts                 # BB-Core API client
    └── hooks/
        ├── useAuth.ts             # Authentication state
        └── useTokenRefresh.ts     # Auto token refresh
```

### App.tsx Flow

```tsx
function App() {
  const [isAuthenticated, setIsAuthenticated] = useState(false);
  const [activeScene, setActiveScene] = useState('pk-mode');
  const [sessionActive, setSessionActive] = useState(false);
  const [user, setUser] = useState(null);
  const [accessToken, setAccessToken] = useState('');

  const handleLogin = async (username: string, password: string) => {
    const response = await Login(username, password);
    setAccessToken(response.accessToken);
    setUser(response.user);
    setIsAuthenticated(true);
  };

  const handleSceneChange = (newScene: string) => {
    if (sessionActive) {
      alert('Please stop current session before switching scenes');
      return;
    }
    setActiveScene(newScene);
  };

  return (
    <div className="container">
      {!isAuthenticated ? (
        <LoginPage onLogin={handleLogin} />
      ) : (
        <>
          <Header user={user} onLogout={handleLogout} />
          <SceneTabs active={activeScene} onChange={handleSceneChange}>
            {activeScene === 'pk-mode' && (
              <PKModeScene
                token={accessToken}
                onSessionChange={setSessionActive}
              />
            )}
            {/* Future scenes */}
          </SceneTabs>
        </>
      )}
    </div>
  );
}
```

### Overlay Routing

**Overlay URL Format:**
```
http://localhost:{port}/overlay?scene=pk-mode&roomId={roomId}&bbCoreUrl={url}&token={jwt}
```

**OverlayRouter.tsx:**
```tsx
function OverlayRouter() {
  const params = new URLSearchParams(window.location.search);
  const scene = params.get('scene');
  const roomId = params.get('roomId');
  const bbCoreUrl = params.get('bbCoreUrl');
  const token = params.get('token');

  if (!scene || !roomId || !bbCoreUrl || !token) {
    return <ErrorPage message="Missing required parameters" />;
  }

  // Route to scene-specific overlay
  switch (scene) {
    case 'pk-mode':
      return <PKBattleOverlay roomId={roomId} bbCoreUrl={bbCoreUrl} token={token} />;
    default:
      return <ErrorPage message={`Unknown scene: ${scene}`} />;
  }
}
```

---

## Section 3: Data Flow & Real-Time Communication

### Complete Data Flow

```
Gift Event (Bigo) → BBapp (intercept) → STOMP → BB-Core (business logic)
                     /app/room/{roomId}/bigo          ↓
                                                  PK Score Engine
                                                       ↓
                                            /topic/room/{roomId}/pk
                                                       ↓
                                                  Overlay (OBS)
```

### STOMP Connection Details

**Endpoint:** `ws://{BB_CORE_URL}/buffb-stream` (from .env)
**Protocol:** STOMP over SockJS or native WebSocket
**Authentication:** JWT token in connection headers

### Overlay STOMP Connection

```typescript
// useSTOMP.ts
export function useSTOMP(roomId: string, bbCoreUrl: string, token: string) {
  const [pkState, setPkState] = useState<PKState | null>(null);
  const [connected, setConnected] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const socket = new SockJS(`${bbCoreUrl}/buffb-stream`);
    const stompClient = Stomp.over(socket);

    stompClient.connect(
      { 'Authorization': `Bearer ${token}` },
      () => {
        setConnected(true);
        setError(null);

        // Subscribe to PK updates
        stompClient.subscribe(`/topic/room/${roomId}/pk`, (message) => {
          const pkSync: PKSyncMessage = JSON.parse(message.body);
          setPkState(transformToState(pkSync.payload));
        });
      },
      (error) => {
        console.error('STOMP connection error:', error);
        setConnected(false);
        setError(error.toString());
        // Auto-reconnect logic with exponential backoff
      }
    );

    return () => {
      if (stompClient.connected) {
        stompClient.disconnect();
      }
    };
  }, [roomId, bbCoreUrl, token]);

  return { pkState, connected, error };
}
```

### PK_SYNC Message Format

**Received by overlay from BB-Core:**

```typescript
interface PKSyncMessage {
  type: "PK_SYNC";
  payload: {
    teamScores: Array<{
      teamId: string;        // UUID
      teamName: string;
      score: number;
      avatar: string;
      bindingGift: string;
    }>;
    leader: string;          // Team UUID with highest score
    totalScore: number;
    activities: Array<{      // Recent gift activities
      type: "GIFT";
      userName: string;
      teamName: string;
      giftName: string;
      giftCount: number;
      diamonds: number;
      timestamp: number;
    }>;
  };
}
```

### API Endpoints Used

**Authentication:**
- `POST /api/v1/auth/login` - Login with username/password
- `POST /api/v1/auth/refresh-token` - Refresh expired token

**Configuration:**
- `GET /api/v1/stream/rooms/{roomId}/bbapp-config` - Fetch room config
- `POST /api/v1/stream/rooms/{roomId}/bbapp-config` - Save/update config

**Session Management:**
- `POST /api/v1/stream/rooms/{roomId}/pk/start-from-bbapp` - Start PK session
- `POST /api/v1/stream/rooms/{roomId}/pk/stop-from-bbapp` - Stop PK session
- `POST /api/v1/stream/rooms/{roomId}/bbapp-status` - Heartbeat (already implemented)

---

## Section 4: User Workflow & UI Components

### Complete User Workflow

#### 1. Initial Launch & Authentication

```
User launches BBapp
→ Loads .env config (BB_CORE_URL only)
→ Shows Login page (if not authenticated)
→ User enters username + password
→ BBapp calls POST /api/v1/auth/login
→ Receives AuthResponse (accessToken, refreshToken, user, agency)
→ Stores tokens in memory (React state)
→ Shows scene tabs: [PK Mode] [Future Scene]
→ Auto-starts HTTP overlay server on available port (3000-3100)
```

#### 2. PK Mode Configuration

```
User switches to "PK Mode" tab
→ Shows PKModeScene component
→ User enters Room ID
→ Clicks "Load Configuration"
→ Fetches config from GET /api/v1/stream/rooms/{roomId}/bbapp-config
   (uses accessToken in Authorization header)
→ If 404: Loads default template (2 teams, 1 streamer each)
→ If 200: Displays teams/streamers in editable cards
→ User edits:
   - Team names, binding gifts
   - Streamer names, Bigo Room IDs, binding gifts
   - Add/remove teams and streamers
→ Clicks "Save Configuration"
→ POST /api/v1/stream/rooms/{roomId}/bbapp-config
→ Shows success message
```

#### 3. Session Start

```
User clicks "Start Session"
→ Validates configuration is saved
→ BBapp calls POST /api/v1/stream/rooms/{roomId}/pk/start-from-bbapp
   Request: { deviceHash, connectedStreamers }
   Authorization: Bearer {accessToken}
→ Receives PkSessionResponse { sessionId, startedAt, endsAt, status }
→ Starts chromedp browsers for each Bigo streamer (existing functionality)
→ Displays overlay URL in UI:
   "OBS Overlay URL: http://localhost:3247/overlay?scene=pk-mode&roomId=room123&bbCoreUrl=http://localhost:8080&token=xxx"
   [Copy URL] button
→ Shows "Session Active" status with:
   - Session ID
   - Timer (countdown to endsAt)
   - Connection status for each streamer
```

#### 4. OBS Integration

```
User copies overlay URL
→ Opens OBS Studio
→ Adds Browser Source
→ Pastes overlay URL
→ Sets dimensions (1920x1080 recommended)
→ Sets background to transparent
→ Overlay loads in OBS:
   - Connects to BB-Core STOMP
   - Subscribes to /topic/room/{roomId}/pk
   - Displays battle visualization
   - Shows initial state (teams with 0 scores)
```

#### 5. Live Session

```
BBapp intercepts Bigo gifts/chat (existing chromedp browsers)
→ Sends to BB-Core via STOMP: /app/room/{roomId}/bigo
   Message: { type: "GIFT", ... }
→ BB-Core processes:
   - Validates gift against team bindings
   - Calculates score with multipliers
   - Updates team scores
   - Broadcasts PK_SYNC to /topic/room/{roomId}/pk
→ Overlay receives PK_SYNC
→ Updates visualization:
   - Team scores animate
   - Progress bar adjusts
   - Gift notification floats up
   - Activities log updates
→ Stream viewers see live PK battle
```

#### 6. Session Stop

```
User clicks "Stop Session"
→ Confirmation prompt: "Stop PK session?"
→ BBapp calls POST /api/v1/stream/rooms/{roomId}/pk/stop-from-bbapp
   Request: { reason: "USER_STOPPED" }
→ Receives StopPkSessionResponse { sessionId, endedAt, finalScores }
→ Closes chromedp browsers
→ Shows final scores in UI
→ Overlay disconnects from STOMP
→ Overlay shows "Session Ended" with final scores
```

#### 7. Token Refresh (Automatic)

```
BBapp monitors token expiration
→ If accessToken expires soon (e.g., < 5 min)
→ Calls POST /api/v1/auth/refresh-token
   Request: { refreshToken }
→ Receives new accessToken and refreshToken
→ Updates tokens in memory
→ Continues session seamlessly
→ If refresh fails: Redirect to login page
```

### Main UI Components

```tsx
// App.tsx - Main app with auth + scene switching
<App>
  {!isAuthenticated ? (
    <LoginPage onLogin={handleLogin} />
  ) : (
    <>
      <Header user={user} onLogout={handleLogout} />
      <SceneTabs active={activeScene} onSwitch={handleSwitch}>
        {activeScene === 'pk-mode' && <PKModeScene token={accessToken} />}
      </SceneTabs>
    </>
  )}
</App>

// LoginPage.tsx - Authentication
<LoginPage>
  <input type="text" placeholder="Username" />
  <input type="password" placeholder="Password" />
  <button onClick={handleLogin}>Login</button>
  <p className="error">{errorMessage}</p>
</LoginPage>

// PKModeScene.tsx - Complete PK mode workflow
<PKModeScene token={accessToken}>
  <ConfigLoader token={token} onLoad={setConfig} />
  <TeamEditor config={config} onChange={setConfig} />
  <SessionControls
    token={token}
    config={config}
    onSessionStart={handleStart}
    onSessionStop={handleStop}
  />
  {sessionActive && (
    <>
      <OverlayURLDisplay url={overlayUrl} />
      <SessionStatus session={sessionData} />
    </>
  )}
</PKModeScene>

// Overlay (served at /overlay)
<OverlayApp>
  <OverlayRouter
    scene={scene}
    roomId={roomId}
    bbCoreUrl={bbCoreUrl}
    token={token}
  />
</OverlayApp>
```

### Environment Variables (.env)

```env
# Only BB-Core URL in .env, no secrets
BB_CORE_URL=http://localhost:8080
OVERLAY_SERVER_PORT_START=3000
```

### Token Storage

- **In-memory only** (React state)
- Not persisted to disk
- User must login on each app launch
- Future enhancement: Secure token storage with encryption

---

## Section 5: Error Handling & Edge Cases

### Authentication Errors

#### Login Failure
```
User enters wrong credentials
→ POST /api/v1/auth/login returns 401
→ Show error: "Invalid username or password"
→ Clear password field
→ Allow retry
```

#### Token Expired During Session
```
API call returns 401 Unauthorized
→ Auto-attempt refresh with refreshToken
→ If refresh succeeds: Retry original request
→ If refresh fails:
   - Show: "Session expired, please login again"
   - Save current state (room ID, scene)
   - Redirect to login page
   - After login: Restore state
```

### Configuration Errors

#### Room Not Found
```
User enters non-existent roomId
→ GET /bbapp-config returns 404
→ Show: "Room configuration not found"
→ Auto-load default template:
   - 2 teams (Team A, Team B)
   - 1 streamer each with placeholder values
→ User can edit and save as new config
```

#### Save Validation Failure
```
User tries to save invalid config
→ POST /bbapp-config returns 400
→ Parse error response for specific field errors
→ Highlight invalid fields in red
→ Show error messages below fields:
   - "Team name is required"
   - "Bigo Room ID must be unique"
   - "At least one streamer required per team"
→ Block save until all errors fixed
```

### Session Errors

#### Session Already Active
```
User tries to start session when one is already running
→ POST /pk/start-from-bbapp returns 409 Conflict
→ Show modal: "Session already active for this room"
→ Options:
   - [View Existing Session] - Show session details
   - [Force Stop & Restart] - Stop existing, start new
   - [Cancel] - Keep existing session
```

#### Session Start Failure - No Streamers Connected
```
User starts session but no streamers connected to Bigo
→ POST /pk/start-from-bbapp returns 400
→ Show error: "No streamers connected to Bigo Live"
→ Display connection status table:
   Streamer | Bigo Room ID | Status
   Alice    | room123      | DISCONNECTED
   Bob      | room456      | DISCONNECTED
→ Button: [Retry Connection]
→ User must fix Bigo connections before starting
```

#### Mid-Session STOMP Disconnection
```
Overlay STOMP connection lost
→ Overlay shows banner: "Connection lost, reconnecting..."
→ Auto-reconnect with exponential backoff:
   - Attempt 1: 1 second
   - Attempt 2: 2 seconds
   - Attempt 3: 4 seconds
   - Attempt 4: 8 seconds
   - Attempt 5: 16 seconds
   - Max: 30 seconds between attempts
→ If reconnect succeeds:
   - Hide banner
   - Resume from last state
→ If fails after 5 attempts:
   - Show error: "Unable to reconnect. Please refresh overlay."
   - Provide [Retry] button
```

#### Chromedp Browser Crash
```
BBapp detects no frames received for 30s
→ Update status: "{streamer} - RECONNECTING"
→ Close existing browser
→ Create new chromedp browser
→ Navigate to Bigo room
→ Resume event interception
→ Update status: "{streamer} - CONNECTED"
→ Log: "Recovered connection to {bigoRoomId}"
```

### Overlay Errors

#### Invalid Overlay URL Parameters
```
User copies incomplete URL
→ Missing roomId, token, or bbCoreUrl parameter
→ Overlay shows error page:
   "Invalid overlay URL. Please copy the complete URL from BBapp."
   Missing: roomId, token
→ Provide instructions:
   1. Open BBapp
   2. Start PK session
   3. Copy the complete overlay URL
   4. Paste into OBS Browser Source
```

#### STOMP Authentication Failure
```
Token invalid/expired in overlay
→ STOMP connection rejected (401)
→ Overlay shows:
   "Authentication failed. Token may be expired."
→ Instructions:
   "Please refresh the overlay URL from BBapp"
→ Provide [Retry Connection] button (in case temporary)
```

#### No PK_SYNC Messages Received
```
Overlay connected but no messages after 10 seconds
→ Show warning banner:
   "Connected, but waiting for session data..."
→ Check:
   - Is PK session started in BBapp?
   - Is roomId correct?
   - Are gifts being sent?
→ After 30 seconds: Show help text
```

### Tab Switching Protection

```
User tries to switch scene tab while session active
→ Intercept tab change
→ Show modal:
   "Active PK session detected"
   "You must stop the current session before switching scenes."
→ Buttons:
   [Stop Session & Switch] - Stops session, switches tab
   [Cancel] - Stay on current tab
→ If "Stop Session & Switch":
   - Call StopPKSession("USER_SWITCHED_SCENES")
   - Close chromedp browsers
   - Switch to new scene tab
```

### Network Errors

#### BB-Core Unreachable
```
All API calls fail with network error
→ Show error banner:
   "Cannot connect to BB-Core at http://localhost:8080"
→ Troubleshooting:
   - Is BB-Core running?
   - Check .env BB_CORE_URL setting
   - Check firewall settings
→ Provide [Retry] button
→ Provide [Settings] button to change URL
```

#### Slow Response (Timeout)
```
API call takes > 10 seconds
→ Show loading spinner with message:
   "Connecting to BB-Core..."
→ After 10s: Update message:
   "Request taking longer than expected. Please wait..."
→ After 30s: Timeout
→ Show error:
   "Request timed out. BB-Core may be slow or unresponsive."
→ Options: [Retry] [Cancel]
```

### Graceful Degradation

**Overlay Connection Issues:**
- Show "Connection Lost" banner, auto-reconnect
- Display last known state while reconnecting
- Animate reconnection attempts (spinner)

**Partial Team Data:**
- Display available teams
- Mark missing teams as "Loading..."
- Continue updating available data

**Missing Assets:**
- Avatar images fail to load → Use default avatar placeholder
- Gift images fail → Use gift name text only
- Background images fail → Use solid color

**Timer Desync:**
- Sync overlay timer with BB-Core `endsAt` timestamp every 5 seconds
- If server time differs significantly, show warning
- Allow manual refresh to resync

---

## Section 6: Implementation Notes & File Changes

### File Structure Summary

#### New Files to Create

**Frontend:**
```
frontend/src/
├── components/
│   ├── LoginPage.tsx              # Authentication UI
│   ├── Header.tsx                 # User info + logout
│   ├── SceneTabs.tsx              # Tab navigation
│   └── OverlayURLDisplay.tsx      # Copyable URL + instructions
├── scenes/
│   └── pk-mode/
│       ├── ui/
│       │   ├── PKModeScene.tsx    # Main PK mode component
│       │   ├── ConfigLoader.tsx   # Load/save config section
│       │   ├── TeamEditor.tsx     # Team list editor
│       │   ├── TeamCard.tsx       # Individual team card
│       │   ├── StreamerCard.tsx   # Individual streamer card
│       │   ├── SessionControls.tsx # Start/stop buttons
│       │   └── SessionStatus.tsx  # Active session display
│       └── overlays/
│           └── battle/
│               ├── Overlay.tsx    # Battle visualization
│               └── useSTOMP.ts    # STOMP connection hook
├── overlay/
│   ├── OverlayApp.tsx             # Overlay entry point
│   └── OverlayRouter.tsx          # Route to scene overlay
├── shared/
│   ├── types.ts                   # Shared TypeScript types
│   ├── services/
│   │   └── api.ts                 # BB-Core API client
│   └── hooks/
│       ├── useAuth.ts             # Authentication state
│       └── useTokenRefresh.ts     # Auto token refresh
└── styles/
    ├── scenes.css                 # Scene-specific styles
    └── overlay.css                # Overlay styles
```

**Backend (Go):**
```
internal/
├── overlayserver/
│   ├── server.go                  # HTTP server implementation
│   ├── port.go                    # Auto port selection logic
│   └── routes.go                  # Route handlers
└── auth/
    └── client.go                  # BB-Core auth client

.env                               # BB-Core URL configuration
```

#### Modified Files

```
frontend/src/
├── App.tsx                        # Complete rewrite: auth + scenes
└── App.css                        # Add scene tab styles

app.go                             # Add methods:
                                   # - Login()
                                   # - RefreshToken()
                                   # - GetOverlayURL()
                                   # - StartOverlayServer()
                                   # - GetBBCoreURL()
```

### Key Implementation Details

#### 1. HTTP Overlay Server (Go)

```go
// internal/overlayserver/server.go
package overlayserver

import (
    "fmt"
    "log"
    "net"
    "net/http"
)

type Server struct {
    port int
    mux  *http.ServeMux
}

func NewServer() (*Server, error) {
    // Find available port in range 3000-3100
    port, err := findAvailablePort(3000, 3100)
    if err != nil {
        return nil, fmt.Errorf("no available ports: %w", err)
    }

    s := &Server{
        port: port,
        mux:  http.NewServeMux(),
    }
    s.setupRoutes()
    return s, nil
}

func findAvailablePort(start, end int) (int, error) {
    for port := start; port <= end; port++ {
        addr := fmt.Sprintf(":%d", port)
        ln, err := net.Listen("tcp", addr)
        if err == nil {
            ln.Close()
            return port, nil
        }
    }
    return 0, fmt.Errorf("no available ports in range %d-%d", start, end)
}

func (s *Server) setupRoutes() {
    // Serve React build for overlay
    fs := http.FileServer(http.Dir("./frontend/dist"))
    s.mux.Handle("/", fs)
}

func (s *Server) Start() error {
    addr := fmt.Sprintf(":%d", s.port)
    log.Printf("Overlay server starting on http://localhost%s", addr)
    return http.ListenAndServe(addr, s.mux)
}

func (s *Server) GetURL() string {
    return fmt.Sprintf("http://localhost:%d", s.port)
}
```

#### 2. App.go Methods

```go
// app.go additions

func (a *App) startup(ctx context.Context) {
    a.ctx = ctx

    // Load .env
    err := godotenv.Load()
    if err != nil {
        log.Println("No .env file found, using defaults")
    }

    bbCoreURL := os.Getenv("BB_CORE_URL")
    if bbCoreURL == "" {
        bbCoreURL = "http://localhost:8080"
    }
    a.bbCoreURL = bbCoreURL

    // Start overlay server
    server, err := overlayserver.NewServer()
    if err != nil {
        log.Fatal("Failed to start overlay server:", err)
    }
    a.overlayServer = server
    go server.Start()
}

// Login authenticates with BB-Core
func (a *App) Login(username, password string) (*api.AuthResponse, error) {
    client := api.NewClient(a.bbCoreURL, "")
    return client.Login(username, password)
}

// RefreshToken gets a new access token
func (a *App) RefreshToken(refreshToken string) (*api.AuthResponse, error) {
    client := api.NewClient(a.bbCoreURL, "")
    return client.RefreshToken(refreshToken)
}

// GetOverlayURL generates the overlay URL for OBS
func (a *App) GetOverlayURL(scene, roomId, token string) string {
    baseURL := a.overlayServer.GetURL()
    return fmt.Sprintf("%s/overlay?scene=%s&roomId=%s&bbCoreUrl=%s&token=%s",
        baseURL, scene, roomId, a.bbCoreURL, token)
}

// GetBBCoreURL returns configured BB-Core URL
func (a *App) GetBBCoreURL() string {
    return a.bbCoreURL
}
```

#### 3. STOMP Hook (TypeScript)

```typescript
// scenes/pk-mode/overlays/battle/useSTOMP.ts
import { useState, useEffect } from 'react';
import SockJS from 'sockjs-client';
import Stomp from 'stompjs';

interface PKState {
  teams: Array<{
    id: string;
    name: string;
    score: number;
    avatar: string;
    bindingGift: string;
  }>;
  leader: string;
  totalScore: number;
  lastActivity: {
    userName: string;
    teamName: string;
    giftName: string;
    diamonds: number;
  } | null;
}

export function useSTOMP(roomId: string, bbCoreUrl: string, token: string) {
  const [pkState, setPkState] = useState<PKState | null>(null);
  const [connected, setConnected] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const socket = new SockJS(`${bbCoreUrl}/buffb-stream`);
    const stompClient = Stomp.over(socket);

    // Disable debug logging in production
    stompClient.debug = null;

    const onConnect = () => {
      console.log('STOMP connected');
      setConnected(true);
      setError(null);

      // Subscribe to PK updates
      stompClient.subscribe(`/topic/room/${roomId}/pk`, (message) => {
        try {
          const pkSync = JSON.parse(message.body);

          // Transform BB-Core format to component state
          const state: PKState = {
            teams: pkSync.payload.teamScores.map((team: any) => ({
              id: team.teamId,
              name: team.teamName,
              score: team.score,
              avatar: team.avatar,
              bindingGift: team.bindingGift,
            })),
            leader: pkSync.payload.leader,
            totalScore: pkSync.payload.totalScore,
            lastActivity: pkSync.payload.activities?.[0] || null,
          };

          setPkState(state);
        } catch (err) {
          console.error('Failed to parse PK_SYNC message:', err);
        }
      });
    };

    const onError = (err: any) => {
      console.error('STOMP connection error:', err);
      setConnected(false);
      setError(err.toString());

      // Auto-reconnect logic with exponential backoff
      // (implement reconnection logic here)
    };

    stompClient.connect(
      { 'Authorization': `Bearer ${token}` },
      onConnect,
      onError
    );

    return () => {
      if (stompClient.connected) {
        stompClient.disconnect(() => {
          console.log('STOMP disconnected');
        });
      }
    };
  }, [roomId, bbCoreUrl, token]);

  return { pkState, connected, error };
}
```

#### 4. Scene Tab Protection

```typescript
// App.tsx
const handleSceneChange = async (newScene: string) => {
  if (sessionActive) {
    const confirmed = window.confirm(
      'Active session detected. Stop the current session before switching scenes?'
    );

    if (!confirmed) return;

    try {
      // Stop session
      await StopPKSession('USER_SWITCHED_SCENES');
      setSessionActive(false);
      setActiveScene(newScene);
    } catch (error) {
      alert(`Failed to stop session: ${error}`);
    }
  } else {
    setActiveScene(newScene);
  }
};
```

#### 5. Environment Configuration

**.env File:**
```env
# BB-Core URL (required)
BB_CORE_URL=http://localhost:8080

# Overlay server port range start (optional)
OVERLAY_SERVER_PORT_START=3000
```

**Production deployment:**
- Update BB_CORE_URL to production BB-Core instance
- Ensure firewall allows overlay server port range
- JWT tokens never persisted, always in-memory

### Dependencies

**Go Packages:**
- `github.com/joho/godotenv` - .env file loading (already used)
- No new dependencies needed

**NPM Packages:**
```json
{
  "dependencies": {
    "sockjs-client": "^1.6.1",
    "stompjs": "^2.3.3",
    "lucide-react": "^0.263.1"
  }
}
```

### Build Process

**Development:**
```bash
# Install dependencies
npm install --prefix frontend

# Run in dev mode
wails dev
```

**Production:**
```bash
# Build frontend
npm run build --prefix frontend

# Build Wails app (includes frontend build)
wails build

# Output: build/bin/bbapp.exe
```

### Testing Strategy

#### Manual Testing Checklist

**Authentication:**
- [ ] Login with valid credentials succeeds
- [ ] Login with invalid credentials shows error
- [ ] Token auto-refresh works before expiration
- [ ] Logout clears tokens and redirects to login

**Configuration Management:**
- [ ] Load existing room config displays correctly
- [ ] Load non-existent room shows default template
- [ ] Edit team names, binding gifts works
- [ ] Add/remove teams works (min 1 team enforced)
- [ ] Edit streamer details works
- [ ] Add/remove streamers works (min 1 per team enforced)
- [ ] Save config with validation succeeds
- [ ] Save invalid config shows errors

**Session Management:**
- [ ] Start session generates overlay URL
- [ ] Copy URL button works
- [ ] Session status displays correctly
- [ ] Timer counts down accurately
- [ ] Connection status shows for each streamer
- [ ] Stop session cleans up properly

**Overlay Integration:**
- [ ] Paste URL into OBS Browser Source works
- [ ] Overlay connects to STOMP
- [ ] Overlay displays initial team state
- [ ] Gift events update scores in real-time
- [ ] Animations play correctly
- [ ] Timer syncs with backend

**Error Handling:**
- [ ] Network error shows helpful message
- [ ] Token expiration handled gracefully
- [ ] STOMP disconnection auto-reconnects
- [ ] Tab switching blocked during session
- [ ] Invalid overlay URL shows error page

**Edge Cases:**
- [ ] Multiple BBapp instances (different ports)
- [ ] Overlay refresh during session maintains state
- [ ] Browser reconnection on Bigo disconnection
- [ ] Long session (> 1 hour) token refresh works

### Deployment Notes

**Environment Variables:**
- Set `BB_CORE_URL` in .env for production
- Overlay server auto-selects port (no manual config)
- JWT tokens never persisted to disk

**Security:**
- Tokens stored in memory only
- HTTPS recommended for production BB-Core
- STOMP authentication via JWT in headers

**Performance:**
- Overlay updates throttled to prevent lag
- STOMP reconnection uses exponential backoff
- Chromedp browsers isolated per streamer

---

## Future Enhancements (Out of Scope)

1. **Persistent Token Storage**: Encrypted token storage for "remember me"
2. **Multiple Overlay Types**: Leaderboard, chat overlay, donation ticker
3. **Overlay Customization**: Colors, fonts, animations via UI
4. **Multi-Session Support**: Run multiple PK sessions simultaneously
5. **Analytics Dashboard**: Session history, statistics, reports
6. **Mobile App**: Control sessions from phone
7. **Cloud Sync**: Sync configs across devices
8. **Replay Mode**: Replay past PK sessions

---

## Success Criteria

- ✅ Login/authentication with BB-Core works
- ✅ PK Mode configuration UI functional
- ✅ Save/load configurations to BB-Core
- ✅ Overlay server starts on available port
- ✅ Overlay URL generated correctly
- ✅ Overlay connects to BB-Core STOMP
- ✅ Real-time score updates display correctly
- ✅ Session lifecycle (start/stop) works
- ✅ Tab switching protection prevents accidents
- ✅ Error handling provides helpful messages
- ✅ OBS Browser Source integration works
- ✅ No JWT tokens persisted to disk
