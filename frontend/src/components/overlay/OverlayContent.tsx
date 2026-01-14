import React from 'react';
import { Gift, MessageCircle } from 'lucide-react';

interface OverlayContentProps {
    scene: string;
    connected: boolean;
    latestMessage: any;
    messages: any[];
    config?: any;
}

export const OverlayContent: React.FC<OverlayContentProps> = ({ scene, connected, latestMessage, messages, config }) => {
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
        // Prioritize message data for scores, but use config for static data (colors, names)
        // If message has team data, use it. If not, use config.
        const msgTeams = latestMessage?.teams;
        const configTeams = config?.teams;

        // Build Team 1 using Config as base, updating with Message data if available
        const team1Config = configTeams?.[0] || { name: 'Red Side', color: '#ff2e4d' };
        const team1Msg = msgTeams?.find((t: any) => t.teamId === team1Config.teamId) || msgTeams?.[0] || {};

        const team1 = {
            name: team1Config.name,
            color: team1Config.color || '#ff2e4d',
            score: team1Msg.score || 0
        };

        // Build Team 2
        const team2Config = configTeams?.[1] || { name: 'Blue Side', color: '#00b4fc' };
        const team2Msg = msgTeams?.find((t: any) => t.teamId === team2Config.teamId) || msgTeams?.[1] || {};

        const team2 = {
            name: team2Config.name,
            color: team2Config.color || '#00b4fc',
            score: team2Msg.score || 0
        };

        // Calculate progress percentage
        const totalScore = team1.score + team2.score;
        const redPercent = totalScore === 0 ? 50 : (team1.score / totalScore) * 100;

        return (
            <div className="w-full h-full relative font-sans">
                {/* Top Overlay Bar */}
                <div className="absolute top-8 left-1/2 -translate-x-1/2 w-[90%] max-w-[1000px] flex flex-col gap-1">

                    {/* Team Names Header */}
                    <div className="flex justify-between items-end px-2">
                        <div className="bg-black/80 text-white px-4 py-1 rounded-t-lg flex items-center gap-2 border-t border-l border-r border-white/10">
                            <div className="w-3 h-3 rounded-full" style={{ backgroundColor: team1.color }} />
                            <span className="font-bold uppercase tracking-wider text-sm">{team1.name}</span>
                        </div>

                        {/* VS Logo (Absolute center of bar) */}
                        <div className="absolute left-1/2 -translate-x-1/2 -top-2 z-20">
                            <span className="text-5xl font-black italic text-white drop-shadow-[0_4px_4px_rgba(0,0,0,0.5)]"
                                style={{ textShadow: '0 0 10px rgba(0,0,0,0.8), 2px 2px 0 #000' }}>
                                VS
                            </span>
                        </div>

                        <div className="bg-black/80 text-white px-4 py-1 rounded-t-lg flex items-center gap-2 border-t border-l border-r border-white/10">
                            <span className="font-bold uppercase tracking-wider text-sm">{team2.name}</span>
                            <div className="w-3 h-3 rounded-full" style={{ backgroundColor: team2.color }} />
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
                                {team1.score.toLocaleString()}
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
                                {team2.score.toLocaleString()}
                            </span>
                            {/* Gradient overlay for shine */}
                            <div className="absolute inset-0 bg-gradient-to-b from-white/20 to-transparent" />
                        </div>

                        {/* Center Divider Line */}
                        <div className="absolute left-1/2 top-0 bottom-0 w-1 bg-black/20 -translate-x-1/2 z-10" />
                    </div>

                    {/* Bottom Info Row */}
                    <div className="flex justify-between px-2 text-xs font-bold text-white/90 drop-shadow-md mt-1">
                        <div className="bg-black/60 px-3 py-1 rounded-full flex items-center gap-1">
                            <span>🏆 Target: 10,000</span>
                        </div>
                        <div className="bg-black/60 px-3 py-1 rounded-full flex items-center gap-1">
                            <span>⚡ MVP Mode</span>
                        </div>
                    </div>
                </div>

                {/* Gift Popup (Floating) - lower right */}
                <div className="absolute bottom-20 right-10 flex flex-col gap-2 pointer-events-none items-end">
                    {latestMessage && latestMessage.type === 'GIFT' && (
                        <div className="animate-in slide-in-from-right fade-in duration-300 bg-black/80 backdrop-blur-md border border-white/20 p-3 rounded-xl shadow-xl flex items-center gap-3 min-w-[200px]">
                            {latestMessage.senderAvatar && (
                                <img src={latestMessage.senderAvatar} alt="" className="w-12 h-12 rounded-full border-2 border-yellow-400" />
                            )}
                            <div className="text-right">
                                <div className="font-bold text-yellow-400 text-base">{latestMessage.senderName}</div>
                                <div className="text-white text-sm flex items-center justify-end gap-1">
                                    Sent <span className="font-bold text-pink-400">{latestMessage.giftName}</span> x{latestMessage.giftCount}
                                </div>
                            </div>
                        </div>
                    )}
                </div>

                {/* Only show logs if specifically requested or in debug */}
                {renderLog()}
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
