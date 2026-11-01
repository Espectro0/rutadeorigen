"use client";

import type { ReactNode } from "react";
import { useActor } from "./session-provider";
import type { ActorType } from "./types";

type Props = Readonly<{ types: readonly ActorType[]; children: ReactNode }>;

export function ForActors({ types, children }: Props) {
    const actor = useActor();
    return types.includes(actor.type) ? <>{children}</> : null;
}
