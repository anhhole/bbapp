package session

import (
	"bbapp/internal/api"
)

type Status struct {
	RoomId      string                 `json:"roomId"`
	SessionId   string                 `json:"sessionId"`
	IsActive    bool                   `json:"isActive"`
	Connections []api.ConnectionStatus `json:"connections"`
	StartTime   int64                  `json:"startTime"`
	DeviceHash  string                 `json:"deviceHash"`
}
