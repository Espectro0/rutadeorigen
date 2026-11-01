import { request } from "@/lib/http-client";
import { parseLotActions, parseLotProcess, parseLotStage, parseLotStatus, type Lot, type LotPage } from "./types";

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
        voidReason: dto.void_reason ?? "",
        createdAt: dto.created_at,
        allowedActions: parseLotActions(dto.allowed_actions),
    };
}

export async function listLots(params: Readonly<{ limit: number; cursor: string }>): Promise<LotPage> {
    const query = new URLSearchParams({ limit: String(params.limit) });
    if (params.cursor !== "") {
        query.set("cursor", params.cursor);
    }
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
