import { useState } from 'react';
import { CreateProfile, UpdateProfile } from '../../../wailsjs/go/main/App';
import type { WizardState } from './types';

interface ReviewStepProps {
  state: WizardState;
  updateState: (updates: Partial<WizardState>) => void;
  addToast: (type: string, message: string, persistent?: boolean) => void;
  accessToken: string;
  onSessionStart: () => void;
}

export function ReviewStep({
  state,
  updateState,
  addToast,
  accessToken,
  onSessionStart,
}: ReviewStepProps) {
  const [saving, setSaving] = useState(false);
  const [starting, setStarting] = useState(false);

  const handleSaveProfile = async () => {
    try {
      setSaving(true);

      if (state.profileId) {
        // Update existing profile
        await UpdateProfile(state.profileId, state.config);
        addToast('success', 'Profile updated successfully');
      } else {
        // Create new profile
        await CreateProfile(state.profileName, state.roomId, state.config);
        addToast('success', 'Profile created successfully');
      }
    } catch (error) {
      addToast('error', `Failed to save profile: ${error}`);
    } finally {
      setSaving(false);
    }
  };

  const handleStartSession = async () => {
    try {
      setStarting(true);
      // TODO: Call StartPKSession with state.roomId, state.config
      // For now, just trigger the callback
      onSessionStart();
      addToast('success', 'Session started successfully');
    } catch (error) {
      addToast('error', `Failed to start session: ${error}`, true);
    } finally {
      setStarting(false);
    }
  };

  const handleSaveAndStart = async () => {
    await handleSaveProfile();
    if (!saving) {
      await handleStartSession();
    }
  };

  const totalStreamers = state.config?.teams?.reduce(
    (acc: number, team: any) => acc + (team.streamers?.length || 0),
    0
  ) || 0;

  return (
    <div className="review-step">
      <h3>Review & Start Session</h3>
      <p>Review your configuration before starting the PK session.</p>

      <div className="review-summary">
        <div className="summary-item">
          <strong>Profile Name:</strong> {state.profileName}
        </div>
        <div className="summary-item">
          <strong>Room ID:</strong> {state.roomId}
        </div>
        <div className="summary-item">
          <strong>Agency ID:</strong> {state.config?.agencyId || 'N/A'}
        </div>
        <div className="summary-item">
          <strong>Total Teams:</strong> {state.config?.teams?.length || 0}
        </div>
        <div className="summary-item">
          <strong>Total Streamers:</strong> {totalStreamers}
        </div>
      </div>

      <div className="review-details">
        <h4>Teams & Streamers:</h4>
        {state.config?.teams?.map((team: any, index: number) => (
          <div key={team.teamId || index} className="team-review">
            <strong>{team.name || `Team ${index + 1}`}</strong>
            <ul>
              {(team.streamers || []).map((s: any, si: number) => (
                <li key={s.streamerId || si}>
                  {s.name || s.bigoId} ({s.bigoRoomId})
                </li>
              ))}
            </ul>
          </div>
        ))}
      </div>

      <div className="review-actions">
        <button
          className="wizard-btn wizard-btn-secondary"
          onClick={handleSaveProfile}
          disabled={saving || starting}
        >
          {saving ? 'Saving...' : 'Save Profile Only'}
        </button>
        <button
          className="wizard-btn wizard-btn-primary"
          onClick={handleSaveAndStart}
          disabled={saving || starting}
        >
          {starting ? 'Starting...' : 'Save & Start Session'}
        </button>
      </div>
    </div>
  );
}
