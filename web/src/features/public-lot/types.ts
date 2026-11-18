import type { LotProcess, LotStage, LotStatus } from "@/features/lots/types";

export type PublicLot = Readonly<{
    id: string;
    code: string;
    variety: string;
    process: LotProcess;
    harvestDate: string;
    farmName: string;
    farmAltitude: number;
    producerName: string;
    stage: LotStage;
    status: LotStatus;
    updatedAt: string;
    evidences: readonly PublicEvidence[];
    stages: readonly PublicStage[];
}>;

export type PublicStage = Readonly<{
    stage: LotStage;
    eventDate: string;
    dryingMethod: string;
    dryingDays: number;
    roastLevel: string;
    roastDurationMinutes: number;
    grind: string;
    packageWeightGrams: number;
    bestBefore: string;
}>;

export type PublicEvidence = Readonly<{
    id: string;
    stage: LotStage;
    thumbnailUrl: string;
    imageUrl: string;
}>;

export type PublicLotResult =
    | Readonly<{ kind: "ok"; lot: PublicLot }>
    | Readonly<{ kind: "not_found" }>
    | Readonly<{ kind: "unavailable" }>;
