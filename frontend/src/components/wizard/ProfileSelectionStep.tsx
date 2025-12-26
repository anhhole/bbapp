import { useEffect, useState } from 'react';
import { ListProfiles } from '../../../wailsjs/go/main/App';
import type { WizardState } from './types';

interface ProfileSelectionStepProps {
  state: WizardState;
  updateState: (updates: Partial<WizardState>) => void;
  addToast: (type: string, message: string, persistent?: boolean) => void;
}

export function ProfileSelectionStep({
  state,
  updateState,
  addToast,
}: ProfileSelectionStepProps) {
  const [profiles, setProfiles] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [createNew, setCreateNew] = useState(false);
  const [newProfileName, setNewProfileName] = useState('');

  useEffect(() => {
    loadProfiles();
  }, []);

  const loadProfiles = async () => {
    try {
      setLoading(true);
      const result = await ListProfiles();
      setProfiles(result || []);
    } catch (error) {
      addToast('error', `Failed to load profiles: ${error}`);
    } finally {
      setLoading(false);
    }
  };

  const handleSelectProfile = (profile: any) => {
    updateState({
      profileId: profile.id,
      profileName: profile.name,
      roomId: profile.roomId,
      config: profile.config,
      isValid: true,
    });
  };

  const handleCreateNew = () => {
    if (newProfileName.trim().length < 3) {
      addToast('error', 'Profile name must be at least 3 characters');
      return;
    }
    updateState({
      profileId: null,
      profileName: newProfileName.trim(),
      roomId: '',
      config: null,
      isValid: true,
    });
  };

  if (loading) {
    return <div>Loading profiles...</div>;
  }

  return (
    <div className="profile-selection-step">
      <h3>Select or Create Profile</h3>

      {!createNew ? (
        <>
          <div className="profile-list">
            {profiles.length === 0 ? (
              <p>No saved profiles. Create a new one to get started.</p>
            ) : (
              profiles.map((profile) => (
                <div
                  key={profile.id}
                  className="profile-item"
                  onClick={() => handleSelectProfile(profile)}
                >
                  <div className="profile-name">{profile.name}</div>
                  <div className="profile-details">
                    Room: {profile.roomId} | Last used:{' '}
                    {profile.lastUsedAt
                      ? new Date(profile.lastUsedAt).toLocaleDateString()
                      : 'Never'}
                  </div>
                </div>
              ))
            )}
          </div>
          <button
            className="wizard-btn wizard-btn-primary"
            onClick={() => setCreateNew(true)}
          >
            Create New Profile
          </button>
        </>
      ) : (
        <div className="create-profile-form">
          <label>
            Profile Name:
            <input
              type="text"
              value={newProfileName}
              onChange={(e) => setNewProfileName(e.target.value)}
              placeholder="Enter profile name..."
              maxLength={50}
            />
          </label>
          <div className="form-actions">
            <button
              className="wizard-btn wizard-btn-secondary"
              onClick={() => setCreateNew(false)}
            >
              Cancel
            </button>
            <button
              className="wizard-btn wizard-btn-primary"
              onClick={handleCreateNew}
              disabled={newProfileName.trim().length < 3}
            >
              Continue
            </button>
          </div>
        </div>
      )}
    </div>
  );
}
