package api_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"bbapp/internal/api"
)

func TestClient_GetConfig(t *testing.T) {
	// Mock BB-Core server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/bbapp-config/test-room" {
			t.Errorf("Expected /bbapp-config/test-room, got %s", r.URL.Path)
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
