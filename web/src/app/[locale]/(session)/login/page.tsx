import { hasLocale } from "next-intl";
import { getTranslations, setRequestLocale } from "next-intl/server";
import { notFound } from "next/navigation";
import { RequireSession } from "@/features/session/require-session";
import { routing } from "@/i18n/routing";

export default async function LoginPage({ params }: PageProps<"/[locale]/login">) {
    const { locale } = await params;
    if (!hasLocale(routing.locales, locale)) notFound();
    setRequestLocale(locale);
    const t = await getTranslations("Login");

    return (
        <RequireSession expected="unauthenticated">
            <main className="flex flex-1 flex-col items-center justify-center gap-4 p-8 text-center">
                <h1 className="text-3xl font-semibold tracking-tight">{t("title")}</h1>
                <p className="max-w-md text-zinc-600 dark:text-zinc-400">{t("subtitle")}</p>
                <a href="/auth/login" className="rounded bg-black px-6 py-3 text-white dark:bg-white dark:text-black">
                    {t("submit")}
                </a>
            </main>
        </RequireSession>
    );
}