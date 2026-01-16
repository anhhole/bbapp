package session

import (
	"fmt"
	"sync"

	"bbapp/internal/api"
	"bbapp/internal/browser"
	"bbapp/internal/config"
)

type Manager struct {
	apiClient      *api.Client
	deviceHash     string
	browserManager *browser.Manager
	bigoListener   *BigoListenerSession
	bbcoreStream   *BBCoreStreamSession
	config         *config.Manager
	mutex          sync.RWMutex
}

func NewManager() *Manager {
	browserMgr := browser.NewManager()
	return &Manager{
		browserManager: browserMgr,
		bigoListener:   NewBigoListenerSession(browserMgr),
	}
}

func (m *Manager) Initialize(apiClient *api.Client, deviceHash string) {
	m.apiClient = apiClient
	m.deviceHash = deviceHash
	m.bbcoreStream = NewBBCoreStreamSession(apiClient, deviceHash)
}

// StartBigoListener starts only the Bigo listener session
func (m *Manager) StartBigoListener(cfg *api.Config) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	fmt.Printf("[Manager] Starting Bigo listener (current state: active=%v)\n", m.bigoListener.IsActive())

	m.config = config.NewManager(cfg)
	err := m.bigoListener.Start(cfg)
	if err != nil {
		fmt.Printf("[Manager] ERROR starting Bigo listener: %v\n", err)
		return err
	}
	fmt.Println("[Manager] ✓ Bigo listener started successfully")
	return nil
}

// StopBigoListener stops only the Bigo listener session
func (m *Manager) StopBigoListener() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	return m.bigoListener.Stop()
}

// StartBBCoreStream starts only the BB-Core streaming session
// Auto-starts Bigo listener if not already active (per user preference)
func (m *Manager) StartBBCoreStream(roomId string, cfg *api.Config, bbCoreURL, accessToken string, durationMinutes int) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Auto-start Bigo listener if not active
	if !m.bigoListener.IsActive() {
		fmt.Println("[Manager] Auto-starting Bigo listener before BB-Core stream...")
		m.config = config.NewManager(cfg)
		if err := m.bigoListener.Start(cfg); err != nil {
			return fmt.Errorf("failed to auto-start Bigo listener: %w", err)
		}
		fmt.Println("[Manager] ✓ Bigo listener auto-started")
	}

	return m.bbcoreStream.Start(roomId, cfg, m.bigoListener, bbCoreURL, accessToken, durationMinutes)
}

// StopBBCoreStream stops only the BB-Core streaming session
// Keeps Bigo listener running (continues buffering events)
func (m *Manager) StopBBCoreStream(reason string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	return m.bbcoreStream.Stop(reason)
}

// Start starts both sessions (convenience method for backward compatibility)
func (m *Manager) Start(roomId string, cfg *api.Config, bbCoreURL, accessToken string, durationMinutes int) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	fmt.Println("[Manager] Starting both Bigo listener and BB-Core stream sessions...")

	// Start Bigo listener first
	m.config = config.NewManager(cfg)
	if err := m.bigoListener.Start(cfg); err != nil {
		return fmt.Errorf("failed to start Bigo listener: %w", err)
	}

	// Then start BB-Core stream
	if err := m.bbcoreStream.Start(roomId, cfg, m.bigoListener, bbCoreURL, accessToken, durationMinutes); err != nil {
		// Rollback: stop Bigo listener
		m.bigoListener.Stop()
		return fmt.Errorf("failed to start BB-Core stream: %w", err)
	}

	fmt.Println("[Manager] ✓✓✓ Both sessions started successfully")
	return nil
}

// Stop stops both sessions (convenience method for backward compatibility)
func (m *Manager) Stop(reason string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	fmt.Printf("[Manager] Stopping both sessions (reason: %s)...\n", reason)

	var streamErr, listenerErr error

	// Stop BB-Core stream first
	if m.bbcoreStream.IsActive() {
		streamErr = m.bbcoreStream.Stop(reason)
	}

	// Then stop Bigo listener
	if m.bigoListener.IsActive() {
		listenerErr = m.bigoListener.Stop()
	}

	if streamErr != nil {
		return fmt.Errorf("failed to stop BB-Core stream: %w", streamErr)
	}
	if listenerErr != nil {
		return fmt.Errorf("failed to stop Bigo listener: %w", listenerErr)
	}

	fmt.Println("[Manager] ✓✓✓ Both sessions stopped successfully")
	return nil
}

// GetBigoListenerStatus returns the status of the Bigo listener session
func (m *Manager) GetBigoListenerStatus() BigoListenerStatus {
	return m.bigoListener.GetStatus()
}

// GetBBCoreStreamStatus returns the status of the BB-Core stream session
func (m *Manager) GetBBCoreStreamStatus() BBCoreStreamStatus {
	return m.bbcoreStream.GetStatus()
}

// GetStatus returns the combined status (for backward compatibility)
func (m *Manager) GetStatus() Status {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	bigoStatus := m.bigoListener.GetStatus()
	streamStatus := m.bbcoreStream.GetStatus()

	// Convert BigoConnection to api.ConnectionStatus for backward compatibility
	connections := make([]api.ConnectionStatus, len(bigoStatus.Connections))
	for i, conn := range bigoStatus.Connections {
		connections[i] = api.ConnectionStatus{
			BigoId:           conn.BigoId,
			BigoRoomId:       conn.BigoRoomId,
			Status:           conn.Status,
			MessagesReceived: conn.MessagesReceived,
			LastMessageAt:    conn.LastMessageAt.UnixMilli(),
			Error:            conn.Error,
		}
	}

	return Status{
		RoomId:      streamStatus.RoomId,
		SessionId:   streamStatus.SessionId,
		IsActive:    bigoStatus.IsActive && streamStatus.IsActive,
		Connections: connections,
		DeviceHash:  m.deviceHash,
	}
}

// GetConfig returns the config manager
func (m *Manager) GetConfig() *config.Manager {
	return m.config
}

// GetStompClient returns the STOMP client from BB-Core stream session
func (m *Manager) GetStompClient() interface{} {
	if m.bbcoreStream == nil {
		return nil
	}
	return m.bbcoreStream.GetStompClient()
}

// GetDeviceHash returns the device hash
func (m *Manager) GetDeviceHash() string {
	return m.deviceHash
}

// UpdateConnectionStatus updates the connection status in Bigo listener
func (m *Manager) UpdateConnectionStatus(bigoRoomId, status, errorMsg string, msgCount int64) {
	m.bigoListener.UpdateConnectionStatus(bigoRoomId, status, errorMsg, msgCount)
}

// SetGiftLibrary updates the gift library in the Bigo listener
func (m *Manager) SetGiftLibrary(lib []api.GiftDefinition) {
	m.bigoListener.SetGiftLibrary(lib)
}

// BufferEvent buffers an event in the Bigo listener session
func (m *Manager) BufferEvent(event interface{}) {
	// Type assertion would be needed here based on actual event type
	// For now, this is a placeholder
	// m.bigoListener.BufferEvent(event)
}

// SubscribeOnGift subscribes to gift events
func (m *Manager) SubscribeOnGift(callback func(interface{})) {
	m.bigoListener.SubscribeOnGift(callback)
}
