package stomp

import (
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/go-stomp/stomp/v3"
	"github.com/gorilla/websocket"
)

// Client wraps STOMP connection with auto-reconnection
type Client struct {
	conn         *stomp.Conn
	url          string
	username     string
	password     string
	isHealthy    bool
	mutex        sync.RWMutex
	stopMonitor  chan struct{}
	reconnecting bool
}

// NewClient creates STOMP client with auto-reconnection
// Supports both raw TCP (host:port) and WebSocket (ws://host:port/path or http://host:port/path)
// For WebSocket connections requiring auth, pass token as username parameter
func NewClient(urlStr, username, password string) (*Client, error) {
	client := &Client{
		url:         urlStr,
		username:    username,
		password:    password,
		stopMonitor: make(chan struct{}),
	}

	if err := client.connect(); err != nil {
		return nil, err
	}

	go client.monitorConnection()

	return client, nil
}

// connect establishes STOMP connection
func (c *Client) connect() error {
	fmt.Printf("[STOMP] Connecting to: %s\n", c.url)

	var netConn net.Conn
	var err error

	// Check if it's a WebSocket URL
	if strings.HasPrefix(c.url, "ws://") || strings.HasPrefix(c.url, "wss://") ||
		strings.HasPrefix(c.url, "http://") || strings.HasPrefix(c.url, "https://") {
		fmt.Printf("[STOMP] Using WebSocket transport\n")
		if c.username != "" {
			fmt.Printf("[STOMP] Authentication token provided (length: %d)\n", len(c.username))
		}
		// WebSocket connection (username is used as auth token for WS)
		netConn, err = dialWebSocket(c.url, c.username)
	} else {
		fmt.Printf("[STOMP] Using raw TCP transport\n")
		// Raw TCP connection
		netConn, err = net.DialTimeout("tcp", c.url, 10*time.Second)
	}

	if err != nil {
		fmt.Printf("[STOMP] ERROR: Connection failed: %v\n", err)
		return fmt.Errorf("dial failed: %w", err)
	}

	fmt.Printf("[STOMP] ✓ Network connection established\n")

	var opts []func(*stomp.Conn) error
	if c.username != "" {
		opts = append(opts, stomp.ConnOpt.Login(c.username, c.password))
	}

	fmt.Printf("[STOMP] Performing STOMP handshake...\n")
	conn, err := stomp.Connect(netConn, opts...)
	if err != nil {
		fmt.Printf("[STOMP] ERROR: STOMP handshake failed: %v\n", err)
		netConn.Close()
		return fmt.Errorf("STOMP connect failed: %w", err)
	}

	c.mutex.Lock()
	c.conn = conn
	c.isHealthy = true
	c.mutex.Unlock()

	fmt.Printf("[STOMP] ✓ STOMP connection established successfully\n")
	return nil
}

// monitorConnection monitors health and reconnects if needed
func (c *Client) monitorConnection() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.mutex.RLock()
			healthy := c.isHealthy
			c.mutex.RUnlock()

			if !healthy {
				c.reconnect()
			}
		case <-c.stopMonitor:
			return
		}
	}
}

// reconnect attempts to reconnect with exponential backoff
func (c *Client) reconnect() {
	c.mutex.Lock()
	if c.reconnecting {
		c.mutex.Unlock()
		return
	}
	c.reconnecting = true
	c.mutex.Unlock()

	fmt.Printf("[STOMP] Connection lost, attempting reconnection...\n")

	maxRetries := 5
	for attempt := 1; attempt <= maxRetries; attempt++ {
		time.Sleep(time.Duration(attempt) * 2 * time.Second)

		if err := c.connect(); err == nil {
			c.mutex.Lock()
			c.reconnecting = false
			c.mutex.Unlock()
			fmt.Printf("[STOMP] ✓ Reconnected successfully\n")
			return
		}

		fmt.Printf("[STOMP] Reconnection attempt %d/%d failed\n", attempt, maxRetries)
	}

	c.mutex.Lock()
	c.reconnecting = false
	c.mutex.Unlock()

	fmt.Printf("[STOMP] ERROR: Failed to reconnect after %d attempts\n", maxRetries)
}

// dialWebSocket establishes WebSocket connection for STOMP
// token parameter is used for Bearer authentication
func dialWebSocket(urlStr, token string) (net.Conn, error) {
	fmt.Printf("[STOMP] Setting up WebSocket connection...\n")

	// Convert http:// to ws:// if needed
	if strings.HasPrefix(urlStr, "http://") {
		urlStr = "ws://" + strings.TrimPrefix(urlStr, "http://")
		fmt.Printf("[STOMP] Converted http:// to ws://\n")
	} else if strings.HasPrefix(urlStr, "https://") {
		urlStr = "wss://" + strings.TrimPrefix(urlStr, "https://")
		fmt.Printf("[STOMP] Converted https:// to wss://\n")
	}

	// Parse URL to ensure it's valid
	u, err := url.Parse(urlStr)
	if err != nil {
		fmt.Printf("[STOMP] ERROR: Invalid URL: %v\n", err)
		return nil, fmt.Errorf("invalid WebSocket URL: %w", err)
	}

	// Add token as query parameter if provided
	if token != "" {
		query := u.Query()
		query.Set("token", token)
		u.RawQuery = query.Encode()
		fmt.Printf("[STOMP] Added authentication token to query parameters\n")
	}

	// Prepare headers
	headers := make(map[string][]string)
	if token != "" {
		headers["Authorization"] = []string{"Bearer " + token}
		fmt.Printf("[STOMP] Added Authorization header\n")
	}

	// Connect to WebSocket
	fmt.Printf("[STOMP] Dialing WebSocket: %s\n", u.String())
	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
		Subprotocols:     []string{"v12.stomp", "v11.stomp", "v10.stomp"},
	}

	wsConn, resp, err := dialer.Dial(u.String(), headers)
	if err != nil {
		if resp != nil {
			fmt.Printf("[STOMP] ERROR: WebSocket handshake failed (status: %d)\n", resp.StatusCode)
		} else {
			fmt.Printf("[STOMP] ERROR: WebSocket connection failed: %v\n", err)
		}
		return nil, fmt.Errorf("WebSocket dial failed: %w", err)
	}

	fmt.Printf("[STOMP] ✓ WebSocket connection established\n")
	if resp != nil {
		fmt.Printf("[STOMP] WebSocket handshake status: %d %s\n", resp.StatusCode, resp.Status)
	}

	// Wrap WebSocket connection to implement net.Conn interface
	return &websocketConn{ws: wsConn}, nil
}

// websocketConn wraps gorilla/websocket to implement net.Conn
type websocketConn struct {
	ws *websocket.Conn
}

func (w *websocketConn) Read(p []byte) (n int, err error) {
	msgType, data, err := w.ws.ReadMessage()
	if err != nil {
		return 0, err
	}
	// STOMP over WebSocket uses text or binary frames
	if msgType != websocket.TextMessage && msgType != websocket.BinaryMessage {
		return 0, fmt.Errorf("unexpected WebSocket message type: %d", msgType)
	}
	n = copy(p, data)
	return n, nil
}

func (w *websocketConn) Write(p []byte) (n int, err error) {
	err = w.ws.WriteMessage(websocket.TextMessage, p)
	if err != nil {
		return 0, err
	}
	return len(p), nil
}

func (w *websocketConn) Close() error {
	return w.ws.Close()
}

func (w *websocketConn) LocalAddr() net.Addr {
	return w.ws.LocalAddr()
}

func (w *websocketConn) RemoteAddr() net.Addr {
	return w.ws.RemoteAddr()
}

func (w *websocketConn) SetDeadline(t time.Time) error {
	if err := w.ws.SetReadDeadline(t); err != nil {
		return err
	}
	return w.ws.SetWriteDeadline(t)
}

func (w *websocketConn) SetReadDeadline(t time.Time) error {
	return w.ws.SetReadDeadline(t)
}

func (w *websocketConn) SetWriteDeadline(t time.Time) error {
	return w.ws.SetWriteDeadline(t)
}

// Disconnect closes connection
func (c *Client) Disconnect() error {
	fmt.Println("[STOMP] Disconnecting from STOMP server...")

	close(c.stopMonitor)

	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.conn != nil {
		err := c.conn.Disconnect()
		if err != nil {
			fmt.Printf("[STOMP] ERROR: Disconnect failed: %v\n", err)
			return err
		}
		fmt.Println("[STOMP] ✓ Disconnected successfully")
		return nil
	}

	fmt.Println("[STOMP] Already disconnected")
	return nil
}

// Publish sends message to destination
func (c *Client) Publish(destination string, payload interface{}) error {
	c.mutex.RLock()
	conn := c.conn
	c.mutex.RUnlock()

	if conn == nil {
		return fmt.Errorf("not connected")
	}

	fmt.Printf("[STOMP] Publishing to: %s\n", destination)

	data, err := json.Marshal(payload)
	if err != nil {
		fmt.Printf("[STOMP] ERROR: Failed to marshal payload: %v\n", err)
		return fmt.Errorf("marshal failed: %w", err)
	}

	fmt.Printf("[STOMP] Payload size: %d bytes\n", len(data))

	err = conn.Send(
		destination,
		"application/json",
		data,
	)

	if err != nil {
		c.mutex.Lock()
		c.isHealthy = false
		c.mutex.Unlock()
		fmt.Printf("[STOMP] ERROR: Failed to send message: %v\n", err)
		return err
	}

	fmt.Printf("[STOMP] ✓ Message published successfully\n")
	return nil
}
