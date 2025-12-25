package session_test

import (
	"testing"
	"time"
	"bbapp/internal/session"
)

func TestHeartbeat_Start(t *testing.T) {
	manager := session.NewManager()
	heartbeat := session.NewHeartbeat(manager, nil, "test-room", 100*time.Millisecond)

	// Start heartbeat
	heartbeat.Start()
	defer heartbeat.Stop()

	// Wait for at least one tick
	time.Sleep(150 * time.Millisecond)

	// Heartbeat should be running
	// (Real test would verify API call was made)
}
