"use client";

import { useEffect, useRef, useState, type ChangeEvent, type FormEvent } from "react";
import { useTranslations } from "next-intl";
import { useDescribeError } from "@/lib/use-describe-error";
import { uploadEvidence } from "../api";
import { ACCEPTED_IMAGE_TYPES, MAX_IMAGE_BYTES } from "../types";

type FileProblem = "type" | "size";

function problemWith(file: File): FileProblem | null {
    if (!(ACCEPTED_IMAGE_TYPES as readonly string[]).includes(file.type)) return "type";
    if (file.size > MAX_IMAGE_BYTES) return "size";
    return null;
}

type Props = Readonly<{
    lotId: string;
    onDone: () => void;
    onCancel: () => void;
}>;

export function EvidenceUploader({ lotId, onDone, onCancel }: Props) {
    const t = useTranslations("Evidences.uploader");
    const describe = useDescribeError();
    const [file, setFile] = useState<File | null>(null);
    const [previewUrl, setPreviewUrl] = useState("");
    const [problem, setProblem] = useState<FileProblem | null>(null);
    const [uploading, setUploading] = useState(false);
    const [error, setError] = useState<string>();

    const previewRef = useRef("");

    useEffect(() => {
        return () => {
            if (previewRef.current !== "") URL.revokeObjectURL(previewRef.current);
        };
    }, []);

    function showPreview(picked: File | null) {
        if (previewRef.current !== "") URL.revokeObjectURL(previewRef.current);
        previewRef.current = picked === null ? "" : URL.createObjectURL(picked);
        setPreviewUrl(previewRef.current);
    }

    function onPick(event: ChangeEvent<HTMLInputElement>) {
        const picked = event.target.files?.[0] ?? null;
        setError(undefined);
        const found = picked === null ? null : problemWith(picked);
        const accepted = picked !== null && found === null ? picked : null;
        setProblem(found);
        setFile(accepted);
        showPreview(accepted);
    }

    async function onSubmit(event: FormEvent<HTMLFormElement>) {
        event.preventDefault();
        if (file === null) return;
        setUploading(true);
        setError(undefined);

        try {
            await uploadEvidence(lotId, file);
            onDone();
        } catch (cause) {
            setError(describe(cause));
            setUploading(false);
        }
    }

    return (
        <form noValidate onSubmit={onSubmit} className="card flex max-w-xl flex-col gap-4 p-6">
            <h2 className="text-lg font-semibold">{t("title")}</h2>

            <label className="flex flex-col gap-1 text-sm">
                <span className="font-medium">{t("file")}</span>
                <input
                    type="file"
                    accept={ACCEPTED_IMAGE_TYPES.join(",")}
                    onChange={onPick}
                    disabled={uploading}
                    className="rounded-md border border-hairline bg-ground px-3 py-2"
                />
                <span className="text-ink-dim">{t("hint", { size: MAX_IMAGE_BYTES / (1024 * 1024) })}</span>
            </label>

            {problem !== null && (
                <p role="alert" className="text-sm text-red-600">
                    {t(`problems.${problem}`, { size: MAX_IMAGE_BYTES / (1024 * 1024) })}
                </p>
            )}

            {previewUrl !== "" && (
                <img src={previewUrl} alt={t("previewAlt")} className="max-h-64 w-full rounded-lg object-contain" />
            )}

            {uploading && (
                <p role="status" className="text-sm text-ink-dim">
                    {t("uploading")}
                </p>
            )}

            {error !== undefined && (
                <p role="alert" className="text-sm text-red-600">
                    {error}
                </p>
            )}

            <div className="flex gap-3">
                <button type="submit" disabled={uploading || file === null} className="btn-primary">
                    {t("submit")}
                </button>
                <button type="button" onClick={onCancel} disabled={uploading} className="btn-secondary">
                    {t("cancel")}
                </button>
            </div>
        </form>
    );
}
