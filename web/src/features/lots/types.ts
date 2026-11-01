export const LOT_STAGES = ["none", "farm", "roasting", "packaging"] as const;
export type ValidLotStage = (typeof LOT_STAGES)[number];
export type LotStage = ValidLotStage | "unknown";

export function parseLotStage(value: string): LotStage {
    const normalized = value.trim().toLowerCase();
    return (LOT_STAGES as readonly string[]).includes(normalized)
        ? (normalized as ValidLotStage)
        : "unknown";
}

export type LotStatus = "active" | "voided" | "unknown";

export function parseLotStatus(value: string): LotStatus {
    return value === "active" || value === "voided" ? value : "unknown";
}

export const LOT_PROCESSES = ["washed", "natural", "honey"] as const;
export type ValidLotProcess = (typeof LOT_PROCESSES)[number];
export type LotProcess = ValidLotProcess | "unknown";

export function parseLotProcess(value: string): LotProcess {
    const normalized = value.trim().toLowerCase();
    return (LOT_PROCESSES as readonly string[]).includes(normalized)
        ? (normalized as ValidLotProcess)
        : "unknown";
}

export type Lot = Readonly<{
    id: string;
    version: number;
    farmId: string;
    custodianId: string;
    code: string;
    variety: string;
    process: LotProcess;
    harvestDate: string;
    stage: LotStage;
    status: LotStatus;
    voidReason: string;
    createdAt: string;
}>;

export type LotPage = Readonly<{
    items: readonly Lot[];
    nextCursor: string;
}>;

const LOT_ID_PATTERN = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/;

export function isLotId(value: string): boolean {
    return LOT_ID_PATTERN.test(value);
}
