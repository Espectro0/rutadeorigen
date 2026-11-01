import Link from "next/link";
import { LuPlus } from "react-icons/lu";
import { getTranslations } from "next-intl/server";
import { LotList } from "@/features/lots/components/lot-list";

export default async function LotsPage() {
    const t = await getTranslations("Lots");

    return (
        <div className="flex flex-col gap-6">
            <div className="flex flex-wrap items-start justify-between gap-3">
                <div>
                    <h1 className="text-2xl font-semibold tracking-tight">{t("title")}</h1>
                    <p className="text-ink-dim">{t("subtitle")}</p>
                </div>
                <Link href="/lots/new" className="btn-primary inline-flex items-center gap-2">
                    <LuPlus aria-hidden className="size-4" />
                    {t("new")}
                </Link>
            </div>
            <LotList />
        </div>
    );
}
