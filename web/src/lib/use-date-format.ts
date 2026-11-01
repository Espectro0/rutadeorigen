"use client";

import { useFormatter } from "next-intl";

export type DateFormat = Readonly<{
    day: (value: string) => string;
    moment: (value: string) => string;
}>;

export function useDateFormat(): DateFormat {
    const format = useFormatter();

    return {
        day: (value) => {
            const date = new Date(`${value}T00:00:00Z`);
            return Number.isNaN(date.getTime())
                ? value
                : format.dateTime(date, { dateStyle: "medium", timeZone: "UTC" });
        },
        moment: (value) => {
            const date = new Date(value);
            return Number.isNaN(date.getTime())
                ? value
                : format.dateTime(date, { dateStyle: "medium", timeStyle: "short" });
        },
    };
}
