"use client";

import { useEffect, useRef, useState } from "react";

type Settled<T> =
    | Readonly<{ key: string; ok: true; value: T }>
    | Readonly<{ key: string; ok: false; error: unknown }>;

export type Remote<T> = Readonly<{
    loading: boolean;
    data: T | undefined;
    error: unknown;
    reload: () => void;
}>;

export function useRemote<T>(key: string, load: () => Promise<T>): Remote<T> {
    const [settled, setSettled] = useState<Settled<T> | null>(null);
    const [attempt, setAttempt] = useState(0);
    const loadRef = useRef(load);
    const requestKey = `${key}#${attempt}`;

    useEffect(() => {
        loadRef.current = load;
    });

    useEffect(() => {
        let active = true;
        loadRef
            .current()
            .then((value) => {
                if (active) setSettled({ key: requestKey, ok: true, value });
            })
            .catch((error: unknown) => {
                if (active) setSettled({ key: requestKey, ok: false, error });
            });
        return () => {
            active = false;
        };
    }, [requestKey]);

    const current = settled !== null && settled.key === requestKey ? settled : null;

    return {
        loading: current === null,
        data: current !== null && current.ok ? current.value : undefined,
        error: current !== null && !current.ok ? current.error : undefined,
        reload: () => setAttempt((value) => value + 1),
    };
}
