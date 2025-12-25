package api

type Config struct {
	RoomId   string       `json:"roomId"`
	AgencyId int          `json:"agencyId"`
	Session  SessionInfo  `json:"session"`
	Teams    []Team       `json:"teams"`
}

type SessionInfo struct {
	SessionId string `json:"sessionId"`
	StartTime int64  `json:"startTime"`
	Status    string `json:"status"`
}

type Team struct {
	TeamId           string            `json:"teamId"`
	Name             string            `json:"name"`
	BindingGift      string            `json:"bindingGift"`
	ScoreMultipliers map[string]int64  `json:"scoreMultipliers"`
	Streamers        []Streamer        `json:"streamers"`
}

type Streamer struct {
	StreamerId   string `json:"streamerId"`
	BigoId       string `json:"bigoId"`
	BigoRoomId   string `json:"bigoRoomId"`
	Name         string `json:"name"`
	Avatar       string `json:"avatar"`
	BindingGift  string `json:"bindingGift"`
}

type StartSessionRequest struct {
	DeviceHash string `json:"deviceHash"`
}

type StartSessionResponse struct {
	SessionId string `json:"sessionId"`
	Success   bool   `json:"success"`
	Message   string `json:"message"`
}

type StopSessionRequest struct {
	Reason string `json:"reason"`
}

type StopSessionResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

type HeartbeatRequest struct {
	Connections []ConnectionStatus `json:"connections"`
}

type ConnectionStatus struct {
	BigoRoomId       string `json:"bigoRoomId"`
	StreamerId       string `json:"streamerId"`
	Status           string `json:"status"` // CONNECTED, DISCONNECTED, ERROR
	MessagesReceived int64  `json:"messagesReceived"`
	LastMessageTime  int64  `json:"lastMessageTime"`
	ErrorMessage     string `json:"errorMessage,omitempty"`
}
