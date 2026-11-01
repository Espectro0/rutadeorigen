"use client";

import { useEffect, useId, useRef, useState } from "react";
import { useTranslations } from "next-intl";
import { LuChevronsUpDown, LuLogOut } from "react-icons/lu";
import { shortName } from "@/features/session/display-name";
import { useActor } from "@/features/session/session-provider";

export function UserMenu() {
    const t = useTranslations("Nav");
    const header = useTranslations("Header");
    const actor = useActor();
    const [open, setOpen] = useState(false);
    const rootRef = useRef<HTMLDivElement>(null);
    const panelId = useId();
    const initial = actor.firstName.charAt(0).toUpperCase() || "?";

    useEffect(() => {
        if (!open) return;

        function onPointerDown(event: PointerEvent) {
            if (!rootRef.current?.contains(event.target as Node)) setOpen(false);
        }
        function onKeyDown(event: KeyboardEvent) {
            if (event.key === "Escape") setOpen(false);
        }

        document.addEventListener("pointerdown", onPointerDown);
        document.addEventListener("keydown", onKeyDown);
        return () => {
            document.removeEventListener("pointerdown", onPointerDown);
            document.removeEventListener("keydown", onKeyDown);
        };
    }, [open]);

    return (
        <div ref={rootRef} className="relative">
            {open && (
                <div
                    id={panelId}
                    className="card absolute bottom-full left-0 right-0 mb-2 p-1 shadow-lg"
                >
                    <a
                        href="/auth/logout"
                        className="flex items-center gap-2 rounded px-3 py-2 text-sm text-ink hover:bg-ground-raised"
                    >
                        <LuLogOut className="size-4" aria-hidden />
                        {header("logout")}
                    </a>
                </div>
            )}

            <button
                type="button"
                onClick={() => setOpen((value) => !value)}
                aria-expanded={open}
                aria-controls={open ? panelId : undefined}
                aria-label={t("userMenu")}
                className="flex w-full items-center gap-3 rounded-md p-2 text-left hover:bg-ground"
            >
                <span
                    aria-hidden
                    className="flex size-9 shrink-0 items-center justify-center rounded-full bg-terracotta text-sm font-semibold text-on-accent"
                >
                    {initial}
                </span>
                <span className="min-w-0 flex-1 leading-tight">
                    <span className="block truncate text-sm font-medium text-ink">{shortName(actor)}</span>
                    <span className="block truncate text-xs text-ink-dim">{actor.name}</span>
                </span>
                <LuChevronsUpDown className="size-4 shrink-0 text-ink-dim" aria-hidden />
            </button>
        </div>
    );
}