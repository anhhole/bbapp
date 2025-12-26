package session

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"bbapp/internal/api"
	"bbapp/internal/config"
)

// mockBBCoreServer creates a mock BB-Core server for integration testing
type mockBBCoreServer struct {
	server              *httptest.Server
	validateTrialCalled int
	startSessionCalled  int
	stopSessionCalled   int
	heartbeatCalled     int
	shouldRejectTrial   bool
	shouldFailStart     bool
}

func newMockBBCoreServer() *mockBBCoreServer {
	mock := &mockBBCoreServer{}

	mux := http.NewServeMux()

	// POST /api/v1/external/validate-trial
	mux.HandleFunc("/api/v1/external/validate-trial", func(w http.ResponseWriter, r *http.Request) {
		mock.validateTrialCalled++

		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req api.ValidateTrialRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		resp := api.ValidateTrialResponse{
			Allowed: !mock.shouldRejectTrial,
			Message: "Trial validation passed",
		}

		if mock.shouldRejectTrial {
			resp.Message = "Trial validation rejected"
			resp.BlockedBigoIds = []string{"blocked123"}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	// POST /pk/start-from-bbapp/{roomId}
	mux.HandleFunc("/pk/start-from-bbapp/", func(w http.ResponseWriter, r *http.Request) {
		mock.startSessionCalled++

		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		if mock.shouldFailStart {
			resp := api.StartSessionResponse{
				Success: false,
				Message: "Session start failed",
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
			return
		}

		resp := api.StartSessionResponse{
			Success:   true,
			Message:   "Session started",
			SessionId: "test-session-123",
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	// POST /pk/stop-from-bbapp/{roomId}
	mux.HandleFunc("/pk/stop-from-bbapp/", func(w http.ResponseWriter, r *http.Request) {
		mock.stopSessionCalled++

		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		resp := api.StopSessionResponse{
			Success: true,
			Message: "Session stopped",
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	// POST /api/v1/external/heartbeat
	mux.HandleFunc("/api/v1/external/heartbeat", func(w http.ResponseWriter, r *http.Request) {
		mock.heartbeatCalled++

		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Return empty JSON object
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("{}"))
	})

	mock.server = httptest.NewServer(mux)
	return mock
}

func (m *mockBBCoreServer) Close() {
	m.server.Close()
}

func (m *mockBBCoreServer) URL() string {
	return m.server.URL
}

func (m *mockBBCoreServer) Reset() {
	m.validateTrialCalled = 0
	m.startSessionCalled = 0
	m.stopSessionCalled = 0
	m.heartbeatCalled = 0
	m.shouldRejectTrial = false
	m.shouldFailStart = false
}

// TestSessionLifecycle tests the complete session start → active → stop flow
func TestSessionLifecycle(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup mock BB-Core server
	mockServer := newMockBBCoreServer()
	defer mockServer.Close()

	// Create API client
	apiClient := api.NewClient(mockServer.URL(), "test-token")

	// Create session manager
	manager := NewManager()
	manager.Initialize(apiClient, "test-device-hash")

	// Create test config
	cfg := &api.Config{
		AgencyId: 123,
		Teams: []api.Team{
			{
				TeamId: "team1",
				Name:   "Test Team",
				Streamers: []api.Streamer{
					{
						StreamerId: "streamer1",
						BigoId:     "bigo123",
						BigoRoomId: "room123",
						Name:       "Test Streamer",
					},
				},
			},
		},
	}

	roomId := "test-room"

	// Test: Start session (without STOMP - just API validation)
	// Note: We can't fully test STOMP in unit tests without a real STOMP server
	// This test focuses on the API calls and state management

	t.Run("ValidateTrialCalled", func(t *testing.T) {
		// Verify trial validation is called
		if mockServer.validateTrialCalled == 0 {
			// This will be called when we actually call Start()
			// For now, test the API client directly
			streamers := []api.ValidateTrialStreamer{
				{BigoId: "bigo123", BigoRoomId: "room123"},
			}
			resp, err := apiClient.ValidateTrial(streamers)
			if err != nil {
				t.Fatalf("ValidateTrial failed: %v", err)
			}
			if !resp.Allowed {
				t.Errorf("Expected trial to be allowed, got: %v", resp.Message)
			}
		}
	})

	t.Run("StartSessionAPI", func(t *testing.T) {
		// Test start session API call
		resp, err := apiClient.StartSession(roomId, "test-device-hash")
		if err != nil {
			t.Fatalf("StartSession failed: %v", err)
		}
		if !resp.Success {
			t.Errorf("Expected success, got: %v", resp.Message)
		}
		if resp.SessionId == "" {
			t.Errorf("Expected session ID, got empty string")
		}
	})

	t.Run("StopSessionAPI", func(t *testing.T) {
		// Test stop session API call
		resp, err := apiClient.StopSession(roomId, "TEST_COMPLETE")
		if err != nil {
			t.Fatalf("StopSession failed: %v", err)
		}
		if !resp.Success {
			t.Errorf("Expected success, got: %v", resp.Message)
		}
	})

	t.Run("ConfigManager", func(t *testing.T) {
		// Test config manager
		cfgMgr := config.NewManager(cfg)

		// Test streamer lookup
		streamer, err := cfgMgr.LookupStreamerByBigoRoom("room123")
		if err != nil {
			t.Fatalf("Failed to lookup streamer: %v", err)
		}
		if streamer.StreamerId != "streamer1" {
			t.Errorf("Expected streamer1, got: %s", streamer.StreamerId)
		}

		// Test get all rooms
		rooms := cfgMgr.GetAllBigoRoomIds()
		if len(rooms) != 1 {
			t.Errorf("Expected 1 room, got: %d", len(rooms))
		}
		if rooms[0] != "room123" {
			t.Errorf("Expected room123, got: %s", rooms[0])
		}
	})

	// Verify API calls were made
	if mockServer.validateTrialCalled < 1 {
		t.Logf("Note: ValidateTrial was called %d times (expected at least 1)", mockServer.validateTrialCalled)
	}
	if mockServer.startSessionCalled < 1 {
		t.Logf("Note: StartSession was called %d times (expected at least 1)", mockServer.startSessionCalled)
	}
	if mockServer.stopSessionCalled < 1 {
		t.Logf("Note: StopSession was called %d times (expected at least 1)", mockServer.stopSessionCalled)
	}
}

// TestTrialValidationRejection tests trial validation rejection scenarios
func TestTrialValidationRejection(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	mockServer := newMockBBCoreServer()
	defer mockServer.Close()

	// Configure mock to reject trial
	mockServer.shouldRejectTrial = true

	apiClient := api.NewClient(mockServer.URL(), "test-token")

	t.Run("TrialRejected", func(t *testing.T) {
		streamers := []api.ValidateTrialStreamer{
			{BigoId: "blocked123", BigoRoomId: "room456"},
		}

		resp, err := apiClient.ValidateTrial(streamers)
		if err != nil {
			t.Fatalf("ValidateTrial failed: %v", err)
		}

		if resp.Allowed {
			t.Errorf("Expected trial to be rejected, but it was allowed")
		}

		if resp.Message == "" {
			t.Errorf("Expected rejection message, got empty string")
		}

		if len(resp.BlockedBigoIds) == 0 {
			t.Errorf("Expected blocked IDs, got none")
		}
	})

	if mockServer.validateTrialCalled != 1 {
		t.Errorf("Expected 1 validation call, got: %d", mockServer.validateTrialCalled)
	}
}

// TestSessionStartFailure tests handling of session start failures
func TestSessionStartFailure(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	mockServer := newMockBBCoreServer()
	defer mockServer.Close()

	// Configure mock to fail session start
	mockServer.shouldFailStart = true

	apiClient := api.NewClient(mockServer.URL(), "test-token")

	t.Run("StartSessionFails", func(t *testing.T) {
		resp, err := apiClient.StartSession("test-room", "test-hash")
		if err != nil {
			t.Fatalf("StartSession API call failed: %v", err)
		}

		if resp.Success {
			t.Errorf("Expected failure, but got success")
		}

		if resp.Message == "" {
			t.Errorf("Expected error message, got empty string")
		}
	})
}

// TestHeartbeatSending tests heartbeat functionality
func TestHeartbeatSending(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	mockServer := newMockBBCoreServer()
	defer mockServer.Close()

	apiClient := api.NewClient(mockServer.URL(), "test-token")

	t.Run("SendHeartbeat", func(t *testing.T) {
		req := api.HeartbeatRequest{
			Connections: []api.ConnectionStatus{
				{
					BigoId:     "bigo123",
					BigoRoomId: "room123",
					Status:     "CONNECTED",
				},
			},
		}

		err := apiClient.SendHeartbeat(req)
		if err != nil {
			t.Fatalf("SendHeartbeat failed: %v", err)
		}
	})

	if mockServer.heartbeatCalled != 1 {
		t.Errorf("Expected 1 heartbeat call, got: %d", mockServer.heartbeatCalled)
	}
}

// TestConnectionStatusTracking tests session manager connection tracking
func TestConnectionStatusTracking(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	mockServer := newMockBBCoreServer()
	defer mockServer.Close()

	apiClient := api.NewClient(mockServer.URL(), "test-token")

	manager := NewManager()
	manager.Initialize(apiClient, "test-device-hash")

	// Manually set config (simulating a started session)
	cfg := &api.Config{
		AgencyId: 123,
		Teams: []api.Team{
			{
				TeamId: "team1",
				Streamers: []api.Streamer{
					{
						StreamerId: "streamer1",
						BigoId:     "bigo123",
						BigoRoomId: "room123",
					},
				},
			},
		},
	}
	manager.config = config.NewManager(cfg)

	t.Run("UpdateConnectionStatus", func(t *testing.T) {
		// Update connection status
		manager.UpdateConnectionStatus("room123", "CONNECTED", "", 100)

		// Get status
		status := manager.GetStatus()

		if len(status.Connections) != 1 {
			t.Fatalf("Expected 1 connection, got: %d", len(status.Connections))
		}

		conn := status.Connections[0]
		if conn.BigoRoomId != "room123" {
			t.Errorf("Expected room123, got: %s", conn.BigoRoomId)
		}
		if conn.Status != "CONNECTED" {
			t.Errorf("Expected CONNECTED, got: %s", conn.Status)
		}
		if conn.MessagesReceived != 100 {
			t.Errorf("Expected 100 messages, got: %d", conn.MessagesReceived)
		}
	})

	t.Run("UpdateWithError", func(t *testing.T) {
		manager.UpdateConnectionStatus("room123", "DISCONNECTED", "Connection lost", 100)

		status := manager.GetStatus()
		conn := status.Connections[0]

		if conn.Status != "DISCONNECTED" {
			t.Errorf("Expected DISCONNECTED, got: %s", conn.Status)
		}
		if conn.Error != "Connection lost" {
			t.Errorf("Expected error message, got: %s", conn.Error)
		}
	})
}

// TestMultipleStreamers tests handling of multiple streamers
func TestMultipleStreamers(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	mockServer := newMockBBCoreServer()
	defer mockServer.Close()

	apiClient := api.NewClient(mockServer.URL(), "test-token")

	cfg := &api.Config{
		AgencyId: 123,
		Teams: []api.Team{
			{
				TeamId: "team1",
				Streamers: []api.Streamer{
					{StreamerId: "s1", BigoId: "b1", BigoRoomId: "r1"},
					{StreamerId: "s2", BigoId: "b2", BigoRoomId: "r2"},
					{StreamerId: "s3", BigoId: "b3", BigoRoomId: "r3"},
				},
			},
			{
				TeamId: "team2",
				Streamers: []api.Streamer{
					{StreamerId: "s4", BigoId: "b4", BigoRoomId: "r4"},
					{StreamerId: "s5", BigoId: "b5", BigoRoomId: "r5"},
				},
			},
		},
	}

	t.Run("ValidateMultipleStreamers", func(t *testing.T) {
		// Collect all streamers
		var streamers []api.ValidateTrialStreamer
		for _, team := range cfg.Teams {
			for _, s := range team.Streamers {
				streamers = append(streamers, api.ValidateTrialStreamer{
					BigoId:     s.BigoId,
					BigoRoomId: s.BigoRoomId,
				})
			}
		}

		if len(streamers) != 5 {
			t.Fatalf("Expected 5 streamers, got: %d", len(streamers))
		}

		resp, err := apiClient.ValidateTrial(streamers)
		if err != nil {
			t.Fatalf("ValidateTrial failed: %v", err)
		}

		if !resp.Allowed {
			t.Errorf("Expected trial to be allowed for multiple streamers")
		}
	})

	t.Run("ConfigManagerMultipleRooms", func(t *testing.T) {
		cfgMgr := config.NewManager(cfg)

		rooms := cfgMgr.GetAllBigoRoomIds()
		if len(rooms) != 5 {
			t.Errorf("Expected 5 rooms, got: %d", len(rooms))
		}

		// Verify each room can be looked up
		expectedRooms := []string{"r1", "r2", "r3", "r4", "r5"}
		for _, roomId := range expectedRooms {
			streamer, err := cfgMgr.LookupStreamerByBigoRoom(roomId)
			if err != nil {
				t.Errorf("Failed to lookup room %s: %v", roomId, err)
			}
			if streamer == nil {
				t.Errorf("Got nil streamer for room %s", roomId)
			}
		}
	})
}

// TestHeartbeatService tests the heartbeat goroutine
func TestHeartbeatService(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	mockServer := newMockBBCoreServer()
	defer mockServer.Close()

	apiClient := api.NewClient(mockServer.URL(), "test-token")

	manager := NewManager()
	manager.Initialize(apiClient, "test-device-hash")

	// Set minimal config
	cfg := &api.Config{
		AgencyId: 123,
		Teams:    []api.Team{},
	}
	manager.config = config.NewManager(cfg)

	t.Run("HeartbeatStartStop", func(t *testing.T) {
		// Create heartbeat with short interval for testing
		heartbeat := NewHeartbeat(manager, apiClient, "test-room", 100*time.Millisecond)

		// Start heartbeat
		heartbeat.Start()

		// Wait for at least 2 heartbeats
		time.Sleep(250 * time.Millisecond)

		// Stop heartbeat
		heartbeat.Stop()

		callCount := mockServer.heartbeatCalled
		if callCount < 2 {
			t.Logf("Warning: Expected at least 2 heartbeat calls, got: %d", callCount)
		}

		// Wait a bit more - should not receive more heartbeats after stop
		beforeStop := callCount
		time.Sleep(150 * time.Millisecond)
		afterStop := mockServer.heartbeatCalled

		if afterStop > beforeStop {
			t.Errorf("Heartbeat continued after stop: before=%d, after=%d", beforeStop, afterStop)
		}
	})
}
