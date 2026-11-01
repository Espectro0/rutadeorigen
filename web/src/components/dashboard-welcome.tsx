"use client";

import { useTranslations } from "next-intl";
import { useActor } from "@/features/session/session-provider";

export function DashboardWelcome() {
    const t = useTranslations("Dashboard");
    const actor = useActor();

    const name = actor.firstName.trim();

    return (
        <div>
            <h1 className="text-2xl font-semibold tracking-tight">
                {name
                    ? t("welcome", { name })
                    : t("welcomeFallback")}
            </h1>

            <p className="text-ink-dim">
                {t("subtitle")}
            </p>
        </div>
    );
}