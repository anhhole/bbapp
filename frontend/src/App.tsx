import { useState } from 'react';
import './App.css';
import { ConnectToCore, AddStreamer, RemoveStreamer } from '../wailsjs/go/main/App';

function App() {
  const [coreUrl, setCoreUrl] = useState('localhost:61613');
  const [connected, setConnected] = useState(false);

  const [bigoRoomId, setBigoRoomId] = useState('');
  const [teamId, setTeamId] = useState('');
  const [roomId, setRoomId] = useState('');

  const handleConnect = async () => {
    try {
      await ConnectToCore(coreUrl, '', '');
      setConnected(true);
      alert('Connected to BB-Core!');
    } catch (error) {
      alert(`Failed: ${error}`);
    }
  };

  const handleAddStreamer = async () => {
    if (!bigoRoomId || !teamId || !roomId) {
      alert('Fill all fields');
      return;
    }

    try {
      await AddStreamer(bigoRoomId, teamId, roomId);
      alert('Streamer added!');
      setBigoRoomId('');
    } catch (error) {
      alert(`Failed: ${error}`);
    }
  };

  return (
    <div className="container">
      <h1>BBapp - Bigo Stream Manager</h1>

      <div className="card">
        <h2>BB-Core Connection</h2>
        <input
          type="text"
          placeholder="STOMP URL"
          value={coreUrl}
          onChange={(e) => setCoreUrl(e.target.value)}
        />
        <button onClick={handleConnect} disabled={connected}>
          {connected ? '✓ Connected' : 'Connect'}
        </button>
      </div>

      {connected && (
        <div className="card">
          <h2>Add Streamer</h2>
          <input
            placeholder="Bigo Room ID"
            value={bigoRoomId}
            onChange={(e) => setBigoRoomId(e.target.value)}
          />
          <input
            placeholder="Team ID (UUID)"
            value={teamId}
            onChange={(e) => setTeamId(e.target.value)}
          />
          <input
            placeholder="Room ID"
            value={roomId}
            onChange={(e) => setRoomId(e.target.value)}
          />
          <button onClick={handleAddStreamer}>Add</button>
        </div>
      )}
    </div>
  );
}

export default App;
