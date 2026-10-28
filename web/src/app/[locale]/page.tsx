import Image from "next/image";
import { hasLocale } from "next-intl";
import { getTranslations, setRequestLocale } from "next-intl/server";
import { notFound } from "next/navigation";
import { SiteFooter } from "@/components/site-footer";
import { Link } from "@/i18n/navigation";
import { routing } from "@/i18n/routing";
import {SiteHeader} from "@/components/site-header";

export default async function HomePage({ params }: PageProps<"/[locale]">) {
    const { locale } = await params;
    if (!hasLocale(routing.locales, locale)) notFound();
    setRequestLocale(locale);
    const t = await getTranslations("Home");

    return (
        <>
            <SiteHeader />
            <main className="flex flex-1 flex-col items-center justify-center gap-6 p-8 text-center">
                <Image src="/logo-without-bg.svg" alt="" width={96} height={96} priority />
                <h1 className="font-display text-5xl font-semibold tracking-tight text-terracotta">
                    {t("title")}
                </h1>
                <p className="max-w-md text-lg text-ink-dim">{t("subtitle")}</p>
                {/* eslint-disable-next-line @next/next/no-html-link-for-pages */}
                <Link href="/auth/login" className="btn-primary">
                    {t("login")}
                </Link>
                <p className="max-w-md text-lg text-ink-dim">{t("info")}</p>
            </main>
            <SiteFooter />
        </>
    );
}