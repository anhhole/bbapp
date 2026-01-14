import { useState, useEffect, useRef } from 'react';
import { Button } from "@/components/ui/button";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Badge } from "@/components/ui/badge";
import { Trash2, Download, Activity, Wifi } from "lucide-react";

interface LogEntry {
    id: string;
    timestamp: string;
    level: 'info' | 'warn' | 'error' | 'success';
    message: string;
}

export function MonitorTab() {
    const [logs, setLogs] = useState<LogEntry[]>([]);
    const scrollRef = useRef<HTMLDivElement>(null);
    const [isConnected, setIsConnected] = useState(true); // Mock connection status

    // Mock incoming logs
    useEffect(() => {
        const interval = setInterval(() => {
            const types: LogEntry['level'][] = ['info', 'info', 'success', 'warn'];
            const randomType = types[Math.floor(Math.random() * types.length)];
            const newLog: LogEntry = {
                id: Math.random().toString(36).substr(2, 9),
                timestamp: new Date().toLocaleTimeString(),
                level: randomType,
                message: `System event occurred: ${Math.random().toString(36).substring(7)}`
            };
            addLog(newLog);
        }, 3000);

        return () => clearInterval(interval);
    }, []);

    const addLog = (log: LogEntry) => {
        setLogs(prev => [...prev, log].slice(-100)); // Keep last 100 logs
    };

    // Auto-scroll to bottom
    useEffect(() => {
        if (scrollRef.current) {
            scrollRef.current.scrollIntoView({ behavior: 'smooth' });
        }
    }, [logs]);

    const clearLogs = () => setLogs([]);

    const downloadLogs = () => {
        const content = logs.map(l => `[${l.timestamp}] [${l.level.toUpperCase()}] ${l.message}`).join('\n');
        const blob = new Blob([content], { type: 'text/plain' });
        const url = URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        a.download = `bbapp-logs-${new Date().toISOString().split('T')[0]}.txt`;
        a.click();
    };

    const getLevelColor = (level: LogEntry['level']) => {
        switch (level) {
            case 'error': return 'text-red-500';
            case 'warn': return 'text-yellow-500';
            case 'success': return 'text-green-500';
            default: return 'text-blue-400';
        }
    };

    return (
        <div className="flex flex-col h-[calc(100vh-8rem)] space-y-4 animate-in fade-in duration-500">

            {/* Status Bar */}
            <div className="flex items-center justify-between p-4 border rounded-lg bg-card shadow-sm">
                <div className="flex items-center space-x-6">
                    <div className="flex items-center space-x-2">
                        <Activity className="h-5 w-5 text-primary" />
                        <span className="font-medium">System Status</span>
                    </div>
                    <div className="flex items-center space-x-2">
                        <Badge variant={isConnected ? "default" : "destructive"} className="transition-colors">
                            {isConnected ? "Online" : "Offline"}
                        </Badge>
                        <span className="text-sm text-muted-foreground">{isConnected ? "Connected to Gateway" : "Disconnected"}</span>
                    </div>
                </div>
                <div className="flex items-center space-x-2">
                    <Button variant="outline" size="sm" onClick={clearLogs}>
                        <Trash2 className="h-4 w-4 mr-2" />
                        Clear
                    </Button>
                    <Button variant="outline" size="sm" onClick={downloadLogs}>
                        <Download className="h-4 w-4 mr-2" />
                        Export
                    </Button>
                </div>
            </div>

            {/* Log Console */}
            <div className="flex-1 border rounded-lg bg-black/90 font-mono text-sm overflow-hidden relative shadow-inner">
                <div className="absolute top-0 left-0 right-0 bg-muted/20 backdrop-blur-sm p-2 border-b border-white/10 flex justify-between px-4 text-xs text-muted-foreground z-10">
                    <span>Console Output</span>
                    <span>{logs.length} events</span>
                </div>
                <ScrollArea className="h-full pt-10 pb-4 px-4 w-full">
                    <div className="space-y-1">
                        {logs.length === 0 && (
                            <div className="text-muted-foreground italic opacity-50 text-center mt-20">No logs generated yet...</div>
                        )}
                        {logs.map((log) => (
                            <div key={log.id} className="grid grid-cols-[80px_60px_1fr] gap-2 hover:bg-white/5 p-0.5 rounded">
                                <span className="text-muted-foreground opacity-70">[{log.timestamp}]</span>
                                <span className={`font-bold ${getLevelColor(log.level)}`}>{log.level.toUpperCase()}</span>
                                <span className="text-gray-300 break-all">{log.message}</span>
                            </div>
                        ))}
                        <div ref={scrollRef} />
                    </div>
                </ScrollArea>
            </div>
        </div>
    );
}
