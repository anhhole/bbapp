package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	baseURL      string
	authToken    string
	refreshToken string
	httpClient   *http.Client
}

func NewClient(baseURL, authToken string) *Client {
	return &Client{
		baseURL:   baseURL,
		authToken: authToken,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// SetTokens updates both access and refresh tokens
func (c *Client) SetTokens(accessToken, refreshToken string) {
	c.authToken = accessToken
	c.refreshToken = refreshToken
}

// GetAccessToken returns the current access token
func (c *Client) GetAccessToken() string {
	return c.authToken
}

func (c *Client) GetConfig(roomId string) (*Config, error) {
	// Migrated to official BB-Core API endpoint
	url := fmt.Sprintf("%s/api/v1/external/config?roomId=%s", c.baseURL, roomId)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.authToken)
	req.Header.Set("Content-Type", "application/json")

	var config Config
	if err := c.doRequest(req, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

// SaveConfig saves room configuration to BB-Core
func (c *Client) SaveConfig(roomId string, config *Config) error {
	// Force the config RoomID to match the requested RoomID
	// This prevents issues where the config object has "default" or mismatching IDs
	config.RoomId = roomId

	// Migrated to official BB-Core API endpoint
	url := fmt.Sprintf("%s/api/v1/external/config?roomId=%s", c.baseURL, roomId)

	// Wrap config in the expected DTO
	reqBody := SaveConfigRequest{
		RoomId:      roomId,
		ConfigData:  *config,
		Description: "Updated via BBApp",
		IsActive:    true,
	}

	// Marshal config wrapper
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	// Debug logging - FULL PAYLOAD
	fmt.Printf("[SaveConfig] URL: %s\n", url)
	fmt.Printf("[SaveConfig] Payload: %s\n", string(jsonData))

	// Use the same pattern as other POST methods
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	// Enable request body recreation for retries
	req.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(jsonData)), nil
	}

	req.Header.Set("Authorization", "Bearer "+c.authToken)
	req.Header.Set("Content-Type", "application/json")

	var resp map[string]interface{}
	return c.doRequest(req, &resp)
}

// StartSession starts a new PK script session using /api/v1/scripts/start
func (c *Client) StartSession(roomId string, durationMinutes int, scriptPayload map[string]interface{}) (*StartScriptResponse, error) {
	url := fmt.Sprintf("%s/api/v1/scripts/start", c.baseURL)

	reqBody := StartScriptRequest{
		RoomId:          roomId,
		ScriptType:      "PK",
		DurationMinutes: durationMinutes,
		ScriptPayload:   scriptPayload,
	}
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// Enable request body recreation for retries
	req.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(jsonData)), nil
	}

	req.Header.Set("Authorization", "Bearer "+c.authToken)
	req.Header.Set("Content-Type", "application/json")

	var resp StartScriptResponse
	if err := c.doRequest(req, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// StopSession stops an active script session using /api/v1/scripts/stop
func (c *Client) StopSession(sessionId string) (*StopScriptResponse, error) {
	url := fmt.Sprintf("%s/api/v1/scripts/stop", c.baseURL)

	reqBody := StopScriptRequest{SessionId: sessionId}
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// Enable request body recreation for retries
	req.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(jsonData)), nil
	}

	req.Header.Set("Authorization", "Bearer "+c.authToken)
	req.Header.Set("Content-Type", "application/json")

	var resp StopScriptResponse
	if err := c.doRequest(req, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// ValidateTrial validates if streamers can be used for trial accounts
func (c *Client) ValidateTrial(streamers []ValidateTrialStreamer) (*ValidateTrialResponse, error) {
	url := fmt.Sprintf("%s/api/v1/external/validate-trial", c.baseURL)

	reqBody := ValidateTrialRequest{Streamers: streamers}
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// Enable request body recreation for retries
	req.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(jsonData)), nil
	}

	req.Header.Set("Authorization", "Bearer "+c.authToken)
	req.Header.Set("Content-Type", "application/json")

	var resp ValidateTrialResponse
	if err := c.doRequest(req, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

func (c *Client) SendHeartbeat(status HeartbeatRequest) error {
	// Migrated to official BB-Core API endpoint
	url := fmt.Sprintf("%s/api/v1/external/heartbeat", c.baseURL)

	jsonData, err := json.Marshal(status)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	// Enable request body recreation for retries
	req.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(jsonData)), nil
	}

	req.Header.Set("Authorization", "Bearer "+c.authToken)
	req.Header.Set("Content-Type", "application/json")

	var resp map[string]interface{}
	return c.doRequest(req, &resp)
}

// Login authenticates with BB-Core
func (c *Client) Login(username, password string) (*AuthResponse, error) {
	if strings.TrimSpace(username) == "" {
		return nil, fmt.Errorf("username cannot be empty")
	}
	if strings.TrimSpace(password) == "" {
		return nil, fmt.Errorf("password cannot be empty")
	}

	url := fmt.Sprintf("%s/api/v1/auth/login", c.baseURL)

	reqBody := LoginRequest{
		Username: username,
		Password: password,
	}
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// Enable request body recreation for retries
	req.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(jsonData)), nil
	}

	req.Header.Set("Content-Type", "application/json")

	var resp AuthResponse
	if err := c.doRequest(req, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// Register creates a new user account with BB-Core
func (c *Client) Register(username, email, password, agencyName, firstName, lastName string) (*AuthResponse, error) {
	if strings.TrimSpace(username) == "" {
		return nil, fmt.Errorf("username cannot be empty")
	}
	if strings.TrimSpace(email) == "" {
		return nil, fmt.Errorf("email cannot be empty")
	}
	if strings.TrimSpace(password) == "" {
		return nil, fmt.Errorf("password cannot be empty")
	}
	if strings.TrimSpace(agencyName) == "" {
		return nil, fmt.Errorf("agencyName cannot be empty")
	}

	url := fmt.Sprintf("%s/api/v1/auth/register", c.baseURL)

	reqBody := RegisterRequest{
		Username:   username,
		Email:      email,
		Password:   password,
		AgencyName: agencyName,
		FirstName:  firstName,
		LastName:   lastName,
	}
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// Enable request body recreation for retries
	req.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(jsonData)), nil
	}

	req.Header.Set("Content-Type", "application/json")

	var resp AuthResponse
	if err := c.doRequest(req, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// RefreshToken gets a new access token
func (c *Client) RefreshToken(refreshToken string) (*AuthResponse, error) {
	if strings.TrimSpace(refreshToken) == "" {
		return nil, fmt.Errorf("refresh token cannot be empty")
	}

	url := fmt.Sprintf("%s/api/v1/auth/refresh-token", c.baseURL)

	reqBody := RefreshTokenRequest{RefreshToken: refreshToken}
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// Enable request body recreation for retries
	req.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(jsonData)), nil
	}

	req.Header.Set("Content-Type", "application/json")

	var resp AuthResponse
	if err := c.doRequest(req, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

func (c *Client) doRequest(req *http.Request, result interface{}) error {
	// Retry logic with exponential backoff
	maxRetries := 3
	baseDelay := 1 * time.Second

	for attempt := 0; attempt < maxRetries; attempt++ {
		// Recreate request body for retries if GetBody is available
		if attempt > 0 && req.GetBody != nil {
			body, err := req.GetBody()
			if err != nil {
				return fmt.Errorf("failed to recreate request body: %w", err)
			}
			req.Body = body
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			if attempt < maxRetries-1 {
				time.Sleep(baseDelay * time.Duration(1<<attempt))
				continue
			}
			return fmt.Errorf("request failed: %w", err)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("read response: %w", err)
		}

		if resp.StatusCode >= 400 {
			if attempt < maxRetries-1 && resp.StatusCode >= 500 {
				time.Sleep(baseDelay * time.Duration(1<<attempt))
				continue
			}

			// Try to parse as APIError
			var apiErr APIError
			if err := json.Unmarshal(body, &apiErr); err == nil && apiErr.ErrorCode != 0 {
				// Check for token expired (401 with error code 2003)
				// Only attempt auto-refresh if we have a refresh token
				if resp.StatusCode == 401 && apiErr.ErrorCode == 2003 && c.refreshToken != "" {
					// Attempt token refresh
					authResp, refreshErr := c.RefreshToken(c.refreshToken)
					if refreshErr != nil {
						// Refresh failed, return original error
						return &apiErr
					}

					// Update tokens
					c.SetTokens(authResp.AccessToken, authResp.RefreshToken)

					// Retry original request with new token
					// Recreate request body if available
					if req.GetBody != nil {
						newBody, bodyErr := req.GetBody()
						if bodyErr != nil {
							return fmt.Errorf("failed to recreate request body for retry: %w", bodyErr)
						}
						req.Body = newBody
					}

					// Update Authorization header with new token
					req.Header.Set("Authorization", "Bearer "+c.authToken)

					// Retry the request once
					retryResp, retryErr := c.httpClient.Do(req)
					if retryErr != nil {
						return fmt.Errorf("request failed after token refresh: %w", retryErr)
					}
					defer retryResp.Body.Close()

					retryBody, retryErr := io.ReadAll(retryResp.Body)
					if retryErr != nil {
						return fmt.Errorf("read response after refresh: %w", retryErr)
					}

					// Check if retry succeeded
					if retryResp.StatusCode >= 400 {
						// Parse error from retry
						var retryApiErr APIError
						if err := json.Unmarshal(retryBody, &retryApiErr); err == nil && retryApiErr.ErrorCode != 0 {
							return &retryApiErr
						}
						return fmt.Errorf("HTTP %d after refresh: %s", retryResp.StatusCode, string(retryBody))
					}

					// Success - unmarshal result
					if result != nil {
						if err := json.Unmarshal(retryBody, result); err != nil {
							return fmt.Errorf("unmarshal response after refresh: %w", err)
						}
					}
					return nil
				}

				return &apiErr
			}

			// Fallback to generic error
			return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
		}

		if result != nil {
			if err := json.Unmarshal(body, result); err != nil {
				return fmt.Errorf("unmarshal response: %w", err)
			}
		}

		return nil
	}

	return fmt.Errorf("max retries exceeded")
}
