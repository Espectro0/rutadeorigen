import { request } from "@/lib/http-client";
import { parseActorType, type ValidActorType } from "@/features/session/types";
import type { ActorSummary } from "./types";

type ActorSummaryDto = Readonly<{ id: string; type: string; name: string }>;
type ActorListDto = Readonly<{ items: ActorSummaryDto[] | null }>;

export async function listActors(type: ValidActorType): Promise<readonly ActorSummary[]> {
    const result = await request<ActorListDto>(`/actors?type=${encodeURIComponent(type)}`);
    return (result.items ?? []).flatMap((dto) => {
        const parsed = parseActorType(dto.type);
        return parsed === "unknown" ? [] : [{ id: dto.id, type: parsed, name: dto.name }];
    });
}
