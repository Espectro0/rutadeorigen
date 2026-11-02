"use client";

import { useEffect, useRef, useState } from "react";
import { useTranslations } from "next-intl";
import { useDescribeError } from "@/lib/use-describe-error";
import { deleteFarm } from "../api";
import type { Farm } from "../types";

type Props = Readonly<{
    farm: Farm;
    onDone: () => void;
    onCancel: () => void;
}>;

export function DeleteFarmDialog({ farm, onDone, onCancel }: Props) {
    const t = useTranslations("Farms.delete");
    const describe = useDescribeError();
    const dialog = useRef<HTMLDialogElement>(null);
    const [submitting, setSubmitting] = useState(false);
    const [error, setError] = useState<string>();

    useEffect(() => {
        const element = dialog.current;
        if (element !== null && !element.open) element.showModal();
        return () => element?.close();
    }, []);

    async function onConfirm() {
        setSubmitting(true);
        setError(undefined);
        try {
            await deleteFarm(farm.id);
            onDone();
        } catch (cause) {
            setError(describe(cause));
            setSubmitting(false);
        }
    }

    return (
        <dialog
            ref={dialog}
            aria-labelledby="delete-farm-title"
            onCancel={(event) => {
                event.preventDefault();
                if (!submitting) onCancel();
            }}
            className="m-auto w-full max-w-md rounded-lg border border-hairline bg-ground p-0 text-ink backdrop:bg-black/40"
        >
            <div className="flex flex-col gap-4 p-6">
                <h2 id="delete-farm-title" className="text-lg font-semibold">
                    {t("title")}
                </h2>

                <p className="text-sm">{t("confirm", { name: farm.name })}</p>
                <p className="rounded-lg bg-terracotta/15 p-4 text-sm text-terracotta">{t("hint")}</p>

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
                        type="button"
                        onClick={onConfirm}
                        disabled={submitting}
                        className="btn-primary bg-red-600 hover:bg-red-700"
                    >
                        {t("submit")}
                    </button>
                </div>
            </div>
        </dialog>
    );
}
