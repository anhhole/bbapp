import { LayoutDashboard, Settings, Activity, FileBarChart } from "lucide-react";
import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";

interface SidebarProps extends React.HTMLAttributes<HTMLDivElement> {
    activeTab: string;
    onTabChange: (tab: string) => void;
}

export function Sidebar({ className, activeTab, onTabChange }: SidebarProps) {
    return (
        <div className={cn("pb-12 w-64 glass border-r bg-background/50", className)}>
            <div className="space-y-4 py-4">
                <div className="px-4 py-2">
                    <h2 className="mb-2 px-2 text-lg font-semibold tracking-tight text-foreground">
                        BBapp Studio
                    </h2>
                    <div className="space-y-1">
                        <Button
                            variant={activeTab === "dashboard" ? "secondary" : "ghost"}
                            className="w-full justify-start"
                            onClick={() => onTabChange("dashboard")}
                        >
                            <LayoutDashboard className="mr-2 h-4 w-4" />
                            Home
                        </Button>
                        <Button
                            variant={activeTab === "pk-mode" ? "secondary" : "ghost"}
                            className="w-full justify-start"
                            onClick={() => onTabChange("pk-mode")}
                        >
                            <Activity className="mr-2 h-4 w-4" />
                            PK Mode
                        </Button>
                        <Button
                            variant={activeTab === "monitor" ? "secondary" : "ghost"}
                            className="w-full justify-start"
                            onClick={() => onTabChange("monitor")}
                        >
                            <FileBarChart className="mr-2 h-4 w-4" />
                            Monitor
                        </Button>
                        <Button
                            variant={activeTab === "configuration" ? "secondary" : "ghost"}
                            className="w-full justify-start"
                            onClick={() => onTabChange("configuration")}
                        >
                            <Settings className="mr-2 h-4 w-4" />
                            Configuration
                        </Button>
                    </div>
                </div>
            </div>
        </div>
    );
}
