package message

import (
	"encoding/json"
	"testing"
)

func TestGiftMessageSerialization(t *testing.T) {
	gift := GiftMessage{
		Type:           "GIFT",
		RoomID:         "room123",
		BigoRoomID:     "bigo456",
		SenderID:       "sender789",
		SenderName:     "Alice",
		SenderAvatar:   "https://example.com/avatar.jpg",
		SenderLevel:    25,
		StreamerID:     "streamer001",
		StreamerName:   "Bob",
		StreamerAvatar: "https://example.com/streamer.jpg",
		GiftID:         "gift123",
		GiftName:       "Rose",
		GiftCount:      5,
		Diamonds:       100,
		GiftImageURL:   "https://example.com/gift.png",
		Timestamp:      1672531200000,
		DeviceHash:     "abc123def456",
	}

	// Test JSON marshaling
	data, err := json.Marshal(gift)
	if err != nil {
		t.Fatalf("Failed to marshal GiftMessage: %v", err)
	}

	// Test JSON unmarshaling
	var decoded GiftMessage
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal GiftMessage: %v", err)
	}

	// Verify all fields are preserved
	if decoded.Type != gift.Type {
		t.Errorf("Type mismatch: got %s, want %s", decoded.Type, gift.Type)
	}
	if decoded.RoomID != gift.RoomID {
		t.Errorf("RoomID mismatch: got %s, want %s", decoded.RoomID, gift.RoomID)
	}
	if decoded.BigoRoomID != gift.BigoRoomID {
		t.Errorf("BigoRoomID mismatch: got %s, want %s", decoded.BigoRoomID, gift.BigoRoomID)
	}
	if decoded.SenderID != gift.SenderID {
		t.Errorf("SenderID mismatch: got %s, want %s", decoded.SenderID, gift.SenderID)
	}
	if decoded.SenderName != gift.SenderName {
		t.Errorf("SenderName mismatch: got %s, want %s", decoded.SenderName, gift.SenderName)
	}
	if decoded.SenderAvatar != gift.SenderAvatar {
		t.Errorf("SenderAvatar mismatch: got %s, want %s", decoded.SenderAvatar, gift.SenderAvatar)
	}
	if decoded.SenderLevel != gift.SenderLevel {
		t.Errorf("SenderLevel mismatch: got %d, want %d", decoded.SenderLevel, gift.SenderLevel)
	}
	if decoded.StreamerID != gift.StreamerID {
		t.Errorf("StreamerID mismatch: got %s, want %s", decoded.StreamerID, gift.StreamerID)
	}
	if decoded.StreamerName != gift.StreamerName {
		t.Errorf("StreamerName mismatch: got %s, want %s", decoded.StreamerName, gift.StreamerName)
	}
	if decoded.StreamerAvatar != gift.StreamerAvatar {
		t.Errorf("StreamerAvatar mismatch: got %s, want %s", decoded.StreamerAvatar, gift.StreamerAvatar)
	}
	if decoded.GiftID != gift.GiftID {
		t.Errorf("GiftID mismatch: got %s, want %s", decoded.GiftID, gift.GiftID)
	}
	if decoded.GiftName != gift.GiftName {
		t.Errorf("GiftName mismatch: got %s, want %s", decoded.GiftName, gift.GiftName)
	}
	if decoded.GiftCount != gift.GiftCount {
		t.Errorf("GiftCount mismatch: got %d, want %d", decoded.GiftCount, gift.GiftCount)
	}
	if decoded.Diamonds != gift.Diamonds {
		t.Errorf("Diamonds mismatch: got %d, want %d", decoded.Diamonds, gift.Diamonds)
	}
	if decoded.GiftImageURL != gift.GiftImageURL {
		t.Errorf("GiftImageURL mismatch: got %s, want %s", decoded.GiftImageURL, gift.GiftImageURL)
	}
	if decoded.Timestamp != gift.Timestamp {
		t.Errorf("Timestamp mismatch: got %d, want %d", decoded.Timestamp, gift.Timestamp)
	}
	if decoded.DeviceHash != gift.DeviceHash {
		t.Errorf("DeviceHash mismatch: got %s, want %s", decoded.DeviceHash, gift.DeviceHash)
	}
}

func TestGiftMessageJSONFieldNames(t *testing.T) {
	gift := GiftMessage{
		Type:       "GIFT",
		RoomID:     "room123",
		BigoRoomID: "bigo456",
		DeviceHash: "abc123",
	}

	data, err := json.Marshal(gift)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	// Verify JSON field names match specification
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("Failed to unmarshal to map: %v", err)
	}

	// Check critical fields use correct JSON names
	expectedFields := []string{
		"type", "roomId", "bigoRoomId", "senderId", "senderName",
		"streamerId", "streamerName", "giftId", "giftName",
		"giftCount", "diamonds", "timestamp", "deviceHash",
	}

	for _, field := range expectedFields {
		if _, exists := raw[field]; !exists && field == "type" {
			t.Errorf("Required field %s missing from JSON", field)
		}
	}
}

func TestChatMessageSerialization(t *testing.T) {
	chat := ChatMessage{
		Type:         "CHAT",
		RoomID:       "room123",
		BigoRoomID:   "bigo456",
		SenderID:     "sender789",
		SenderName:   "Charlie",
		SenderAvatar: "https://example.com/charlie.jpg",
		SenderLevel:  15,
		Message:      "Hello, world!",
		Timestamp:    1672531200000,
		DeviceHash:   "xyz789abc123",
	}

	// Test JSON marshaling
	data, err := json.Marshal(chat)
	if err != nil {
		t.Fatalf("Failed to marshal ChatMessage: %v", err)
	}

	// Test JSON unmarshaling
	var decoded ChatMessage
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal ChatMessage: %v", err)
	}

	// Verify all fields are preserved
	if decoded.Type != chat.Type {
		t.Errorf("Type mismatch: got %s, want %s", decoded.Type, chat.Type)
	}
	if decoded.RoomID != chat.RoomID {
		t.Errorf("RoomID mismatch: got %s, want %s", decoded.RoomID, chat.RoomID)
	}
	if decoded.BigoRoomID != chat.BigoRoomID {
		t.Errorf("BigoRoomID mismatch: got %s, want %s", decoded.BigoRoomID, chat.BigoRoomID)
	}
	if decoded.SenderID != chat.SenderID {
		t.Errorf("SenderID mismatch: got %s, want %s", decoded.SenderID, chat.SenderID)
	}
	if decoded.SenderName != chat.SenderName {
		t.Errorf("SenderName mismatch: got %s, want %s", decoded.SenderName, chat.SenderName)
	}
	if decoded.SenderAvatar != chat.SenderAvatar {
		t.Errorf("SenderAvatar mismatch: got %s, want %s", decoded.SenderAvatar, chat.SenderAvatar)
	}
	if decoded.SenderLevel != chat.SenderLevel {
		t.Errorf("SenderLevel mismatch: got %d, want %d", decoded.SenderLevel, chat.SenderLevel)
	}
	if decoded.Message != chat.Message {
		t.Errorf("Message mismatch: got %s, want %s", decoded.Message, chat.Message)
	}
	if decoded.Timestamp != chat.Timestamp {
		t.Errorf("Timestamp mismatch: got %d, want %d", decoded.Timestamp, chat.Timestamp)
	}
	if decoded.DeviceHash != chat.DeviceHash {
		t.Errorf("DeviceHash mismatch: got %s, want %s", decoded.DeviceHash, chat.DeviceHash)
	}
}

func TestChatMessageJSONFieldNames(t *testing.T) {
	chat := ChatMessage{
		Type:       "CHAT",
		RoomID:     "room123",
		BigoRoomID: "bigo456",
		Message:    "Test message",
		DeviceHash: "abc123",
	}

	data, err := json.Marshal(chat)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	// Verify JSON field names match specification
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("Failed to unmarshal to map: %v", err)
	}

	// Check critical fields use correct JSON names
	expectedFields := []string{
		"type", "roomId", "bigoRoomId", "senderId", "senderName",
		"message", "timestamp", "deviceHash",
	}

	for _, field := range expectedFields {
		if _, exists := raw[field]; !exists && field == "type" {
			t.Errorf("Required field %s missing from JSON", field)
		}
	}
}

func TestGiftMessageOmitEmptyFields(t *testing.T) {
	// Test with minimal fields (omitempty should exclude empty optional fields)
	gift := GiftMessage{
		Type:         "GIFT",
		RoomID:       "room123",
		BigoRoomID:   "bigo456",
		SenderID:     "sender789",
		SenderName:   "Alice",
		StreamerID:   "streamer001",
		StreamerName: "Bob",
		GiftID:       "gift123",
		GiftName:     "Rose",
		GiftCount:    1,
		Diamonds:     10,
		Timestamp:    1672531200000,
		DeviceHash:   "abc123",
		// Omit: SenderAvatar, SenderLevel, StreamerAvatar, GiftImageURL
	}

	data, err := json.Marshal(gift)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("Failed to unmarshal to map: %v", err)
	}

	// senderLevel should be omitted when 0 (due to omitempty tag)
	if _, exists := raw["senderLevel"]; exists {
		t.Errorf("senderLevel should be omitted when 0 (omitempty)")
	}
}

func TestChatMessageOmitEmptyFields(t *testing.T) {
	// Test with minimal fields
	chat := ChatMessage{
		Type:       "CHAT",
		RoomID:     "room123",
		BigoRoomID: "bigo456",
		SenderID:   "sender789",
		SenderName: "Charlie",
		Message:    "Hello",
		Timestamp:  1672531200000,
		DeviceHash: "abc123",
		// Omit: SenderAvatar, SenderLevel
	}

	data, err := json.Marshal(chat)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("Failed to unmarshal to map: %v", err)
	}

	// senderLevel should be omitted when 0 (due to omitempty tag)
	if _, exists := raw["senderLevel"]; exists {
		t.Errorf("senderLevel should be omitted when 0 (omitempty)")
	}
}
