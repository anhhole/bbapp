package config_test

import (
	"testing"
	"bbapp/internal/config"
	"bbapp/internal/api"
)

func TestManager_LookupStreamer(t *testing.T) {
	// Mock config
	cfg := &api.Config{
		RoomId: "test-room",
		Teams: []api.Team{
			{
				TeamId: "team1",
				Name:   "Team A",
				Streamers: []api.Streamer{
					{
						StreamerId: "streamer1",
						BigoRoomId: "12345",
						Name:       "Alice",
					},
				},
			},
		},
	}

	manager := config.NewManager(cfg)

	streamer, err := manager.LookupStreamerByBigoRoom("12345")
	if err != nil {
		t.Fatalf("LookupStreamerByBigoRoom failed: %v", err)
	}

	if streamer.Name != "Alice" {
		t.Errorf("Expected Alice, got %s", streamer.Name)
	}
}
