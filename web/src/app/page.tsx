import Image from "next/image";
import { getTranslations } from "next-intl/server";
import { SiteFooter } from "@/components/site-footer";
import { SiteHeader } from "@/components/site-header";

export default async function HomePage() {
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
                {/* eslint-disable-next-line @next/next/no-html-link-for-pages -- /auth/login lo sirve el backend */}
                <a href="/auth/login" className="btn-primary">
                    {t("login")}
                </a>
                <p className="max-w-md text-lg text-ink-dim">{t("info")}</p>
            </main>
            <SiteFooter />
        </>
    );
}
