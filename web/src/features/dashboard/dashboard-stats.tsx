"use client";

import { useEffect, useState } from "react";
import { useTranslations } from "next-intl";
import { request } from "@/lib/http-client";

type FarmList = Readonly<{ items: unknown[] }>;
type LotItem = Readonly<{ id: string; stage: string }>;
type LotPage = Readonly<{ items: LotItem[]; next_cursor?: string }>;

const PAGE_SIZE = 100;
const MAX_PAGES = 10;

async function loadAllLots(): Promise<LotItem[]> {
    const lots: LotItem[] = [];
    let cursor = "";
    for (let page = 0; page < MAX_PAGES; page++) {
        const query = new URLSearchParams({ limit: String(PAGE_SIZE) });
        if (cursor) query.set("cursor", cursor);
        const result = await request<LotPage>(`/lots?${query}`);
        lots.push(...result.items);
        if (!result.next_cursor) break;
        cursor = result.next_cursor;
    }
    return lots;
}

export function DashboardStats() {
    const t = useTranslations("Dashboard");
    const [farms, setFarms] = useState<number | null>(null);
    const [lots, setLots] = useState<number | null>(null);

    useEffect(() => {
        request<FarmList>("/farms")
            .then((result) => setFarms(result.items.length))
            .catch(() => setFarms(0));
        loadAllLots()
            .then((all) => setLots(all.length))
            .catch(() => setLots(0));
    }, []);

    const cards = [
        { label: t("cards.farms"), value: farms },
        { label: t("cards.lots"), value: lots },
    ];

    return (
        <section className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
            {cards.map((card) => (
                <div key={card.label} className="rounded-lg bg-white p-5 shadow-sm dark:bg-zinc-900">
                    <p className="text-sm text-zinc-500">{card.label}</p>
                    <p className="mt-2 text-3xl font-semibold tabular-nums">{card.value ?? "…"}</p>
                </div>
            ))}
        </section>
    );
}