import { request } from "@/lib/http-client";
import type { Farm } from "./types";

type FarmDto = Readonly<{
    id: string;
    producer_id: string;
    name: string;
    altitude: number;
    created_at: string;
}>;

type FarmListDto = Readonly<{ items: FarmDto[] | null }>;

function toFarm(dto: FarmDto): Farm {
    return {
        id: dto.id,
        producerId: dto.producer_id,
        name: dto.name,
        altitude: dto.altitude,
        createdAt: dto.created_at,
    };
}

export async function listFarms(): Promise<readonly Farm[]> {
    const result = await request<FarmListDto>("/farms");
    return (result.items ?? []).map(toFarm);
}
