package session

import (
	"fmt"
	"sync"
	"time"

	"bbapp/internal/api"
)

type Heartbeat struct {
	manager  *Manager
	apiClient *api.Client
	roomId    string
	interval  time.Duration
	ticker    *time.Ticker
	stopChan  chan struct{}
	mutex     sync.Mutex
	running   bool
}

func NewHeartbeat(manager *Manager, apiClient *api.Client, roomId string, interval time.Duration) *Heartbeat {
	if interval == 0 {
		interval = 30 * time.Second
	}

	return &Heartbeat{
		manager:   manager,
		apiClient: apiClient,
		roomId:    roomId,
		interval:  interval,
		stopChan:  make(chan struct{}),
	}
}

func (h *Heartbeat) Start() {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	if h.running {
		return
	}

	h.running = true
	h.ticker = time.NewTicker(h.interval)

	go h.run()

	fmt.Printf("[Heartbeat] Started (interval: %s)\n", h.interval)
}

func (h *Heartbeat) Stop() {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	if !h.running {
		return
	}

	h.running = false
	h.ticker.Stop()
	close(h.stopChan)

	fmt.Printf("[Heartbeat] Stopped\n")
}

func (h *Heartbeat) run() {
	for {
		select {
		case <-h.ticker.C:
			h.sendStatus()
		case <-h.stopChan:
			return
		}
	}
}

func (h *Heartbeat) sendStatus() {
	if h.apiClient == nil {
		return
	}

	status := h.manager.GetStatus()

	req := api.HeartbeatRequest{
		Connections: status.Connections,
	}

	if err := h.apiClient.SendHeartbeat(req); err != nil {
		fmt.Printf("[Heartbeat] ERROR: Failed to send heartbeat: %v\n", err)
	} else {
		fmt.Printf("[Heartbeat] ✓ Sent (connections: %d)\n", len(status.Connections))
	}
}
