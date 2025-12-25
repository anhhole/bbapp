import React, { useState } from 'react';
import './PKModeScene.css';
import { GetBBAppConfig, InitializeBBCoreClient, GetBBCoreURL } from '../../../../wailsjs/go/main/App';
import type { PKConfig } from '../../../shared/types';

interface PKModeSceneProps {
  accessToken: string;
  onSessionChange: (active: boolean) => void;
}

export const PKModeScene: React.FC<PKModeSceneProps> = ({
  accessToken,
  onSessionChange,
}) => {
  const [roomId, setRoomId] = useState('');
  const [config, setConfig] = useState<PKConfig | null>(null);
  const [configLoaded, setConfigLoaded] = useState(false);
  const [sessionActive, setSessionActive] = useState(false);

  const generateDefaultTemplate = (roomId: string): PKConfig => ({
    roomId,
    teams: [
      {
        teamId: 'team-1',
        name: 'Team A',
        bindingGift: 'Rose',
        streamers: [
          {
            streamerId: 1,
            bigoId: '',
            bigoRoomId: '',
            name: 'Streamer 1',
            bindingGift: 'Rose',
          },
        ],
      },
      {
        teamId: 'team-2',
        name: 'Team B',
        bindingGift: 'Diamond',
        streamers: [
          {
            streamerId: 2,
            bigoId: '',
            bigoRoomId: '',
            name: 'Streamer 2',
            bindingGift: 'Diamond',
          },
        ],
      },
    ],
  });

  const handleLoadConfig = async () => {
    if (!roomId.trim()) {
      alert('Please enter a Room ID');
      return;
    }

    try {
      const bbCoreUrl = await GetBBCoreURL();
      await InitializeBBCoreClient(bbCoreUrl, accessToken);

      try {
        const fetchedConfig = await GetBBAppConfig(roomId);
        setConfig(fetchedConfig as PKConfig);
        setConfigLoaded(true);
      } catch (error: any) {
        if (error.toString().includes('404')) {
          // Room not found, load default template
          const defaultConfig = generateDefaultTemplate(roomId);
          setConfig(defaultConfig);
          setConfigLoaded(true);
          alert('Room not found. Loaded default template.');
        } else {
          throw error;
        }
      }
    } catch (error: any) {
      alert(`Failed to load config: ${error.toString()}`);
    }
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
