# PK Mode Configuration UI Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add a PK Mode configuration UI that allows users to create, edit, and save room configurations directly in BBapp before starting PK sessions.

**Architecture:** Hybrid approach with two-tab UI (Quick Start + PK Mode). Users can load existing configs from BB-Core or use a default template, edit teams/streamers with card-based UI, save to BB-Core, then start session.

**Tech Stack:** React (TypeScript), Wails v2, Go, existing BB-Core REST API

**Design Document:** See `docs/plans/2025-12-25-pk-mode-config-ui-design.md`

---

## Task 1: Add SaveConfig API Method (Backend)

**Files:**
- Test: `internal/api/client_test.go`
- Modify: `internal/api/client.go`

**Step 1: Write the failing test**

Add to `internal/api/client_test.go`:

```go
func TestClient_SaveConfig(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.URL.Path != "/api/v1/stream/rooms/test-room/bbapp-config" {
			t.Errorf("Expected /api/v1/stream/rooms/test-room/bbapp-config, got %s", r.URL.Path)
		}
		if r.Method != "POST" {
			t.Errorf("Expected POST, got %s", r.Method)
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("Expected Bearer test-token, got %s", r.Header.Get("Authorization"))
		}

		// Read and verify body
		var config Config
		if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
			t.Fatalf("Failed to decode body: %v", err)
		}
		if len(config.Teams) != 1 {
			t.Errorf("Expected 1 team, got %d", len(config.Teams))
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success":true}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	config := Config{
		RoomId: "test-room",
		Teams: []Team{
			{
				TeamId:      "team1",
				Name:        "Team A",
				BindingGift: "Rose",
				Streamers:   []Streamer{},
			},
		},
	}

	err := client.SaveConfig("test-room", config)
	if err != nil {
		t.Fatalf("SaveConfig failed: %v", err)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/api -v -run TestClient_SaveConfig`
Expected: FAIL with "undefined: SaveConfig"

**Step 3: Write minimal implementation**

Add to `internal/api/client.go` after the `SendHeartbeat` method:

```go
// SaveConfig saves room configuration to BB-Core
func (c *Client) SaveConfig(roomId string, config Config) error {
	url := fmt.Sprintf("%s/api/v1/stream/rooms/%s/bbapp-config", c.baseURL, roomId)

	jsonData, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.authToken)
	req.Header.Set("Content-Type", "application/json")

	var resp map[string]interface{}
	return c.doRequest(req, &resp)
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/api -v -run TestClient_SaveConfig`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/api/client_test.go internal/api/client.go
git commit -m "feat(api): add SaveConfig method for PK mode configuration"
```

---

## Task 2: Add PK Mode Methods to App

**Files:**
- Modify: `app.go`

**Step 1: Add GetBBAppConfig method**

Add to `app.go` after existing methods:

```go
// GetBBAppConfig fetches configuration from BB-Core
func (a *App) GetBBAppConfig(roomId string) (*api.Config, error) {
	if a.apiClient == nil {
		return nil, fmt.Errorf("not connected to BB-Core")
	}
	return a.apiClient.GetConfig(roomId)
}
```

**Step 2: Add SaveBBAppConfig method**

Add to `app.go`:

```go
// SaveBBAppConfig saves configuration to BB-Core
func (a *App) SaveBBAppConfig(roomId string, config api.Config) error {
	if a.apiClient == nil {
		return fmt.Errorf("not connected to BB-Core")
	}
	return a.apiClient.SaveConfig(roomId, config)
}
```

**Step 3: Add InitializeBBCoreClient method**

Add to `app.go`:

```go
// InitializeBBCoreClient sets up API client (called before loading config)
func (a *App) InitializeBBCoreClient(bbCoreUrl, authToken string) error {
	a.apiClient = api.NewClient(bbCoreUrl, authToken)
	return nil
}
```

**Step 4: Test build**

Run: `wails build`
Expected: Build succeeds, TypeScript bindings generated

**Step 5: Commit**

```bash
git add app.go frontend/wailsjs/go/main/App.js frontend/wailsjs/go/main/App.d.ts
git commit -m "feat(app): add PK mode configuration methods"
```

---

## Task 3: Create StreamerCard Component

**Files:**
- Create: `frontend/src/components/StreamerCard.tsx`

**Step 1: Create StreamerCard component**

Create `frontend/src/components/StreamerCard.tsx`:

```tsx
import React from 'react';

interface Streamer {
  streamerId: string;
  name: string;
  bigoRoomId: string;
  bindingGift: string;
}

interface StreamerCardProps {
  streamer: Streamer;
  onUpdate: (streamer: Streamer) => void;
  onRemove: () => void;
  canRemove: boolean;
}

export const StreamerCard: React.FC<StreamerCardProps> = ({
  streamer,
  onUpdate,
  onRemove,
  canRemove,
}) => {
  const handleChange = (field: keyof Streamer, value: string) => {
    onUpdate({ ...streamer, [field]: value });
  };

  return (
    <div className="streamer-card">
      <div className="card-header">
        <span className="card-title">Streamer</span>
        {canRemove && (
          <button className="remove-btn" onClick={onRemove} title="Remove streamer">
            ×
          </button>
        )}
      </div>
      <div className="card-body">
        <label>
          Name:
          <input
            type="text"
            value={streamer.name}
            onChange={(e) => handleChange('name', e.target.value)}
            placeholder="Streamer name"
            required
          />
        </label>
        <label>
          Bigo Room ID:
          <input
            type="text"
            value={streamer.bigoRoomId}
            onChange={(e) => handleChange('bigoRoomId', e.target.value)}
            placeholder="e.g., room123"
            required
          />
        </label>
        <label>
          Binding Gift:
          <input
            type="text"
            value={streamer.bindingGift}
            onChange={(e) => handleChange('bindingGift', e.target.value)}
            placeholder="e.g., Rose"
            required
          />
        </label>
      </div>
    </div>
  );
};
```

**Step 2: Test build**

Run: `npm run build --prefix frontend`
Expected: Build succeeds

**Step 3: Commit**

```bash
git add frontend/src/components/StreamerCard.tsx
git commit -m "feat(ui): add StreamerCard component for PK mode"
```

---

## Task 4: Create TeamCard Component

**Files:**
- Create: `frontend/src/components/TeamCard.tsx`

**Step 1: Create TeamCard component**

Create `frontend/src/components/TeamCard.tsx`:

```tsx
import React from 'react';
import { StreamerCard } from './StreamerCard';

interface Streamer {
  streamerId: string;
  name: string;
  bigoRoomId: string;
  bindingGift: string;
}

interface Team {
  teamId: string;
  name: string;
  bindingGift: string;
  streamers: Streamer[];
}

interface TeamCardProps {
  team: Team;
  onUpdate: (team: Team) => void;
  onRemove: () => void;
  canRemove: boolean;
}

export const TeamCard: React.FC<TeamCardProps> = ({
  team,
  onUpdate,
  onRemove,
  canRemove,
}) => {
  const handleTeamChange = (field: keyof Team, value: string) => {
    onUpdate({ ...team, [field]: value });
  };

  const handleAddStreamer = () => {
    const newStreamer: Streamer = {
      streamerId: `streamer-${Date.now()}`,
      name: `Streamer ${team.streamers.length + 1}`,
      bigoRoomId: '',
      bindingGift: team.bindingGift,
    };
    onUpdate({ ...team, streamers: [...team.streamers, newStreamer] });
  };

  const handleUpdateStreamer = (index: number, updatedStreamer: Streamer) => {
    const newStreamers = [...team.streamers];
    newStreamers[index] = updatedStreamer;
    onUpdate({ ...team, streamers: newStreamers });
  };

  const handleRemoveStreamer = (index: number) => {
    if (team.streamers.length <= 1) {
      alert('Each team must have at least one streamer');
      return;
    }
    const newStreamers = team.streamers.filter((_, i) => i !== index);
    onUpdate({ ...team, streamers: newStreamers });
  };

  return (
    <div className="team-card">
      <div className="card-header">
        <span className="card-title">Team</span>
        {canRemove && (
          <button className="remove-btn" onClick={onRemove} title="Remove team">
            ×
          </button>
        )}
      </div>
      <div className="card-body">
        <label>
          Team Name:
          <input
            type="text"
            value={team.name}
            onChange={(e) => handleTeamChange('name', e.target.value)}
            placeholder="Team name"
            required
          />
        </label>
        <label>
          Binding Gift:
          <input
            type="text"
            value={team.bindingGift}
            onChange={(e) => handleTeamChange('bindingGift', e.target.value)}
            placeholder="e.g., Rose"
            required
          />
        </label>

        <div className="streamers-section">
          <div className="section-header">
            <h4>Streamers</h4>
            <button className="add-btn" onClick={handleAddStreamer}>
              + Add Streamer
            </button>
          </div>
          <div className="streamers-list">
            {team.streamers.map((streamer, index) => (
              <StreamerCard
                key={streamer.streamerId}
                streamer={streamer}
                onUpdate={(updated) => handleUpdateStreamer(index, updated)}
                onRemove={() => handleRemoveStreamer(index)}
                canRemove={team.streamers.length > 1}
              />
            ))}
          </div>
        </div>
      </div>
    </div>
  );
};
```

**Step 2: Test build**

Run: `npm run build --prefix frontend`
Expected: Build succeeds

**Step 3: Commit**

```bash
git add frontend/src/components/TeamCard.tsx
git commit -m "feat(ui): add TeamCard component for PK mode"
```

---

## Task 5: Create PKMode Component

**Files:**
- Create: `frontend/src/components/PKMode.tsx`

**Step 1: Create PKMode component (part 1 - setup and state)**

Create `frontend/src/components/PKMode.tsx`:

```tsx
import React, { useState } from 'react';
import { TeamCard } from './TeamCard';
import {
  InitializeBBCoreClient,
  GetBBAppConfig,
  SaveBBAppConfig,
} from '../../wailsjs/go/main/App';

interface Streamer {
  streamerId: string;
  name: string;
  bigoRoomId: string;
  bindingGift: string;
}

interface Team {
  teamId: string;
  name: string;
  bindingGift: string;
  streamers: Streamer[];
}

interface PKConfig {
  roomId: string;
  agencyId?: number;
  teams: Team[];
}

const generateDefaultTemplate = (roomId: string): PKConfig => ({
  roomId,
  teams: [
    {
      teamId: 'team-1',
      name: 'Team A',
      bindingGift: 'Rose',
      streamers: [
        {
          streamerId: 'streamer-1',
          name: 'Streamer 1',
          bigoRoomId: '',
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
          streamerId: 'streamer-2',
          name: 'Streamer 2',
          bigoRoomId: '',
          bindingGift: 'Diamond',
        },
      ],
    },
  ],
});

export const PKMode: React.FC = () => {
  const [bbCoreUrl, setBbCoreUrl] = useState('http://localhost:8080');
  const [authToken, setAuthToken] = useState('');
  const [roomId, setRoomId] = useState('');
  const [pkConfig, setPkConfig] = useState<PKConfig | null>(null);
  const [configLoaded, setConfigLoaded] = useState(false);
  const [configSaved, setConfigSaved] = useState(false);
  const [isLoading, setIsLoading] = useState(false);
```

**Step 2: Create PKMode component (part 2 - handlers)**

Continue in `frontend/src/components/PKMode.tsx`:

```tsx
  const handleLoadConfig = async () => {
    if (!roomId || !authToken) {
      alert('Please enter BB-Core URL, Auth Token, and Room ID');
      return;
    }

    setIsLoading(true);
    try {
      // Initialize BB-Core client
      await InitializeBBCoreClient(bbCoreUrl, authToken);

      // Try to fetch existing config
      const config = await GetBBAppConfig(roomId);
      setPkConfig(config as PKConfig);
      setConfigLoaded(true);
      setConfigSaved(false);
    } catch (error: any) {
      if (error.toString().includes('404')) {
        // Load default template
        setPkConfig(generateDefaultTemplate(roomId));
        setConfigLoaded(true);
        setConfigSaved(false);
      } else {
        alert(`Failed to load config: ${error}`);
      }
    } finally {
      setIsLoading(false);
    }
  };

  const validateConfig = (config: PKConfig | null): boolean => {
    if (!config || config.teams.length === 0) {
      return false;
    }

    for (const team of config.teams) {
      if (!team.name.trim() || !team.bindingGift.trim()) {
        return false;
      }
      if (team.streamers.length === 0) {
        return false;
      }
      for (const streamer of team.streamers) {
        if (!streamer.name.trim() || !streamer.bigoRoomId.trim() || !streamer.bindingGift.trim()) {
          return false;
        }
      }
    }

    return true;
  };

  const handleSaveConfig = async () => {
    if (!pkConfig) {
      alert('No configuration to save');
      return;
    }

    if (!validateConfig(pkConfig)) {
      alert('Please fill in all required fields (team names, binding gifts, streamer details)');
      return;
    }

    setIsLoading(true);
    try {
      await SaveBBAppConfig(roomId, pkConfig as any);
      setConfigSaved(true);
      alert('Configuration saved successfully!');
    } catch (error) {
      alert(`Failed to save config: ${error}`);
    } finally {
      setIsLoading(false);
    }
  };

  const handleAddTeam = () => {
    if (!pkConfig) return;

    const newTeam: Team = {
      teamId: `team-${Date.now()}`,
      name: `Team ${pkConfig.teams.length + 1}`,
      bindingGift: 'Rose',
      streamers: [
        {
          streamerId: `streamer-${Date.now()}`,
          name: 'Streamer 1',
          bigoRoomId: '',
          bindingGift: 'Rose',
        },
      ],
    };

    setPkConfig({ ...pkConfig, teams: [...pkConfig.teams, newTeam] });
    setConfigSaved(false);
  };

  const handleUpdateTeam = (index: number, updatedTeam: Team) => {
    if (!pkConfig) return;
    const newTeams = [...pkConfig.teams];
    newTeams[index] = updatedTeam;
    setPkConfig({ ...pkConfig, teams: newTeams });
    setConfigSaved(false);
  };

  const handleRemoveTeam = (index: number) => {
    if (!pkConfig) return;
    if (pkConfig.teams.length <= 1) {
      alert('Cannot remove the last team. At least one team is required.');
      return;
    }
    const newTeams = pkConfig.teams.filter((_, i) => i !== index);
    setPkConfig({ ...pkConfig, teams: newTeams });
    setConfigSaved(false);
  };
```

**Step 3: Create PKMode component (part 3 - render)**

Continue in `frontend/src/components/PKMode.tsx`:

```tsx
  return (
    <div className="pk-mode">
      {/* Configuration Loader */}
      <div className="card">
        <h2>Load Room Configuration</h2>
        <input
          type="text"
          placeholder="BB-Core URL (e.g., http://localhost:8080)"
          value={bbCoreUrl}
          onChange={(e) => setBbCoreUrl(e.target.value)}
          disabled={configLoaded}
        />
        <input
          type="password"
          placeholder="Authentication Token"
          value={authToken}
          onChange={(e) => setAuthToken(e.target.value)}
          disabled={configLoaded}
        />
        <input
          type="text"
          placeholder="Room ID"
          value={roomId}
          onChange={(e) => setRoomId(e.target.value)}
          disabled={configLoaded}
        />
        {!configLoaded ? (
          <button onClick={handleLoadConfig} disabled={isLoading}>
            {isLoading ? 'Loading...' : 'Load Configuration'}
          </button>
        ) : (
          <div className="config-status">
            <span className="status-indicator success">✓ Configuration loaded</span>
            <button
              onClick={() => {
                setConfigLoaded(false);
                setPkConfig(null);
                setConfigSaved(false);
              }}
              className="secondary-btn"
            >
              Load Different Room
            </button>
          </div>
        )}
      </div>

      {/* Teams Editor */}
      {configLoaded && pkConfig && (
        <>
          <div className="card">
            <div className="section-header">
              <h2>Teams Configuration</h2>
              <button className="add-btn" onClick={handleAddTeam}>
                + Add Team
              </button>
            </div>
            <div className="teams-list">
              {pkConfig.teams.map((team, index) => (
                <TeamCard
                  key={team.teamId}
                  team={team}
                  onUpdate={(updated) => handleUpdateTeam(index, updated)}
                  onRemove={() => handleRemoveTeam(index)}
                  canRemove={pkConfig.teams.length > 1}
                />
              ))}
            </div>
          </div>

          {/* Action Buttons */}
          <div className="card">
            <div className="action-buttons">
              <button
                onClick={handleSaveConfig}
                disabled={isLoading || configSaved}
                className="primary-btn"
              >
                {isLoading ? 'Saving...' : configSaved ? 'Configuration Saved ✓' : 'Save Configuration'}
              </button>
              <button
                disabled={!configSaved}
                className="start-session-btn"
                onClick={() => alert('Start Session functionality coming soon')}
              >
                Start Session
              </button>
            </div>
            {!configSaved && (
              <p className="warning-text">
                ⚠️ Save configuration before starting session
              </p>
            )}
          </div>
        </>
      )}
    </div>
  );
};
```

**Step 4: Close the component**

Add closing brace to `frontend/src/components/PKMode.tsx`:

```tsx
};
```

**Step 5: Test build**

Run: `npm run build --prefix frontend`
Expected: Build succeeds

**Step 6: Commit**

```bash
git add frontend/src/components/PKMode.tsx
git commit -m "feat(ui): add PKMode component with config management"
```

---

## Task 6: Update App.tsx with Tab Navigation

**Files:**
- Modify: `frontend/src/App.tsx`

**Step 1: Read current App.tsx**

Read: `frontend/src/App.tsx`
Note: Understand current structure

**Step 2: Add imports and state**

At top of `frontend/src/App.tsx`, add import:

```tsx
import { PKMode } from './components/PKMode';
```

Add state for tab navigation after existing useState declarations:

```tsx
const [activeTab, setActiveTab] = useState<'quick' | 'pk'>('quick');
```

**Step 3: Replace return statement with tabbed UI**

Replace the entire return statement in `frontend/src/App.tsx`:

```tsx
return (
  <div className="container">
    <h1>BBapp - PK Session Manager</h1>

    {/* Tab Navigation */}
    <div className="tabs">
      <button
        className={`tab ${activeTab === 'quick' ? 'active' : ''}`}
        onClick={() => setActiveTab('quick')}
      >
        Quick Start
      </button>
      <button
        className={`tab ${activeTab === 'pk' ? 'active' : ''}`}
        onClick={() => setActiveTab('pk')}
      >
        PK Mode
      </button>
    </div>

    {/* Tab Content */}
    {activeTab === 'quick' ? (
      // Quick Start Mode (existing functionality)
      <>
        {!sessionActive ? (
          <div className="card">
            <h2>Start PK Session</h2>
            <input
              type="text"
              placeholder="BB-Core URL (e.g., http://localhost:8080)"
              value={bbCoreUrl}
              onChange={(e) => setBbCoreUrl(e.target.value)}
            />
            <input
              type="password"
              placeholder="Authentication Token (required)"
              value={authToken}
              onChange={(e) => setAuthToken(e.target.value)}
            />
            <input
              type="text"
              placeholder="Room ID"
              value={roomId}
              onChange={(e) => setRoomId(e.target.value)}
            />
            <button onClick={handleStartSession}>Start Session</button>
          </div>
        ) : (
          <>
            <div className="card">
              <h2>Session Active</h2>
              <p><strong>Room ID:</strong> {sessionStatus?.roomId || roomId}</p>
              <p><strong>Session ID:</strong> {sessionStatus?.sessionId || 'Loading...'}</p>
              <button onClick={handleStopSession} className="stop-button">
                Stop Session
              </button>
            </div>

            {sessionStatus && sessionStatus.connections && sessionStatus.connections.length > 0 && (
              <div className="card">
                <h2>Active Connections</h2>
                <div className="connections">
                  {sessionStatus.connections.map((conn) => (
                    <div key={conn.bigoRoomId} className="connection-item">
                      <div className="connection-header">
                        <span className="streamer-id">{conn.streamerId}</span>
                        <span className={`status status-${conn.status.toLowerCase()}`}>
                          {conn.status}
                        </span>
                      </div>
                      <div className="connection-details">
                        <span>Bigo Room: {conn.bigoRoomId}</span>
                        <span>Messages: {conn.messagesReceived}</span>
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            )}
          </>
        )}
      </>
    ) : (
      // PK Mode
      <PKMode />
    )}
  </div>
);
```

**Step 4: Test build**

Run: `wails build`
Expected: Build succeeds

**Step 5: Commit**

```bash
git add frontend/src/App.tsx
git commit -m "feat(ui): add tab navigation for Quick Start and PK Mode"
```

---

## Task 7: Update App.css with PK Mode Styles

**Files:**
- Modify: `frontend/src/App.css`

**Step 1: Add tab navigation styles**

Add to end of `frontend/src/App.css`:

```css
/* Tab Navigation */
.tabs {
  display: flex;
  gap: 10px;
  margin-bottom: 20px;
  border-bottom: 2px solid #e0e0e0;
}

.tab {
  padding: 12px 24px;
  background: transparent;
  border: none;
  border-bottom: 3px solid transparent;
  cursor: pointer;
  font-size: 16px;
  font-weight: 500;
  color: #666;
  width: auto;
  margin: 0;
}

.tab:hover {
  background: #f5f5f5;
  color: #333;
}

.tab.active {
  color: #007bff;
  border-bottom-color: #007bff;
  background: transparent;
}

/* PK Mode */
.pk-mode {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.config-status {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  margin-top: 10px;
}

.status-indicator {
  padding: 8px 16px;
  border-radius: 4px;
  font-weight: 500;
}

.status-indicator.success {
  background: #d4edda;
  color: #155724;
}

.secondary-btn {
  background: #6c757d;
  width: auto;
  padding: 8px 16px;
  margin: 0;
}

.secondary-btn:hover {
  background: #5a6268;
}

/* Section Header */
.section-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 15px;
}

.section-header h2,
.section-header h4 {
  margin: 0;
}

/* Teams and Streamers Lists */
.teams-list,
.streamers-list {
  display: flex;
  flex-direction: column;
  gap: 15px;
}

.streamers-section {
  margin-top: 20px;
  padding-top: 20px;
  border-top: 1px solid #e0e0e0;
}

/* Team Card */
.team-card {
  background: #e3f2fd;
  border-radius: 8px;
  padding: 20px;
  position: relative;
}

/* Streamer Card */
.streamer-card {
  background: white;
  border: 1px solid #ddd;
  border-radius: 6px;
  padding: 15px;
  position: relative;
}

/* Card Common */
.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 15px;
}

.card-title {
  font-weight: bold;
  font-size: 14px;
  color: #666;
  text-transform: uppercase;
}

.card-body {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.card-body label {
  display: flex;
  flex-direction: column;
  gap: 5px;
  font-size: 14px;
  font-weight: 500;
  color: #333;
}

.card-body input {
  margin: 0;
}

/* Buttons */
.add-btn {
  background: #4caf50;
  color: white;
  border: none;
  padding: 8px 16px;
  border-radius: 4px;
  cursor: pointer;
  font-size: 14px;
  font-weight: 500;
  width: auto;
  margin: 0;
}

.add-btn:hover {
  background: #45a049;
}

.remove-btn {
  background: #f44336;
  color: white;
  border: none;
  width: 28px;
  height: 28px;
  border-radius: 50%;
  cursor: pointer;
  font-size: 20px;
  line-height: 1;
  padding: 0;
  margin: 0;
  display: flex;
  align-items: center;
  justify-content: center;
}

.remove-btn:hover {
  background: #da190b;
}

.primary-btn {
  background: #007bff;
  color: white;
}

.primary-btn:hover {
  background: #0056b3;
}

.primary-btn:disabled {
  background: #6c757d;
  cursor: not-allowed;
}

.start-session-btn {
  background: #28a745;
}

.start-session-btn:hover {
  background: #218838;
}

.start-session-btn:disabled {
  background: #6c757d;
  cursor: not-allowed;
}

/* Action Buttons */
.action-buttons {
  display: flex;
  gap: 10px;
}

.warning-text {
  margin-top: 10px;
  color: #856404;
  background: #fff3cd;
  padding: 10px;
  border-radius: 4px;
  font-size: 14px;
}
```

**Step 2: Test build**

Run: `wails build`
Expected: Build succeeds

**Step 3: Commit**

```bash
git add frontend/src/App.css
git commit -m "style: add PK mode UI styles"
```

---

## Task 8: Manual Testing

**Files:**
- Reference: `docs/plans/2025-12-25-pk-mode-config-ui-design.md` (Section 6)

**Step 1: Build and run the app**

Run: `wails build`
Run: `build/bin/bbapp.exe`

**Step 2: Test Load Existing Configuration**

1. Switch to "PK Mode" tab
2. Enter BB-Core URL, auth token, and existing room ID
3. Click "Load Configuration"
4. Verify: Configuration displays with all teams/streamers

**Step 3: Test Load Non-Existent Configuration**

1. Use non-existent room ID
2. Click "Load Configuration"
3. Verify: Default template loads (2 teams, 1 streamer each)

**Step 4: Test Add/Remove Operations**

1. Click "+ Add Team"
2. Verify: New team added with default values
3. Click "+ Add Streamer" within a team
4. Verify: New streamer added
5. Try to remove last team
6. Verify: Blocked with error message
7. Try to remove last streamer in a team
8. Verify: Blocked with error message

**Step 5: Test Edit and Validation**

1. Clear a required field (e.g., team name)
2. Click "Save Configuration"
3. Verify: Error shown, save blocked
4. Fill in all required fields
5. Click "Save Configuration"
6. Verify: Success message, "Start Session" enabled

**Step 6: Test Save Configuration**

1. Edit team names, binding gifts, streamer details
2. Click "Save Configuration"
3. Verify: Success message shown
4. Verify: BB-Core logs show POST to `/api/v1/stream/rooms/{roomId}/bbapp-config`

**Step 7: Document results**

Create checklist:
- [ ] Load existing config works
- [ ] Load non-existent config loads template
- [ ] Add/remove teams works
- [ ] Add/remove streamers works
- [ ] Validation blocks save on empty fields
- [ ] Save configuration succeeds
- [ ] UI is responsive and intuitive

---

## Success Criteria

1. ✅ All tests pass (`go test ./...`)
2. ✅ Build succeeds (`wails build`)
3. ✅ Can load existing configuration from BB-Core
4. ✅ Can create new configuration with default template
5. ✅ Can add/remove teams and streamers with validation
6. ✅ Can save configuration to BB-Core
7. ✅ UI is intuitive and visually consistent
8. ✅ All required fields validated before save

---

## Notes

- Keep Quick Start mode unchanged for users who prefer simple workflow
- PK Mode provides advanced configuration for power users
- Validation ensures data integrity before sending to BB-Core
- Future: Connect "Start Session" button to existing StartPKSession flow
