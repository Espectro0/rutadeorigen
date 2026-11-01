"use client";

import { useTranslations } from "next-intl";
import type { LotAction } from "../types";

const ORDER: readonly LotAction[] = ["lot:register_stage", "lot:transfer", "qr:generate", "lot:void"];

const LABEL_KEY = {
    "lot:register_stage": "registerStage",
    "lot:transfer": "transfer",
    "qr:generate": "generateQr",
    "lot:void": "void",
} as const satisfies Record<LotAction, string>;

type Props = Readonly<{
    allowed: readonly LotAction[];
    handlers?: Partial<Record<LotAction, () => void>>;
}>;

export function LotActions({ allowed, handlers = {} }: Props) {
    const t = useTranslations("Lots.actions");
    const visible = ORDER.filter((action) => allowed.includes(action));

    if (visible.length === 0) return null;

    return (
        <section aria-label={t("title")} className="flex flex-wrap items-center gap-3">
            {visible.map((action) => {
                const onClick = handlers[action];
                const danger = action === "lot:void";
                return (
                    <button
                        key={action}
                        type="button"
                        onClick={onClick}
                        disabled={onClick === undefined}
                        title={onClick === undefined ? t("soon") : undefined}
                        className={`${danger ? "btn-secondary text-red-600" : "btn-primary"} disabled:cursor-not-allowed disabled:opacity-60`}
                    >
                        {t(LABEL_KEY[action])}
                    </button>
                );
            })}
        </section>
    );
}
