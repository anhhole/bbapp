package main

import (
	"context"
	"fmt"
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
	a.browserMgr = browser.NewManager()

	// Initialize logger
	log, err := logger.NewLogger("./logs")
	if err != nil {
		panic(err)
	}
	a.logger = log
}

// shutdown is called on app termination
func (a *App) shutdown(ctx context.Context) {
	if a.stompClient != nil {
		a.stompClient.Disconnect()
	}
	if a.logger != nil {
		a.logger.Close()
	}
}

// ConnectToCore connects to BB-Core STOMP
func (a *App) ConnectToCore(url, username, password string) error {
	client, err := stomp.NewClient(url, username, password)
	if err != nil {
		return fmt.Errorf("connection failed: %w", err)
	}

	a.stompClient = client
	return nil
}

// AddStreamer adds Bigo streamer to monitor
func (a *App) AddStreamer(bigoRoomId, teamId, roomId string) error {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	// Check if already exists
	if _, exists := a.listeners[bigoRoomId]; exists {
		return fmt.Errorf("already monitoring room %s", bigoRoomId)
	}

	// Create browser
	ctx, cancel, err := a.browserMgr.CreateBrowser(bigoRoomId)
	if err != nil {
		return err
	}

	// Create listener
	bigoListener := listener.NewBigoListener(bigoRoomId, ctx)

	// Setup gift handler
	bigoListener.OnGift(func(gift listener.Gift) {
		// Log activity
		a.logger.LogGift(bigoRoomId, gift.Nickname, gift.GiftName, gift.GiftValue)

		// Send to BB-Core
		if a.stompClient != nil {
			payload := map[string]interface{}{
				"type":      "GIFT",
				"bigoId":    gift.BigoUid,
				"nickname":  gift.Nickname,
				"giftName":  gift.GiftName,
				"giftValue": gift.GiftValue,
			}
			a.stompClient.Publish("/app/room/"+roomId+"/bigo", payload)
		}
	})

	// Start listening
	if err := bigoListener.Start(); err != nil {
		cancel()
		return err
	}

	a.listeners[bigoRoomId] = bigoListener
	return nil
}
