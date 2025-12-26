import { useEffect, useState } from 'react';
import { ValidateTrial } from '../../../wailsjs/go/main/App';
import type { WizardState } from './types';

interface StreamerConfigStepProps {
  state: WizardState;
  updateState: (updates: Partial<WizardState>) => void;
  addToast: (type: string, message: string, persistent?: boolean) => void;
  accessToken: string;
}

export function StreamerConfigStep({
  state,
  updateState,
  addToast,
  accessToken,
}: StreamerConfigStepProps) {
  const [validating, setValidating] = useState(false);
  const [validationResult, setValidationResult] = useState<any>(null);

  useEffect(() => {
    // Validate on mount
    handleValidate();
  }, []);

  const handleValidate = async () => {
    if (!state.config?.teams) {
      addToast('error', 'No streamers to validate');
      return;
    }

    try {
      setValidating(true);

      // Collect all streamers from all teams
      const streamers = state.config.teams.flatMap((team: any) =>
        (team.streamers || []).map((s: any) => ({
          bigoId: s.bigoId,
          bigoRoomId: s.bigoRoomId,
        }))
      );

      const result = await ValidateTrial(streamers);
      setValidationResult(result);

      if (result.allowed) {
        updateState({ isValid: true });
        addToast('success', 'All streamers validated successfully');
      } else {
        updateState({ isValid: false });
        addToast(
          'error',
          `Validation failed: ${result.message}`,
          true
        );
      }
    } catch (error) {
      addToast('error', `Validation error: ${error}`, true);
      updateState({ isValid: false });
    } finally {
      setValidating(false);
    }
  };

  const totalStreamers = state.config?.teams?.reduce(
    (acc: number, team: any) => acc + (team.streamers?.length || 0),
    0
  ) || 0;

  return (
    <div className="streamer-config-step">
      <h3>Streamer Configuration & Validation</h3>
      <p>Review and validate streamers before starting the session.</p>

      <div className="streamer-summary">
        <p>
          <strong>Total Streamers:</strong> {totalStreamers}
        </p>
        <p>
          <strong>Teams:</strong> {state.config?.teams?.length || 0}
        </p>
      </div>

      {state.config?.teams?.map((team: any, index: number) => (
        <div key={team.teamId || index} className="team-group">
          <h4>{team.name || `Team ${index + 1}`}</h4>
          <table className="streamer-table">
            <thead>
              <tr>
                <th>Name</th>
                <th>Bigo ID</th>
                <th>Bigo Room ID</th>
                <th>Binding Gift</th>
              </tr>
            </thead>
            <tbody>
              {(team.streamers || []).map((streamer: any, sIndex: number) => (
                <tr key={streamer.streamerId || sIndex}>
                  <td>{streamer.name || '-'}</td>
                  <td>{streamer.bigoId}</td>
                  <td>{streamer.bigoRoomId}</td>
                  <td>{streamer.bindingGift || '-'}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      ))}

      <button
        className="wizard-btn wizard-btn-primary"
        onClick={handleValidate}
        disabled={validating}
      >
        {validating ? 'Validating...' : 'Validate Streamers'}
      </button>

      {validationResult && (
        <div
          className={`validation-result ${
            validationResult.allowed ? 'success' : 'error'
          }`}
        >
          <p>
            {validationResult.allowed
              ? '✓ Validation Successful'
              : `✗ Validation Failed: ${validationResult.message}`}
          </p>
          {validationResult.blockedBigoIds?.length > 0 && (
            <p>Blocked IDs: {validationResult.blockedBigoIds.join(', ')}</p>
          )}
        </div>
      )}
    </div>
  );
}
