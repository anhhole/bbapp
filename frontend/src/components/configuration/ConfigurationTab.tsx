import { useState, useEffect } from 'react';
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { SaveBBAppConfig, FetchConfig, FetchBigoUser, FetchGlobalIdols, SaveGlobalIdols, SaveGiftLibrary, GetGiftLibrary } from '../../../wailsjs/go/main/App';
import { useToast } from "@/hooks/use-toast";

export function ConfigurationTab() {
    const { toast } = useToast();
    const [roomId, setRoomId] = useState('');
    const [loading, setLoading] = useState(false);
    const [config, setConfig] = useState<any>(null);
    const [activeTab, setActiveTab] = useState<'bindings' | 'streamers' | 'gifts'>('streamers');

    // Gift Library State
    const [giftLibrary, setGiftLibrary] = useState<any[]>([]);
    const [newGift, setNewGift] = useState({ id: '', name: '', diamonds: 0, image: '' });

    // Idol Library State
    const [globalIdols, setGlobalIdols] = useState<any[]>([]);
    const [newIdolBigoId, setNewIdolBigoId] = useState('');
    const [isFetchingIdol, setIsFetchingIdol] = useState(false);

    useEffect(() => {
        fetchGlobalIdols();
        loadGiftLibrary();
    }, []);

    const loadGiftLibrary = async () => {
        try {
            let lib = await GetGiftLibrary();

            // Migration Logic: If backend empty, check local storage
            if (!lib || lib.length === 0) {
                const localStored = localStorage.getItem('bbapp_gift_library');
                if (localStored) {
                    try {
                        const localLib = JSON.parse(localStored);
                        if (Array.isArray(localLib) && localLib.length > 0) {
                            console.log("Migrating gift library from localStorage to backend...");
                            lib = localLib;
                            await SaveGiftLibrary(localLib); // Save to backend
                            toast({ description: "Gift Library migrated to new storage!" });
                        }
                    } catch (e) {
                        console.error("Failed to parse local gift library", e);
                    }
                }
            }

            if (lib && Array.isArray(lib)) {
                setGiftLibrary(lib);
            }
        } catch (e) {
            console.error("Failed to parse gift library", e);
        }
    };

    const saveGiftLibrary = async (updatedLib: any[]) => {
        setGiftLibrary(updatedLib);
        try {
            await SaveGiftLibrary(updatedLib);
            toast({ description: "Gift Library saved!" });
        } catch (e) {
            console.error(e)
            toast({ variant: "destructive", description: "Failed to save gift library" });
        }
    };

    const handleAddGift = () => {
        if (!newGift.id || !newGift.name) {
            toast({ variant: "destructive", description: "Gift ID and Name are required" });
            return;
        }
        const updated = [...giftLibrary, { ...newGift }];
        saveGiftLibrary(updated);
        setNewGift({ id: '', name: '', diamonds: 0, image: '' });
    };

    const handleUpdateGiftValue = (index: number, val: string) => {
        const updated = [...giftLibrary];
        updated[index].diamonds = parseInt(val) || 0;
        saveGiftLibrary(updated);
    }

    const handleRemoveGift = (index: number) => {
        const updated = [...giftLibrary];
        updated.splice(index, 1);
        saveGiftLibrary(updated);
    };

    const fetchGlobalIdols = async () => {
        try {
            const data = await FetchGlobalIdols();
            if (data && Array.isArray(data)) {
                setGlobalIdols(data);
            } else {
                setGlobalIdols([]);
            }
        } catch (error) {
            console.log("No global streamers found, starting fresh.");
            setGlobalIdols([]);
        }
    };

    const saveGlobalIdols = async (updatedIdols: any[]) => {
        try {
            await SaveGlobalIdols(updatedIdols);
            setGlobalIdols(updatedIdols);
        } catch (error) {
            console.error("Failed to save global streamers:", error);
        }
    };

    const handleAddIdol = async () => {
        if (!newIdolBigoId) return;

        setIsFetchingIdol(true);
        let finalName = "";
        let avatarUrl = "";

        try {
            // Fetch info from Bigo
            const info = await FetchBigoUser(newIdolBigoId) as any;
            if (info) {
                if (info.avatar) avatarUrl = info.avatar;
                if (info.nick_name) finalName = info.nick_name;
            }
        } catch (error) {
            console.error("Failed to fetch Bigo user info:", error);
        } finally {
            setIsFetchingIdol(false);
        }

        if (!finalName) {
            finalName = newIdolBigoId; // Use ID as fallback name
        }

        const updatedIdols = [...globalIdols, { name: finalName, bigoRoomId: newIdolBigoId, avatar: avatarUrl }];
        await saveGlobalIdols(updatedIdols);
        setNewIdolBigoId('');
    };

    const handleRemoveIdol = async (index: number) => {
        const updatedIdols = [...globalIdols];
        updatedIdols.splice(index, 1);
        await saveGlobalIdols(updatedIdols);
    };

    const handleFetch = async () => {
        if (!roomId) return;
        setLoading(true);
        try {
            const data = await FetchConfig(roomId);
            setConfig(data);
        } catch (error) {
            console.error(error);
        } finally {
            setLoading(false);
        }
    };

    const handleSave = async () => {
        if (!roomId || !config) return;
        setLoading(true);
        try {
            await SaveBBAppConfig(roomId, config);
        } catch (error) {
            console.error(error);
        } finally {
            setLoading(false);
        }
    };

    // Helper to update a binding
    const updateBinding = (teamIndex: number, idolIndex: number | null, giftName: string) => {
        const newConfig = { ...config };
        if (idolIndex === null) {
            // Update Team Binding
            newConfig.teams[teamIndex].bindingGift = giftName;
        } else {
            // Update Idol Binding
            newConfig.teams[teamIndex].streamers[idolIndex].bindingGift = giftName;
        }
        setConfig(newConfig);
    };

    return (
        <div className="space-y-6 animate-in fade-in duration-500">
            <div className="flex space-x-2 border-b pb-2">
                <Button
                    variant={activeTab === 'streamers' ? 'default' : 'ghost'}
                    onClick={() => setActiveTab('streamers')}
                >
                    Idol Library
                </Button>
                <Button
                    variant={activeTab === 'bindings' ? 'default' : 'ghost'}
                    onClick={() => setActiveTab('bindings')}
                >
                    Room Bindings
                </Button>
                <Button
                    variant={activeTab === 'gifts' ? 'default' : 'ghost'}
                    onClick={() => setActiveTab('gifts')}
                >
                    Gift Library
                </Button>
            </div>

            {activeTab === 'streamers' && (
                <div className="space-y-4">
                    <div className="border rounded-lg p-4 bg-card">
                        <h3 className="font-semibold mb-4">Add New Idol</h3>
                        <div className="flex gap-4">
                            <Input
                                placeholder="Bigo Idol ID"
                                value={newIdolBigoId}
                                onChange={(e) => setNewIdolBigoId(e.target.value)}
                            />
                            <Button onClick={handleAddIdol} disabled={!newIdolBigoId || isFetchingIdol}>
                                {isFetchingIdol ? "Fetching..." : "Add & Fetch"}
                            </Button>
                        </div>
                    </div>

                    <div className="border rounded-lg bg-card overflow-hidden">
                        <Table>
                            <TableHeader>
                                <TableRow>
                                    <TableHead className="w-[60px]">Avatar</TableHead>
                                    <TableHead>Name</TableHead>
                                    <TableHead>Bigo Idol ID</TableHead>
                                    <TableHead className="w-[100px]">Action</TableHead>
                                </TableRow>
                            </TableHeader>
                            <TableBody>
                                {globalIdols.length === 0 && (
                                    <TableRow>
                                        <TableCell colSpan={3} className="text-center text-muted-foreground">
                                            No streamers found. Add one above.
                                        </TableCell>
                                    </TableRow>
                                )}
                                {globalIdols.map((idol, index) => (
                                    <TableRow key={index}>
                                        <TableCell>
                                            <div className="h-8 w-8 rounded-full overflow-hidden bg-muted border">
                                                {idol.avatar ? (
                                                    <img src={idol.avatar} alt={idol.name} className="h-full w-full object-cover" />
                                                ) : (
                                                    <div className="h-full w-full flex items-center justify-center text-xs text-muted-foreground">?</div>
                                                )}
                                            </div>
                                        </TableCell>
                                        <TableCell className="font-medium">{idol.name}</TableCell>
                                        <TableCell className="font-mono text-xs">{idol.bigoRoomId}</TableCell>
                                        <TableCell>
                                            <Button variant="destructive" size="sm" onClick={() => handleRemoveIdol(index)}>Remove</Button>
                                        </TableCell>
                                    </TableRow>
                                ))}
                            </TableBody>
                        </Table>
                    </div>
                </div>
            )}

            {activeTab === 'bindings' && (
                <>
                    <div className="flex items-center space-x-4 p-4 border rounded-lg bg-card">
                        <Input
                            placeholder="Enter Room ID"
                            value={roomId}
                            onChange={(e) => setRoomId(e.target.value)}
                            className="max-w-xs"
                        />
                        <Button onClick={handleFetch} disabled={loading}>
                            {loading ? 'Loading...' : 'Load Config'}
                        </Button>
                    </div>

                    {config && (
                        <div className="border rounded-lg bg-card overflow-hidden">
                            <div className="p-4 border-b bg-muted/50 flex justify-between items-center">
                                <h3 className="font-semibold">Gift Bindings</h3>
                                <Button onClick={handleSave} disabled={loading} size="sm">Save Changes</Button>
                            </div>
                            <Table>
                                <TableHeader>
                                    <TableRow>
                                        <TableHead>Entity</TableHead>
                                        <TableHead>Type</TableHead>
                                        <TableHead>Bound Gift (Trigger)</TableHead>
                                    </TableRow>
                                </TableHeader>
                                <TableBody>
                                    {config.teams?.map((team: any, tIndex: number) => (
                                        <>
                                            {/* Team Row */}
                                            <TableRow key={`team-${tIndex}`} className="bg-muted/20">
                                                <TableCell className="font-medium">{team.name}</TableCell>
                                                <TableCell><span className="text-xs bg-blue-100 text-blue-800 px-2 py-0.5 rounded">Team</span></TableCell>
                                                <TableCell>
                                                    <Input
                                                        value={team.bindingGift}
                                                        onChange={(e) => updateBinding(tIndex, null, e.target.value)}
                                                        className="h-8 w-[200px]"
                                                        placeholder="e.g. Rose"
                                                    />
                                                </TableCell>
                                            </TableRow>
                                            {/* Streamer Rows */}
                                            {team.streamers?.map((idol: any, sIndex: number) => (
                                                <TableRow key={`idol-${tIndex}-${sIndex}`}>
                                                    <TableCell className="pl-8 text-muted-foreground">↳ {idol.name}</TableCell>
                                                    <TableCell><span className="text-xs bg-purple-100 text-purple-800 px-2 py-0.5 rounded">Idol</span></TableCell>
                                                    <TableCell>
                                                        <Input
                                                            value={idol.bindingGift}
                                                            onChange={(e) => updateBinding(tIndex, sIndex, e.target.value)}
                                                            className="h-8 w-[200px]"
                                                            placeholder="Inherit or Override"
                                                        />
                                                    </TableCell>
                                                </TableRow>
                                            ))}
                                        </>
                                    ))}
                                </TableBody>
                            </Table>
                        </div>
                    )}
                </>
            )}

            {activeTab === 'gifts' && (
                <div className="space-y-4">
                    <div className="border rounded-lg p-4 bg-card">
                        <h3 className="font-semibold mb-4">Add New Gift</h3>
                        <div className="grid grid-cols-4 gap-4">
                            <Input
                                placeholder="Gift ID"
                                value={newGift.id}
                                onChange={(e) => setNewGift({ ...newGift, id: e.target.value })}
                            />
                            <Input
                                placeholder="Gift Name"
                                value={newGift.name}
                                onChange={(e) => setNewGift({ ...newGift, name: e.target.value })}
                            />
                            <Input
                                type="number"
                                placeholder="Diamonds"
                                value={newGift.diamonds}
                                onChange={(e) => setNewGift({ ...newGift, diamonds: parseInt(e.target.value) || 0 })}
                            />
                            <Input
                                placeholder="Image URL (Optional)"
                                value={newGift.image}
                                onChange={(e) => setNewGift({ ...newGift, image: e.target.value })}
                            />
                        </div>
                        <Button className="mt-4" onClick={handleAddGift} disabled={!newGift.id || !newGift.name}>
                            Add to Library
                        </Button>
                    </div>

                    <div className="border rounded-lg bg-card overflow-hidden">
                        <Table>
                            <TableHeader>
                                <TableRow>
                                    <TableHead className="w-[60px]">Image</TableHead>
                                    <TableHead>Gift Name</TableHead>
                                    <TableHead>ID</TableHead>
                                    <TableHead>Diamond Value</TableHead>
                                    <TableHead className="w-[100px]">Action</TableHead>
                                </TableRow>
                            </TableHeader>
                            <TableBody>
                                {giftLibrary.length === 0 && (
                                    <TableRow>
                                        <TableCell colSpan={5} className="text-center text-muted-foreground">
                                            No gifts in library. Add one above or from Session Log.
                                        </TableCell>
                                    </TableRow>
                                )}
                                {giftLibrary.map((gift, index) => (
                                    <TableRow key={index}>
                                        <TableCell>
                                            <div className="h-8 w-8 rounded bg-muted border p-0.5">
                                                {gift.image ? (
                                                    <img src={gift.image} alt={gift.name} className="h-full w-full object-contain" />
                                                ) : (
                                                    <div className="h-full w-full flex items-center justify-center text-xs text-muted-foreground">?</div>
                                                )}
                                            </div>
                                        </TableCell>
                                        <TableCell className="font-medium">{gift.name}</TableCell>
                                        <TableCell className="font-mono text-xs">{gift.id}</TableCell>
                                        <TableCell>
                                            <Input
                                                type="number"
                                                className="w-24 h-8"
                                                value={gift.diamonds}
                                                onChange={(e) => handleUpdateGiftValue(index, e.target.value)}
                                            />
                                        </TableCell>
                                        <TableCell>
                                            <Button variant="destructive" size="sm" onClick={() => handleRemoveGift(index)}>Remove</Button>
                                        </TableCell>
                                    </TableRow>
                                ))}
                            </TableBody>
                        </Table>
                    </div>
                </div>
            )}
        </div>
    );
}
