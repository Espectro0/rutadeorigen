export const LOCALES = ["es", "en"] as const;

export type AppLocale = (typeof LOCALES)[number];

export const DEFAULT_LOCALE: AppLocale = "es";

export const LOCALE_COOKIE = "NEXT_LOCALE";

export function isLocale(value: string | undefined): value is AppLocale {
    return value !== undefined && (LOCALES as readonly string[]).includes(value);
}

export function nextLocale(current: string): AppLocale {
    const index = LOCALES.findIndex((code) => code === current);
    return LOCALES[(index + 1) % LOCALES.length];
}
