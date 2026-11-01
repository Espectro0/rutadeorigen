"use client";

import { useTranslations } from "next-intl";
import { useDateFormat } from "@/lib/use-date-format";
import { useDescribeError } from "@/lib/use-describe-error";
import { useRemote } from "@/lib/use-remote";
import { listFarms } from "../api";
import {PiFarm} from "react-icons/pi";

export function FarmList() {
    const t = useTranslations("Farms");
    const common = useTranslations("Common");
    const describe = useDescribeError();
    const dates = useDateFormat();
    const farms = useRemote("farms", listFarms);

    if (farms.loading) {
        return (
            <p role="status" className="text-ink-dim">
                {common("loading")}
            </p>
        );
    }

    if (farms.error !== undefined) {
        return (
            <div role="alert" className="flex flex-col items-start gap-3">
                <p className="text-red-600">{describe(farms.error)}</p>
                <button type="button" onClick={farms.reload} className="btn-secondary">
                    {common("retry")}
                </button>
            </div>
        );
    }

    const items = farms.data ?? [];

    if (items.length === 0) {
        return (
            <div className="flex flex-col items-center justify-center gap-3 w-full py-10">
                <PiFarm className="size-12 text-ink-dim" />
                <p className="text-ink-dim text-center">{t("empty")}</p>
            </div>
        );
    }

    return (
        <div className="card overflow-x-auto">
            <table className="w-full min-w-[32rem] text-left text-sm">
                <thead className="border-b border-hairline text-ink-dim">
                    <tr>
                        <th scope="col" className="px-4 py-3 font-medium">
                            {t("columns.name")}
                        </th>
                        <th scope="col" className="px-4 py-3 font-medium">
                            {t("columns.altitude")}
                        </th>
                        <th scope="col" className="px-4 py-3 font-medium">
                            {t("columns.createdAt")}
                        </th>
                    </tr>
                </thead>
                <tbody>
                    {items.map((farm) => (
                        <tr key={farm.id} className="border-b border-hairline last:border-b-0">
                            <td className="px-4 py-3 font-medium">{farm.name}</td>
                            <td className="px-4 py-3 tabular-nums">
                                {t("altitudeValue", { value: farm.altitude })}
                            </td>
                            <td className="px-4 py-3">{dates.moment(farm.createdAt)}</td>
                        </tr>
                    ))}
                </tbody>
            </table>
        </div>
    );
}
