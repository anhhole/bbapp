import { useState } from 'react';
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Link } from "lucide-react";

interface QuickConnectProps {
    onConnect: (roomId: string) => void;
}

export function QuickConnect({ onConnect }: QuickConnectProps) {
    const [roomId, setRoomId] = useState('');

    const handleSubmit = (e: React.FormEvent) => {
        e.preventDefault();
        if (roomId.trim()) {
            onConnect(roomId.trim());
        }
    };

    return (
        <div className="rounded-xl border bg-card text-card-foreground shadow p-6 glass-card relative overflow-hidden">
            <div className="absolute top-0 right-0 p-4 opacity-10">
                <Link className="h-24 w-24" />
            </div>
            <div className="flex flex-col space-y-1.5 mb-4 relative z-10">
                <h3 className="font-semibold text-lg leading-none tracking-tight">Quick Connect</h3>
                <p className="text-sm text-muted-foreground">Enter a Room ID to jump straight into a session.</p>
            </div>
            <form onSubmit={handleSubmit} className="flex w-full max-w-sm items-center space-x-2 relative z-10">
                <Input
                    type="text"
                    placeholder="Enter Room ID..."
                    value={roomId}
                    onChange={(e) => setRoomId(e.target.value)}
                    className="bg-background/50"
                />
                <Button type="submit" className="bg-primary hover:bg-primary/90 text-white font-medium shadow-md transition-all hover:scale-105 active:scale-95">
                    Connect
                </Button>
            </form>
        </div>
    );
}
