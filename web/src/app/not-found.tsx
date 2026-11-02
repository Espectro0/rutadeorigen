import Link from "next/link";
import { getTranslations } from "next-intl/server";
import { SiteFooter } from "@/components/site-footer";
import { SiteHeader } from "@/components/site-header";

export default async function NotFound() {
    const t = await getTranslations("NotFound");

    return (
        <>
            <SiteHeader />
            <main className="flex-1 p-6">
                <div className="mx-auto flex max-w-md flex-col items-center gap-4 py-16 text-center">
                    <h1 className="text-2xl font-semibold tracking-tight">{t("title")}</h1>
                    <p className="text-ink-dim">{t("body")}</p>
                    <Link href="/" className="btn-secondary">
                        {t("home")}
                    </Link>
                </div>
            </main>
            <SiteFooter />
        </>
    );
}
