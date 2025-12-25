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
	baseURL    string
	authToken  string
	httpClient *http.Client
}

func NewClient(baseURL, authToken string) *Client {
	return &Client{
		baseURL:    baseURL,
		authToken:  authToken,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *Client) GetConfig(roomId string) (*Config, error) {
	url := fmt.Sprintf("%s/bbapp-config/%s", c.baseURL, roomId)

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
	url := fmt.Sprintf("%s/api/v1/stream/rooms/%s/bbapp-config", c.baseURL, roomId)

	// Wrap config in BbappConfigRequest structure
	requestBody := map[string]interface{}{
		"configData":  config,
		"description": "BBapp PK Mode Configuration",
		"isActive":    true,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	// Debug logging
	fmt.Printf("[SaveConfig] URL: %s\n", url)
	fmt.Printf("[SaveConfig] JSON length: %d bytes\n", len(jsonData))
	fmt.Printf("[SaveConfig] JSON preview: %s\n", string(jsonData[:min(200, len(jsonData))]))

	// Use the same pattern as other POST methods
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.authToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Length", fmt.Sprintf("%d", len(jsonData)))

	var resp map[string]interface{}
	return c.doRequest(req, &resp)
}

func (c *Client) StartSession(roomId string, deviceHash string) (*StartSessionResponse, error) {
	url := fmt.Sprintf("%s/pk/start-from-bbapp/%s", c.baseURL, roomId)

	reqBody := StartSessionRequest{DeviceHash: deviceHash}
	jsonData, _ := json.Marshal(reqBody)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.authToken)
	req.Header.Set("Content-Type", "application/json")

	var resp StartSessionResponse
	if err := c.doRequest(req, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

func (c *Client) StopSession(roomId, reason string) (*StopSessionResponse, error) {
	url := fmt.Sprintf("%s/pk/stop-from-bbapp/%s", c.baseURL, roomId)

	reqBody := StopSessionRequest{Reason: reason}
	jsonData, _ := json.Marshal(reqBody)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.authToken)
	req.Header.Set("Content-Type", "application/json")

	var resp StopSessionResponse
	if err := c.doRequest(req, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

func (c *Client) SendHeartbeat(roomId string, status HeartbeatRequest) error {
	url := fmt.Sprintf("%s/bbapp-status/%s", c.baseURL, roomId)

	jsonData, _ := json.Marshal(status)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
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
