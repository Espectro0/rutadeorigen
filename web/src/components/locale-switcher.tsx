"use client";

import { useLocale, useTranslations } from "next-intl";
import { useTransition } from "react";
import { usePathname, useRouter } from "@/i18n/navigation";
import { routing } from "@/i18n/routing";
import { LuLanguages } from "react-icons/lu";

type AppLocale = (typeof routing.locales)[number];

function nextLocale(current: string): AppLocale {
    const index = routing.locales.findIndex((code) => code === current);
    return routing.locales[(index + 1) % routing.locales.length];
}

export function LocaleSwitcher() {
    const t = useTranslations("Header");
    const locale = useLocale();
    const pathname = usePathname();
    const router = useRouter();
    const [pending, startTransition] = useTransition();

    const target = nextLocale(locale);

    function change() {
        startTransition(() => {
            router.replace(pathname, { locale: target });
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
            <LuLanguages  />
            <span>{locale.toUpperCase()}</span>
        </button>
    );
}