import { parseLotStage, type LotStage } from "@/features/lots/types";

export const EVIDENCE_STATUSES = ["pending_upload", "uploaded", "processed", "rejected"] as const;
export type ValidEvidenceStatus = (typeof EVIDENCE_STATUSES)[number];
export type EvidenceStatus = ValidEvidenceStatus | "unknown";

export function parseEvidenceStatus(value: string): EvidenceStatus {
    return (EVIDENCE_STATUSES as readonly string[]).includes(value) ? (value as ValidEvidenceStatus) : "unknown";
}

export const ACCEPTED_IMAGE_TYPES = ["image/jpeg", "image/png", "image/webp"] as const;
export const MAX_IMAGE_BYTES = 5 * 1024 * 1024;

export type Evidence = Readonly<{
    id: string;
    lotId: string;
    stage: LotStage;
    fileName: string;
    contentType: string;
    sizeBytes: number;
    status: EvidenceStatus;
    rejectionReason: string;
    createdAt: string;
    thumbnailUrl: string;
    imageUrl: string;
    urlExpiresAt: string;
}>;

export type UploadTarget = Readonly<{
    url: string;
    method: string;
    headers: Readonly<Record<string, string>>;
}>;

export type EvidenceTile = Readonly<{
    id: string;
    stage: LotStage;
    state: "ready" | "processing" | "rejected";
    fileName: string;
    thumbnailUrl: string;
    imageUrl: string;
    rejectionReason: string;
}>;

export function isProcessing(evidence: Evidence): boolean {
    return evidence.status === "pending_upload" || evidence.status === "uploaded";
}

export function toTile(evidence: Evidence): EvidenceTile {
    const ready = evidence.status === "processed" && evidence.thumbnailUrl !== "" && evidence.imageUrl !== "";
    const rejected = evidence.status === "rejected";
    return {
        id: evidence.id,
        stage: evidence.stage,
        state: ready ? "ready" : rejected ? "rejected" : "processing",
        fileName: evidence.fileName,
        thumbnailUrl: evidence.thumbnailUrl,
        imageUrl: evidence.imageUrl,
        rejectionReason: evidence.rejectionReason,
    };
}

const POLL_MS = 3000;
const MIN_EXPIRY_REFRESH_MS = 5000;
const EXPIRY_MARGIN_MS = 60_000;
const MAX_TIMER_MS = 2_147_483_647;

export function nextRefreshDelay(items: readonly Evidence[], now: number): number | null {
    if (items.some(isProcessing)) return POLL_MS;

    const expirations = items
        .filter((item) => item.urlExpiresAt !== "")
        .map((item) => Date.parse(item.urlExpiresAt))
        .filter((time) => !Number.isNaN(time));
    if (expirations.length === 0) return null;

    const untilExpiry = Math.min(...expirations) - now - EXPIRY_MARGIN_MS;
    return Math.min(Math.max(untilExpiry, MIN_EXPIRY_REFRESH_MS), MAX_TIMER_MS);
}

export function parseEvidenceStage(value: string): LotStage {
    return parseLotStage(value);
}
