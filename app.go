package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"
	"sync"

	"bbapp/internal/api"
	"bbapp/internal/browser"
	"bbapp/internal/fingerprint"
	"bbapp/internal/listener"
	"bbapp/internal/logger"
	"bbapp/internal/overlayserver"
	"bbapp/internal/profile"
	"bbapp/internal/session"
	"bbapp/internal/stomp"

	"github.com/joho/godotenv"
)

// App struct
type App struct {
	ctx            context.Context
	browserMgr     *browser.Manager
	stompClient    *stomp.Client
	logger         *logger.Logger
	listeners      map[string]*listener.BigoListener
	cancels        map[string]context.CancelFunc
	session        *session.Manager
	heartbeat      *session.Heartbeat
	deviceHash     string
	mutex          sync.RWMutex
	overlayServer  *overlayserver.Server
	bbCoreURL      string
	apiClient      *api.Client
	profileManager *profile.Manager
	giftLibrary    []api.GiftDefinition
	currentConfig  *api.Config // Cache for internal use
}

// NewApp creates new App
func NewApp() *App {
	return &App{
		listeners: make(map[string]*listener.BigoListener),
		cancels:   make(map[string]context.CancelFunc),
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

	// Initialize profile manager
	profileDir := "./data/profiles"
	if err := os.MkdirAll(profileDir, 0755); err != nil {
		fmt.Printf("[App] WARNING: Could not create profiles directory: %v\n", err)
	}
	a.profileManager = profile.NewManager(profileDir)
	fmt.Println("[App] Profile manager initialized (data stored in ./data/profiles)")

	// Create debug frames directory
	if err := os.MkdirAll("./debug_frames", 0755); err != nil {
		fmt.Printf("[App] WARNING: Could not create debug_frames directory: %v\n", err)
	} else {
		fmt.Println("[App] Debug frames directory ready (./debug_frames)")
	}

	// Load Gift Library
	a.giftLibrary = a.loadGiftLibraryFromFile()
	if len(a.giftLibrary) == 0 {
		fmt.Printf("[App] WARNING: Gift library is empty or failed to load\n")
	} else {
		fmt.Printf("[App] Gift library loaded in startup (%d items)\n", len(a.giftLibrary))
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

	// Ensure all browser instances are closed
	fmt.Println("[App] Stopping all sessions and browsers...")
	if err := a.StopPKSession("App shutdown"); err != nil {
		fmt.Printf("[App] WARNING: Error stopping sessions during shutdown: %v\n", err)
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

	// Store cancel function for cleanup
	a.cancels[bigoRoomId] = cancel

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

		// Check library for diamond override/lookup
		// Bigo often sends 0 for specific gifts, or we want to override values
		for _, def := range a.giftLibrary {
			// Match by ID if possible, otherwise by Name
			if (def.ID != "" && def.ID == gift.GiftId) || (strings.EqualFold(def.Name, gift.GiftName)) {
				// Only override if the library has a value (and potentially if the event value is 0 or different)
				// For now, let's treat the Library as the source of truth if it has a non-zero value
				if def.Diamonds > 0 {
					// fmt.Printf("[App] Overriding gift value for %s: %d -> %d\n", gift.GiftName, gift.Diamonds, def.Diamonds)
					gift.Diamonds = int64(def.Diamonds)
				}
				break
			}
		}

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
				"deviceHash":     a.deviceHash,
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
				"deviceHash":   a.deviceHash,
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
	if _, err := bigoListener.Start(); err != nil {
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

	// Cancel browser context to clean up
	if cancel, exists := a.cancels[bigoRoomId]; exists && cancel != nil {
		cancel()
		delete(a.cancels, bigoRoomId)
	}

	delete(a.listeners, bigoRoomId)

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
// Takes roomId and config from the wizard (config already fetched and validated)
func (a *App) StartPKSession(bbCoreUrl, authToken, roomId string, cfg api.Config, durationMinutes int) error {
	fmt.Printf("[App] Starting PK session for room: %s (duration: %dm)\n", roomId, durationMinutes)

	// Generate device hash if not already set
	if a.deviceHash == "" {
		deviceHash, err := fingerprint.GenerateDeviceHash()
		if err != nil {
			return fmt.Errorf("device fingerprint: %w", err)
		}
		a.deviceHash = deviceHash
		fmt.Printf("[App] Device hash: %s\n", deviceHash)
	}

	// Initialize API client if not already set
	if a.apiClient == nil {
		a.apiClient = api.NewClient(bbCoreUrl, authToken)
		a.bbCoreURL = bbCoreUrl
		fmt.Printf("[App] API client initialized\n")
	}

	// Initialize session manager
	a.session = session.NewManager()
	a.session.Initialize(a.apiClient, a.deviceHash)

	// Force reload Gift Library to ensure freshness
	fmt.Println("[App] Reloading Gift Library from file before session start...")
	a.giftLibrary = a.loadGiftLibraryFromFile()

	// Inject Gift Library
	fmt.Printf("[App] Injecting Gift Library to session (Size: %d)\n", len(a.giftLibrary))
	if len(a.giftLibrary) == 0 {
		fmt.Println("[App] WARNING: Injecting EMPTY gift library! Overrides will not work.")
	}
	a.session.SetGiftLibrary(a.giftLibrary)

	fmt.Printf("[App] Session manager initialized\n")

	// Start session (validates trial, connects STOMP, starts heartbeat)
	// Enhanced session manager now handles everything
	if err := a.session.Start(roomId, &cfg, bbCoreUrl, authToken, durationMinutes); err != nil {
		return fmt.Errorf("session start failed: %w", err)
	}

	// Get all Bigo rooms from config
	cfgMgr := a.session.GetConfig()
	bigoRooms := cfgMgr.GetAllBigoRoomIds()
	fmt.Printf("[App] Starting %d Bigo listeners from config\n", len(bigoRooms))

	// Start browsers for all streamers
	for _, bigoRoomId := range bigoRooms {
		if err := a.addBigoListenerForSession(bigoRoomId, roomId); err != nil {
			fmt.Printf("[App] ERROR: Failed to start listener for %s: %v\n", bigoRoomId, err)
			// Continue with other listeners even if one fails
			continue
		}
	}

	fmt.Printf("[App] ✓✓✓ PK session started successfully with %d active listeners\n", len(a.listeners))
	return nil
}

// StopPKSession stops the current PK session
func (a *App) StopPKSession(reason string) error {
	fmt.Printf("[App] Stopping PK session: %s\n", reason)

	// Stop all browser listeners
	a.mutex.Lock()
	for bigoRoomId, cancel := range a.cancels {
		fmt.Printf("[App] Stopping browser for room: %s\n", bigoRoomId)
		if cancel != nil {
			cancel()
		}
	}
	a.listeners = make(map[string]*listener.BigoListener)
	a.cancels = make(map[string]context.CancelFunc)
	a.mutex.Unlock()
	fmt.Printf("[App] ✓ All browsers stopped\n")

	// Stop session (handles heartbeat, STOMP, BB-Core notification)
	if a.session != nil {
		if err := a.session.Stop(reason); err != nil {
			fmt.Printf("[App] WARNING: Session stop failed: %v\n", err)
			// Continue cleanup even if session stop fails
		}
		a.session = nil
	}

	fmt.Printf("[App] ✓✓✓ PK session stopped successfully\n")
	return nil
}

// ResetSession completely wipes the current session manager and creates a new one
func (a *App) ResetSession() error {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	fmt.Println("[App] Force resetting session manager...")
	if a.session != nil {
		a.session.Stop("Force reset")
	}

	a.session = session.NewManager()
	if a.apiClient != nil {
		a.session.Initialize(a.apiClient, a.deviceHash)
	}
	fmt.Println("[App] ✓ Session manager reset")
	return nil
}

// ensureSessionManager is a safety check to ensure session manager is initialized
func (a *App) ensureSessionManager() error {
	// Always inject the latest library to be safe, even if session exists
	if a.session != nil {
		if len(a.giftLibrary) > 0 {
			a.session.SetGiftLibrary(a.giftLibrary)
		}
		return nil
	}

	// Try to initialize using existing client
	if a.apiClient == nil {
		return fmt.Errorf("session manager not initialized and API client missing - please login again")
	}

	fmt.Printf("[App] Safety initializing session manager...\n")

	// Generate device hash if not set
	if a.deviceHash == "" {
		deviceHash, err := fingerprint.GenerateDeviceHash()
		if err != nil {
			fmt.Printf("[App] ERROR: Safety initialization failed to generate device hash: %v\n", err)
		} else {
			a.deviceHash = deviceHash
		}
	}

	a.session = session.NewManager()
	a.session.Initialize(a.apiClient, a.deviceHash)
	// Inject Gift Library
	fmt.Printf("[App] Injecting Gift Library to safety session (Size: %d)\n", len(a.giftLibrary))
	a.session.SetGiftLibrary(a.giftLibrary)

	// Subscribe to internal listeners for SSE broadcasting
	a.session.SubscribeOnGift(func(event interface{}) {
		gift, ok := event.(listener.BigoGift)
		if !ok {
			return
		}

		// Log minimal
		fmt.Printf("[App] Internal Gift Event: %s x%d (Room: %s)\n", gift.GiftName, gift.GiftCount, gift.BigoRoomId)

		// Resolve TeamId using a.currentConfig (This ensures consistency with Overlay)
		var teamId string
		a.mutex.RLock()
		if a.currentConfig != nil {
			// 1. Check Binding Gifts first (Priority)
			if gift.GiftName != "" {
				for _, team := range a.currentConfig.Teams {
					if strings.EqualFold(team.BindingGift, gift.GiftName) {
						teamId = team.TeamId
						// fmt.Printf("[App] Resolved Team via Binding Gift: %s -> %s\n", gift.GiftName, team.TeamId)
						break
					}
				}
			}

			// 2. Check Streamers if not found
			if teamId == "" {
				for _, team := range a.currentConfig.Teams {
					for _, s := range team.Streamers {
						// Check BigoRoomId (numeric) or BigoId (username/id)
						if (s.BigoRoomId != "" && s.BigoRoomId == gift.BigoRoomId) ||
							(s.BigoId != "" && strings.EqualFold(s.BigoId, gift.BigoRoomId)) {
							teamId = team.TeamId
							break
						}
					}
					if teamId != "" {
						break
					}
				}
			}
		}
		a.mutex.RUnlock()

		if teamId != "" {
			fmt.Printf("[App] SSE Broadcast: Resolved %s -> %s\n", gift.BigoRoomId, teamId)
		} else {
			// Debug: why missing?
			fmt.Printf("[App] SSE Broadcast WARNING: No TeamID for %s\n", gift.BigoRoomId)
		}

		// Construct payload for Overlay
		// Using map to be flexible and match JS expectations
		payload := map[string]interface{}{
			"type":           "GIFT",
			"roomId":         "INTERNAL", // Or get from session status if needed
			"teamId":         teamId,     // CRITICAL: Inject Resolved ID
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

		if a.overlayServer != nil {
			a.overlayServer.BroadcastEvent(payload)
		}
	})

	fmt.Printf("[App] Session manager safety initialized\n")
	return nil
}

// GetSessionStatus returns current session status
func (a *App) GetSessionStatus() session.Status {
	if a.session == nil {
		return session.Status{IsActive: false}
	}
	return a.session.GetStatus()
}

// StartBigoListener starts only the Bigo listener session
func (a *App) StartBigoListener(cfg api.Config) error {
	fmt.Printf("[App] StartBigoListener called. Session address: %p\n", a.session)
	if err := a.ensureSessionManager(); err != nil {
		return err
	}

	// Ensure library is set before starting
	if len(a.giftLibrary) > 0 && a.session != nil {
		a.session.SetGiftLibrary(a.giftLibrary)
	}

	fmt.Printf("[App] Using session manager at %p\n", a.session)
	return a.session.StartBigoListener(&cfg)
}

// StopBigoListener stops only the Bigo listener session
func (a *App) StopBigoListener() error {
	if err := a.ensureSessionManager(); err != nil {
		return err
	}
	return a.session.StopBigoListener()
}

// StartBBCoreStream starts only the BB-Core streaming session
func (a *App) StartBBCoreStream(roomId string, cfg api.Config, durationMinutes int) error {
	if err := a.ensureSessionManager(); err != nil {
		return err
	}

	// Ensure config has correct room ID
	cfg.RoomId = roomId

	// Save configuration to ensure backend has the latest teams data
	// This fixes the "Room has 0 teams" error
	if err := a.SaveBBAppConfig(roomId, cfg); err != nil {
		fmt.Printf("[App] WARNING: Failed to save config before stream start: %v\n", err)
		// We could return error here, but we'll try to proceed in case it was a network glitch
		// and the config was already saved previously.
		// return fmt.Errorf("failed to save config: %w", err)
	} else {
		fmt.Printf("[App] ✓ Config saved to BB-Core before stream start\n")
	}

	// Get BB-Core URL and access token
	bbCoreURL := a.bbCoreURL
	accessToken := ""
	if a.apiClient != nil {
		accessToken = a.apiClient.GetAccessToken()
	}

	return a.session.StartBBCoreStream(roomId, &cfg, bbCoreURL, accessToken, durationMinutes)
}

// StopBBCoreStream stops only the BB-Core streaming session
func (a *App) StopBBCoreStream(reason string) error {
	if err := a.ensureSessionManager(); err != nil {
		return err
	}
	return a.session.StopBBCoreStream(reason)
}

// GetBigoListenerStatus returns the status of the Bigo listener session
func (a *App) GetBigoListenerStatus() session.BigoListenerStatus {
	if a.session == nil {
		return session.BigoListenerStatus{}
	}
	status := a.session.GetBigoListenerStatus()
	// fmt.Printf("[App] GetBigoListenerStatus from %p: active=%v, idols=%d\n", a.session, status.IsActive, status.TotalIdols)
	return status
}

// GetBBCoreStreamStatus returns the status of the BB-Core stream session
func (a *App) GetBBCoreStreamStatus() session.BBCoreStreamStatus {
	if a.session == nil {
		return session.BBCoreStreamStatus{}
	}
	return a.session.GetBBCoreStreamStatus()
}

// addBigoListenerForSession adds a Bigo listener for session-based workflow
func (a *App) addBigoListenerForSession(bigoRoomId, roomId string) error {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	// Create browser
	ctx, cancel, err := a.browserMgr.CreateBrowser(bigoRoomId)
	if err != nil {
		return err
	}

	// Store cancel function for cleanup
	a.cancels[bigoRoomId] = cancel

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

			// Broadcast directly to overlay via SSE (Local, Robust)
			// This bypasses STOMP broker issues completely
			if a.overlayServer != nil {
				fmt.Printf("[App] Broadcasting GIFT event via SSE. Payload type: %v\n", payload["type"])
				// We attach the teamId if possible, but the overlay can resolve it.
				// For consistency with STOMP, we send the same payload.
				a.overlayServer.BroadcastEvent(payload)
			} else {
				fmt.Printf("[App] WARNING: OverlayServer is nil, cannot broadcast SSE event\n")
			}
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
	if _, err := bigoListener.Start(); err != nil {
		return err
	}

	a.listeners[bigoRoomId] = bigoListener
	return nil
}

// GetOverlayURL generates the overlay URL for OBS Browser Source.
// Returns the complete URL with scene, roomId, bbCoreUrl, and token as query parameters.
// Returns empty string if inputs are invalid or overlay server is unavailable.
func (a *App) GetOverlayURL(scene, roomId, token string) string {
	// Validate inputs
	if strings.TrimSpace(scene) == "" || strings.TrimSpace(roomId) == "" || strings.TrimSpace(token) == "" {
		return ""
	}

	// Check if overlay server is available
	if a.overlayServer == nil {
		return ""
	}

	baseURL := a.overlayServer.GetURL()

	// Use url.Values for safe encoding
	params := url.Values{}
	params.Add("scene", scene)
	params.Add("roomId", roomId)
	params.Add("bbCoreUrl", a.bbCoreURL)
	params.Add("token", token)

	return fmt.Sprintf("%s/overlay?%s", baseURL, params.Encode())
}

// GetBBCoreURL returns the configured BB-Core base URL from environment.
// Defaults to "http://localhost:8080" if not set in .env file.
func (a *App) GetBBCoreURL() string {
	return a.bbCoreURL
}

// InitializeBBCoreClient initializes the API client for BB-Core communication.
// This must be called before GetBBAppConfig.
func (a *App) InitializeBBCoreClient(bbCoreUrl, authToken string) error {
	a.apiClient = api.NewClient(bbCoreUrl, authToken)
	a.bbCoreURL = bbCoreUrl
	fmt.Printf("[App] BB-Core API client initialized: %s\n", bbCoreUrl)

	// Initialize session manager if not already set by calling ensureSessionManager
	// This ensures the global subscription to events is properly registered
	if err := a.ensureSessionManager(); err != nil {
		fmt.Printf("[App] ERROR: Failed to ensure session manager in InitializeBBCoreClient: %v\n", err)
	} else {
		fmt.Printf("[App] Session manager initialized via ensureSessionManager\n")
	}

	return nil
}

// GetBBAppConfig fetches the PK configuration for a specific room from BB-Core.
// Returns the config or an error if the room doesn't exist (404) or other errors occur.
func (a *App) GetBBAppConfig(roomId string) (*api.Config, error) {
	if a.apiClient == nil {
		return nil, fmt.Errorf("API client not initialized - call InitializeBBCoreClient first")
	}

	fmt.Printf("[App] Fetching config for room: %s\n", roomId)
	config, err := a.apiClient.GetConfig(roomId)
	if err != nil {
		fmt.Printf("[App] ERROR: Failed to fetch config: %v\n", err)
		return nil, err
	}

	// Update local cache and overlay server with the authoritative config
	a.mutex.Lock()
	a.currentConfig = config
	a.mutex.Unlock()

	if a.overlayServer != nil {
		fmt.Printf("[App] Syncing local overlay server with authoritative config\n")
		a.overlayServer.SetConfig(config)

		// Optional: Broadcast generic update if needed, but Wizard serves as main driver here
		// a.overlayServer.BroadcastEvent(...)
	}

	fmt.Printf("[App] ✓ Config fetched and synced successfully for room: %s\n", roomId)
	return config, nil
}

// SaveBBAppConfig saves configuration to BB-Core
func (a *App) SaveBBAppConfig(roomId string, config api.Config) error {
	if a.apiClient == nil {
		return fmt.Errorf("not connected to BB-Core")
	}

	// Enrich config with Binding Gift Images from Library
	if len(a.giftLibrary) > 0 {
		for i, team := range config.Teams {
			// Only populate if not already set or if we want to ensure it's correct
			// Let's look up by name
			if team.BindingGift != "" {
				for _, gift := range a.giftLibrary {
					if strings.EqualFold(gift.Name, team.BindingGift) {
						if gift.Image != "" {
							config.Teams[i].BindingGiftImage = gift.Image
							// fmt.Printf("[App] Populated BindingGiftImage for %s: %s\n", team.Name, gift.Image)
						}
						break
					}
				}
			}
		}
	}

	if err := a.apiClient.SaveConfig(roomId, &config); err != nil {
		return err
	}

	// CRITICAL: Refetch config from Core immediately to ensure we have the Authoritative IDs.
	// The Core might have generated different IDs or modified the data.
	// We must align with Core to avoid ID mismatches.
	authoritativeConfig, err := a.GetBBAppConfig(roomId)
	if err != nil {
		fmt.Printf("[App] WARNING: Failed to refetch config after save: %v. Using local version (risk of desync).\n", err)
	} else {
		// Use the authoritative config
		config = *authoritativeConfig
		fmt.Printf("[App] ✓ Refetched authoritative config from Core. Teams: %d\n", len(config.Teams))
	}

	// Broadcast config update to overlay
	if a.stompClient != nil {
		fmt.Printf("[App] Broadcasting config update for room: %s\n", roomId)
		fmt.Printf("[App] Overlay Settings: %+v\n", config.OverlaySettings)
		// We send the full config object
		// Note: We normally use /app/ prefix for client-to-server, but here we are acting as a client (the app)
		// sending TO the broker. If the broker is simple relay, we might need to send to /topic/ directly
		// or if we are using the BB-Core relay, we use the standard prefix.
		// Assuming BB-Core relays /topic/ subscriptions.
		// Let's rely on the direct topic broadcast if we have rights, or use a relay endpoint.
		// Looking at other publishes: destination := "/app/room/" + roomId + "/bigo"
		// This suggests we send to an application destination.
		// Let's try sending to a config topic.

		// If BB-Core doesn't have a specific handler for config updates, we might have to use a generic one
		// or if we are just relaying to other clients (overlays).
		// SAFEST BADGE: Use a known working channel or a new one if supported.
		// Let's assume we can publish to /topic/room/{roomId}/config if the broker allows.
		// If not, we might need a specific app endpoint.
		// Based on `bigo` gift logic: a.stompClient.Publish("/app/room/" + roomId + "/bigo", payload)
		// This implies there is a backend handler at @MessageMapping("/room/{roomId}/bigo").
		// If we don't have a backend handler for config, we can't "Push" unless the broker blindly relays /topic/.
		// Standard Spring Boot STOMP usually restricts /topic/ publish to server-side only unless configured otherwise.
		// BUT, if `OverlayApp` subscribes to `/topic/...`, we need the server to send it there.

		// Wait, if I cannot change the backend (Java/Spring?) code easily to add a handler,
		// I am stuck with existing endpoints or Polling.
		// The user said "no need to have auto update after 3s".
		// IF I cannot rely on STOMP relay, I have to assume the backend has a general "broadcast" or "forward" capability.

		// HYPOTHESIS: The backend likely has a generic relay or we can reuse `script` or `activity` channels?
		// Better approach: Since I modified the backend for `gifts.json` (server.go), I cannot modify the JAVA backend of BB-Core.
		// I am finding `SaveBBAppConfig` calls `a.apiClient.SaveConfig` (HTTP).
		// The HTTP endpoint saves to DB. Does IT broadcast? If it did, `OverlayApp` would just need to listen.
		// The fact it doesn't means it probably doesn't.

		// WORKAROUND: Client-Side Broadcasting.
		// The `OverlayApp` is a client. `SessionControlPanel` is a client (inside Wails).
		// They share the same Wails `App` backend instance in `app.go`.
		// But `OverlayApp` is running in a separate browser window (OBS).
		// Wails `App` (Go) is the bridge.
		// `OverlayApp` connects to STOMP (BB-Core). `App` (Go) connects to STOMP (BB-Core).
		// Use `App` (Go) to send a message to STOMP that `OverlayApp` listens to.

		// I will try publishing to `/app/room/{roomId}/broadcast` if it exists, or just try `/topic/room/{roomId}/config` directly.
		// Many configs allow client publish to /topic. Let's try `/topic/room/{roomId}/config`.
		destination := "/topic/room/" + roomId + "/config"
		if err := a.stompClient.Publish(destination, config); err != nil {
			fmt.Printf("[App] WARNING: Failed to broadcast config: %v\n", err)
		}
	}

	// Update local cache
	a.mutex.Lock()
	a.currentConfig = &config
	a.mutex.Unlock()

	// Update local overlay server config (for visual settings persistence)
	if a.overlayServer != nil {
		fmt.Printf("[App] Updating local overlay server config\n")
		a.overlayServer.SetConfig(&config)

		// Broadcast update via SSE to ensure Overlay stays in sync (bypassing flaky STOMP)
		fmt.Printf("[App] Broadcasting CONFIG_UPDATE via SSE\n")
		a.overlayServer.BroadcastEvent(map[string]interface{}{
			"type": "CONFIG_UPDATE",
			"data": config,
		})
	}

	return nil
}

// Profile Management Wails Bindings

// CreateProfile creates a new profile with the given name, room ID, and config
func (a *App) CreateProfile(name, roomID string, config api.Config) (*profile.Profile, error) {
	if a.profileManager == nil {
		return nil, fmt.Errorf("profile manager not initialized")
	}
	return a.profileManager.CreateProfile(name, roomID, config)
}

// LoadProfile loads a profile by ID and updates its lastUsedAt timestamp
func (a *App) LoadProfile(id string) (*profile.Profile, error) {
	if a.profileManager == nil {
		return nil, fmt.Errorf("profile manager not initialized")
	}
	return a.profileManager.LoadProfile(id)
}

// UpdateProfile updates a profile's config
func (a *App) UpdateProfile(id string, config api.Config) (*profile.Profile, error) {
	if a.profileManager == nil {
		return nil, fmt.Errorf("profile manager not initialized")
	}
	return a.profileManager.UpdateProfile(id, config)
}

// UpdateProfileBigoInfo updates a profile's Bigo info
func (a *App) UpdateProfileBigoInfo(id, avatar, nickname string) (*profile.Profile, error) {
	if a.profileManager == nil {
		return nil, fmt.Errorf("profile manager not initialized")
	}
	return a.profileManager.UpdateProfileBigoInfo(id, avatar, nickname)
}

// DeleteProfile deletes a profile by ID
func (a *App) DeleteProfile(id string) error {
	if a.profileManager == nil {
		return fmt.Errorf("profile manager not initialized")
	}
	return a.profileManager.DeleteProfile(id)
}

// ListProfiles returns all profiles sorted by lastUsedAt desc
func (a *App) ListProfiles() []*profile.Profile {
	if a.profileManager == nil {
		return []*profile.Profile{}
	}
	return a.profileManager.ListProfiles()
}

// Authentication Wails Bindings

// Login authenticates a user with username and password
func (a *App) Login(username, password string) (*api.AuthResponse, error) {
	// Create temporary client if not connected
	client := a.apiClient
	if client == nil {
		client = api.NewClient(a.bbCoreURL, "")
	}

	authResp, err := client.Login(username, password)
	if err != nil {
		return nil, err
	}

	// Update API client with new tokens
	if a.apiClient != nil {
		a.apiClient.SetTokens(authResp.AccessToken, authResp.RefreshToken)
	}

	return authResp, nil
}

// Register creates a new user account
func (a *App) Register(username, email, password, agencyName, firstName, lastName string) (*api.AuthResponse, error) {
	// Create temporary client if not connected
	client := a.apiClient
	if client == nil {
		client = api.NewClient(a.bbCoreURL, "")
	}

	authResp, err := client.Register(username, email, password, agencyName, firstName, lastName)
	if err != nil {
		return nil, err
	}

	// Update API client with new tokens
	if a.apiClient != nil {
		a.apiClient.SetTokens(authResp.AccessToken, authResp.RefreshToken)
	}

	return authResp, nil
}

// RefreshAuthToken refreshes the authentication token
func (a *App) RefreshAuthToken(refreshToken string) (*api.AuthResponse, error) {
	client := a.apiClient
	if client == nil {
		client = api.NewClient(a.bbCoreURL, "")
	}

	authResp, err := client.RefreshToken(refreshToken)
	if err != nil {
		return nil, err
	}

	// Update API client with new tokens
	if a.apiClient != nil {
		a.apiClient.SetTokens(authResp.AccessToken, authResp.RefreshToken)
	}

	return authResp, nil
}

// Config and Validation Wails Bindings

// FetchConfig fetches room configuration from BB-Core (alias for GetBBAppConfig for consistency)
func (a *App) FetchConfig(roomID string) (*api.Config, error) {
	return a.GetBBAppConfig(roomID)
}

// ValidateTrial validates streamers for trial accounts
func (a *App) ValidateTrial(streamers []api.ValidateTrialStreamer) (*api.ValidateTrialResponse, error) {
	if a.apiClient == nil {
		return nil, fmt.Errorf("not connected to BB-Core")
	}
	return a.apiClient.ValidateTrial(streamers)
}

// Bigo API Integration

// FetchBigoUser fetches user info from Bigo's official API
func (a *App) FetchBigoUser(bigoId string) (*listener.BigoUserInfo, error) {
	return listener.GetUserInfo(bigoId)
}

// Local Global Idols Persistence

const globalIdolsFile = "global-idols.json"

// FetchGlobalIdols reads the global idols list from a local file
func (a *App) FetchGlobalIdols() ([]api.GlobalIdol, error) {
	fmt.Println("[App] Fetching global idols from local file")

	file, err := os.Open(globalIdolsFile)
	if os.IsNotExist(err) {
		return []api.GlobalIdol{}, nil // Return empty list if file doesn't exist
	}
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var idols []api.GlobalIdol
	if err := json.NewDecoder(file).Decode(&idols); err != nil {
		return nil, fmt.Errorf("failed to decode idols: %v", err)
	}

	return idols, nil
}

// SaveGlobalIdols saves the global idols list to a local file
func (a *App) SaveGlobalIdols(idols []api.GlobalIdol) error {
	fmt.Printf("[App] Saving %d global idols to local file\n", len(idols))

	file, err := os.Create(globalIdolsFile)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(idols); err != nil {
		return fmt.Errorf("failed to encode idols: %v", err)
	}

	return nil
}

// Gift Library Management

// SaveGiftLibrary saves the gift library to disk
func (a *App) SaveGiftLibrary(gifts []api.GiftDefinition) error {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	a.giftLibrary = gifts

	// Update active session if exists
	if a.session != nil {
		fmt.Println("[App] Push updated gift library to active session")
		a.session.SetGiftLibrary(gifts)
	}

	// Ensure data directory exists
	if err := os.MkdirAll("./data", 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(gifts, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile("./data/gifts.json", data, 0644)
}

// GetGiftLibrary returns the current gift library
func (a *App) GetGiftLibrary() []api.GiftDefinition {
	// a.mutex.RLock()
	// defer a.mutex.RUnlock()
	return a.giftLibrary
}

// loadGiftLibraryFromFile loads from disk (internal use)
func (a *App) loadGiftLibraryFromFile() []api.GiftDefinition {
	path := "./data/gifts.json"
	fmt.Printf("[App] Attempting to load gift library from: %s\n", path)

	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Printf("[App] Failed to read gift library file: %v\n", err)
		return nil
	}

	var gifts []api.GiftDefinition
	if err := json.Unmarshal(data, &gifts); err != nil {
		fmt.Printf("[App] Failed to parse gift library JSON: %v\n", err)
		return nil
	}

	fmt.Printf("[App] Successfully loaded %d gifts from file\n", len(gifts))
	return gifts
}
