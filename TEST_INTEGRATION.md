# Integration Testing Guide

## Prerequisites

1. BB-Core running at `http://localhost:8080`
2. STOMP broker enabled on port 61613
3. Valid authentication token
4. Test room configured in BB-Core
5. Active Bigo Live streams (for full end-to-end testing)

## Test Scenarios

### 1. Session Lifecycle

**Test:** Start and stop PK session

1. Launch BBapp
2. Enter BB-Core URL: `http://localhost:8080`
3. Enter auth token
4. Enter room ID (must exist in BB-Core)
5. Click "Start Session"
6. Verify:
   - Session ID displayed
   - All streamer connections show CONNECTED status
   - No error messages in logs
7. Click "Stop Session"
8. Verify:
   - Session marked as inactive
   - All browsers closed
   - BB-Core received stop notification

**Expected:** Clean start and stop with no errors

### 2. Config Fetching

**Test:** Verify config is fetched from BB-Core

1. Start session
2. Check BB-Core logs for GET `/bbapp-config/{roomId}`
3. Verify BBapp creates browsers for each Bigo room in config

**Expected:** One browser per streamer in config

### 3. Gift Event Flow

**Test:** Send gift in Bigo Live, verify received at BB-Core

1. Start session
2. Send gift from test account in Bigo Live
3. Check BBapp logs for gift detection
4. Check BB-Core STOMP logs for message on `/app/room/{roomId}/bigo`
5. Verify payload contains all required fields:
   - senderId, senderName, senderAvatar, senderLevel
   - streamerId, streamerName, streamerAvatar
   - giftId, giftName, giftCount, diamonds, giftImageUrl
   - timestamp, deviceHash

**Expected:** Complete gift data forwarded to BB-Core within 1 second

### 4. Chat Message Flow

**Test:** Send chat in Bigo Live, verify received at BB-Core

1. Start session
2. Send chat message from test account
3. Check BBapp logs for chat detection
4. Check BB-Core STOMP logs for CHAT message

**Expected:** Chat message forwarded with sender details

### 5. Heartbeat

**Test:** Verify heartbeat sent every 30 seconds

1. Start session
2. Wait 35 seconds
3. Check BB-Core logs for POST `/bbapp-status/{roomId}`
4. Verify heartbeat contains connection status for all streamers

**Expected:** Heartbeat every 30s with accurate status

### 6. STOMP Reconnection

**Test:** Verify auto-reconnection when STOMP disconnects

1. Start session
2. Stop STOMP broker
3. Wait 15 seconds
4. Check BBapp logs for reconnection attempts
5. Restart STOMP broker
6. Verify BBapp reconnects automatically

**Expected:** Automatic reconnection within 30 seconds

### 7. Browser Reconnection

**Test:** Verify browser recreated if connection stale

1. Start session
2. Kill Chrome process manually
3. Wait 35 seconds
4. Check BBapp logs for reconnection

**Expected:** Browser relaunched automatically

### 8. Device Fingerprinting

**Test:** Verify device hash is stable and sent

1. Start session
2. Check device hash in logs
3. Restart BBapp
4. Start session again
5. Verify device hash is identical

**Expected:** Same hash across restarts

### 9. Trial Validation

**Test:** Verify trial validation happens at config fetch

1. Use device hash that exceeded trial limit (configured in BB-Core)
2. Attempt to start session
3. Verify session start fails with trial error

**Expected:** Clear error message about trial limit

## Manual Test Checklist

- [ ] Session starts successfully
- [ ] Config fetched from BB-Core
- [ ] All browsers created
- [ ] Gift events forwarded with complete data
- [ ] Chat messages forwarded
- [ ] Heartbeat sent every 30s
- [ ] STOMP auto-reconnects
- [ ] Browsers auto-reconnect
- [ ] Session stops cleanly
- [ ] Device hash stable
- [ ] Trial validation works

## Performance Benchmarks

- Session start time: < 5 seconds
- Gift event latency: < 1 second
- Chat event latency: < 1 second
- Heartbeat interval: 30 seconds ± 2 seconds
- STOMP reconnection time: < 30 seconds
- Browser reconnection time: < 60 seconds
