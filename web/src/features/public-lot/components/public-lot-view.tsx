import type { ReactNode } from "react";
import { getFormatter, getTranslations } from "next-intl/server";
import { LotStatusBadge } from "@/features/lots/components/lot-status-badge";
import { EvidenceGrid } from "@/features/evidences/components/evidence-grid";
import type { EvidenceTile } from "@/features/evidences/types";
import { StageProgress } from "@/features/lots/components/stage-progress";
import { isDetailOption } from "@/features/lots/types";
import type { PublicLot, PublicStage } from "../types";
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

type StageRow = Readonly<{ key: string; label: string; value: string }>;

export async function PublicLotView({ lot }: Readonly<{ lot: PublicLot }>) {
    const t = await getTranslations("Public");
    const lots = await getTranslations("Lots");
    const options = await getTranslations("Lots.stageForm");
    const format = await getFormatter();

    const formatDay = (value: string) =>
        format.dateTime(new Date(`${value}T00:00:00Z`), { dateStyle: "medium", timeZone: "UTC" });
    const optionText = (value: string) => (isDetailOption(value) ? options(`options.${value}`) : value);

    const stageRows = (stage: PublicStage): readonly StageRow[] => {
        const rows: StageRow[] = [];
        if (stage.dryingMethod !== "") {
            rows.push({ key: "dryingMethod", label: t("details.dryingMethod"), value: optionText(stage.dryingMethod) });
        }
        if (stage.dryingDays > 0) {
            rows.push({
                key: "dryingDays",
                label: t("details.dryingDays"),
                value: t("details.daysValue", { value: stage.dryingDays }),
            });
        }
        if (stage.roastLevel !== "") {
            rows.push({ key: "roastLevel", label: t("details.roastLevel"), value: optionText(stage.roastLevel) });
        }
        if (stage.roastDurationMinutes > 0) {
            rows.push({
                key: "roastDuration",
                label: t("details.roastDuration"),
                value: t("details.minutesValue", { value: stage.roastDurationMinutes }),
            });
        }
        if (stage.grind !== "") {
            rows.push({ key: "grind", label: t("details.grind"), value: optionText(stage.grind) });
        }
        if (stage.packageWeightGrams > 0) {
            rows.push({
                key: "packageWeight",
                label: t("details.packageWeight"),
                value: t("details.gramsValue", { value: stage.packageWeightGrams }),
            });
        }
        if (stage.bestBefore !== "") {
            rows.push({ key: "bestBefore", label: t("details.bestBefore"), value: formatDay(stage.bestBefore) });
        }
        return rows;
    };

    const voided = lot.status === "voided";
    const inProcess = !voided && lot.stage !== "packaging";
    const harvest = formatDay(lot.harvestDate);

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

            {!voided && lot.stages.length > 0 && (
                <section aria-label={t("journey")} className="card p-5">
                    <h2 className="mb-4 text-lg font-semibold">{t("journey")}</h2>
                    <ol className="flex flex-col gap-5">
                        {lot.stages.map((stage) => (
                            <li key={stage.stage} className="flex flex-col gap-3">
                                <div className="flex flex-wrap items-baseline gap-x-3 gap-y-1">
                                    <h3 className="font-semibold">{lots(`stage.${stage.stage}`)}</h3>
                                    <span className="text-sm text-ink-dim">{formatDay(stage.eventDate)}</span>
                                </div>
                                <dl className="grid gap-4 sm:grid-cols-3">
                                    {stageRows(stage).map((row) => (
                                        <Field key={row.key} label={row.label}>
                                            {row.value}
                                        </Field>
                                    ))}
                                </dl>
                            </li>
                        ))}
                    </ol>
                </section>
            )}

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
