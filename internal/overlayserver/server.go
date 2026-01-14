package overlayserver

import (
	"fmt"
	"log"
	"net/http"
)

type Server struct {
	port int
	mux  *http.ServeMux
}

func NewServer() (*Server, error) {
	// Find available port in range 3000-3100
	port, err := findAvailablePort(3000, 3100)
	if err != nil {
		return nil, fmt.Errorf("no available ports: %w", err)
	}

	s := &Server{
		port: port,
		mux:  http.NewServeMux(),
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
