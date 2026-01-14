package api_test

import (
	"bbapp/internal/api"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClient_GetConfig(t *testing.T) {
	// Mock BB-Core server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify new official endpoint
		if r.URL.Path != "/api/v1/external/config" {
			t.Errorf("Expected /api/v1/external/config, got %s", r.URL.Path)
		}

		// Verify Authorization header
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("Expected Bearer token, got %s", r.Header.Get("Authorization"))
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"roomId":"test-room","agencyId":1,"teams":[]}`))
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "test-token")
	config, err := client.GetConfig("test-room")

	if err != nil {
		t.Fatalf("GetConfig failed: %v", err)
	}

	if config.RoomId != "test-room" {
		t.Errorf("Expected roomId=test-room, got %s", config.RoomId)
	}
}

// TestDoRequest_APIError tests that doRequest parses APIError responses
func TestDoRequest_APIError(t *testing.T) {
	// Mock BB-Core server returning APIError
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{
			"timestamp": "2025-12-26T12:00:00.000+0000",
			"status": "BAD_REQUEST",
			"errorCode": 1003,
			"message": "Validation failed",
			"details": "Invalid input"
		}`))
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "test-token")
	_, err := client.GetConfig("test-room")

	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	// Check if error is APIError
	apiErr, ok := err.(*api.APIError)
	if !ok {
		t.Fatalf("Expected APIError, got %T: %v", err, err)
	}

	if apiErr.ErrorCode != 1003 {
		t.Errorf("Expected error code 1003, got %d", apiErr.ErrorCode)
	}

	if apiErr.Message != "Validation failed" {
		t.Errorf("Expected message 'Validation failed', got %s", apiErr.Message)
	}

	if apiErr.Status != "BAD_REQUEST" {
		t.Errorf("Expected status 'BAD_REQUEST', got %s", apiErr.Status)
	}
}

// TestDoRequest_GenericError tests fallback to generic error for non-APIError responses
func TestDoRequest_GenericError(t *testing.T) {
	// Mock server returning plain text error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Not found"))
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "test-token")
	_, err := client.GetConfig("test-room")

	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	// Should NOT be APIError
	if _, ok := err.(*api.APIError); ok {
		t.Fatal("Expected generic error, got APIError")
	}

	// Should contain status code
	expectedMsg := "HTTP 404"
	if len(err.Error()) < len(expectedMsg) || err.Error()[:8] != expectedMsg {
		t.Errorf("Expected error to start with '%s', got: %v", expectedMsg, err)
	}
}

// TestClient_Login_Success tests successful login
func TestClient_Login_Success(t *testing.T) {
	// Mock BB-Core server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify endpoint
		if r.URL.Path != "/api/v1/auth/login" {
			t.Errorf("Expected /api/v1/auth/login, got %s", r.URL.Path)
		}

		// Verify method
		if r.Method != "POST" {
			t.Errorf("Expected POST, got %s", r.Method)
		}

		// Verify Content-Type
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type: application/json, got %s", r.Header.Get("Content-Type"))
		}

		// Verify no Authorization header (login is public)
		if r.Header.Get("Authorization") != "" {
			t.Errorf("Expected no Authorization header, got %s", r.Header.Get("Authorization"))
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"accessToken": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
			"refreshToken": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.refresh...",
			"tokenType": "Bearer",
			"expiresIn": 86400000,
			"expiresAt": "2025-12-27T12:00:00.000Z",
			"user": {
				"id": 123,
				"username": "john_doe",
				"email": "john@example.com",
				"firstName": "John",
				"lastName": "Doe",
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
		}`))
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "")
	resp, err := client.Login("john_doe", "SecurePass123!")

	if err != nil {
		t.Fatalf("Login failed: %v", err)
	}

	if resp.AccessToken != "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..." {
		t.Errorf("Unexpected access token: %s", resp.AccessToken)
	}

	if resp.User.Username != "john_doe" {
		t.Errorf("Expected username 'john_doe', got %s", resp.User.Username)
	}

	if resp.Agency.Name != "Test Agency" {
		t.Errorf("Expected agency 'Test Agency', got %s", resp.Agency.Name)
	}
}

// TestClient_Login_InvalidCredentials tests login with wrong credentials
func TestClient_Login_InvalidCredentials(t *testing.T) {
	// Mock BB-Core server returning error 2002
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{
			"timestamp": "2025-12-26T12:00:00.000+0000",
			"status": "UNAUTHORIZED",
			"errorCode": 2002,
			"message": "Invalid username or password",
			"details": "Authentication failed"
		}`))
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "")
	resp, err := client.Login("john_doe", "WrongPassword")

	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	if resp != nil {
		t.Fatalf("Expected nil response, got %+v", resp)
	}

	// Check if error is APIError with code 2002
	apiErr, ok := err.(*api.APIError)
	if !ok {
		t.Fatalf("Expected APIError, got %T: %v", err, err)
	}

	if apiErr.ErrorCode != 2002 {
		t.Errorf("Expected error code 2002, got %d", apiErr.ErrorCode)
	}

	if apiErr.Message != "Invalid username or password" {
		t.Errorf("Expected message 'Invalid username or password', got %s", apiErr.Message)
	}

	if apiErr.Status != "UNAUTHORIZED" {
		t.Errorf("Expected status 'UNAUTHORIZED', got %s", apiErr.Status)
	}
}

// TestClient_Login_ValidationError tests login with invalid input
func TestClient_Login_ValidationError(t *testing.T) {
	// Mock BB-Core server returning validation error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{
			"timestamp": "2025-12-26T12:00:00.000+0000",
			"status": "BAD_REQUEST",
			"errorCode": 1003,
			"message": "Validation failed",
			"details": "Invalid username or password format",
			"subErrors": [
				{
					"object": "LoginRequest",
					"field": "username",
					"rejectedValue": "ab",
					"message": "Username must be between 3 and 50 characters"
				}
			]
		}`))
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "")
	resp, err := client.Login("ab", "password123")

	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	if resp != nil {
		t.Fatalf("Expected nil response, got %+v", resp)
	}

	// Check if error is APIError with code 1003
	apiErr, ok := err.(*api.APIError)
	if !ok {
		t.Fatalf("Expected APIError, got %T: %v", err, err)
	}

	if apiErr.ErrorCode != 1003 {
		t.Errorf("Expected error code 1003, got %d", apiErr.ErrorCode)
	}

	if apiErr.Message != "Validation failed" {
		t.Errorf("Expected message 'Validation failed', got %s", apiErr.Message)
	}

	// Check subErrors
	if len(apiErr.SubErrors) != 1 {
		t.Errorf("Expected 1 subError, got %d", len(apiErr.SubErrors))
	} else {
		subErr := apiErr.SubErrors[0]
		if subErr.Field != "username" {
			t.Errorf("Expected field 'username', got %s", subErr.Field)
		}
		if subErr.Message != "Username must be between 3 and 50 characters" {
			t.Errorf("Unexpected validation message: %s", subErr.Message)
		}
	}
}

// TestClient_Login_ServerError tests 5xx error with retry logic
func TestClient_Login_ServerError(t *testing.T) {
	attemptCount := 0

	// Mock server that fails 3 times with 500
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attemptCount++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{
			"timestamp": "2025-12-26T12:00:00.000+0000",
			"status": "INTERNAL_SERVER_ERROR",
			"errorCode": 5000,
			"message": "Server error",
			"details": "Database connection failed"
		}`))
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "")
	resp, err := client.Login("john_doe", "password123")

	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	if resp != nil {
		t.Fatalf("Expected nil response, got %+v", resp)
	}

	// Verify retry logic was triggered (should have tried 3 times)
	if attemptCount != 3 {
		t.Errorf("Expected 3 attempts, got %d", attemptCount)
	}

	// Check if error is APIError with code 5000
	apiErr, ok := err.(*api.APIError)
	if !ok {
		t.Fatalf("Expected APIError, got %T: %v", err, err)
	}

	if apiErr.ErrorCode != 5000 {
		t.Errorf("Expected error code 5000, got %d", apiErr.ErrorCode)
	}
}

// TestClient_Login_EmptyCredentials tests validation of empty credentials
func TestClient_Login_EmptyCredentials(t *testing.T) {
	client := api.NewClient("http://localhost:8080", "")

	// Test empty username
	_, err := client.Login("", "password")
	if err == nil {
		t.Error("Expected error for empty username, got nil")
	}

	// Test empty password
	_, err = client.Login("username", "")
	if err == nil {
		t.Error("Expected error for empty password, got nil")
	}

	// Test whitespace username
	_, err = client.Login("   ", "password")
	if err == nil {
		t.Error("Expected error for whitespace username, got nil")
	}

	// Test whitespace password
	_, err = client.Login("username", "   ")
	if err == nil {
		t.Error("Expected error for whitespace password, got nil")
	}
}

// TestClient_Register_Success tests successful user registration
func TestClient_Register_Success(t *testing.T) {
	// Mock BB-Core server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify endpoint
		if r.URL.Path != "/api/v1/auth/register" {
			t.Errorf("Expected /api/v1/auth/register, got %s", r.URL.Path)
		}

		// Verify method
		if r.Method != "POST" {
			t.Errorf("Expected POST, got %s", r.Method)
		}

		// Verify Content-Type
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type: application/json, got %s", r.Header.Get("Content-Type"))
		}

		// Verify no Authorization header (register is public)
		if r.Header.Get("Authorization") != "" {
			t.Errorf("Expected no Authorization header, got %s", r.Header.Get("Authorization"))
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"accessToken": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
			"refreshToken": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.refresh...",
			"tokenType": "Bearer",
			"expiresIn": 86400000,
			"expiresAt": "2025-12-27T12:00:00.000Z",
			"user": {
				"id": 123,
				"username": "john_doe",
				"email": "john@example.com",
				"firstName": "John",
				"lastName": "Doe",
				"roleCode": "OWNER"
			},
			"agency": {
				"id": 456,
				"name": "My Agency",
				"plan": "TRIAL",
				"status": "ACTIVE",
				"maxRooms": 1,
				"currentRooms": 0,
				"expiresAt": "2025-12-31T23:59:59.000Z"
			}
		}`))
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "")
	resp, err := client.Register("john_doe", "john@example.com", "SecurePass123!", "My Agency", "John", "Doe")

	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	if resp.AccessToken != "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..." {
		t.Errorf("Unexpected access token: %s", resp.AccessToken)
	}

	if resp.User.Username != "john_doe" {
		t.Errorf("Expected username 'john_doe', got %s", resp.User.Username)
	}

	if resp.User.Email != "john@example.com" {
		t.Errorf("Expected email 'john@example.com', got %s", resp.User.Email)
	}

	if resp.Agency.Name != "My Agency" {
		t.Errorf("Expected agency 'My Agency', got %s", resp.Agency.Name)
	}
}

// TestClient_Register_DuplicateEntity tests registration with existing username/email
func TestClient_Register_DuplicateEntity(t *testing.T) {
	// Mock BB-Core server returning error 1002
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		w.Write([]byte(`{
			"timestamp": "2025-12-26T12:00:00.000+0000",
			"status": "CONFLICT",
			"errorCode": 1002,
			"message": "Duplicate entity",
			"details": "Username 'john_doe' already exists"
		}`))
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "")
	resp, err := client.Register("john_doe", "john@example.com", "SecurePass123!", "My Agency", "", "")

	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	if resp != nil {
		t.Fatalf("Expected nil response, got %+v", resp)
	}

	// Check if error is APIError with code 1002
	apiErr, ok := err.(*api.APIError)
	if !ok {
		t.Fatalf("Expected APIError, got %T: %v", err, err)
	}

	if apiErr.ErrorCode != 1002 {
		t.Errorf("Expected error code 1002, got %d", apiErr.ErrorCode)
	}

	if apiErr.Message != "Duplicate entity" {
		t.Errorf("Expected message 'Duplicate entity', got %s", apiErr.Message)
	}

	if apiErr.Status != "CONFLICT" {
		t.Errorf("Expected status 'CONFLICT', got %s", apiErr.Status)
	}
}

// TestClient_Register_ValidationError tests registration with invalid input
func TestClient_Register_ValidationError(t *testing.T) {
	// Mock BB-Core server returning validation error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{
			"timestamp": "2025-12-26T12:00:00.000+0000",
			"status": "BAD_REQUEST",
			"errorCode": 1003,
			"message": "Validation failed",
			"details": "Invalid email format",
			"subErrors": [
				{
					"object": "RegisterRequest",
					"field": "email",
					"rejectedValue": "invalid-email",
					"message": "Email must be a valid email address"
				}
			]
		}`))
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "")
	resp, err := client.Register("john_doe", "invalid-email", "SecurePass123!", "My Agency", "", "")

	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	if resp != nil {
		t.Fatalf("Expected nil response, got %+v", resp)
	}

	// Check if error is APIError with code 1003
	apiErr, ok := err.(*api.APIError)
	if !ok {
		t.Fatalf("Expected APIError, got %T: %v", err, err)
	}

	if apiErr.ErrorCode != 1003 {
		t.Errorf("Expected error code 1003, got %d", apiErr.ErrorCode)
	}

	if apiErr.Message != "Validation failed" {
		t.Errorf("Expected message 'Validation failed', got %s", apiErr.Message)
	}

	// Check subErrors
	if len(apiErr.SubErrors) != 1 {
		t.Errorf("Expected 1 subError, got %d", len(apiErr.SubErrors))
	} else {
		subErr := apiErr.SubErrors[0]
		if subErr.Field != "email" {
			t.Errorf("Expected field 'email', got %s", subErr.Field)
		}
		if subErr.Message != "Email must be a valid email address" {
			t.Errorf("Unexpected validation message: %s", subErr.Message)
		}
	}
}

// TestClient_Register_ServerError tests 5xx error with retry logic
func TestClient_Register_ServerError(t *testing.T) {
	attemptCount := 0

	// Mock server that fails 3 times with 500
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attemptCount++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{
			"timestamp": "2025-12-26T12:00:00.000+0000",
			"status": "INTERNAL_SERVER_ERROR",
			"errorCode": 5000,
			"message": "Server error",
			"details": "Database connection failed"
		}`))
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "")
	resp, err := client.Register("john_doe", "john@example.com", "SecurePass123!", "My Agency", "", "")

	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	if resp != nil {
		t.Fatalf("Expected nil response, got %+v", resp)
	}

	// Verify retry logic was triggered (should have tried 3 times)
	if attemptCount != 3 {
		t.Errorf("Expected 3 attempts, got %d", attemptCount)
	}

	// Check if error is APIError with code 5000
	apiErr, ok := err.(*api.APIError)
	if !ok {
		t.Fatalf("Expected APIError, got %T: %v", err, err)
	}

	if apiErr.ErrorCode != 5000 {
		t.Errorf("Expected error code 5000, got %d", apiErr.ErrorCode)
	}
}

// TestClient_Register_EmptyRequiredFields tests validation of empty required fields
func TestClient_Register_EmptyRequiredFields(t *testing.T) {
	client := api.NewClient("http://localhost:8080", "")

	// Test empty username
	_, err := client.Register("", "john@example.com", "password123", "My Agency", "", "")
	if err == nil {
		t.Error("Expected error for empty username, got nil")
	}

	// Test empty email
	_, err = client.Register("john_doe", "", "password123", "My Agency", "", "")
	if err == nil {
		t.Error("Expected error for empty email, got nil")
	}

	// Test empty password
	_, err = client.Register("john_doe", "john@example.com", "", "My Agency", "", "")
	if err == nil {
		t.Error("Expected error for empty password, got nil")
	}

	// Test empty agencyName
	_, err = client.Register("john_doe", "john@example.com", "password123", "", "", "")
	if err == nil {
		t.Error("Expected error for empty agencyName, got nil")
	}

	// Test whitespace username
	_, err = client.Register("   ", "john@example.com", "password123", "My Agency", "", "")
	if err == nil {
		t.Error("Expected error for whitespace username, got nil")
	}

	// Test whitespace email
	_, err = client.Register("john_doe", "   ", "password123", "My Agency", "", "")
	if err == nil {
		t.Error("Expected error for whitespace email, got nil")
	}

	// Test whitespace password
	_, err = client.Register("john_doe", "john@example.com", "   ", "My Agency", "", "")
	if err == nil {
		t.Error("Expected error for whitespace password, got nil")
	}

	// Test whitespace agencyName
	_, err = client.Register("john_doe", "john@example.com", "password123", "   ", "", "")
	if err == nil {
		t.Error("Expected error for whitespace agencyName, got nil")
	}

	// Test valid with empty optional fields (should not fail client-side)
	// This test would fail without a mock server, so we skip the actual call
	// The validation above confirms required fields are checked
}

// TestClient_RefreshToken_Success tests successful token refresh
func TestClient_RefreshToken_Success(t *testing.T) {
	// Mock BB-Core server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify endpoint
		if r.URL.Path != "/api/v1/auth/refresh-token" {
			t.Errorf("Expected /api/v1/auth/refresh-token, got %s", r.URL.Path)
		}

		// Verify method
		if r.Method != "POST" {
			t.Errorf("Expected POST, got %s", r.Method)
		}

		// Verify Content-Type
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type: application/json, got %s", r.Header.Get("Content-Type"))
		}

		// Verify no Authorization header (refresh is public endpoint)
		if r.Header.Get("Authorization") != "" {
			t.Errorf("Expected no Authorization header, got %s", r.Header.Get("Authorization"))
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"accessToken": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.new_access...",
			"refreshToken": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.new_refresh...",
			"tokenType": "Bearer",
			"expiresIn": 86400000,
			"expiresAt": "2025-12-27T12:00:00.000Z",
			"user": {
				"id": 123,
				"username": "john_doe",
				"email": "john@example.com",
				"firstName": "John",
				"lastName": "Doe",
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
		}`))
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "")
	resp, err := client.RefreshToken("eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.old_refresh...")

	if err != nil {
		t.Fatalf("RefreshToken failed: %v", err)
	}

	if resp.AccessToken != "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.new_access..." {
		t.Errorf("Unexpected access token: %s", resp.AccessToken)
	}

	if resp.RefreshToken != "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.new_refresh..." {
		t.Errorf("Unexpected refresh token: %s", resp.RefreshToken)
	}

	if resp.User.Username != "john_doe" {
		t.Errorf("Expected username 'john_doe', got %s", resp.User.Username)
	}

	if resp.Agency.Name != "Test Agency" {
		t.Errorf("Expected agency 'Test Agency', got %s", resp.Agency.Name)
	}
}

// TestClient_RefreshToken_InvalidToken tests refresh with invalid token
func TestClient_RefreshToken_InvalidToken(t *testing.T) {
	// Mock BB-Core server returning error 2002
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{
			"timestamp": "2025-12-26T12:00:00.000+0000",
			"status": "UNAUTHORIZED",
			"errorCode": 2002,
			"message": "Invalid credentials",
			"details": "Invalid refresh token signature"
		}`))
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "")
	resp, err := client.RefreshToken("invalid.token.signature")

	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	if resp != nil {
		t.Fatalf("Expected nil response, got %+v", resp)
	}

	// Check if error is APIError with code 2002
	apiErr, ok := err.(*api.APIError)
	if !ok {
		t.Fatalf("Expected APIError, got %T: %v", err, err)
	}

	if apiErr.ErrorCode != 2002 {
		t.Errorf("Expected error code 2002, got %d", apiErr.ErrorCode)
	}

	if apiErr.Message != "Invalid credentials" {
		t.Errorf("Expected message 'Invalid credentials', got %s", apiErr.Message)
	}

	if apiErr.Status != "UNAUTHORIZED" {
		t.Errorf("Expected status 'UNAUTHORIZED', got %s", apiErr.Status)
	}
}

// TestClient_RefreshToken_TokenExpired tests refresh with expired token
func TestClient_RefreshToken_TokenExpired(t *testing.T) {
	// Mock BB-Core server returning error 2003
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{
			"timestamp": "2025-12-26T12:00:00.000+0000",
			"status": "UNAUTHORIZED",
			"errorCode": 2003,
			"message": "Token expired",
			"details": "Refresh token has expired"
		}`))
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "")
	resp, err := client.RefreshToken("eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.expired...")

	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	if resp != nil {
		t.Fatalf("Expected nil response, got %+v", resp)
	}

	// Check if error is APIError with code 2003
	apiErr, ok := err.(*api.APIError)
	if !ok {
		t.Fatalf("Expected APIError, got %T: %v", err, err)
	}

	if apiErr.ErrorCode != 2003 {
		t.Errorf("Expected error code 2003, got %d", apiErr.ErrorCode)
	}

	if apiErr.Message != "Token expired" {
		t.Errorf("Expected message 'Token expired', got %s", apiErr.Message)
	}

	if apiErr.Status != "UNAUTHORIZED" {
		t.Errorf("Expected status 'UNAUTHORIZED', got %s", apiErr.Status)
	}
}

// TestClient_RefreshToken_ValidationError tests refresh with validation error
func TestClient_RefreshToken_ValidationError(t *testing.T) {
	// Mock BB-Core server returning validation error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{
			"timestamp": "2025-12-26T12:00:00.000+0000",
			"status": "BAD_REQUEST",
			"errorCode": 1003,
			"message": "Validation failed",
			"details": "Invalid refresh token format",
			"subErrors": [
				{
					"object": "RefreshTokenRequest",
					"field": "refreshToken",
					"rejectedValue": "malformed",
					"message": "Refresh token must be a valid JWT"
				}
			]
		}`))
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "")
	resp, err := client.RefreshToken("malformed")

	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	if resp != nil {
		t.Fatalf("Expected nil response, got %+v", resp)
	}

	// Check if error is APIError with code 1003
	apiErr, ok := err.(*api.APIError)
	if !ok {
		t.Fatalf("Expected APIError, got %T: %v", err, err)
	}

	if apiErr.ErrorCode != 1003 {
		t.Errorf("Expected error code 1003, got %d", apiErr.ErrorCode)
	}

	if apiErr.Message != "Validation failed" {
		t.Errorf("Expected message 'Validation failed', got %s", apiErr.Message)
	}

	// Check subErrors
	if len(apiErr.SubErrors) != 1 {
		t.Errorf("Expected 1 subError, got %d", len(apiErr.SubErrors))
	} else {
		subErr := apiErr.SubErrors[0]
		if subErr.Field != "refreshToken" {
			t.Errorf("Expected field 'refreshToken', got %s", subErr.Field)
		}
		if subErr.Message != "Refresh token must be a valid JWT" {
			t.Errorf("Unexpected validation message: %s", subErr.Message)
		}
	}
}

// TestClient_RefreshToken_ServerError tests 5xx error with retry logic
func TestClient_RefreshToken_ServerError(t *testing.T) {
	attemptCount := 0

	// Mock server that fails 3 times with 500
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attemptCount++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{
			"timestamp": "2025-12-26T12:00:00.000+0000",
			"status": "INTERNAL_SERVER_ERROR",
			"errorCode": 5000,
			"message": "Server error",
			"details": "Database connection failed"
		}`))
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "")
	resp, err := client.RefreshToken("eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.refresh...")

	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	if resp != nil {
		t.Fatalf("Expected nil response, got %+v", resp)
	}

	// Verify retry logic was triggered (should have tried 3 times)
	if attemptCount != 3 {
		t.Errorf("Expected 3 attempts, got %d", attemptCount)
	}

	// Check if error is APIError with code 5000
	apiErr, ok := err.(*api.APIError)
	if !ok {
		t.Fatalf("Expected APIError, got %T: %v", err, err)
	}

	if apiErr.ErrorCode != 5000 {
		t.Errorf("Expected error code 5000, got %d", apiErr.ErrorCode)
	}
}

// TestClient_RefreshToken_EmptyToken tests validation of empty token
func TestClient_RefreshToken_EmptyToken(t *testing.T) {
	client := api.NewClient("http://localhost:8080", "")

	// Test empty token
	_, err := client.RefreshToken("")
	if err == nil {
		t.Error("Expected error for empty token, got nil")
	}

	// Test whitespace token
	_, err = client.RefreshToken("   ")
	if err == nil {
		t.Error("Expected error for whitespace token, got nil")
	}
}

// ValidateTrial Tests

func TestClient_ValidateTrial_Allowed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/external/validate-trial" {
			t.Errorf("Expected /api/v1/external/validate-trial, got %s", r.URL.Path)
		}

		if r.Method != "POST" {
			t.Errorf("Expected POST, got %s", r.Method)
		}

		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("Expected Bearer token, got %s", r.Header.Get("Authorization"))
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"allowed": true,
			"message": "All streamers validated",
			"blockedBigoIds": [],
			"reason": ""
		}`))
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "test-token")

	streamers := []api.ValidateTrialStreamer{
		{BigoId: "123456789", BigoRoomId: "7269255640400014299"},
		{BigoId: "987654321", BigoRoomId: "7478500464273093441"},
	}

	resp, err := client.ValidateTrial(streamers)
	if err != nil {
		t.Fatalf("ValidateTrial failed: %v", err)
	}

	if !resp.Allowed {
		t.Error("Expected allowed=true")
	}

	if resp.Message != "All streamers validated" {
		t.Errorf("Expected success message, got %s", resp.Message)
	}

	if len(resp.BlockedBigoIds) != 0 {
		t.Errorf("Expected empty blockedBigoIds, got %v", resp.BlockedBigoIds)
	}
}

func TestClient_ValidateTrial_Rejected(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"allowed": false,
			"message": "Bigo ID 829454322 has already been used in a trial",
			"blockedBigoIds": ["829454322"],
			"reason": "TRIAL_BIGO_ID_USED"
		}`))
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "test-token")

	streamers := []api.ValidateTrialStreamer{
		{BigoId: "829454322", BigoRoomId: "7478500464273093441"},
	}

	resp, err := client.ValidateTrial(streamers)
	if err != nil {
		t.Fatalf("ValidateTrial failed: %v", err)
	}

	if resp.Allowed {
		t.Error("Expected allowed=false")
	}

	if len(resp.BlockedBigoIds) != 1 {
		t.Errorf("Expected 1 blocked ID, got %d", len(resp.BlockedBigoIds))
	}

	if resp.BlockedBigoIds[0] != "829454322" {
		t.Errorf("Expected blocked ID 829454322, got %s", resp.BlockedBigoIds[0])
	}

	if resp.Reason != "TRIAL_BIGO_ID_USED" {
		t.Errorf("Expected reason TRIAL_BIGO_ID_USED, got %s", resp.Reason)
	}
}

// SendHeartbeat Tests

func TestClient_SendHeartbeat_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify new official endpoint
		if r.URL.Path != "/api/v1/external/heartbeat" {
			t.Errorf("Expected /api/v1/external/heartbeat, got %s", r.URL.Path)
		}

		if r.Method != "POST" {
			t.Errorf("Expected POST, got %s", r.Method)
		}

		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("Expected Bearer token, got %s", r.Header.Get("Authorization"))
		}

		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected application/json, got %s", r.Header.Get("Content-Type"))
		}

		// Verify request body structure
		var reqBody api.HeartbeatRequest
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			t.Fatalf("Failed to decode request body: %v", err)
		}

		if len(reqBody.Connections) != 2 {
			t.Errorf("Expected 2 connections, got %d", len(reqBody.Connections))
		}

		// Verify ConnectionStatus structure
		if reqBody.Connections[0].BigoId != "123456789" {
			t.Errorf("Expected BigoId 123456789, got %s", reqBody.Connections[0].BigoId)
		}

		if reqBody.Connections[0].Status != "CONNECTED" {
			t.Errorf("Expected CONNECTED, got %s", reqBody.Connections[0].Status)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success": true}`))
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "test-token")

	status := api.HeartbeatRequest{
		Connections: []api.ConnectionStatus{
			{
				BigoId:           "123456789",
				BigoRoomId:       "7269255640400014299",
				Status:           "CONNECTED",
				LastMessageAt:    1234567890,
				MessagesReceived: 42,
			},
			{
				BigoId:           "987654321",
				BigoRoomId:       "7478500464273093441",
				Status:           "CONNECTED",
				LastMessageAt:    1234567891,
				MessagesReceived: 38,
			},
		},
	}

	err := client.SendHeartbeat(status)
	if err != nil {
		t.Fatalf("SendHeartbeat failed: %v", err)
	}
}

func TestClient_SendHeartbeat_WithError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request body includes error information
		var reqBody api.HeartbeatRequest
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			t.Fatalf("Failed to decode request body: %v", err)
		}

		if len(reqBody.Connections) != 1 {
			t.Errorf("Expected 1 connection, got %d", len(reqBody.Connections))
		}

		if reqBody.Connections[0].Status != "DISCONNECTED" {
			t.Errorf("Expected DISCONNECTED, got %s", reqBody.Connections[0].Status)
		}

		if reqBody.Connections[0].Error != "WebSocket connection lost" {
			t.Errorf("Expected error message, got %s", reqBody.Connections[0].Error)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success": true}`))
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "test-token")

	status := api.HeartbeatRequest{
		Connections: []api.ConnectionStatus{
			{
				BigoId:           "123456789",
				BigoRoomId:       "7269255640400014299",
				Status:           "DISCONNECTED",
				LastMessageAt:    1234567890,
				MessagesReceived: 42,
				Error:            "WebSocket connection lost",
			},
		},
	}

	err := client.SendHeartbeat(status)
	if err != nil {
		t.Fatalf("SendHeartbeat failed: %v", err)
	}
}
