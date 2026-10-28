import { hasLocale } from "next-intl";
import { getTranslations, setRequestLocale } from "next-intl/server";
import { notFound } from "next/navigation";
import { routing } from "@/i18n/routing";

export default async function DashboardPage({ params }: PageProps<"/[locale]/dashboard">) {
    const { locale } = await params;
    if (!hasLocale(routing.locales, locale)) notFound();
    setRequestLocale(locale);
    const t = await getTranslations("Dashboard");

    return (
        <main className="flex flex-1 flex-col gap-2 p-6">
            <h1 className="text-2xl font-semibold tracking-tight">{t("title")}</h1>
            <p className="text-zinc-600 dark:text-zinc-400">{t("subtitle")}</p>
        </main>
    );
}