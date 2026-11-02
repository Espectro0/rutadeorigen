"use client";

import { useTranslations } from "next-intl";
import { useDateFormat } from "@/lib/use-date-format";
import { useDescribeError } from "@/lib/use-describe-error";
import { useRemote } from "@/lib/use-remote";
import { listLotEvents } from "../api";
import { isDetailKey, isDetailOption, type LotEvent } from "../types";

type Props = Readonly<{ lotId: string; version: number }>;

function DetailRow({ name, value }: Readonly<{ name: string; value: string | number }>) {
    const t = useTranslations("Lots.stageForm");
    const label = isDetailKey(name) ? t(`fields.${name}`) : name;
    const text = typeof value === "string" && isDetailOption(value) ? t(`options.${value}`) : String(value);

    return (
        <div className="flex flex-col gap-0.5">
            <dt className="text-xs text-ink-dim">{label}</dt>
            <dd className="text-sm font-medium">{text}</dd>
        </div>
    );
}

function EventItem({ event }: Readonly<{ event: LotEvent }>) {
    const t = useTranslations("Lots");
    const types = useTranslations("ActorTypes");
    const dates = useDateFormat();
    const details = Object.entries(event.details);
    const title =
        event.type === "transfer"
            ? t("timeline.type.transfer")
            : event.type === "correction"
              ? t("timeline.type.correction")
              : t(`stage.${event.stage}`);

    return (
        <li className="relative flex flex-col gap-2 pl-8">
            <span
                aria-hidden
                className="absolute left-0 top-0.5 flex size-6 items-center justify-center rounded-full bg-terracotta text-xs font-semibold text-on-accent"
            >
                {event.sequence}
            </span>

            <div className="flex flex-wrap items-baseline gap-x-3 gap-y-1">
                <h3 className="font-semibold">{title}</h3>
                <span className="text-sm text-ink-dim">{dates.day(event.eventDate)}</span>
                <span className="text-sm text-ink-dim">{t("timeline.by", { role: types(event.actorRole) })}</span>
            </div>

            {details.length > 0 && (
                <dl className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
                    {details.map(([name, value]) => (
                        <DetailRow key={name} name={name} value={value} />
                    ))}
                </dl>
            )}

            {event.note !== "" && <p className="text-sm">{event.note}</p>}

            <p className="font-mono text-xs text-ink-dim" title={event.hash}>
                {t("timeline.hash")}: {event.hash.slice(0, 12)}…
            </p>
        </li>
    );
}

export function LotTimeline({ lotId, version }: Props) {
    const t = useTranslations("Lots.timeline");
    const common = useTranslations("Common");
    const describe = useDescribeError();
    const history = useRemote(`lot-events:${lotId}:${version}`, () => listLotEvents(lotId));

    return (
        <section aria-label={t("title")} className="card flex flex-col gap-5 p-5">
            <div className="flex flex-wrap items-center justify-between gap-3">
                <h2 className="text-lg font-semibold">{t("title")}</h2>
                {history.data !== undefined && history.data.items.length > 0 && (
                    <span
                        role="status"
                        className={`rounded-full px-3 py-1 text-xs font-medium ${
                            history.data.chainValid ? "bg-verified/15 text-verified" : "bg-red-600/15 text-red-600"
                        }`}
                    >
                        {history.data.chainValid ? t("chainValid") : t("chainBroken")}
                    </span>
                )}
            </div>

            {history.loading && (
                <p role="status" className="text-sm text-ink-dim">
                    {common("loading")}
                </p>
            )}

            {!history.loading && history.error !== undefined && (
                <div role="alert" className="flex flex-col items-start gap-3">
                    <p className="text-sm text-red-600">{describe(history.error)}</p>
                    <button type="button" onClick={history.reload} className="btn-secondary">
                        {common("retry")}
                    </button>
                </div>
            )}

            {history.data !== undefined && history.data.items.length === 0 && (
                <p className="text-sm text-ink-dim">{t("empty")}</p>
            )}

            {history.data !== undefined && !history.data.chainValid && history.data.items.length > 0 && (
                <p role="alert" className="rounded-lg bg-red-600/10 p-4 text-sm text-red-600">
                    {t("chainBrokenHelp")}
                </p>
            )}

            {history.data !== undefined && history.data.items.length > 0 && (
                <ol className="flex flex-col gap-6 border-l border-hairline ml-3">
                    {history.data.items.map((event) => (
                        <EventItem key={event.id} event={event} />
                    ))}
                </ol>
            )}
        </section>
    );
}
