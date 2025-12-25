package listener

import (
	"sync"
)

// FrameHandler handles WebSocket frames
type FrameHandler func(data string)

// WebSocketListener listens to WebSocket frames
type WebSocketListener struct {
	handlers []FrameHandler
	mutex    sync.RWMutex
}

// NewWebSocketListener creates new listener
func NewWebSocketListener() *WebSocketListener {
	return &WebSocketListener{
		handlers: make([]FrameHandler, 0),
	}
}

// OnFrame registers a frame handler
func (w *WebSocketListener) OnFrame(handler FrameHandler) {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	w.handlers = append(w.handlers, handler)
}

// HandleFrame processes a WebSocket frame
func (w *WebSocketListener) HandleFrame(data string) {
	w.mutex.RLock()
	defer w.mutex.RUnlock()

	for _, handler := range w.handlers {
		handler(data)
	}
}
