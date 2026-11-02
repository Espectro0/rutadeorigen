import { parseLotProcess, parseLotStage, parseLotStatus } from "@/features/lots/types";
import type { PublicLot, PublicLotResult } from "./types";

const REVALIDATE_SECONDS = 60;
const DEV_QR_URL = "http://localhost:8081";

// El motivo de anulación (void_reason) no se lee a propósito: la página pública solo dice que el lote fue anulado.
type PublicLotDto = Readonly<{
    id: string;
    code: string;
    variety: string;
    process: string;
    harvest_date: string;
    farm: Readonly<{ name: string; altitude: number }>;
    producer: Readonly<{ name: string }>;
    stage: string;
    status: string;
    updated_at: string;
}>;

function baseUrl(): string {
    const configured = process.env.QR_INTERNAL_URL?.trim() ?? "";
    if (configured !== "") return configured.replace(/\/+$/, "");
    return process.env.NODE_ENV === "development" ? DEV_QR_URL : "";
}

function toPublicLot(dto: PublicLotDto): PublicLot {
    return {
        id: dto.id,
        code: dto.code,
        variety: dto.variety,
        process: parseLotProcess(dto.process),
        harvestDate: dto.harvest_date,
        farmName: dto.farm.name,
        farmAltitude: dto.farm.altitude,
        producerName: dto.producer.name,
        stage: parseLotStage(dto.stage),
        status: parseLotStatus(dto.status),
        updatedAt: dto.updated_at,
    };
}

export async function getPublicLot(id: string): Promise<PublicLotResult> {
    const base = baseUrl();
    if (base === "") return { kind: "unavailable" };

    try {
        const response = await fetch(`${base}/public/lots/${encodeURIComponent(id)}`, {
            headers: { Accept: "application/json" },
            next: { revalidate: REVALIDATE_SECONDS },
        });

        if (response.status === 404 || response.status === 400) return { kind: "not_found" };
        if (!response.ok) return { kind: "unavailable" };

        return { kind: "ok", lot: toPublicLot((await response.json()) as PublicLotDto) };
    } catch {
        return { kind: "unavailable" };
    }
}
