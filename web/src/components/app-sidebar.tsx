"use client";

import { useEffect } from "react";
import { useTranslations } from "next-intl";
import { LuLayoutDashboard, LuMapPin, LuPackage } from "react-icons/lu";
import Image from "next/image";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { UserMenu } from "@/components/user-menu";

const ITEMS = [
    { href: "/dashboard", key: "dashboard", Icon: LuLayoutDashboard },
    { href: "/farms", key: "farms", Icon: LuMapPin },
    { href: "/lots", key: "lots", Icon: LuPackage },
] as const;

type Props = Readonly<{ open: boolean; onClose: () => void }>;

export function AppSidebar({ open, onClose }: Props) {
    const t = useTranslations("Nav");
    const pathname = usePathname();

    useEffect(() => {
        if (!open) return;
        function onKeyDown(event: KeyboardEvent) {
            if (event.key === "Escape") onClose();
        }
        document.addEventListener("keydown", onKeyDown);
        return () => document.removeEventListener("keydown", onKeyDown);
    }, [open, onClose]);

    return (
        <>
            {open && (
                <div
                    aria-hidden
                    onClick={onClose}
                    className="fixed inset-0 z-30 bg-black/40 md:hidden"
                />
            )}

            <aside
                id="app-sidebar"
                className={`fixed inset-y-0 left-0 z-40 flex w-60 shrink-0 flex-col border-r border-hairline bg-ground-raised text-ink-dim transition-[transform,visibility] duration-200 md:sticky md:top-0 md:h-screen md:translate-x-0 md:visible ${
                    open ? "translate-x-0" : "invisible -translate-x-full"
                }`}
            >
                <div className="flex items-center gap-3 px-6 py-5 text-lg font-semibold tracking-tight text-ink">
                    <Image src="/logo-with-bg.svg" alt="Logo" width={35} height={35} priority />
                    {t("brand")}
                </div>

                <nav className="flex flex-col gap-1 px-3">
                    {ITEMS.map(({ href, key, Icon }) => {
                        const active = pathname === href || pathname.startsWith(`${href}/`);
                        return (
                            <Link
                                key={href}
                                href={href}
                                onClick={onClose}
                                aria-current={active ? "page" : undefined}
                                className={`flex items-center gap-3 rounded-md px-3 py-2.5 text-sm transition-colors ${
                                    active
                                        ? "bg-terracotta text-on-accent"
                                        : "hover:bg-ground hover:text-ink"
                                }`}
                            >
                                <Icon className="size-5" aria-hidden />
                                {t(key)}
                            </Link>
                        );
                    })}
                </nav>

                <div className="mt-auto border-t border-hairline p-3">
                    <UserMenu />
                </div>
            </aside>
        </>
    );
}