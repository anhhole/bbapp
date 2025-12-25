package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"

	"bbapp/internal/api"
	"bbapp/internal/browser"
	"bbapp/internal/fingerprint"
	"bbapp/internal/listener"
	"bbapp/internal/logger"
	"bbapp/internal/overlayserver"
	"bbapp/internal/session"
	"bbapp/internal/stomp"

	"github.com/joho/godotenv"
)

// App struct
type App struct {
	ctx           context.Context
	browserMgr    *browser.Manager
	stompClient   *stomp.Client
	logger        *logger.Logger
	listeners     map[string]*listener.BigoListener
	session       *session.Manager
	heartbeat     *session.Heartbeat
	deviceHash    string
	mutex         sync.RWMutex
	overlayServer *overlayserver.Server
	bbCoreURL     string
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
	fmt.Println("[App] BBapp starting up...")

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

	a.browserMgr = browser.NewManager()
	fmt.Println("[App] Browser manager initialized")

	// Initialize logger
	logger, err := logger.NewLogger("./logs")
	if err != nil {
		fmt.Printf("[App] ERROR: Failed to initialize logger: %v\n", err)
		panic(err)
	}
	a.logger = logger
	fmt.Println("[App] Activity logger initialized (logs will be written to ./logs)")

	// Create debug frames directory
	if err := os.MkdirAll("./debug_frames", 0755); err != nil {
		fmt.Printf("[App] WARNING: Could not create debug_frames directory: %v\n", err)
	} else {
		fmt.Println("[App] Debug frames directory ready (./debug_frames)")
	}

	fmt.Println("[App] Startup complete - ready to connect to BB-Core")
}

// shutdown is called on app termination
func (a *App) shutdown(ctx context.Context) {
	fmt.Println("[App] Shutting down BBapp...")
	
	if a.stompClient != nil {
		fmt.Println("[App] Disconnecting from BB-Core STOMP...")
		a.stompClient.Disconnect()
	}
	
	if a.logger != nil {
		fmt.Println("[App] Closing activity logger...")
		a.logger.Close()
	}
	
	fmt.Println("[App] Shutdown complete")
}

// ConnectToCore connects to BB-Core STOMP
func (a *App) ConnectToCore(url, username, password string) error {
	fmt.Printf("[App] Connecting to BB-Core STOMP at: %s\n", url)
	
	if username != "" {
		fmt.Println("[App] Using authentication token")
	}
	
	client, err := stomp.NewClient(url, username, password)
	if err != nil {
		fmt.Printf("[App] ERROR: STOMP connection failed: %v\n", err)
		return fmt.Errorf("connection failed: %w", err)
	}

	a.stompClient = client
	fmt.Println("[App] ✓ Connected to BB-Core successfully")
	return nil
}

// AddStreamer adds Bigo streamer to monitor
func (a *App) AddStreamer(bigoRoomId, teamId, roomId string) error {
	fmt.Printf("[App] AddStreamer called: bigoRoom=%s, team=%s, room=%s\n", bigoRoomId, teamId, roomId)
	
	a.mutex.Lock()
	defer a.mutex.Unlock()

	// Check if already exists
	if _, exists := a.listeners[bigoRoomId]; exists {
		fmt.Printf("[App] ERROR: Already monitoring room %s\n", bigoRoomId)
		return fmt.Errorf("already monitoring room %s", bigoRoomId)
	}

	// Create browser
	fmt.Printf("[App] Creating headless browser for Bigo room: %s\n", bigoRoomId)
	ctx, cancel, err := a.browserMgr.CreateBrowser(bigoRoomId)
	if err != nil {
		fmt.Printf("[App] ERROR: Failed to create browser: %v\n", err)
		return err
	}
	fmt.Printf("[App] ✓ Browser created successfully\n")

	// Create listener
	bigoListener := listener.NewBigoListener(bigoRoomId, ctx)
	fmt.Printf("[App] BigoListener created for room: %s\n", bigoRoomId)
	
	// Enable debug mode to capture ALL raw WebSocket frames
	debugFilePath := fmt.Sprintf("./debug_frames/room_%s.log", bigoRoomId)
	if err := bigoListener.EnableDebugMode(debugFilePath); err != nil {
		fmt.Printf("[App] WARNING: Could not enable debug mode: %v\n", err)
	} else {
		fmt.Printf("[App] ✓ Debug mode enabled - ALL frames will be saved to: %s\n", debugFilePath)
	}

	// Setup gift handler (ENHANCED with complete payload)
	bigoListener.OnGift(func(gift listener.Gift) {
		fmt.Printf("[App] 🎁 GIFT RECEIVED: %s (%d diamonds) from %s in room %s\n",
			gift.GiftName, gift.Diamonds, gift.SenderName, bigoRoomId)

		// Log activity
		if err := a.logger.LogGift(bigoRoomId, gift.SenderName, gift.GiftName, gift.Diamonds); err != nil {
			fmt.Printf("[App] ERROR: Failed to log gift: %v\n", err)
		} else {
			fmt.Printf("[App] ✓ Gift logged to file\n")
		}

		// Send to BB-Core with COMPLETE payload
		if a.stompClient != nil {
			payload := map[string]interface{}{
				"type":           "GIFT",
				"roomId":         roomId,
				"bigoRoomId":     gift.BigoRoomId,
				"senderId":       gift.SenderId,
				"senderName":     gift.SenderName,
				"senderAvatar":   gift.SenderAvatar,
				"senderLevel":    gift.SenderLevel,
				"streamerId":     gift.StreamerId,
				"streamerName":   gift.StreamerName,
				"streamerAvatar": gift.StreamerAvatar,
				"giftId":         gift.GiftId,
				"giftName":       gift.GiftName,
				"giftCount":      gift.GiftCount,
				"diamonds":       gift.Diamonds,
				"giftImageUrl":   gift.GiftImageUrl,
				"timestamp":      gift.Timestamp,
			}

			destination := "/app/room/" + roomId + "/bigo"
			if err := a.stompClient.Publish(destination, payload); err != nil {
				fmt.Printf("[App] ERROR: Failed to forward to BB-Core: %v\n", err)
			} else {
				fmt.Printf("[App] ✓ Gift forwarded to BB-Core: %s\n", destination)
			}
		} else {
			fmt.Println("[App] WARNING: STOMP client not connected, gift not forwarded to BB-Core")
		}
	})

	// Setup chat handler
	bigoListener.OnChat(func(chat listener.BigoChat) {
		fmt.Printf("[App] 💬 CHAT RECEIVED: %s said \"%s\" in room %s\n",
			chat.SenderName, chat.Message, bigoRoomId)

		if a.stompClient != nil {
			payload := map[string]interface{}{
				"type":         "CHAT",
				"roomId":       roomId,
				"bigoRoomId":   chat.BigoRoomId,
				"senderId":     chat.SenderId,
				"senderName":   chat.SenderName,
				"senderAvatar": chat.SenderAvatar,
				"senderLevel":  chat.SenderLevel,
				"message":      chat.Message,
				"timestamp":    chat.Timestamp,
			}

			destination := "/app/room/" + roomId + "/bigo"
			if err := a.stompClient.Publish(destination, payload); err != nil {
				fmt.Printf("[App] ERROR: Failed to publish chat: %v\n", err)
			} else {
				fmt.Printf("[App] ✓ Chat forwarded to BB-Core: %s\n", destination)
			}
		} else {
			fmt.Println("[App] WARNING: STOMP client not connected, chat not forwarded to BB-Core")
		}
	})

	// Start listening
	fmt.Printf("[App] Starting WebSocket listener for room: %s\n", bigoRoomId)
	if err := bigoListener.Start(); err != nil {
		fmt.Printf("[App] ERROR: Failed to start listener: %v\n", err)
		cancel()
		return err
	}

	a.listeners[bigoRoomId] = bigoListener
	fmt.Printf("[App] ✓ Successfully added streamer: %s (monitoring active)\n", bigoRoomId)
	return nil
}

// RemoveStreamer stops monitoring a streamer
func (a *App) RemoveStreamer(bigoRoomId string) error {
	fmt.Printf("[App] RemoveStreamer called for room: %s\n", bigoRoomId)
	
	a.mutex.Lock()
	defer a.mutex.Unlock()

	if _, exists := a.listeners[bigoRoomId]; !exists {
		fmt.Printf("[App] ERROR: Not monitoring room %s\n", bigoRoomId)
		return fmt.Errorf("not monitoring room %s", bigoRoomId)
	}

	delete(a.listeners, bigoRoomId)
	// Browser cleanup handled by context cancel
	
	fmt.Printf("[App] ✓ Stopped monitoring room: %s\n", bigoRoomId)
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

// StartPKSession starts a complete PK session with BB-Core integration
func (a *App) StartPKSession(bbCoreUrl, authToken, roomId string) error {
	fmt.Printf("[App] Starting PK session for room: %s\n", roomId)

	// Generate device hash
	deviceHash, err := fingerprint.GenerateDeviceHash()
	if err != nil {
		return fmt.Errorf("device fingerprint: %w", err)
	}
	a.deviceHash = deviceHash
	fmt.Printf("[App] Device hash: %s\n", deviceHash)

	// Initialize API client
	apiClient := api.NewClient(bbCoreUrl, authToken)
	fmt.Printf("[App] API client initialized\n")

	// Initialize session manager
	a.session = session.NewManager()
	a.session.Initialize(apiClient, deviceHash)
	fmt.Printf("[App] Session manager initialized\n")

	// Start session (fetches config, calls BB-Core API)
	if err := a.session.Start(roomId); err != nil {
		return fmt.Errorf("session start: %w", err)
	}

	// Connect to STOMP (extract from bbCoreUrl or use WebSocket endpoint)
	// For now, assume STOMP is on same host with /ws path
	stompUrl := bbCoreUrl + "/ws"
	if err := a.ConnectToCore(stompUrl, authToken, ""); err != nil {
		a.session.Stop("STOMP_FAILED")
		return fmt.Errorf("STOMP connect: %w", err)
	}

	// Get all Bigo rooms from config
	cfg := a.session.GetConfig()
	bigoRooms := cfg.GetAllBigoRoomIds()
	fmt.Printf("[App] Starting %d Bigo listeners from config\n", len(bigoRooms))

	// Start browsers for all streamers
	for _, bigoRoomId := range bigoRooms {
		// Use addBigoListener helper (we'll create this)
		if err := a.addBigoListenerForSession(bigoRoomId, roomId); err != nil {
			fmt.Printf("[App] ERROR: Failed to start listener for %s: %v\n", bigoRoomId, err)
			continue
		}
	}

	// Start heartbeat
	a.heartbeat = session.NewHeartbeat(a.session, apiClient, roomId, 0)
	a.heartbeat.Start()
	fmt.Printf("[App] Heartbeat service started\n")

	fmt.Printf("[App] ✓ PK session started successfully\n")
	return nil
}

// StopPKSession stops the current PK session
func (a *App) StopPKSession(reason string) error {
	fmt.Printf("[App] Stopping PK session: %s\n", reason)

	// Stop heartbeat
	if a.heartbeat != nil {
		a.heartbeat.Stop()
		a.heartbeat = nil
		fmt.Printf("[App] Heartbeat stopped\n")
	}

	// Stop all browsers
	a.mutex.Lock()
	for bigoRoomId := range a.listeners {
		fmt.Printf("[App] Stopping listener for room: %s\n", bigoRoomId)
	}
	a.listeners = make(map[string]*listener.BigoListener)
	a.mutex.Unlock()

	// Stop session at BB-Core
	if a.session != nil {
		if err := a.session.Stop(reason); err != nil {
			fmt.Printf("[App] WARNING: Session stop failed: %v\n", err)
		}
		a.session = nil
	}

	// Disconnect STOMP
	if a.stompClient != nil {
		a.stompClient.Disconnect()
		a.stompClient = nil
		fmt.Printf("[App] STOMP disconnected\n")
	}

	fmt.Printf("[App] ✓ PK session stopped\n")
	return nil
}

// GetSessionStatus returns current session status
func (a *App) GetSessionStatus() session.Status {
	if a.session == nil {
		return session.Status{IsActive: false}
	}
	return a.session.GetStatus()
}

// addBigoListenerForSession adds a Bigo listener for session-based workflow
func (a *App) addBigoListenerForSession(bigoRoomId, roomId string) error {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	// Create browser
	ctx, _, err := a.browserMgr.CreateBrowser(bigoRoomId)
	if err != nil {
		return err
	}

	// Create listener
	bigoListener := listener.NewBigoListener(bigoRoomId, ctx)

	// Setup gift handler with enhanced payload
	bigoListener.OnGift(func(gift listener.Gift) {
		// Log activity
		a.logger.LogGift(gift.BigoRoomId, gift.SenderName, gift.GiftName, gift.Diamonds)

		// Send to BB-Core with COMPLETE payload
		if a.stompClient != nil && a.session != nil {
			payload := map[string]interface{}{
				"type":           "GIFT",
				"roomId":         a.session.GetStatus().RoomId,
				"bigoRoomId":     gift.BigoRoomId,
				"senderId":       gift.SenderId,
				"senderName":     gift.SenderName,
				"senderAvatar":   gift.SenderAvatar,
				"senderLevel":    gift.SenderLevel,
				"streamerId":     gift.StreamerId,
				"streamerName":   gift.StreamerName,
				"streamerAvatar": gift.StreamerAvatar,
				"giftId":         gift.GiftId,
				"giftName":       gift.GiftName,
				"giftCount":      gift.GiftCount,
				"diamonds":       gift.Diamonds,
				"giftImageUrl":   gift.GiftImageUrl,
				"timestamp":      gift.Timestamp,
				"deviceHash":     a.deviceHash,
			}

			destination := "/app/room/" + a.session.GetStatus().RoomId + "/bigo"
			a.stompClient.Publish(destination, payload)
		}

		// Update session connection status
		if a.session != nil {
			a.session.UpdateConnectionStatus(gift.BigoRoomId, "CONNECTED", "", bigoListener.GetStats()["frameCount"].(int64))
		}
	})

	// Setup chat handler
	bigoListener.OnChat(func(chat listener.BigoChat) {
		if a.stompClient != nil && a.session != nil {
			payload := map[string]interface{}{
				"type":         "CHAT",
				"roomId":       a.session.GetStatus().RoomId,
				"bigoRoomId":   chat.BigoRoomId,
				"senderId":     chat.SenderId,
				"senderName":   chat.SenderName,
				"senderAvatar": chat.SenderAvatar,
				"senderLevel":  chat.SenderLevel,
				"message":      chat.Message,
				"timestamp":    chat.Timestamp,
				"deviceHash":   a.deviceHash,
			}

			destination := "/app/room/" + a.session.GetStatus().RoomId + "/bigo"
			a.stompClient.Publish(destination, payload)
		}
	})

	// Start listening
	if err := bigoListener.Start(); err != nil {
		return err
	}

	a.listeners[bigoRoomId] = bigoListener
	return nil
}

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
