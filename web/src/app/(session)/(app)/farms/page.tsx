import { getTranslations } from "next-intl/server";
import { FarmList } from "@/features/farms/components/farm-list";

export default async function FarmsPage() {
    const t = await getTranslations("Farms");

    return (
        <div className="flex flex-col gap-6">
            <div>
                <h1 className="text-2xl font-semibold tracking-tight">{t("title")}</h1>
                <p className="text-ink-dim">{t("subtitle")}</p>
            </div>
            <FarmList />
        </div>
    );
}
