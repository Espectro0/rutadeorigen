import { cookies, headers } from "next/headers";
import { getRequestConfig } from "next-intl/server";
import { DEFAULT_LOCALE, LOCALE_COOKIE, isLocale, type AppLocale } from "./config";

function fromAcceptLanguage(header: string | null): AppLocale | undefined {
    if (header === null) return undefined;
    for (const part of header.split(",")) {
        const tag = part.split(";")[0].trim().toLowerCase().split("-")[0];
        if (isLocale(tag)) return tag;
    }
    return undefined;
}

export default getRequestConfig(async () => {
    const saved = (await cookies()).get(LOCALE_COOKIE)?.value;
    const locale = isLocale(saved)
        ? saved
        : (fromAcceptLanguage((await headers()).get("accept-language")) ?? DEFAULT_LOCALE);

    return {
        locale,
        messages: (await import(`../../messages/${locale}.json`)).default,
    };
});
