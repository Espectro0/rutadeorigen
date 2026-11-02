"use client";

import { useEffect, useState } from "react";
import { useTranslations } from "next-intl";
import { useDescribeError } from "@/lib/use-describe-error";
import { useRemote } from "@/lib/use-remote";
import { getQrImage } from "../api";
import { QR_FORMATS, QR_SIZES, type QrFormat } from "../types";

type Props = Readonly<{
    lotId: string;
    code: string;
    onClose: () => void;
}>;

function useObjectUrl(blob: Blob | undefined): string {
    const [url, setUrl] = useState("");

    useEffect(() => {
        if (blob === undefined) {
            setUrl("");
            return;
        }
        const created = URL.createObjectURL(blob);
        setUrl(created);
        return () => URL.revokeObjectURL(created);
    }, [blob]);

    return url;
}

export function QrPanel({ lotId, code, onClose }: Props) {
    const t = useTranslations("Lots.qrPanel");
    const common = useTranslations("Common");
    const describe = useDescribeError();
    const [size, setSize] = useState<number>(QR_SIZES[1]);
    const [format, setFormat] = useState<QrFormat>("png");
    const image = useRemote(`qr:${lotId}:${size}:${format}`, () => getQrImage(lotId, size, format));
    const url = useObjectUrl(image.data);

    const inputClass = "rounded-md border border-hairline bg-ground px-3 py-2";

    return (
        <section aria-label={t("title")} className="card flex max-w-xl flex-col gap-4 p-6">
            <h2 className="text-lg font-semibold">{t("title")}</h2>

            <div className="flex flex-wrap gap-4">
                <label className="flex flex-col gap-1 text-sm">
                    <span className="font-medium">{t("size")}</span>
                    <select
                        value={size}
                        onChange={(event) => setSize(Number(event.target.value))}
                        className={inputClass}
                    >
                        {QR_SIZES.map((value) => (
                            <option key={value} value={value}>
                                {value} px
                            </option>
                        ))}
                    </select>
                </label>
                <label className="flex flex-col gap-1 text-sm">
                    <span className="font-medium">{t("format")}</span>
                    <select
                        value={format}
                        onChange={(event) => setFormat(event.target.value as QrFormat)}
                        className={inputClass}
                    >
                        {QR_FORMATS.map((value) => (
                            <option key={value} value={value}>
                                {value.toUpperCase()}
                            </option>
                        ))}
                    </select>
                </label>
            </div>

            {image.loading && (
                <p role="status" className="text-sm text-ink-dim">
                    {common("loading")}
                </p>
            )}

            {!image.loading && image.error !== undefined && (
                <div role="alert" className="flex flex-col items-start gap-3">
                    <p className="text-sm text-red-600">{describe(image.error)}</p>
                    <button type="button" onClick={image.reload} className="btn-secondary">
                        {common("retry")}
                    </button>
                </div>
            )}

            {url !== "" && image.error === undefined && (
                // eslint-disable-next-line @next/next/no-img-element
                <img
                    src={url}
                    alt={t("alt", { code })}
                    className="size-64 max-w-full rounded-md border border-hairline bg-white object-contain p-2"
                />
            )}

            <div className="flex gap-3">
                {url !== "" && image.error === undefined && (
                    <a href={url} download={`qr-${code}.${format}`} className="btn-primary">
                        {t("download")}
                    </a>
                )}
                <button type="button" onClick={onClose} className="btn-secondary">
                    {t("close")}
                </button>
            </div>
        </section>
    );
}
