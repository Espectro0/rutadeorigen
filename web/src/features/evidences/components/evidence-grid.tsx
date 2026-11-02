"use client";

import { useState } from "react";
import { useTranslations } from "next-intl";
import { LOT_STAGES, type LotStage } from "@/features/lots/types";
import type { EvidenceTile } from "../types";
import { EvidenceViewer } from "./evidence-viewer";

const STAGES_IN_ORDER = LOT_STAGES.filter((stage) => stage !== "none");

type Props = Readonly<{ tiles: readonly EvidenceTile[] }>;

export function EvidenceGrid({ tiles }: Props) {
    const t = useTranslations("Evidences");
    const stages = useTranslations("Lots.stage");
    const [openId, setOpenId] = useState<string | null>(null);

    const viewable = tiles.filter((tile) => tile.state === "ready");
    const openIndex = viewable.findIndex((tile) => tile.id === openId);

    function tilesOf(stage: LotStage): readonly EvidenceTile[] {
        return tiles.filter((tile) => tile.stage === stage);
    }

    return (
        <div className="flex flex-col gap-5">
            {STAGES_IN_ORDER.map((stage) => {
                const stageTiles = tilesOf(stage);
                if (stageTiles.length === 0) return null;

                return (
                    <section key={stage} aria-label={stages(stage)} className="flex flex-col gap-2">
                        <h3 className="text-sm font-medium text-ink-dim">{stages(stage)}</h3>
                        <ul className="grid grid-cols-2 gap-3 sm:grid-cols-3 md:grid-cols-4">
                            {stageTiles.map((tile) => (
                                <li key={tile.id}>
                                    {tile.state === "ready" && (
                                        <button
                                            type="button"
                                            onClick={() => setOpenId(tile.id)}
                                            className="block w-full overflow-hidden rounded-lg border border-hairline"
                                        >
                                            <img
                                                src={tile.thumbnailUrl}
                                                alt={t("thumbnailAlt", { stage: stages(stage) })}
                                                loading="lazy"
                                                className="aspect-square w-full object-cover"
                                            />
                                        </button>
                                    )}
                                    {tile.state === "processing" && (
                                        <p
                                            role="status"
                                            className="flex aspect-square items-center justify-center rounded-lg bg-ground-raised p-3 text-center text-sm text-ink-dim"
                                        >
                                            {t("processing")}
                                        </p>
                                    )}
                                    {tile.state === "rejected" && (
                                        <p
                                            role="alert"
                                            className="flex aspect-square flex-col items-center justify-center gap-1 rounded-lg bg-terracotta/15 p-3 text-center text-sm text-terracotta"
                                        >
                                            <span className="font-medium">{t("rejected")}</span>
                                            <span>{tile.rejectionReason}</span>
                                        </p>
                                    )}
                                </li>
                            ))}
                        </ul>
                    </section>
                );
            })}

            {openIndex >= 0 && (
                <EvidenceViewer
                    tiles={viewable}
                    index={openIndex}
                    onChange={(next) => setOpenId(viewable[next]?.id ?? null)}
                    onClose={() => setOpenId(null)}
                />
            )}
        </div>
    );
}
