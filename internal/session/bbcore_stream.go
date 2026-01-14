package session

import (
	"fmt"
	"strings"
	"sync"

	"bbapp/internal/api"
	"bbapp/internal/listener"
	"bbapp/internal/stomp"
	"time"
)

// SenderBindingCache tracks which streamer a sender is bound to temporarily
type SenderBindingCache struct {
	StreamerBigoId string
	ExpiresAt      time.Time
}

// BBCoreStreamSession manages the BB-Core streaming session
type BBCoreStreamSession struct {
	apiClient    *api.Client
	stompClient  *stomp.Client
	heartbeat    *Heartbeat
	sessionId    string
	roomId       string
	deviceHash   string
	isActive     bool
	config       *api.Config
	bigoListener *BigoListenerSession // Reference to get buffered events
	mutex        sync.RWMutex
	stopChan     chan struct{}

	senderBindings map[string]SenderBindingCache
	bindingsMutex  sync.Mutex
}

// NewBBCoreStreamSession creates a new BB-Core stream session
func NewBBCoreStreamSession(apiClient *api.Client, deviceHash string) *BBCoreStreamSession {
	return &BBCoreStreamSession{
		apiClient:      apiClient,
		deviceHash:     deviceHash,
		isActive:       false,
		stopChan:       make(chan struct{}),
		senderBindings: make(map[string]SenderBindingCache),
	}
}

// Start starts the BB-Core streaming session
// Requires an active Bigo listener session
func (s *BBCoreStreamSession) Start(roomId string, config *api.Config, bigoListener *BigoListenerSession, bbCoreURL, accessToken string, durationMinutes int) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.isActive {
		return fmt.Errorf("BB-Core stream session already active")
	}

	// Validate that Bigo listener is active
	if !bigoListener.IsActive() {
		return fmt.Errorf("Bigo listener session must be active before starting BB-Core stream")
	}

	s.config = config
	s.roomId = roomId
	s.bigoListener = bigoListener

	fmt.Println("[BBCoreStream] Starting BB-Core stream session...")

	// Step 1: Validate trial
	fmt.Println("[BBCoreStream] Step 1: Validating trial...")
	streamers := make([]api.ValidateTrialStreamer, 0)
	for _, team := range config.Teams {
		for _, idol := range team.Streamers {
			streamers = append(streamers, api.ValidateTrialStreamer{
				BigoId:     idol.BigoId,
				BigoRoomId: idol.BigoRoomId,
			})
		}
	}

	validationResp, err := s.apiClient.ValidateTrial(streamers)
	if err != nil {
		return fmt.Errorf("trial validation failed: %w", err)
	}

	if !validationResp.Allowed {
		return fmt.Errorf("trial validation rejected: %s (blocked IDs: %v)",
			validationResp.Message, validationResp.BlockedBigoIds)
	}

	fmt.Println("[BBCoreStream] ✓ Trial validation passed")

	// Step 2: Start session at BB-Core
	fmt.Println("[BBCoreStream] Step 2: Starting session at BB-Core...")
	scriptPayload := map[string]interface{}{
		"minTeams": len(config.Teams),
	}

	resp, err := s.apiClient.StartSession(roomId, durationMinutes, scriptPayload)
	if err != nil {
		return fmt.Errorf("start session API call failed: %w", err)
	}

	if resp.Status != "ACTIVE" {
		return fmt.Errorf("session start failed: status=%s", resp.Status)
	}

	s.sessionId = resp.SessionId
	fmt.Printf("[BBCoreStream] ✓ Session started: %s (duration=%dm)\n", s.sessionId, resp.DurationMinutes)

	// Step 3: Establish STOMP connection
	fmt.Println("[BBCoreStream] Step 3: Establishing STOMP connection...")

	// Ensure URL ends with /ws for STOMP connection
	stompURL := bbCoreURL
	if !strings.HasSuffix(stompURL, "/ws") {
		stompURL = strings.TrimSuffix(stompURL, "/") + "/ws"
	}

	stompClient, err := stomp.NewClient(stompURL, accessToken, "")
	if err != nil {
		// Rollback: stop session at BB-Core
		s.apiClient.StopSession(s.sessionId)
		return fmt.Errorf("STOMP connection failed: %w", err)
	}

	s.stompClient = stompClient
	fmt.Println("[BBCoreStream] ✓ STOMP connected")

	// Step 4: Publish any buffered events from Bigo listener
	fmt.Println("[BBCoreStream] Step 4: Publishing buffered events...")
	bufferedEvents := bigoListener.GetBufferedEvents()
	if len(bufferedEvents) > 0 {
		fmt.Printf("[BBCoreStream] Publishing %d buffered events...\n", len(bufferedEvents))
		for _, event := range bufferedEvents {
			s.publishEvent(event)
		}
		fmt.Printf("[BBCoreStream] ✓ Published %d buffered events\n", len(bufferedEvents))
	} else {
		fmt.Println("[BBCoreStream] No buffered events to publish")
	}

	// Step 5: Start heartbeat service
	fmt.Println("[BBCoreStream] Step 5: Starting heartbeat service...")
	// TODO: Implement heartbeat for BBCoreStreamSession
	// For now, skip heartbeat - it will be added in a future update
	fmt.Println("[BBCoreStream] ⚠ Heartbeat not yet implemented for separated sessions")

	// Step 6: Subscribe to live events
	fmt.Println("[BBCoreStream] Step 6: Subscribing to Bigo listener events...")
	s.bigoListener.SubscribeOnGift(func(event interface{}) {
		s.publishEvent(event)
	})
	fmt.Println("[BBCoreStream] ✓ Subscribed to live events")

	// Step 7: Subscribe to topic
	topic := fmt.Sprintf("/topic/room/%s/scene", s.roomId)
	fmt.Printf("[BBCoreStream] Step 7: Subscribing to %s...\n", topic)
	s.stompClient.Subscribe(topic, func(msg []byte) {
		fmt.Printf("[BBCoreStream] Received scene update: %s\n", string(msg))
	})
	fmt.Println("[BBCoreStream] ✓ Subscribed to scene updates")

	s.isActive = true

	fmt.Printf("[BBCoreStream] ✓✓✓ Stream session fully started: %s\n", s.sessionId)
	return nil
}

// Stop stops the BB-Core streaming session
func (s *BBCoreStreamSession) Stop(reason string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if !s.isActive {
		return fmt.Errorf("BB-Core stream session not active")
	}

	fmt.Printf("[BBCoreStream] Stopping stream session (reason: %s)...\n", reason)

	// Step 1: Stop heartbeat
	if s.heartbeat != nil {
		fmt.Println("[BBCoreStream] Step 1: Stopping heartbeat...")
		s.heartbeat.Stop()
		s.heartbeat = nil
		fmt.Println("[BBCoreStream] ✓ Heartbeat stopped")
	}

	// Step 2: Disconnect STOMP
	if s.stompClient != nil {
		fmt.Println("[BBCoreStream] Step 2: Disconnecting STOMP...")
		s.stompClient.Disconnect()
		s.stompClient = nil
		fmt.Println("[BBCoreStream] ✓ STOMP disconnected")
	}

	// Step 3: Stop session at BB-Core
	fmt.Printf("[BBCoreStream] Step 3: Stopping session %s at BB-Core...\n", s.sessionId)
	resp, err := s.apiClient.StopSession(s.sessionId)
	if err != nil {
		fmt.Printf("[BBCoreStream] WARNING: Failed to stop session at BB-Core: %v\n", err)
	} else if resp.Status != "COMPLETED" && resp.Status != "STOPPED" {
		fmt.Printf("[BBCoreStream] WARNING: Unexpected stop status: %s\n", resp.Status)
	} else {
		fmt.Printf("[BBCoreStream] ✓ Session stopped at BB-Core (status=%s)\n", resp.Status)
	}

	s.isActive = false
	s.sessionId = ""

	fmt.Println("[BBCoreStream] ✓✓✓ Stream session stopped successfully")
	return nil
}

// PublishEvent publishes a gift event to BB-Core via STOMP
func (s *BBCoreStreamSession) publishEvent(event interface{}) error {
	s.mutex.RLock()
	stompClient := s.stompClient
	isActive := s.isActive
	s.mutex.RUnlock()

	if !isActive || stompClient == nil {
		return fmt.Errorf("stream session not active")
	}

	// Use the Client.Publish method which handles marshaling and sending
	var dest string
	var payload interface{}

	switch e := event.(type) {
	case listener.BigoGift:
		// Resolve Streamer ID first
		resolvedStreamerId := s.resolveStreamerId(e.StreamerId, e.GiftName, e.SenderId)
		if resolvedStreamerId == "" {
			fmt.Printf("[BBCoreStream] IGNORED gift '%s' from '%s' (SenderId: %s) - No binding or history match.\n",
				e.GiftName, e.SenderName, e.SenderId)
			return nil
		}

		// Use legacy endpoint /bigo instead of /gift to match original app.go behavior
		dest = fmt.Sprintf("/app/room/%s/bigo", s.roomId)
		giftPayload := BBCoreGiftPayload{
			Sender:    e.SenderName,
			TeamId:    s.resolveTeamId(e.BigoRoomId, e.GiftName),
			GiftName:  e.GiftName,
			Value:     int64(e.Diamonds),
			Count:     e.GiftCount,
			Avatar:    e.SenderAvatar,
			Timestamp: e.Timestamp,
			Diamonds:  int64(e.Diamonds), // Populate legacy field
		}

		// Create a map to ensure we send exactly what the legacy endpoint expects,
		// including the "type" field which might be required for the /bigo endpoint dispatcher.
		payloadMap := map[string]interface{}{
			"type":       "GIFT",
			"roomId":     s.roomId,
			"bigoRoomId": e.BigoRoomId,
			"senderName": giftPayload.Sender,
			"teamId":     giftPayload.TeamId,
			"giftName":   giftPayload.GiftName,
			"value":      giftPayload.Value,
			"diamonds":   giftPayload.Diamonds,
			"count":      giftPayload.Count,
			"avatar":     giftPayload.Avatar,
			"timestamp":  giftPayload.Timestamp,
			// Add extra fields if needed by backend, matching app.go
			"senderId":     e.SenderId,
			"senderLevel":  e.SenderLevel,
			"streamerId":   resolvedStreamerId, // Use the STRICTLY resolved ID
			"streamerName": e.StreamerName,
			"giftId":       e.GiftId,
			"giftImageUrl": e.GiftImageUrl,
		}

		payload = payloadMap

		// Explicitly log the value being sent to identify issues
		fmt.Printf("[BBCoreStream] >>> SENDING GIFT PAYLOAD (Legacy Mode): Name=%s, Diamonds=%d\n",
			e.GiftName, e.Diamonds)

		fmt.Printf("[BBCoreStream] Publishing GIFT to %s: %+v\n", dest, payload)

	case listener.BigoChat:
		dest = fmt.Sprintf("/app/room/%s/chat", s.roomId)
		payload = BBCoreChatPayload{
			Sender:    e.SenderName,
			TeamId:    s.resolveTeamId(e.BigoRoomId, ""),
			Message:   e.Message,
			Avatar:    e.SenderAvatar,
			Timestamp: e.Timestamp,
		}
		fmt.Printf("[BBCoreStream] Publishing CHAT to %s: %+v\n", dest, payload)

	default:
		// Fallback for unknown events, or log error
		dest = fmt.Sprintf("/app/room/%s/gift", s.roomId)
		payload = event
		fmt.Printf("[BBCoreStream] Publishing UNKNOWN event to %s: %+v\n", dest, event)
	}

	return stompClient.Publish(dest, payload)
}

// GetStatus returns the current status of the BB-Core stream session
func (s *BBCoreStreamSession) GetStatus() BBCoreStreamStatus {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	return BBCoreStreamStatus{
		IsActive:   s.isActive,
		SessionId:  s.sessionId,
		RoomId:     s.roomId,
		DeviceHash: s.deviceHash,
	}
}

// IsActive returns whether the session is active
func (s *BBCoreStreamSession) IsActive() bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.isActive
}

// GetStompClient returns the STOMP client for external use
func (s *BBCoreStreamSession) GetStompClient() *stomp.Client {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.stompClient
}

// GetSessionId returns the current session ID
func (s *BBCoreStreamSession) GetSessionId() string {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.sessionId
}

// BBCoreStreamStatus represents the status of the BB-Core stream session
type BBCoreStreamStatus struct {
	IsActive   bool   `json:"isActive"`
	SessionId  string `json:"sessionId"`
	RoomId     string `json:"roomId"`
	DeviceHash string `json:"deviceHash"`
}

// BBCoreGiftPayload represents the payload sent to BB-Core for gifts
type BBCoreGiftPayload struct {
	Sender    string `json:"sender"`
	TeamId    string `json:"teamId"`
	GiftName  string `json:"giftName"`
	Value     int64  `json:"value"`
	Count     int    `json:"count"`
	Avatar    string `json:"avatar"`
	Timestamp int64  `json:"timestamp"`
	Diamonds  int64  `json:"diamonds"` // Legacy field for backend compatibility
}

// BBCoreChatPayload represents the payload sent to BB-Core for chat/votes
type BBCoreChatPayload struct {
	Sender    string `json:"sender"`
	TeamId    string `json:"teamId"`
	Message   string `json:"message"`
	Avatar    string `json:"avatar"`
	Timestamp int64  `json:"timestamp"`
}

// resolveTeamId finds the TeamId for a given BigoRoomId or GiftName from the config
func (s *BBCoreStreamSession) resolveTeamId(bigoRoomId string, giftName string) string {
	if s.config == nil {
		return ""
	}

	// 1. Check Binding Gifts first
	if giftName != "" {
		for _, team := range s.config.Teams {
			if strings.EqualFold(team.BindingGift, giftName) {
				return team.TeamId
			}
		}
	}

	// 2. Check Streamers
	for _, team := range s.config.Teams {
		for _, streamer := range team.Streamers {
			if streamer.BigoRoomId == bigoRoomId || streamer.BigoId == bigoRoomId {
				return team.TeamId
			}
		}
	}
	return ""
}

// resolveStreamerId finds the internal StreamerId using Binding Gift priority, then Sender History, then raw ID.
func (s *BBCoreStreamSession) resolveStreamerId(bigoId string, giftName string, senderId string) string {
	if s.config == nil {
		return bigoId
	}

	// 1. Check Streamer Binding Gifts first (Highest Priority)
	if giftName != "" {
		for _, team := range s.config.Teams {
			for _, streamer := range team.Streamers {
				if strings.EqualFold(streamer.BindingGift, giftName) {
					// MATCH FOUND!
					targetId := streamer.BigoId
					if targetId == "" {
						targetId = streamer.StreamerId
					}

					// Cache this binding for the sender
					if senderId != "" {
						s.bindingsMutex.Lock()
						s.senderBindings[senderId] = SenderBindingCache{
							StreamerBigoId: targetId,
							ExpiresAt:      time.Now().Add(60 * time.Second),
						}
						s.bindingsMutex.Unlock()
						fmt.Printf("[BBCoreStream] Sender %s bound to streamer %s for 60s (Trigger: %s)\n", senderId, targetId, giftName)
					}

					return targetId
				}
			}
		}
	}

	// 2. Check Sender History Cache (If no direct binding match)
	if senderId != "" {
		s.bindingsMutex.Lock()
		binding, exists := s.senderBindings[senderId]
		// Clean up expired while looking (lazy cleanup)
		if exists && time.Now().After(binding.ExpiresAt) {
			delete(s.senderBindings, senderId)
			exists = false
		}
		s.bindingsMutex.Unlock()

		if exists {
			fmt.Printf("[BBCoreStream] Sender %s using cached binding to streamer %s\n", senderId, binding.StreamerBigoId)
			return binding.StreamerBigoId
		}
	}

	// 3. Fallback: Return empty string to IGNORE unbound gifts (Strict Mode)
	// User Requirement: "no using the raw id if not have binding gift and history ingore that gift"
	return ""
}
