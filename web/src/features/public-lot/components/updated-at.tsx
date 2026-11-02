"use client";

import { useSyncExternalStore } from "react";
import { useFormatter, useTranslations } from "next-intl";

const subscribe = () => () => {};

export function UpdatedAt({ iso }: Readonly<{ iso: string }>) {
    const t = useTranslations("Public");
    const format = useFormatter();
    const mounted = useSyncExternalStore(
        subscribe,
        () => true,
        () => false,
    );

    const text = mounted
        ? t("updated", {
              date: format.dateTime(new Date(iso), {
                  dateStyle: "medium",
                  timeStyle: "short",
                  timeZone: Intl.DateTimeFormat().resolvedOptions().timeZone,
              }),
          })
        : " ";

    return <p className="min-h-5 text-center text-sm text-ink-dim">{text}</p>;
}
