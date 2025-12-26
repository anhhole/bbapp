package api

import (
	"encoding/json"
	"testing"
)

func TestAuthResponse_UnmarshalJSON(t *testing.T) {
	jsonData := `{
		"accessToken": "eyJhbGc...",
		"refreshToken": "eyJhbGc...",
		"tokenType": "Bearer",
		"expiresIn": 86400000,
		"expiresAt": "2025-12-27T12:00:00.000Z",
		"user": {
			"id": 123,
			"username": "testuser",
			"email": "test@example.com",
			"roleCode": "OWNER"
		},
		"agency": {
			"id": 456,
			"name": "Test Agency",
			"plan": "TRIAL",
			"status": "ACTIVE",
			"maxRooms": 1,
			"currentRooms": 0,
			"expiresAt": "2025-12-31T23:59:59.000Z"
		}
	}`

	var resp AuthResponse
	if err := json.Unmarshal([]byte(jsonData), &resp); err != nil {
		t.Fatalf("Unmarshal error = %v", err)
	}

	if resp.AccessToken != "eyJhbGc..." {
		t.Errorf("AccessToken = %s, want eyJhbGc...", resp.AccessToken)
	}
	if resp.User.Username != "testuser" {
		t.Errorf("Username = %s, want testuser", resp.User.Username)
	}
	if resp.Agency.Plan != "TRIAL" {
		t.Errorf("Plan = %s, want TRIAL", resp.Agency.Plan)
	}
}
