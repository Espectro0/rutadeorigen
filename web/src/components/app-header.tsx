"use client";

import { useTranslations } from "next-intl";
import { LuMenu } from "react-icons/lu";
import { LocaleSwitcher } from "@/components/locale-switcher";

type Props = Readonly<{ onOpenMenu: () => void; menuOpen: boolean }>;

export function AppHeader({ onOpenMenu, menuOpen }: Props) {
    const t = useTranslations("Nav");

    return (
        <header className="flex items-center justify-between border-b border-hairline bg-ground-deep px-4 py-3 md:justify-end md:px-6">
            <button
                type="button"
                onClick={onOpenMenu}
                aria-label={t("openMenu")}
                aria-expanded={menuOpen}
                aria-controls="app-sidebar"
                className="rounded p-1.5 text-ink hover:bg-ground-raised md:hidden"
            >
                <LuMenu className="size-6" aria-hidden />
            </button>
            <LocaleSwitcher />
        </header>
    );
}