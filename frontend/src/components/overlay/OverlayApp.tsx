import React, { useEffect, useState, useRef, useCallback } from 'react';
import { useParams, useSearchParams } from 'react-router-dom';
import SockJS from 'sockjs-client';
import Stomp from 'stompjs';
import { OverlayContent } from './OverlayContent';

interface OverlayAppProps {
    // No props passed from router directly, uses params
}

export const OverlayApp: React.FC<OverlayAppProps> = () => {
    const [connected, setConnected] = useState(false);
    const [messages, setMessages] = useState<any[]>([]);
    const [giftLibrary, setGiftLibrary] = useState<any[]>([]);
    const giftLibraryRef = useRef<any[]>([]); // Ref to access inside interval closure
    const [config, setConfig] = useState<any>(null);
    const [error, setError] = useState<string>('');
    const [gameState, setGameState] = useState<any>(null);

    // Refs to access latest state inside event listeners (closures)
    const configRef = useRef<any>(null);
    const gameStateRef = useRef<any>(null);

    // Sync refs
    useEffect(() => { configRef.current = config; }, [config]);
    useEffect(() => { gameStateRef.current = gameState; }, [gameState]);

    // Parse query params
    const params = new URLSearchParams(window.location.search);
    const bbCoreUrl = params.get('bbCoreUrl');
    const token = params.get('token');
    const roomId = params.get('roomId');
    const scene = params.get('scene') || 'default';
    const team1ColorOverride = params.get('team1Color');
    const team2ColorOverride = params.get('team2Color');

    // Helper to extract state/teams from various payload structures
    const extractState = (msg: any): any => {
        if (!msg) return null;
        if (Array.isArray(msg.teams)) return msg;
        if (msg.data && Array.isArray(msg.data.teams)) return msg.data;
        if (msg.payload && Array.isArray(msg.payload.teams)) return msg.payload;
        if (msg.body && Array.isArray(msg.body.teams)) return msg.body;
        return msg; // Fallback
    };

    const handleMessage = (msg: any) => {
        console.log("[DataFlow] 5. Received Message:", msg?.type, msg);
        setMessages(prev => [msg, ...prev].slice(0, 50));

        // 0. Handle CONFIG_UPDATE (Priority)
        if (msg.type === 'CONFIG' || msg.type === 'CONFIG_UPDATE') {
            console.log("[DataFlow] Received CONFIG update via SSE", msg);
            const newConfig = msg.data || msg.payload || msg;
            if (newConfig && newConfig.teams) {
                setConfig(newConfig);
            }
            return;
        }

        // 1. Try to extract FULL state (sync)
        const potentialState = extractState(msg);
        if (potentialState && potentialState.teams) {
            setGameState(potentialState);
            return;
        }

        // 2. Handle Incremental Updates (GIFT events)
        if (msg.type === 'GIFT') {
            console.log("[DataFlow] Gift received. Triggering OPTIMISTIC score update.");

            // Use Ref to access latest config in closure
            const currentConfig = configRef.current;

            // If teamId is missing (direct broadcast from App), resolve it from config
            let targetTeamId = msg.teamId;
            if (!targetTeamId && currentConfig?.teams) {
                const foundTeam = currentConfig.teams.find((t: any) =>
                    t.streamers?.some((s: any) => s.bigoRoomId === msg.bigoRoomId)
                );
                if (foundTeam) {
                    targetTeamId = foundTeam.teamId;
                    console.log(`[DataFlow] Resolved teamId ${targetTeamId} for bigoRoomId ${msg.bigoRoomId}`);
                }
            }

            if (targetTeamId) {
                setGameState((prevState: any) => {
                    // Use Ref for fallback if prevState is missing
                    const cfg = configRef.current;
                    // If no state yet, try to build from config or wait
                    if (!prevState && !cfg) return null;
                    // Create base state from prev or config
                    const resultState = prevState ? { ...prevState } : {
                        ...cfg,
                        teams: cfg.teams?.map((t: any) => ({ ...t, score: t.score || 0 })) || []
                    };

                    const currentTeams = resultState.teams || [];
                    let matchFound = false;

                    const newTeams = currentTeams.map((team: any, index: number) => {
                        // 1. Strict ID Match
                        if (team.teamId === targetTeamId) {
                            matchFound = true;
                            const addedScore = msg.diamonds || msg.value || 1;
                            console.log(`[DataFlow] OPTIMISTIC: Adding score ${addedScore} to team ${team.teamId} (ID MATCH).`);
                            return { ...team, score: (team.score || 0) + addedScore };
                        }

                        // 2. Fallback: Index Match (if only 2 teams and we can infer side)
                        return team;
                    });

                    if (!matchFound) {
                        console.warn(`[DataFlow] MATCH FAILED. Target: ${targetTeamId}`);
                        console.log(`[DataFlow] Available Local Teams:`, currentTeams.map((t: any) => `${t.name} (${t.teamId})`));
                    }

                    return { ...resultState, teams: newTeams };
                });
            } else {
                console.warn("[DataFlow] Could not resolve teamId for gift event:", msg);
            }
        }

        // 3. Handle Other State Updates
        else if (msg.type === 'SCENE_UPDATE' || msg.type === 'STATE_UPDATE') {
            setGameState(msg);
        }
    };

    useEffect(() => {
        // Enforce transparent background for the overlay window
        document.documentElement.style.backgroundColor = 'transparent';
        document.body.style.backgroundColor = 'transparent';

        if (!bbCoreUrl || !roomId) {
            setError('Missing required parameters (bbCoreUrl, roomId)');
            return;
        }

        // 1. Fetch Config
        // Helper to enrich config with images from library
        const enrichConfig = (cfg: any, lib: any[]) => {
            console.log("[DataFlow] 2. Enriching Config. Input Settings:", cfg?.overlaySettings);
            if (!cfg || !cfg.teams || !lib || lib.length === 0) {
                console.log("[DataFlow] 2b. Enriching skipped (missing data/lib). Returning cfg.");
                return cfg;
            }
            const updatedTeams = cfg.teams.map((team: any) => {
                // If image is missing but we have a binding gift name
                if (!team.bindingGiftImage && team.bindingGift) {
                    const match = lib.find((g: any) => g.name.toLowerCase() === team.bindingGift.toLowerCase());
                    if (match && match.image) {
                        return { ...team, bindingGiftImage: match.image };
                    }
                }
                return team;
            });
            const result = { ...cfg, teams: updatedTeams };
            console.log("[DataFlow] 3. Enriched. Settings:", result?.overlaySettings);
            return result;
        };

        const fetchConfig = async () => {
            // 1. Try Local Config first (persisted by Wails App, contains OverlaySettings)
            try {
                // In dev (port 5173), hit localhost:3000. In prod (port 3000), hit relative path.
                const localUrl = window.location.port === "5173" ? "http://localhost:3000/config" : "/config";
                const localRes = await fetch(localUrl);
                if (localRes.ok) {
                    const localData = await localRes.json();
                    console.log("[DataFlow] 1a. Loaded LOCAL config:", localData?.overlaySettings);
                    setConfig((prev: any) => enrichConfig(localData, giftLibraryRef.current));
                    return; // Success! Local config has the full struct including visual settings.
                }
            } catch (e) {
                console.warn("[DataFlow] Local config fetch failed, falling back to BB-Core", e);
            }

            // 2. Fallback to BB-Core (Remote)
            try {
                // Determine headers for auth if token exists
                const headers: HeadersInit = token ? {
                    'Authorization': `Bearer ${token}`,
                    'X-Authorization': `Bearer ${token}`
                } : {};
                const res = await fetch(`${bbCoreUrl}/api/v1/external/config?roomId=${roomId}`, { headers });
                if (res.ok) {
                    const data = await res.json();
                    console.log("[DataFlow] 1b. Loaded REMOTE config:", data?.overlaySettings);
                    setConfig((prevConfig: any) => {
                        return enrichConfig(data, giftLibraryRef.current);
                    });
                } else if (res.status === 401) {
                    setError('Unauthorized: Check token or login status');
                } else {
                    setError(`Failed to load config: ${res.status} `);
                }
            } catch (e) {
                console.error("Failed to fetch overlay config", e);
            }
        };





        // Fetch gift library first
        const loadLib = async () => {
            try {
                const res = await fetch('/data/gifts.json');
                if (res.ok) {
                    const lib = await res.json();
                    setGiftLibrary(lib);
                    giftLibraryRef.current = lib;
                    // Trigger initial config fetch after lib load
                    fetchConfig();
                } else {
                    console.warn("Could not load local gifts.json");
                    fetchConfig();
                }
            } catch (e) {
                console.error("Error loading gifts.json", e);
                fetchConfig();
            }
        };

        loadLib();

        // Setup SSE for local events (Robust Gift Delivery)
        const localUrl = window.location.port === "5173" ? "http://localhost:3000/events" : "/events";
        console.log("[SSE] Connecting to:", localUrl);
        const eventSource = new EventSource(localUrl);

        eventSource.onopen = () => console.log("[SSE] Connected to local OverlayServer events");

        eventSource.onmessage = (event) => {
            try {
                const data = JSON.parse(event.data);
                // Filter by room if possible (though local server is usually 1 room active)
                if (data.roomId && data.roomId !== roomId) {
                    return;
                }
                console.log("[SSE] Received event:", data);
                handleMessage(data);
            } catch (e) {
                console.error("[SSE] Failed to parse event", e);
            }
        };


        // 2. Connect to WebSocket via SockJS/Stomp directly
        // Ensure clean base URL without trailing slash
        const cleanBaseUrl = bbCoreUrl.replace(/\/+$/, '');
        // Append token to URL query params for initial handshake authentication
        const socketUrl = token ? `${cleanBaseUrl}/ws?token=${encodeURIComponent(token)}` : `${cleanBaseUrl}/ws`;
        const socket = new SockJS(socketUrl);
        const stompClient = Stomp.over(socket);

        // Disable debug logs in production
        stompClient.debug = () => { };

        const stompHeaders = token ? {
            'Authorization': `Bearer ${token}`,
            'X-Authorization': `Bearer ${token}`
        } : {};

        stompClient.connect(stompHeaders, () => {
            setConnected(true);
            console.log('Overlay connected to BB-Core');

            // Subscribe to Config Updates (Push model)
            stompClient.subscribe(`/topic/room/${roomId}/config`, (message) => {
                if (message.body) {
                    try {
                        console.log("Received config update via STOMP");
                        console.log("[DataFlow] 1. Received config update via STOMP");
                        const newConfig = JSON.parse(message.body);
                        console.log("DEBUG: settings received:", newConfig.overlaySettings);
                        setConfig((prevConfig: any) => {
                            return enrichConfig(newConfig, giftLibraryRef.current);
                        });
                    } catch (e) {
                        console.error("Failed to parse config update", e);
                    }
                }
            });



            // Subscribe to Scene specific updates (Layout/Mode switching)
            stompClient.subscribe(`/topic/room/${roomId}/scene`, (message) => {
                try {
                    const body = JSON.parse(message.body);
                    handleMessage(body);
                } catch (e) {
                    console.error('Failed to parse scene message', e);
                }
            });

            // Subscribe to Gift specific updates (Visual popups)
            stompClient.subscribe(`/topic/room/${roomId}/gift`, (message) => {
                try {
                    const body = JSON.parse(message.body);
                    handleMessage(body);
                } catch (e) {
                    console.error('Failed to parse gift message', e);
                }
            });

            // Subscribe to Activity updates (Scores, Game State)
            stompClient.subscribe(`/topic/room/${roomId}/activity`, (message) => {
                try {
                    const body = JSON.parse(message.body);
                    handleMessage(body);
                } catch (e) {
                    console.error('Failed to parse activity message', e);
                }
            });

        }, (err) => {
            console.error('STOMP connection error:', err);
            setError('Connection lost. Retrying...');
            setConnected(false);
        });

        return () => {
            // Cleanup SSE
            if (eventSource) {
                eventSource.close();
            }
            // Cleanup STOMP
            if (stompClient && stompClient.connected) {
                stompClient.disconnect(() => { });
            }
        };
    }, [bbCoreUrl, roomId, token]);



    // Initialize GameState from Config
    // Initialize GameState from Config (and handle Config updates)
    useEffect(() => {
        if (!config) return;

        setGameState((prev: any) => {
            // 1. Initial Load: Use config directly (scores default to 0)
            if (!prev) {
                return {
                    ...config,
                    teams: config.teams?.map((t: any) => ({
                        ...t,
                        score: t.score || 0
                    })) || []
                };
            }

            // 2. Config Update (e.g., Settings Changed): Merge new visuals but PRESERVE scores
            const newTeams = config.teams?.map((t: any, i: number) => {
                // Try to find matching team in previous state
                // Match by teamId if possible, else index
                const oldTeam = prev.teams?.find((ot: any) => ot.teamId === t.teamId) || prev.teams?.[i];

                return {
                    ...t, // Use new config (visuals, names)
                    score: oldTeam ? oldTeam.score : (t.score || 0) // Preserve local score!
                };
            }) || [];

            return {
                ...config, // Use new config metadata
                teams: newTeams
            };
        });
        console.log("[DataFlow] State Refreshed via Config Update. New State Teams:", config.teams.map((t: any) => t.teamId));
    }, [config]);

    // Functions moved to top to prevent hoisting issues


    // ...

    // Merge URL overrides into config
    const displayConfig = {
        ...config,
        teams: config?.teams?.map((t: any, i: number) => ({
            ...t,
            color: (i === 0 && team1ColorOverride ? team1ColorOverride :
                i === 1 && team2ColorOverride ? team2ColorOverride : t.color)
        })) || []
    };
    console.log("[DataFlow] 4. DisplayConfig Params:", displayConfig?.overlaySettings);

    // If no config yet but we have overrides, makeshift a config to show *something* immediately
    if (!config && (team1ColorOverride || team2ColorOverride)) {
        displayConfig.teams = [
            { name: 'Red Side', color: team1ColorOverride || '#ff2e4d', teamId: 't1' },
            { name: 'Blue Side', color: team2ColorOverride || '#00b4fc', teamId: 't2' }
        ];
    }

    const [timer, setTimer] = useState("00:00");
    const [round, setRound] = useState(1);
    const [pkStats, setPkStats] = useState<any>({});

    // Parse Session Data from Config/GameState
    useEffect(() => {
        const source = gameState || config;
        if (source?.session?.scriptData) {
            setRound(source.session.scriptData.roundNumber || 1);
            setPkStats(source.session.scriptData.teamStats || {});
        }
    }, [config, gameState]);

    // Timer Logic
    useEffect(() => {
        const currentSession = gameState?.session || config?.session;
        if (!currentSession) return;

        const interval = setInterval(() => {
            const { startTime, durationMinutes, status, pausedAt, sessionId } = currentSession;

            if (status === 'PAUSED') {
                setTimer("PAUSED");
                return;
            }

            if (status !== 'ACTIVE') {
                setTimer("00:00");
                return;
            }

            const now = Date.now();
            const startMs = startTime < 10000000000 ? startTime * 1000 : startTime;
            const durationMs = (durationMinutes || 0) * 60 * 1000;
            const endTime = startMs + durationMs;

            const remaining = endTime - now;

            if (remaining <= 0) {
                setTimer("00:00");
                triggerNextRound(sessionId);
            } else {
                const m = Math.floor(remaining / 60000);
                const s = Math.floor((remaining % 60000) / 1000);
                setTimer(`${m.toString().padStart(2, '0')}:${s.toString().padStart(2, '0')}`);
            }
        }, 1000);

        return () => clearInterval(interval);
    }, [config, gameState]);

    // Trigger Next Round Helper
    const [isTransitioning, setIsTransitioning] = useState(false);

    const triggerNextRound = async (sessionId: string) => {
        if (isTransitioning || !sessionId) return;
        setIsTransitioning(true);
        try {
            console.log("Triggering Next Round for Session:", sessionId);
            const headers: HeadersInit = token ? {
                'Authorization': `Bearer ${token}`,
                'Content-Type': 'application/json'
            } : { 'Content-Type': 'application/json' };

            await fetch(`${bbCoreUrl}/api/v1/scripts/${sessionId}/pk/next-round`, {
                method: 'POST',
                headers
            });
            // Reset logic or wait for sync?
            // Ideally, the backend sends a new STATE_UPDATE which resets the timer/round
        } catch (e) {
            console.error("Failed to trigger next round:", e);
        } finally {
            // Allow retry after delay if still stuck
            setTimeout(() => setIsTransitioning(false), 5000);
        }
    };

    return (
        <div className="fixed inset-0 overflow-hidden bg-transparent">
            <OverlayContent
                scene={scene}
                connected={connected}
                latestMessage={messages[0]}
                gameState={gameState}
                messages={messages}
                config={displayConfig}
                timer={timer}
                round={round}
                pkStats={pkStats}
            />
        </div>
    );
};

export default OverlayApp;
