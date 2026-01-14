import { useState, useCallback } from 'react';
import { ProfileSelectionStep } from './ProfileSelectionStep';
import { RoomConfigStep } from './RoomConfigStep';

import { ReviewStep } from './ReviewStep';
import { ToastNotification } from './ToastNotification';
import type { WizardStep, WizardState, Toast } from './types';
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { cn } from "@/lib/utils";
import { Check } from "lucide-react";

import { SaveBBAppConfig } from "../../../wailsjs/go/main/App";

interface WizardContainerProps {
  accessToken: string;
  onSessionStart: (roomId: string, config: any) => void;
  roomId?: string;
  config?: any;
}

const STEPS: WizardStep[] = ['profile', 'config', 'review'];

export function WizardContainer({ accessToken, onSessionStart, roomId: initialRoomId, config: initialConfig }: WizardContainerProps) {
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

  // ...

  const goNext = useCallback(async () => {
    if (canGoNext) {
      // Logic for saving config before proceeding
      if (state.currentStep === 'config') {
        try {
          // Ensure we have a valid config to save
          if (state.config) {
            await SaveBBAppConfig(state.roomId, state.config);
            // The hook is defined above.
          }
        } catch (e: any) {
          console.error("Save failed", e);
          // We access to 'addToast' here? Yes, it's in scope.
          addToast('error', `Failed to save configuration: ${e.toString()}`);
          return; // Stop navigation
        }
      }

      setState((prev) => ({
        ...prev,
        currentStep: STEPS[currentStepIndex + 1],
        isValid: false, // Reset validation for next step
      }));
    }
  }, [canGoNext, currentStepIndex, state.currentStep, state.config, state.roomId, addToast]);

  const goBack = useCallback(() => {
    if (canGoBack) {
      setState((prev) => ({
        ...prev,
        currentStep: STEPS[currentStepIndex - 1],
        isValid: false, // Don't reset validity when going back, usually
      }));
    }
  }, [canGoBack, currentStepIndex]);

  // Step-specific state updates
  const updateState = useCallback((updates: Partial<WizardState>) => {
    setState((prev) => ({ ...prev, ...updates }));
  }, []);

  return (
    <div className="max-w-5xl mx-auto p-6 space-y-8 animate-in fade-in slide-in-from-bottom-4 duration-500">

      {/* Toast Notification Container */}
      <div className="fixed bottom-4 right-4 z-50">
        <ToastNotification toasts={toasts} onDismiss={dismissToast} />
      </div>

      <div className="space-y-6">
        <h2 className="text-3xl font-bold text-center tracking-tight">PK Session Setup</h2>

        {/* Progress Stepper */}
        <div className="relative flex justify-between max-w-2xl mx-auto">
          <div className="absolute top-5 left-0 right-0 h-0.5 bg-muted -z-10" />
          {STEPS.map((step, index) => {
            const isActive = index === currentStepIndex;
            const isCompleted = index < currentStepIndex;

            return (
              <div key={step} className="flex flex-col items-center gap-2">
                <div className={cn(
                  "w-10 h-10 rounded-full flex items-center justify-center font-semibold text-sm transition-all duration-300 ring-offset-background",
                  isActive ? "bg-primary text-primary-foreground ring-2 ring-ring ring-offset-2 scale-110" :
                    isCompleted ? "bg-green-500 text-white" : "bg-muted text-muted-foreground"
                )}>
                  {isCompleted ? <Check className="h-5 w-5" /> : index + 1}
                </div>
                <span className={cn(
                  "text-xs font-medium uppercase tracking-wider transition-colors",
                  isActive ? "text-primary" : "text-muted-foreground"
                )}>
                  {getStepLabel(step)}
                </span>
              </div>
            );
          })}
        </div>
      </div>

      <Card className="min-h-[500px] glass-card shadow-lg border-primary/10">
        <CardContent className="p-8">
          {state.currentStep === 'profile' && (
            <ProfileSelectionStep
              state={state}
              updateState={updateState}
              addToast={addToast}
              goNext={goNext}
            />
          )}
          {state.currentStep === 'config' && (
            <RoomConfigStep
              state={state}
              updateState={updateState}
              addToast={addToast}
              accessToken={accessToken}
              goNext={goNext}
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
        </CardContent>
      </Card>

      <div className="flex justify-between items-center pt-4">
        <Button
          variant="outline"
          onClick={goBack}
          disabled={!canGoBack}
          className="w-32"
        >
          Back
        </Button>
        <Button
          onClick={goNext}
          disabled={!canGoNext}
          className="w-32"
        >
          Next
        </Button>
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
    case 'review':
      return 'Review';
    default:
      return '';
  }
}
