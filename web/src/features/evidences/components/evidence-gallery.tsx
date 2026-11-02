"use client";

import { useEffect, useState } from "react";
import { useTranslations } from "next-intl";
import { useDescribeError } from "@/lib/use-describe-error";
import { listEvidences } from "../api";
import { nextRefreshDelay, toTile, type Evidence } from "../types";
import { EvidenceGrid } from "./evidence-grid";

type Props = Readonly<{ lotId: string; refreshToken: number }>;

export function EvidenceGallery({ lotId, refreshToken }: Props) {
    const t = useTranslations("Evidences");
    const common = useTranslations("Common");
    const describe = useDescribeError();
    const [items, setItems] = useState<readonly Evidence[] | null>(null);
    const [failure, setFailure] = useState<Readonly<{ cause: unknown }> | null>(null);
    const [tick, setTick] = useState(0);

    useEffect(() => {
        let active = true;
        listEvidences(lotId)
            .then((loaded) => {
                if (!active) return;
                setItems(loaded);
                setFailure(null);
            })
            .catch((cause: unknown) => {
                if (active) setFailure({ cause });
            });
        return () => {
            active = false;
        };
    }, [lotId, refreshToken, tick]);

    useEffect(() => {
        if (items === null) return;
        const delay = nextRefreshDelay(items, Date.now());
        if (delay === null) return;
        const timer = setTimeout(() => setTick((value) => value + 1), delay);
        return () => clearTimeout(timer);
    }, [items]);

    return (
        <section aria-label={t("title")} className="card flex flex-col gap-4 p-5">
            <h2 className="text-lg font-semibold">{t("title")}</h2>

            {items === null && failure === null && <p className="text-sm text-ink-dim">{common("loading")}</p>}

            {failure !== null && (
                <div role="alert" className="flex items-center gap-3 text-sm text-red-600">
                    <span>{describe(failure.cause)}</span>
                    <button type="button" onClick={() => setTick((value) => value + 1)} className="btn-secondary">
                        {common("retry")}
                    </button>
                </div>
            )}

            {items !== null && items.length === 0 && <p className="text-sm text-ink-dim">{t("empty")}</p>}

            {items !== null && items.length > 0 && <EvidenceGrid tiles={items.map(toTile)} />}
        </section>
    );
}
