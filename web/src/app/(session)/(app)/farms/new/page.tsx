import { getTranslations } from "next-intl/server";
import { FarmForm } from "@/features/farms/components/farm-form";

export default async function NewFarmPage() {
    const t = await getTranslations("Farms.form");

    return (
        <div className="flex flex-col gap-6">
            <h1 className="text-2xl font-semibold tracking-tight">{t("title")}</h1>
            <FarmForm />
        </div>
    );
}
