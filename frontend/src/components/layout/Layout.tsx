import { Sidebar } from "./Sidebar";

interface LayoutProps {
    children: React.ReactNode;
    activeTab: string;
    onTabChange: (tab: string) => void;
}

export function Layout({ children, activeTab, onTabChange }: LayoutProps) {
    return (
        <div className="flex h-screen overflow-hidden bg-background">
            <Sidebar activeTab={activeTab} onTabChange={onTabChange} />
            <main className="flex-1 overflow-y-auto p-8 relative">
                <div className="absolute inset-0 bg-gradient-to-br from-blue-500/5 to-purple-500/5 pointer-events-none" />
                <div className="relative z-10">
                    {children}
                </div>
            </main>
        </div>
    );
}
