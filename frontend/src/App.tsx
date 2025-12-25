import { useState, useEffect } from 'react';
import './App.css';
import { StartPKSession, StopPKSession, GetSessionStatus } from '../wailsjs/go/main/App';

interface ConnectionStatus {
  bigoRoomId: string;
  streamerId: string;
  status: string;
  messagesReceived: number;
}

interface SessionStatus {
  roomId: string;
  sessionId: string;
  isActive: boolean;
  connections: ConnectionStatus[];
}

function App() {
  const [bbCoreUrl, setBbCoreUrl] = useState('http://localhost:8080');
  const [authToken, setAuthToken] = useState('');
  const [roomId, setRoomId] = useState('');
  const [sessionActive, setSessionActive] = useState(false);
  const [sessionStatus, setSessionStatus] = useState<SessionStatus | null>(null);

  // Poll session status
  useEffect(() => {
    if (!sessionActive) return;

    const interval = setInterval(async () => {
      try {
        const status = await GetSessionStatus();
        setSessionStatus(status);
      } catch (error) {
        console.error('Failed to get session status:', error);
      }
    }, 5000);

    return () => clearInterval(interval);
  }, [sessionActive]);

  const handleStartSession = async () => {
    if (!roomId || !authToken) {
      alert('Please fill in all fields');
      return;
    }

    try {
      await StartPKSession(bbCoreUrl, authToken, roomId);
      setSessionActive(true);
      alert('PK Session started successfully!');
    } catch (error) {
      alert(`Failed to start session: ${error}`);
    }
  };

  const handleStopSession = async () => {
    try {
      await StopPKSession('USER_STOPPED');
      setSessionActive(false);
      setSessionStatus(null);
      alert('PK Session stopped');
    } catch (error) {
      alert(`Failed to stop session: ${error}`);
    }
  };

  return (
    <div className="container">
      <h1>BBapp - PK Session Manager</h1>

      {!sessionActive ? (
        <div className="card">
          <h2>Start PK Session</h2>
          <input
            type="text"
            placeholder="BB-Core URL (e.g., http://localhost:8080)"
            value={bbCoreUrl}
            onChange={(e) => setBbCoreUrl(e.target.value)}
          />
          <input
            type="password"
            placeholder="Authentication Token (required)"
            value={authToken}
            onChange={(e) => setAuthToken(e.target.value)}
          />
          <input
            type="text"
            placeholder="Room ID"
            value={roomId}
            onChange={(e) => setRoomId(e.target.value)}
          />
          <button onClick={handleStartSession}>Start Session</button>
        </div>
      ) : (
        <>
          <div className="card">
            <h2>Session Active</h2>
            <p><strong>Room ID:</strong> {sessionStatus?.roomId || roomId}</p>
            <p><strong>Session ID:</strong> {sessionStatus?.sessionId || 'Loading...'}</p>
            <button onClick={handleStopSession} className="stop-button">
              Stop Session
            </button>
          </div>

          {sessionStatus && sessionStatus.connections && sessionStatus.connections.length > 0 && (
            <div className="card">
              <h2>Active Connections</h2>
              <div className="connections">
                {sessionStatus.connections.map((conn) => (
                  <div key={conn.bigoRoomId} className="connection-item">
                    <div className="connection-header">
                      <span className="streamer-id">{conn.streamerId}</span>
                      <span className={`status status-${conn.status.toLowerCase()}`}>
                        {conn.status}
                      </span>
                    </div>
                    <div className="connection-details">
                      <span>Bigo Room: {conn.bigoRoomId}</span>
                      <span>Messages: {conn.messagesReceived}</span>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          )}
        </>
      )}
    </div>
  );
}

export default App;
