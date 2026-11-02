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

export const LOT_ACTIONS = ["lot:register_stage", "lot:transfer", "lot:void", "qr:generate"] as const;
export type LotAction = (typeof LOT_ACTIONS)[number];

export function parseLotActions(values: readonly string[] | null | undefined): readonly LotAction[] {
    return LOT_ACTIONS.filter((action) => (values ?? []).includes(action));
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
    allowedActions: readonly LotAction[];
}>;

export type LotPage = Readonly<{
    items: readonly Lot[];
    nextCursor: string;
}>;

const LOT_ID_PATTERN = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/;

export function isLotId(value: string): boolean {
    return LOT_ID_PATTERN.test(value);
}

export type StageInput = Exclude<ValidLotStage, "none">;

export function nextStage(current: LotStage): StageInput | null {
    const index = (LOT_STAGES as readonly string[]).indexOf(current);
    if (index < 0) return null;
    const next = LOT_STAGES[index + 1];
    return next === undefined || next === "none" ? null : next;
}

export const DRYING_METHODS = ["sun_patio", "african_bed", "greenhouse", "mechanical"] as const;
export const ROAST_LEVELS = ["light", "medium", "dark"] as const;
export const GRINDS = ["whole_bean", "ground"] as const;
export const PACKAGE_WEIGHTS_G = [250, 340, 500, 1000] as const;

export type StageRegistration = Readonly<{
    stage: StageInput;
    eventDate: string;
    note: string;
    details: Readonly<Record<string, string | number>>;
}>;

export const QR_SIZES = [256, 512, 1024] as const;
export const QR_FORMATS = ["png", "svg"] as const;
export type QrFormat = (typeof QR_FORMATS)[number];
