# PK Mode with Browser Overlay Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Transform BBapp into scene-based PK platform with login, configuration UI, and OBS overlay integration.

**Architecture:** Remove Quick Start, add authentication layer, implement tab-based scenes (PK Mode + future), embed HTTP server to serve overlay React app, overlay connects to BB-Core STOMP for real-time updates.

**Tech Stack:** Wails v2, React, TypeScript, Go HTTP server, STOMP (sockjs-client + stompjs), BB-Core REST API

**Design Document:** See `docs/plans/2025-12-25-pk-mode-with-overlay-design.md`

---

## Phase 1: Backend Foundation

### Task 1: Create HTTP Overlay Server

**Files:**
- Create: `internal/overlayserver/server.go`
- Create: `internal/overlayserver/port.go`

**Step 1: Create port selection utility**

Create `internal/overlayserver/port.go`:

```go
package overlayserver

import (
	"fmt"
	"net"
)

// findAvailablePort finds an available port in the given range
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
```

**Step 2: Create HTTP server**

Create `internal/overlayserver/server.go`:

```go
package overlayserver

import (
	"fmt"
	"log"
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

func (s *Server) setupRoutes() {
	// Serve React build (frontend/dist)
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

**Step 3: Test build**

Run: `go build ./...`
Expected: Build succeeds

**Step 4: Commit**

```bash
git add internal/overlayserver/
git commit -m "feat(server): add HTTP overlay server with auto-port selection"
```

---

### Task 2: Add Auth Client for BB-Core

**Files:**
- Modify: `internal/api/client.go`
- Modify: `internal/api/types.go`

**Step 1: Add auth types**

Add to `internal/api/types.go`:

```go
// Auth request/response types
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refreshToken"`
}

type AuthResponse struct {
	AccessToken  string   `json:"accessToken"`
	RefreshToken string   `json:"refreshToken"`
	TokenType    string   `json:"tokenType"`
	ExpiresIn    int64    `json:"expiresIn"`
	ExpiresAt    string   `json:"expiresAt"`
	User         UserInfo `json:"user"`
}

type UserInfo struct {
	ID        int64  `json:"id"`
	Username  string `json:"username"`
	Email     string `json:"email"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	RoleCode  string `json:"roleCode"`
}
```

**Step 2: Add auth methods to client**

Add to `internal/api/client.go`:

```go
// Login authenticates with BB-Core
func (c *Client) Login(username, password string) (*AuthResponse, error) {
	url := fmt.Sprintf("%s/api/v1/auth/login", c.baseURL)

	reqBody := LoginRequest{
		Username: username,
		Password: password,
	}
	jsonData, _ := json.Marshal(reqBody)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	var resp AuthResponse
	if err := c.doRequest(req, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// RefreshToken gets a new access token
func (c *Client) RefreshToken(refreshToken string) (*AuthResponse, error) {
	url := fmt.Sprintf("%s/api/v1/auth/refresh-token", c.baseURL)

	reqBody := RefreshTokenRequest{RefreshToken: refreshToken}
	jsonData, _ := json.Marshal(reqBody)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	var resp AuthResponse
	if err := c.doRequest(req, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}
```

**Step 3: Test build**

Run: `go build ./...`
Expected: Build succeeds

**Step 4: Commit**

```bash
git add internal/api/client.go internal/api/types.go
git commit -m "feat(api): add login and refresh token methods"
```

---

### Task 3: Add App Methods for Overlay & Auth

**Files:**
- Modify: `app.go`

**Step 1: Add imports**

Add to imports in `app.go`:

```go
import (
	"bbapp/internal/overlayserver"
	"os"
	"github.com/joho/godotenv"
)
```

**Step 2: Add fields to App struct**

Add to `App` struct:

```go
type App struct {
	// ... existing fields ...
	overlayServer *overlayserver.Server
	bbCoreURL     string
}
```

**Step 3: Update startup to initialize overlay server**

Modify `startup` method in `app.go`:

```go
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	// Load .env file
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found, using defaults")
	}

	// Get BB-Core URL from environment
	bbCoreURL := os.Getenv("BB_CORE_URL")
	if bbCoreURL == "" {
		bbCoreURL = "http://localhost:8080"
	}
	a.bbCoreURL = bbCoreURL

	// Start overlay HTTP server
	server, err := overlayserver.NewServer()
	if err != nil {
		log.Fatal("Failed to start overlay server:", err)
	}
	a.overlayServer = server

	// Start server in background
	go func() {
		if err := server.Start(); err != nil {
			log.Printf("Overlay server error: %v", err)
		}
	}()
}
```

**Step 4: Add new methods**

Add to `app.go`:

```go
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

// GetBBCoreURL returns the configured BB-Core URL
func (a *App) GetBBCoreURL() string {
	return a.bbCoreURL
}
```

**Step 5: Create .env file**

Create `.env` in project root:

```env
BB_CORE_URL=http://localhost:8080
```

**Step 6: Test build**

Run: `wails build`
Expected: Build succeeds, TypeScript bindings generated

**Step 7: Commit**

```bash
git add app.go .env frontend/wailsjs/
git commit -m "feat(app): add overlay server, auth methods, and .env config"
```

---

## Phase 2: Frontend Authentication

### Task 4: Install NPM Dependencies

**Files:**
- Modify: `frontend/package.json`

**Step 1: Install dependencies**

Run:
```bash
npm install --prefix frontend sockjs-client stompjs lucide-react
npm install --prefix frontend --save-dev @types/sockjs-client @types/stompjs
```

**Step 2: Verify package.json updated**

Check `frontend/package.json` contains:
```json
{
  "dependencies": {
    "sockjs-client": "^1.6.1",
    "stompjs": "^2.3.3",
    "lucide-react": "^0.263.1"
  }
}
```

**Step 3: Commit**

```bash
git add frontend/package.json frontend/package-lock.json
git commit -m "deps: add sockjs-client, stompjs, lucide-react"
```

---

### Task 5: Create Shared Types

**Files:**
- Create: `frontend/src/shared/types.ts`

**Step 1: Create types file**

Create `frontend/src/shared/types.ts`:

```typescript
export interface User {
  id: number;
  username: string;
  email: string;
  firstName: string;
  lastName: string;
  roleCode: string;
}

export interface AuthResponse {
  accessToken: string;
  refreshToken: string;
  tokenType: string;
  expiresIn: number;
  expiresAt: string;
  user: User;
}

export interface Team {
  teamId: string;
  name: string;
  avatar?: string;
  bindingGift: string;
  scoreMultipliers?: Record<string, number>;
  streamers: Streamer[];
}

export interface Streamer {
  streamerId: number;
  bigoId: string;
  bigoRoomId: string;
  name: string;
  avatar?: string;
  bindingGift: string;
}

export interface PKConfig {
  roomId: string;
  agencyId?: number;
  teams: Team[];
}

export interface SessionInfo {
  sessionId: string;
  status: string;
  startedAt: number;
  endsAt: number;
}
```

**Step 2: Commit**

```bash
git add frontend/src/shared/types.ts
git commit -m "feat(types): add shared TypeScript types"
```

---

### Task 6: Create LoginPage Component

**Files:**
- Create: `frontend/src/components/LoginPage.tsx`
- Create: `frontend/src/components/LoginPage.css`

**Step 1: Create LoginPage component**

Create `frontend/src/components/LoginPage.tsx`:

```tsx
import React, { useState } from 'react';
import { Login } from '../../wailsjs/go/main/App';
import './LoginPage.css';

interface LoginPageProps {
  onLoginSuccess: (accessToken: string, refreshToken: string, user: any) => void;
}

export const LoginPage: React.FC<LoginPageProps> = ({ onLoginSuccess }) => {
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setLoading(true);

    try {
      const response = await Login(username, password);
      onLoginSuccess(response.accessToken, response.refreshToken, response.user);
    } catch (err: any) {
      setError(err.toString() || 'Login failed. Please check your credentials.');
      setPassword('');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="login-page">
      <div className="login-card">
        <h1>BBapp Login</h1>
        <form onSubmit={handleSubmit}>
          <div className="form-group">
            <label htmlFor="username">Username</label>
            <input
              id="username"
              type="text"
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              placeholder="Enter username"
              disabled={loading}
              required
            />
          </div>
          <div className="form-group">
            <label htmlFor="password">Password</label>
            <input
              id="password"
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              placeholder="Enter password"
              disabled={loading}
              required
            />
          </div>
          {error && <div className="error-message">{error}</div>}
          <button type="submit" disabled={loading}>
            {loading ? 'Logging in...' : 'Login'}
          </button>
        </form>
      </div>
    </div>
  );
};
```

**Step 2: Create styles**

Create `frontend/src/components/LoginPage.css`:

```css
.login-page {
  display: flex;
  justify-content: center;
  align-items: center;
  min-height: 100vh;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
}

.login-card {
  background: white;
  border-radius: 12px;
  padding: 40px;
  box-shadow: 0 10px 40px rgba(0, 0, 0, 0.2);
  width: 100%;
  max-width: 400px;
}

.login-card h1 {
  margin: 0 0 30px 0;
  text-align: center;
  color: #333;
}

.form-group {
  margin-bottom: 20px;
}

.form-group label {
  display: block;
  margin-bottom: 8px;
  font-weight: 500;
  color: #333;
}

.form-group input {
  width: 100%;
  padding: 12px;
  border: 1px solid #ddd;
  border-radius: 6px;
  font-size: 14px;
  box-sizing: border-box;
}

.form-group input:focus {
  outline: none;
  border-color: #667eea;
}

.error-message {
  background: #fee;
  color: #c33;
  padding: 12px;
  border-radius: 6px;
  margin-bottom: 20px;
  font-size: 14px;
}

.login-card button {
  width: 100%;
  padding: 14px;
  background: #667eea;
  color: white;
  border: none;
  border-radius: 6px;
  font-size: 16px;
  font-weight: 600;
  cursor: pointer;
  margin-top: 10px;
}

.login-card button:hover:not(:disabled) {
  background: #5568d3;
}

.login-card button:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}
```

**Step 3: Test build**

Run: `npm run build --prefix frontend`
Expected: Build succeeds

**Step 4: Commit**

```bash
git add frontend/src/components/LoginPage.tsx frontend/src/components/LoginPage.css
git commit -m "feat(ui): add login page component"
```

---

### Task 7: Create SceneTabs Component

**Files:**
- Create: `frontend/src/components/SceneTabs.tsx`
- Create: `frontend/src/components/SceneTabs.css`

**Step 1: Create SceneTabs component**

Create `frontend/src/components/SceneTabs.tsx`:

```tsx
import React from 'react';
import './SceneTabs.css';

interface SceneTabsProps {
  activeScene: string;
  onSceneChange: (scene: string) => void;
  children: React.ReactNode;
}

export const SceneTabs: React.FC<SceneTabsProps> = ({
  activeScene,
  onSceneChange,
  children,
}) => {
  return (
    <div className="scene-container">
      <div className="scene-tabs">
        <button
          className={`scene-tab ${activeScene === 'pk-mode' ? 'active' : ''}`}
          onClick={() => onSceneChange('pk-mode')}
        >
          PK Mode
        </button>
        {/* Future scenes can be added here */}
      </div>
      <div className="scene-content">{children}</div>
    </div>
  );
};
```

**Step 2: Create styles**

Create `frontend/src/components/SceneTabs.css`:

```css
.scene-container {
  display: flex;
  flex-direction: column;
  height: 100%;
}

.scene-tabs {
  display: flex;
  gap: 4px;
  background: #f5f5f5;
  padding: 10px 10px 0 10px;
  border-bottom: 2px solid #ddd;
}

.scene-tab {
  padding: 12px 24px;
  background: transparent;
  border: none;
  border-bottom: 3px solid transparent;
  cursor: pointer;
  font-size: 15px;
  font-weight: 500;
  color: #666;
  border-radius: 6px 6px 0 0;
  transition: all 0.2s;
  width: auto;
  margin: 0;
}

.scene-tab:hover {
  background: #eee;
  color: #333;
}

.scene-tab.active {
  background: white;
  color: #007bff;
  border-bottom-color: #007bff;
}

.scene-content {
  flex: 1;
  overflow-y: auto;
  background: white;
}
```

**Step 3: Commit**

```bash
git add frontend/src/components/SceneTabs.tsx frontend/src/components/SceneTabs.css
git commit -m "feat(ui): add scene tabs component"
```

---

### Task 8: Update App.tsx with Authentication

**Files:**
- Modify: `frontend/src/App.tsx`
- Modify: `frontend/src/App.css`

**Step 1: Read current App.tsx**

Read: `frontend/src/App.tsx`
Note: Understand current structure

**Step 2: Backup and rewrite App.tsx**

Replace entire `frontend/src/App.tsx`:

```tsx
import { useState, useEffect } from 'react';
import { LoginPage } from './components/LoginPage';
import { SceneTabs } from './components/SceneTabs';
import { RefreshToken } from '../wailsjs/go/main/App';
import type { User } from './shared/types';
import './App.css';

function App() {
  const [isAuthenticated, setIsAuthenticated] = useState(false);
  const [user, setUser] = useState<User | null>(null);
  const [accessToken, setAccessToken] = useState('');
  const [refreshToken, setRefreshToken] = useState('');
  const [activeScene, setActiveScene] = useState('pk-mode');
  const [sessionActive, setSessionActive] = useState(false);

  // Token refresh timer
  useEffect(() => {
    if (!isAuthenticated || !refreshToken) return;

    // Refresh token every 50 minutes (assuming 60 min expiry)
    const interval = setInterval(async () => {
      try {
        const response = await RefreshToken(refreshToken);
        setAccessToken(response.accessToken);
        setRefreshToken(response.refreshToken);
        console.log('Token refreshed successfully');
      } catch (error) {
        console.error('Token refresh failed:', error);
        handleLogout();
      }
    }, 50 * 60 * 1000); // 50 minutes

    return () => clearInterval(interval);
  }, [isAuthenticated, refreshToken]);

  const handleLoginSuccess = (
    newAccessToken: string,
    newRefreshToken: string,
    newUser: User
  ) => {
    setAccessToken(newAccessToken);
    setRefreshToken(newRefreshToken);
    setUser(newUser);
    setIsAuthenticated(true);
  };

  const handleLogout = () => {
    setAccessToken('');
    setRefreshToken('');
    setUser(null);
    setIsAuthenticated(false);
    setActiveScene('pk-mode');
    setSessionActive(false);
  };

  const handleSceneChange = async (newScene: string) => {
    if (sessionActive) {
      const confirmed = window.confirm(
        'Active session detected. You must stop the current session before switching scenes.'
      );
      if (!confirmed) return;
      // TODO: Stop session here
      // await StopPKSession('USER_SWITCHED_SCENES');
      setSessionActive(false);
    }
    setActiveScene(newScene);
  };

  if (!isAuthenticated) {
    return <LoginPage onLoginSuccess={handleLoginSuccess} />;
  }

  return (
    <div className="app">
      <header className="app-header">
        <h1>BBapp - PK Session Manager</h1>
        <div className="user-info">
          <span>Welcome, {user?.username}</span>
          <button onClick={handleLogout} className="logout-btn">
            Logout
          </button>
        </div>
      </header>

      <SceneTabs activeScene={activeScene} onSceneChange={handleSceneChange}>
        {activeScene === 'pk-mode' && (
          <div className="scene-placeholder">
            <h2>PK Mode</h2>
            <p>PK Mode UI will be implemented here</p>
            {/* PKModeScene will be added in next phase */}
          </div>
        )}
      </SceneTabs>
    </div>
  );
}

export default App;
```

**Step 3: Update App.css**

Add to `frontend/src/App.css`:

```css
.app {
  display: flex;
  flex-direction: column;
  height: 100vh;
  background: #f5f5f5;
}

.app-header {
  background: #2c3e50;
  color: white;
  padding: 15px 30px;
  display: flex;
  justify-content: space-between;
  align-items: center;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
}

.app-header h1 {
  margin: 0;
  font-size: 20px;
  font-weight: 600;
}

.user-info {
  display: flex;
  align-items: center;
  gap: 15px;
}

.user-info span {
  font-size: 14px;
}

.logout-btn {
  padding: 8px 16px;
  background: #e74c3c;
  color: white;
  border: none;
  border-radius: 4px;
  cursor: pointer;
  font-size: 14px;
  width: auto;
  margin: 0;
}

.logout-btn:hover {
  background: #c0392b;
}

.scene-placeholder {
  padding: 40px;
  text-align: center;
}

.scene-placeholder h2 {
  color: #333;
  margin-bottom: 10px;
}

.scene-placeholder p {
  color: #666;
}
```

**Step 4: Test build**

Run: `wails build`
Expected: Build succeeds

**Step 5: Commit**

```bash
git add frontend/src/App.tsx frontend/src/App.css
git commit -m "feat(app): rewrite App with authentication and scene tabs"
```

---

## Phase 3: PK Mode UI Components

### Task 9: Create PKModeScene Component (Shell)

**Files:**
- Create: `frontend/src/scenes/pk-mode/ui/PKModeScene.tsx`
- Create: `frontend/src/scenes/pk-mode/ui/PKModeScene.css`

**Step 1: Create directory structure**

Run: `mkdir -p frontend/src/scenes/pk-mode/ui`

**Step 2: Create PKModeScene shell**

Create `frontend/src/scenes/pk-mode/ui/PKModeScene.tsx`:

```tsx
import React, { useState } from 'react';
import './PKModeScene.css';

interface PKModeSceneProps {
  accessToken: string;
  onSessionChange: (active: boolean) => void;
}

export const PKModeScene: React.FC<PKModeSceneProps> = ({
  accessToken,
  onSessionChange,
}) => {
  const [roomId, setRoomId] = useState('');
  const [config, setConfig] = useState(null);
  const [configLoaded, setConfigLoaded] = useState(false);
  const [sessionActive, setSessionActive] = useState(false);

  const handleLoadConfig = async () => {
    // TODO: Implement config loading
    alert('Config loading not yet implemented');
  };

  return (
    <div className="pk-mode-scene">
      <div className="card">
        <h2>Load Room Configuration</h2>
        <input
          type="text"
          placeholder="Room ID"
          value={roomId}
          onChange={(e) => setRoomId(e.target.value)}
          disabled={configLoaded}
        />
        <button onClick={handleLoadConfig} disabled={configLoaded}>
          Load Configuration
        </button>
      </div>

      {configLoaded && (
        <div className="card">
          <h2>Configuration</h2>
          <p>Team editor will be added here</p>
        </div>
      )}

      {configLoaded && (
        <div className="card">
          <h2>Session Controls</h2>
          <button disabled={sessionActive}>
            {sessionActive ? 'Session Active' : 'Start Session'}
          </button>
        </div>
      )}
    </div>
  );
};
```

**Step 3: Create styles**

Create `frontend/src/scenes/pk-mode/ui/PKModeScene.css`:

```css
.pk-mode-scene {
  padding: 20px;
  max-width: 1200px;
  margin: 0 auto;
}

.pk-mode-scene .card {
  background: white;
  border-radius: 8px;
  padding: 20px;
  margin-bottom: 20px;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
}

.pk-mode-scene .card h2 {
  margin: 0 0 20px 0;
  color: #333;
}

.pk-mode-scene input {
  width: 100%;
  padding: 12px;
  border: 1px solid #ddd;
  border-radius: 6px;
  margin-bottom: 10px;
  box-sizing: border-box;
}

.pk-mode-scene button {
  padding: 12px 24px;
  background: #007bff;
  color: white;
  border: none;
  border-radius: 6px;
  cursor: pointer;
  font-size: 15px;
  width: auto;
  margin: 0;
}

.pk-mode-scene button:hover:not(:disabled) {
  background: #0056b3;
}

.pk-mode-scene button:disabled {
  background: #ccc;
  cursor: not-allowed;
}
```

**Step 4: Update App.tsx to use PKModeScene**

Modify `frontend/src/App.tsx` - replace the placeholder div:

```tsx
import { PKModeScene } from './scenes/pk-mode/ui/PKModeScene';

// In the return statement, replace:
{activeScene === 'pk-mode' && (
  <PKModeScene
    accessToken={accessToken}
    onSessionChange={setSessionActive}
  />
)}
```

**Step 5: Test build**

Run: `wails build`
Expected: Build succeeds

**Step 6: Commit**

```bash
git add frontend/src/scenes/pk-mode/ frontend/src/App.tsx
git commit -m "feat(pk-mode): add PKModeScene component shell"
```

---

### Task 10: Implement Config Loading

**Files:**
- Modify: `frontend/src/scenes/pk-mode/ui/PKModeScene.tsx`

**Step 1: Add import**

Add to imports:

```tsx
import { GetBBAppConfig, InitializeBBCoreClient, GetBBCoreURL } from '../../../../wailsjs/go/main/App';
import type { PKConfig } from '../../../shared/types';
```

**Step 2: Add state and default template**

Update state:

```tsx
const [config, setConfig] = useState<PKConfig | null>(null);

const generateDefaultTemplate = (roomId: string): PKConfig => ({
  roomId,
  teams: [
    {
      teamId: 'team-1',
      name: 'Team A',
      bindingGift: 'Rose',
      streamers: [
        {
          streamerId: 1,
          bigoId: '',
          bigoRoomId: '',
          name: 'Streamer 1',
          bindingGift: 'Rose',
        },
      ],
    },
    {
      teamId: 'team-2',
      name: 'Team B',
      bindingGift: 'Diamond',
      streamers: [
        {
          streamerId: 2,
          bigoId: '',
          bigoRoomId: '',
          name: 'Streamer 2',
          bindingGift: 'Diamond',
        },
      ],
    },
  ],
});
```

**Step 3: Implement handleLoadConfig**

Replace handleLoadConfig:

```tsx
const handleLoadConfig = async () => {
  if (!roomId.trim()) {
    alert('Please enter a Room ID');
    return;
  }

  try {
    const bbCoreUrl = await GetBBCoreURL();
    await InitializeBBCoreClient(bbCoreUrl, accessToken);

    try {
      const fetchedConfig = await GetBBAppConfig(roomId);
      setConfig(fetchedConfig as PKConfig);
      setConfigLoaded(true);
    } catch (error: any) {
      if (error.toString().includes('404')) {
        // Room not found, load default template
        const defaultConfig = generateDefaultTemplate(roomId);
        setConfig(defaultConfig);
        setConfigLoaded(true);
        alert('Room not found. Loaded default template.');
      } else {
        throw error;
      }
    }
  } catch (error: any) {
    alert(`Failed to load config: ${error.toString()}`);
  }
};
```

**Step 4: Test build**

Run: `wails build`
Expected: Build succeeds

**Step 5: Commit**

```bash
git add frontend/src/scenes/pk-mode/ui/PKModeScene.tsx
git commit -m "feat(pk-mode): implement config loading with default template"
```

---

## Note: Implementation plan continues...

Due to length constraints, this plan covers the foundational setup. The remaining tasks would include:

**Remaining Tasks (To be added):**
- Task 11-15: Team/Streamer editing components (TeamCard, StreamerCard, TeamEditor)
- Task 16: Save configuration functionality
- Task 17-18: Session controls (Start/Stop)
- Task 19: Overlay URL display
- Task 20-22: Overlay components (OverlayApp, OverlayRouter, useSTOMP hook)
- Task 23: PKBattleOverlay component (user's example code)
- Task 24: Integration testing

**Implementation Approach:**

Each remaining task would follow the same TDD-style structure:
1. Create component
2. Add types/interfaces
3. Implement functionality
4. Test build
5. Commit

**Total estimated tasks:** ~24 tasks
**Current progress:** Tasks 1-10 complete (foundation & auth)

---

## Manual Testing Checklist

After all tasks complete:

- [ ] Login with valid credentials succeeds
- [ ] Login with invalid credentials shows error
- [ ] Token auto-refresh works
- [ ] Load existing config works
- [ ] Load non-existent config shows template
- [ ] Edit teams/streamers works
- [ ] Save configuration works
- [ ] Start session generates overlay URL
- [ ] Overlay connects to STOMP
- [ ] Real-time updates display correctly
- [ ] Stop session cleans up
- [ ] Tab switching protection works
- [ ] Logout clears state

---

## Build Commands

```bash
# Development
wails dev

# Production
wails build

# Frontend only
npm run build --prefix frontend
```
