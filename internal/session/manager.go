package session

import (
	"fmt"
	"sync"
	"time"

	"bbapp/internal/api"
	"bbapp/internal/config"
	"bbapp/internal/stomp"
)

type Manager struct {
	apiClient   *api.Client
	stompClient *stomp.Client
	config      *config.Manager
	heartbeat   *Heartbeat
	sessionId   string
	roomId      string
	deviceHash  string
	isActive    bool
	connections map[string]*api.ConnectionStatus
	mutex       sync.RWMutex
}

func NewManager() *Manager {
	return &Manager{
		connections: make(map[string]*api.ConnectionStatus),
		isActive:    false,
	}
}

func (m *Manager) Initialize(apiClient *api.Client, deviceHash string) {
	m.apiClient = apiClient
	m.deviceHash = deviceHash
}

// Start starts a new PK session with trial validation and STOMP connection
func (m *Manager) Start(roomId string, cfg *api.Config, bbCoreURL, accessToken string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.isActive {
		return fmt.Errorf("session already active")
	}

	// Step 1: Validate trial (final check before session start)
	fmt.Println("[Session] Step 1: Validating streamers for trial...")
	streamers := make([]api.ValidateTrialStreamer, 0)
	for _, team := range cfg.Teams {
		for _, s := range team.Streamers {
			streamers = append(streamers, api.ValidateTrialStreamer{
				BigoId:     s.BigoId,
				BigoRoomId: s.BigoRoomId,
			})
		}
	}

	validationResp, err := m.apiClient.ValidateTrial(streamers)
	if err != nil {
		return fmt.Errorf("trial validation failed: %w", err)
	}

	if !validationResp.Allowed {
		return fmt.Errorf("trial validation rejected: %s (blocked IDs: %v)",
			validationResp.Message, validationResp.BlockedBigoIds)
	}

	fmt.Printf("[Session] ✓ Trial validation passed\n")

	// Step 2: Start session at BB-Core
	fmt.Println("[Session] Step 2: Starting session at BB-Core...")
	resp, err := m.apiClient.StartSession(roomId, m.deviceHash)
	if err != nil {
		return fmt.Errorf("start session API call failed: %w", err)
	}

	if !resp.Success {
		return fmt.Errorf("session start failed: %s", resp.Message)
	}

	m.sessionId = resp.SessionId
	m.roomId = roomId
	m.config = config.NewManager(cfg)

	fmt.Printf("[Session] ✓ Session started at BB-Core: %s\n", m.sessionId)

	// Step 3: Establish STOMP connection
	fmt.Println("[Session] Step 3: Establishing STOMP connection...")
	stompClient, err := stomp.NewClient(bbCoreURL, accessToken)
	if err != nil {
		// Rollback: stop session at BB-Core
		m.apiClient.StopSession(roomId, "STOMP_CONNECTION_FAILED")
		return fmt.Errorf("STOMP connection failed: %w", err)
	}

	if err := stompClient.Connect(); err != nil {
		// Rollback: stop session at BB-Core
		m.apiClient.StopSession(roomId, "STOMP_CONNECTION_FAILED")
		return fmt.Errorf("STOMP connect failed: %w", err)
	}

	m.stompClient = stompClient
	fmt.Printf("[Session] ✓ STOMP connected successfully\n")

	// Step 4: Start heartbeat service
	fmt.Println("[Session] Step 4: Starting heartbeat service...")
	m.heartbeat = NewHeartbeat(m.apiClient, m)
	m.heartbeat.Start()
	fmt.Printf("[Session] ✓ Heartbeat service started (30s interval)\n")

	m.isActive = true

	fmt.Printf("[Session] ✓✓✓ Session fully started: %s\n", m.sessionId)
	return nil
}

// Stop stops the PK session and cleans up all resources
func (m *Manager) Stop(reason string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if !m.isActive {
		return fmt.Errorf("session not active")
	}

	fmt.Printf("[Session] Stopping session (reason: %s)...\n", reason)

	// Step 1: Stop heartbeat service
	if m.heartbeat != nil {
		fmt.Println("[Session] Step 1: Stopping heartbeat service...")
		m.heartbeat.Stop()
		m.heartbeat = nil
		fmt.Println("[Session] ✓ Heartbeat stopped")
	}

	// Step 2: Disconnect STOMP
	if m.stompClient != nil {
		fmt.Println("[Session] Step 2: Disconnecting STOMP...")
		m.stompClient.Disconnect()
		m.stompClient = nil
		fmt.Println("[Session] ✓ STOMP disconnected")
	}

	// Step 3: Notify BB-Core that session is stopping
	fmt.Println("[Session] Step 3: Notifying BB-Core...")
	resp, err := m.apiClient.StopSession(m.roomId, reason)
	if err != nil {
		// Log error but don't fail - session should still stop locally
		fmt.Printf("[Session] WARNING: Failed to notify BB-Core: %v\n", err)
	} else if !resp.Success {
		fmt.Printf("[Session] WARNING: BB-Core stop failed: %s\n", resp.Message)
	} else {
		fmt.Println("[Session] ✓ BB-Core notified")
	}

	// Step 4: Clean up local state
	m.isActive = false
	m.connections = make(map[string]*api.ConnectionStatus)
	m.sessionId = ""
	m.roomId = ""

	fmt.Printf("[Session] ✓✓✓ Session stopped successfully\n")
	return nil
}

func (m *Manager) UpdateConnectionStatus(bigoRoomId, status, errorMsg string, msgCount int64) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	streamer, err := m.config.LookupStreamerByBigoRoom(bigoRoomId)
	if err != nil {
		fmt.Printf("[Session] WARNING: Unknown bigo room %s\n", bigoRoomId)
		return
	}

	conn := &api.ConnectionStatus{
		BigoRoomId:       bigoRoomId,
		StreamerId:       streamer.StreamerId,
		Status:           status,
		MessagesReceived: msgCount,
		LastMessageTime:  time.Now().UnixMilli(),
		ErrorMessage:     errorMsg,
	}

	m.connections[bigoRoomId] = conn
}

func (m *Manager) GetStatus() Status {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	connList := make([]api.ConnectionStatus, 0, len(m.connections))
	for _, conn := range m.connections {
		connList = append(connList, *conn)
	}

	return Status{
		RoomId:      m.roomId,
		SessionId:   m.sessionId,
		IsActive:    m.isActive,
		Connections: connList,
		DeviceHash:  m.deviceHash,
	}
}

func (m *Manager) GetConfig() *config.Manager {
	return m.config
}

// GetStompClient returns the STOMP client for publishing messages
func (m *Manager) GetStompClient() *stomp.Client {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.stompClient
}

// GetDeviceHash returns the device hash for including in messages
func (m *Manager) GetDeviceHash() string {
	return m.deviceHash
}
