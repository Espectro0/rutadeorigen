"use client";

import { useState } from "react";
import { useTranslations } from "next-intl";
import { listFarms } from "@/features/farms/api";
import Link from "next/link";
import { useDateFormat } from "@/lib/use-date-format";
import { useDescribeError } from "@/lib/use-describe-error";
import { useRemote } from "@/lib/use-remote";
import { listLots, NO_FILTERS, type LotFilters } from "../api";
import { LOT_STAGES } from "../types";
import { LotPager } from "./lot-pager";
import { LotCompletedBadge, LotStatusBadge } from "./lot-status-badge";
import {CiCoffeeBean} from "react-icons/ci";

const PAGE_SIZES = [10, 20, 50] as const;
const DEFAULT_PAGE_SIZE = 20;

export function LotList() {
    const t = useTranslations("Lots");
    const common = useTranslations("Common");
    const describe = useDescribeError();
    const dates = useDateFormat();

    const [limit, setLimit] = useState<number>(DEFAULT_PAGE_SIZE);
    const [cursors, setCursors] = useState<readonly string[]>([""]);
    const [pageIndex, setPageIndex] = useState(0);
    const [filters, setFilters] = useState<LotFilters>(NO_FILTERS);

    const cursor = cursors[pageIndex] ?? "";
    const filterKey = `${filters.status}:${filters.stage}:${filters.farmId}`;
    const page = useRemote(`lots:${limit}:${cursor}:${filterKey}`, () => listLots({ limit, cursor, filters }));
    const filtered = filterKey !== "::";
    const farms = useRemote("farms", listFarms);

    const farmNames = new Map((farms.data ?? []).map((farm) => [farm.id, farm.name]));
    const nextCursor = page.data?.nextCursor ?? "";

    function select(index: number) {
        if (index === pageIndex) return;
        if (index > pageIndex) {
            if (nextCursor === "") return;
            setCursors((previous) => {
                const copy = previous.slice();
                copy[pageIndex + 1] = nextCursor;
                return copy;
            });
        }
        setPageIndex(index);
    }

    function changeFilters(next: LotFilters) {
        setFilters(next);
        setCursors([""]);
        setPageIndex(0);
    }

    function changeLimit(value: number) {
        setLimit(value);
        setCursors([""]);
        setPageIndex(0);
    }

    const selectClass = "rounded border px-3 py-2";

    return (
        <div className="flex flex-col gap-4">
            <div role="group" aria-label={t("filters.label")} className="flex flex-wrap items-end gap-3">
                <label className="flex flex-col gap-1 text-sm">
                    <span className="font-medium">{t("filters.status")}</span>
                    <select
                        value={filters.status}
                        onChange={(event) => changeFilters({ ...filters, status: event.target.value })}
                        className={selectClass}
                    >
                        <option value="">{t("filters.all")}</option>
                        <option value="active">{t("status.active")}</option>
                        <option value="voided">{t("status.voided")}</option>
                    </select>
                </label>
                <label className="flex flex-col gap-1 text-sm">
                    <span className="font-medium">{t("filters.stage")}</span>
                    <select
                        value={filters.stage}
                        onChange={(event) => changeFilters({ ...filters, stage: event.target.value })}
                        className={selectClass}
                    >
                        <option value="">{t("filters.all")}</option>
                        {LOT_STAGES.map((stage) => (
                            <option key={stage} value={stage}>
                                {t(`stage.${stage}`)}
                            </option>
                        ))}
                    </select>
                </label>
                <label className="flex flex-col gap-1 text-sm">
                    <span className="font-medium">{t("filters.farm")}</span>
                    <select
                        value={filters.farmId}
                        onChange={(event) => changeFilters({ ...filters, farmId: event.target.value })}
                        className={selectClass}
                    >
                        <option value="">{t("filters.allFarms")}</option>
                        {(farms.data ?? []).map((farm) => (
                            <option key={farm.id} value={farm.id}>
                                {farm.name}
                            </option>
                        ))}
                    </select>
                </label>
                {filtered && (
                    <button type="button" onClick={() => changeFilters(NO_FILTERS)} className="btn-secondary">
                        {t("filters.clear")}
                    </button>
                )}
            </div>

            {page.loading && (
                <p role="status" className="text-ink-dim">
                    {common("loading")}
                </p>
            )}

            {!page.loading && page.error !== undefined && (
                <div role="alert" className="flex flex-col items-start gap-3">
                    <p className="text-red-600">{describe(page.error)}</p>
                    <button type="button" onClick={page.reload} className="btn-secondary">
                        {common("retry")}
                    </button>
                </div>
            )}

            {!page.loading && page.error === undefined && (page.data?.items.length ?? 0) === 0 && (
                <div className="flex flex-col items-center justify-center gap-3 w-full py-10">
                    <CiCoffeeBean className="size-12 text-ink-dim" />
                    <p className="text-ink-dim text-center">{filtered ? t("emptyFiltered") : t("empty")}</p>
                </div>
            )}

            {!page.loading && page.data !== undefined && page.data.items.length > 0 && (
                <div className="card overflow-x-auto">
                    <table className="w-full min-w-[48rem] text-left text-sm">
                        <thead className="border-b border-hairline text-ink-dim">
                        <tr>
                            <th scope="col" className="px-4 py-3 font-medium">
                                {t("columns.code")}
                            </th>
                            <th scope="col" className="px-4 py-3 font-medium">
                                {t("columns.variety")}
                            </th>
                            <th scope="col" className="px-4 py-3 font-medium">
                                {t("columns.process")}
                            </th>
                            <th scope="col" className="px-4 py-3 font-medium">
                                {t("columns.harvestDate")}
                            </th>
                            <th scope="col" className="px-4 py-3 font-medium">
                                {t("columns.farm")}
                            </th>
                            <th scope="col" className="px-4 py-3 font-medium">
                                {t("columns.stage")}
                            </th>
                            <th scope="col" className="px-4 py-3 font-medium">
                                {t("columns.status")}
                            </th>
                        </tr>
                        </thead>
                        <tbody>
                        {page.data.items.map((lot) => (
                            <tr key={lot.id} className="border-b border-hairline last:border-b-0">
                                <td className="px-4 py-3 font-medium">
                                    <Link href={`/lots/${lot.id}`} className="underline-offset-2 hover:underline">
                                        {lot.code}
                                    </Link>
                                </td>
                                <td className="px-4 py-3">{lot.variety}</td>
                                <td className="px-4 py-3">{t(`process.${lot.process}`)}</td>
                                <td className="px-4 py-3">{dates.day(lot.harvestDate)}</td>
                                <td className="px-4 py-3">{farmNames.get(lot.farmId) ?? "—"}</td>
                                <td className="px-4 py-3">{t(`stage.${lot.stage}`)}</td>
                                <td className="px-4 py-3">
                                    <div className="flex flex-wrap items-center gap-2">
                                        <LotStatusBadge status={lot.status} label={t(`status.${lot.status}`)} />
                                        {lot.completed && <LotCompletedBadge label={t("completed")} />}
                                    </div>
                                </td>
                            </tr>
                        ))}
                        </tbody>
                    </table>
                </div>
            )}

            <div className="grid grid-cols-1 md:grid-cols-3 items-center gap-4 pt-2">
                {/* 1. Lado izquierdo: Selector Por página */}
                <div className="flex items-center gap-2 justify-start">
                    <label htmlFor="lot-page-size" className="text-sm font-medium">
                        {t("pageSize")}
                    </label>
                    <select
                        id="lot-page-size"
                        value={limit}
                        onChange={(event) => changeLimit(Number(event.target.value))}
                        className="rounded border px-3 py-2"
                    >
                        {PAGE_SIZES.map((size) => (
                            <option key={size} value={size}>
                                {size}
                            </option>
                        ))}
                    </select>
                </div>

                <div className="flex justify-center">
                    <LotPager
                        pages={cursors.length}
                        current={pageIndex}
                        hasNext={nextCursor !== ""}
                        disabled={page.loading}
                        onSelect={select}
                    />
                </div>

                <div />
            </div>
        </div>
    );
}