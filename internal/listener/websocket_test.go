package listener_test

import (
	"testing"
	"bbapp/internal/listener"
)

func TestWebSocketListener_OnFrame(t *testing.T) {
	frameReceived := false

	wsl := listener.NewWebSocketListener()
	wsl.OnFrame(func(data string) {
		frameReceived = true
	})

	// Simulate frame
	wsl.HandleFrame(`{"type":"test"}`)

	if !frameReceived {
		t.Fatal("Expected frame to be received")
	}
}
