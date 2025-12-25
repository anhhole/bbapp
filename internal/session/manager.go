package session

import (
	"fmt"
	"sync"
	"time"

	"bbapp/internal/api"
	"bbapp/internal/config"
)

type Manager struct {
	apiClient   *api.Client
	config      *config.Manager
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

func (m *Manager) Start(roomId string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.isActive {
		return fmt.Errorf("session already active")
	}

	// Fetch config
	cfg, err := m.apiClient.GetConfig(roomId)
	if err != nil {
		return fmt.Errorf("fetch config: %w", err)
	}

	m.config = config.NewManager(cfg)
	m.roomId = roomId

	// Start session at BB-Core
	resp, err := m.apiClient.StartSession(roomId, m.deviceHash)
	if err != nil {
		return fmt.Errorf("start session: %w", err)
	}

	if !resp.Success {
		return fmt.Errorf("session start failed: %s", resp.Message)
	}

	m.sessionId = resp.SessionId
	m.isActive = true

	fmt.Printf("[Session] ✓ Session started: %s\n", m.sessionId)
	return nil
}

func (m *Manager) Stop(reason string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if !m.isActive {
		return fmt.Errorf("session not active")
	}

	resp, err := m.apiClient.StopSession(m.roomId, reason)
	if err != nil {
		return fmt.Errorf("stop session: %w", err)
	}

	if !resp.Success {
		return fmt.Errorf("session stop failed: %s", resp.Message)
	}

	m.isActive = false
	m.connections = make(map[string]*api.ConnectionStatus)

	fmt.Printf("[Session] ✓ Session stopped: %s\n", reason)
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
