package main

import (
	"context"
	"fmt"
	"os"
	"sync"

	"bbapp/internal/browser"
	"bbapp/internal/listener"
	"bbapp/internal/logger"
	"bbapp/internal/stomp"
)

// App struct
type App struct {
	ctx         context.Context
	browserMgr  *browser.Manager
	stompClient *stomp.Client
	logger      *logger.Logger
	listeners   map[string]*listener.BigoListener
	mutex       sync.RWMutex
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
	
	a.browserMgr = browser.NewManager()
	fmt.Println("[App] Browser manager initialized")

	// Initialize logger
	log, err := logger.NewLogger("./logs")
	if err != nil {
		fmt.Printf("[App] ERROR: Failed to initialize logger: %v\n", err)
		panic(err)
	}
	a.logger = log
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
