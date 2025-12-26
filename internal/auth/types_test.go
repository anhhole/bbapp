package auth

import (
	"testing"
	"time"
)

func TestCredentials_IsExpired(t *testing.T) {
	tests := []struct {
		name      string
		expiresAt time.Time
		want      bool
	}{
		{
			name:      "not expired",
			expiresAt: time.Now().Add(1 * time.Hour),
			want:      false,
		},
		{
			name:      "expired",
			expiresAt: time.Now().Add(-1 * time.Hour),
			want:      true,
		},
		{
			name:      "expires in 1 minute (not expired)",
			expiresAt: time.Now().Add(1 * time.Minute),
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Credentials{
				AccessToken:  "token",
				RefreshToken: "refresh",
				ExpiresAt:    tt.expiresAt,
			}
			if got := c.IsExpired(); got != tt.want {
				t.Errorf("IsExpired() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCredentials_NeedsRefresh(t *testing.T) {
	tests := []struct {
		name      string
		expiresAt time.Time
		want      bool
	}{
		{
			name:      "needs refresh in 5 minutes",
			expiresAt: time.Now().Add(5 * time.Minute),
			want:      true,
		},
		{
			name:      "needs refresh in 9 minutes",
			expiresAt: time.Now().Add(9 * time.Minute),
			want:      true,
		},
		{
			name:      "doesnt need refresh in 15 minutes",
			expiresAt: time.Now().Add(15 * time.Minute),
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Credentials{
				AccessToken:  "token",
				RefreshToken: "refresh",
				ExpiresAt:    tt.expiresAt,
			}
			if got := c.NeedsRefresh(); got != tt.want {
				t.Errorf("NeedsRefresh() = %v, want %v", got, tt.want)
			}
		})
	}
}
