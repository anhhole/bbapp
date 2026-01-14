import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Pickaxe } from "lucide-react";

export function FreeModeScene() {
    return (
        <div className="flex flex-col items-center justify-center h-full p-8 space-y-8">
            <div className="text-center space-y-4 max-w-2xl">
                <div className="flex justify-center mb-6">
                    <div className="bg-primary/10 p-6 rounded-full">
                        <Pickaxe className="w-16 h-16 text-primary" />
                    </div>
                </div>
                <h2 className="text-3xl font-bold tracking-tight">Free Mode</h2>
                <p className="text-muted-foreground text-lg">
                    This mode allows for unrestricted control and custom configurations without the constraints of a battle or dance session.
                </p>
            </div>

            <Card className="w-full max-w-md border-dashed border-2">
                <CardHeader>
                    <CardTitle>Coming Soon</CardTitle>
                    <CardDescription>This feature is currently under development.</CardDescription>
                </CardHeader>
                <CardContent className="space-y-4">
                    <p className="text-sm text-muted-foreground">
                        In Free Mode, you will be able to:
                    </p>
                    <ul className="list-disc pl-5 text-sm text-muted-foreground space-y-2">
                        <li>Manually trigger animations</li>
                        <li>Test gift bindings in real-time</li>
                        <li>Configure custom overlays on the fly</li>
                    </ul>
                    <Button disabled className="w-full mt-4">Launch Free Mode (Soon)</Button>
                </CardContent>
            </Card>
        </div>
    );
}
