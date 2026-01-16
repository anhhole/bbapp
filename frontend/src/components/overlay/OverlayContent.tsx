import React from 'react';
import { Gift, MessageCircle } from 'lucide-react';

interface OverlayContentProps {
    scene: string;
    connected: boolean;
    latestMessage: any;
    messages: any[];
    config?: any;
    gameState?: any;
    timer?: string;
    round?: number;
    pkStats?: any;
}

export const OverlayContent: React.FC<OverlayContentProps> = ({
    scene, connected, latestMessage, messages, config, gameState,
    timer = "00:00", round = 1, pkStats = {}
}) => {
    // Common visual for logs/debug if scene requires it
    const renderLog = () => (
        <div className="absolute bottom-0 left-0 p-4 w-full bg-gradient-to-t from-black/80 to-transparent text-white font-mono text-xs">
            <div className="mb-2 flex items-center gap-2">
                <div className={`w-2 h-2 rounded-full ${connected ? 'bg-green-500' : 'bg-red-500'}`} />
                <span>{connected ? 'Connected' : 'Disconnected'}</span>
            </div>
            <div className="space-y-1 max-h-32 overflow-hidden flex flex-col-reverse">
                {messages.map((m, i) => (
                    <div key={i} className="opacity-80">
                        [{m.type}] {m.senderName}: {m.giftName || m.message}
                    </div>
                ))}
            </div>
        </div>
    );

    // Render different scenes
    if (scene === 'pk-mode') {
        // Prioritize gameState for scores (persistent), fall back to latestMessage (transient) for initial state
        const stateSource = gameState || latestMessage;
        const msgTeams = stateSource?.teams;
        const configTeams = config?.teams;

        // Build Team 1 using Config as base, updating with Message data if available
        const team1Config = configTeams?.[0] || { name: 'Red Side', color: '#ff2e4d' };

        // Robust find: ID match -> Index match -> Empty
        let team1Msg = msgTeams?.find((t: any) => t.teamId && team1Config.teamId && t.teamId === team1Config.teamId);
        if (!team1Msg && msgTeams && msgTeams.length > 0) team1Msg = msgTeams[0]; // Fallback to first item
        team1Msg = team1Msg || {};

        const team1 = {
            name: team1Config.name,
            color: team1Config.color || '#ff2e4d',
            score: team1Msg.score || team1Msg.totalScore || team1Msg.diamonds || team1Msg.points || 0,
            avatar: team1Config.streamers?.[0]?.avatar || '',
            bindingGift: team1Config.bindingGift || '',
            bindingGiftImage: team1Config.bindingGiftImage || ''
        };

        // Build Team 2
        const team2Config = configTeams?.[1] || { name: 'Blue Side', color: '#00b4fc' };

        // Robust find: ID match -> Index match -> Empty
        let team2Msg = msgTeams?.find((t: any) => t.teamId && team2Config.teamId && t.teamId === team2Config.teamId);
        if (!team2Msg && msgTeams && msgTeams.length > 1) team2Msg = msgTeams[1]; // Fallback to second item
        team2Msg = team2Msg || {};

        const team2 = {
            name: team2Config.name,
            color: team2Config.color || '#00b4fc',
            score: team2Msg.score || team2Msg.totalScore || team2Msg.diamonds || team2Msg.points || 0,
            avatar: team2Config.streamers?.[0]?.avatar || '',
            bindingGift: team2Config.bindingGift || '',
            bindingGiftImage: team2Config.bindingGiftImage || ''
        };

        // Calculate progress percentage
        const totalScore = team1.score + team2.score;
        const redPercent = totalScore === 0 ? 50 : (team1.score / totalScore) * 100;

        const team1Stats = pkStats?.[team1Config.teamId] || { wins: 0, streak: 0 };
        const team2Stats = pkStats?.[team2Config.teamId] || { wins: 0, streak: 0 };

        const showScores = true; // Always show scores
        const showTimer = true; // Always show timer
        const showAvatar = config?.overlaySettings?.showStreamerAvatar !== false;
        const showWinStreak = config?.overlaySettings?.showWinStreak !== false;

        console.log("OverlayContent Render [v2]:", {
            settings: config?.overlaySettings,
            showAvatar,
            typeOfShowAvatar: typeof showAvatar,
            team1Avatar: team1.avatar
        });

        return (
            <div className="w-full h-full relative font-sans">
                {/* Top Overlay Bar */}
                <div className="absolute top-8 left-1/2 -translate-x-1/2 w-[90%] max-w-[1000px] flex flex-col gap-1">

                    {/* Team Names Header & Avatars */}
                    <div className="flex justify-between items-end px-2">
                        {/* Team 1 Header */}
                        <div className="flex items-center gap-2">
                            {showAvatar && team1.avatar && (
                                <img src={team1.avatar} alt="Avatar" className="w-12 h-12 rounded-full border-2 border-white shadow-lg bg-gray-800 object-cover" />
                            )}
                            <div className="flex flex-col items-start">
                                <div className="bg-black/80 text-white px-4 py-1 rounded-t-lg flex items-center gap-2 border-t border-l border-r border-white/10">
                                    <div className="w-3 h-3 rounded-full" style={{ backgroundColor: team1.color }} />
                                    <span className="font-bold uppercase tracking-wider text-sm">{team1.name}</span>
                                    {team1.bindingGiftImage ? (
                                        <img src={team1.bindingGiftImage} alt="Gift" className="w-6 h-6 object-contain ml-2" />
                                    ) : team1.bindingGift && (
                                        <span className="text-xs text-yellow-400 border border-yellow-400/30 px-1 rounded bg-yellow-400/10 ml-2">
                                            {team1.bindingGift}
                                        </span>
                                    )}
                                </div>
                                {/* Wins Badge */}
                                {showWinStreak && (
                                    <div className="bg-yellow-500/90 text-black text-xs font-bold px-2 py-0.5 rounded-b ml-2 shadow-sm">
                                        Wins: {team1Stats.wins}
                                    </div>
                                )}
                            </div>
                        </div>

                        {/* VS Logo & Timer (Absolute center of bar) */}
                        <div className="absolute left-1/2 -translate-x-1/2 -top-6 z-20 flex flex-col items-center">
                            {/* Round Badge */}
                            {showTimer && (
                                <div className="bg-blue-600 text-white text-[10px] font-bold px-2 py-0.5 rounded-full mb-1 shadow border border-blue-400">
                                    ROUND {round}
                                </div>
                            )}
                            {showTimer && (
                                <div className="bg-black/90 text-white font-mono font-bold text-xl px-4 py-1 rounded-full border border-white/20 shadow-[0_0_15px_rgba(255,255,255,0.2)] mb-1">
                                    {timer}
                                </div>
                            )}
                            <span className="text-5xl font-black italic text-white drop-shadow-[0_4px_4px_rgba(0,0,0,0.5)]"
                                style={{ textShadow: '0 0 10px rgba(0,0,0,0.8), 2px 2px 0 #000' }}>
                                VS
                            </span>
                        </div>

                        {/* Team 2 Header */}
                        <div className="flex items-center gap-2 flex-row-reverse">
                            {showAvatar && team2.avatar && (
                                <img src={team2.avatar} alt="Avatar" className="w-12 h-12 rounded-full border-2 border-white shadow-lg bg-gray-800 object-cover" />
                            )}
                            <div className="flex flex-col items-end">
                                <div className="bg-black/80 text-white px-4 py-1 rounded-t-lg flex items-center gap-2 border-t border-l border-r border-white/10">
                                    {team2.bindingGiftImage ? (
                                        <img src={team2.bindingGiftImage} alt="Gift" className="w-6 h-6 object-contain mr-2" />
                                    ) : team2.bindingGift && (
                                        <span className="text-xs text-yellow-400 border border-yellow-400/30 px-1 rounded bg-yellow-400/10 mr-2">
                                            {team2.bindingGift}
                                        </span>
                                    )}
                                    <span className="font-bold uppercase tracking-wider text-sm">{team2.name}</span>
                                    <div className="w-3 h-3 rounded-full" style={{ backgroundColor: team2.color }} />
                                </div>
                                {/* Wins Badge */}
                                {showWinStreak && (
                                    <div className="bg-yellow-500/90 text-black text-xs font-bold px-2 py-0.5 rounded-b mr-2 shadow-sm">
                                        Wins: {team2Stats.wins}
                                    </div>
                                )}
                            </div>
                        </div>
                    </div>

                    {/* Main Progress Bar Container */}
                    <div className="h-16 relative w-full rounded-2xl overflow-hidden shadow-2xl border-4 border-black/50 flex bg-black/80">
                        {/* Red Side (Left) */}
                        <div
                            className="h-full flex items-center justify-start px-6 transition-all duration-500 ease-out relative"
                            style={{
                                width: `${redPercent}%`,
                                backgroundColor: team1.color,
                                minWidth: '0%' // transitions handle the rest
                            }}
                        >
                            <span className="text-4xl font-black text-white drop-shadow-md z-10 whitespace-nowrap">
                                {showScores && team1.score.toLocaleString()}
                            </span>
                            {/* Gradient overlay for shine */}
                            <div className="absolute inset-0 bg-gradient-to-b from-white/20 to-transparent" />
                        </div>

                        {/* Blue Side (Right) */}
                        <div
                            className="h-full flex-1 flex items-center justify-end px-6 relative"
                            style={{
                                backgroundColor: team2.color,
                                minWidth: '0%'
                            }}
                        >
                            <span className="text-4xl font-black text-white drop-shadow-md z-10 whitespace-nowrap">
                                {showScores && team2.score.toLocaleString()}
                            </span>
                            {/* Gradient overlay for shine */}
                            <div className="absolute inset-0 bg-gradient-to-b from-white/20 to-transparent" />
                        </div>

                        {/* Center Divider Line */}
                        <div className="absolute left-1/2 top-0 bottom-0 w-1 bg-black/20 -translate-x-1/2 z-10" />
                    </div>

                    {/* Bottom Info Row - Removed as per request */}
                    {/* <div className="flex justify-between px-2 text-xs font-bold text-white/90 drop-shadow-md mt-1"> ... </div> */}
                </div>

                {/* Gift Popup (Floating) - REMOVED as per user request */}
                {/* <div className="absolute bottom-20 right-10 flex flex-col gap-2 pointer-events-none items-end"> ... </div> */}

                {/* Only show logs if specifically requested or in debug */}
                {/* {renderLog()} */}
            </div>
        );
    }

    // Default / Fallback
    return (
        <div className="w-full h-full flex items-center justify-center text-white">
            <div className="bg-black/40 p-6 rounded-xl backdrop-blur-sm text-center">
                <h2 className="text-xl font-bold mb-2">Overlay Active</h2>
                <p className="opacity-70">Scene: {scene}</p>
                <p className="opacity-70 text-sm mt-2">{connected ? 'Waiting for events...' : 'Connecting...'}</p>
            </div>
        </div>
    );
};
