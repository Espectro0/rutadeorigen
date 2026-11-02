"use client";

import { useEffect } from "react";
import { useRouter } from "next/navigation";

export function TimezoneSync({ current }: Readonly<{ current: string | undefined }>) {
    const router = useRouter();

    useEffect(() => {
        const zone = Intl.DateTimeFormat().resolvedOptions().timeZone;
        if (zone && zone !== current) {
            document.cookie = `tz=${encodeURIComponent(zone)}; path=/; max-age=31536000; samesite=lax`;
            router.refresh();
        }
    }, [current, router]);

    return null;
}