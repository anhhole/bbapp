import React, { useEffect, useState } from 'react';
import SockJS from 'sockjs-client';
import Stomp from 'stompjs';
import { OverlayContent } from './OverlayContent';

interface OverlayAppProps { }

export const OverlayApp: React.FC<OverlayAppProps> = () => {
    const [connected, setConnected] = useState(false);
    const [messages, setMessages] = useState<any[]>([]);
    const [config, setConfig] = useState<any>(null);
    const [error, setError] = useState<string>('');

    // Parse query params
    const params = new URLSearchParams(window.location.search);
    const bbCoreUrl = params.get('bbCoreUrl');
    const token = params.get('token');
    const roomId = params.get('roomId');
    const scene = params.get('scene') || 'default';
    const team1ColorOverride = params.get('team1Color');
    const team2ColorOverride = params.get('team2Color');

    useEffect(() => {
        // Enforce transparent background for the overlay window
        document.documentElement.style.backgroundColor = 'transparent';
        document.body.style.backgroundColor = 'transparent';

        if (!bbCoreUrl || !roomId) {
            setError('Missing required parameters (bbCoreUrl, roomId)');
            return;
        }

        // 1. Fetch Config
        const fetchConfig = async () => {
            try {
                // Determine headers for auth if token exists
                // Determine headers for auth if token exists
                const headers: HeadersInit = token ? {
                    'Authorization': `Bearer ${token}`,
                    'X-Authorization': `Bearer ${token}`
                } : {};
                const res = await fetch(`${bbCoreUrl}/api/v1/external/config?roomId=${roomId}`, { headers });
                if (res.ok) {
                    const data = await res.json();
                    setConfig(data);
                } else if (res.status === 401) {
                    setError('Unauthorized: Check token or login status');
                } else {
                    setError(`Failed to load config: ${res.status}`);
                }
            } catch (e) {
                console.error("Failed to fetch overlay config", e);
            }
        };
        fetchConfig();

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

            // Subscribe to room updates
            stompClient.subscribe(`/topic/room/${roomId}`, (message) => {
                try {
                    const body = JSON.parse(message.body);
                    handleMessage(body);
                } catch (e) {
                    console.error('Failed to parse message', e);
                }
            });

            // Subscribe to Scene specific updates (likely where high-freq events go)
            stompClient.subscribe(`/topic/room/${roomId}/scene`, (message) => {
                try {
                    const body = JSON.parse(message.body);
                    handleMessage(body);
                } catch (e) {
                    console.error('Failed to parse scene message', e);
                }
            });

            // Subscribe to app-specific updates
            stompClient.subscribe(`/queue/app/room/${roomId}`, (message) => {
                // Handle direct messages
            });

        }, (err) => {
            console.error('STOMP connection error:', err);
            setError('Connection lost. Retrying...');
            setConnected(false);
        });

        return () => {
            if (stompClient && stompClient.connected) {
                stompClient.disconnect(() => { });
            }
        };
    }, [bbCoreUrl, roomId, token]);

    const handleMessage = (msg: any) => {
        setMessages(prev => [msg, ...prev].slice(0, 50));
    };

    if (error) {
        return (
            <div className="fixed inset-0 flex items-center justify-center bg-black/80 text-white p-4 font-mono text-sm border-2 border-red-500 rounded m-4">
                Error: {error}
            </div>
        );
    }

    // Merge URL overrides into config
    const displayConfig = {
        ...config,
        teams: config?.teams?.map((t: any, i: number) => ({
            ...t,
            color: (i === 0 && team1ColorOverride ? team1ColorOverride :
                i === 1 && team2ColorOverride ? team2ColorOverride : t.color)
        })) || []
    };

    // If no config yet but we have overrides, makeshift a config to show *something* immediately
    if (!config && (team1ColorOverride || team2ColorOverride)) {
        displayConfig.teams = [
            { name: 'Red Side', color: team1ColorOverride || '#ff2e4d', teamId: 't1' },
            { name: 'Blue Side', color: team2ColorOverride || '#00b4fc', teamId: 't2' }
        ];
    }

    return (
        <div className="fixed inset-0 overflow-hidden bg-transparent">
            <OverlayContent
                scene={scene}
                connected={connected}
                latestMessage={messages[0]}
                messages={messages}
                config={displayConfig}
            />
        </div>
    );
};
