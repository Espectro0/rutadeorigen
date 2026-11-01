"use client";

import { useEffect, useRef, useState, type FormEvent } from "react";
import { useTranslations } from "next-intl";
import { useDescribeError } from "@/lib/use-describe-error";
import { voidLot } from "../api";

type Props = Readonly<{
    lotId: string;
    version: number;
    onDone: () => void;
    onCancel: () => void;
}>;

export function VoidDialog({ lotId, version, onDone, onCancel }: Props) {
    const t = useTranslations("Lots.voidDialog");
    const describe = useDescribeError();
    const dialog = useRef<HTMLDialogElement>(null);
    const [reason, setReason] = useState("");
    const [submitting, setSubmitting] = useState(false);
    const [error, setError] = useState<string>();

    useEffect(() => {
        const element = dialog.current;
        if (element !== null && !element.open) element.showModal();
        return () => element?.close();
    }, []);

    async function onSubmit(event: FormEvent<HTMLFormElement>) {
        event.preventDefault();
        setSubmitting(true);
        setError(undefined);
        try {
            await voidLot(lotId, version, reason.trim());
            onDone();
        } catch (cause) {
            setError(describe(cause));
            setSubmitting(false);
        }
    }

    return (
        <dialog
            ref={dialog}
            aria-labelledby="void-dialog-title"
            onCancel={(event) => {
                event.preventDefault();
                if (!submitting) onCancel();
            }}
            className="m-auto w-full max-w-md rounded-lg border border-hairline bg-ground p-0 text-ink backdrop:bg-black/40"
        >
            <form noValidate onSubmit={onSubmit} className="flex flex-col gap-4 p-6">
                <h2 id="void-dialog-title" className="text-lg font-semibold">
                    {t("title")}
                </h2>

                <p role="alert" className="rounded-lg bg-terracotta/15 p-4 text-sm text-terracotta">
                    {t("warning")}
                </p>

                <label className="flex flex-col gap-1 text-sm">
                    <span className="font-medium">{t("reason")}</span>
                    <textarea
                        rows={3}
                        value={reason}
                        onChange={(event) => setReason(event.target.value)}
                        className="rounded-md border border-hairline bg-ground px-3 py-2"
                    />
                </label>

                {error !== undefined && (
                    <p role="alert" className="text-sm text-red-600">
                        {error}
                    </p>
                )}

                <div className="flex justify-end gap-3">
                    <button type="button" onClick={onCancel} disabled={submitting} className="btn-secondary">
                        {t("cancel")}
                    </button>
                    <button
                        type="submit"
                        disabled={submitting || reason.trim() === ""}
                        className="btn-primary bg-red-600 hover:bg-red-700"
                    >
                        {t("submit")}
                    </button>
                </div>
            </form>
        </dialog>
    );
}
