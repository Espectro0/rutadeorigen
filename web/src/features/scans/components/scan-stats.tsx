"use client";

import { useState, type ReactNode } from "react";
import Link from "next/link";
import { useTranslations } from "next-intl";
import { useActor } from "@/features/session/session-provider";
import { useDateFormat } from "@/lib/use-date-format";
import { useDescribeError } from "@/lib/use-describe-error";
import { useRemote } from "@/lib/use-remote";
import { getScanStats } from "../api";
import { DEFAULT_STATS_DAYS, STATS_DAYS, type ScanStats as ScanStatsData } from "../types";

const DAY_MS = 24 * 60 * 60 * 1000;

function windowDays(days: number): string[] {
    const now = Date.now();
    return Array.from({ length: days }, (_, index) =>
        new Date(now - (days - 1 - index) * DAY_MS).toISOString().slice(0, 10),
    );
}

function Summary({ label, children }: Readonly<{ label: string; children: ReactNode }>) {
    return (
        <div className="card flex flex-col gap-1 p-4">
            <dt className="text-sm text-ink-dim">{label}</dt>
            <dd className="text-xl font-semibold tabular-nums">{children}</dd>
        </div>
    );
}

function Chart({ stats }: Readonly<{ stats: ScanStatsData }>) {
    const t = useTranslations("Stats");
    const dates = useDateFormat();
    const counts = new Map(stats.daily.map((entry) => [entry.date, entry.count]));
    const series = windowDays(stats.days).map((date) => ({ date, count: counts.get(date) ?? 0 }));
    const max = Math.max(1, ...series.map((entry) => entry.count));

    return (
        <section aria-label={t("chartTitle")} className="card flex flex-col gap-4 p-5">
            <h2 className="text-lg font-semibold">{t("chartTitle")}</h2>
            <div className="flex h-48 items-end gap-px" role="img" aria-label={t("chartLabel", { days: stats.days, total: stats.total })}>
                {series.map((entry) => (
                    <div
                        key={entry.date}
                        title={`${dates.day(entry.date)}: ${entry.count}`}
                        className="flex h-full min-w-0 flex-1 items-end"
                    >
                        <div
                            className={entry.count > 0 ? "w-full rounded-t bg-terracotta" : "w-full bg-ground-raised"}
                            style={{ height: entry.count > 0 ? `${Math.max(4, (entry.count / max) * 100)}%` : "2px" }}
                        />
                    </div>
                ))}
            </div>
            <div className="flex justify-between text-xs text-ink-dim">
                <span>{dates.day(series[0]?.date ?? "")}</span>
                <span>{dates.day(series[series.length - 1]?.date ?? "")}</span>
            </div>
        </section>
    );
}

function StatsView({ lotId }: Readonly<{ lotId: string }>) {
    const t = useTranslations("Stats");
    const common = useTranslations("Common");
    const describe = useDescribeError();
    const dates = useDateFormat();
    const [days, setDays] = useState<number>(DEFAULT_STATS_DAYS);
    const stats = useRemote(`scan-stats:${lotId}:${days}`, () => getScanStats(lotId, days));

    return (
        <div className="flex flex-col gap-6">
            <label className="flex w-fit flex-col gap-1 text-sm">
                <span className="font-medium">{t("period")}</span>
                <select
                    value={days}
                    onChange={(event) => setDays(Number(event.target.value))}
                    className="rounded border px-3 py-2"
                >
                    {STATS_DAYS.map((value) => (
                        <option key={value} value={value}>
                            {t("daysOption", { days: value })}
                        </option>
                    ))}
                </select>
            </label>

            {stats.loading && (
                <p role="status" className="text-ink-dim">
                    {common("loading")}
                </p>
            )}

            {!stats.loading && stats.error !== undefined && (
                <div role="alert" className="flex flex-col items-start gap-3">
                    <p className="text-red-600">{describe(stats.error)}</p>
                    <button type="button" onClick={stats.reload} className="btn-secondary">
                        {common("retry")}
                    </button>
                </div>
            )}

            {stats.data !== undefined && (
                <>
                    <dl className="grid gap-4 sm:grid-cols-3">
                        <Summary label={t("total")}>{stats.data.total}</Summary>
                        <Summary label={t("first")}>
                            {stats.data.firstScanAt === "" ? "—" : dates.moment(stats.data.firstScanAt)}
                        </Summary>
                        <Summary label={t("last")}>
                            {stats.data.lastScanAt === "" ? "—" : dates.moment(stats.data.lastScanAt)}
                        </Summary>
                    </dl>

                    {stats.data.total === 0 ? (
                        <p className="text-ink-dim">{t("empty")}</p>
                    ) : (
                        <Chart stats={stats.data} />
                    )}
                </>
            )}
        </div>
    );
}

export function ScanStats({ lotId }: Readonly<{ lotId: string }>) {
    const t = useTranslations("Stats");
    const actor = useActor();

    return (
        <div className="flex flex-col gap-6">
            <Link href={`/lots/${lotId}`} className="text-sm underline-offset-2 hover:underline">
                {t("back")}
            </Link>
            <h1 className="text-2xl font-semibold tracking-tight">{t("title")}</h1>
            {actor.type === "organization" ? (
                <StatsView lotId={lotId} />
            ) : (
                <p role="alert" className="text-ink-dim">
                    {t("notAllowed")}
                </p>
            )}
        </div>
    );
}
