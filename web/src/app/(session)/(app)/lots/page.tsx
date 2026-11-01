import { getTranslations } from "next-intl/server";
import { LotList } from "@/features/lots/components/lot-list";

export default async function LotsPage() {
    const t = await getTranslations("Lots");

    return (
        <div className="flex flex-col gap-6">
            <div>
                <h1 className="text-2xl font-semibold tracking-tight">{t("title")}</h1>
                <p className="text-ink-dim">{t("subtitle")}</p>
            </div>
            <LotList />
        </div>
    );
}
