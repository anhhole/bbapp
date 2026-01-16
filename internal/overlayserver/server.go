package overlayserver

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
)

type Server struct {
	port          int
	mux           *http.ServeMux
	currentConfig interface{}
	configMutex   sync.RWMutex
	clients       map[chan []byte]bool
	clientsMutex  sync.RWMutex
}

func NewServer() (*Server, error) {
	// Find available port in range 3000-3100
	port, err := findAvailablePort(3000, 3100)
	if err != nil {
		return nil, fmt.Errorf("no available ports: %w", err)
	}

	s := &Server{
		port:    port,
		mux:     http.NewServeMux(),
		clients: make(map[chan []byte]bool),
	}
	s.setupRoutes()
	return s, nil
}

func (s *Server) setupRoutes() {
	// Serve React build (frontend/dist)
	fs := http.FileServer(http.Dir("./frontend/dist"))

	// specific handler for overlay to support SPA routing
	s.mux.HandleFunc("/overlay", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./frontend/dist/index.html")
	})

	// Serve data directory (for gifts.json)
	s.mux.Handle("/data/", http.StripPrefix("/data/", http.FileServer(http.Dir("./data"))))

	// Serve current local configuration
	s.mux.HandleFunc("/config", s.handleConfig)

	// Server Sent Events for real-time local updates
	s.mux.HandleFunc("/events", s.handleEvents)

	s.mux.Handle("/", fs)
}

func (s *Server) Start() error {
	addr := fmt.Sprintf(":%d", s.port)
	log.Printf("Overlay server starting on http://localhost%s", addr)
	return http.ListenAndServe(addr, s.mux)
}

func (s *Server) GetURL() string {
	return fmt.Sprintf("http://localhost:%d", s.port)
}

// SetConfig updates the current configuration
func (s *Server) SetConfig(config interface{}) {
	s.configMutex.Lock()
	defer s.configMutex.Unlock()
	s.currentConfig = config
}

// handleConfig serves the current configuration as JSON
func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	// Enable CORS for development
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	s.configMutex.RLock()
	config := s.currentConfig
	s.configMutex.RUnlock()

	if config == nil {
		http.Error(w, "No config available", http.StatusNotFound)
		return
	}

	if err := json.NewEncoder(w).Encode(config); err != nil {
		http.Error(w, "Failed to encode config", http.StatusInternalServerError)
	}
}

// BroadcastEvent sends a payload to all connected SSE clients
func (s *Server) BroadcastEvent(payload interface{}) {
	s.clientsMutex.RLock()
	defer s.clientsMutex.RUnlock()

	log.Printf("[OverlayServer] Broadcasting event to %d clients. Payload: %+v", len(s.clients), payload)

	data, err := json.Marshal(payload)
	if err != nil {
		log.Printf("[OverlayServer] Failed to marshal SSE payload: %v", err)
		return
	}

	for clientChan := range s.clients {
		select {
		case clientChan <- data:
		default:
			log.Printf("[OverlayServer] Client channel blocked, skipping update")
		}
	}
}

// handleEvents manages SSE connections
func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	// CORS and SSE headers
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	clientChan := make(chan []byte, 10)

	s.clientsMutex.Lock()
	s.clients[clientChan] = true
	s.clientsMutex.Unlock()

	log.Printf("SSE Client connected. Total: %d", len(s.clients))

	defer func() {
		s.clientsMutex.Lock()
		delete(s.clients, clientChan)
		s.clientsMutex.Unlock()
		close(clientChan)
		log.Printf("SSE Client disconnected")
	}()

	// Flush immediately to establish connection
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}

	notify := r.Context().Done()

	for {
		select {
		case <-notify:
			return
		case data := <-clientChan:
			fmt.Fprintf(w, "data: %s\n\n", data)
			if flusher, ok := w.(http.Flusher); ok {
				flusher.Flush()
			}
		}
	}
}
