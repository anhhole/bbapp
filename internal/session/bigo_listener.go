package session

import (
	"context"
	"fmt"
	"sync"
	"time"

	"bbapp/internal/api"
	"bbapp/internal/browser"
	"bbapp/internal/listener"
	"strings"
)

func strEqual(a, b string) bool {
	return strings.EqualFold(a, b)
}

// BufferedEvent represents an event with timestamp for time-based buffering
type BufferedEvent struct {
	Event     interface{} `json:"event"`
	Timestamp time.Time   `json:"timestamp"`
}

// BigoConnection represents a connection to a single Bigo room
type BigoConnection struct {
	BigoRoomId       string    `json:"bigoRoomId"`
	BigoId           string    `json:"bigoId"`
	IdolName         string    `json:"idolName"`
	Avatar           string    `json:"avatar"`
	Username         string    `json:"username"`
	Status           string    `json:"status"` // CONNECTING, CONNECTED, DISCONNECTED, ERROR
	MessagesReceived int64     `json:"messagesReceived"`
	LastMessageAt    time.Time `json:"lastMessageAt"`
	TotalDiamonds    int64     `json:"totalDiamonds"`
	Error            string    `json:"error"`
	// Browser instance would be managed here in future
}

// BigoListenerSession manages hidden browser connections to Bigo rooms
type BigoListenerSession struct {
	connections     map[string]*BigoConnection        // bigoRoomId -> connection
	listeners       map[string]*listener.BigoListener // bigoRoomId -> listener
	browserManager  *browser.Manager
	eventBuffer     []BufferedEvent
	bufferTTL       time.Duration // How long to keep events in buffer
	isActive        bool
	startTime       time.Time
	config          *api.Config
	mutex           sync.RWMutex
	stopChan        chan struct{}
	recentGifts     []listener.BigoGift
	onGiftCallbacks []func(interface{})
	giftLibrary     []api.GiftDefinition
}

// NewBigoListenerSession creates a new Bigo listener session
func NewBigoListenerSession(browserManager *browser.Manager) *BigoListenerSession {
	return &BigoListenerSession{
		connections:     make(map[string]*BigoConnection),
		listeners:       make(map[string]*listener.BigoListener),
		browserManager:  browserManager,
		eventBuffer:     make([]BufferedEvent, 0),
		bufferTTL:       5 * time.Minute, // 5-minute time-based buffer
		isActive:        false,
		recentGifts:     make([]listener.BigoGift, 0),
		onGiftCallbacks: make([]func(interface{}), 0),
		giftLibrary:     make([]api.GiftDefinition, 0),
	}
}

// Start starts the Bigo listener session and connects to the main room (config.RoomId)
func (b *BigoListenerSession) Start(config *api.Config) error {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	fmt.Printf("[BigoListener] [%p] Start called. Current isActive: %v\n", b, b.isActive)

	if b.isActive {
		fmt.Printf("[BigoListener] [%p] Warning: Start called while already active. Forcing stop first.\n", b)
		b.mutex.Unlock() // Unlock to call Stop
		b.Stop()
		b.mutex.Lock() // Re-lock
	}

	b.config = config
	b.isActive = true
	b.startTime = time.Now()
	b.stopChan = make(chan struct{})

	fmt.Printf("[BigoListener] Starting with Gift Library size: %d\n", len(b.giftLibrary))

	// Clear previous connections
	b.connections = make(map[string]*BigoConnection)
	b.listeners = make(map[string]*listener.BigoListener)

	roomID := config.RoomId
	if roomID == "" {
		fmt.Printf("[BigoListener] ERROR: No Room ID provided in config\n")
		b.isActive = false // Abort
		return fmt.Errorf("no room ID provided in config")
	}

	fmt.Printf("[BigoListener] Starting session for MAIN ROOM ID: %s. (Ignoring %d teams for connection purposes)\n", roomID, len(config.Teams))

	// Initialize single connection for the main room
	conn := &BigoConnection{
		BigoRoomId:       roomID,
		BigoId:           roomID,      // Initially assume same, will update if resolved
		IdolName:         "Main Room", // Default name
		Status:           "CONNECTING",
		MessagesReceived: 0,
		TotalDiamonds:    0,
		LastMessageAt:    time.Time{},
	}

	b.connections[roomID] = conn
	fmt.Printf("[BigoListener] Initialized connection for Main Room %s\n", roomID)

	// Fetch user info from API
	go func() {
		info, err := listener.GetUserInfo(roomID)
		if err == nil && info != nil {
			b.mutex.Lock()
			if c, ok := b.connections[roomID]; ok {
				c.Avatar = info.Avatar
				c.Username = info.NickName
				c.IdolName = info.NickName // Update display name
			}
			b.mutex.Unlock()
		} else {
			fmt.Printf("[BigoListener] Failed to fetch user info: %v\n", err)
		}
	}()

	// Start real listener for the main room
	// We pass "Main Room" as the name, or we could try to find a matching name in the teams if we wanted to be fancy,
	// but "Main Room" is clear enough for the single-listener context.
	go b.startRealListener(roomID, roomID, "Main Room")

	// Start buffer cleanup goroutine
	go b.cleanupBufferLoop()

	fmt.Printf("[BigoListener] ✓ Started successfully: Listening to %s\n", roomID)
	return nil
}

// GetBigoIdByRoomId finds the original BigoId (input string) for a given resolved Room ID (numeric)
func (b *BigoListenerSession) GetBigoIdByRoomId(numericId string) string {
	b.mutex.RLock()
	defer b.mutex.RUnlock()

	for _, conn := range b.connections {
		// key is typically input BigoId
		// conn.BigoRoomId should be the resolved numeric ID (if updated)
		// conn.BigoId is the input string

		// First check if numericId matches the resolved room ID
		if conn.BigoRoomId == numericId {
			return conn.BigoId
		}

		// Also check if numericId matches the input string (just in case they are same)
		if conn.BigoId == numericId {
			return conn.BigoId
		}
	}
	return ""
}

// Stop stops the Bigo listener session and closes all browser connections
func (b *BigoListenerSession) Stop() error {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	fmt.Printf("[BigoListener] [%p] Stop called. Current isActive: %v\n", b, b.isActive)

	if !b.isActive {
		// Even if not active, ensure resources are cleared
		b.isActive = false
		return nil
	}

	fmt.Println("[BigoListener] Stopping Bigo listener session...")

	// Signal cleanup goroutine and listeners to stop
	if b.stopChan != nil {
		close(b.stopChan)
		b.stopChan = nil
	}

	// Close all browser connections and listeners
	for roomId, conn := range b.connections {
		conn.Status = "DISCONNECTED"
		fmt.Printf("[BigoListener] Disconnected from %s (room: %s)\n", conn.IdolName, roomId)
	}

	// Clear state
	b.listeners = make(map[string]*listener.BigoListener)
	b.connections = make(map[string]*BigoConnection)
	b.isActive = false

	fmt.Println("[BigoListener] ✓ Stopped successfully")
	return nil
}

// startRealListener starts a real BigoListener for a room
func (b *BigoListenerSession) startRealListener(mapKey, urlId, idolName string) {
	fmt.Printf("[BigoListener] Starting real listener for %s (mapKey: %s, urlId: %s)...\n", idolName, mapKey, urlId)

	// 1. Create browser context
	ctx, cancel, err := b.browserManager.CreateBrowser(urlId)
	if err != nil {
		fmt.Printf("[BigoListener] ERROR: Failed to create browser for %s: %v\n", idolName, err)
		b.UpdateConnectionStatus(mapKey, "ERROR", err.Error(), 0)
		return
	}
	defer cancel()

	// 2. Create listener
	l := listener.NewBigoListener(urlId, ctx)

	b.mutex.Lock()
	b.listeners[mapKey] = l
	b.mutex.Unlock()

	// 3. Register handlers
	l.OnGift(func(gift listener.BigoGift) {
		fmt.Printf("[BigoListener] Received gift from %s: %s (x%d)\n", gift.SenderName, gift.GiftName, gift.GiftCount)

		// Update msg count and accumulate diamonds
		b.mutex.Lock()
		if conn, ok := b.connections[mapKey]; ok {
			conn.MessagesReceived++
			conn.LastMessageAt = time.Now()

			// Forecast/Accumulate logic
			// Apply Gift Library override first
			// Apply Gift Library override first
			if len(b.giftLibrary) > 0 {
				fmt.Printf("[BigoListener] Checking %d gifts in library for '%s' (ID: %s)...\n", len(b.giftLibrary), gift.GiftName, gift.GiftId)
				for _, def := range b.giftLibrary {
					// Check for ID match (primary) or Name match (secondary, case-insensitive)
					matchID := (def.ID != "" && def.ID == gift.GiftId)
					matchName := (gift.GiftName != "" && strEqual(def.Name, gift.GiftName))

					if matchID || matchName {
						if def.Diamonds > 0 {
							fmt.Printf("[BigoListener] OVERRIDE: %s (val: %d -> %d) [MatchID: %v, MatchName: %v]\n",
								gift.GiftName, gift.Diamonds, def.Diamonds, matchID, matchName)
							gift.Diamonds = int64(def.Diamonds)
						} else {
							// Found in library but value is 0.
							// fmt.Printf("[BigoListener] Library match found but value is 0\n")
						}
						break
					}
				}
			}

			// conn.TotalDiamonds += gift.Diamonds

			// Set the room total on the gift event for the payload
			gift.RoomTotalDiamonds = conn.TotalDiamonds
		}
		b.mutex.Unlock()

		// Buffer the event
		b.BufferEvent(gift)

		// Notify subscribers
		b.notifySubscribers(gift)

		// Add to recent gifts log
		b.mutex.Lock()
		b.recentGifts = append([]listener.BigoGift{gift}, b.recentGifts...)
		if len(b.recentGifts) > 50 {
			b.recentGifts = b.recentGifts[:50]
		}
		b.mutex.Unlock()
	})

	l.OnChat(func(chat listener.BigoChat) {
		fmt.Printf("[BigoListener] Received chat from %s in %s: %s\n", chat.SenderName, urlId, chat.Message)
		// Update msg count
		b.mutex.Lock()
		if conn, ok := b.connections[mapKey]; ok {
			conn.MessagesReceived++
			conn.LastMessageAt = time.Now()
		}
		b.mutex.Unlock()

		// Notify subscribers (send to BB-Core)
		b.notifySubscribers(chat)
	})

	// 4. Start listening
	b.UpdateConnectionStatus(mapKey, "CONNECTED", "", 0)

	// Create a context that is cancelled when stopChan is closed
	listenCtx, listenCancel := context.WithCancel(ctx)
	defer listenCancel()

	go func() {
		select {
		case <-b.stopChan:
			listenCancel()
		case <-listenCtx.Done():
		}
	}()

	// Capture resolved Room ID from Start()
	if resolvedId, err := l.Start(); err != nil {
		fmt.Printf("[BigoListener] ERROR: Listener for %s stopped: %v\n", idolName, err)
		b.UpdateConnectionStatus(mapKey, "ERROR", err.Error(), 0)
		return
	} else {
		// Update connection with resolved ID!
		fmt.Printf("[BigoListener] Listener started. Resolved ID: %s (was: %s)\n", resolvedId, urlId)
		// Update the BigoId in the connection to reflect the resolved room ID
		b.mutex.Lock()
		if conn, ok := b.connections[mapKey]; ok {
			conn.BigoId = resolvedId
			// Also update BigoRoomId if that's what we want to track as the "real" room ID
			conn.BigoRoomId = resolvedId
		}
		b.mutex.Unlock()
	}

	<-listenCtx.Done()
	fmt.Printf("[BigoListener] Listener for %s room %s finished\n", idolName, urlId)
}

// BufferEvent adds an event to the time-based buffer
func (b *BigoListenerSession) BufferEvent(event interface{}) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	bufferedEvent := BufferedEvent{
		Event:     event,
		Timestamp: time.Now(),
	}

	b.eventBuffer = append(b.eventBuffer, bufferedEvent)
}

// GetBufferedEvents returns all events in the buffer and clears it
func (b *BigoListenerSession) GetBufferedEvents() []interface{} {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	events := make([]interface{}, len(b.eventBuffer))
	for i, buffered := range b.eventBuffer {
		events[i] = buffered.Event
	}

	// Clear buffer after retrieval
	b.eventBuffer = make([]BufferedEvent, 0)

	return events
}

// cleanupBufferLoop periodically removes old events from buffer
func (b *BigoListenerSession) cleanupBufferLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			b.cleanupOldEvents()
		case <-b.stopChan:
			return
		}
	}
}

// cleanupOldEvents removes events older than bufferTTL
func (b *BigoListenerSession) cleanupOldEvents() {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	if len(b.eventBuffer) == 0 {
		return
	}

	cutoffTime := time.Now().Add(-b.bufferTTL)
	newBuffer := make([]BufferedEvent, 0)

	for _, buffered := range b.eventBuffer {
		if buffered.Timestamp.After(cutoffTime) {
			newBuffer = append(newBuffer, buffered)
		}
	}

	removed := len(b.eventBuffer) - len(newBuffer)
	if removed > 0 {
		fmt.Printf("[BigoListener] Cleaned up %d old events from buffer\n", removed)
	}

	b.eventBuffer = newBuffer
}

// UpdateConnectionStatus updates the status of a specific connection
func (b *BigoListenerSession) UpdateConnectionStatus(bigoRoomId, status, errorMsg string, msgCount int64) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	conn, exists := b.connections[bigoRoomId]
	if !exists {
		fmt.Printf("[BigoListener] WARNING: Unknown room %s\n", bigoRoomId)
		return
	}

	conn.Status = status
	conn.Error = errorMsg
	conn.MessagesReceived = msgCount
	// conn.TotalDiamonds is not updated here, preserved from state
	conn.LastMessageAt = time.Now()
}

// parsePayloadGift logic also calls giftHandlers, so startRealListener handles it via l.OnGift?
// Wait, startRealListener uses l.OnGift.
// parsePayloadGift is called by handleFrame which then calls handlers.
// So modifying the handler in startRealListener is sufficient.

// GetStatus returns the current status of the Bigo listener session
func (b *BigoListenerSession) GetStatus() BigoListenerStatus {
	b.mutex.RLock()
	defer b.mutex.RUnlock()

	connections := make([]BigoConnection, 0, len(b.connections))
	for _, conn := range b.connections {
		connections = append(connections, *conn)
	}

	// copy gifts to avoid race
	gifts := make([]listener.BigoGift, len(b.recentGifts))
	copy(gifts, b.recentGifts)

	return BigoListenerStatus{
		IsActive:       b.isActive,
		StartTime:      b.startTime,
		TotalIdols:     len(b.connections),
		ConnectedIdols: b.countConnected(),
		BufferedEvents: len(b.eventBuffer),
		Connections:    connections,
		RecentGifts:    gifts,
	}
}

// countConnected counts how many connections are in CONNECTED status
func (b *BigoListenerSession) countConnected() int {
	count := 0
	for _, conn := range b.connections {
		if conn.Status == "CONNECTED" {
			count++
		}
	}
	return count
}

// IsActive returns whether the session is active
func (b *BigoListenerSession) IsActive() bool {
	b.mutex.RLock()
	defer b.mutex.RUnlock()
	return b.isActive
}

// simulateConnection simulates a browser connection (placeholder for actual implementation)
func (b *BigoListenerSession) simulateConnection(conn *BigoConnection) {
	// Simulate connection delay
	time.Sleep(2 * time.Second)

	b.mutex.Lock()
	conn.Status = "CONNECTED"
	b.mutex.Unlock()

	fmt.Printf("[BigoListener] ✓ Connected to %s (room: %s)\n", conn.IdolName, conn.BigoRoomId)
}

// BigoListenerStatus represents the status of the Bigo listener session
type BigoListenerStatus struct {
	IsActive       bool                `json:"isActive"`
	StartTime      time.Time           `json:"startTime"`
	TotalIdols     int                 `json:"totalIdols"`
	ConnectedIdols int                 `json:"connectedIdols"`
	BufferedEvents int                 `json:"bufferedEvents"`
	Connections    []BigoConnection    `json:"connections"`
	RecentGifts    []listener.BigoGift `json:"recentGifts"`
}

// SetGiftLibrary updates the gift library used for diamond value lookup
func (b *BigoListenerSession) SetGiftLibrary(lib []api.GiftDefinition) {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	b.giftLibrary = lib
	fmt.Printf("[BigoListener] SetGiftLibrary called. Library size: %d\n", len(lib))
	for _, g := range lib {
		fmt.Printf("   - %s (%s): %d\n", g.Name, g.ID, g.Diamonds)
	}
}

// SubscribeOnGift registers a callback for gift events
func (b *BigoListenerSession) SubscribeOnGift(callback func(interface{})) {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	b.onGiftCallbacks = append(b.onGiftCallbacks, callback)
}

// notifySubscribers notifies all registered callbacks
func (b *BigoListenerSession) notifySubscribers(event interface{}) {
	// We use a read lock to copy callbacks, then call them outside the lock to avoid deadlocks
	b.mutex.RLock()
	callbacks := make([]func(interface{}), len(b.onGiftCallbacks))
	copy(callbacks, b.onGiftCallbacks)
	b.mutex.RUnlock()

	for _, cb := range callbacks {
		// Run in goroutine to allow non-blocking processing
		go cb(event)
	}
}
