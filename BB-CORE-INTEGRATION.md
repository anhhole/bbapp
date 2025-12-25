# BB-Core Integration Guide

## Architecture

BBapp integrates with BB-Core via:

1. **REST API** - Session lifecycle management
2. **STOMP** - Real-time event streaming
3. **WebSocket Interception** - Bigo Live protocol parsing

## API Endpoints

### GET /bbapp-config/{roomId}

Fetches room configuration including all teams and streamers.

**Headers:**
- `Authorization: Bearer {token}`

**Response:**
```json
{
  "roomId": "abc123",
  "agencyId": 1,
  "session": {...},
  "teams": [
    {
      "teamId": "team1",
      "name": "Team A",
      "bindingGift": "Rose",
      "scoreMultipliers": {...},
      "streamers": [
        {
          "streamerId": "s1",
          "bigoId": "12345",
          "bigoRoomId": "room123",
          "name": "Alice",
          "avatar": "...",
          "bindingGift": "Rose"
        }
      ]
    }
  ]
}
```

### POST /pk/start-from-bbapp/{roomId}

Starts PK session.

**Payload:**
```json
{
  "deviceHash": "abc..."
}
```

**Response:**
```json
{
  "sessionId": "session123",
  "success": true,
  "message": "Session started"
}
```

### POST /pk/stop-from-bbapp/{roomId}

Stops PK session.

**Payload:**
```json
{
  "reason": "USER_STOPPED"
}
```

### POST /bbapp-status/{roomId}

Heartbeat with connection status.

**Payload:**
```json
{
  "connections": [
    {
      "bigoRoomId": "room123",
      "streamerId": "s1",
      "status": "CONNECTED",
      "messagesReceived": 42,
      "lastMessageTime": 1234567890,
      "errorMessage": ""
    }
  ]
}
```

## STOMP Message Format

### Gift Event

**Destination:** `/app/room/{roomId}/bigo`

```json
{
  "type": "GIFT",
  "roomId": "abc123",
  "bigoRoomId": "room123",
  "senderId": "sender1",
  "senderName": "Bob",
  "senderAvatar": "...",
  "senderLevel": 25,
  "streamerId": "s1",
  "streamerName": "Alice",
  "streamerAvatar": "...",
  "giftId": "g1",
  "giftName": "Rose",
  "giftCount": 1,
  "diamonds": 100,
  "giftImageUrl": "...",
  "timestamp": 1234567890,
  "deviceHash": "abc..."
}
```

### Chat Message

```json
{
  "type": "CHAT",
  "roomId": "abc123",
  "bigoRoomId": "room123",
  "senderId": "sender1",
  "senderName": "Bob",
  "senderAvatar": "...",
  "senderLevel": 25,
  "message": "Hello!",
  "timestamp": 1234567890,
  "deviceHash": "abc..."
}
```

### Status Update

```json
{
  "type": "STATUS",
  "roomId": "abc123",
  "bigoRoomId": "room123",
  "streamerId": "s1",
  "status": "CONNECTED",
  "errorMessage": "",
  "timestamp": 1234567890
}
```

## Device Fingerprinting

BBapp generates a stable device hash based on:
- Hostname
- OS + Architecture

This hash is used for trial validation and tracking.

## Connection Resilience

### STOMP Reconnection

- Monitors connection every 10 seconds
- Auto-reconnects with exponential backoff (max 5 attempts)
- Continues from last known state

### Browser Reconnection

- Monitors frame reception every 30 seconds
- Recreates browser if no frames received
- Logs all reconnection attempts

## Error Handling

- **Auth failures:** Return clear error, do not retry
- **Network errors:** Retry with exponential backoff
- **Config errors:** Abort session start, notify user
- **STOMP errors:** Auto-reconnect, queue messages if possible
