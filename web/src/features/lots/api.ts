import { request, requestBlob } from "@/lib/http-client";
import { parseActorType } from "@/features/session/types";
import {
    parseLotActions,
    parseLotEventType,
    parseLotProcess,
    parseLotStage,
    parseLotStatus,
    type Lot,
    type LotEvent,
    type LotHistory,
    type LotPage,
    type StageRegistration,
} from "./types";

type LotDto = Readonly<{
    id: string;
    version: number;
    farm_id: string;
    custodian_id: string;
    code: string;
    variety: string;
    process: string;
    harvest_date: string;
    stage: string;
    status: string;
    completed?: boolean;
    void_reason?: string;
    created_at: string;
    allowed_actions?: string[] | null;
}>;

type LotPageDto = Readonly<{
    items: LotDto[] | null;
    next_cursor?: string;
}>;

function toLot(dto: LotDto): Lot {
    return {
        id: dto.id,
        version: dto.version,
        farmId: dto.farm_id,
        custodianId: dto.custodian_id,
        code: dto.code,
        variety: dto.variety,
        process: parseLotProcess(dto.process),
        harvestDate: dto.harvest_date,
        stage: parseLotStage(dto.stage),
        status: parseLotStatus(dto.status),
        completed: dto.completed ?? false,
        voidReason: dto.void_reason ?? "",
        createdAt: dto.created_at,
        allowedActions: parseLotActions(dto.allowed_actions),
    };
}

export type LotFilters = Readonly<{ status: string; stage: string; farmId: string }>;

export const NO_FILTERS: LotFilters = { status: "", stage: "", farmId: "" };

export async function listLots(
    params: Readonly<{ limit: number; cursor: string; filters?: LotFilters }>,
): Promise<LotPage> {
    const query = new URLSearchParams({ limit: String(params.limit) });
    if (params.cursor !== "") {
        query.set("cursor", params.cursor);
    }
    const filters = params.filters ?? NO_FILTERS;
    if (filters.status !== "") query.set("status", filters.status);
    if (filters.stage !== "") query.set("stage", filters.stage);
    if (filters.farmId !== "") query.set("farm_id", filters.farmId);
    const result = await request<LotPageDto>(`/lots?${query}`);
    return {
        items: (result.items ?? []).map(toLot),
        nextCursor: result.next_cursor ?? "",
    };
}

export async function getLot(id: string): Promise<Lot> {
    return toLot(await request<LotDto>(`/lots/${encodeURIComponent(id)}`));
}

export type NewLot = Readonly<{
    farmId: string;
    code: string;
    variety: string;
    process: string;
    harvestDate: string;
}>;

export async function createLot(input: NewLot): Promise<Lot> {
    const dto = await request<LotDto>("/lots", {
        method: "POST",
        body: JSON.stringify({
            farm_id: input.farmId,
            code: input.code,
            variety: input.variety,
            process: input.process,
            harvest_date: input.harvestDate,
        }),
    });
    return toLot(dto);
}

export async function registerStage(lotId: string, version: number, input: StageRegistration): Promise<void> {
    await request<unknown>(`/lots/${encodeURIComponent(lotId)}/events`, {
        method: "POST",
        headers: { "If-Match": `"${version}"` },
        body: JSON.stringify({
            stage: input.stage,
            event_date: input.eventDate,
            note: input.note,
            details: input.details,
        }),
    });
}

export type LotTransfer = Readonly<{ targetActorId: string; note: string }>;

export async function transferLot(lotId: string, version: number, input: LotTransfer): Promise<void> {
    await request<unknown>(`/lots/${encodeURIComponent(lotId)}/transfers`, {
        method: "POST",
        headers: { "If-Match": `"${version}"` },
        body: JSON.stringify({ target_actor_id: input.targetActorId, note: input.note }),
    });
}

export async function voidLot(lotId: string, version: number, reason: string): Promise<void> {
    await request<unknown>(`/lots/${encodeURIComponent(lotId)}/void`, {
        method: "POST",
        headers: { "If-Match": `"${version}"` },
        body: JSON.stringify({ reason }),
    });
}

function toDataUrl(blob: Blob): Promise<string> {
    return new Promise((resolve, reject) => {
        const reader = new FileReader();
        reader.onload = () => resolve(String(reader.result));
        reader.onerror = () => reject(reader.error);
        reader.readAsDataURL(blob);
    });
}

export async function getQrImage(lotId: string, size: number, format: string): Promise<string> {
    const query = new URLSearchParams({ size: String(size), format });
    const blob = await requestBlob(`/public/lots/${encodeURIComponent(lotId)}/qr?${query}`);
    return toDataUrl(blob);
}

type LotEventDto = Readonly<{
    id: string;
    sequence: number;
    type: string;
    stage: string;
    event_date: string;
    note?: string;
    actor_role: string;
    details?: Record<string, unknown> | null;
    hash: string;
    registered_at: string;
}>;

type LotHistoryDto = Readonly<{
    items: LotEventDto[] | null;
    chain_valid: boolean;
}>;

function toEvent(dto: LotEventDto): LotEvent {
    const details: Record<string, string | number> = {};
    for (const [key, value] of Object.entries(dto.details ?? {})) {
        if (typeof value === "string" || typeof value === "number") {
            details[key] = value;
        }
    }
    return {
        id: dto.id,
        sequence: dto.sequence,
        type: parseLotEventType(dto.type),
        stage: parseLotStage(dto.stage),
        eventDate: dto.event_date,
        note: dto.note ?? "",
        actorRole: parseActorType(dto.actor_role),
        details,
        hash: dto.hash,
        registeredAt: dto.registered_at,
    };
}

export async function listLotEvents(lotId: string): Promise<LotHistory> {
    const result = await request<LotHistoryDto>(`/lots/${encodeURIComponent(lotId)}/events`);
    return {
        items: (result.items ?? []).map(toEvent),
        chainValid: result.chain_valid,
    };
}
