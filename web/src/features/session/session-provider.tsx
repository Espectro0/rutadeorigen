"use client";

import {
    createContext,
    useCallback,
    useContext,
    useEffect,
    useMemo,
    useState,
    type ReactNode,
} from "react";
import { isApiError, type ErrorKind } from "@/lib/api-error";
import { getSession } from "./api";
import {Actor, DEFAULT_ACTOR, SessionState} from "./types";

export type SessionView =
    | SessionState
    | Readonly<{ status: "loading" }>
    | Readonly<{ status: "error"; kind: ErrorKind; correlationId: string }>;

type SessionContextValue = Readonly<{
    session: SessionView;
    reload: () => void;
}>;

const SessionContext = createContext<SessionContextValue>({
    session: { status: "loading" },
    reload: () => {},
});

function toErrorView(error: unknown): SessionView {
    if (isApiError(error)) {
        return { status: "error", kind: error.kind, correlationId: error.correlationId };
    }
    return { status: "error", kind: "unknown", correlationId: "" };
}

export function SessionProvider({ children }: { children: ReactNode }) {
    const [session, setSession] = useState<SessionView>({ status: "loading" });
    const [attempt, setAttempt] = useState(0);

    useEffect(() => {
        let active = true;
        getSession()
            .then((state) => {
                if (active) setSession(state);
            })
            .catch((error: unknown) => {
                if (active) setSession(toErrorView(error));
            });
        return () => {
            active = false;
        };
    }, [attempt]);

    const reload = useCallback(() => {
        setSession({ status: "loading" });
        setAttempt((n) => n + 1);
    }, []);

    const value = useMemo(() => ({ session, reload }), [session, reload]);

    return <SessionContext value={value}>{children}</SessionContext>;
}

export function useSession(): SessionContextValue {
    return useContext(SessionContext);
}

export function useActor(): Actor {
    const { session } = useSession();
    return session.status === "authenticated" ? session.actor : DEFAULT_ACTOR;
}