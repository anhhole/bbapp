import { useState, useEffect } from 'react';
import { FetchConfig, SaveBBAppConfig, FetchGlobalIdols, FetchBigoUser } from '../../../wailsjs/go/main/App';
import type { WizardState, ToastType } from './types';
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Card, CardHeader, CardTitle, CardContent, CardDescription } from "@/components/ui/card";
import { Plus, Trash2, Save, DownloadCloud, AlertTriangle, CheckCircle, Search, Users } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { DndContext, DragEndEvent, DragOverlay, DragStartEvent, useDraggable, useDroppable } from '@dnd-kit/core';

interface RoomConfigStepProps {
  state: WizardState;
  updateState: (updates: Partial<WizardState>) => void;
  addToast: (type: ToastType, message: string, persistent?: boolean) => void;
  accessToken: string;
  goNext: () => void;
}

// Draggable Idol Component
function DraggableIdol({ idol }: { idol: any }) {
  const { attributes, listeners, setNodeRef, transform, isDragging } = useDraggable({
    id: idol.bigoRoomId,
    data: { idol },
  });

  const style = transform ? {
    transform: `translate3d(${transform.x}px, ${transform.y}px, 0)`,
    opacity: isDragging ? 0.5 : 1,
  } : undefined;

  return (
    <div
      ref={setNodeRef}
      style={style}
      {...listeners}
      {...attributes}
      className="flex items-center gap-2 p-2 rounded-md bg-muted/50 hover:bg-muted cursor-grab active:cursor-grabbing border border-transparent hover:border-primary/50 transition-all"
    >
      {idol.avatar ? (
        <img src={idol.avatar} alt={idol.name} className="h-8 w-8 rounded-full object-cover" />
      ) : (
        <div className="h-8 w-8 rounded-full bg-primary/20 flex items-center justify-center text-xs">
          {idol.name.charAt(0)}
        </div>
      )}
      <div className="flex-1 min-w-0">
        <div className="text-xs font-medium truncate">{idol.name}</div>
        <div className="text-[10px] text-muted-foreground font-mono truncate">{idol.bigoRoomId}</div>
      </div>
    </div>
  );
}

// Droppable Team Card Component
function DroppableTeamCard({ team, teamIndex, onUpdateTeam, onRemoveTeam, canRemove, onRemoveIdol, giftLibrary }: any) {
  const { setNodeRef, isOver } = useDroppable({
    id: team.teamId,
    data: { teamIndex },
  });

  return (
    <Card
      ref={setNodeRef}
      className={`border-dashed transition-all ${isOver ? 'ring-2 ring-primary bg-primary/5' : ''}`}
    >
      <CardHeader className="py-3 px-4 bg-muted/20">
        <div className="flex justify-between items-center gap-4">
          <div className="flex items-center gap-4 flex-1">
            <div className="grid gap-1 flex-1">
              <Label className="text-xs">Team Name</Label>
              <Input
                value={team.name}
                onChange={(e) => onUpdateTeam(teamIndex, 'name', e.target.value)}
                className="h-8"
              />
            </div>
            <div className="grid gap-1 flex-1">
              <Label className="text-xs">Binding Gift</Label>
              <Select
                value={team.bindingGift}
                onValueChange={(val) => onUpdateTeam(teamIndex, 'bindingGift', val)}
              >
                <SelectTrigger className="h-8">
                  <SelectValue placeholder="Select gift" />
                </SelectTrigger>
                <SelectContent>
                  {/* Add current value if not in library to prevent it from disappearing */}
                  {team.bindingGift && !giftLibrary?.find((g: any) => g.name === team.bindingGift) && (
                    <SelectItem value={team.bindingGift}>{team.bindingGift} (Current)</SelectItem>
                  )}
                  {giftLibrary?.map((gift: any) => (
                    <SelectItem key={gift.id || gift.name} value={gift.name}>
                      <div className="flex items-center gap-2">
                        {gift.image && <img src={gift.image} className="w-4 h-4 object-contain" alt="" />}
                        <span>{gift.name}</span>
                      </div>
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className="grid gap-1 flex-1">
              <Label className="text-xs">Color (Hex)</Label>
              <div className="flex gap-2">
                <div
                  className="w-8 h-8 rounded border shadow-sm cursor-pointer relative overflow-hidden"
                  style={{ backgroundColor: team.color || '#000000' }}
                >
                  <input
                    type="color"
                    value={team.color || '#000000'}
                    onChange={(e) => onUpdateTeam(teamIndex, 'color', e.target.value)}
                    className="absolute inset-0 opacity-0 cursor-pointer w-full h-full p-0 border-0"
                  />
                </div>
                <Input
                  value={team.color || ''}
                  onChange={(e) => onUpdateTeam(teamIndex, 'color', e.target.value)}
                  placeholder="#000000"
                  className="h-8 font-mono"
                />
              </div>
            </div>
          </div>
          {canRemove && (
            <Button variant="ghost" size="icon" className="text-destructive h-8 w-8" onClick={() => onRemoveTeam(teamIndex)}>
              <Trash2 className="h-4 w-4" />
            </Button>
          )}
        </div>
      </CardHeader>
      <CardContent className="pt-4 pb-4 px-4">
        <div className="space-y-3">
          <div className="flex justify-between items-center">
            <Label className="text-xs font-semibold text-muted-foreground uppercase">Idols ({team.streamers?.length || 0})</Label>
          </div>

          {team.streamers?.length === 0 ? (
            <div className="text-center py-8 border-2 border-dashed rounded-lg bg-muted/20">
              <p className="text-xs text-muted-foreground">Drag streamers from the library to add them</p>
            </div>
          ) : (
            <div className="space-y-2">
              {team.streamers?.map((idol: any, idolIndex: number) => (
                <div key={idol.id || `idol-${idolIndex}`} className="flex items-center gap-2 p-2 rounded-md bg-muted/30 border">
                  {idol.avatar ? (
                    <img src={idol.avatar} alt={idol.name} className="h-8 w-8 rounded-full object-cover" />
                  ) : (
                    <div className="h-8 w-8 rounded-full bg-primary/20 flex items-center justify-center text-xs">
                      {idol.name.charAt(0)}
                    </div>
                  )}
                  <div className="flex-1 min-w-0">
                    <div className="text-sm font-medium">{idol.name}</div>
                    <div className="text-xs text-muted-foreground font-mono">{idol.bigoRoomId}</div>
                  </div>
                  <Button variant="ghost" size="icon" className="h-7 w-7 text-destructive hover:bg-destructive/10" onClick={() => onRemoveIdol(teamIndex, idolIndex)}>
                    <Trash2 className="h-3 w-3" />
                  </Button>
                </div>
              ))}
            </div>
          )}
        </div>
      </CardContent>
    </Card>
  );
}

export function RoomConfigStep({
  state,
  updateState,
  addToast,
  accessToken,
  goNext,
}: RoomConfigStepProps) {
  const [roomId, setRoomId] = useState(state.roomId);
  const [loading, setLoading] = useState(false);
  const [editableConfig, setEditableConfig] = useState<any>(null);
  const [isEditing, setIsEditing] = useState(false);
  const [globalIdols, setGlobalIdols] = useState<any[]>([]);
  const [activeIdol, setActiveIdol] = useState<any>(null);
  const [giftLibrary, setGiftLibrary] = useState<any[]>([]);

  useEffect(() => {
    const stored = localStorage.getItem('bbapp_gift_library');
    if (stored) {
      try {
        setGiftLibrary(JSON.parse(stored));
      } catch (e) { }
    }
  }, []);

  useEffect(() => {
    if (state.roomId && state.roomId !== roomId) {
      setRoomId(state.roomId);
    }
  }, [state.roomId]);

  useEffect(() => {
    if (state.config && !editableConfig) {
      setEditableConfig(JSON.parse(JSON.stringify(state.config)));
      setIsEditing(true);
    }
  }, [state.config]);

  useEffect(() => {
    const loadIdols = async () => {
      try {
        const data = await FetchGlobalIdols();
        if (data && Array.isArray(data)) {
          setGlobalIdols(data);
        }
      } catch (e) {
        console.log("No global streamers found");
      }
    };
    loadIdols();
  }, []);

  const generateDefaultTemplate = (roomId: string) => ({
    roomId,
    agencyId: 0,
    session: { sessionId: '', startTime: 0, status: '' },
    teams: [
      { teamId: 'team-1', name: 'Team A', color: '#ff2e4d', bindingGift: 'Rose', scoreMultipliers: {}, streamers: [] },
      { teamId: 'team-2', name: 'Team B', color: '#00b4fc', bindingGift: 'Diamond', scoreMultipliers: {}, streamers: [] },
    ],
  });

  const handleFetchConfig = async () => {
    if (!roomId.trim()) {
      addToast('error', 'Please enter a Room ID');
      return;
    }

    try {
      setLoading(true);

      // 1. Fetch Room Config from BB-Core
      let config;
      try {
        config = await FetchConfig(roomId.trim());
        addToast('success', 'Configuration fetched successfully');
      } catch (error: any) {
        const errorStr = error.toString().toLowerCase();
        if (errorStr.includes('404') || errorStr.includes('not found') || errorStr.includes('stream.not.found')) {
          // Room doesn't exist in BB-Core, create a default template locally
          config = generateDefaultTemplate(roomId.trim());
          console.log('Room not found, loading default template locally:', config);
          addToast('success', 'New room configuration created locally.');
        } else {
          throw error;
        }
      }

      // 2. Fetch Bigo User Info (Avatar/Name)
      try {
        // Import FetchBigoUser at top level if not already, or assume it's available via wailsjs
        // We need to import it first. 
        // NOTE: I will add the import in a separate edit if it's missing, but based on the code it's imported in the file view I saw.
        // Wait, I saw "FetchGlobalIdols" but not "FetchBigoUser" in the imports in line 2.
        // I will add the import in a separate hunk.
        const { FetchBigoUser } = await import('../../../wailsjs/go/main/App');

        const bigoInfo = await FetchBigoUser(roomId.trim());
        if (bigoInfo) {
          console.log("Fetched Bigo Info:", bigoInfo);
          const roomHostIdol = {
            id: `idol-${Date.now()}`, // Generate a unique ID
            name: bigoInfo.nick_name,
            avatar: bigoInfo.avatar,
            bigoRoomId: roomId.trim(),
            bigoId: roomId.trim(),
            bindingGift: config?.teams?.[0]?.bindingGift || 'Rose', // Inherit,
            isRoomHost: true
          };

          setActiveIdol(roomHostIdol);
          addToast('success', `Verified Bigo Room: ${bigoInfo.nick_name}`);

          // Save Bigo Info to Profile if we have a profile ID
          if (state.profileId) {
            try {
              const { UpdateProfileBigoInfo } = await import('../../../wailsjs/go/main/App');
              await UpdateProfileBigoInfo(state.profileId, bigoInfo.avatar, bigoInfo.nick_name);
              console.log("Updated profile with Bigo info");
            } catch (err) {
              console.error("Failed to update profile bigo info:", err);
            }
          }
        }
      } catch (bigoError) {
        console.warn("Failed to fetch Bigo info for room:", bigoError);
        // Non-fatal, just continue
      }

      updateState({ roomId: roomId.trim(), config, isValid: true });
      if (!editableConfig) {
        setEditableConfig(config);
      }
      setTimeout(() => goNext(), 500);

    } catch (error: any) {
      addToast('error', `Failed to fetch config: ${error.toString()}`, true);
      updateState({ isValid: false });
    } finally {
      setLoading(false);
    }
  };

  const handleTeamChange = (teamIndex: number, field: string, value: string) => {
    if (!editableConfig) return;
    const newConfig = { ...editableConfig };
    newConfig.teams[teamIndex] = { ...newConfig.teams[teamIndex], [field]: value };
    setEditableConfig(newConfig);

    // Persist color to localStorage for overlay usage without backend dependency
    if (field === 'color') {
      try {
        const bgKey = `bbapp_bg_color_${roomId}_${teamIndex}`;
        localStorage.setItem(bgKey, value);
        // Also save to a unified config object in LS if needed, but individual keys is fine for now
      } catch (e) {
        console.error("Failed to save color to LS", e);
      }
    }
  };

  const handleAddTeam = () => {
    if (!editableConfig) return;
    const newTeam = {
      teamId: `team-${Date.now()}`,
      name: `Team ${editableConfig.teams.length + 1}`,
      color: '#888888',
      bindingGift: 'Rose',
      scoreMultipliers: {},
      streamers: [],
    };
    setEditableConfig({ ...editableConfig, teams: [...editableConfig.teams, newTeam] });
  };

  const handleRemoveTeam = (teamIndex: number) => {
    if (!editableConfig || editableConfig.teams.length <= 1) {
      addToast('error', 'At least one team is required');
      return;
    }
    const newConfig = { ...editableConfig };
    newConfig.teams.splice(teamIndex, 1);
    setEditableConfig(newConfig);
  };

  const handleRemoveIdol = (teamIndex: number, idolIndex: number) => {
    if (!editableConfig) return;
    const newConfig = { ...editableConfig };
    newConfig.teams[teamIndex].streamers.splice(idolIndex, 1);
    setEditableConfig(newConfig);
  };

  const handleDragStart = (event: DragStartEvent) => {
    const idol = globalIdols.find(i => i.bigoRoomId === event.active.id);
    setActiveIdol(idol);
  };

  const handleDragEnd = (event: DragEndEvent) => {
    setActiveIdol(null);
    const { active, over } = event;

    if (!over || !editableConfig) return;

    const idolBigoId = active.id as string;
    const teamId = over.id as string;

    const selectedIdol = globalIdols.find(i => i.bigoRoomId === idolBigoId);
    const teamIndex = editableConfig.teams.findIndex((t: any) => t.teamId === teamId);

    if (!selectedIdol || teamIndex === -1) return;

    const existingIdol = editableConfig.teams[teamIndex].streamers?.find(
      (i: any) => i.bigoRoomId === selectedIdol.bigoRoomId
    );

    if (existingIdol) {
      addToast('error', `${selectedIdol.name} is already in this team`);
      return;
    }

    const newIdol = {
      id: `idol-${Date.now()}`,
      bigoId: selectedIdol.bigoRoomId,
      bigoRoomId: selectedIdol.bigoRoomId,
      name: selectedIdol.name,
      avatar: selectedIdol.avatar,
      bindingGift: editableConfig.teams[teamIndex].bindingGift,
    };

    const newConfig = { ...editableConfig };
    if (!newConfig.teams[teamIndex].streamers) {
      newConfig.teams[teamIndex].streamers = [];
    }
    newConfig.teams[teamIndex].streamers.push(newIdol);
    setEditableConfig(newConfig);
    addToast('success', `Added ${selectedIdol.name} to ${newConfig.teams[teamIndex].name}`);
  };

  const handleSaveConfig = async () => {
    if (!editableConfig) return;
    try {
      setLoading(true);
      await SaveBBAppConfig(state.roomId, editableConfig);
      updateState({ config: editableConfig, isValid: true });
      setIsEditing(false);
      addToast('success', 'Configuration saved. Click Next to continue.');
      setTimeout(() => goNext(), 500);
    } catch (error: any) {
      addToast('error', `Failed to save: ${error.toString()}`, true);
    } finally {
      setLoading(false);
    }
  };

  // Sync editableConfig to parent state whenever it changes
  useEffect(() => {
    if (editableConfig) {
      // Basic validation: at least one team with one idol? 
      // User didn't specify strict validation, but we should ensure it's not empty.
      const isValid = editableConfig.teams.length > 0;
      updateState({ config: editableConfig, isValid });
    }
  }, [editableConfig, updateState]);

  // Bind Enter key to goNext
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Enter') {
        // If we are in the config step and it's valid, go next
        if (editableConfig && editableConfig.teams.length > 0) {
          goNext();
        }
      }
    };
    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [goNext, editableConfig]);

  return (
    <DndContext onDragStart={handleDragStart} onDragEnd={handleDragEnd}>
      <div className="space-y-6">
        <div className="text-center space-y-2">
          <h3 className="text-lg font-medium">Room Configuration</h3>
          <p className="text-sm text-muted-foreground">Enter your BB-Core Room ID to fetch the configuration.</p>
        </div>

        {/* ... (Keep Fetch Card logic same) ... */}

        <Card className="max-w-md mx-auto">
          <CardContent className="pt-6">
            <div className="flex gap-2">
              <Input
                placeholder="Enter Room ID..."
                value={roomId}
                onChange={(e) => setRoomId(e.target.value)}
                disabled={loading}
                className="font-mono text-center"
                onKeyDown={(e) => {
                  if (e.key === 'Enter') {
                    e.stopPropagation(); // prevent global enter if focusing fetch
                    handleFetchConfig();
                  }
                }}
              />
              <Button onClick={handleFetchConfig} disabled={loading || !roomId.trim()}>
                {loading ? <DownloadCloud className="h-4 w-4 animate-bounce mr-2" /> : <Search className="h-4 w-4 mr-2" />}
                Fetch
              </Button>
            </div>
            {/* ... Verified Room display ... */}
            {activeIdol && activeIdol.isRoomHost && (
              <div className="mt-4 flex items-center gap-3 p-3 bg-green-500/10 border border-green-500/20 rounded-md animate-in fade-in slide-in-from-top-2">
                {activeIdol.avatar ? (
                  <img src={activeIdol.avatar} alt={activeIdol.name} className="h-10 w-10 rounded-full object-cover ring-2 ring-green-500/30" />
                ) : (
                  <div className="h-10 w-10 rounded-full bg-green-500/20 flex items-center justify-center text-sm font-bold text-green-700">
                    {activeIdol.name.charAt(0)}
                  </div>
                )}
                <div className="flex-1">
                  <div className="text-xs text-green-600 font-medium uppercase tracking-wider">Verified Bigo Room</div>
                  <div className="font-semibold text-foreground flex items-center gap-2">
                    {activeIdol.name}
                    <Badge variant="outline" className="text-[10px] h-5 bg-background font-mono">{activeIdol.bigoRoomId}</Badge>
                  </div>
                </div>
              </div>
            )}
          </CardContent>
        </Card>

        {editableConfig && isEditing ? (
          <div className="grid grid-cols-[300px_1fr] gap-6">
            {/* Idol Library Card */}
            <Card className="border-primary/30 h-fit sticky top-4">
              <CardHeader className="pb-3">
                <CardTitle className="text-sm flex items-center gap-2">
                  <Users className="h-4 w-4" />
                  Idol Library
                </CardTitle>
                <CardDescription className="text-xs">Drag streamers to teams</CardDescription>
              </CardHeader>
              <CardContent className="space-y-2">
                {globalIdols.length === 0 ? (
                  <div className="text-center py-8 text-muted-foreground text-xs">
                    No streamers in library. Add them in Configuration tab.
                  </div>
                ) : (
                  globalIdols.map((idol) => <DraggableIdol key={idol.bigoRoomId} idol={idol} />)
                )}
              </CardContent>
            </Card>

            {/* Edit Configuration Card */}
            <Card className="border-primary/50 shadow-md">
              <CardHeader className="bg-muted/30 pb-4">
                <div className="flex justify-between items-center">
                  <CardTitle className="text-xl flex items-center gap-2">
                    Edit Configuration
                    <Badge variant="outline" className="font-mono">{editableConfig.roomId || state.roomId}</Badge>
                  </CardTitle>
                  <Button variant="outline" size="sm" onClick={handleAddTeam}>
                    <Plus className="h-4 w-4 mr-2" />
                    Add Team
                  </Button>
                </div>
              </CardHeader>
              <CardContent className="space-y-6 pt-6">
                {editableConfig.teams.map((team: any, teamIndex: number) => (
                  <DroppableTeamCard
                    key={team.teamId || `team-${teamIndex}`}
                    team={team}
                    teamIndex={teamIndex}
                    onUpdateTeam={handleTeamChange}
                    onRemoveTeam={handleRemoveTeam}
                    canRemove={editableConfig.teams.length > 1}
                    onRemoveIdol={handleRemoveIdol}
                    giftLibrary={giftLibrary}
                  />
                ))}
              </CardContent>
            </Card>
          </div>
        ) : state.config ? (
          <Card className="border-green-500/50 bg-green-500/5">
            <CardHeader>
              <CardTitle className="text-green-700 flex items-center gap-2">
                <CheckCircle className="h-5 w-5" />
                Configuration Ready
              </CardTitle>
              <CardDescription>
                The configuration for Room <strong>{state.config.roomId}</strong> is valid.
              </CardDescription>
            </CardHeader>
            <CardContent>
              <div className="grid grid-cols-3 gap-4 text-center">
                <div className="bg-background p-3 rounded-lg border">
                  <div className="text-2xl font-bold">{state.config.teams?.length || 0}</div>
                  <div className="text-xs text-muted-foreground uppercase">Teams</div>
                </div>
                <div className="bg-background p-3 rounded-lg border">
                  <div className="text-2xl font-bold">
                    {state.config.teams?.reduce((sum: number, team: any) => sum + (team.streamers?.length || 0), 0) || 0}
                  </div>
                  <div className="text-xs text-muted-foreground uppercase">Idols</div>
                </div>
                <div className="bg-background p-3 rounded-lg border flex flex-col justify-center items-center">
                  <span className="text-xs text-muted-foreground">Auto-Next in</span>
                  <span className="font-bold text-green-600">Proceeding...</span>
                </div>
              </div>
            </CardContent>
          </Card>
        ) : (
          <div className="text-center p-8 bg-muted/20 rounded-lg border-2 border-dashed">
            <AlertTriangle className="h-12 w-12 text-muted-foreground mx-auto mb-4 opacity-50" />
            <p className="text-muted-foreground">Fetch a configuration to get started.</p>
          </div>
        )}

        <DragOverlay>
          {activeIdol ? (
            <div className="flex items-center gap-2 p-2 rounded-md bg-primary text-primary-foreground shadow-lg">
              {activeIdol.avatar ? (
                <img src={activeIdol.avatar} alt={activeIdol.name} className="h-8 w-8 rounded-full object-cover" />
              ) : (
                <div className="h-8 w-8 rounded-full bg-primary-foreground/20 flex items-center justify-center text-xs">
                  {activeIdol.name.charAt(0)}
                </div>
              )}
              <div className="text-sm font-medium">{activeIdol.name}</div>
            </div>
          ) : null}
        </DragOverlay>
      </div>
    </DndContext>
  );
}
