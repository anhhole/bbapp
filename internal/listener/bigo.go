package listener

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
)

// BigoGift represents a complete Bigo gift event
type BigoGift struct {
	// Sender
	SenderId     string
	SenderName   string
	SenderAvatar string
	SenderLevel  int

	// Receiver (Streamer)
	StreamerId     string
	StreamerName   string
	StreamerAvatar string

	// Gift details
	GiftId       string
	GiftName     string
	GiftCount    int
	Diamonds     int64
	GiftImageUrl string

	// Metadata
	Timestamp  int64
	BigoRoomId string
}

// Keep old Gift for compatibility during transition
type Gift = BigoGift

// GiftHandler handles gift events
type GiftHandler func(Gift)

// BigoChat represents a Bigo chat message
type BigoChat struct {
	SenderId     string
	SenderName   string
	SenderAvatar string
	SenderLevel  int
	Message      string
	Timestamp    int64
	BigoRoomId   string
}

// ChatHandler handles chat events
type ChatHandler func(BigoChat)

// BigoListener listens to Bigo room WebSocket
type BigoListener struct {
	roomId        string
	ctx           context.Context
	giftHandlers  []GiftHandler
	chatHandlers  []ChatHandler
	debugMode     bool
	debugFile     *os.File
	debugMutex    sync.Mutex
	lastFrameTime time.Time
	frameCount    int64
}

// NewBigoListener creates new Bigo listener
func NewBigoListener(roomId string, ctx context.Context) *BigoListener {
	fmt.Printf("[BigoListener] Creating listener for room: %s\n", roomId)
	return &BigoListener{
		roomId:        roomId,
		ctx:           ctx,
		giftHandlers:  make([]GiftHandler, 0),
		chatHandlers:  make([]ChatHandler, 0),
		debugMode:     false,
		lastFrameTime: time.Now(),
		frameCount:    0,
	}
}

// OnGift registers gift handler
func (b *BigoListener) OnGift(handler GiftHandler) {
	b.giftHandlers = append(b.giftHandlers, handler)
}

// OnChat registers chat handler
func (b *BigoListener) OnChat(handler ChatHandler) {
	b.chatHandlers = append(b.chatHandlers, handler)
}

// Start starts listening
func (b *BigoListener) Start() error {
	fmt.Printf("[BigoListener] Starting listener for room: %s\n", b.roomId)
	
	// Setup WebSocket frame listener
	fmt.Printf("[BigoListener] Attaching WebSocket frame interceptor...\n")
	chromedp.ListenTarget(b.ctx, func(ev interface{}) {
		switch ev := ev.(type) {
		case *network.EventWebSocketFrameReceived:
			b.frameCount++
			b.lastFrameTime = time.Now()
			b.handleFrame(ev.Response.PayloadData)
		}
	})

	// Navigate to Bigo room
	bigoUrl := "https://www.bigo.tv/" + b.roomId
	fmt.Printf("[BigoListener] Navigating to: %s\n", bigoUrl)
	
	err := chromedp.Run(b.ctx,
		network.Enable(),
		chromedp.Navigate(bigoUrl),
	)
	
	if err != nil {
		fmt.Printf("[BigoListener] ERROR: Failed to start: %v\n", err)
		return err
	}
	
	fmt.Printf("[BigoListener] ✓ Listening started for room: %s\n", b.roomId)
	return nil
}

// EnableDebugMode enables raw frame capture to file
func (b *BigoListener) EnableDebugMode(filepath string) error {
	fmt.Printf("[BigoListener] Enabling debug mode: %s\n", filepath)
	
	file, err := os.OpenFile(filepath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open debug file: %w", err)
	}
	
	b.debugMode = true
	b.debugFile = file
	
	fmt.Printf("[BigoListener] ✓ Debug mode enabled (all WebSocket frames will be captured)\n")
	return nil
}

// DisableDebugMode disables debug mode
func (b *BigoListener) DisableDebugMode() {
	if b.debugFile != nil {
		b.debugFile.Close()
		b.debugFile = nil
	}
	b.debugMode = false
	fmt.Printf("[BigoListener] Debug mode disabled\n")
}

// IsHealthy checks if connection is receiving frames
func (b *BigoListener) IsHealthy() bool {
	timeSinceLastFrame := time.Since(b.lastFrameTime)
	healthy := timeSinceLastFrame < 30*time.Second
	
	if !healthy {
		fmt.Printf("[BigoListener] WARNING: No frames received for %v (room: %s)\n", timeSinceLastFrame, b.roomId)
	}
	
	return healthy
}

// GetStats returns listener statistics
func (b *BigoListener) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"roomId":         b.roomId,
		"frameCount":     b.frameCount,
		"lastFrameTime":  b.lastFrameTime,
		"timeSinceLast":  time.Since(b.lastFrameTime).Seconds(),
		"healthy":        b.IsHealthy(),
	}
}

// handleFrame processes WebSocket frame
func (b *BigoListener) handleFrame(data string) {
	// Debug mode: log all frames to file
	if b.debugMode && b.debugFile != nil {
		b.debugMutex.Lock()
		fmt.Fprintf(b.debugFile, "\n========== Frame #%d [%s] ==========\n", 
			b.frameCount, time.Now().Format(time.RFC3339))
		fmt.Fprintf(b.debugFile, "Room: %s\n", b.roomId)
		fmt.Fprintf(b.debugFile, "Raw Data: %s\n", data)
		b.debugMutex.Unlock()
	}
	
	// Log frame reception every 100 frames
	if b.frameCount%100 == 0 {
		fmt.Printf("[BigoListener] WebSocket frames received: %d (room: %s)\n", b.frameCount, b.roomId)
	}
	
	// Show FULL raw packet for first 20 frames
	if b.frameCount <= 20 {
		fmt.Printf("\n========== RAW PACKET #%d (Room: %s) ==========\n", b.frameCount, b.roomId)
		fmt.Printf("Length: %d bytes\n", len(data))
		fmt.Printf("Content: %s\n", data)
		fmt.Printf("==============================================\n\n")
	}
	
	// Bigo frames have numeric prefix + whitespace + JSON
	// Example: "2584        {"from_uid":"0","seqId":"2329098922"...}"
	// Find the start of JSON (first '{' character)
	jsonStart := -1
	for i, ch := range data {
		if ch == '{' {
			jsonStart = i
			break
		}
	}
	
	if jsonStart == -1 {
		// No JSON found
		if b.frameCount <= 5 {
			fmt.Printf("[BigoListener] No JSON found in frame (size: %d bytes): %s\n", 
				len(data), data[:min(100, len(data))])
		}
		return
	}
	
	// Extract prefix and JSON portions
	prefix := data[:jsonStart]
	jsonData := data[jsonStart:]
	
	// Show prefix for first 20 frames
	if b.frameCount <= 20 {
		fmt.Printf("[BigoListener] Prefix: '%s' | JSON starts at byte %d\n", prefix, jsonStart)
	}
	
	var msg map[string]interface{}
	if err := json.Unmarshal([]byte(jsonData), &msg); err != nil {
		// Still not valid JSON
		if b.frameCount <= 20 {
			fmt.Printf("[BigoListener] ⚠️ Invalid JSON after stripping prefix: %v\n", err)
		}
		return
	}

	// Successfully parsed! Log every 50th message
	if b.frameCount%50 == 0 {
		fmt.Printf("[BigoListener] ✓ Frame #%d parsed successfully for room %s\n", b.frameCount, b.roomId)
	}

	// Check for gift messages
	// Bigo's real protocol might use different field names - let's check common patterns
	msgType, hasType := msg["type"].(string)
	
	// Check for gift indicators in Bigo's actual protocol
	if hasType && msgType == "GIFT" {
		fmt.Printf("\n🎁🎁🎁 GIFT MESSAGE DETECTED (type=GIFT)! 🎁🎁🎁\n")
		fmt.Printf("Full packet: %s\n", data)
		fmt.Printf("Parsed JSON: %+v\n\n", msg)
		
		gift, err := b.parseGift(msg)
		if err != nil {
			fmt.Printf("[BigoListener] ERROR: Failed to parse gift: %v\n", err)
			return
		}
		fmt.Printf("[BigoListener] ✓ Gift parsed: %s sent %s (%d diamonds)\n",
			gift.SenderName, gift.GiftName, gift.Diamonds)
		
		// Notify handlers
		for _, handler := range b.giftHandlers {
			handler(gift)
		}
		return
	}

	// Check for chat messages
	if hasType && msgType == "CHAT" {
		fmt.Printf("\n💬 CHAT MESSAGE DETECTED! 💬\n")
		fmt.Printf("Parsed JSON: %+v\n\n", msg)

		chat, err := b.parseChat(msg)
		if err != nil {
			fmt.Printf("[BigoListener] ERROR: Failed to parse chat: %v\n", err)
			return
		}

		fmt.Printf("[BigoListener] ✓ Chat parsed: %s said: %s\n", chat.SenderName, chat.Message)

		// Notify handlers
		for _, handler := range b.chatHandlers {
			handler(chat)
		}
		return
	}

	// Check for alternative gift indicators (Bigo might use different field names)
	// Common patterns: msgType, action, event, cmd, payload.type, etc.
	
	// Check numeric prefix (2584 might indicate gifts)
	if len(prefix) > 0 {
		// Try to extract the message type code from prefix
		prefixTrimmed := ""
		for _, ch := range prefix {
			if ch >= '0' && ch <= '9' {
				prefixTrimmed += string(ch)
			}
		}
		
		if prefixTrimmed == "2584" {
			fmt.Printf("\n🎁 POSSIBLE GIFT (prefix=2584)! 🎁\n")
			fmt.Printf("Full packet: %s\n", data)
			
			// Check if there's a nested payload
			if payload, ok := msg["payload"].(string); ok {
				fmt.Printf("Nested payload detected, trying to parse...\n")
				var nestedMsg map[string]interface{}
				if err := json.Unmarshal([]byte(payload), &nestedMsg); err == nil {
					fmt.Printf("Nested JSON: %+v\n", nestedMsg)
					msg = nestedMsg // Use nested message for further processing
				}
			}
			
			fmt.Printf("Parsed JSON: %+v\n\n", msg)
		}
	}
	
	// Log structure for first 20 frames to understand protocol
	if b.frameCount <= 20 {
		fmt.Printf("[BigoListener] Frame #%d fields: ", b.frameCount)
		for key := range msg {
			fmt.Printf("%s ", key)
		}
		fmt.Printf("\n")
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// parseGift extracts gift data
func (b *BigoListener) parseGift(msg map[string]interface{}) (BigoGift, error) {
	gift := BigoGift{
		Timestamp:  time.Now().UnixMilli(),
		BigoRoomId: b.roomId,
	}

	// Extract sender info
	if sender, ok := msg["sender"].(map[string]interface{}); ok {
		gift.SenderId, _ = sender["id"].(string)
		gift.SenderName, _ = sender["nickname"].(string)
		gift.SenderAvatar, _ = sender["avatar"].(string)

		if level, ok := sender["level"].(float64); ok {
			gift.SenderLevel = int(level)
		}
	} else {
		return gift, fmt.Errorf("missing sender field")
	}

	// Extract receiver (streamer) info
	if receiver, ok := msg["receiver"].(map[string]interface{}); ok {
		gift.StreamerId, _ = receiver["id"].(string)
		gift.StreamerName, _ = receiver["nickname"].(string)
		gift.StreamerAvatar, _ = receiver["avatar"].(string)
	}

	// Extract gift details
	if giftData, ok := msg["gift"].(map[string]interface{}); ok {
		gift.GiftId, _ = giftData["id"].(string)
		gift.GiftName, _ = giftData["name"].(string)
		gift.GiftImageUrl, _ = giftData["image"].(string)

		if count, ok := giftData["count"].(float64); ok {
			gift.GiftCount = int(count)
		}

		if diamonds, ok := giftData["diamonds"].(float64); ok {
			gift.Diamonds = int64(diamonds)
		}
	} else {
		return gift, fmt.Errorf("missing gift field")
	}

	return gift, nil
}

// parseChat extracts chat message data
func (b *BigoListener) parseChat(msg map[string]interface{}) (BigoChat, error) {
	chat := BigoChat{
		Timestamp:  time.Now().UnixMilli(),
		BigoRoomId: b.roomId,
	}

	// Extract sender info
	if sender, ok := msg["sender"].(map[string]interface{}); ok {
		chat.SenderId, _ = sender["id"].(string)
		chat.SenderName, _ = sender["nickname"].(string)
		chat.SenderAvatar, _ = sender["avatar"].(string)

		if level, ok := sender["level"].(float64); ok {
			chat.SenderLevel = int(level)
		}
	} else {
		return chat, fmt.Errorf("missing sender field")
	}

	// Extract message
	if message, ok := msg["message"].(string); ok {
		chat.Message = message
	} else {
		return chat, fmt.Errorf("missing message field")
	}

	return chat, nil
}
