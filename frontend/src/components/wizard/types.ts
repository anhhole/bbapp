// Wizard step types
export type WizardStep = 'profile' | 'config' | 'review';

// Wizard state
export interface WizardState {
  currentStep: WizardStep;
  profileId: string | null;
  profileName: string;
  roomId: string;
  config: any | null; // Will use generated types from Wails
  isValid: boolean;
}

// Toast notification types
export type ToastType = 'success' | 'error' | 'warning' | 'info';

export interface Toast {
  id: string;
  type: ToastType;
  message: string;
  persistent?: boolean; // If true, requires manual dismiss
}
