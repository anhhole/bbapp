package api

import "time"

type Config struct {
	RoomId          string          `json:"roomId"`
	AgencyId        int             `json:"agencyId"`
	Session         SessionInfo     `json:"session"`
	Teams           []Team          `json:"teams"`
	OverlaySettings OverlaySettings `json:"overlaySettings"`
}

type OverlaySettings struct {
	ShowStreamerAvatar bool `json:"showStreamerAvatar"`
	ShowWinStreak      bool `json:"showWinStreak"`
	TimerDuration      int  `json:"timerDuration"` // in seconds, if using independent timer
}

type SaveConfigRequest struct {
	RoomId      string `json:"roomId"`
	ConfigData  Config `json:"configData"`
	Description string `json:"description"`
	IsActive    bool   `json:"isActive"`
}

type GlobalIdol struct {
	Name       string `json:"name"`
	BigoRoomId string `json:"bigoRoomId"`
	Avatar     string `json:"avatar"`
}

type SessionInfo struct {
	SessionId       string                 `json:"sessionId"`
	StartTime       int64                  `json:"startTime"`
	Status          string                 `json:"status"`
	DurationMinutes int                    `json:"durationMinutes"`
	PausedAt        int64                  `json:"pausedAt"`
	ResumedAt       int64                  `json:"resumedAt"`
	ScriptData      map[string]interface{} `json:"scriptData"`
}

type Team struct {
	TeamId           string           `json:"teamId"`
	Name             string           `json:"name"`
	Avatar           string           `json:"avatar"`
	BindingGift      string           `json:"bindingGift"`
	BindingGiftImage string           `json:"bindingGiftImage"`
	ScoreMultipliers map[string]int64 `json:"scoreMultipliers"`
	Streamers        []Streamer       `json:"streamers"`
}

type Streamer struct {
	StreamerId  string `json:"id"`
	BigoId      string `json:"bigoId"`
	BigoRoomId  string `json:"bigoRoomId"`
	Name        string `json:"name"`
	Avatar      string `json:"avatar"`
	BindingGift string `json:"bindingGift"`
}

// Script Management Types (New /api/v1/scripts endpoints)

type StartScriptRequest struct {
	RoomId          string                 `json:"roomId"`
	ScriptType      string                 `json:"scriptType"` // "PK", "CHAMP", "CHAT_RANKING"
	DurationMinutes int                    `json:"durationMinutes"`
	ScriptPayload   map[string]interface{} `json:"scriptPayload,omitempty"`
}

type StartScriptResponse struct {
	SessionId       string `json:"sessionId"`
	RoomId          string `json:"roomId"`
	ScriptType      string `json:"scriptType"`
	Status          string `json:"status"`
	StartedAt       int64  `json:"startedAt"`
	EndsAt          int64  `json:"endsAt"`
	DurationMinutes int    `json:"durationMinutes"`
}

type StopScriptRequest struct {
	SessionId string `json:"sessionId"`
}

type StopScriptResponse struct {
	SessionId  string                 `json:"sessionId"`
	RoomId     string                 `json:"roomId"`
	ScriptType string                 `json:"scriptType"`
	Status     string                 `json:"status"`
	StartedAt  int64                  `json:"startedAt"`
	EndedAt    int64                  `json:"endedAt"`
	FinalData  map[string]interface{} `json:"finalData,omitempty"`
}

type HeartbeatRequest struct {
	Connections []ConnectionStatus `json:"connections"`
}

type ConnectionStatus struct {
	BigoId           string `json:"bigoId"`
	BigoRoomId       string `json:"bigoRoomId"`
	Status           string `json:"status"` // CONNECTED, DISCONNECTED
	LastMessageAt    int64  `json:"lastMessageAt,omitempty"`
	MessagesReceived int64  `json:"messagesReceived,omitempty"`
	Error            string `json:"error,omitempty"`
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
	Streamers []ValidateTrialStreamer `json:"idols"` // JSON uses "idols" for frontend compatibility
}

// ValidateTrialResponse is the response from trial validation
type ValidateTrialResponse struct {
	Allowed        bool     `json:"allowed"`
	Message        string   `json:"message"`
	BlockedBigoIds []string `json:"blockedBigoIds"`
	Reason         string   `json:"reason,omitempty"`
}

// GiftDefinition represents a gift in the library
type GiftDefinition struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Diamonds int    `json:"diamonds"`
	Image    string `json:"image"`
}
