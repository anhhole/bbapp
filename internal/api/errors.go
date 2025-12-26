package api

import "fmt"

// APIError represents a standardized error response from BB-Core API.
// It implements the Go error interface for idiomatic error handling.
//
// According to BBAPP_INTEGRATION_GUIDE.md Section 7 (Error Handling),
// BB-Core returns errors in this format for all error responses.
type APIError struct {
	Timestamp  string            `json:"timestamp"`           // ISO-8601 timestamp of when error occurred
	Status     string            `json:"status"`              // HTTP status name (e.g., "BAD_REQUEST", "NOT_FOUND")
	ErrorCode  int               `json:"errorCode"`           // Application-specific error code
	Message    string            `json:"message"`             // Human-readable error message
	Details    string            `json:"details,omitempty"`   // Additional error details (optional)
	SubErrors  []ValidationError `json:"subErrors,omitempty"` // Field-level validation errors (optional)
}

// ValidationError represents a field-level validation error.
// Used when BB-Core rejects specific fields in a request.
type ValidationError struct {
	Object        string      `json:"object"`        // Name of the DTO/object being validated
	Field         string      `json:"field"`         // Name of the invalid field
	RejectedValue interface{} `json:"rejectedValue"` // The value that was rejected (can be any type)
	Message       string      `json:"message"`       // Human-readable validation message
}

// Error implements the error interface for APIError.
// Returns a formatted string with error code and message.
func (e APIError) Error() string {
	return fmt.Sprintf("[%d] %s", e.ErrorCode, e.Message)
}
