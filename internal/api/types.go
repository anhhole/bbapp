package api

import "time"

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

// Authentication Types

// LoginRequest is the request body for login
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// RegisterRequest is the request body for registration
type RegisterRequest struct {
	Username   string `json:"username"`
	Email      string `json:"email"`
	Password   string `json:"password"`
	FirstName  string `json:"firstName,omitempty"`
	LastName   string `json:"lastName,omitempty"`
	AgencyName string `json:"agencyName"`
}

// RefreshTokenRequest is the request body for token refresh
type RefreshTokenRequest struct {
	RefreshToken string `json:"refreshToken"`
}

// AuthResponse is the response from login/register/refresh
type AuthResponse struct {
	AccessToken  string    `json:"accessToken"`
	RefreshToken string    `json:"refreshToken"`
	TokenType    string    `json:"tokenType"`
	ExpiresIn    int64     `json:"expiresIn"`
	ExpiresAt    time.Time `json:"expiresAt"`
	User         User      `json:"user"`
	Agency       Agency    `json:"agency"`
}

// User represents authenticated user info
type User struct {
	ID        int64  `json:"id"`
	Username  string `json:"username"`
	Email     string `json:"email"`
	FirstName string `json:"firstName,omitempty"`
	LastName  string `json:"lastName,omitempty"`
	RoleCode  string `json:"roleCode"`
}

// Agency represents user's agency info
type Agency struct {
	ID           int64     `json:"id"`
	Name         string    `json:"name"`
	Plan         string    `json:"plan"` // TRIAL, PAID, PROFESSIONAL, ENTERPRISE
	Status       string    `json:"status"`
	MaxRooms     int       `json:"maxRooms"`
	CurrentRooms int       `json:"currentRooms"`
	ExpiresAt    time.Time `json:"expiresAt"`
}
// ValidateTrial Types

// ValidateTrialStreamer represents a streamer to validate
type ValidateTrialStreamer struct {
	BigoId     string `json:"bigoId"`
	BigoRoomId string `json:"bigoRoomId"`
}

// ValidateTrialRequest is the request body for trial validation
type ValidateTrialRequest struct {
	Streamers []ValidateTrialStreamer `json:"streamers"`
}

// ValidateTrialResponse is the response from trial validation
type ValidateTrialResponse struct {
	Allowed        bool     `json:"allowed"`
	Message        string   `json:"message"`
	BlockedBigoIds []string `json:"blockedBigoIds"`
	Reason         string   `json:"reason,omitempty"`
}
