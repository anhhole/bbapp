package profile

import (
	"time"
	"bbapp/internal/api"
)

// Profile represents a saved configuration profile
type Profile struct {
	ID         string     `json:"id"`          // UUID
	Name       string     `json:"name"`        // User-friendly name
	RoomID     string     `json:"roomId"`      // BB-Core room ID
	CreatedAt  time.Time  `json:"createdAt"`   // Creation timestamp
	UpdatedAt  time.Time  `json:"updatedAt"`   // Last update timestamp
	LastUsedAt *time.Time `json:"lastUsedAt"`  // Last time profile was loaded (nullable)
	Config     api.Config `json:"config"`      // Cached BB-Core config
}
