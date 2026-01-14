import { useEffect, useState } from 'react';
import { ListProfiles, DeleteProfile } from '../../../wailsjs/go/main/App';
import type { WizardState, ToastType } from './types';
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from "@/components/ui/card";
import { Plus, User, Clock, Loader2, Trash2 } from "lucide-react";

interface ProfileSelectionStepProps {
  state: WizardState;
  updateState: (updates: Partial<WizardState>) => void;
  addToast: (type: ToastType, message: string, persistent?: boolean) => void;
  goNext: () => void;
}

export function ProfileSelectionStep({
  state,
  updateState,
  addToast,
  goNext,
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
    addToast('success', `Profile "${profile.name}" loaded`);
    // Auto-advance to next step
    setTimeout(() => goNext(), 300);
  };

  const handleDeleteProfile = async (e: React.MouseEvent, profileId: string, profileName: string) => {
    e.stopPropagation(); // Prevent card click
    if (window.confirm(`Are you sure you want to delete profile "${profileName}"?`)) {
      try {
        await DeleteProfile(profileId);
        addToast('success', `Profile "${profileName}" deleted`);
        loadProfiles(); // Refresh list
      } catch (error: any) {
        addToast('error', `Failed to delete profile: ${error.toString()}`);
      }
    }
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
    addToast('success', 'Profile created, proceeding to room configuration');
    // Auto-advance to next step
    setTimeout(() => goNext(), 300);
  };

  if (loading) {
    return (
      <div className="flex flex-col items-center justify-center p-12 space-y-4 text-muted-foreground">
        <Loader2 className="h-8 w-8 animate-spin" />
        <p>Loading profiles...</p>
      </div>
    );
  }

  if (createNew) {
    return (
      <div className="max-w-md mx-auto space-y-6 animate-in fade-in slide-in-from-bottom-2">
        <div className="space-y-2 text-center">
          <h3 className="text-xl font-semibold">Create New Profile</h3>
          <p className="text-sm text-center text-muted-foreground">
            Enter a name for this configuration profile to save it for later.
          </p>
        </div>
        <div className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="profileName">Profile Name</Label>
            <Input
              id="profileName"
              placeholder="e.g. Daily PK Battle"
              value={newProfileName}
              onChange={(e) => setNewProfileName(e.target.value)}
              maxLength={50}
              autoFocus
            />
          </div>
          <div className="flex gap-3 pt-2">
            <Button variant="outline" className="flex-1" onClick={() => setCreateNew(false)}>
              Cancel
            </Button>
            <Button
              className="flex-1"
              onClick={handleCreateNew}
              disabled={newProfileName.trim().length < 3}
            >
              Continue
            </Button>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        {/* Create New Card */}
        <Card
          className="border-dashed border-2 cursor-pointer hover:border-primary hover:bg-muted/50 transition-all flex flex-col items-center justify-center min-h-[140px] group"
          onClick={() => setCreateNew(true)}
        >
          <div className="p-6 text-center space-y-2">
            <div className="w-12 h-12 rounded-full bg-primary/10 flex items-center justify-center mx-auto group-hover:bg-primary/20 transition-colors">
              <Plus className="h-6 w-6 text-primary" />
            </div>
            <h3 className="font-semibold text-primary">Create New Profile</h3>
          </div>
        </Card>

        {/* Existing Profiles */}
        {profiles.map((profile) => (
          <Card
            key={profile.id}
            className="cursor-pointer hover:border-primary hover:shadow-md transition-all relative overflow-hidden group"
            onClick={() => handleSelectProfile(profile)}
          >
            <div className="absolute inset-0 bg-primary/5 opacity-0 group-hover:opacity-100 transition-opacity" />

            {/* Delete Button */}
            <div className="absolute top-2 right-2 z-10 opacity-0 group-hover:opacity-100 transition-opacity duration-200">
              <Button
                variant="destructive"
                size="icon"
                className="h-7 w-7 rounded-full shadow-sm"
                onClick={(e) => handleDeleteProfile(e, profile.id, profile.name)}
              >
                <Trash2 className="h-3.5 w-3.5" />
              </Button>
            </div>

            <CardHeader className="space-y-1 pb-2">
              <div className="flex justify-between items-start gap-2 pr-6">
                <CardTitle className="text-base font-semibold truncate flex-1">
                  {profile.name}
                </CardTitle>
                {/* Bigo Avatar Display */}
                {profile.bigoAvatar ? (
                  <img src={profile.bigoAvatar} alt="Bigo Avatar" className="h-8 w-8 rounded-full object-cover ring-2 ring-primary/20" />
                ) : (
                  <div className="h-8 w-8 rounded-full bg-primary/10 flex items-center justify-center">
                    <User className="h-4 w-4 text-primary" />
                  </div>
                )}
              </div>
              <CardDescription className="flex items-center gap-1 text-[10px]">
                <Clock className="h-3 w-3" />
                Last used: {profile.lastUsedAt ? new Date(profile.lastUsedAt).toLocaleDateString() : 'Never'}
              </CardDescription>
            </CardHeader>
            <CardContent className="pt-0">
              {profile.bigoNickName && (
                <div className="text-sm font-medium text-emerald-600 mb-1 truncate">
                  {profile.bigoNickName}
                </div>
              )}
              <div className="text-xs text-muted-foreground flex flex-col gap-0.5">
                <span className="font-mono">{profile.roomId}</span>
                {/* <span className="text-[10px] opacity-70">Room ID</span> */}
              </div>
            </CardContent>
          </Card>
        ))}
      </div>
    </div>
  );
}
