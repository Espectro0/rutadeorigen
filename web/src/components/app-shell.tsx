"use client";

import { useState, type ReactNode } from "react";
import { AppHeader } from "@/components/app-header";
import { AppSidebar } from "@/components/app-sidebar";

export function AppShell({ children }: { children: ReactNode }) {
    const [open, setOpen] = useState(false);

    return (
        <div className="flex min-h-screen">
            <AppSidebar open={open} onClose={() => setOpen(false)} />
            <div className="flex min-w-0 flex-1 flex-col">
                <AppHeader onOpenMenu={() => setOpen(true)} menuOpen={open} />
                <main className="flex-1 p-6">{children}</main>
            </div>
        </div>
    );
}