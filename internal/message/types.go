package message

// GiftMessage represents a gift event sent to BB-Core via STOMP.
// Matches the specification from docs/BBAPP_INTEGRATION_GUIDE.md
type GiftMessage struct {
	Type           string `json:"type"` // "GIFT"
	RoomID         string `json:"roomId"`
	BigoRoomID     string `json:"bigoRoomId"`
	SenderID       string `json:"senderId"`
	SenderName     string `json:"senderName"`
	SenderAvatar   string `json:"senderAvatar,omitempty"`
	SenderLevel    int    `json:"senderLevel,omitempty"`
	StreamerID     string `json:"streamerId"`
	StreamerName   string `json:"streamerName"`
	StreamerAvatar string `json:"streamerAvatar,omitempty"`
	GiftID         string `json:"giftId"`
	GiftName       string `json:"giftName"`
	GiftCount      int    `json:"giftCount"`
	Diamonds       int64  `json:"diamonds"`
	GiftImageURL   string `json:"giftImageUrl,omitempty"`
	Timestamp      int64  `json:"timestamp"`
	DeviceHash     string `json:"deviceHash"`
}

// ChatMessage represents a chat message sent to BB-Core via STOMP.
// Matches the specification from docs/BBAPP_INTEGRATION_GUIDE.md
type ChatMessage struct {
	Type         string `json:"type"` // "CHAT"
	RoomID       string `json:"roomId"`
	BigoRoomID   string `json:"bigoRoomId"`
	SenderID     string `json:"senderId"`
	SenderName   string `json:"senderName"`
	SenderAvatar string `json:"senderAvatar,omitempty"`
	SenderLevel  int    `json:"senderLevel,omitempty"`
	Message      string `json:"message"`
	Timestamp    int64  `json:"timestamp"`
	DeviceHash   string `json:"deviceHash"`
}
