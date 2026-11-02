import type { ValidActorType } from "@/features/session/types";

export type ActorSummary = Readonly<{
    id: string;
    type: ValidActorType;
    name: string;
}>;
