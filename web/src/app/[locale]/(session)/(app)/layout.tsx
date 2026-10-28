import type { ReactNode } from "react";
import { AppHeader } from "@/components/app-header";
import { RequireSession } from "@/features/session/require-session";

export default function AppLayout({ children }: { children: ReactNode }) {
    return (
        <RequireSession expected="authenticated">
            <AppHeader />
            {children}
        </RequireSession>
    );
}