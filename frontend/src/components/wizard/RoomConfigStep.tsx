import { useState } from 'react';
import { FetchConfig } from '../../../wailsjs/go/main/App';
import type { WizardState, ToastType } from './types';

interface RoomConfigStepProps {
  state: WizardState;
  updateState: (updates: Partial<WizardState>) => void;
  addToast: (type: ToastType, message: string, persistent?: boolean) => void;
  accessToken: string;
}

export function RoomConfigStep({
  state,
  updateState,
  addToast,
  accessToken,
}: RoomConfigStepProps) {
  const [roomId, setRoomId] = useState(state.roomId);
  const [loading, setLoading] = useState(false);

  const handleFetchConfig = async () => {
    if (!roomId.trim()) {
      addToast('error', 'Please enter a Room ID');
      return;
    }

    try {
      setLoading(true);
      const config = await FetchConfig(roomId.trim());
      updateState({
        roomId: roomId.trim(),
        config,
        isValid: true,
      });
      addToast('success', 'Configuration fetched successfully');
    } catch (error) {
      addToast('error', `Failed to fetch config: ${error}`, true);
      updateState({ isValid: false });
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="room-config-step">
      <h3>Room Configuration</h3>
      <p>Enter your BB-Core Room ID to fetch the configuration.</p>

      <div className="form-group">
        <label>
          Room ID:
          <input
            type="text"
            value={roomId}
            onChange={(e) => setRoomId(e.target.value)}
            placeholder="Enter Room ID..."
            disabled={loading}
          />
        </label>
        <button
          className="wizard-btn wizard-btn-primary"
          onClick={handleFetchConfig}
          disabled={loading || !roomId.trim()}
        >
          {loading ? 'Fetching...' : 'Fetch Configuration'}
        </button>
      </div>

      {state.config && (
        <div className="config-preview">
          <h4>Configuration Loaded</h4>
          <p>
            Agency ID: {state.config.agencyId} | Teams: {state.config.teams?.length || 0}
          </p>
        </div>
      )}
    </div>
  );
}
