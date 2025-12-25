# BBapp MVP (Wails + chromedp) Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build minimal viable BBapp that intercepts Bigo Live WebSocket messages via hidden browser and forwards to BB-Core via STOMP.

**Architecture:** Wails desktop app with Go backend (chromedp for browser control, STOMP client) and React frontend (connection management UI). Each owner  gets a hidden Chrome instance that navigates to their room, intercepts WebSocket frames, parses gift/chat events, and forwards to BB-Core.

**Tech Stack:**
- Wails v2 (Go + React)
- chromedp (Chrome DevTools Protocol)
- go-stomp/stomp (STOMP client)
- React + TypeScript (UI)

**MVP Scope:**
- ✅ Launch hidden browsers for Bigo rooms
- ✅ Intercept WebSocket frames
- ✅ Parse gift events (chat optional)
- ✅ Forward to BB-Core via STOMP
- ✅ Basic file logging
- ✅ Simple connection management UI
- ❌ Advanced statistics (post-MVP)
- ❌ Real-time charts (post-MVP)
- ❌ Export features (post-MVP)

---

## Pre-Implementation Setup

### Task 0: Project Initialization

**Files:**
- Create: Project structure

**Step 1: Install Wails CLI**

```bash
go install github.com/wailsapp/wails/v2/cmd/wails@latest
```

**Step 2: Create Wails project**

```bash
wails init -n bbapp -t react-ts
cd bbapp
```

**Step 3: Add Go dependencies**

```bash
go get github.com/chromedp/chromedp@latest
go get github.com/chromedp/cdproto@latest
go get github.com/go-stomp/stomp/v3@latest
```

**Step 4: Verify project builds**

```bash
wails dev
```

Expected: App window opens with default Wails template

**Step 5: Commit initial setup**

```bash
git init
git add .
git commit -m "chore: initialize Wails project with dependencies"
```

---

## Phase 1: Core Browser Management

### Task 1: Browser Manager Structure

**Files:**
- Create: `internal/browser/manager.go`
- Create: `internal/browser/manager_test.go`

**Step 1: Write failing test for browser creation**

Create `internal/browser/manager_test.go`:

```go
package browser_test

import (
    "testing"
    "bbapp/internal/browser"
)

func TestManager_CreateBrowser(t *testing.T) {
    manager := browser.NewManager()

    ctx, cancel, err := manager.CreateBrowser("test-id")
    if err != nil {
        t.Fatalf("CreateBrowser failed: %v", err)
    }
    defer cancel()

    if ctx == nil {
        t.Fatal("Expected non-nil context")
    }
}
```

**Step 2: Run test to verify failure**

```bash
go test ./internal/browser -v
```

Expected: FAIL - package browser does not exist

**Step 3: Create Manager struct**

Create `internal/browser/manager.go`:

```go
package browser

import (
    "context"
    "sync"

    "github.com/chromedp/chromedp"
)

// Manager manages browser instances
type Manager struct {
    browsers map[string]context.Context
    mutex    sync.RWMutex
}

// NewManager creates a new browser manager
func NewManager() *Manager {
    return &Manager{
        browsers: make(map[string]context.Context),
    }
}

// CreateBrowser creates a headless Chrome instance
func (m *Manager) CreateBrowser(id string) (context.Context, context.CancelFunc, error) {
    opts := append(chromedp.DefaultExecAllocatorOptions[:],
        chromedp.Flag("headless", true),
        chromedp.Flag("disable-gpu", true),
        chromedp.Flag("no-sandbox", true),
    )

    allocCtx, cancelAlloc := chromedp.NewExecAllocator(context.Background(), opts...)
    ctx, cancel := chromedp.NewContext(allocCtx)

    // Start browser
    if err := chromedp.Run(ctx); err != nil {
        cancel()
        cancelAlloc()
        return nil, nil, err
    }

    m.mutex.Lock()
    m.browsers[id] = ctx
    m.mutex.Unlock()

    return ctx, func() {
        cancel()
        cancelAlloc()
    }, nil
}
```

**Step 4: Run test to verify pass**

```bash
go test ./internal/browser -v
```

Expected: PASS

**Step 5: Commit**

```bash
git add internal/browser/
git commit -m "feat: add browser manager with creation"
```

---

### Task 2: Browser Navigation

**Files:**
- Modify: `internal/browser/manager.go`
- Modify: `internal/browser/manager_test.go`

**Step 1: Write test for navigation**

Add to `internal/browser/manager_test.go`:

```go
func TestManager_Navigate(t *testing.T) {
    manager := browser.NewManager()
    ctx, cancel, _ := manager.CreateBrowser("nav-test")
    defer cancel()

    err := manager.Navigate(ctx, "https://example.com")
    if err != nil {
        t.Fatalf("Navigate failed: %v", err)
    }
}
```

**Step 2: Run test**

```bash
go test ./internal/browser -v -run TestManager_Navigate
```

Expected: FAIL - Manager.Navigate undefined

**Step 3: Implement Navigate**

Add to `internal/browser/manager.go`:

```go
import (
    "github.com/chromedp/chromedp"
    "time"
)

// Navigate navigates browser to URL
func (m *Manager) Navigate(ctx context.Context, url string) error {
    return chromedp.Run(ctx,
        chromedp.Navigate(url),
        chromedp.Sleep(2*time.Second), // Wait for page load
    )
}
```

**Step 4: Run test**

```bash
go test ./internal/browser -v -run TestManager_Navigate
```

Expected: PASS

**Step 5: Commit**

```bash
git add internal/browser/
git commit -m "feat: add browser navigation"
```

---

## Phase 2: WebSocket Interception

### Task 3: WebSocket Frame Listener

**Files:**
- Create: `internal/listener/websocket.go`
- Create: `internal/listener/websocket_test.go`

**Step 1: Write test for frame interception**

Create `internal/listener/websocket_test.go`:

```go
package listener_test

import (
    "testing"
    "bbapp/internal/listener"
)

func TestWebSocketListener_OnFrame(t *testing.T) {
    frameReceived := false

    wsl := listener.NewWebSocketListener()
    wsl.OnFrame(func(data string) {
        frameReceived = true
    })

    // Simulate frame
    wsl.HandleFrame(`{"type":"test"}`)

    if !frameReceived {
        t.Fatal("Expected frame to be received")
    }
}
```

**Step 2: Run test**

```bash
go test ./internal/listener -v
```

Expected: FAIL - package does not exist

**Step 3: Implement WebSocketListener**

Create `internal/listener/websocket.go`:

```go
package listener

import (
    "sync"
)

// FrameHandler handles WebSocket frames
type FrameHandler func(data string)

// WebSocketListener listens to WebSocket frames
type WebSocketListener struct {
    handlers []FrameHandler
    mutex    sync.RWMutex
}

// NewWebSocketListener creates new listener
func NewWebSocketListener() *WebSocketListener {
    return &WebSocketListener{
        handlers: make([]FrameHandler, 0),
    }
}

// OnFrame registers a frame handler
func (w *WebSocketListener) OnFrame(handler FrameHandler) {
    w.mutex.Lock()
    defer w.mutex.Unlock()
    w.handlers = append(w.handlers, handler)
}

// HandleFrame processes a WebSocket frame
func (w *WebSocketListener) HandleFrame(data string) {
    w.mutex.RLock()
    defer w.mutex.RUnlock()

    for _, handler := range w.handlers {
        handler(data)
    }
}
```

**Step 4: Run test**

```bash
go test ./internal/listener -v
```

Expected: PASS

**Step 5: Commit**

```bash
git add internal/listener/
git commit -m "feat: add WebSocket frame listener"
```

---

### Task 4: Integrate WebSocket Interception with chromedp

**Files:**
- Create: `internal/listener/bigo.go`
- Create: `internal/listener/bigo_test.go`

**Step 1: Write test for Bigo listener**

Create `internal/listener/bigo_test.go`:

```go
package listener_test

import (
    "context"
    "testing"
    "bbapp/internal/listener"
    "bbapp/internal/browser"
)

func TestBigoListener_Start(t *testing.T) {
    manager := browser.NewManager()
    ctx, cancel, _ := manager.CreateBrowser("bigo-test")
    defer cancel()

    bigoListener := listener.NewBigoListener("12345", ctx)

    frameCount := 0
    bigoListener.OnGift(func(gift listener.Gift) {
        frameCount++
    })

    err := bigoListener.Start()
    if err != nil {
        t.Fatalf("Start failed: %v", err)
    }
}
```

**Step 2: Run test**

```bash
go test ./internal/listener -v -run TestBigoListener
```

Expected: FAIL - BigoListener undefined

**Step 3: Implement BigoListener**

Create `internal/listener/bigo.go`:

```go
package listener

import (
    "context"
    "encoding/json"
    "log"

    "github.com/chromedp/cdproto/network"
    "github.com/chromedp/chromedp"
)

// Gift represents a Bigo gift
type Gift struct {
    BigoUid   string
    Nickname  string
    GiftName  string
    GiftValue int64
}

// GiftHandler handles gift events
type GiftHandler func(Gift)

// BigoListener listens to Bigo room WebSocket
type BigoListener struct {
    roomId       string
    ctx          context.Context
    giftHandlers []GiftHandler
}

// NewBigoListener creates new Bigo listener
func NewBigoListener(roomId string, ctx context.Context) *BigoListener {
    return &BigoListener{
        roomId:       roomId,
        ctx:          ctx,
        giftHandlers: make([]GiftHandler, 0),
    }
}

// OnGift registers gift handler
func (b *BigoListener) OnGift(handler GiftHandler) {
    b.giftHandlers = append(b.giftHandlers, handler)
}

// Start starts listening
func (b *BigoListener) Start() error {
    // Setup WebSocket frame listener
    chromedp.ListenTarget(b.ctx, func(ev interface{}) {
        switch ev := ev.(type) {
        case *network.EventWebSocketFrameReceived:
            b.handleFrame(ev.Response.PayloadData)
        }
    })

    // Navigate to Bigo room
    return chromedp.Run(b.ctx,
        network.Enable(),
        chromedp.Navigate("https://www.bigo.tv/"+b.roomId),
    )
}

// handleFrame processes WebSocket frame
func (b *BigoListener) handleFrame(data string) {
    var msg map[string]interface{}
    if err := json.Unmarshal([]byte(data), &msg); err != nil {
        return
    }

    msgType, ok := msg["type"].(string)
    if !ok || msgType != "GIFT" {
        return
    }

    // Parse gift
    gift := b.parseGift(msg)

    // Notify handlers
    for _, handler := range b.giftHandlers {
        handler(gift)
    }
}

// parseGift extracts gift data
func (b *BigoListener) parseGift(msg map[string]interface{}) Gift {
    sender := msg["sender"].(map[string]interface{})
    giftData := msg["gift"].(map[string]interface{})

    return Gift{
        BigoUid:   sender["id"].(string),
        Nickname:  sender["nickname"].(string),
        GiftName:  giftData["name"].(string),
        GiftValue: int64(giftData["diamonds"].(float64)),
    }
}
```

**Step 4: Run test**

```bash
go test ./internal/listener -v -run TestBigoListener
```

Expected: PASS

**Step 5: Commit**

```bash
git add internal/listener/
git commit -m "feat: add Bigo listener with WebSocket interception"
```

---

## Phase 3: STOMP Client

### Task 5: STOMP Connection

**Files:**
- Create: `internal/stomp/client.go`
- Create: `internal/stomp/client_test.go`

**Step 1: Write test for STOMP client**

Create `internal/stomp/client_test.go`:

```go
package stomp_test

import (
    "testing"
    "bbapp/internal/stomp"
)

func TestClient_Connect(t *testing.T) {
    // Skip if no STOMP server available
    if testing.Short() {
        t.Skip("Skipping integration test")
    }

    client, err := stomp.NewClient("localhost:61613", "", "")
    if err != nil {
        t.Fatalf("Connect failed: %v", err)
    }
    defer client.Disconnect()

    if client == nil {
        t.Fatal("Expected non-nil client")
    }
}
```

**Step 2: Run test**

```bash
go test ./internal/stomp -v -short
```

Expected: SKIP (no server) or FAIL (package not found)

**Step 3: Implement STOMP client**

Create `internal/stomp/client.go`:

```go
package stomp

import (
    "fmt"
    "net"
    "time"

    "github.com/go-stomp/stomp/v3"
)

// Client wraps STOMP connection
type Client struct {
    conn *stomp.Conn
}

// NewClient creates STOMP client
func NewClient(url, username, password string) (*Client, error) {
    netConn, err := net.DialTimeout("tcp", url, 10*time.Second)
    if err != nil {
        return nil, fmt.Errorf("dial failed: %w", err)
    }

    var opts []func(*stomp.Conn) error
    if username != "" {
        opts = append(opts, stomp.ConnOpt.Login(username, password))
    }

    conn, err := stomp.Connect(netConn, opts...)
    if err != nil {
        netConn.Close()
        return nil, fmt.Errorf("STOMP connect failed: %w", err)
    }

    return &Client{conn: conn}, nil
}

// Disconnect closes connection
func (c *Client) Disconnect() error {
    if c.conn != nil {
        return c.conn.Disconnect()
    }
    return nil
}
```

**Step 4: Run test with STOMP server**

```bash
# Start BB-Core or STOMP server first
go test ./internal/stomp -v
```

Expected: PASS (if server running) or SKIP

**Step 5: Commit**

```bash
git add internal/stomp/
git commit -m "feat: add STOMP client"
```

---

### Task 6: STOMP Message Publishing

**Files:**
- Modify: `internal/stomp/client.go`
- Modify: `internal/stomp/client_test.go`

**Step 1: Write test for publishing**

Add to `internal/stomp/client_test.go`:

```go
func TestClient_Publish(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test")
    }

    client, _ := stomp.NewClient("localhost:61613", "", "")
    defer client.Disconnect()

    payload := map[string]interface{}{
        "type": "TEST",
        "data": "hello",
    }

    err := client.Publish("/app/test", payload)
    if err != nil {
        t.Fatalf("Publish failed: %v", err)
    }
}
```

**Step 2: Run test**

```bash
go test ./internal/stomp -v -run TestClient_Publish
```

Expected: FAIL - Publish undefined

**Step 3: Implement Publish**

Add to `internal/stomp/client.go`:

```go
import (
    "encoding/json"
)

// Publish sends message to destination
func (c *Client) Publish(destination string, payload interface{}) error {
    data, err := json.Marshal(payload)
    if err != nil {
        return fmt.Errorf("marshal failed: %w", err)
    }

    return c.conn.Send(
        destination,
        "application/json",
        data,
    )
}
```

**Step 4: Run test**

```bash
go test ./internal/stomp -v -run TestClient_Publish
```

Expected: PASS

**Step 5: Commit**

```bash
git add internal/stomp/
git commit -m "feat: add STOMP message publishing"
```

---

## Phase 4: Activity Logging

### Task 7: Basic File Logger

**Files:**
- Create: `internal/logger/logger.go`
- Create: `internal/logger/logger_test.go`

**Step 1: Write test for logging**

Create `internal/logger/logger_test.go`:

```go
package logger_test

import (
    "os"
    "testing"
    "bbapp/internal/logger"
)

func TestLogger_Log(t *testing.T) {
    tempDir := t.TempDir()

    log, err := logger.NewLogger(tempDir)
    if err != nil {
        t.Fatalf("NewLogger failed: %v", err)
    }
    defer log.Close()

    err = log.LogGift("12345", "user1", "Rose", 100)
    if err != nil {
        t.Fatalf("LogGift failed: %v", err)
    }

    // Verify file exists
    files, _ := os.ReadDir(tempDir)
    if len(files) == 0 {
        t.Fatal("Expected log file to be created")
    }
}
```

**Step 2: Run test**

```bash
go test ./internal/logger -v
```

Expected: FAIL - package not found

**Step 3: Implement Logger**

Create `internal/logger/logger.go`:

```go
package logger

import (
    "encoding/json"
    "fmt"
    "os"
    "path/filepath"
    "time"
)

// Logger handles activity logging
type Logger struct {
    logDir string
    file   *os.File
}

// NewLogger creates new logger
func NewLogger(logDir string) (*Logger, error) {
    if err := os.MkdirAll(logDir, 0755); err != nil {
        return nil, err
    }

    filename := filepath.Join(logDir, fmt.Sprintf("bbapp_%s.jsonl",
        time.Now().Format("2006-01-02")))

    file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
    if err != nil {
        return nil, err
    }

    return &Logger{
        logDir: logDir,
        file:   file,
    }, nil
}

// LogGift logs a gift event
func (l *Logger) LogGift(bigoRoomId, nickname, giftName string, value int64) error {
    entry := map[string]interface{}{
        "timestamp":  time.Now().Unix(),
        "type":       "GIFT",
        "bigoRoomId": bigoRoomId,
        "nickname":   nickname,
        "giftName":   giftName,
        "value":      value,
    }

    data, err := json.Marshal(entry)
    if err != nil {
        return err
    }

    _, err = l.file.Write(append(data, '\n'))
    return err
}

// Close closes logger
func (l *Logger) Close() error {
    if l.file != nil {
        return l.file.Close()
    }
    return nil
}
```

**Step 4: Run test**

```bash
go test ./internal/logger -v
```

Expected: PASS

**Step 5: Commit**

```bash
git add internal/logger/
git commit -m "feat: add activity file logger"
```

---

## Phase 5: Wails App Integration

### Task 8: Main App Structure

**Files:**
- Modify: `app.go`

**Step 1: Write app structure**

Replace `app.go` content:

```go
package main

import (
    "context"
    "fmt"
    "sync"

    "bbapp/internal/browser"
    "bbapp/internal/listener"
    "bbapp/internal/logger"
    "bbapp/internal/stomp"
)

// App struct
type App struct {
    ctx         context.Context
    browserMgr  *browser.Manager
    stompClient *stomp.Client
    logger      *logger.Logger
    listeners   map[string]*listener.BigoListener
    mutex       sync.RWMutex
}

// NewApp creates new App
func NewApp() *App {
    return &App{
        listeners: make(map[string]*listener.BigoListener),
    }
}

// startup is called when app starts
func (a *App) startup(ctx context.Context) {
    a.ctx = ctx
    a.browserMgr = browser.NewManager()

    // Initialize logger
    log, err := logger.NewLogger("./logs")
    if err != nil {
        panic(err)
    }
    a.logger = log
}

// shutdown is called on app termination
func (a *App) shutdown(ctx context.Context) {
    if a.stompClient != nil {
        a.stompClient.Disconnect()
    }
    if a.logger != nil {
        a.logger.Close()
    }
}
```

**Step 2: Build to verify**

```bash
wails build
```

Expected: Build succeeds

**Step 3: Commit**

```bash
git add app.go
git commit -m "feat: integrate core services in App"
```

---

### Task 9: STOMP Connection Method

**Files:**
- Modify: `app.go`

**Step 1: Add ConnectToCore method**

Add to `app.go`:

```go
// ConnectToCore connects to BB-Core STOMP
func (a *App) ConnectToCore(url, username, password string) error {
    client, err := stomp.NewClient(url, username, password)
    if err != nil {
        return fmt.Errorf("connection failed: %w", err)
    }

    a.stompClient = client
    return nil
}
```

**Step 2: Build to verify**

```bash
wails build
```

Expected: Build succeeds

**Step 3: Commit**

```bash
git add app.go
git commit -m "feat: add STOMP connection method"
```

---

### Task 10: Add Streamer Method

**Files:**
- Modify: `app.go`

**Step 1: Add AddStreamer method**

Add to `app.go`:

```go
// AddStreamer adds Bigo streamer to monitor
func (a *App) AddStreamer(bigoRoomId, teamId, roomId string) error {
    a.mutex.Lock()
    defer a.mutex.Unlock()

    // Check if already exists
    if _, exists := a.listeners[bigoRoomId]; exists {
        return fmt.Errorf("already monitoring room %s", bigoRoomId)
    }

    // Create browser
    ctx, cancel, err := a.browserMgr.CreateBrowser(bigoRoomId)
    if err != nil {
        return err
    }

    // Create listener
    bigoListener := listener.NewBigoListener(bigoRoomId, ctx)

    // Setup gift handler
    bigoListener.OnGift(func(gift listener.Gift) {
        // Log activity
        a.logger.LogGift(bigoRoomId, gift.Nickname, gift.GiftName, gift.GiftValue)

        // Send to BB-Core
        if a.stompClient != nil {
            payload := map[string]interface{}{
                "type":      "GIFT",
                "bigoId":    gift.BigoUid,
                "nickname":  gift.Nickname,
                "giftName":  gift.GiftName,
                "giftValue": gift.GiftValue,
            }
            a.stompClient.Publish("/app/room/"+roomId+"/bigo", payload)
        }
    })

    // Start listening
    if err := bigoListener.Start(); err != nil {
        cancel()
        return err
    }

    a.listeners[bigoRoomId] = bigoListener
    return nil
}
```

**Step 2: Build to verify**

```bash
wails build
```

Expected: Build succeeds

**Step 3: Commit**

```bash
git add app.go
git commit -m "feat: add streamer monitoring with gift forwarding"
```

---

### Task 11: Remove Streamer Method

**Files:**
- Modify: `app.go`

**Step 1: Add RemoveStreamer method**

Add to `app.go`:

```go
// RemoveStreamer stops monitoring a streamer
func (a *App) RemoveStreamer(bigoRoomId string) error {
    a.mutex.Lock()
    defer a.mutex.Unlock()

    if _, exists := a.listeners[bigoRoomId]; !exists {
        return fmt.Errorf("not monitoring room %s", bigoRoomId)
    }

    delete(a.listeners, bigoRoomId)
    // Browser cleanup handled by context cancel

    return nil
}

// GetConnections returns active connections
func (a *App) GetConnections() []map[string]string {
    a.mutex.RLock()
    defer a.mutex.RUnlock()

    var connections []map[string]string
    for roomId := range a.listeners {
        connections = append(connections, map[string]string{
            "bigoRoomId": roomId,
            "status":     "connected",
        })
    }
    return connections
}
```

**Step 2: Build to verify**

```bash
wails build
```

Expected: Build succeeds

**Step 3: Commit**

```bash
git add app.go
git commit -m "feat: add remove streamer and get connections"
```

---

## Phase 6: React Frontend (MVP)

### Task 12: Basic UI Components

**Files:**
- Modify: `frontend/src/App.tsx`
- Modify: `frontend/src/App.css`

**Step 1: Create basic UI**

Replace `frontend/src/App.tsx`:

```typescript
import { useState } from 'react';
import './App.css';
import { ConnectToCore, AddStreamer, RemoveStreamer } from '../wailsjs/go/main/App';

function App() {
  const [coreUrl, setCoreUrl] = useState('localhost:61613');
  const [connected, setConnected] = useState(false);

  const [bigoRoomId, setBigoRoomId] = useState('');
  const [teamId, setTeamId] = useState('');
  const [roomId, setRoomId] = useState('');

  const handleConnect = async () => {
    try {
      await ConnectToCore(coreUrl, '', '');
      setConnected(true);
      alert('Connected to BB-Core!');
    } catch (error) {
      alert(`Failed: ${error}`);
    }
  };

  const handleAddStreamer = async () => {
    if (!bigoRoomId || !teamId || !roomId) {
      alert('Fill all fields');
      return;
    }

    try {
      await AddStreamer(bigoRoomId, teamId, roomId);
      alert('Streamer added!');
      setBigoRoomId('');
    } catch (error) {
      alert(`Failed: ${error}`);
    }
  };

  return (
    <div className="container">
      <h1>BBapp - Bigo Stream Manager</h1>

      <div className="card">
        <h2>BB-Core Connection</h2>
        <input
          type="text"
          placeholder="STOMP URL"
          value={coreUrl}
          onChange={(e) => setCoreUrl(e.target.value)}
        />
        <button onClick={handleConnect} disabled={connected}>
          {connected ? '✓ Connected' : 'Connect'}
        </button>
      </div>

      {connected && (
        <div className="card">
          <h2>Add Streamer</h2>
          <input
            placeholder="Bigo Room ID"
            value={bigoRoomId}
            onChange={(e) => setBigoRoomId(e.target.value)}
          />
          <input
            placeholder="Team ID (UUID)"
            value={teamId}
            onChange={(e) => setTeamId(e.target.value)}
          />
          <input
            placeholder="Room ID"
            value={roomId}
            onChange={(e) => setRoomId(e.target.value)}
          />
          <button onClick={handleAddStreamer}>Add</button>
        </div>
      )}
    </div>
  );
}

export default App;
```

**Step 2: Add basic styles**

Replace `frontend/src/App.css`:

```css
.container {
  max-width: 800px;
  margin: 0 auto;
  padding: 20px;
}

.card {
  background: white;
  border-radius: 8px;
  padding: 20px;
  margin-bottom: 20px;
  box-shadow: 0 2px 4px rgba(0,0,0,0.1);
}

input {
  width: 100%;
  padding: 10px;
  margin: 10px 0;
  border: 1px solid #ddd;
  border-radius: 4px;
}

button {
  padding: 10px 20px;
  background: #007bff;
  color: white;
  border: none;
  border-radius: 4px;
  cursor: pointer;
  width: 100%;
}

button:hover {
  background: #0056b3;
}

button:disabled {
  background: #28a745;
}
```

**Step 3: Test in dev mode**

```bash
wails dev
```

Expected: UI opens with connection form

**Step 4: Commit**

```bash
git add frontend/
git commit -m "feat: add basic React UI for MVP"
```

---

## Phase 7: Testing & Build

### Task 13: Manual Testing

**Step 1: Start BB-Core**

Ensure BB-Core is running with STOMP enabled

**Step 2: Run BBapp**

```bash
wails dev
```

**Step 3: Test flow**

1. Enter BB-Core STOMP URL
2. Click Connect
3. Enter Bigo Room ID, Team ID, Room ID
4. Click Add Streamer
5. Check logs directory for activity logs
6. Verify BB-Core receives STOMP messages

**Step 4: Check logs**

```bash
cat logs/bbapp_*.jsonl
```

Expected: JSON lines with gift events

**Step 5: Document test results**

Create `TEST_RESULTS.md` with observations

---

### Task 14: Production Build

**Step 1: Build for Windows**

```bash
wails build -clean
```

**Step 2: Test executable**

```bash
./build/bin/bbapp.exe
```

Expected: App launches, all features work

**Step 3: Commit build config**

```bash
git add wails.json build/
git commit -m "build: production build configuration"
```

---

## MVP Complete! 🎉

**What Works:**
✅ Hidden browser management
✅ Bigo WebSocket interception
✅ Gift event parsing
✅ STOMP forwarding to BB-Core
✅ File-based activity logging
✅ Basic connection UI

**Post-MVP Enhancements:**
- Real-time activity feed in UI
- Connection health monitoring
- Chat message support
- Statistics dashboard
- Export functionality
- Auto-reconnect on errors
- Multi-language support

---

## Execution Options

Plan saved to `docs/plans/2025-12-25-bbapp-mvp-wails.md`

**Two execution approaches:**

**1. Subagent-Driven (this session)**
- Use superpowers:subagent-driven-development
- Fresh subagent per task
- Review between tasks
- Fast iteration

**2. Parallel Session (separate)**
- Open new session in worktree
- Use superpowers:executing-plans
- Batch execution with checkpoints

**Which approach do you prefer?**
