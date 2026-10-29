import type { ReactNode } from "react";
import { AppHeader } from "@/components/app-header";
import { AppSidebar } from "@/components/app-sidebar";
import { RequireSession } from "@/features/session/require-session";

export default function AppLayout({ children }: { children: ReactNode }) {
    return (
        <RequireSession expected="authenticated">
            <div className="flex min-h-screen">
                <AppSidebar />
                <div className="flex min-w-0 flex-1 flex-col bg-zinc-100 dark:bg-zinc-950">
                    <AppHeader />
                    <main className="flex-1 p-6">{children}</main>
                </div>
            </div>
        </RequireSession>
    );
}