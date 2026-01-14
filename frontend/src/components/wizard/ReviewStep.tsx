import { useState } from 'react';
import { CreateProfile, UpdateProfile } from '../../../wailsjs/go/main/App';
import type { WizardState, ToastType } from './types';
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Separator } from "@/components/ui/separator";
import { Save, Play, Rocket, AlertCircle, CheckCircle2, User, Trophy, Users } from "lucide-react";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { ScrollArea } from "@/components/ui/scroll-area";

interface ReviewStepProps {
  state: WizardState;
  updateState: (updates: Partial<WizardState>) => void;
  addToast: (type: ToastType, message: string, persistent?: boolean) => void;
  accessToken: string;
  onSessionStart: (roomId: string, config: any) => void;
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
      // Ensure config has the correct roomId from state
      const finalConfig = {
        ...state.config,
        roomId: state.roomId
      };

      onSessionStart(state.roomId, finalConfig);
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

  const totalIdols = state.config?.teams?.reduce(
    (acc: number, team: any) => acc + (team.streamers?.length || 0),
    0
  ) || 0;

  return (
    <div className="space-y-6">
      <div className="text-center space-y-2">
        <h3 className="text-lg font-medium">Review & Start Session</h3>
        <p className="text-sm text-muted-foreground">Review your configuration before starting the PK session.</p>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        {/* Profile Info */}
        <Card>
          <CardHeader className="py-4">
            <CardTitle className="text-sm font-medium uppercase text-muted-foreground flex items-center gap-2">
              <User className="h-4 w-4" /> Profile Info
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-2">
            <div>
              <div className="text-xs text-muted-foreground">Profile Name</div>
              <div className="font-semibold">{state.profileName}</div>
            </div>
            <div>
              <div className="text-xs text-muted-foreground">Room ID</div>
              <div className="font-mono text-sm bg-muted inline-block px-2 py-0.5 rounded">{state.roomId}</div>
            </div>
          </CardContent>
        </Card>

        {/* Team Stats */}
        <Card>
          <CardHeader className="py-4">
            <CardTitle className="text-sm font-medium uppercase text-muted-foreground flex items-center gap-2">
              <Trophy className="h-4 w-4" /> Competition Stats
            </CardTitle>
          </CardHeader>
          <CardContent className="grid grid-cols-2 gap-4">
            <div>
              <div className="text-xs text-muted-foreground">Teams</div>
              <div className="text-2xl font-bold text-primary">{state.config?.teams?.length || 0}</div>
            </div>
            <div>
              <div className="text-xs text-muted-foreground">Agency ID</div>
              <div className="font-mono">{state.config?.agencyId || 'N/A'}</div>
            </div>
          </CardContent>
        </Card>

        {/* Idol Stats */}
        <Card>
          <CardHeader className="py-4">
            <CardTitle className="text-sm font-medium uppercase text-muted-foreground flex items-center gap-2">
              <Users className="h-4 w-4" /> Participants
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div>
              <div className="text-xs text-muted-foreground">Total Idols</div>
              <div className="text-2xl font-bold text-primary">{totalIdols}</div>
            </div>
          </CardContent>
        </Card>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Teams Configuration</CardTitle>
        </CardHeader>
        <CardContent>
          <ScrollArea className="h-[250px] pr-4">
            <div className="space-y-6">
              {state.config?.teams?.map((team: any, index: number) => (
                <div key={team.teamId || index} className="space-y-3">
                  <div className="flex items-center justify-between border-b pb-2">
                    <h4 className="font-semibold flex items-center gap-2">
                      <span className="bg-primary/10 text-primary w-6 h-6 rounded-full flex items-center justify-center text-xs">
                        {index + 1}
                      </span>
                      {team.name || `Team ${index + 1}`}
                    </h4>
                    <Badge variant="secondary">{team.bindingGift}</Badge>
                  </div>
                  <ul className="space-y-2 pl-8">
                    {(team.streamers || []).map((s: any, si: number) => (
                      <div key={si} className="flex items-center gap-2 text-xs p-2 bg-muted/30 rounded">
                        <div className="flex-1">
                          <div className="font-medium">{s.name || 'Unnamed'}</div>
                          <div className="text-muted-foreground font-mono text-[10px]">{s.bigoRoomId}</div>
                        </div>
                        <Badge variant="outline" className="text-[10px]">{s.bindingGift}</Badge>
                      </div>
                    ))}
                  </ul>
                </div>
              ))}
            </div>
          </ScrollArea>
        </CardContent>
      </Card>

      <div className="flex justify-end gap-3 pt-4 border-t">
        <Button
          variant="secondary"
          className="w-40"
          onClick={handleSaveProfile}
          disabled={saving || starting}
        >
          {saving ? 'Saving...' : (
            <>
              <Save className="h-4 w-4 mr-2" />
              Save Profile
            </>
          )}
        </Button>
        <Button
          className="w-48 bg-green-600 hover:bg-green-700"
          onClick={handleSaveAndStart}
          disabled={saving || starting}
        >
          {starting ? 'Starting...' : (
            <>
              <Rocket className="h-4 w-4 mr-2" />
              Save & Start
            </>
          )}
        </Button>
      </div>
    </div>
  );
}
