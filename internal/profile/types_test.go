package profile

import (
	"encoding/json"
	"testing"
	"time"
	"bbapp/internal/api"
)

func TestProfile_JSONSerialization(t *testing.T) {
	now := time.Now()
	profile := &Profile{
		ID:        "test-uuid-123",
		Name:      "Test Profile",
		RoomID:    "room-456",
		CreatedAt: now,
		UpdatedAt: now,
		LastUsedAt: &now,
		Config: api.Config{
			RoomId:   "room-456",
			AgencyId: 789,
			Teams: []api.Team{
				{
					TeamId: "team1",
					Name:   "Team Alpha",
					Streamers: []api.Streamer{
						{
							StreamerId: "s1",
							BigoId:     "123456789",
							BigoRoomId: "7269255640400014299",
							Name:       "Streamer One",
						},
					},
				},
			},
		},
	}

	// Test JSON marshaling
	jsonData, err := json.Marshal(profile)
	if err != nil {
		t.Fatalf("Failed to marshal profile: %v", err)
	}

	// Test JSON unmarshaling
	var decoded Profile
	if err := json.Unmarshal(jsonData, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal profile: %v", err)
	}

	// Verify fields
	if decoded.ID != profile.ID {
		t.Errorf("ID = %s, want %s", decoded.ID, profile.ID)
	}
	if decoded.Name != profile.Name {
		t.Errorf("Name = %s, want %s", decoded.Name, profile.Name)
	}
	if decoded.RoomID != profile.RoomID {
		t.Errorf("RoomID = %s, want %s", decoded.RoomID, profile.RoomID)
	}
	if decoded.Config.RoomId != profile.Config.RoomId {
		t.Errorf("Config.RoomId = %s, want %s", decoded.Config.RoomId, profile.Config.RoomId)
	}
	if len(decoded.Config.Teams) != 1 {
		t.Errorf("Config.Teams length = %d, want 1", len(decoded.Config.Teams))
	}
}

func TestProfile_NilLastUsedAt(t *testing.T) {
	now := time.Now()
	profile := &Profile{
		ID:        "test-uuid-123",
		Name:      "Test Profile",
		RoomID:    "room-456",
		CreatedAt: now,
		UpdatedAt: now,
		LastUsedAt: nil, // Should handle nil value
		Config: api.Config{
			RoomId:   "room-456",
			AgencyId: 789,
		},
	}

	// Test JSON marshaling with nil LastUsedAt
	jsonData, err := json.Marshal(profile)
	if err != nil {
		t.Fatalf("Failed to marshal profile with nil LastUsedAt: %v", err)
	}

	// Test JSON unmarshaling
	var decoded Profile
	if err := json.Unmarshal(jsonData, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal profile: %v", err)
	}

	// Verify LastUsedAt is nil
	if decoded.LastUsedAt != nil {
		t.Errorf("LastUsedAt should be nil, got %v", *decoded.LastUsedAt)
	}
}
