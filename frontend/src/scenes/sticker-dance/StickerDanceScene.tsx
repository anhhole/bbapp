import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Sticker } from "lucide-react";

export function StickerDanceScene() {
    return (
        <div className="container mx-auto p-8 flex flex-col items-center justify-center h-[calc(100vh-8rem)] animate-in fade-in duration-500">
            <Card className="w-full max-w-md text-center glass-card">
                <CardHeader>
                    <div className="mx-auto bg-pink-100 dark:bg-pink-900/20 p-4 rounded-full mb-4 w-fit">
                        <Sticker className="h-12 w-12 text-pink-500" />
                    </div>
                    <CardTitle className="text-2xl">Sticker Dance Mode</CardTitle>
                    <CardDescription>
                        Interactive dance challenges triggered by audience stickers.
                    </CardDescription>
                </CardHeader>
                <CardContent>
                    <p className="text-muted-foreground mb-6">
                        This feature is currently under development. Stay tuned for updates!
                    </p>
                    <Button variant="outline" disabled>
                        Launch Demo (Coming Soon)
                    </Button>
                </CardContent>
            </Card>
        </div>
    );
}
