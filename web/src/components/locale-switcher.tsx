"use client";

import { useLocale, useTranslations } from "next-intl";
import { useTransition } from "react";
import { LuLanguages } from "react-icons/lu";
import { setLocale } from "@/i18n/actions";
import { nextLocale } from "@/i18n/config";

export function LocaleSwitcher() {
    const t = useTranslations("Header");
    const locale = useLocale();
    const [pending, startTransition] = useTransition();

    const target = nextLocale(locale);

    function change() {
        startTransition(async () => {
            await setLocale(target);
        });
    }

    return (
        <button
            type="button"
            onClick={change}
            disabled={pending}
            lang={target}
            aria-label={t("changeLanguage", { language: t(`languages.${target}`) })}
            title={t("changeLanguage", { language: t(`languages.${target}`) })}
            className="flex items-center gap-1.5 font-mono text-sm text-ink-dim hover:text-terracotta disabled:opacity-60"
        >
            <LuLanguages />
            <span>{locale.toUpperCase()}</span>
        </button>
    );
}
