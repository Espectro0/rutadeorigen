import type { ReactNode } from "react";
import { AppShell } from "@/components/app-shell";
import { RequireSession } from "@/features/session/require-session";

export default function AppLayout({ children }: { children: ReactNode }) {
    return (
        <RequireSession expected="authenticated">
            <AppShell>{children}</AppShell>
        </RequireSession>
    );
}
