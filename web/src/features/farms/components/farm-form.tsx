"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useState, type FormEvent } from "react";
import { useTranslations } from "next-intl";
import { useDescribeError } from "@/lib/use-describe-error";
import { createFarm } from "../api";

export function FarmForm() {
    const t = useTranslations("Farms.form");
    const router = useRouter();
    const describe = useDescribeError();
    const [name, setName] = useState("");
    const [altitude, setAltitude] = useState("");
    const [submitting, setSubmitting] = useState(false);
    const [error, setError] = useState<string>();

    async function onSubmit(event: FormEvent<HTMLFormElement>) {
        event.preventDefault();
        setSubmitting(true);
        setError(undefined);
        try {
            await createFarm({ name, altitude });
            router.push("/farms");
        } catch (cause) {
            setError(describe(cause));
            setSubmitting(false);
        }
    }

    return (
        <form noValidate onSubmit={onSubmit} className="card flex max-w-xl flex-col gap-4 p-6">
            <label className="flex flex-col gap-1 text-sm">
                <span className="font-medium">{t("name")}</span>
                <input
                    type="text"
                    value={name}
                    onChange={(event) => setName(event.target.value)}
                    className="rounded-md border border-hairline bg-ground px-3 py-2"
                />
            </label>
            <label className="flex flex-col gap-1 text-sm">
                <span className="font-medium">{t("altitude")}</span>
                <input
                    type="number"
                    value={altitude}
                    onChange={(event) => setAltitude(event.target.value)}
                    className="rounded-md border border-hairline bg-ground px-3 py-2"
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
                <Link href="/farms" className="btn-secondary">
                    {t("cancel")}
                </Link>
            </div>
        </form>
    );
}
