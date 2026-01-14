import { useState } from 'react';
import { WizardContainer } from '../../../components/wizard/WizardContainer';
import { SessionControlPanel } from './components/SessionControlPanel';
import { ToastNotification } from '../../../components/wizard/ToastNotification';
import { ToastType } from '../../../components/wizard/types';
import './PKModeScene.css';

export interface PKModeSceneProps {
  accessToken: string;
  onSessionChange?: (isActive: boolean) => void;
}

export function PKModeScene({ accessToken, onSessionChange }: PKModeSceneProps) {
  const [toasts, setToasts] = useState<{ id: string; type: ToastType; message: string }[]>([]);
  const [showSessionControl, setShowSessionControl] = useState(false);
  const [sessionConfig, setSessionConfig] = useState<any>(null);
  const [sessionRoomId, setSessionRoomId] = useState<string>('');
  const [sessionDuration, setSessionDuration] = useState<number>(60);

  // Toast helper
  const addToast = (type: ToastType, message: string, persistent = false) => {
    const id = Math.random().toString(36).substr(2, 9);
    setToasts(prev => [...prev, { id, type, message }]);

    if (!persistent) {
      setTimeout(() => {
        setToasts(prev => prev.filter(t => t.id !== id));
      }, 3000);
    }
  };

  const removeToast = (id: string) => {
    setToasts(prev => prev.filter(t => t.id !== id));
  };

  const handleSessionStart = (roomId: string, config: any) => {
    console.log('Session setup completed for room:', roomId);
    setSessionConfig(config);
    setSessionRoomId(roomId);
    setSessionDuration(60); // Default 60 minutes
    setShowSessionControl(true);
    if (onSessionChange) onSessionChange(true);
  };

  const handleBackToSetup = () => {
    setShowSessionControl(false);
    setSessionConfig(null);
    setSessionRoomId('');
    if (onSessionChange) onSessionChange(false);
  };

  return (
    <div className="pk-mode-scene h-full flex flex-col">
      <div className="flex-1 relative overflow-auto">
        {!showSessionControl ? (
          <WizardContainer
            onSessionStart={handleSessionStart}
            accessToken={accessToken}
          />
        ) : (
          <SessionControlPanel
            config={sessionConfig}
            roomId={sessionRoomId}
            durationMinutes={sessionDuration}
            onBack={handleBackToSetup}
            onSessionActiveChange={onSessionChange}
          />
        )}
      </div>

      <div className="fixed bottom-4 right-4 z-50">
        <ToastNotification
          toasts={toasts}
          onDismiss={removeToast}
        />
      </div>
    </div>
  );
}
