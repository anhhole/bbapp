package main

import (
	"context"
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
