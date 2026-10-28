export const ACTOR_TYPES = ["organization", "producer", "roaster", "brand"] as const;

export type ValidActorType = (typeof ACTOR_TYPES)[number];
export type ActorType = ValidActorType | "unknown";

export function parseActorType(value: string): ActorType {
    const normalized = value.trim().toLowerCase();
    return (ACTOR_TYPES as readonly string[]).includes(normalized)
        ? (normalized as ValidActorType)
        : "unknown";
}

export type ActorStatus = "active" | "inactive" | "unknown";

export function parseActorStatus(value: string): ActorStatus {
    return value === "active" || value === "inactive" ? value : "unknown";
}

export type Actor = Readonly<{
    id: string;
    type: ActorType;
    name: string;
    email: string;
    status: ActorStatus;
    createdAt: string;
    firstName: string;
    lastName: string;
}>;

export const DEFAULT_ACTOR: Actor = {
    id: "",
    type: "unknown",
    name: "",
    email: "",
    status: "unknown",
    createdAt: "",
    firstName: "",
    lastName: "",
};

export type RegisterActorInput = Readonly<{
    type: ValidActorType;
    name: string;
}>;

export type SessionState =
    | Readonly<{ status: "authenticated"; actor: Actor }>
    | Readonly<{ status: "unauthenticated" }>
    | Readonly<{ status: "unregistered" }>;