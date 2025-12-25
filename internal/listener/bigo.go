package listener

import (
	"context"
	"encoding/json"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
)

// Gift represents a Bigo gift
type Gift struct {
	BigoUid   string
	Nickname  string
	GiftName  string
	GiftValue int64
}

// GiftHandler handles gift events
type GiftHandler func(Gift)

// BigoListener listens to Bigo room WebSocket
type BigoListener struct {
	roomId       string
	ctx          context.Context
	giftHandlers []GiftHandler
}

// NewBigoListener creates new Bigo listener
func NewBigoListener(roomId string, ctx context.Context) *BigoListener {
	return &BigoListener{
		roomId:       roomId,
		ctx:          ctx,
		giftHandlers: make([]GiftHandler, 0),
	}
}

// OnGift registers gift handler
func (b *BigoListener) OnGift(handler GiftHandler) {
	b.giftHandlers = append(b.giftHandlers, handler)
}

// Start starts listening
func (b *BigoListener) Start() error {
	// Setup WebSocket frame listener
	chromedp.ListenTarget(b.ctx, func(ev interface{}) {
		switch ev := ev.(type) {
		case *network.EventWebSocketFrameReceived:
			b.handleFrame(ev.Response.PayloadData)
		}
	})

	// Navigate to Bigo room
	return chromedp.Run(b.ctx,
		network.Enable(),
		chromedp.Navigate("https://www.bigo.tv/"+b.roomId),
	)
}

// handleFrame processes WebSocket frame
func (b *BigoListener) handleFrame(data string) {
	var msg map[string]interface{}
	if err := json.Unmarshal([]byte(data), &msg); err != nil {
		return
	}

	msgType, ok := msg["type"].(string)
	if !ok || msgType != "GIFT" {
		return
	}

	// Parse gift
	gift := b.parseGift(msg)

	// Notify handlers
	for _, handler := range b.giftHandlers {
		handler(gift)
	}
}

// parseGift extracts gift data
func (b *BigoListener) parseGift(msg map[string]interface{}) Gift {
	sender := msg["sender"].(map[string]interface{})
	giftData := msg["gift"].(map[string]interface{})

	return Gift{
		BigoUid:   sender["id"].(string),
		Nickname:  sender["nickname"].(string),
		GiftName:  giftData["name"].(string),
		GiftValue: int64(giftData["diamonds"].(float64)),
	}
}
