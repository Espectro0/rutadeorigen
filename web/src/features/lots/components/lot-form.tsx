"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useState, type FormEvent } from "react";
import { useTranslations } from "next-intl";
import { listFarms } from "@/features/farms/api";
import { useDescribeError } from "@/lib/use-describe-error";
import { useRemote } from "@/lib/use-remote";
import { createLot } from "../api";
import { LOT_PROCESSES } from "../types";

export function LotForm() {
    const t = useTranslations("Lots.form");
    const lots = useTranslations("Lots");
    const router = useRouter();
    const describe = useDescribeError();
    const farms = useRemote("farms", listFarms);
    const [farmId, setFarmId] = useState("");
    const [code, setCode] = useState("");
    const [variety, setVariety] = useState("");
    const [process, setProcess] = useState<string>(LOT_PROCESSES[0]);
    const [harvestDate, setHarvestDate] = useState("");
    const [submitting, setSubmitting] = useState(false);
    const [error, setError] = useState<string>();

    async function onSubmit(event: FormEvent<HTMLFormElement>) {
        event.preventDefault();
        setSubmitting(true);
        setError(undefined);
        try {
            const created = await createLot({ farmId, code, variety, process, harvestDate });
            router.push(`/lots/${created.id}`);
        } catch (cause) {
            setError(describe(cause));
            setSubmitting(false);
        }
    }

    const inputClass = "rounded-md border border-hairline bg-ground px-3 py-2";

    return (
        <form noValidate onSubmit={onSubmit} className="card flex max-w-xl flex-col gap-4 p-6">
            <label className="flex flex-col gap-1 text-sm">
                <span className="font-medium">{t("farm")}</span>
                <select value={farmId} onChange={(event) => setFarmId(event.target.value)} className={inputClass}>
                    <option value="">{t("farmPlaceholder")}</option>
                    {(farms.data ?? []).map((farm) => (
                        <option key={farm.id} value={farm.id}>
                            {farm.name}
                        </option>
                    ))}
                </select>
            </label>
            <label className="flex flex-col gap-1 text-sm">
                <span className="font-medium">{t("code")}</span>
                <input type="text" value={code} onChange={(event) => setCode(event.target.value)} className={inputClass} />
            </label>
            <label className="flex flex-col gap-1 text-sm">
                <span className="font-medium">{t("variety")}</span>
                <input
                    type="text"
                    value={variety}
                    onChange={(event) => setVariety(event.target.value)}
                    className={inputClass}
                />
            </label>
            <label className="flex flex-col gap-1 text-sm">
                <span className="font-medium">{t("process")}</span>
                <select value={process} onChange={(event) => setProcess(event.target.value)} className={inputClass}>
                    {LOT_PROCESSES.map((value) => (
                        <option key={value} value={value}>
                            {lots(`process.${value}`)}
                        </option>
                    ))}
                </select>
            </label>
            <label className="flex flex-col gap-1 text-sm">
                <span className="font-medium">{t("harvestDate")}</span>
                <input
                    type="date"
                    value={harvestDate}
                    onChange={(event) => setHarvestDate(event.target.value)}
                    className={inputClass}
                />
            </label>
            {error !== undefined && (
                <p role="alert" className="text-sm text-red-600">
                    {error}
                </p>
            )}
            <div className="flex gap-3">
                <button type="submit" disabled={submitting} className="btn-primary">
                    {t("submit")}
                </button>
                <Link href="/lots" className="btn-secondary">
                    {t("cancel")}
                </Link>
            </div>
        </form>
    );
}
