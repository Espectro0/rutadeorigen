"use client";

import { useTranslations } from "next-intl";
import { LuLayoutDashboard, LuMapPin, LuPackage } from "react-icons/lu";
import { Link, usePathname } from "@/i18n/navigation";

const ITEMS = [
    { href: "/dashboard", key: "dashboard", Icon: LuLayoutDashboard },
    { href: "/farms", key: "farms", Icon: LuMapPin },
    { href: "/lots", key: "lots", Icon: LuPackage },
] as const;

export function AppSidebar() {
    const t = useTranslations("Nav");
    const pathname = usePathname();

    return (
        <aside className="hidden w-60 shrink-0 flex-col bg-slate-900 text-slate-300 md:flex">
            <div className="px-6 py-5 text-lg font-semibold tracking-tight text-white">
                {t("brand")}
            </div>
            <nav className="flex flex-col gap-1 px-3">
                {ITEMS.map(({ href, key, Icon }) => {
                    const active = pathname === href || pathname.startsWith(`${href}/`);
                    return (
                        <Link
                            key={href}
                            href={href}
                            aria-current={active ? "page" : undefined}
                            className={`flex items-center gap-3 rounded-md px-3 py-2.5 text-sm transition-colors ${
                                active
                                    ? "bg-blue-500 text-white"
                                    : "hover:bg-slate-800 hover:text-white"
                            }`}
                        >
                            <Icon className="size-5" aria-hidden />
                            {t(key)}
                        </Link>
                    );
                })}
            </nav>
        </aside>
    );
}