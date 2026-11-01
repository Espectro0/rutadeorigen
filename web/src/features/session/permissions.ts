import type { ActorType, ValidActorType } from "./types";

export type NavKey = "dashboard" | "farms" | "lots";

const NAV_BY_ACTOR: Readonly<Record<ActorType, readonly NavKey[]>> = {
    organization: ["dashboard", "farms", "lots"],
    producer: ["dashboard", "farms", "lots"],
    roaster: ["dashboard", "lots"],
    brand: ["dashboard", "lots"],
    unknown: ["dashboard"],
};

export function navFor(type: ActorType): readonly NavKey[] {
    return NAV_BY_ACTOR[type];
}

export const FARM_WRITERS: readonly ActorType[] = ["producer"];
export const LOT_CREATORS: readonly ActorType[] = ["producer"];

const TRANSFER_TARGET: Readonly<Partial<Record<ActorType, ValidActorType>>> = {
    producer: "roaster",
    roaster: "brand",
};

export function transferTargetFor(type: ActorType): ValidActorType | null {
    return TRANSFER_TARGET[type] ?? null;
}
