import React, { useState } from 'react';
import './PKModeScene.css';

interface PKModeSceneProps {
  accessToken: string;
  onSessionChange: (active: boolean) => void;
}

export const PKModeScene: React.FC<PKModeSceneProps> = ({
  accessToken,
  onSessionChange,
}) => {
  const [roomId, setRoomId] = useState('');
  const [config, setConfig] = useState(null);
  const [configLoaded, setConfigLoaded] = useState(false);
  const [sessionActive, setSessionActive] = useState(false);

  const handleLoadConfig = async () => {
    // TODO: Implement config loading
    alert('Config loading not yet implemented');
  };

  return (
    <div className="pk-mode-scene">
      <div className="card">
        <h2>Load Room Configuration</h2>
        <input
          type="text"
          placeholder="Room ID"
          value={roomId}
          onChange={(e) => setRoomId(e.target.value)}
          disabled={configLoaded}
        />
        <button onClick={handleLoadConfig} disabled={configLoaded}>
          Load Configuration
        </button>
      </div>

      {configLoaded && (
        <div className="card">
          <h2>Configuration</h2>
          <p>Team editor will be added here</p>
        </div>
      )}

      {configLoaded && (
        <div className="card">
          <h2>Session Controls</h2>
          <button disabled={sessionActive}>
            {sessionActive ? 'Session Active' : 'Start Session'}
          </button>
        </div>
      )}
    </div>
  );
};
