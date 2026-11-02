"use client";

import { useEffect, useRef } from "react";
import { useTranslations } from "next-intl";
import type { EvidenceTile } from "../types";

type Props = Readonly<{
    tiles: readonly EvidenceTile[];
    index: number;
    onChange: (index: number) => void;
    onClose: () => void;
}>;

export function EvidenceViewer({ tiles, index, onChange, onClose }: Props) {
    const t = useTranslations("Evidences.viewer");
    const stages = useTranslations("Lots.stage");
    const closeButton = useRef<HTMLButtonElement>(null);
    const tile = tiles[index];
    const hasPrevious = index > 0;
    const hasNext = index < tiles.length - 1;

    useEffect(() => {
        closeButton.current?.focus();
    }, []);

    useEffect(() => {
        function onKeyDown(event: KeyboardEvent) {
            if (event.key === "Escape") onClose();
            if (event.key === "ArrowLeft" && hasPrevious) onChange(index - 1);
            if (event.key === "ArrowRight" && hasNext) onChange(index + 1);
        }
        window.addEventListener("keydown", onKeyDown);
        return () => window.removeEventListener("keydown", onKeyDown);
    }, [index, hasPrevious, hasNext, onChange, onClose]);

    if (tile === undefined) return null;

    return (
        <div
            role="dialog"
            aria-modal="true"
            aria-label={t("title")}
            className="fixed inset-0 z-50 flex items-center justify-center bg-black/80 p-4"
            onClick={onClose}
        >
            <div className="flex max-h-full w-full max-w-4xl flex-col gap-3" onClick={(event) => event.stopPropagation()}>
                <div className="flex items-center justify-between gap-3 text-sm text-white">
                    <span>{t("position", { current: index + 1, total: tiles.length, stage: stages(tile.stage) })}</span>
                    <button ref={closeButton} type="button" onClick={onClose} className="btn-secondary">
                        {t("close")}
                    </button>
                </div>

                <img
                    src={tile.imageUrl}
                    alt={t("alt", { stage: stages(tile.stage) })}
                    className="max-h-[75vh] w-full rounded-lg bg-black object-contain"
                />

                <div className="flex justify-between gap-3">
                    <button type="button" disabled={!hasPrevious} onClick={() => onChange(index - 1)} className="btn-secondary">
                        {t("previous")}
                    </button>
                    <button type="button" disabled={!hasNext} onClick={() => onChange(index + 1)} className="btn-secondary">
                        {t("next")}
                    </button>
                </div>
            </div>
        </div>
    );
}
