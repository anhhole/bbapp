import { LucideIcon } from "lucide-react";
import { Button } from "@/components/ui/button";

interface ModeCardProps {
    title: string;
    description: string;
    icon: LucideIcon;
    color: string;
    onClick: () => void;
}

export function ModeCard({ title, description, icon: Icon, color, onClick }: ModeCardProps) {
    return (
        <div
            onClick={onClick}
            className="group relative overflow-hidden rounded-xl border bg-card text-card-foreground shadow cursor-pointer transition-all hover:shadow-lg hover:-translate-y-1 glass-card"
        >
            <div className={`absolute inset-0 opacity-0 group-hover:opacity-5 transition-opacity bg-${color}-500`} />
            <div className="p-6 flex flex-col items-center text-center space-y-4">
                <div className={`p-4 rounded-full bg-${color}-100/50 dark:bg-${color}-900/20 text-${color}-600 dark:text-${color}-400 mb-2 transition-transform group-hover:scale-110`}>
                    <Icon className="h-8 w-8" />
                </div>
                <div>
                    <h3 className="font-semibold text-lg tracking-tight mb-1">{title}</h3>
                    <p className="text-sm text-muted-foreground">{description}</p>
                </div>
                <Button variant="ghost" className="w-full mt-4 group-hover:bg-background/50">
                    Select Mode →
                </Button>
            </div>
        </div>
    );
}
