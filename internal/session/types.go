package session

import (
	"bbapp/internal/api"
)

type Status struct {
	RoomId      string
	SessionId   string
	IsActive    bool
	Connections []api.ConnectionStatus
	StartTime   int64
	DeviceHash  string
}
