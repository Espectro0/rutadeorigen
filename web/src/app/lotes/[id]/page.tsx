import type { Metadata } from "next";
import Link from "next/link";
import { notFound } from "next/navigation";
import { getTranslations } from "next-intl/server";
import { SiteFooter } from "@/components/site-footer";
import { SiteHeader } from "@/components/site-header";
import { PublicLotView } from "@/features/public-lot/components/public-lot-view";
import { getPublicLot } from "@/features/public-lot/api";

type Props = Readonly<{ params: Promise<{ id: string }> }>;

export async function generateMetadata({ params }: Props): Promise<Metadata> {
    const { id } = await params;
    const t = await getTranslations("Public");
    const result = await getPublicLot(id);

    if (result.kind !== "ok") return { title: t("metaTitleFallback") };

    const { lot } = result;
    return {
        title: t("metaTitle", { code: lot.code }),
        description: t("metaDescription", {
            variety: lot.variety,
            farm: lot.farmName,
            producer: lot.producerName,
        }),
    };
}

export default async function PublicLotPage({ params }: Props) {
    const { id } = await params;
    const t = await getTranslations("Public");
    const result = await getPublicLot(id);

    if (result.kind === "not_found") notFound();

    return (
        <>
            <SiteHeader />
            <main className="flex-1 p-6">
                {result.kind === "ok" ? (
                    <PublicLotView lot={result.lot} />
                ) : (
                    <div role="alert" className="mx-auto flex max-w-md flex-col items-center gap-4 py-16 text-center">
                        <h1 className="text-2xl font-semibold tracking-tight">{t("unavailable.title")}</h1>
                        <p className="text-ink-dim">{t("unavailable.body")}</p>
                        <Link href="/" className="btn-secondary">
                            {t("home")}
                        </Link>
                    </div>
                )}
            </main>
            <SiteFooter />
        </>
    );
}
