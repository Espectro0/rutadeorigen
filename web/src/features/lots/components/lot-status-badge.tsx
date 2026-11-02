import type { LotStatus } from "../types";

const STYLES: Record<LotStatus, string> = {
    active: "bg-verified/15 text-verified",
    voided: "bg-terracotta/15 text-terracotta",
    unknown: "bg-ground-raised text-ink-dim",
};

export function LotStatusBadge({ status, label }: Readonly<{ status: LotStatus; label: string }>) {
    return (
        <span className={`inline-block rounded-full px-2.5 py-0.5 text-xs font-medium ${STYLES[status]}`}>
            {label}
        </span>
    );
}


export function LotCompletedBadge({ label }: Readonly<{ label: string }>) {
    return (
        <span className="inline-block rounded-full border border-verified px-2.5 py-0.5 text-xs font-medium text-verified">
            {label}
        </span>
    );
}
