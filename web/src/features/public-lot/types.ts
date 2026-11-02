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
