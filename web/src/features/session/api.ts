import { isApiError } from "@/lib/api-error";
import { request } from "@/lib/http-client";
import {
    parseActorStatus,
    parseActorType,
    type Actor,
    type RegisterActorInput,
    type SessionState,
} from "./types";

const ACTOR_NOT_FOUND = "TRACEABILITY_ACTOR_NOT_FOUND";

type MeDto = Readonly<{
    id: string;
    type: string;
    name: string;
    email: string;
    status: string;
    created_at: string;
    first_name: string;
    last_name: string;
}>;

function toActor(dto: MeDto): Actor {
    return {
        id: dto.id,
        type: parseActorType(dto.type),
        name: dto.name,
        email: dto.email,
        status: parseActorStatus(dto.status),
        createdAt: dto.created_at,
        firstName: dto.first_name,
        lastName: dto.last_name,
    };
}

async function getMe(): Promise<Actor> {
    return toActor(await request<MeDto>("/me"));
}

async function loadSession(): Promise<SessionState> {
    try {
        return { status: "authenticated", actor: await getMe() };
    } catch (error) {
        if (isApiError(error)) {
            if (error.status === 401) {
                return { status: "unauthenticated" };
            }
            if (error.code === ACTOR_NOT_FOUND) {
                return { status: "unregistered" };
            }
        }
        throw error;
    }
}

let pending: Promise<SessionState> | undefined;

export function getSession(): Promise<SessionState> {
    pending ??= loadSession().finally(() => {
        pending = undefined;
    });

    return pending;
}

export async function registerActor(input: RegisterActorInput): Promise<Actor> {
    await request<unknown>("/me", {
        method: "POST",
        body: JSON.stringify({ type: input.type, name: input.name }),
    });
    return getMe();
}
