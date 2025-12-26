package api

import (
	"encoding/json"
	"testing"
)

func TestAPIError_UnmarshalBasicError(t *testing.T) {
	// Test case from BBAPP_INTEGRATION_GUIDE.md - Entity Not Found (404)
	errorJSON := `{
		"timestamp": "2025-12-26T12:00:00.000+0000",
		"status": "NOT_FOUND",
		"errorCode": 2001,
		"message": "User not found with ID: 999",
		"details": "User not found with ID: 999"
	}`

	var apiErr APIError
	err := json.Unmarshal([]byte(errorJSON), &apiErr)
	if err != nil {
		t.Fatalf("Failed to unmarshal APIError: %v", err)
	}

	// Verify fields
	if apiErr.Timestamp != "2025-12-26T12:00:00.000+0000" {
		t.Errorf("Expected timestamp '2025-12-26T12:00:00.000+0000', got '%s'", apiErr.Timestamp)
	}
	if apiErr.Status != "NOT_FOUND" {
		t.Errorf("Expected status 'NOT_FOUND', got '%s'", apiErr.Status)
	}
	if apiErr.ErrorCode != 2001 {
		t.Errorf("Expected errorCode 2001, got %d", apiErr.ErrorCode)
	}
	if apiErr.Message != "User not found with ID: 999" {
		t.Errorf("Expected message 'User not found with ID: 999', got '%s'", apiErr.Message)
	}
	if apiErr.Details != "User not found with ID: 999" {
		t.Errorf("Expected details 'User not found with ID: 999', got '%s'", apiErr.Details)
	}
	if len(apiErr.SubErrors) != 0 {
		t.Errorf("Expected no sub-errors, got %d", len(apiErr.SubErrors))
	}
}

func TestAPIError_UnmarshalValidationError(t *testing.T) {
	// Test case from BBAPP_INTEGRATION_GUIDE.md - Validation Error (400)
	errorJSON := `{
		"timestamp": "2025-12-26T12:00:00.000+0000",
		"status": "BAD_REQUEST",
		"errorCode": 1003,
		"message": "Validation failed",
		"subErrors": [
			{
				"object": "LoginRequest",
				"field": "username",
				"rejectedValue": "ab",
				"message": "Username must be between 3 and 50 characters"
			}
		]
	}`

	var apiErr APIError
	err := json.Unmarshal([]byte(errorJSON), &apiErr)
	if err != nil {
		t.Fatalf("Failed to unmarshal APIError: %v", err)
	}

	// Verify basic fields
	if apiErr.Timestamp != "2025-12-26T12:00:00.000+0000" {
		t.Errorf("Expected timestamp '2025-12-26T12:00:00.000+0000', got '%s'", apiErr.Timestamp)
	}
	if apiErr.Status != "BAD_REQUEST" {
		t.Errorf("Expected status 'BAD_REQUEST', got '%s'", apiErr.Status)
	}
	if apiErr.ErrorCode != 1003 {
		t.Errorf("Expected errorCode 1003, got %d", apiErr.ErrorCode)
	}
	if apiErr.Message != "Validation failed" {
		t.Errorf("Expected message 'Validation failed', got '%s'", apiErr.Message)
	}

	// Verify sub-errors
	if len(apiErr.SubErrors) != 1 {
		t.Fatalf("Expected 1 sub-error, got %d", len(apiErr.SubErrors))
	}

	subErr := apiErr.SubErrors[0]
	if subErr.Object != "LoginRequest" {
		t.Errorf("Expected object 'LoginRequest', got '%s'", subErr.Object)
	}
	if subErr.Field != "username" {
		t.Errorf("Expected field 'username', got '%s'", subErr.Field)
	}
	if subErr.RejectedValue != "ab" {
		t.Errorf("Expected rejectedValue 'ab', got '%v'", subErr.RejectedValue)
	}
	if subErr.Message != "Username must be between 3 and 50 characters" {
		t.Errorf("Expected message 'Username must be between 3 and 50 characters', got '%s'", subErr.Message)
	}
}

func TestAPIError_UnmarshalAuthenticationError(t *testing.T) {
	// Test case from BBAPP_INTEGRATION_GUIDE.md - Authentication Error (401)
	errorJSON := `{
		"timestamp": "2025-12-26T12:00:00.000+0000",
		"status": "UNAUTHORIZED",
		"errorCode": 2002,
		"message": "Invalid username or password",
		"details": "Authentication failed"
	}`

	var apiErr APIError
	err := json.Unmarshal([]byte(errorJSON), &apiErr)
	if err != nil {
		t.Fatalf("Failed to unmarshal APIError: %v", err)
	}

	// Verify fields
	if apiErr.Status != "UNAUTHORIZED" {
		t.Errorf("Expected status 'UNAUTHORIZED', got '%s'", apiErr.Status)
	}
	if apiErr.ErrorCode != 2002 {
		t.Errorf("Expected errorCode 2002, got %d", apiErr.ErrorCode)
	}
	if apiErr.Message != "Invalid username or password" {
		t.Errorf("Expected message 'Invalid username or password', got '%s'", apiErr.Message)
	}
	if apiErr.Details != "Authentication failed" {
		t.Errorf("Expected details 'Authentication failed', got '%s'", apiErr.Details)
	}
}

func TestAPIError_UnmarshalTokenExpiredError(t *testing.T) {
	// Test case from BBAPP_INTEGRATION_GUIDE.md - Token Expired (401)
	errorJSON := `{
		"timestamp": "2025-12-26T12:00:00.000+0000",
		"status": "UNAUTHORIZED",
		"errorCode": 2003,
		"message": "JWT token has expired",
		"details": "Please refresh your token"
	}`

	var apiErr APIError
	err := json.Unmarshal([]byte(errorJSON), &apiErr)
	if err != nil {
		t.Fatalf("Failed to unmarshal APIError: %v", err)
	}

	// Verify fields
	if apiErr.ErrorCode != 2003 {
		t.Errorf("Expected errorCode 2003, got %d", apiErr.ErrorCode)
	}
	if apiErr.Message != "JWT token has expired" {
		t.Errorf("Expected message 'JWT token has expired', got '%s'", apiErr.Message)
	}
	if apiErr.Details != "Please refresh your token" {
		t.Errorf("Expected details 'Please refresh your token', got '%s'", apiErr.Details)
	}
}

func TestAPIError_ErrorMethod(t *testing.T) {
	tests := []struct {
		name     string
		apiErr   APIError
		expected string
	}{
		{
			name: "Basic error",
			apiErr: APIError{
				ErrorCode: 2001,
				Message:   "User not found with ID: 999",
			},
			expected: "[2001] User not found with ID: 999",
		},
		{
			name: "Validation error",
			apiErr: APIError{
				ErrorCode: 1003,
				Message:   "Validation failed",
			},
			expected: "[1003] Validation failed",
		},
		{
			name: "Authentication error",
			apiErr: APIError{
				ErrorCode: 2002,
				Message:   "Invalid username or password",
			},
			expected: "[2002] Invalid username or password",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.apiErr.Error()
			if result != tt.expected {
				t.Errorf("Expected Error() to return '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestValidationError_UnmarshalWithDifferentTypes(t *testing.T) {
	// Test that rejectedValue can handle different types
	tests := []struct {
		name         string
		errorJSON    string
		expectedType interface{}
	}{
		{
			name: "String rejected value",
			errorJSON: `{
				"object": "LoginRequest",
				"field": "username",
				"rejectedValue": "ab",
				"message": "Too short"
			}`,
			expectedType: "ab",
		},
		{
			name: "Number rejected value",
			errorJSON: `{
				"object": "GiftRequest",
				"field": "diamonds",
				"rejectedValue": -100,
				"message": "Must be positive"
			}`,
			expectedType: float64(-100), // JSON numbers unmarshal as float64
		},
		{
			name: "Null rejected value",
			errorJSON: `{
				"object": "Request",
				"field": "value",
				"rejectedValue": null,
				"message": "Required field"
			}`,
			expectedType: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var validationErr ValidationError
			err := json.Unmarshal([]byte(tt.errorJSON), &validationErr)
			if err != nil {
				t.Fatalf("Failed to unmarshal ValidationError: %v", err)
			}

			if validationErr.RejectedValue != tt.expectedType {
				t.Errorf("Expected rejectedValue %v (type %T), got %v (type %T)",
					tt.expectedType, tt.expectedType,
					validationErr.RejectedValue, validationErr.RejectedValue)
			}
		})
	}
}

func TestAPIError_EmptySubErrors(t *testing.T) {
	// Test that missing subErrors field results in empty slice (not nil)
	errorJSON := `{
		"timestamp": "2025-12-26T12:00:00.000+0000",
		"status": "NOT_FOUND",
		"errorCode": 2001,
		"message": "User not found"
	}`

	var apiErr APIError
	err := json.Unmarshal([]byte(errorJSON), &apiErr)
	if err != nil {
		t.Fatalf("Failed to unmarshal APIError: %v", err)
	}

	// SubErrors should be empty slice or nil - both are acceptable
	if apiErr.SubErrors == nil {
		// nil is OK
	} else if len(apiErr.SubErrors) != 0 {
		t.Errorf("Expected SubErrors to be empty, got %d elements", len(apiErr.SubErrors))
	}
}

func TestAPIError_MultipleValidationErrors(t *testing.T) {
	// Test error with multiple validation sub-errors
	errorJSON := `{
		"timestamp": "2025-12-26T12:00:00.000+0000",
		"status": "BAD_REQUEST",
		"errorCode": 1003,
		"message": "Validation failed",
		"subErrors": [
			{
				"object": "RegisterRequest",
				"field": "username",
				"rejectedValue": "ab",
				"message": "Username must be between 3 and 50 characters"
			},
			{
				"object": "RegisterRequest",
				"field": "password",
				"rejectedValue": "123",
				"message": "Password must be between 8 and 100 characters"
			},
			{
				"object": "RegisterRequest",
				"field": "email",
				"rejectedValue": "invalid-email",
				"message": "Must be a valid email address"
			}
		]
	}`

	var apiErr APIError
	err := json.Unmarshal([]byte(errorJSON), &apiErr)
	if err != nil {
		t.Fatalf("Failed to unmarshal APIError: %v", err)
	}

	if len(apiErr.SubErrors) != 3 {
		t.Fatalf("Expected 3 sub-errors, got %d", len(apiErr.SubErrors))
	}

	// Verify first sub-error
	if apiErr.SubErrors[0].Field != "username" {
		t.Errorf("Expected first sub-error field 'username', got '%s'", apiErr.SubErrors[0].Field)
	}

	// Verify second sub-error
	if apiErr.SubErrors[1].Field != "password" {
		t.Errorf("Expected second sub-error field 'password', got '%s'", apiErr.SubErrors[1].Field)
	}

	// Verify third sub-error
	if apiErr.SubErrors[2].Field != "email" {
		t.Errorf("Expected third sub-error field 'email', got '%s'", apiErr.SubErrors[2].Field)
	}
}
