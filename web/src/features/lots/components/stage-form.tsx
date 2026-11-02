"use client";

import { useState, type FormEvent } from "react";
import { useTranslations } from "next-intl";
import { useDescribeError } from "@/lib/use-describe-error";
import { registerStage } from "../api";
import { DRYING_METHODS, GRINDS, PACKAGE_WEIGHTS_G, ROAST_LEVELS, type StageInput } from "../types";

type FieldName =
    | "drying_method"
    | "drying_days"
    | "moisture_percent"
    | "parchment_weight_kg"
    | "roast_level"
    | "duration_minutes"
    | "green_weight_kg"
    | "roasted_weight_kg"
    | "grind"
    | "package_weight_g"
    | "units"
    | "best_before";

type OptionValue = (typeof DRYING_METHODS | typeof ROAST_LEVELS | typeof GRINDS)[number];

type Field =
    | Readonly<{ name: FieldName; kind: "choice"; options: readonly OptionValue[] }>
    | Readonly<{ name: FieldName; kind: "weight" }>
    | Readonly<{ name: FieldName; kind: "number"; step: string }>
    | Readonly<{ name: FieldName; kind: "date" }>;

const FIELDS: Readonly<Record<StageInput, readonly Field[]>> = {
    farm: [
        { name: "drying_method", kind: "choice", options: DRYING_METHODS },
        { name: "drying_days", kind: "number", step: "1" },
        { name: "moisture_percent", kind: "number", step: "0.1" },
        { name: "parchment_weight_kg", kind: "number", step: "0.1" },
    ],
    roasting: [
        { name: "roast_level", kind: "choice", options: ROAST_LEVELS },
        { name: "duration_minutes", kind: "number", step: "1" },
        { name: "green_weight_kg", kind: "number", step: "0.1" },
        { name: "roasted_weight_kg", kind: "number", step: "0.1" },
    ],
    packaging: [
        { name: "grind", kind: "choice", options: GRINDS },
        { name: "package_weight_g", kind: "weight" },
        { name: "units", kind: "number", step: "1" },
        { name: "best_before", kind: "date" },
    ],
};

const TEXT_FIELDS: readonly FieldName[] = ["drying_method", "roast_level", "grind", "best_before"];

function today(): string {
    const now = new Date();
    const month = String(now.getMonth() + 1).padStart(2, "0");
    const day = String(now.getDate()).padStart(2, "0");
    return `${now.getFullYear()}-${month}-${day}`;
}

type Props = Readonly<{
    lotId: string;
    version: number;
    stage: StageInput;
    onDone: () => void;
    onCancel: () => void;
}>;

export function StageForm({ lotId, version, stage, onDone, onCancel }: Props) {
    const t = useTranslations("Lots.stageForm");
    const lots = useTranslations("Lots");
    const describe = useDescribeError();
    const [eventDate, setEventDate] = useState(today);
    const [note, setNote] = useState("");
    const [values, setValues] = useState<Readonly<Partial<Record<FieldName, string>>>>({});
    const [submitting, setSubmitting] = useState(false);
    const [error, setError] = useState<string>();

    function set(name: FieldName, value: string) {
        setValues((previous) => ({ ...previous, [name]: value }));
    }

    async function onSubmit(event: FormEvent<HTMLFormElement>) {
        event.preventDefault();
        setSubmitting(true);
        setError(undefined);

        const details: Record<string, string | number> = {};
        for (const field of FIELDS[stage]) {
            const raw = values[field.name] ?? "";
            details[field.name] = TEXT_FIELDS.includes(field.name) ? raw : Number(raw);
        }

        try {
            await registerStage(lotId, version, { stage, eventDate, note, details });
            onDone();
        } catch (cause) {
            setError(describe(cause));
            setSubmitting(false);
        }
    }

    const inputClass = "rounded-md border border-hairline bg-ground px-3 py-2";

    return (
        <form noValidate onSubmit={onSubmit} className="card flex max-w-xl flex-col gap-4 p-6">
            <h2 className="text-lg font-semibold">{t("title", { stage: lots(`stage.${stage}`) })}</h2>

            <label className="flex flex-col gap-1 text-sm">
                <span className="font-medium">{t("eventDate")}</span>
                <input
                    type="date"
                    value={eventDate}
                    onChange={(event) => setEventDate(event.target.value)}
                    className={inputClass}
                />
            </label>

            {FIELDS[stage].map((field) => (
                <label key={field.name} className="flex flex-col gap-1 text-sm">
                    <span className="font-medium">{t(`fields.${field.name}`)}</span>
                    {field.kind === "choice" && (
                        <select
                            value={values[field.name] ?? ""}
                            onChange={(event) => set(field.name, event.target.value)}
                            className={inputClass}
                        >
                            <option value="">{t("placeholder")}</option>
                            {field.options.map((option) => (
                                <option key={option} value={option}>
                                    {t(`options.${option}`)}
                                </option>
                            ))}
                        </select>
                    )}
                    {field.kind === "weight" && (
                        <select
                            value={values[field.name] ?? ""}
                            onChange={(event) => set(field.name, event.target.value)}
                            className={inputClass}
                        >
                            <option value="">{t("placeholder")}</option>
                            {PACKAGE_WEIGHTS_G.map((weight) => (
                                <option key={weight} value={weight}>
                                    {weight} g
                                </option>
                            ))}
                        </select>
                    )}
                    {field.kind === "number" && (
                        <input
                            type="number"
                            inputMode="decimal"
                            step={field.step}
                            value={values[field.name] ?? ""}
                            onChange={(event) => set(field.name, event.target.value)}
                            className={inputClass}
                        />
                    )}
                    {field.kind === "date" && (
                        <input
                            type="date"
                            value={values[field.name] ?? ""}
                            onChange={(event) => set(field.name, event.target.value)}
                            className={inputClass}
                        />
                    )}
                </label>
            ))}

            <label className="flex flex-col gap-1 text-sm">
                <span className="font-medium">{t("note")}</span>
                <textarea
                    rows={3}
                    value={note}
                    onChange={(event) => setNote(event.target.value)}
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
                <button type="button" onClick={onCancel} disabled={submitting} className="btn-secondary">
                    {t("cancel")}
                </button>
            </div>
        </form>
    );
}
