package auth

import (
	"time"
)

// Credentials represents stored authentication credentials
type Credentials struct {
	AccessToken  string    `json:"accessToken"`
	RefreshToken string    `json:"refreshToken"`
	ExpiresAt    time.Time `json:"expiresAt"`
	User         User      `json:"user"`
	Agency       Agency    `json:"agency"`
}

// User represents authenticated user info
type User struct {
	ID        int64  `json:"id"`
	Username  string `json:"username"`
	Email     string `json:"email"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
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

// IsExpired checks if access token is expired
func (c *Credentials) IsExpired() bool {
	return time.Now().After(c.ExpiresAt)
}

// NeedsRefresh checks if token expires within 10 minutes
func (c *Credentials) NeedsRefresh() bool {
	return time.Until(c.ExpiresAt) < 10*time.Minute
}
