import { useEffect, useState } from 'react';
import { ValidateTrial, StartPKSession, GetBBCoreURL } from '../../../wailsjs/go/main/App';
import { api } from '../../../wailsjs/go/models';
import type { WizardState, ToastType } from './types';
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Alert, AlertTitle, AlertDescription } from "@/components/ui/alert";
import { Loader2, CheckCircle, AlertTriangle, AlertCircle, ShieldCheck, Play } from "lucide-react";

interface StreamerConfigStepProps {
  state: WizardState;
  updateState: (updates: Partial<WizardState>) => void;
  addToast: (type: ToastType, message: string, persistent?: boolean) => void;
  accessToken: string;
  onSessionStart: (roomId: string, config: any) => void;
}

export function StreamerConfigStep({
  state,
  updateState,
  addToast,
  accessToken,
  onSessionStart,
}: StreamerConfigStepProps) {
  const [validating, setValidating] = useState(false);
  const [validationResult, setValidationResult] = useState<any>(null);
  const [validated, setValidated] = useState(false);
  const [starting, setStarting] = useState(false);

  useEffect(() => {
    // Auto-validate on mount
    if (!validated) {
      handleValidate();
    }
  }, []);

  const handleValidate = async () => {
    if (!state.config?.teams) {
      addToast('error', 'No streamers to validate');
      return;
    }

    try {
      setValidating(true);

      // Collect all streamers from all teams, filtering out those without bigoRoomId
      const allStreamers = state.config.teams.flatMap((team: any) =>
        (team.streamers || [])
      );

      // Check for missing bigoRoomId
      const missingRoomIds = allStreamers.filter((s: any) => !s.bigoRoomId || s.bigoRoomId.trim() === '');
      if (missingRoomIds.length > 0) {
        addToast('error', `${missingRoomIds.length} streamer(s) are missing Bigo Room ID. Please fill in all Room IDs before validating.`, true);
        updateState({ isValid: false });
        setValidating(false);
        return;
      }

      // Prepare streamers for validation - only include those with bigoRoomId
      const streamers = allStreamers
        .filter((s: any) => s.bigoRoomId && s.bigoRoomId.trim() !== '')
        .map((s: any) => ({
          bigoId: s.bigoId || s.bigoRoomId, // Use bigoRoomId as fallback for bigoId
          bigoRoomId: s.bigoRoomId,
        }));

      console.log('Validating streamers:', streamers);

      if (streamers.length === 0) {
        addToast('error', 'No valid streamers to validate. Please add Bigo Room IDs.', true);
        updateState({ isValid: false });
        setValidating(false);
        return;
      }

      const result = await ValidateTrial(streamers);
      console.log('Validation result:', result);

      setValidationResult(result);
      setValidated(true);

      if (result.allowed) {
        updateState({ isValid: true });
        addToast('success', 'All streamers validated successfully');
        // Auto-start session after successful validation
        setTimeout(() => handleStartSession(), 1000);
      } else {
        updateState({ isValid: false });
        addToast(
          'error',
          `Validation failed: ${result.message || 'Unknown error'}`,
          true
        );
      }
    } catch (error: any) {
      console.error('Validation error:', error);
      const errorMessage = error?.message || error?.toString() || 'Unknown validation error';
      addToast('error', `Validation error: ${errorMessage}`, true);
      updateState({ isValid: false });
      setValidated(true);
      setValidationResult({
        allowed: false,
        message: errorMessage,
        blockedBigoIds: []
      });
    } finally {
      setValidating(false);
    }
  };

  const handleStartSession = async () => {
    if (!state.config || !state.roomId) {
      addToast('error', 'Missing configuration or room ID');
      return;
    }

    try {
      setStarting(true);
      addToast('info', 'Starting PK session...');

      const bbCoreUrl = await GetBBCoreURL();

      // Convert config to API format
      const apiConfig = new api.Config({
        roomId: state.roomId,
        agencyId: state.config.agencyId || 0,
        session: new api.SessionInfo({
          sessionId: '',
          status: 'pending',
          startedAt: 0,
          endsAt: 0,
          roomId: state.roomId,
        }),
        teams: state.config.teams.map((team: any) => new api.Team({
          teamId: team.teamId,
          name: team.name,
          bindingGift: team.bindingGift,
          scoreMultipliers: team.scoreMultipliers || {},
          streamers: team.streamers || [],
        })),
      });

      // Default duration: 60 minutes
      await StartPKSession(bbCoreUrl, accessToken, state.roomId, apiConfig, 60);

      addToast('success', 'PK Session started successfully!');
      // Trigger session start callback to hide wizard and show session interface
      onSessionStart(state.roomId, state.config);
    } catch (error: any) {
      console.error('Failed to start session:', error);
      addToast('error', `Failed to start session: ${error.toString()}`, true);
    } finally {
      setStarting(false);
    }
  };

  const totalStreamers = state.config?.teams?.reduce(
    (acc: number, team: any) => acc + (team.streamers?.length || 0),
    0
  ) || 0;

  // Check for missing Room IDs
  const missingRoomIds = state.config?.teams?.flatMap((team: any) =>
    (team.streamers || []).filter((s: any) => !s.bigoRoomId || s.bigoRoomId.trim() === '')
  ) || [];

  return (
    <div className="space-y-6">
      <div className="text-center space-y-2">
        <h3 className="text-lg font-medium">Streamer Validation</h3>
        <p className="text-sm text-muted-foreground">Review your streamers and validate trial eligibility before proceeding.</p>
      </div>

      {/* Warning for missing Room IDs */}
      {missingRoomIds.length > 0 && (
        <Alert variant="destructive">
          <AlertCircle className="h-4 w-4" />
          <AlertTitle>Missing Bigo Room IDs</AlertTitle>
          <AlertDescription>
            {missingRoomIds.length} streamer{missingRoomIds.length !== 1 ? 's are' : ' is'} missing Room ID{missingRoomIds.length !== 1 ? 's' : ''}.
            Please go back to the previous step and fill in all Bigo Room IDs before validating.
          </AlertDescription>
        </Alert>
      )}

      {/* Summary Stats */}
      <div className="grid grid-cols-2 gap-4">
        <Card>
          <CardContent className="p-4 flex flex-col items-center justify-center">
            <span className="text-2xl font-bold">{state.config?.teams?.length || 0}</span>
            <span className="text-xs text-muted-foreground uppercase font-medium">Total Teams</span>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="p-4 flex flex-col items-center justify-center">
            <span className="text-2xl font-bold">{totalStreamers}</span>
            <span className="text-xs text-muted-foreground uppercase font-medium">Total Streamers</span>
          </CardContent>
        </Card>
      </div>

      {/* Teams and Streamers List */}
      <div className="space-y-4">
        {state.config?.teams?.map((team: any, teamIndex: number) => (
          <Card key={team.teamId || teamIndex} className="overflow-hidden">
            <CardHeader className="bg-muted/40 py-3 px-4">
              <div className="flex justify-between items-center">
                <div>
                  <h4 className="font-semibold text-sm">{team.name || `Team ${teamIndex + 1}`}</h4>
                  <div className="text-xs text-muted-foreground flex items-center gap-2">
                    <span>Binding: <span className="font-medium text-foreground">{team.bindingGift}</span></span>
                    <span>•</span>
                    <span>{team.streamers?.length || 0} streamers</span>
                  </div>
                </div>
              </div>
            </CardHeader>
            <div className="divide-y">
              <div className="grid grid-cols-[2fr_2fr_1fr] gap-2 px-4 py-2 bg-muted/20 text-xs font-medium text-muted-foreground uppercase">
                <div>Name</div>
                <div>Bigo Room ID</div>
                <div>Gift</div>
              </div>
              {(team.streamers || []).map((streamer: any, sIndex: number) => (
                <div key={streamer.streamerId || sIndex} className="grid grid-cols-[2fr_2fr_1fr] gap-2 px-4 py-3 text-sm items-center hover:bg-muted/10 transition-colors">
                  <div className="font-medium">{streamer.name || '-'}</div>
                  <div className={`font-mono text-xs ${!streamer.bigoRoomId ? 'text-destructive font-bold' : ''}`}>
                    {streamer.bigoRoomId || 'MISSING'}
                  </div>
                  <div className="text-muted-foreground text-xs">{streamer.bindingGift || '-'}</div>
                </div>
              ))}
            </div>
          </Card>
        ))}
      </div>

      {/* Validation & Action Section */}
      <Card className="border-2 border-muted">
        <CardContent className="p-6 space-y-4">
          <h4 className="font-semibold flex items-center gap-2">
            <ShieldCheck className="h-5 w-5 text-primary" />
            Trial Validation
          </h4>

          <Button
            onClick={handleValidate}
            disabled={validating || starting}
            className="w-full"
            size="lg"
          >
            {validating ? (
              <>
                <Loader2 className="h-4 w-4 animate-spin mr-2" />
                Validating Streamers...
              </>
            ) : (
              'Validate Streamers'
            )}
          </Button>

          {starting && (
            <Alert className="bg-primary/10 border-primary/20">
              <Loader2 className="h-4 w-4 animate-spin text-primary" />
              <AlertTitle>Starting PK Session...</AlertTitle>
              <AlertDescription>Please wait while we initialize the session and connect to streamers.</AlertDescription>
            </Alert>
          )}

          {validationResult && !starting && (
            <Alert variant={validationResult.allowed ? "default" : "destructive"} className={validationResult.allowed ? "border-green-500/50 bg-green-500/10 text-green-700 dark:text-green-400" : ""}>
              {validationResult.allowed ? <CheckCircle className="h-4 w-4 text-green-600" /> : <AlertTriangle className="h-4 w-4" />}
              <AlertTitle>{validationResult.allowed ? "Validation Successful" : "Validation Failed"}</AlertTitle>
              <AlertDescription>
                {validationResult.allowed ? (
                  "All streamers are eligible. Session starting automatically..."
                ) : (
                  <div className="space-y-2">
                    <p>{validationResult.message}</p>
                    {validationResult.blockedBigoIds?.length > 0 && (
                      <div className="p-2 bg-background/50 rounded text-xs font-mono border">
                        Blocked: {validationResult.blockedBigoIds.join(', ')}
                      </div>
                    )}
                  </div>
                )}
              </AlertDescription>
            </Alert>
          )}

        </CardContent>
      </Card>
    </div>
  );
}
