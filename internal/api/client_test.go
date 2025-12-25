package api_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"bbapp/internal/api"
)

func TestClient_GetConfig(t *testing.T) {
	// Mock BB-Core server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/bbapp-config/test-room" {
			t.Errorf("Expected /bbapp-config/test-room, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"roomId":"test-room","agencyId":1,"teams":[]}`))
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "test-token")
	config, err := client.GetConfig("test-room")

	if err != nil {
		t.Fatalf("GetConfig failed: %v", err)
	}

	if config.RoomId != "test-room" {
		t.Errorf("Expected roomId=test-room, got %s", config.RoomId)
	}
}
