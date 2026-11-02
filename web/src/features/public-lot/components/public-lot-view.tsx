import type { ReactNode } from "react";
import { getFormatter, getTranslations } from "next-intl/server";
import { LotStatusBadge } from "@/features/lots/components/lot-status-badge";
import { EvidenceGrid } from "@/features/evidences/components/evidence-grid";
import type { EvidenceTile } from "@/features/evidences/types";
import { StageProgress } from "@/features/lots/components/stage-progress";
import type { PublicLot } from "../types";
import { UpdatedAt } from "./updated-at";

function Field({ label, children }: Readonly<{ label: string; children: ReactNode }>) {
    return (
        <div className="flex flex-col gap-0.5">
            <dt className="text-sm text-ink-dim">{label}</dt>
            <dd className="break-words font-medium">{children}</dd>
        </div>
    );
}

function toTile(evidence: PublicLot["evidences"][number]): EvidenceTile {
    return {
        id: evidence.id,
        stage: evidence.stage,
        state: "ready",
        fileName: "",
        thumbnailUrl: evidence.thumbnailUrl,
        imageUrl: evidence.imageUrl,
        rejectionReason: "",
    };
}

export async function PublicLotView({ lot }: Readonly<{ lot: PublicLot }>) {
    const t = await getTranslations("Public");
    const lots = await getTranslations("Lots");
    const format = await getFormatter();

    const voided = lot.status === "voided";
    const inProcess = !voided && lot.stage !== "packaging";
    const harvest = format.dateTime(new Date(`${lot.harvestDate}T00:00:00Z`), {
        dateStyle: "medium",
        timeZone: "UTC",
    });

    return (
        <div className="mx-auto flex w-full max-w-2xl flex-col gap-6">
            <div className="flex flex-wrap items-center gap-3">
                <h1 className="font-display text-3xl font-semibold tracking-tight">{lot.code}</h1>
                <LotStatusBadge status={lot.status} label={lots(`status.${lot.status}`)} />
            </div>

            {voided && (
                <p role="alert" className="rounded-lg bg-terracotta/15 p-4 font-medium text-terracotta">
                    {t("voided")}
                </p>
            )}

            {!voided && <StageProgress stage={lot.stage} />}

            {inProcess && (
                <p role="status" className="rounded-lg bg-ground-raised p-4 text-sm text-ink-dim">
                    {t("inProcess")}
                </p>
            )}

            <section aria-label={t("origin")} className="card p-5">
                <h2 className="mb-4 text-lg font-semibold">{t("origin")}</h2>
                <dl className="grid gap-4 sm:grid-cols-3">
                    <Field label={t("farm")}>{lot.farmName}</Field>
                    <Field label={t("altitude")}>{t("altitudeValue", { value: lot.farmAltitude })}</Field>
                    <Field label={t("producer")}>{lot.producerName}</Field>
                </dl>
            </section>

            <section aria-label={t("coffee")} className="card p-5">
                <h2 className="mb-4 text-lg font-semibold">{t("coffee")}</h2>
                <dl className="grid gap-4 sm:grid-cols-3">
                    <Field label={t("variety")}>{lot.variety}</Field>
                    <Field label={t("process")}>{lots(`process.${lot.process}`)}</Field>
                    <Field label={t("harvest")}>{harvest}</Field>
                </dl>
            </section>

            {!voided && lot.evidences.length > 0 && (
                <section aria-label={t("photos")} className="card p-5">
                    <h2 className="mb-4 text-lg font-semibold">{t("photos")}</h2>
                    <EvidenceGrid tiles={lot.evidences.map(toTile)} />
                </section>
            )}

            <UpdatedAt iso={lot.updatedAt} />
        </div>
    );
}
