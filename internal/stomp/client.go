package stomp

import (
	"encoding/json"
	"fmt"
	"net"
	"time"

	"github.com/go-stomp/stomp/v3"
)

// Client wraps STOMP connection
type Client struct {
	conn *stomp.Conn
}

// NewClient creates STOMP client
func NewClient(url, username, password string) (*Client, error) {
	netConn, err := net.DialTimeout("tcp", url, 10*time.Second)
	if err != nil {
		return nil, fmt.Errorf("dial failed: %w", err)
	}

	var opts []func(*stomp.Conn) error
	if username != "" {
		opts = append(opts, stomp.ConnOpt.Login(username, password))
	}

	conn, err := stomp.Connect(netConn, opts...)
	if err != nil {
		netConn.Close()
		return nil, fmt.Errorf("STOMP connect failed: %w", err)
	}

	return &Client{conn: conn}, nil
}

// Disconnect closes connection
func (c *Client) Disconnect() error {
	if c.conn != nil {
		return c.conn.Disconnect()
	}
	return nil
}

// Publish sends message to destination
func (c *Client) Publish(destination string, payload interface{}) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal failed: %w", err)
	}

	return c.conn.Send(
		destination,
		"application/json",
		data,
	)
}
