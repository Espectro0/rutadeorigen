"use client";

import { useState, type FormEvent } from "react";
import { useTranslations } from "next-intl";
import { listActors } from "@/features/actors/api";
import type { ValidActorType } from "@/features/session/types";
import { useDescribeError } from "@/lib/use-describe-error";
import { useRemote } from "@/lib/use-remote";
import { transferLot } from "../api";

type Props = Readonly<{
    lotId: string;
    version: number;
    targetType: ValidActorType;
    onDone: () => void;
    onCancel: () => void;
}>;

export function TransferForm({ lotId, version, targetType, onDone, onCancel }: Props) {
    const t = useTranslations("Lots.transferForm");
    const types = useTranslations("ActorTypes");
    const common = useTranslations("Common");
    const describe = useDescribeError();
    const candidates = useRemote(`actors:${targetType}`, () => listActors(targetType));
    const [targetActorId, setTargetActorId] = useState("");
    const [note, setNote] = useState("");
    const [submitting, setSubmitting] = useState(false);
    const [error, setError] = useState<string>();

    async function onSubmit(event: FormEvent<HTMLFormElement>) {
        event.preventDefault();
        setSubmitting(true);
        setError(undefined);
        try {
            await transferLot(lotId, version, { targetActorId, note });
            onDone();
        } catch (cause) {
            setError(describe(cause));
            setSubmitting(false);
        }
    }

    const inputClass = "rounded-md border border-hairline bg-ground px-3 py-2";
    const typeLabel = types(targetType);
    const options = candidates.data ?? [];

    return (
        <form noValidate onSubmit={onSubmit} className="card flex max-w-xl flex-col gap-4 p-6">
            <h2 className="text-lg font-semibold">{t("title")}</h2>

            {candidates.loading && (
                <p role="status" className="text-sm text-ink-dim">
                    {common("loading")}
                </p>
            )}

            {!candidates.loading && candidates.error !== undefined && (
                <div role="alert" className="flex flex-col items-start gap-3">
                    <p className="text-sm text-red-600">{describe(candidates.error)}</p>
                    <button type="button" onClick={candidates.reload} className="btn-secondary">
                        {common("retry")}
                    </button>
                </div>
            )}

            {!candidates.loading && candidates.error === undefined && options.length === 0 && (
                <p className="text-sm text-ink-dim">{t("empty", { type: typeLabel })}</p>
            )}

            {options.length > 0 && (
                <label className="flex flex-col gap-1 text-sm">
                    <span className="font-medium">{t("target", { type: typeLabel })}</span>
                    <select
                        value={targetActorId}
                        onChange={(event) => setTargetActorId(event.target.value)}
                        className={inputClass}
                    >
                        <option value="">{t("placeholder")}</option>
                        {options.map((actor) => (
                            <option key={actor.id} value={actor.id}>
                                {actor.name}
                            </option>
                        ))}
                    </select>
                </label>
            )}

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
                <button type="submit" disabled={submitting || targetActorId === ""} className="btn-primary">
                    {t("submit")}
                </button>
                <button type="button" onClick={onCancel} disabled={submitting} className="btn-secondary">
                    {t("cancel")}
                </button>
            </div>
        </form>
    );
}
