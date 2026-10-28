import type { ReactNode } from "react";
import { SessionProvider } from "@/features/session/session-provider";

export default function SessionLayout({ children }: { children: ReactNode }) {
    return <SessionProvider>{children}</SessionProvider>;
}