import { useState, useCallback } from 'react';
import { ProfileSelectionStep } from './ProfileSelectionStep';
import { RoomConfigStep } from './RoomConfigStep';
import { StreamerConfigStep } from './StreamerConfigStep';
import { ReviewStep } from './ReviewStep';
import { ToastNotification } from './ToastNotification';
import type { WizardStep, WizardState, Toast } from './types';
import './WizardContainer.css';

interface WizardContainerProps {
  accessToken: string;
  onSessionStart: () => void;
}

const STEPS: WizardStep[] = ['profile', 'config', 'streamers', 'review'];

export function WizardContainer({ accessToken, onSessionStart }: WizardContainerProps) {
  const [state, setState] = useState<WizardState>({
    currentStep: 'profile',
    profileId: null,
    profileName: '',
    roomId: '',
    config: null,
    isValid: false,
  });

  const [toasts, setToasts] = useState<Toast[]>([]);

  // Toast management
  const addToast = useCallback((type: Toast['type'], message: string, persistent = false) => {
    const id = `toast-${Date.now()}-${Math.random()}`;
    setToasts((prev) => [...prev, { id, type, message, persistent }]);
  }, []);

  const dismissToast = useCallback((id: string) => {
    setToasts((prev) => prev.filter((t) => t.id !== id));
  }, []);

  // Step navigation
  const currentStepIndex = STEPS.indexOf(state.currentStep);
  const canGoNext = state.isValid && currentStepIndex < STEPS.length - 1;
  const canGoBack = currentStepIndex > 0;

  const goNext = useCallback(() => {
    if (canGoNext) {
      setState((prev) => ({
        ...prev,
        currentStep: STEPS[currentStepIndex + 1],
        isValid: false, // Reset validation for next step
      }));
    }
  }, [canGoNext, currentStepIndex]);

  const goBack = useCallback(() => {
    if (canGoBack) {
      setState((prev) => ({
        ...prev,
        currentStep: STEPS[currentStepIndex - 1],
      }));
    }
  }, [canGoBack, currentStepIndex]);

  // Step-specific state updates
  const updateState = useCallback((updates: Partial<WizardState>) => {
    setState((prev) => ({ ...prev, ...updates }));
  }, []);

  return (
    <div className="wizard-container">
      <ToastNotification toasts={toasts} onDismiss={dismissToast} />

      <div className="wizard-header">
        <h2>PK Session Setup Wizard</h2>
        <div className="wizard-progress">
          {STEPS.map((step, index) => (
            <div
              key={step}
              className={`progress-step ${
                index === currentStepIndex
                  ? 'active'
                  : index < currentStepIndex
                  ? 'completed'
                  : ''
              }`}
            >
              <div className="progress-circle">{index + 1}</div>
              <div className="progress-label">{getStepLabel(step)}</div>
            </div>
          ))}
        </div>
      </div>

      <div className="wizard-content">
        {state.currentStep === 'profile' && (
          <ProfileSelectionStep
            state={state}
            updateState={updateState}
            addToast={addToast}
          />
        )}
        {state.currentStep === 'config' && (
          <RoomConfigStep
            state={state}
            updateState={updateState}
            addToast={addToast}
            accessToken={accessToken}
          />
        )}
        {state.currentStep === 'streamers' && (
          <StreamerConfigStep
            state={state}
            updateState={updateState}
            addToast={addToast}
            accessToken={accessToken}
          />
        )}
        {state.currentStep === 'review' && (
          <ReviewStep
            state={state}
            updateState={updateState}
            addToast={addToast}
            accessToken={accessToken}
            onSessionStart={onSessionStart}
          />
        )}
      </div>

      <div className="wizard-actions">
        <button
          onClick={goBack}
          disabled={!canGoBack}
          className="wizard-btn wizard-btn-secondary"
        >
          Back
        </button>
        <button
          onClick={goNext}
          disabled={!canGoNext}
          className="wizard-btn wizard-btn-primary"
        >
          Next
        </button>
      </div>
    </div>
  );
}

function getStepLabel(step: WizardStep): string {
  switch (step) {
    case 'profile':
      return 'Profile';
    case 'config':
      return 'Room Config';
    case 'streamers':
      return 'Streamers';
    case 'review':
      return 'Review';
  }
}
