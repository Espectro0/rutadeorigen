"use client";

import { useTranslations } from "next-intl";
import { shortName } from "@/features/session/display-name";
import { useActor } from "@/features/session/session-provider";
import { LocaleSwitcher } from "@/components/locale-switcher";

export function AppHeader() {
    const t = useTranslations("Header");
    const actor = useActor();
    const initial = actor.firstName?.charAt(0).toUpperCase() ?? "?";

    return (
        <header className="flex items-center justify-end gap-4 bg-white px-6 py-3 shadow-sm dark:bg-zinc-900">
            <LocaleSwitcher />
            <div className="flex items-center gap-3">
                <span
                    aria-hidden
                    className="flex size-9 items-center justify-center rounded-full bg-blue-500 text-sm font-semibold text-white"
                >
                    {initial}
                </span>
                <div className="leading-tight">
                    <p className="text-sm font-medium">{shortName(actor)}</p>
                    <p className="text-xs text-zinc-500">{actor.name}</p>
                </div>
            </div>
            <a href="/auth/logout" className="rounded border px-3 py-1.5 text-sm">
                {t("logout")}
            </a>
        </header>
    );
}