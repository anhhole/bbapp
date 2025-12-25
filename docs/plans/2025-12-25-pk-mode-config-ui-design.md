# PK Mode Configuration UI - Design Document

**Date:** 2025-12-25
**Status:** Approved
**Author:** Claude Code (based on user requirements)

## Overview

Add a "PK Mode" configuration UI to BBapp that allows users to create, edit, and save room configurations directly in the application before starting PK sessions. This provides a hybrid approach: fetch existing configurations from BB-Core OR load a template, edit as needed, save to BB-Core, then start the session.

## Goals

1. Enable users to manage PK room configurations without leaving BBapp
2. Support both editing existing configurations and creating new ones
3. Focus on PK-specific fields: team names, binding gifts, streamer details
4. Provide intuitive card-based UI for managing teams and streamers
5. Maintain separation between configuration management and session lifecycle

## Non-Goals

- Full configuration management (agencyId, scoreMultipliers remain optional/hidden)
- Real-time sync/auto-save (manual save required)
- Advanced features like drag-and-drop or configuration templates library
- Replacing the existing Quick Start workflow

---

## Section 1: Overall Architecture & User Flow

### New UI Structure

The app will have **two modes** accessible via tabs:

1. **Quick Start** (existing): Simple form → Start Session
2. **PK Mode** (new): Configure → Save → Start Session

### PK Mode Workflow

1. User enters Room ID + Auth Token + BB-Core URL
2. Clicks "Load Configuration"
   - If config exists: Fetches from `GET /bbapp-config/{roomId}` and displays it
   - If config doesn't exist (404): Loads default template
   - On error: Shows error message
3. User edits teams/streamers in card-based UI
   - Add/remove teams
   - Add/remove streamers
   - Edit names and binding gifts
4. Clicks "Save Configuration" → `POST /api/v1/stream/rooms/{roomId}/bbapp-config`
5. Success feedback shown
6. "Start Session" button becomes enabled
7. User starts session (uses existing `StartPKSession` flow)

### Default Template Structure

When a room configuration doesn't exist, load this template:

```json
{
  "teams": [
    {
      "teamId": "team-1",
      "name": "Team A",
      "bindingGift": "Rose",
      "streamers": [
        {
          "streamerId": "streamer-1",
          "name": "Streamer 1",
          "bigoRoomId": "",
          "bindingGift": "Rose"
        }
      ]
    },
    {
      "teamId": "team-2",
      "name": "Team B",
      "bindingGift": "Diamond",
      "streamers": [
        {
          "streamerId": "streamer-2",
          "name": "Streamer 2",
          "bigoRoomId": "",
          "bindingGift": "Diamond"
        }
      ]
    }
  ]
}
```

---

## Section 2: UI Components & Layout

### Tab Navigation

- Top-level tabs: "Quick Start" | "PK Mode"
- Current simple form moves to "Quick Start" tab
- New "PK Mode" tab contains configuration UI

### PK Mode Layout (3 main sections)

#### 1. Configuration Loader (top card)

```
┌─────────────────────────────────────────────────┐
│ Load Room Configuration                         │
│ BB-Core URL: [http://localhost:8080________]    │
│ Auth Token:  [**************************]       │
│ Room ID:     [________]  [Load Configuration]   │
│ Status: ✓ Configuration loaded                  │
└─────────────────────────────────────────────────┘
```

#### 2. Teams Editor (scrollable middle section)

```
┌──────────────────────────────────────────────────┐
│ Teams Configuration              [+ Add Team]    │
├──────────────────────────────────────────────────┤
│ ┌─ Team A ────────────────────────────── [×] ───┐│
│ │ Team Name:     [Team A__________________]     ││
│ │ Binding Gift:  [Rose____________________]     ││
│ │                                                ││
│ │ Streamers:                        [+ Add]     ││
│ │ ┌─ Streamer 1 ──────────────────── [×] ─────┐││
│ │ │ Name:         [Alice___________________]  │││
│ │ │ Bigo Room ID: [room123_________________]  │││
│ │ │ Binding Gift: [Rose____________________]  │││
│ │ └───────────────────────────────────────────┘││
│ └──────────────────────────────────────────────┘│
│                                                  │
│ ┌─ Team B ────────────────────────────── [×] ───┐│
│ │ ... similar structure ...                     ││
│ └──────────────────────────────────────────────┘│
└──────────────────────────────────────────────────┘
```

#### 3. Action Buttons (bottom)

```
[Save Configuration] [Start Session]
```

### Visual Design Specifications

- **Team cards:** Light blue background (#e3f2fd), rounded corners, collapsible
- **Streamer cards:** White background, nested inside teams, subtle border
- **Add buttons:** Green (#4caf50), prominent, icon + text
- **Remove buttons:** Red [×] icon (#f44336), top-right corner, small
- **Disabled "Start Session":** Gray until config is saved
- **Loading states:** Spinner + disabled buttons during API calls

---

## Section 3: Data Management & State

### React State Structure

```typescript
interface PKConfig {
  roomId: string;
  agencyId?: number;
  teams: Team[];
}

interface Team {
  teamId: string;
  name: string;
  bindingGift: string;
  streamers: Streamer[];
}

interface Streamer {
  streamerId: string;
  name: string;
  bigoRoomId: string;
  bindingGift: string;
}

// Component state
const [bbCoreUrl, setBbCoreUrl] = useState('http://localhost:8080');
const [authToken, setAuthToken] = useState('');
const [roomId, setRoomId] = useState('');
const [pkConfig, setPkConfig] = useState<PKConfig | null>(null);
const [configLoaded, setConfigLoaded] = useState(false);
const [configSaved, setConfigSaved] = useState(false);
const [isLoading, setIsLoading] = useState(false);
```

### State Management Operations

#### Load Configuration

```typescript
const handleLoadConfig = async () => {
  setIsLoading(true);
  try {
    // Initialize BB-Core client
    await InitializeBBCoreClient(bbCoreUrl, authToken);

    // Try to fetch existing config
    const config = await GetBBAppConfig(roomId);
    setPkConfig(config);
    setConfigLoaded(true);
  } catch (error) {
    if (error.includes('404')) {
      // Load default template
      setPkConfig(generateDefaultTemplate(roomId));
      setConfigLoaded(true);
    } else {
      alert(`Failed to load config: ${error}`);
    }
  } finally {
    setIsLoading(false);
  }
};
```

#### Edit Operations

- **Update team name/gift:** Direct state update via `setPkConfig`
- **Add team:** Generate new `teamId` (e.g., `team-${Date.now()}`), append to teams array
- **Remove team:** Filter out by teamId (validation: min 1 team)
- **Add streamer:** Generate `streamerId`, append to team's streamers array
- **Remove streamer:** Filter out by streamerId (validation: min 1 streamer per team)

#### Save Configuration

```typescript
const handleSaveConfig = async () => {
  // Validate first
  if (!validateConfig(pkConfig)) {
    alert('Please fill in all required fields');
    return;
  }

  setIsLoading(true);
  try {
    await SaveBBAppConfig(roomId, pkConfig);
    setConfigSaved(true);
    alert('Configuration saved successfully!');
  } catch (error) {
    alert(`Failed to save config: ${error}`);
  } finally {
    setIsLoading(false);
  }
};
```

---

## Section 4: API Integration

### New Go Methods in `app.go`

```go
// GetBBAppConfig fetches configuration from BB-Core
func (a *App) GetBBAppConfig(roomId string) (*api.Config, error) {
    if a.apiClient == nil {
        return nil, fmt.Errorf("not connected to BB-Core")
    }
    return a.apiClient.GetConfig(roomId)
}

// SaveBBAppConfig saves configuration to BB-Core
func (a *App) SaveBBAppConfig(roomId string, config api.Config) error {
    if a.apiClient == nil {
        return fmt.Errorf("not connected to BB-Core")
    }
    return a.apiClient.SaveConfig(roomId, config)
}

// InitializeBBCoreClient sets up API client (called before loading config)
func (a *App) InitializeBBCoreClient(bbCoreUrl, authToken string) error {
    a.apiClient = api.NewClient(bbCoreUrl, authToken)
    return nil
}
```

### New Method in `internal/api/client.go`

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

### Frontend Integration

The TypeScript bindings will be auto-generated by Wails:

- `GetBBAppConfig(roomId: string): Promise<Config>`
- `SaveBBAppConfig(roomId: string, config: Config): Promise<void>`
- `InitializeBBCoreClient(bbCoreUrl: string, authToken: string): Promise<void>`

These will be imported from `../wailsjs/go/main/App` and used in the React component.

---

## Section 5: Validation & Error Handling

### Validation Rules

#### Required Fields
- Team name: Cannot be empty
- Binding gift: Cannot be empty
- Streamer name: Cannot be empty
- Bigo Room ID: Cannot be empty

#### Business Rules
- Minimum 1 team required (can't delete last team)
- Minimum 1 streamer per team (can't delete last streamer in a team)
- Bigo Room IDs should be unique across all streamers (warn if duplicate)
- Team names should be unique (warn if duplicate)

#### Validation Timing
- **Real-time:** Show red border on empty required fields as user types
- **On save:** Validate all fields before sending to API
- **Block save:** Disable "Save Configuration" button if validation fails

### Error Handling

#### Load Configuration Errors

| Error Code | User Message |
|------------|--------------|
| 401 Unauthorized | "Invalid authentication token. Please check your token and try again." |
| 404 Not Found | Load default template (not an error - silent fallback) |
| 500 Server Error | "BB-Core error: {error message from server}" |
| Network Error | "Cannot connect to BB-Core at {url}. Please check the URL and try again." |

#### Save Configuration Errors

| Error Code | User Message |
|------------|--------------|
| 400 Bad Request | "Invalid configuration: {error message}. Please check your inputs." |
| 401 Unauthorized | "Authentication expired. Please re-enter your token." |
| 500 Server Error | "Failed to save configuration: {error message}" |
| Network Error | "Cannot connect to BB-Core. Please check your connection." |

### User Feedback

- **Success:** Green toast notification "Configuration saved successfully ✓"
- **Errors:** Red alert box with specific error message
- **Loading states:** Disable buttons, show spinner during API calls
- **Unsaved changes:** Yellow warning banner if user tries to switch tabs with unsaved changes

---

## Section 6: Testing Approach

### Unit Tests (Go)

**File:** `internal/api/client_test.go`

```go
func TestClient_SaveConfig(t *testing.T) {
    // Test successful save
    // Test 400 bad request
    // Test 401 unauthorized
    // Test retry logic on 500 errors
}
```

### Integration Testing

#### Manual Test Cases

**Test 1: Load Existing Configuration**
- **Setup:** Create room in BB-Core with config (2 teams, multiple streamers)
- **Action:** Enter room ID, auth token, click "Load Configuration"
- **Verify:** Config displays correctly in UI with all teams and streamers

**Test 2: Load Non-Existent Configuration**
- **Setup:** Use non-existent room ID
- **Action:** Click "Load Configuration"
- **Verify:** Default template (2 teams, 1 streamer each) loads

**Test 3: Edit and Save Configuration**
- **Action:** Modify team names, binding gifts, streamer details
- **Action:** Click "Save Configuration"
- **Verify:** Success message shown, "Start Session" button enabled
- **Verify:** BB-Core receives POST request with updated config

**Test 4: Add/Remove Teams and Streamers**
- **Action:** Add team, add streamers to teams, remove items
- **Verify:** UI updates correctly, IDs generated properly
- **Verify:** Can't delete last team or last streamer in a team

**Test 5: Validation**
- **Action:** Leave required fields empty, click "Save Configuration"
- **Verify:** Error shown, save blocked
- **Action:** Enter duplicate Bigo Room IDs
- **Verify:** Warning shown

**Test 6: Full Workflow**
- **Flow:** Load config → Edit → Save → Start Session
- **Verify:** Session starts with updated configuration
- **Verify:** Browsers created for all configured streamers

**Test 7: Error Handling**
- **Action:** Use invalid auth token
- **Verify:** Clear error message shown
- **Action:** Enter invalid BB-Core URL
- **Verify:** Network error shown with helpful message

---

## Implementation Notes

### File Changes Required

**New Files:**
- `frontend/src/components/PKMode.tsx` - PK Mode tab component
- `frontend/src/components/TeamCard.tsx` - Team card component
- `frontend/src/components/StreamerCard.tsx` - Streamer card component

**Modified Files:**
- `frontend/src/App.tsx` - Add tab navigation, integrate PKMode component
- `frontend/src/App.css` - Add styles for new components
- `app.go` - Add new methods (GetBBAppConfig, SaveBBAppConfig, InitializeBBCoreClient)
- `internal/api/client.go` - Add SaveConfig method

### Dependencies

No new dependencies required. Uses existing:
- React hooks (useState, useEffect)
- Wails bindings
- Existing API client infrastructure

### Migration Path

This is additive - no breaking changes:
1. Quick Start mode remains unchanged
2. Users can continue using existing workflow
3. PK Mode is optional new feature

---

## Future Enhancements (Out of Scope)

- Drag-and-drop to reorder streamers between teams
- Configuration templates library (save/load presets)
- Import/export configuration as JSON
- Real-time validation of Bigo Room IDs (check if room exists)
- Score multipliers and advanced team settings
- Configuration versioning/history
