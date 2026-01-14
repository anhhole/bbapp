package stomp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/go-stomp/stomp/v3"
	"github.com/gorilla/websocket"
)

// Client wraps STOMP connection with auto-reconnection
type Client struct {
	conn          *stomp.Conn
	url           string
	username      string
	password      string
	isHealthy     bool
	mutex         sync.RWMutex
	stopMonitor   chan struct{}
	reconnecting  bool
	subscriptions map[string]func([]byte) // destination -> handler
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
		// Pass token in MULTIPLE headers to be safe (login, Authorization, X-Authorization)
		// Standard STOMP
		opts = append(opts, stomp.ConnOpt.Login(c.username, c.password))

		// Common for JWT/Spring Security
		token := c.username
		if !strings.HasPrefix(token, "Bearer ") {
			token = "Bearer " + token
		}
		opts = append(opts, stomp.ConnOpt.Header("Authorization", token))
		opts = append(opts, stomp.ConnOpt.Header("X-Authorization", token))
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

	// Restore subscriptions if any exist (async to avoid deadlock if called with lock held)
	// But here we are in connect(), which is called by NewClient or reconnect.
	// We should be careful about locks.
	// If this is called from NewClient, we don't have lock on c yet?
	// NewClient calls connect() before returning instance.
	// Reconnect calls connect().

	// We'll call resubscribeAll() from the caller to be safe or ensure connect usage is consistent.
	// Actually, let's just trigger it here. We held c.mutex inside connect() just above for setting c.conn.
	// We released it at line 105. So it's safe to call resubscribeAll which takes lock.
	go c.resubscribeAll()

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
				// If reconnection fails after max retries, stop monitoring to avoid abuse
				if success := c.reconnect(); !success {
					fmt.Println("[STOMP] Reconnection failed permanently. Stopping monitor to avoid connection abuse.")
					// Ensure we are in a disconnected state
					c.Disconnect()
					return
				}
			}
		case <-c.stopMonitor:
			return
		}
	}
}

// reconnect attempts to reconnect with exponential backoff
// Returns true if reconnected successfully, false if all attempts failed
func (c *Client) reconnect() bool {
	c.mutex.Lock()
	if c.reconnecting {
		c.mutex.Unlock()
		return false
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
			return true
		}

		fmt.Printf("[STOMP] Reconnection attempt %d/%d failed\n", attempt, maxRetries)
	}

	c.mutex.Lock()
	c.reconnecting = false
	c.mutex.Unlock()

	fmt.Printf("[STOMP] ERROR: Failed to reconnect after %d attempts\n", maxRetries)
	return false
}

// dialWebSocket establishes WebSocket connection for STOMP
// token parameter is used for Bearer authentication
func dialWebSocket(urlStr, token string) (net.Conn, error) {
	fmt.Printf("[STOMP] Setting up WebSocket connection...\n")

	// Defensive: Strip "Bearer " prefix if present to avoid duplication
	if len(token) > 7 && strings.EqualFold(token[:7], "Bearer ") {
		token = strings.TrimSpace(token[7:])
		fmt.Printf("[STOMP] Stripped 'Bearer ' prefix from token\n")
	}

	// Log token detail for debugging (masked)
	if len(token) > 10 {
		fmt.Printf("[STOMP] Token provided (len=%d, start=%s...)\n", len(token), token[:5])
	} else if token != "" {
		fmt.Printf("[STOMP] Token provided (len=%d)\n", len(token))
	} else {
		fmt.Printf("[STOMP] No token provided\n")
	}

	// Normalize URL (handle scheme and path)
	wsURL, err := normalizeURL(urlStr)
	if err != nil {
		return nil, err
	}

	u, err := url.Parse(wsURL)
	if err != nil {
		return nil, err
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
// It handles SockJS framing (unwrapping received JSON arrays, wrapping sent data)
type websocketConn struct {
	ws         *websocket.Conn
	readBuffer bytes.Buffer
}

func (w *websocketConn) Read(p []byte) (n int, err error) {
	// If we have buffered data, return it first
	if w.readBuffer.Len() > 0 {
		return w.readBuffer.Read(p)
	}

	for {
		msgType, data, err := w.ws.ReadMessage()
		if err != nil {
			return 0, err
		}

		// STOMP over WebSocket uses text or binary frames
		if msgType != websocket.TextMessage && msgType != websocket.BinaryMessage {
			continue // skip unknown frame types
		}

		// Handle SockJS frames
		// SockJS frames are single characters or JSON arrays
		if len(data) == 0 {
			continue
		}

		frameType := data[0]
		switch frameType {
		case 'o': // Open frame
			continue
		case 'h': // Heartbeat frame
			// Translate SockJS heartbeat to STOMP heartbeat (newline)
			// This keeps the STOMP connection alive even if no real data is flowing
			w.readBuffer.WriteByte('\n')
			return w.readBuffer.Read(p)
		case 'c': // Close frame
			// e.g. [3000,"Go away!"]
			return 0, fmt.Errorf("SockJS connection closed by server")
		case 'a': // Data array frame: ["msg1", "msg2"]
			var messages []string
			if err := json.Unmarshal(data[1:], &messages); err != nil {
				return 0, fmt.Errorf("failed to parse SockJS data frame: %w", err)
			}

			for _, msg := range messages {
				w.readBuffer.WriteString(msg)
			}

			// Return data from buffer
			return w.readBuffer.Read(p)
		default:
			// Treat as raw frame if not SockJS
			n = copy(p, data)
			return n, nil
		}
	}
}

func (w *websocketConn) Write(p []byte) (n int, err error) {
	// Wrap in SockJS array: ["message"]
	// Use manual string construction to avoid full JSON marshaling overhead for simple string
	// payload needs to be a JSON string, so we use json.Marshal for the content part to be safe
	content, err := json.Marshal(string(p))
	if err != nil {
		return 0, err
	}

	sockJsFrame := fmt.Sprintf("[%s]", content)

	err = w.ws.WriteMessage(websocket.TextMessage, []byte(sockJsFrame))
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
		return err
	}

	fmt.Printf("[STOMP] ✓ Message published successfully\n")
	return nil
}

// Subscribe subscribes to a destination. The handler will be called when a message is received.
// Subscriptions are automatically restored on reconnection.
func (c *Client) Subscribe(destination string, handler func([]byte)) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.subscriptions == nil {
		c.subscriptions = make(map[string]func([]byte))
	}
	c.subscriptions[destination] = handler

	if c.conn != nil && c.isHealthy {
		return c.subscribeInternal(destination, handler)
	}

	// If not connected, it will be subscribed upon connection
	return nil
}

// subscribeInternal performs the actual STOMP subscription
func (c *Client) subscribeInternal(destination string, handler func([]byte)) error {
	sub, err := c.conn.Subscribe(destination, stomp.AckAuto)
	if err != nil {
		return err
	}

	fmt.Printf("[STOMP] Subscribed to %s\n", destination)

	// Start goroutine to handle messages for this subscription
	go func() {
		for {
			msg, ok := <-sub.C
			if !ok {
				fmt.Printf("[STOMP] Subscription channel closed for %s\n", destination)
				return
			}
			if msg.Err != nil {
				fmt.Printf("[STOMP] Error on subscription %s: %v\n", destination, msg.Err)
				continue
			}
			// Call handler
			go handler(msg.Body)
		}
	}()

	return nil
}

// resubscribeAll restores all subscriptions after reconnection
func (c *Client) resubscribeAll() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.conn == nil {
		return
	}

	fmt.Printf("[STOMP] Restoring %d subscriptions...\n", len(c.subscriptions))
	for dest, handler := range c.subscriptions {
		if err := c.subscribeInternal(dest, handler); err != nil {
			fmt.Printf("[STOMP] ERROR: Failed to restore subscription to %s: %v\n", dest, err)
		}
	}
}

// normalizeURL processes the input URL to ensure it's a valid WebSocket URL
// It handles http->ws conversion and SockJS path construction
func normalizeURL(urlStr string) (string, error) {
	u, err := url.Parse(urlStr)
	if err != nil {
		return "", fmt.Errorf("invalid URL: %w", err)
	}

	// Handle scheme conversion
	if u.Scheme == "http" {
		u.Scheme = "ws"
	} else if u.Scheme == "https" {
		u.Scheme = "wss"
	}

	// 1. If no path, append /ws (assuming standard BB-Core convention)
	if u.Path == "" || u.Path == "/" {
		u.Path = "/ws"
	}

	// 2. If path ends in /ws, it's likely a SockJS base URL.
	// standard gorilla/websocket doesn't speak SockJS protocol, so we must
	// construct the raw websocket endpoint manually: /base/server_id/session_id/websocket
	if strings.HasSuffix(u.Path, "/ws") {
		// Append dummy server_id (0-999) and session_id
		// Using fixed values is fine for a client connection
		u.Path = path.Join(u.Path, "999", "bbapp", "websocket")
	}

	return u.String(), nil
}
