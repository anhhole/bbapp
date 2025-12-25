package session_test

import (
	"testing"
	"bbapp/internal/session"
)

func TestManager_GetStatus(t *testing.T) {
	manager := session.NewManager()

	status := manager.GetStatus()

	if status.RoomId != "" {
		t.Error("Expected empty roomId for new manager")
	}

	if status.IsActive {
		t.Error("Expected inactive status for new manager")
	}
}
