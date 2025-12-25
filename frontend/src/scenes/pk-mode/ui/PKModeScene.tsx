import React, { useState } from 'react';
import './PKModeScene.css';
import { GetBBAppConfig, InitializeBBCoreClient, GetBBCoreURL, SaveBBAppConfig, StartPKSession, StopPKSession, GetOverlayURL } from '../../../../wailsjs/go/main/App';
import type { PKConfig, Team } from '../../../shared/types';
import { Plus } from 'lucide-react';
import { TeamCard } from './components/TeamCard';

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
  const [configSaved, setConfigSaved] = useState(false);
  const [saving, setSaving] = useState(false);
  const [sessionActive, setSessionActive] = useState(false);
  const [startingSession, setStartingSession] = useState(false);
  const [stoppingSession, setStoppingSession] = useState(false);
  const [overlayUrl, setOverlayUrl] = useState('');

  const generateDefaultTemplate = (roomId: string): PKConfig => ({
    roomId,
    teams: [
      {
        teamId: 'team-1',
        name: 'Team A',
        bindingGift: 'Rose',
        streamers: [
          {
            streamerId: 'streamer-1',
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
            streamerId: 'streamer-2',
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

  const validateConfig = (): string | null => {
    if (!config) return 'No configuration loaded';
    if (!config.roomId.trim()) return 'Room ID is required';
    if (config.teams.length === 0) return 'At least one team is required';

    for (const team of config.teams) {
      if (!team.name.trim()) return 'All teams must have a name';
      if (!team.bindingGift.trim()) return 'All teams must have a binding gift';
      if (team.streamers.length === 0) return 'Each team must have at least one streamer';

      for (const streamer of team.streamers) {
        if (!streamer.name.trim()) return 'All streamers must have a name';
        if (!streamer.bigoRoomId.trim()) return 'All streamers must have a Bigo Room ID';
        if (!streamer.bindingGift.trim()) return 'All streamers must have a binding gift';
      }
    }

    return null; // Valid
  };

  const handleSaveConfig = async () => {
    if (!config) {
      alert('No configuration to save');
      return;
    }

    // Validate configuration
    const validationError = validateConfig();
    if (validationError) {
      alert(`Validation failed: ${validationError}`);
      return;
    }

    setSaving(true);
    try {
      // Send the config directly - the backend expects the full config object
      await SaveBBAppConfig(config.roomId, config as any);
      setConfigSaved(true);
      alert('Configuration saved successfully!');
    } catch (error: any) {
      alert(`Failed to save configuration: ${error.toString()}`);
    } finally {
      setSaving(false);
    }
  };

  const handleStartSession = async () => {
    if (!config || !configSaved) {
      alert('Please save the configuration before starting a session');
      return;
    }

    setStartingSession(true);
    try {
      const bbCoreUrl = await GetBBCoreURL();
      await StartPKSession(bbCoreUrl, accessToken, config.roomId);

      setSessionActive(true);
      onSessionChange(true);

      // Generate overlay URL
      const url = await GetOverlayURL('pk-battle', config.roomId, accessToken);
      setOverlayUrl(url);

      alert('PK Session started successfully!');
    } catch (error: any) {
      alert(`Failed to start session: ${error.toString()}`);
    } finally {
      setStartingSession(false);
    }
  };

  const handleStopSession = async () => {
    const confirmed = window.confirm('Are you sure you want to stop the session?');
    if (!confirmed) return;

    setStoppingSession(true);
    try {
      await StopPKSession('USER_STOPPED');

      setSessionActive(false);
      onSessionChange(false);
      setOverlayUrl(''); // Clear overlay URL

      alert('PK Session stopped successfully!');
    } catch (error: any) {
      alert(`Failed to stop session: ${error.toString()}`);
    } finally {
      setStoppingSession(false);
    }
  };

  const handleAddTeam = () => {
    if (!config) return;

    const newTeamId = `team-${Date.now()}`;
    const newTeam: Team = {
      teamId: newTeamId,
      name: `Team ${config.teams.length + 1}`,
      bindingGift: 'Rose',
      streamers: [
        {
          streamerId: `streamer-${Date.now()}`,
          bigoId: '',
          bigoRoomId: '',
          name: 'Streamer 1',
          bindingGift: 'Rose',
        },
      ],
    };

    setConfig({
      ...config,
      teams: [...config.teams, newTeam],
    });
    setConfigSaved(false);
  };

  const handleUpdateTeam = (teamIndex: number, updatedTeam: Team) => {
    if (!config) return;

    const updatedTeams = [...config.teams];
    updatedTeams[teamIndex] = updatedTeam;
    setConfig({
      ...config,
      teams: updatedTeams,
    });
    setConfigSaved(false);
  };

  const handleRemoveTeam = (teamIndex: number) => {
    if (!config) return;

    if (config.teams.length <= 1) {
      alert('Cannot remove the last team. At least one team is required.');
      return;
    }

    const updatedTeams = config.teams.filter((_, index) => index !== teamIndex);
    setConfig({
      ...config,
      teams: updatedTeams,
    });
    setConfigSaved(false);
  };

  const handleCopyUrl = () => {
    navigator.clipboard.writeText(overlayUrl).then(() => {
      alert('Overlay URL copied to clipboard!');
    }).catch((err) => {
      alert('Failed to copy URL: ' + err.toString());
    });
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
          disabled={configLoaded || sessionActive}
        />
        <button onClick={handleLoadConfig} disabled={configLoaded || sessionActive}>
          Load Configuration
        </button>
      </div>

      {configLoaded && config && (
        <div className="card">
          <div className="teams-header">
            <h2>Teams Configuration</h2>
            <button className="add-team-btn" onClick={handleAddTeam}>
              <Plus size={16} /> Add Team
            </button>
          </div>

          <div className="teams-list">
            {config.teams.map((team, index) => (
              <TeamCard
                key={team.teamId}
                team={team}
                onUpdateTeam={(updatedTeam) => handleUpdateTeam(index, updatedTeam)}
                onRemoveTeam={() => handleRemoveTeam(index)}
                canRemove={config.teams.length > 1}
              />
            ))}
          </div>
        </div>
      )}

      {sessionActive && overlayUrl && (
        <div className="card overlay-url-card">
          <h2>Overlay URL</h2>
          <p className="overlay-instructions">
            Copy this URL and add it as a Browser Source in OBS:
          </p>
          <div className="overlay-url-container">
            <input
              type="text"
              className="overlay-url-input"
              value={overlayUrl}
              readOnly
            />
            <button className="copy-url-btn" onClick={handleCopyUrl}>
              Copy URL
            </button>
          </div>
          <p className="overlay-note">
            <strong>Note:</strong> The overlay will connect to BB-Core and display real-time PK battle data.
          </p>
        </div>
      )}

      {configLoaded && config && (
        <div className="card">
          <div className="action-buttons">
            <button
              className="save-config-btn"
              onClick={handleSaveConfig}
              disabled={saving || configSaved || sessionActive}
            >
              {saving ? 'Saving...' : configSaved ? 'Configuration Saved ✓' : 'Save Configuration'}
            </button>

            {!sessionActive ? (
              <button
                className="start-session-btn"
                onClick={handleStartSession}
                disabled={!configSaved || startingSession}
              >
                {startingSession ? 'Starting Session...' : 'Start Session'}
              </button>
            ) : (
              <button
                className="stop-session-btn"
                onClick={handleStopSession}
                disabled={stoppingSession}
              >
                {stoppingSession ? 'Stopping Session...' : 'Stop Session'}
              </button>
            )}
          </div>
        </div>
      )}
    </div>
  );
};
