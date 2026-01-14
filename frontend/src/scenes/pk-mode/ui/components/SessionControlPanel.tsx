import { useState, useEffect } from 'react';
import { useToast } from "@/hooks/use-toast";
import { Card, CardHeader, CardTitle, CardContent, CardDescription } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Play, Square, Radio, Wifi, WifiOff, AlertCircle, Copy, Check, Gift, PlusCircle } from "lucide-react";
import {
    StartBigoListener,
    StopBigoListener,
    StartBBCoreStream,
    StopBBCoreStream,
    GetBigoListenerStatus,
    GetBBCoreStreamStatus,
    ResetSession,
    GetOverlayURL
} from '../../../../../wailsjs/go/main/App';


interface SessionControlPanelProps {
    config: any;
    roomId: string;
    durationMinutes: number;
    onBack: () => void;
    onSessionActiveChange?: (isActive: boolean) => void;
}

export function SessionControlPanel({ config, roomId, durationMinutes, onBack, onSessionActiveChange }: SessionControlPanelProps) {
    const [listenerStatus, setListenerStatus] = useState<any>(null);
    const [streamStatus, setStreamStatus] = useState<any>(null);
    const [listenerLoading, setListenerLoading] = useState(false);
    const [streamLoading, setStreamLoading] = useState(false);
    const [currentTime, setCurrentTime] = useState(new Date());
    const [overlayUrl, setOverlayUrl] = useState('');
    const [copied, setCopied] = useState(false);
    const { toast } = useToast();
    const [resetConfirm, setResetConfirm] = useState(false);
    const [giftLibrary, setGiftLibrary] = useState<any[]>([]);

    useEffect(() => {
        const loadLib = () => {
            const stored = localStorage.getItem('bbapp_gift_library');
            if (stored) {
                try {
                    setGiftLibrary(JSON.parse(stored));
                } catch (e) { console.error(e); }
            }
        };
        loadLib();
        // Listen for storage events in case config tab updates (simple poll for now if needed, or just reload on mount)
        const interval = setInterval(loadLib, 5000);
        return () => clearInterval(interval);
    }, []);

    const addToLibrary = (giftAttributes: any) => {
        const newEntry = {
            id: giftAttributes.GiftId,
            name: giftAttributes.GiftName,
            diamonds: giftAttributes.Diamonds || 0, // default to what we saw, or 0
            image: giftAttributes.GiftImageUrl
        };

        const updated = [...giftLibrary];
        // Check if exists
        const exists = updated.find(g => g.id === newEntry.id);
        if (!exists) {
            updated.push(newEntry);
            localStorage.setItem('bbapp_gift_library', JSON.stringify(updated));
            setGiftLibrary(updated);
            toast({ description: `Added ${newEntry.name} to Gift Library` });
        } else {
            toast({ description: "Gift already in library" });
        }
    };

    // Update current time every second for timers
    useEffect(() => {
        const interval = setInterval(() => setCurrentTime(new Date()), 1000);
        return () => clearInterval(interval);
    }, []);

    // Poll status every 2 seconds
    useEffect(() => {
        const fetchStatus = async () => {
            try {
                const [listener, stream] = await Promise.all([
                    GetBigoListenerStatus(),
                    GetBBCoreStreamStatus()
                ]);
                setListenerStatus(listener);
                setStreamStatus(stream);

                // Notify parent about active state
                if (onSessionActiveChange) {
                    const isActive = (listener && listener.isActive) || (stream && stream.isActive);
                    onSessionActiveChange(isActive);
                }
            } catch (error) {
                console.error('Failed to fetch status:', error);
            }
        };

        fetchStatus(); // Initial fetch
        const interval = setInterval(fetchStatus, 2000);
        return () => clearInterval(interval);
    }, [onSessionActiveChange]);

    // Generate Overlay URL with localstorage colors
    useEffect(() => {
        const generateUrl = async () => {
            if (!roomId) {
                setOverlayUrl("Waiting for Room ID...");
                return;
            }

            // Retrieve real token from localStorage
            const token = localStorage.getItem('auth_token') || "";
            try {
                // Temporary set to generating to show activity if roomId changed
                setOverlayUrl("Generating...");

                const url = await GetOverlayURL("pk-mode", roomId, token);
                if (url) {
                    // Append local colors
                    const c1 = localStorage.getItem(`bbapp_bg_color_${roomId}_0`);
                    const c2 = localStorage.getItem(`bbapp_bg_color_${roomId}_1`);
                    const newUrl = new URL(url);
                    if (c1) newUrl.searchParams.set('team1Color', c1);
                    if (c2) newUrl.searchParams.set('team2Color', c2);
                    setOverlayUrl(newUrl.toString());
                } else {
                    setOverlayUrl("Error: Backend returned empty URL");
                }
            } catch (e: any) {
                console.error("Failed to generate overlay URL", e);
                setOverlayUrl(`Error: ${e.toString()}`);
            }
        };
        generateUrl();
    }, [roomId]);

    const copyToClipboard = () => {
        if (!overlayUrl || overlayUrl.startsWith("Error") || overlayUrl.startsWith("Waiting") || overlayUrl === "Generating...") return;
        navigator.clipboard.writeText(overlayUrl);
        setCopied(true);
        toast({
            description: "Broadcast Overlay URL copied to clipboard",
        });
        setTimeout(() => setCopied(false), 2000);
    };

    const handleStartListener = async () => {
        try {
            setListenerLoading(true);
            await StartBigoListener(config);
            toast({
                title: "Listener Started",
                description: "Connected to Bigo room successfully.",
            });
        } catch (error: any) {
            toast({
                variant: "destructive",
                title: "Error",
                description: `Failed to start listener: ${error.toString()}`,
            });
        } finally {
            setListenerLoading(false);
        }
    };

    const handleStopListener = async () => {
        try {
            setListenerLoading(true);
            await StopBigoListener();
            toast({
                title: "Listener Stopped",
                description: "Bigo listener session ended.",
            });
        } catch (error: any) {
            toast({
                variant: "destructive",
                title: "Error",
                description: `Failed to stop listener: ${error.toString()}`,
            });
        } finally {
            setListenerLoading(false);
        }
    };

    const handleStartStream = async () => {
        try {
            setStreamLoading(true);
            await StartBBCoreStream(roomId, config, durationMinutes);
            toast({
                title: "Streaming Started",
                description: "Session is now live on BB-Core.",
            });
        } catch (error: any) {
            toast({
                variant: "destructive",
                title: "Error",
                description: `Failed to start streaming: ${error.toString()}`,
            });
        } finally {
            setStreamLoading(false);
        }
    };

    const handleStopStream = async () => {
        try {
            setStreamLoading(true);
            await StopBBCoreStream("User requested");
            toast({
                title: "Streaming Stopped",
                description: "BB-Core session ended.",
            });
        } catch (error: any) {
            toast({
                variant: "destructive",
                title: "Error",
                description: `Failed to stop streaming: ${error.toString()}`,
            });
        } finally {
            setStreamLoading(false);
        }
    };

    const handleReset = async () => {
        if (!resetConfirm) {
            setResetConfirm(true);
            // Auto-cancel confirmation after 3 seconds
            setTimeout(() => setResetConfirm(false), 3000);
            return;
        }

        try {
            setListenerLoading(true);
            await ResetSession();
            toast({
                title: "Session Reset",
                description: "Session manager has been forcibly reset.",
            });
        } catch (error: any) {
            toast({
                variant: "destructive",
                title: "Error",
                description: `Reset failed: ${error.toString()}`,
            });
        } finally {
            setListenerLoading(false);
            setResetConfirm(false);
        }
    };

    const listenerActive = listenerStatus?.isActive || false;
    const streamActive = streamStatus?.isActive || false;

    return (
        <div className="space-y-6 p-6">
            <div className="flex justify-between items-center">
                <div>
                    <h2 className="text-2xl font-bold">Session Control</h2>
                    <p className="text-sm text-muted-foreground">Room: {roomId}</p>
                </div>
                <div className="flex gap-2">
                    <Button
                        size="sm"
                        onClick={handleReset}
                        disabled={listenerLoading || streamLoading}
                        className={resetConfirm ? "bg-red-700 hover:bg-red-800 animate-pulse text-white transition-all" : "bg-red-500 hover:bg-red-600 text-white transition-all"}
                    >
                        {resetConfirm ? "Confirm Reset?" : "Reset Session"}
                    </Button>
                    <Button variant="outline" size="sm" onClick={onBack}>
                        ← Back to Setup
                    </Button>
                </div>
            </div>

            {/* Bigo Listener Session Card */}
            <Card className={listenerActive ? "border-green-500" : "border-muted"}>
                <CardHeader>
                    <div className="flex justify-between items-start">
                        <div>
                            <CardTitle className="flex items-center gap-2">
                                <Radio className="h-5 w-5" />
                                Bigo Listener Session
                                {listenerActive ? (
                                    <Badge variant="default" className="bg-green-600">Active</Badge>
                                ) : (
                                    <Badge variant="secondary">Inactive</Badge>
                                )}
                            </CardTitle>
                            <CardDescription>
                                Manages browser connections to Bigo rooms
                            </CardDescription>
                        </div>
                        {listenerActive && listenerStatus?.startTime && (
                            <div className="text-right">
                                <div className="text-sm font-mono text-green-600 font-bold">
                                    {(() => {
                                        const start = new Date(listenerStatus.startTime);
                                        const diff = Math.floor((currentTime.getTime() - start.getTime()) / 1000);
                                        if (diff < 0) return "00:00:00";
                                        const h = Math.floor(diff / 3600).toString().padStart(2, '0');
                                        const m = Math.floor((diff % 3600) / 60).toString().padStart(2, '0');
                                        const s = (diff % 60).toString().padStart(2, '0');
                                        return `${h}:${m}:${s}`;
                                    })()}
                                </div>
                                <div className="text-[10px] text-muted-foreground uppercase tracking-wider">Duration</div>
                            </div>
                        )}
                    </div>
                </CardHeader>
                <CardContent className="space-y-4">
                    {listenerStatus && (
                        <div className="grid grid-cols-2 gap-4 text-sm">
                            <div>
                                <div className="text-muted-foreground">Connected</div>
                                <div className="text-2xl font-bold text-green-600">
                                    {listenerStatus.connectedIdols || 0}
                                </div>
                            </div>
                            <div>
                                <div className="text-muted-foreground">Buffered Events</div>
                                <div className="text-2xl font-bold text-blue-600">
                                    {listenerStatus.bufferedEvents || 0}
                                </div>
                            </div>
                        </div>
                    )}

                    {listenerStatus?.connections && listenerStatus.connections.length > 0 && (
                        <div className="space-y-2">
                            <div className="text-sm font-semibold">Connections:</div>
                            <div className="space-y-1">
                                {listenerStatus.connections.map((conn: any, idx: number) => (
                                    <div key={idx} className="flex items-center gap-3 text-sm p-3 rounded bg-muted/40 border border-border/50 transition-all hover:bg-muted/60">
                                        {/* Avatar */}
                                        <div className="relative h-10 w-10 shrink-0">
                                            {conn.avatar ? (
                                                <img
                                                    src={conn.avatar}
                                                    alt={conn.idolName}
                                                    className="h-full w-full rounded-full object-cover border border-white/10"
                                                    onError={(e) => {
                                                        (e.target as HTMLImageElement).src = "https://ui-avatars.com/api/?name=" + (conn.idolName || "User") + "&background=random";
                                                    }}
                                                />
                                            ) : (
                                                <div className="h-full w-full rounded-full bg-secondary flex items-center justify-center text-xs font-bold text-secondary-foreground border border-white/10">
                                                    {(conn.idolName || "U").substring(0, 2).toUpperCase()}
                                                </div>
                                            )}
                                            {/* Status Indicator Dot */}
                                            <div className={`absolute bottom-0 right-0 h-3 w-3 rounded-full border-2 border-background ${conn.status === 'CONNECTED' ? 'bg-green-500' :
                                                conn.status === 'CONNECTING' ? 'bg-yellow-500 animate-pulse' :
                                                    'bg-red-500'
                                                }`} />
                                        </div>

                                        <div className="flex-1 flex flex-col min-w-0">
                                            <div className="flex items-center gap-2">
                                                <span className="font-semibold text-sm truncate text-foreground">
                                                    {conn.username || conn.idolName || "Unknown User"}
                                                </span>
                                                {conn.status === 'CONNECTED' && (
                                                    <Badge variant="outline" className="text-[10px] h-4 px-1 py-0 border-green-500/30 text-green-600 bg-green-500/5">
                                                        LIVE
                                                    </Badge>
                                                )}
                                            </div>
                                            <div className="flex items-center gap-2 text-xs text-muted-foreground font-mono">
                                                <span>ID: {conn.bigoId || conn.bigoRoomId}</span>
                                                {conn.messagesReceived > 0 && (
                                                    <>
                                                        <span className="opacity-30">•</span>
                                                        <span className="flex items-center gap-1">
                                                            <Wifi className="h-3 w-3 opacity-70" />
                                                            {conn.messagesReceived.toLocaleString()} msgs
                                                        </span>
                                                    </>
                                                )}
                                            </div>
                                        </div>

                                        <div className="text-right">
                                            <Badge
                                                variant={conn.status === 'CONNECTED' ? "default" : "outline"}
                                                className={`text-xs ${conn.status === 'CONNECTED' ? 'bg-green-600 hover:bg-green-700' :
                                                    conn.status === 'CONNECTING' ? 'text-yellow-600 border-yellow-600/30 bg-yellow-500/5' :
                                                        'text-red-600 border-red-600/30 bg-red-500/5'
                                                    }`}
                                            >
                                                {conn.status}
                                            </Badge>
                                        </div>
                                    </div>
                                ))}
                            </div>
                        </div>
                    )}

                    <div className="flex gap-2">
                        {!listenerActive ? (
                            <Button
                                onClick={handleStartListener}
                                disabled={listenerLoading}
                                className="bg-green-600 hover:bg-green-700 h-10 px-4"
                            >
                                <Play className="h-4 w-4 mr-2" />
                                {listenerLoading ? 'Starting...' : 'Start Listener'}
                            </Button>
                        ) : (
                            <Button
                                onClick={handleStopListener}
                                disabled={listenerLoading}
                                className="bg-red-600 hover:bg-red-700 text-white h-10 px-4"
                            >
                                <Square className="h-4 w-4 mr-2 fill-current" />
                                {listenerLoading ? 'Stopping...' : (
                                    <span className="flex items-center gap-2">
                                        Stop Listener
                                        <span className="bg-black/20 px-2 py-0.5 rounded text-xs font-mono">
                                            {(() => {
                                                if (!listenerStatus?.startTime) return "00:00:00";
                                                const start = new Date(listenerStatus.startTime);
                                                const diff = Math.floor((currentTime.getTime() - start.getTime()) / 1000);
                                                if (diff < 0) return "00:00:00";
                                                const h = Math.floor(diff / 3600).toString().padStart(2, '0');
                                                const m = Math.floor((diff % 3600) / 60).toString().padStart(2, '0');
                                                const s = (diff % 60).toString().padStart(2, '0');
                                                return `${h}:${m}:${s}`;
                                            })()}
                                        </span>
                                    </span>
                                )}
                            </Button>
                        )}
                    </div>
                </CardContent>
            </Card>

            {/* BB-Core Streaming Session Card */}
            <Card className={streamActive ? "border-blue-500" : "border-muted"}>
                <CardHeader>
                    <div className="flex justify-between items-start">
                        <div>
                            <CardTitle className="flex items-center gap-2">
                                <Wifi className="h-5 w-5" />
                                BB-Core Streaming Session
                                {streamActive ? (
                                    <Badge variant="default" className="bg-blue-600">Streaming</Badge>
                                ) : (
                                    <Badge variant="secondary">Inactive</Badge>
                                )}
                            </CardTitle>
                            <CardDescription>
                                Streams events to BB-Core via STOMP
                            </CardDescription>
                        </div>
                    </div>
                </CardHeader>
                <CardContent className="space-y-4">
                    {streamStatus && (
                        <div className="grid grid-cols-2 gap-4 text-sm">
                            <div>
                                <div className="text-muted-foreground">Session ID</div>
                                <div className="font-mono text-xs">
                                    {streamStatus.sessionId || '-'}
                                </div>
                            </div>
                            <div>
                                <div className="text-muted-foreground">Room ID</div>
                                <div className="font-mono text-xs">
                                    {streamStatus.roomId || roomId}
                                </div>
                            </div>
                        </div>
                    )}

                    {!listenerActive && (
                        <div className="flex items-center gap-2 p-3 rounded bg-yellow-500/10 border border-yellow-500/20">
                            <AlertCircle className="h-4 w-4 text-yellow-600" />
                            <span className="text-sm text-yellow-700">
                                Bigo listener must be active before starting streaming
                            </span>
                        </div>
                    )}

                    <div className="flex gap-2">
                        {!streamActive ? (
                            <Button
                                onClick={handleStartStream}
                                disabled={streamLoading || !listenerActive}
                                className="bg-blue-600 hover:bg-blue-700"
                            >
                                <Play className="h-4 w-4 mr-2" />
                                {streamLoading ? 'Starting...' : 'Start Streaming'}
                            </Button>
                        ) : (
                            <Button
                                onClick={handleStopStream}
                                disabled={streamLoading}
                                variant="destructive"
                            >
                                <Square className="h-4 w-4 mr-2" />
                                {streamLoading ? 'Stopping...' : 'Stop Streaming'}
                            </Button>
                        )}
                    </div>

                    {streamActive && (
                        <div className="text-xs text-muted-foreground">
                            Duration: {durationMinutes} minutes
                        </div>
                    )}
                </CardContent>
            </Card>
            {/* Overlay URL Card */}
            <Card className="border-purple-500/50">
                <CardHeader>
                    <CardTitle className="flex items-center gap-2 text-base">
                        <div className="bg-purple-500/10 p-2 rounded-full text-purple-600">
                            <Copy className="h-4 w-4" />
                        </div>
                        Broadcast Overlay URL
                    </CardTitle>
                    <CardDescription>
                        Use this URL in OBS Browser Source (Width: 1920, Height: 1080)
                    </CardDescription>
                </CardHeader>
                <CardContent>
                    <div className="flex gap-2">
                        <div className="flex-1 bg-muted p-2 rounded text-xs font-mono truncate border select-all">
                            {overlayUrl || "Generating..."}
                        </div>
                        <Button
                            size="sm"
                            variant="outline"
                            onClick={copyToClipboard}
                            className={copied ? "text-green-600 border-green-600" : ""}
                        >
                            {copied ? <Check className="h-4 w-4" /> : <Copy className="h-4 w-4" />}
                        </Button>
                    </div>
                </CardContent>
            </Card>
            {/* Recent Gifts Log Card */}
            <Card className="border-purple-500/30">
                <CardHeader>
                    <CardTitle className="flex items-center gap-2 text-base">
                        <div className="bg-pink-500/10 p-2 rounded-full text-pink-600">
                            <Gift className="h-4 w-4" />
                        </div>
                        Recent Gifts Log
                    </CardTitle>
                    <CardDescription>
                        Real-time log of received gifts. Click IDs to copy.
                    </CardDescription>
                </CardHeader>
                <CardContent>
                    <div className="border rounded-md max-h-[300px] overflow-y-auto bg-muted/20">
                        {listenerStatus?.recentGifts && listenerStatus.recentGifts.length > 0 ? (
                            <div className="divide-y divide-border">
                                {listenerStatus.recentGifts.map((gift: any, idx: number) => (
                                    <div key={idx} className="flex items-center gap-3 p-3 hover:bg-muted/50 transition-colors">
                                        {/* Gift Image (Click to copy URL) */}
                                        <div
                                            className="h-10 w-10 shrink-0 bg-background rounded border p-0.5 cursor-pointer hover:border-pink-500 transition-colors relative group"
                                            onClick={() => {
                                                navigator.clipboard.writeText(gift.GiftImageUrl);
                                                toast({ description: "Gift Image URL copied!" });
                                            }}
                                            title="Click to copy Image URL"
                                        >
                                            <img
                                                src={gift.GiftImageUrl}
                                                alt={gift.GiftName}
                                                className="h-full w-full object-contain"
                                                onError={(e) => {
                                                    (e.target as HTMLImageElement).src = "https://placehold.co/40x40?text=?";
                                                }}
                                            />
                                            <div className="absolute inset-0 bg-black/50 opacity-0 group-hover:opacity-100 flex items-center justify-center rounded">
                                                <Copy className="h-4 w-4 text-white" />
                                            </div>
                                        </div>

                                        <div className="flex-1 min-w-0">
                                            <div className="flex items-center gap-2">
                                                <span className="font-semibold text-sm text-foreground truncate">{gift.GiftName}</span>
                                                <span className="text-xs text-muted-foreground font-mono bg-muted px-1 rounded cursor-pointer hover:text-foreground hover:bg-muted/80 flex items-center gap-1"
                                                    onClick={() => {
                                                        navigator.clipboard.writeText(gift.GiftId);
                                                        toast({ description: "Gift ID copied!" });
                                                    }}
                                                    title="Click to copy ID"
                                                >
                                                    ID: {gift.GiftId}
                                                    <Copy className="h-3 w-3 opacity-50" />
                                                </span>
                                            </div>
                                            <div className="flex items-center gap-2 text-xs text-muted-foreground">
                                                <span>From: <span className="font-medium text-foreground">{gift.SenderName}</span></span>
                                                <span>•</span>
                                                <span>To: {gift.StreamerName || "Me"}</span>
                                                <span>•</span>
                                                <span>x{gift.GiftCount}</span>
                                                <span className="text-pink-500 font-mono">({gift.Diamonds} 💎)</span>
                                            </div>
                                        </div>

                                        <div className="text-xs font-mono text-muted-foreground whitespace-nowrap flex flex-col items-end gap-1">
                                            {new Date(gift.Timestamp).toLocaleTimeString()}
                                            {/* Add to Library Button if unknown */}
                                            {!giftLibrary.find((g: any) => g.id === gift.GiftId) && (
                                                <Button
                                                    variant="ghost"
                                                    size="icon"
                                                    className="h-6 w-6 text-muted-foreground hover:text-green-600"
                                                    title="Add to Gift Library"
                                                    onClick={() => addToLibrary(gift)}
                                                >
                                                    <PlusCircle className="h-4 w-4" />
                                                </Button>
                                            )}
                                        </div>
                                    </div>
                                ))}
                            </div>
                        ) : (
                            <div className="p-8 text-center text-muted-foreground text-sm">
                                Waiting for gifts...
                            </div>
                        )}
                    </div>
                </CardContent>
            </Card>
        </div >
    );
}
