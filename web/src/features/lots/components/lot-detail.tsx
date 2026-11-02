"use client";

import { useState, type ReactNode } from "react";
import { useTranslations } from "next-intl";
import { listFarms } from "@/features/farms/api";
import Link from "next/link";
import { isApiError } from "@/lib/api-error";
import { useDateFormat } from "@/lib/use-date-format";
import { useDescribeError } from "@/lib/use-describe-error";
import { useRemote } from "@/lib/use-remote";
import { getLot } from "../api";
import { transferTargetFor } from "@/features/session/permissions";
import { useActor } from "@/features/session/session-provider";
import { LOT_STAGES, nextStage, type Lot, type LotAction } from "../types";
import { LotActions } from "./lot-actions";
import { LotCompletedBadge, LotStatusBadge } from "./lot-status-badge";
import { LotTimeline } from "./lot-timeline";
import { TransferForm } from "./transfer-form";
import { VoidDialog } from "./void-dialog";
import { QrPanel } from "./qr-panel";
import { StageForm } from "./stage-form";

const PROGRESS_STAGES = LOT_STAGES.filter((stage) => stage !== "none");

function Field({ label, children }: Readonly<{ label: string; children: ReactNode }>) {
    return (
        <div className="flex flex-col gap-0.5">
            <dt className="text-sm text-ink-dim">{label}</dt>
            <dd className="break-words font-medium">{children}</dd>
        </div>
    );
}

export function LotDetail({ id }: Readonly<{ id: string }>) {
    const t = useTranslations("Lots");
    const common = useTranslations("Common");
    const stats = useTranslations("Stats");
    const describe = useDescribeError();
    const dates = useDateFormat();
    const lot = useRemote(`lot:${id}`, () => getLot(id));
    const farms = useRemote("farms", listFarms);
    const [panel, setPanel] = useState<"stage" | "transfer" | "void" | "qr" | null>(null);
    const actor = useActor();

    const back = (
        <Link href="/lots" className="text-sm underline-offset-2 hover:underline">
            {t("detail.back")}
        </Link>
    );

    if (lot.loading) {
        return (
            <p role="status" className="text-ink-dim">
                {common("loading")}
            </p>
        );
    }

    if (lot.error !== undefined) {
        const missing = isApiError(lot.error) && lot.error.kind === "not_found";
        return (
            <div role="alert" className="flex flex-col items-start gap-3">
                {back}
                <p className={missing ? "" : "text-red-600"}>
                    {missing ? t("detail.notFound") : describe(lot.error)}
                </p>
                {!missing && (
                    <button type="button" onClick={lot.reload} className="btn-secondary">
                        {common("retry")}
                    </button>
                )}
            </div>
        );
    }

    const data = lot.data as Lot;
    const farmName = (farms.data ?? []).find((farm) => farm.id === data.farmId)?.name ?? data.farmId;
    const currentIndex = PROGRESS_STAGES.findIndex((stage) => stage === data.stage);
    const upcoming = nextStage(data.stage);
    const transferTarget = transferTargetFor(actor.type);
    const handlers: Partial<Record<LotAction, () => void>> = {};
    if (upcoming !== null) handlers["lot:register_stage"] = () => setPanel("stage");
    if (transferTarget !== null) handlers["lot:transfer"] = () => setPanel("transfer");
    handlers["lot:void"] = () => setPanel("void");
    handlers["qr:generate"] = () => setPanel("qr");
    const closePanel = () => setPanel(null);
    const finishPanel = () => {
        setPanel(null);
        lot.reload();
    };

    return (
        <div className="flex flex-col gap-6">
            {back}

            <div className="flex flex-wrap items-center gap-3">
                <h1 className="text-2xl font-semibold tracking-tight">{data.code}</h1>
                <LotStatusBadge status={data.status} label={t(`status.${data.status}`)} />
                {data.completed && <LotCompletedBadge label={t("completed")} />}
            </div>

            <LotActions allowed={data.allowedActions} handlers={handlers} />

            {actor.type === "organization" && (
                <Link href={`/lots/${data.id}/stats`} className="w-fit text-sm underline-offset-2 hover:underline">
                    {stats("link")}
                </Link>
            )}

            {panel === "stage" && upcoming !== null && (
                <StageForm
                    lotId={data.id}
                    version={data.version}
                    stage={upcoming}
                    onCancel={closePanel}
                    onDone={finishPanel}
                />
            )}

            {panel === "qr" && <QrPanel lotId={data.id} code={data.code} onClose={closePanel} />}

            {panel === "void" && (
                <VoidDialog lotId={data.id} version={data.version} onCancel={closePanel} onDone={finishPanel} />
            )}

            {panel === "transfer" && transferTarget !== null && (
                <TransferForm
                    lotId={data.id}
                    version={data.version}
                    targetType={transferTarget}
                    onCancel={closePanel}
                    onDone={finishPanel}
                />
            )}

            {data.status === "voided" && data.voidReason !== "" && (
                <p role="status" className="rounded-lg bg-terracotta/15 p-4 text-sm text-terracotta">
                    <span className="font-medium">{t("detail.voidReason")}: </span>
                    {data.voidReason}
                </p>
            )}

            <section aria-label={t("detail.progress")} className="card p-5">
                <h2 className="mb-6 text-lg font-semibold">{t("detail.progress")}</h2>
                <ol className="relative flex flex-col gap-6 sm:flex-row sm:justify-between sm:gap-0">
                    <div
                        aria-hidden
                        className="absolute hidden sm:block top-3.5 left-4 right-4 h-0.5 bg-ground-raised -z-0"
                    />

                    {PROGRESS_STAGES.map((stage, index) => {
                        const done = index < currentIndex;
                        const current = index === currentIndex;

                        return (
                            <li
                                key={stage}
                                aria-current={current ? "step" : undefined}
                                className="relative z-10 flex items-center gap-3 sm:flex-col sm:items-center sm:gap-2 sm:text-center"
                            >
                                <span
                                    aria-hidden
                                    className={`flex size-7 shrink-0 items-center justify-center rounded-full text-xs font-semibold transition-colors ${
                                        done
                                            ? "bg-verified text-white"
                                            : current
                                                ? "bg-terracotta text-on-accent ring-4 ring-terracotta/20"
                                                : "bg-ground-raised text-ink-dim"
                                    }`}
                                >
                                    {done ? "✓" : index + 1}
                                </span>
                                <span className={`text-sm ${current ? "font-semibold text-ink" : "text-ink-dim"}`}>
                                    {t(`stage.${stage}`)}
                                </span>
                            </li>
                        );
                    })}
                </ol>
            </section>

            <section className="card p-5">
                <dl className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
                    <Field label={t("columns.variety")}>{data.variety}</Field>
                    <Field label={t("columns.process")}>{t(`process.${data.process}`)}</Field>
                    <Field label={t("columns.harvestDate")}>{dates.day(data.harvestDate)}</Field>
                    <Field label={t("columns.farm")}>{farmName}</Field>
                    <Field label={t("columns.stage")}>{t(`stage.${data.stage}`)}</Field>
                    <Field label={t("detail.createdAt")}>{dates.moment(data.createdAt)}</Field>
                    <Field label={t("detail.custodian")}>
                        <span className="font-mono text-sm">{data.custodianId}</span>
                    </Field>
                    <Field label={t("detail.id")}>
                        <span className="font-mono text-sm">{data.id}</span>
                    </Field>
                    <Field label={t("detail.version")}>{data.version}</Field>
                </dl>
            </section>

            <LotTimeline lotId={data.id} version={data.version} />
        </div>
    );
}
