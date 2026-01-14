package stomp

import (
	"testing"
)

func TestNormalizeURL(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"http://localhost:8080", "ws://localhost:8080/ws/999/bbapp/websocket"}, // auto-append /ws then SockJS
		{"https://localhost:8080", "wss://localhost:8080/ws/999/bbapp/websocket"},
		{"http://localhost:8080/", "ws://localhost:8080/ws/999/bbapp/websocket"},
		{"http://localhost:8080/backend", "ws://localhost:8080/backend"},         // path preserved
		{"ws://localhost:8080", "ws://localhost:8080/ws/999/bbapp/websocket"},    // normalized to SockJS
		{"ws://localhost:8080/ws", "ws://localhost:8080/ws/999/bbapp/websocket"}, // SockJS expansion
		{"wss://localhost:8080/ws/websocket", "wss://localhost:8080/ws/websocket"},
	}

	for _, tt := range tests {
		got, err := normalizeURL(tt.input)
		if err != nil {
			t.Errorf("normalizeURL(%q) returned error: %v", tt.input, err)
			continue
		}
		if got != tt.expected {
			t.Errorf("normalizeURL(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}
