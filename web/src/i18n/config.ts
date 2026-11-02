export const LOCALES = ["es", "en"] as const;

export type AppLocale = (typeof LOCALES)[number];

export const DEFAULT_LOCALE: AppLocale = "es";

export const LOCALE_COOKIE = "NEXT_LOCALE";

export const TIMEZONE_COOKIE = "tz";

export const DEFAULT_TIMEZONE = "UTC";

export function isTimeZone(value: string | undefined): value is string {
    if (value === undefined || value === "") return false;
    try {
        new Intl.DateTimeFormat(undefined, { timeZone: value });
        return true;
    } catch {
        return false;
    }
}

export function isLocale(value: string | undefined): value is AppLocale {
    return value !== undefined && (LOCALES as readonly string[]).includes(value);
}

export function nextLocale(current: string): AppLocale {
    const index = LOCALES.findIndex((code) => code === current);
    return LOCALES[(index + 1) % LOCALES.length];
}
