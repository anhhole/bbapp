# BBapp API Reference - Complete Endpoint Documentation

**Version:** 1.0
**Last Updated:** December 27, 2025
**Base URL:** `http://localhost:8080` (Development)

---

## Table of Contents

1. [Authentication Endpoints](#authentication-endpoints)
2. [External/BBapp Endpoints](#externalbbapp-endpoints)
3. [Script Management Endpoints](#script-management-endpoints)
4. [WebSocket Endpoints](#websocket-endpoints)
5. [Admin Endpoints](#admin-endpoints-optional)
6. [Request/Response Models](#requestresponse-models)
7. [Error Codes](#error-codes)
8. [Integration Flow](#integration-flow)

---

## Authentication Endpoints

### Base Path: `/api/v1/auth`

| Method | Endpoint | Description | Auth Required |
|--------|----------|-------------|---------------|
| `GET` | `/health` | Health check | ❌ No |
| `POST` | `/login` | User login | ❌ No |
| `POST` | `/register` | User registration | ❌ No |
| `POST` | `/refresh-token` | Refresh access token | ❌ No |

---

### 1. Health Check

**GET** `/api/v1/auth/health`

**Description:** Check if auth service is running

**Response:**
```
Auth service is running
```

---

### 2. User Login

**POST** `/api/v1/auth/login`

**Description:** Authenticate user and get JWT tokens

**Request Body:**
```json
{
  "username": "john_doe",
  "password": "SecurePass123!"
}
```

**Validation:**
- `username`: 3-50 characters, required
- `password`: 6-100 characters, required

**Response:** `200 OK`
```json
{
  "accessToken": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "refreshToken": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "tokenType": "Bearer",
  "expiresIn": 86400000,
  "expiresAt": "2025-12-28T12:00:00.000Z",
  "user": {
    "id": 123,
    "username": "john_doe",
    "email": "john@example.com",
    "firstName": "John",
    "lastName": "Doe",
    "roleCode": "OWNER"
  },
  "agency": {
    "id": 456,
    "name": "My Agency",
    "plan": "TRIAL",
    "status": "ACTIVE",
    "maxRooms": 1,
    "currentRooms": 0,
    "expiresAt": "2025-12-31T23:59:59.000Z"
  }
}
```

**Error Responses:**
- `401 Unauthorized` - Invalid credentials
- `400 Bad Request` - Validation error

---

### 3. User Registration

**POST** `/api/v1/auth/register`

**Description:** Register new user and create agency

**Request Body:**
```json
{
  "username": "john_doe",
  "email": "john@example.com",
  "password": "SecurePass123!",
  "firstName": "John",
  "lastName": "Doe",
  "agencyName": "My Agency"
}
```

**Validation:**
- `username`: 3-50 characters, required
- `email`: Valid email, required
- `password`: 8-100 characters, required
- `agencyName`: 2-100 characters, required
- `firstName`, `lastName`: Optional

**Response:** `200 OK` (Same as login response)

**Error Responses:**
- `400 Bad Request` - Validation error
- `409 Conflict` - Username/email already exists

---

### 4. Refresh Token

**POST** `/api/v1/auth/refresh-token`

**Description:** Get new access token using refresh token

**Request Body:**
```json
{
  "refreshToken": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

**Response:** `200 OK` (Same as login response with new tokens)

**Error Responses:**
- `401 Unauthorized` - Invalid or expired refresh token

**Token Lifetimes:**
- Access Token: 24 hours (86400000 ms)
- Refresh Token: 7 days (604800000 ms)

---

## External/BBapp Endpoints

### Base Path: `/api/v1/external`

**Authorization:** All endpoints require `OWNER` or `ADMIN` role

**Headers Required:**
```http
Authorization: Bearer {accessToken}
Content-Type: application/json
```

| Method | Endpoint | Description | Auth Required |
|--------|----------|-------------|---------------|
| `GET` | `/config` | Get BBapp configuration | ✅ OWNER/ADMIN |
| `POST` | `/config` | Update BBapp configuration | ✅ OWNER/ADMIN |
| `POST` | `/validate-trial` | Validate trial restrictions | ✅ OWNER/ADMIN |
| `POST` | `/heartbeat` | Send connection heartbeat | ✅ OWNER/ADMIN |

---

### 1. Get BBapp Configuration

**GET** `/api/v1/external/config`

**Description:** Get team/streamer mapping for BBapp to connect to Bigo rooms

**Response:** `200 OK`
```json
{
  "roomId": "default",
  "agencyId": 456,
  "session": {
    "sessionId": "550e8400-e29b-41d4-a716-446655440000",
    "status": "ACTIVE",
    "startedAt": 1735214400000,
    "endsAt": 1735218000000
  },
  "teams": [
    {
      "teamId": "e7a3c2f1-8b9d-4c5e-a6f3-1234567890ab",
      "name": "Team Red",
      "avatar": "https://example.com/team-red.png",
      "bindingGift": "Kiss",
      "scoreMultipliers": {
        "Kiss": 100,
        "Heart": 50,
        "Rose": 200
      },
      "streamers": [
        {
          "streamerId": 1,
          "bigoId": "829454322",
          "bigoRoomId": "7478500464273093441",
          "name": "Bella",
          "avatar": "https://example.com/bella.png",
          "bindingGift": "Rose"
        }
      ]
    }
  ]
}
```

**Usage:**
1. Call this after authentication
2. Parse team/streamer mapping
3. Connect to each `bigoRoomId`
4. Map incoming messages using `bigoId` → `teamId`

---

### 2. Update BBapp Configuration

**POST** `/api/v1/external/config`

**Description:** Create or update team/streamer configuration

**Request Body:**
```json
{
  "roomId": "default",
  "configData": {
    "bigoAccounts": [
      {
        "username": "user1",
        "cookie": "session_cookie_here"
      }
    ],
    "rooms": ["room123", "room456"],
    "stompUrl": "ws://localhost:8080/ws",
    "loggingEnabled": true
  },
  "description": "Updated BBapp configuration",
  "isActive": true
}
```

**Validation:**
- `configData`: Required (JSON object)
- `roomId`: Optional (defaults to "default")
- `description`: Optional
- `isActive`: Optional (defaults to true)

**Response:** `200 OK` (Same as GET config)

---

### 3. Validate Trial Restrictions

**POST** `/api/v1/external/validate-trial`

**Description:** Check if Bigo IDs can be used for trial accounts (anti-abuse)

**Request Body:**
```json
{
  "deviceHash": "abc123def456",
  "ipAddress": "192.168.1.100",
  "streamers": [
    {
      "bigoId": "829454322",
      "bigoRoomId": "7478500464273093441"
    },
    {
      "bigoId": "123456789",
      "bigoRoomId": "7269255640400014299"
    }
  ]
}
```

**Validation (Currently Bypassed):**
- `deviceHash`: Required (currently optional due to bypass)
- `ipAddress`: Required (currently optional due to bypass)
- `streamers`: Required, at least one (currently optional due to bypass)

**Response (Allowed):** `200 OK`
```json
{
  "allowed": true,
  "message": "All streamers validated",
  "blockedBigoIds": [],
  "reason": null
}
```

**Response (Rejected):** `200 OK`
```json
{
  "allowed": false,
  "message": "Bigo ID 829454322 has already been used in a trial",
  "blockedBigoIds": ["829454322"],
  "reason": "TRIAL_BIGO_ID_USED"
}
```

**Business Logic:**
- **TRIAL accounts:** Validates Bigo IDs haven't been used before
- **PAID/PROFESSIONAL/ENTERPRISE:** Always allowed

**When to Call:** Before connecting to Bigo rooms

---

### 4. Send Heartbeat

**POST** `/api/v1/external/heartbeat`

**Description:** Send periodic connection status updates for monitoring

**Request Body:**
```json
{
  "timestamp": 1735214400000,
  "connections": [
    {
      "bigoId": "829454322",
      "status": "CONNECTED",
      "lastMessageAt": 1735214400000,
      "messagesReceived": 45
    },
    {
      "bigoId": "123456789",
      "status": "DISCONNECTED",
      "lastMessageAt": null,
      "messagesReceived": 0
    }
  ]
}
```

**Response:** `200 OK`
```json
{
  "acknowledged": true
}
```

**Recommended Interval:** Every 30 seconds

---

## Script Management Endpoints

### Base Path: `/api/v1/scripts`

**Authorization:** All endpoints require `OWNER` or `ADMIN` role

| Method | Endpoint | Description | Auth Required |
|--------|----------|-------------|---------------|
| `POST` | `/start` | Start script session | ✅ OWNER/ADMIN |
| `POST` | `/stop` | Stop script session | ✅ OWNER/ADMIN |
| `POST` | `/pause` | Pause script session | ✅ OWNER/ADMIN |
| `POST` | `/resume` | Resume script session | ✅ OWNER/ADMIN |
| `GET` | `/{sessionId}` | Get script status | ✅ OWNER/ADMIN |
| `POST` | `/{sessionId}/next-round` | Next round (CHAMP only) | ✅ OWNER/ADMIN |
| `GET` | `/{sessionId}/ranking` | Get ranking (CHAT_RANKING only) | ✅ OWNER/ADMIN |

**Supported Script Types:**
- `PK` - PK mode script
- `CHAMP` - Champion ranking script
- `CHAT_RANKING` - Chat ranking script

---

### 1. Start Script Session

**POST** `/api/v1/scripts/start`

**Description:** Start a new script session

**Request Body (PK):**
```json
{
  "roomId": "default",
  "scriptType": "PK",
  "durationMinutes": 60,
  "scriptPayload": {
    "minTeams": 2
  }
}
```

**Request Body (CHAMP):**
```json
{
  "roomId": "default",
  "scriptType": "CHAMP",
  "durationMinutes": 120,
  "scriptPayload": {
    "idolId": 1,
    "totalRounds": 3,
    "roundDurationMinutes": 40
  }
}
```

**Request Body (CHAT_RANKING):**
```json
{
  "roomId": "default",
  "scriptType": "CHAT_RANKING",
  "durationMinutes": 30,
  "scriptPayload": {
    "maxVoteValue": 100,
    "minVotesToRank": 5
  }
}
```

**Response:** `200 OK`
```json
{
  "sessionId": "550e8400-e29b-41d4-a716-446655440000",
  "roomId": "default",
  "scriptType": "PK",
  "status": "ACTIVE",
  "startedAt": 1735214400000,
  "endsAt": 1735218000000,
  "durationMinutes": 60
}
```

---

### 2. Stop Script Session

**POST** `/api/v1/scripts/stop`

**Description:** Stop an active script session

**Request Body:**
```json
{
  "sessionId": "550e8400-e29b-41d4-a716-446655440000"
}
```

**Response:** `200 OK`
```json
{
  "sessionId": "550e8400-e29b-41d4-a716-446655440000",
  "roomId": "default",
  "scriptType": "PK",
  "status": "COMPLETED",
  "startedAt": 1735214400000,
  "endedAt": 1735218000000,
  "finalData": {
    "teamScores": [
      {
        "teamId": "e7a3c2f1-8b9d-4c5e-a6f3-1234567890ab",
        "teamName": "Team Red",
        "score": 150000
      }
    ]
  }
}
```

---

### 3. Pause Script Session

**POST** `/api/v1/scripts/pause`

**Request Body:**
```json
{
  "sessionId": "550e8400-e29b-41d4-a716-446655440000"
}
```

**Response:** `200 OK` (Script session response with status "PAUSED")

---

### 4. Resume Script Session

**POST** `/api/v1/scripts/resume`

**Request Body:**
```json
{
  "sessionId": "550e8400-e29b-41d4-a716-446655440000"
}
```

**Response:** `200 OK` (Script session response with status "ACTIVE")

---

### 5. Get Script Status

**GET** `/api/v1/scripts/{sessionId}`

**Parameters:**
- `sessionId` (UUID) - Script session ID

**Response:** `200 OK` (Script session response)

---

### 6. Next Round (CHAMP only)

**POST** `/api/v1/scripts/{sessionId}/next-round`

**Description:** Advance to next round in CHAMP script

**Parameters:**
- `sessionId` (UUID) - CHAMP script session ID

**Response:** `200 OK`
```json
{
  "roundNumber": 2,
  "totalRounds": 3,
  "currentRank": 5
}
```

**Error:** `400 Bad Request` if not a CHAMP script

---

### 7. Get Ranking (CHAT_RANKING only)

**GET** `/api/v1/scripts/{sessionId}/ranking`

**Description:** Get current ranking for CHAT_RANKING script

**Parameters:**
- `sessionId` (UUID) - CHAT_RANKING script session ID

**Response:** `200 OK`
```json
{
  "topSupported": [
    {
      "supporterId": "user123",
      "supporterName": "John Doe",
      "votes": 150,
      "rank": 1
    }
  ],
  "totalVotes": 500,
  "uniqueVoters": 45
}
```

**Error:** `400 Bad Request` if not a CHAT_RANKING script

---

## WebSocket Endpoints

### Connection

**WebSocket URL:** `ws://localhost:8080/ws`

**Authentication Options:**

1. **Query Parameter:**
```
ws://localhost:8080/ws?token={accessToken}
```

2. **Connection Headers (STOMP):**
```javascript
{
  Authorization: 'Bearer {accessToken}'
}
```

### STOMP Destinations

#### Send Messages (Application Destinations)

**Prefix:** `/app`

| Destination | Description |
|-------------|-------------|
| `/app/room/{roomId}/bigo` | Send GIFT/CHAT/STATUS messages from BBapp |

#### Subscribe (Topic Destinations)

**Prefix:** `/topic`

| Destination | Description |
|-------------|-------------|
| `/topic/room/{roomId}/pk` | Receive PK score updates |
| `/topic/room/{roomId}/activity` | Receive activity feed updates |
| `/user/queue/notifications` | Receive user-specific notifications |

---

### Message Types

#### 1. GIFT Message

**Send to:** `/app/room/{roomId}/bigo`

**Payload:**
```json
{
  "type": "GIFT",
  "roomId": "default",
  "bigoRoomId": "7478500464273093441",
  "senderId": "859856177",
  "senderName": "༄༂🎠h✺rse༂࿐",
  "senderAvatar": "https://esx.bigo.sg/...",
  "senderLevel": 30,
  "streamerId": "829454322",
  "streamerName": "Bella",
  "streamerAvatar": "https://...",
  "giftId": "10086",
  "giftName": "Kiss",
  "giftCount": 1,
  "diamonds": 655931,
  "giftImageUrl": "https://giftesx.bigo.sg/...",
  "timestamp": 1735214400000,
  "deviceHash": "abc123def456"
}
```

**Required Fields:**
- `type`, `roomId`, `bigoRoomId`
- `senderId`, `streamerId`
- `giftId`, `giftName`, `giftCount`, `diamonds`
- `timestamp`, `deviceHash`

---

#### 2. CHAT Message

**Send to:** `/app/room/{roomId}/bigo`

**Payload:**
```json
{
  "type": "CHAT",
  "roomId": "default",
  "bigoRoomId": "7269255640400014299",
  "senderId": "818823724",
  "senderName": "ᴾᴿᴹ🌞ฤɉɉฤ๐𝚝🦋ᵐ",
  "senderAvatar": "https://...",
  "senderLevel": 30,
  "message": "wow si himig",
  "timestamp": 1735214500000,
  "deviceHash": "abc123def456"
}
```

---

#### 3. STATUS Message

**Send to:** `/app/room/{roomId}/bigo`

**Payload:**
```json
{
  "type": "STATUS",
  "roomId": "default",
  "streamerId": "829454322",
  "bigoRoomId": "7478500464273093441",
  "status": "CONNECTED",
  "errorMessage": null,
  "timestamp": 1735214600000
}
```

**Status Values:**
- `CONNECTED`
- `DISCONNECTED`
- `ERROR`

---

#### 4. PK Sync (Received from Backend)

**Subscribe to:** `/topic/room/{roomId}/pk`

**Payload:**
```json
{
  "type": "PK_SYNC",
  "payload": {
    "teamScores": [
      {
        "teamId": "e7a3c2f1-8b9d-4c5e-a6f3-1234567890ab",
        "teamName": "Team Red",
        "score": 150000,
        "avatar": "https://..."
      },
      {
        "teamId": "f8b4d3e2-9c0e-5d6f-b7g4-2345678901bc",
        "teamName": "Team Blue",
        "score": 120000,
        "avatar": "https://..."
      }
    ],
    "leader": "e7a3c2f1-8b9d-4c5e-a6f3-1234567890ab",
    "totalScore": 270000,
    "activities": [
      {
        "timestamp": 1735214400000,
        "type": "GIFT",
        "message": "༄༂🎠h✺rse༂࿐ sent Kiss x1 to Bella",
        "metadata": {
          "giftName": "Kiss",
          "diamonds": 655931
        }
      }
    ]
  }
}
```

---

## Admin Endpoints (Optional)

### Base Path: `/api/v1/admin`

**Authorization:** Requires `ADMIN` role only

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/agencies` | Get all agencies (paginated) |
| `GET` | `/agencies/{id}` | Get agency by ID |
| `PUT` | `/agencies/{id}/plan` | Update agency plan |
| `PUT` | `/agencies/{id}/status` | Update agency status |
| `GET` | `/users` | Get all users (paginated) |

**Note:** These are admin-only endpoints, not typically used by BBapp directly.

---

## Request/Response Models

### Common Headers

**All Authenticated Requests:**
```http
Authorization: Bearer {accessToken}
Content-Type: application/json
```

### Pagination (Admin Endpoints)

**Query Parameters:**
- `page` (int): Page number (0-indexed)
- `size` (int): Items per page (default: 20)
- `sort` (string): Sort field and direction (e.g., "name,asc")

**Response:**
```json
{
  "content": [...],
  "totalElements": 100,
  "totalPages": 5,
  "size": 20,
  "number": 0
}
```

---

## Error Codes

### Standard Error Response

```json
{
  "timestamp": "2025-12-27T12:00:00.000+0000",
  "status": "BAD_REQUEST",
  "errorCode": 1003,
  "message": "Validation failed",
  "details": "Additional error details",
  "subErrors": [
    {
      "object": "LoginRequest",
      "field": "username",
      "rejectedValue": "ab",
      "message": "Username must be between 3 and 50 characters"
    }
  ]
}
```

### Error Code Reference

| Code | Description | HTTP Status |
|------|-------------|-------------|
| 1001 | Entity not found | 404 |
| 1002 | Duplicate entity | 400 |
| 1003 | Validation error | 400 |
| 1004 | Invalid parameters | 400 |
| 1005 | Data already exists | 409 |
| 2001 | User not found | 404 |
| 2002 | Invalid credentials | 401 |
| 2003 | Token expired | 401 |
| 5000 | Server error | 500 |

---

## Integration Flow

### Startup Sequence

```
1. Application Launch
   ↓
2. Login/Register (POST /api/v1/auth/login or /register)
   ↓
3. Store access token & refresh token
   ↓
4. Get BBapp Config (GET /api/v1/external/config)
   ↓
5. Validate Trial (POST /api/v1/external/validate-trial)
   ↓
6. Connect WebSocket (ws://host/ws?token={accessToken})
   ↓
7. Subscribe to topics (/topic/room/{roomId}/pk)
   ↓
8. Connect to Bigo Live rooms
   ↓
9. Send heartbeat every 30s (POST /api/v1/external/heartbeat)
   ↓
10. Forward Bigo messages to /app/room/{roomId}/bigo
```

### Runtime Message Flow

```
Bigo Live → BBapp receives gift
   ↓
BBapp maps streamerId → teamId (using config)
   ↓
BBapp creates GIFT payload
   ↓
BBapp sends to /app/room/{roomId}/bigo
   ↓
BB-Core processes and broadcasts to /topic/room/{roomId}/pk
   ↓
BBapp receives PK_SYNC update
   ↓
BBapp updates UI
```

### Token Refresh Flow

```
Access token expires (24h)
   ↓
API returns 401 with errorCode 2003
   ↓
BBapp calls /api/v1/auth/refresh-token
   ↓
Store new access token & refresh token
   ↓
Retry failed request with new token
```

---

## Quick Reference

**Base URL:** `http://localhost:8080`

**WebSocket:** `ws://localhost:8080/ws`

**Documentation:**
- Full Guide: `docs/BBAPP_INTEGRATION_GUIDE.md`
- Controllers:
  - Auth: `AuthController.java`
  - External: `ExternalController.java`
  - Scripts: `ScriptController.java`

**Support:** Refer to BB-Core repository for issues

---

**End of API Reference**
