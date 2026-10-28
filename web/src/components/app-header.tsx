"use client";

import { useTranslations } from "next-intl";
import { Link } from "@/i18n/navigation";
import { shortName } from "@/features/session/display-name";
import { useActor } from "@/features/session/session-provider";
import {LocaleSwitcher} from "@/components/locale-switcher";

export function AppHeader() {
    const t = useTranslations("Header");
    const actor = useActor();

    return (
        <header className="flex items-center justify-between border-b px-6 py-3">
            <Link href="/dashboard" className="text-lg font-semibold tracking-tight">
                {t("brand")}
            </Link>

            <div className="flex items-center gap-4">
                <div className="text-right leading-tight">
                    <p className="font-medium">{shortName(actor)}</p>
                    <p className="text-sm text-zinc-500">{actor.name}</p>
                </div>
                <a href="/auth/logout" className="rounded border px-3 py-1.5 text-sm">
                    {t("logout")}
                </a>
                <LocaleSwitcher />
            </div>
        </header>
    );
}