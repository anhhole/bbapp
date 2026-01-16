package listener

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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

	// Value Stats
	RoomTotalDiamonds int64 // Accumulated diamonds for this room in current session

	// Context
	TeamId string // Resolved Team ID (optional)
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
func (b *BigoListener) Start() (string, error) {
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

	var finalUrl string
	err := chromedp.Run(b.ctx,
		network.Enable(),
		chromedp.Navigate(bigoUrl),
		chromedp.Location(&finalUrl),
	)

	if err != nil {
		fmt.Printf("[BigoListener] ERROR: Failed to start: %v\n", err)
		return "", err
	}

	// Parse resolved Room ID from URL
	// URL format: https://www.bigo.tv/12345678 or https://www.bigo.tv/user/12345678
	resolvedId := b.roomId // Default to current

	// Simple extraction: take the last path segment
	if finalUrl != "" {
		fmt.Printf("[BigoListener] Navigated to URL: %s\n", finalUrl)
		// Strip query params
		for idx := 0; idx < len(finalUrl); idx++ {
			if finalUrl[idx] == '?' {
				finalUrl = finalUrl[:idx]
				break
			}
		}

		// Extract last segment
		if idx := len(finalUrl) - 1; idx >= 0 {
			for i := len(finalUrl) - 1; i >= 0; i-- {
				if finalUrl[i] == '/' {
					candidate := finalUrl[i+1:]
					if len(candidate) > 0 {
						// Basic check if it looks like an ID (numeric)
						isNumeric := true
						for _, ch := range candidate {
							if ch < '0' || ch > '9' {
								isNumeric = false
								break
							}
						}

						if isNumeric && len(candidate) > 5 { // Room IDs are usually long numbers
							resolvedId = candidate
							b.roomId = resolvedId // Update internal state
						}
					}
					break
				}
			}
		}
	}

	fmt.Printf("[BigoListener] ✓ Listening started for room: %s (Resolved: %s)\n", b.roomId, resolvedId)
	return resolvedId, nil
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
		"roomId":        b.roomId,
		"frameCount":    b.frameCount,
		"lastFrameTime": b.lastFrameTime,
		"timeSinceLast": time.Since(b.lastFrameTime).Seconds(),
		"healthy":       b.IsHealthy(),
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
	// Example log: {"from_uid":..., "payload": {"vgift_typeid":..., "nick_name":...}}
	if payloadRaw, hasPayload := msg["payload"]; hasPayload {
		fmt.Printf("[BigoListener] Payload field found. Type: %T\n", payloadRaw)

		if payload, ok := payloadRaw.(map[string]interface{}); ok {
			// Check if this payload is a gift
			if _, hasGiftId := payload["vgift_typeid"]; hasGiftId {
				fmt.Printf("\n🎁 GENERIC PAYLOAD GIFT DETECTED! 🎁\n")

				gift, err := b.parsePayloadGift(msg, payload)
				if err != nil {
					fmt.Printf("[BigoListener] ERROR: Failed to parse payload gift: %v\n", err)
					return
				}
				fmt.Printf("[BigoListener] ✓ Gift parsed: %s sent %s (count: %d)\n",
					gift.SenderName, gift.GiftName, gift.GiftCount)

				// Notify handlers
				for _, handler := range b.giftHandlers {
					handler(gift)
				}
				return
			} else {
				fmt.Printf("[BigoListener] Payload map found but no vgift_typeid. Keys: ")
				for k := range payload {
					fmt.Printf("%s, ", k)
				}
				fmt.Printf("\n")
			}
		} else {
			fmt.Printf("[BigoListener] Payload is not a map[string]interface{}\n")
		}
	} else {
		// Only log "no payload" if we also see the 2584 prefix, to avoid noise
		if len(prefix) > 0 {
			prefixTrimmed := ""
			for _, ch := range prefix {
				if ch >= '0' && ch <= '9' {
					prefixTrimmed += string(ch)
				}
			}
			if prefixTrimmed == "2584" {
				fmt.Printf("[BigoListener] 2584 prefix but no payload field in msg. Msg keys: ")
				for k := range msg {
					fmt.Printf("%s, ", k)
				}
				fmt.Printf("\n")
			}
		}
	}

	// Check for stringified payload (legacy/alternative format)
	if payloadStr, ok := msg["payload"].(string); ok {
		var nestedMsg map[string]interface{}
		if err := json.Unmarshal([]byte(payloadStr), &nestedMsg); err == nil {
			// Recursive check or just trying to parse fields from nestedMsg
			// For now, let's treat nestedMsg as the potential payload map
			if _, hasGiftId := nestedMsg["vgift_typeid"]; hasGiftId {
				fmt.Printf("\n🎁 STRINGIFIED PAYLOAD GIFT DETECTED! 🎁\n")
				gift, err := b.parsePayloadGift(msg, nestedMsg)
				if err == nil {
					fmt.Printf("[BigoListener] ✓ Gift parsed: %s sent %s\n", gift.SenderName, gift.GiftName)
					for _, handler := range b.giftHandlers {
						handler(gift)
					}
					return
				}
			}
		}
	}

	// Check numeric prefix (2584 might indicate gifts) - keep as fallback logging
	if len(prefix) > 0 {
		prefixTrimmed := ""
		for _, ch := range prefix {
			if ch >= '0' && ch <= '9' {
				prefixTrimmed += string(ch)
			}
		}
		if prefixTrimmed == "2584" {
			// already handled above by payload inspections, just log if we missed it
			// fmt.Printf("[BigoListener] Note: Prefix 2584 detected\n")
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

// parsePayloadGift parses the generic payload structure seen in modern Bigo packets
func (b *BigoListener) parsePayloadGift(root map[string]interface{}, payload map[string]interface{}) (BigoGift, error) {
	gift := BigoGift{
		Timestamp:  time.Now().UnixMilli(),
		BigoRoomId: b.roomId,
	}

	// Sender Info
	if nickName, ok := payload["nick_name"].(string); ok {
		gift.SenderName = nickName
	}
	if fromUid, ok := root["from_uid"].(string); ok {
		gift.SenderId = fromUid
	} else if fromUid, ok := payload["from_uid"].(string); ok {
		gift.SenderId = fromUid
	}
	// Avatar might be in head_icon_url or avatar
	if headIcon, ok := payload["head_icon_url"].(string); ok {
		gift.SenderAvatar = headIcon
	}

	// Receiver Info
	if toUid, ok := payload["to_uid"].(string); ok {
		gift.StreamerId = toUid
	}

	// Gift Info
	if giftId, ok := payload["vgift_typeid"].(string); ok {
		gift.GiftId = giftId
	}
	if giftName, ok := payload["vgift_name"].(string); ok {
		gift.GiftName = giftName
	}
	if imgUrl, ok := payload["img_url"].(string); ok {
		gift.GiftImageUrl = imgUrl
	}

	// Counts
	if countStr, ok := payload["vgift_count"].(string); ok {
		fmt.Sscanf(countStr, "%d", &gift.GiftCount)
	} else if count, ok := payload["vgift_count"].(float64); ok {
		gift.GiftCount = int(count)
	}
	if gift.GiftCount == 0 {
		gift.GiftCount = 1
	}

	// Diamonds
	gift.Diamonds = 0

	// Try to get room_id from payload if missing
	if roomId, ok := payload["room_id"].(string); ok {
		if gift.BigoRoomId == "" {
			gift.BigoRoomId = roomId
		}
	}

	return gift, nil
}

// BigoUserInfo represents basic user info from Bigo API
type BigoUserInfo struct {
	Avatar   string `json:"avatar"`
	NickName string `json:"nick_name"`
	BigoID   string `json:"yyuid"`
}

// BigoUserResponse represents the API response
type BigoUserResponse struct {
	Result   int          `json:"result"`
	Data     BigoUserInfo `json:"data"`
	ErrorMsg string       `json:"errorMsg"`
}

// GetUserInfo fetches user info from Bigo's official API
func GetUserInfo(bigoId string) (*BigoUserInfo, error) {
	fmt.Printf("[BigoListener] Fetching Bigo user info for ID: %s\n", bigoId)

	url := fmt.Sprintf("https://www.bigo.tv/bigolivepay-recharge/pay-bigolive-tv/quicklyPay/getUserDetail?bigoId=%s&isFromApp=0", bigoId)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	var response BigoUserResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}

	if response.Result != 0 {
		return nil, fmt.Errorf("api error: %s", response.ErrorMsg)
	}

	return &response.Data, nil
}
